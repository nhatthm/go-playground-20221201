package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bool64/ctxd"

	"github.com/nhatthm/go-playground-20221201/internal/collector"
	"github.com/nhatthm/go-playground-20221201/internal/crawler"
	"github.com/nhatthm/go-playground-20221201/internal/footprint"
	"github.com/nhatthm/go-playground-20221201/internal/logger"
)

const (
	// CodeOK indicates that the program exited with success.
	CodeOK = ExitCode(iota)
	// CodeErrOperationCanceled indicates that the program has been terminated and operation is canceled.
	CodeErrOperationCanceled
	// CodeErrNoInputSource indicates that the program has no input source.
	CodeErrNoInputSource
	// CodeErrOpenInputSource indicates that the program could not open input file.
	CodeErrOpenInputSource
	// CodeErrUnsupportedInputSource indicates that the program could not use the input source.
	CodeErrUnsupportedInputSource
	// CodeErrBadArgs indicates that the provided arguments are invalid.
	CodeErrBadArgs
	// CodeErrOutput indicates that the program could not write to output.
	CodeErrOutput
)

const (
	// Limitation for number of workers to avoid resource saturation.
	maxNumWorkers = 24
)

// ExitCode is the exit code of the program.
type ExitCode int

// Run runs the program to crawl links from sources.
//
// It will take only the first valid source as an input. The source types are:
// - []string: A list of URLs.
// - string: A file path that contains a list of URLs, one on each line.
// - io.ReadCloser: A reader that contains a list of URLs, one on each line.
// - io.Reader: A reader that contains a list of URLs, one on each line.
//
// The URLs can be with or without scheme or www prefix, but must have a hostname. If the scheme is missing, default to https.
func Run(cfg Config, inputSources ...any) ExitCode {
	// Configure input source.
	inputSource, code, err := initInputSource(inputSources...)
	if err != nil {
		_, _ = fmt.Fprintln(cfg.ErrWriter, err.Error())

		return code
	}

	defer inputSource.Close() // nolint: errcheck

	log := initLogger(cfg.VerbosityLevel, cfg.ErrWriter)

	// Configure crawler.
	c, err := initCrawler(cfg.NumWorkers, cfg.Timeout, log)
	if err != nil {
		_, _ = fmt.Fprintln(cfg.ErrWriter, err.Error())

		return CodeErrBadArgs
	}

	// Configure resultWriter.
	var writeResult resultWriter

	if cfg.VerbosityLevel > VerbosityLevelSilent {
		// When the verbosity level is not silent, the log messages will be printed to the output randomly.
		// And the application cannot guarantee the prettified output to human users because stdout and stderr are visualized on the same screen.
		// This is not a problem to machines because the log messages are sent to stderr which is another file descriptor.
		//
		// Therefore, we will buffer the output and send at once when all the links are processed.
		writeResult = bufferedJSONResultWriter(cfg.OutWriter, cfg.PrettyOutput, log)
	} else {
		// When the verbosity level is silent, there is no log messages to print. It would be great to see the progress of the program rather than waiting till
		// the end. Therefore, the program could print out the result as soon as it is ready.
		writeResult = unbufferedJSONResultWriter(cfg.OutWriter, cfg.ErrWriter, cfg.PrettyOutput)
	}

	// Use buffered channel to avoid resource saturation.
	publishSource := bufferedSourcePublisher(cfg.NumWorkers, log)

	return doCrawl(c, publishSource, writeResult, inputSource, log)
}

// initLogger returns a new logger.
//
// If the verbosity level is silent, all the log messages will be discarded by sending them to io.Discard.
// Otherwise, the logger will write to the stderr writer.
//
// Then the verbosity level is
// - VerbosityLevelError, the log level will be set to logger.ErrorLevel.
// - VerbosityLevelDebug, the log level will be set to logger.DebugLevel.
func initLogger(level VerbosityLevel, errWriter io.Writer) ctxd.Logger {
	logCfg := logger.Config{
		Output: io.Discard,
		Level:  logger.ErrorLevel,
	}

	if level > VerbosityLevelSilent {
		logCfg.Output = errWriter
	}

	if level > VerbosityLevelError {
		logCfg.Level = logger.DebugLevel
	}

	return logger.NewLogger(logCfg)
}

// initInputSource returns the first valid input source.
//
// It accepts a list of input sources. The source types are:
// - []string: A list of URLs. If the list is empty, it is ignored.
// - string: A file path that contains a list of URLs, one on each line. If the path is empty, it is ignored.
// - io.ReadCloser: A reader that contains a list of URLs, one on each line.
// - io.Reader: A reader that contains a list of URLs, one on each line.
//
// The URLs can be with or without scheme or www prefix, but must have a hostname. If the scheme is missing, default to https.
//
// The function returns an input source as an io.ReadCloser so that it can be streamed and closed by the caller.
//
// nolint: cyclop,goerr113 // Error will be printed out.
func initInputSource(sources ...any) (io.ReadCloser, ExitCode, error) {
	for _, source := range sources {
		switch s := source.(type) {
		case nil:
			continue

		case []string:
			if len(s) == 0 {
				continue
			}

			return io.NopCloser(strings.NewReader(strings.Join(s, "\n"))), CodeOK, nil

		case string:
			if len(s) == 0 {
				continue
			}

			f, err := os.Open(filepath.Clean(s))
			if err != nil {
				return nil, CodeErrOpenInputSource, fmt.Errorf("could not open input file: %w", err)
			}

			return f, CodeOK, nil

		case io.ReadCloser:
			return s, CodeOK, nil

		case io.Reader:
			return io.NopCloser(s), CodeOK, nil

		default:
			return nil, CodeErrUnsupportedInputSource, fmt.Errorf("unsupported input source: %T", s)
		}
	}

	return nil, CodeErrNoInputSource, errors.New("no input source")
}

// initCrawler initiates a new crawler.LinkCrawler for counting links.
//
// The function returns an error if the number of workers is smaller than 1 or greater than the maximum number of workers.
//
// nolint: goerr113 // Error will be printed out.
func initCrawler(numWorkers int, timeout time.Duration, log ctxd.Logger) (crawler.LinkCrawler, error) {
	if numWorkers < 1 {
		return nil, errors.New(`number of workers must be greater than 0`)
	} else if numWorkers > maxNumWorkers {
		return nil, fmt.Errorf(`maximum workers is %d`, maxNumWorkers)
	}

	c := crawler.NewHTTPLinkCrawler(
		crawler.WithLinkCollectors(map[string]collector.LinkCollector{
			"text/html":  collector.NewHTMLLinkCollector(),
			"text/plain": collector.NewTextLinkCollector(),
		}),
		crawler.WithLinkCollector(collector.NewJSONLinkCollector(), "application/json", "text/x-json"),
		crawler.WithClientTimeout(timeout),
		crawler.WithNumWorkers(numWorkers),
		crawler.WithLogger(log),
	)

	return c, nil
}

// doCrawl crawls the input source and prints the result to the output writer.
//
// In case of SIGINT or SIGTERM, the crawler will be gracefully stopped and the function will return CodeErrOperationCanceled.
// In case of output error, the function will return CodeErrOutput.
//
// The result will be channeled to the result writer for writing to the output.
func doCrawl(c crawler.LinkCrawler, publishSource sourcePublisher, writeResult resultWriter, source io.Reader, log ctxd.Logger) ExitCode {
	ctx, cancel := context.WithCancel(context.Background())

	go footprint.Track(ctx, log)

	code := CodeOK
	codeMu := &sync.Mutex{}
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(2) // nolint: gomnd // WaitGroup is used to wait for goroutines to finish.

	go func() { // Watch for termination to cancel the context in order to signal all the workers to stop.
		defer wg.Done()
		defer close(sigs)

		select {
		case <-sigs:
			codeMu.Lock()
			code = CodeErrOperationCanceled
			codeMu.Unlock()

			cancel()
		case <-ctx.Done():
			return
		}
	}()

	go func() {
		defer wg.Done()
		defer cancel()

		linksCh := publishSource(ctx, source)
		wCode := writeResult(c.CrawLinks(ctx, linksCh))

		codeMu.Lock()
		defer codeMu.Unlock()

		if wCode != CodeOK && code != CodeErrOperationCanceled {
			code = wCode
		}
	}()

	wg.Wait()

	return code
}
