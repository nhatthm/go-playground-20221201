package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nhatthm/go-playground-20221201/internal/app/cli"
)

const (
	// defaultNumWorkers is the default value for number of workers.
	defaultNumWorkers = 10
	// defaultTimeout is the default timeout for requesting an url.
	defaultTimeout = 30 * time.Second

	usage = `Crawl websites and count for internal and external links.

Usage:
  [app] [options] [link1 link2 ... linkN]

Options:
  -f, --file PATH/TO/FILE
                    Path to the input file that contains a list of urls,
                    separated by '\n'.
                    This option is used if no links are provided.
  -p, --parallel NUM
                    Number of workers for crawling. Default to [defaultNumWorkers].
  -t, --timeout TIMEOUT
                    Timeout for requesting an url, in the form "72h3m0.5s".
                    Default to [defaultTimeout].
  --no-pretty       Disable pretty output.
  -v, --verbose     Print out the error log messages.
  -vv               Print out the all log messages.
  -h, --help        Print out the help message.

Examples:
  Crawl all the urls in path/to/file.txt:
    [app] -p 24 -i path/to/file.txt

  Crawl all the urls in arguments:
    [app] -p 10 google.com facebook.com

  Crawl all the urls in stdin:
    echo -n "google.com" | [app] -p 10 -vv

  Crawl with timeout:
    [app] -t 10s google.com

Note:
  - All urls can be with or without scheme or www prefix, but must have a
    hostname. If the scheme is missing, default to https.

Read more:
  - Time Duration format: https://golang.org/pkg/time/#ParseDuration
`
)

var (
	// argInputFile is the path to an input file that contains a list of urls, separated by '\n'.
	argInputFile string
	// argNumWorkers is the number of workers for crawling urls. Default to defaultNumWorkers.
	argNumWorkers = defaultNumWorkers
	// argTimeout is the timeout for requesting an url.
	argTimeout time.Duration
	// argNoPretty is used to turn of json prettifier.
	argNoPretty bool

	// argVerbose is used to set the verbosity level.
	argVerbose bool
	// argVerbose is used to set the verbosity level.
	argVeryVerbose bool
)

// init is for registering all the arguments.
// nolint: gochecknoinits
func init() {
	flag.StringVar(&argInputFile, "file", "", "")
	flag.StringVar(&argInputFile, "f", "", "")
	flag.IntVar(&argNumWorkers, "parallel", defaultNumWorkers, "")
	flag.IntVar(&argNumWorkers, "p", defaultNumWorkers, "")
	flag.DurationVar(&argTimeout, "timeout", 0, "")
	flag.DurationVar(&argTimeout, "t", defaultTimeout, "")
	flag.BoolVar(&argNoPretty, "no-pretty", false, "")
	flag.BoolVar(&argVerbose, "verbose", false, "")
	flag.BoolVar(&argVerbose, "v", false, "")
	flag.BoolVar(&argVeryVerbose, "vv", false, "")

	flag.Usage = func() {
		r := strings.NewReplacer(
			`[app]`, filepath.Base(os.Args[0]),
			`[defaultNumWorkers]`, strconv.Itoa(defaultNumWorkers),
			`[defaultTimeout]`, defaultTimeout.String(),
		)

		fmt.Print(r.Replace(usage))
	}
}

func main() {
	os.Exit(runMain())
}

func runMain() int {
	flag.Parse()

	cfg := cli.Config{
		OutWriter:      os.Stdout,
		ErrWriter:      os.Stderr,
		NumWorkers:     argNumWorkers,
		Timeout:        argTimeout,
		PrettyOutput:   !argNoPretty,
		VerbosityLevel: cli.VerbosityLevelSilent,
	}

	if argVerbose {
		cfg.VerbosityLevel = cli.VerbosityLevelError
	} else if argVeryVerbose {
		cfg.VerbosityLevel = cli.VerbosityLevelDebug
	}

	return int(cli.Run(cfg, flag.Args(), argInputFile, pipeFromStdIn(os.Stdin)))
}

// Detect if stdin is piped from another process.
func pipeFromStdIn(in *os.File) io.ReadCloser {
	fi, err := in.Stat()
	if err != nil {
		// Just ignore because we do not know if it is a pipe or not.
		return nil
	}

	if (fi.Mode() & os.ModeNamedPipe) != 0 {
		return io.NopCloser(in)
	}

	return nil
}
