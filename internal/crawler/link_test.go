package crawler_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nhatthm/go-playground-20221201/internal/crawler"
)

func sendLinks(links ...string) <-chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)

		for _, link := range links {
			ch <- link
		}
	}()

	return ch
}

func contextWithDeadline(t *testing.T, d time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()

	deadline, ok := t.Deadline()
	if !ok {
		deadline = time.Now().Add(d)
	}

	return context.WithDeadline(context.Background(), deadline)
}

func gzipFile(file string) func(*http.Request) ([]byte, error) {
	return func(req *http.Request) ([]byte, error) {
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			return nil, fmt.Errorf("could not open for compression: %w", err)
		}

		defer func() {
			_ = f.Close() // nolint: errcheck
		}()

		buf := new(bytes.Buffer)
		gz := gzip.NewWriter(buf)

		if _, err := io.Copy(gz, f); err != nil {
			return nil, fmt.Errorf("could not compress: %w", err)
		}

		if err := gz.Close(); err != nil {
			return nil, fmt.Errorf("could not close compressor: %w", err)
		}

		return buf.Bytes(), nil
	}
}

func assertLinkCrawlerResult(t *testing.T, results <-chan crawler.LinkCrawlerResult, timeout time.Duration, expected crawler.LinkCrawlerResult) {
	t.Helper()

	ctx, cancel := contextWithDeadline(t, timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		t.Errorf("test timed out")

	case actual := <-results:
		assert.Equal(t, expected, actual)
	}
}

func assertLinkCrawlerError(t *testing.T, results <-chan crawler.LinkCrawlerResult, timeout time.Duration, source string, errMsg string) {
	t.Helper()

	ctx, cancel := contextWithDeadline(t, timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		t.Errorf("test timed out")

	case actual := <-results:
		assert.Equal(t, source, actual.Source)
		assert.EqualError(t, actual.Error, errMsg)
	}
}
