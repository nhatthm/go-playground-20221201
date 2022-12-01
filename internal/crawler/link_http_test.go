//go:build !testsignal

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
	"sync"
	"testing"
	"time"

	"github.com/bool64/ctxd"
	"github.com/nhatthm/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/nhatthm/go-playground-20221201/internal/collector"
	"github.com/nhatthm/go-playground-20221201/internal/crawler"
)

const (
	samplePath = "/path"
	sampleHTML = "../../resources/fixtures/sample.html"
)

func TestLinkCrawler_CrawLinks_OperationCanceled(t *testing.T) {
	t.Parallel()

	shouldCancelCtx := make(chan struct{}, 1)
	ctxCanceled := make(chan struct{}, 1)
	links := make(chan string, 2)

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet(samplePath).
			Run(func(r *http.Request) ([]byte, error) {
				// Inform the test to cancel the context.
				close(shouldCancelCtx)

				// Wait until the context is canceled.
				<-ctxCanceled

				return nil, nil
			})
	})(t)

	c := crawler.NewHTTPLinkCrawler(
		crawler.WithClientTimeout(time.Second),
		crawler.WithNumWorkers(1),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := srv.URL() + samplePath
	actual := make([]crawler.LinkCrawlerResult, 0)
	wg := sync.WaitGroup{}

	wg.Add(2)

	go func() {
		defer wg.Done()

		for r := range c.CrawLinks(ctx, links) {
			actual = append(actual, r)
		}
	}()

	go func() {
		defer wg.Done()
		defer close(links)

		links <- source // This will result in operation canceled error

		<-ctxCanceled // Wait until the context is canceled.

		links <- source // This yields no result because the context is canceled.
	}()

	<-shouldCancelCtx

	cancel()
	time.Sleep(50 * time.Millisecond)
	close(ctxCanceled)

	expected := []crawler.LinkCrawlerResult{
		{
			Source: source,
			Error:  crawler.ErrOperationCanceled,
		},
	}

	wg.Wait()

	assert.Equal(t, expected, actual)
}

func TestLinkCrawler_CrawLinks_CouldNotParseSource(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario string
		source   string
		expected string
	}{
		{
			scenario: "could not parse url",
			source:   "\x1B",
			expected: "parse \"https://\\x1b\": net/url: invalid control character in URL",
		},
		{
			scenario: "missing hostname",
			source:   "https:///relative/path",
			expected: `parse "https:///relative/path": missing hostname`,
		},
		{
			scenario: "unsupported scheme",
			source:   "ftp://file.txt",
			expected: `parse "ftp://file.txt": unsupported scheme "ftp"`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			c := crawler.NewHTTPLinkCrawler()

			results := c.CrawLinks(context.Background(), sendLinks(tc.source))

			assertLinkCrawlerError(t, results, time.Second, tc.source, tc.expected)
		})
	}
}

func TestLinkCrawler_CrawLinks_RequestTimeout(t *testing.T) {
	t.Parallel()

	stopCh := make(chan struct{})

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet(samplePath).
			Run(func(*http.Request) ([]byte, error) {
				<-stopCh

				return nil, nil
			})
	})(t)

	c := crawler.NewHTTPLinkCrawler(crawler.WithClientTimeout(10 * time.Millisecond))

	source := srv.URL() + samplePath
	results := c.CrawLinks(context.Background(), sendLinks(source))

	assertLinkCrawlerError(t, results, time.Second, source, fmt.Sprintf(`failed to send http request: Get %q: context deadline exceeded (Client.Timeout exceeded while awaiting headers)`, source))

	close(stopCh)
}

func TestLinkCrawler_CrawLinks_UnexpectedStatusCode(t *testing.T) {
	t.Parallel()

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet(samplePath).
			ReturnCode(httpmock.StatusNotFound)
	})(t)

	c := crawler.NewHTTPLinkCrawler()

	source := srv.URL() + samplePath
	results := c.CrawLinks(context.Background(), sendLinks(source))

	assertLinkCrawlerError(t, results, time.Second, source, `unexpected status code: 404`)
}

func TestLinkCrawler_CrawLinks_UnsupportedContentType(t *testing.T) {
	t.Parallel()

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet(samplePath).
			ReturnHeader("Content-Type", "text/html; charset=utf-8")
	})(t)

	c := crawler.NewHTTPLinkCrawler()

	source := srv.URL() + samplePath
	results := c.CrawLinks(context.Background(), sendLinks(source))

	assertLinkCrawlerError(t, results, time.Second, source, `unsupported content type: text/html`)
}

func TestLinkCrawler_CrawLinks_BrokenResponse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		contentType   string
		expectedError string
	}{
		{
			scenario:      "missing content type",
			expectedError: `failed to detect content type: unexpected EOF`,
		},
		{
			scenario:      "text/html",
			contentType:   "text/html",
			expectedError: `failed to get links: could not collect links from html doc: unexpected EOF`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			srv := httpmock.New(func(s *httpmock.Server) {
				s.ExpectGet(samplePath).
					ReturnHeader("Content-Type", tc.contentType).
					ReturnHeader("Content-Encoding", "gzip").
					Run(func(*http.Request) ([]byte, error) {
						f, err := os.Open(filepath.Clean(sampleHTML))
						if err != nil {
							return nil, err // nolint: wrapcheck
						}

						defer func() {
							_ = f.Close() // nolint: errcheck
						}()

						buf := new(bytes.Buffer)
						gz := gzip.NewWriter(buf)

						defer gz.Close() // nolint: errcheck

						if _, err := io.Copy(gz, f); err != nil {
							return nil, fmt.Errorf("could not compress: %w", err)
						}

						// Client will get unexpected EOF because gzip writer is not flushed at this point.
						return buf.Bytes(), nil
					})
			})(t)

			c := crawler.NewHTTPLinkCrawler(crawler.WithLinkCollector(collector.NewHTMLLinkCollector(), "text/html"))

			source := srv.URL() + samplePath
			results := c.CrawLinks(context.Background(), sendLinks(source))

			assertLinkCrawlerError(t, results, time.Second, source, tc.expectedError)
		})
	}
}

func TestLinkCrawler_CrawLinks_HTML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario   string
		mockServer func(s *httpmock.Server)
	}{
		{
			scenario: "detect content type when missing",
			mockServer: func(s *httpmock.Server) {
				s.ExpectGet(samplePath).
					ReturnHeader("Content-Type", "").
					ReturnFile(sampleHTML)
			},
		},
		{
			scenario: "detect content type when missing and content is compressed",
			mockServer: func(s *httpmock.Server) {
				s.ExpectGet(samplePath).
					ReturnHeader("Content-Type", "").
					ReturnHeader("Content-Encoding", "gzip").
					Run(gzipFile(sampleHTML))
			},
		},
		{
			scenario: "text/html",
			mockServer: func(s *httpmock.Server) {
				s.ExpectGet(samplePath).
					ReturnHeader("Content-Type", "text/html; charset=utf-8").
					ReturnFile(sampleHTML)
			},
		},
		{
			scenario: "text/html with gzip",
			mockServer: func(s *httpmock.Server) {
				s.ExpectGet(samplePath).
					ReturnHeader("Content-Type", "text/html; charset=utf-8").
					ReturnHeader("Content-Encoding", "gzip").
					Run(gzipFile(sampleHTML))
			},
		},
		{
			scenario: "application/octet-stream",
			mockServer: func(s *httpmock.Server) {
				s.ExpectGet(samplePath).
					ReturnHeader("Content-Type", "application/octet-stream; charset=utf-8").
					ReturnFile(sampleHTML)
			},
		},
		{
			scenario: "application/octet-stream with gzip",
			mockServer: func(s *httpmock.Server) {
				s.ExpectGet(samplePath).
					ReturnHeader("Content-Type", "application/octet-stream; charset=utf-8").
					ReturnHeader("Content-Encoding", "gzip").
					Run(gzipFile(sampleHTML))
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			srv := httpmock.New(tc.mockServer)(t)

			c := crawler.NewHTTPLinkCrawler(
				crawler.WithLinkCollector(collector.NewHTMLLinkCollector(), "text/html"),
				crawler.WithNumWorkers(1),
				crawler.WithClientTimeout(time.Second),
				crawler.WithLogger(ctxd.NoOpLogger{}),
			)

			source := srv.URL() + samplePath
			results := c.CrawLinks(context.Background(), sendLinks(source))

			assertLinkCrawlerResult(t, results, time.Hour, crawler.LinkCrawlerResult{
				Source: source,
				InternalLinks: []string{
					srv.URL() + "/",
					srv.URL() + "/absolute/path",
					srv.URL() + "/relative/path",
					srv.URL() + "/path#anchor",
					srv.URL() + "/path?message=hello%20world",
					srv.URL() + "/",
					srv.URL() + "/path",
				},
				ExternalLinks: []string{
					"http://google.com",
					"http://www.google.com",
					"https://google.com",
					"https://www.google.com",
					"https://example.org/link-is-broken",
					"http://www.bing.com",
				},
			})
		})
	}
}

func TestLinkCrawler_CrawLinks_HTML_CouldNotParseURL(t *testing.T) {
	t.Parallel()

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet(samplePath).
			ReturnHeader("Content-Type", "text/html; charset=utf-8").
			Return(`
				<a href="http://%24">Broken Link</a>
				<a href="/">Working Link</a>
			`)
	})(t)

	c := crawler.NewHTTPLinkCrawler(
		crawler.WithLinkCollector(collector.NewHTMLLinkCollector(), "text/html"),
		crawler.WithNumWorkers(1),
	)

	source := srv.URL() + samplePath
	results := c.CrawLinks(context.Background(), sendLinks(source))

	assertLinkCrawlerResult(t, results, time.Hour, crawler.LinkCrawlerResult{
		Source: source,
		InternalLinks: []string{
			srv.URL() + "/",
		},
		ExternalLinks: []string{},
	})
}

func TestWithNumWorkers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario   string
		numWorkers int
	}{
		{
			scenario:   "negative",
			numWorkers: 1,
		},
		{
			scenario:   "zero",
			numWorkers: 0,
		},
		{
			scenario:   "very big number",
			numWorkers: 2147483647,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			srv := httpmock.New(func(s *httpmock.Server) {
				s.ExpectGet(samplePath).
					ReturnHeader("Content-Type", "text/html; charset=utf-8").
					Return(`
						<a href="/">Working Link</a>
					`)
			})(t)

			c := crawler.NewHTTPLinkCrawler(
				crawler.WithLinkCollector(collector.NewHTMLLinkCollector(), "text/html"),
				crawler.WithNumWorkers(tc.numWorkers),
			)

			source := srv.URL() + samplePath
			results := c.CrawLinks(context.Background(), sendLinks(source))

			assertLinkCrawlerResult(t, results, time.Hour, crawler.LinkCrawlerResult{
				Source: source,
				InternalLinks: []string{
					srv.URL() + "/",
				},
				ExternalLinks: []string{},
			})
		})
	}
}
