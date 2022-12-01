package crawler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/bool64/ctxd"

	"github.com/nhatthm/go-playground-20221201/internal/collector"
)

const (
	// ErrMissingHostname indicates that the source url is missing hostname.
	ErrMissingHostname = Error("missing hostname")
	// ErrUnsupportedContentType indicates that the content type is not supported. It is due to no collector for the given content type.
	ErrUnsupportedContentType = Error("unsupported content type")
	// ErrUnsupportedScheme indicates that the source url contains an unsupported scheme.
	ErrUnsupportedScheme = Error("unsupported scheme")
	// ErrUnexpectedStatusCode indicates that the status code is not supported.
	ErrUnexpectedStatusCode = Error("unexpected status code")
)

const (
	// sniffLen is used for detecting content type. See http.sniffLen.
	sniffLen = 512

	// defaultUserAgent is the default user agent to disguise.
	defaultUserAgent = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.51 Safari/537.36`
)

var _ LinkCrawler = (*HTTPLinkCrawler)(nil)

// HTTPLinkCrawler crawls links from HTTP sources.
type HTTPLinkCrawler struct {
	client     *http.Client
	collectors map[string]collector.LinkCollector // Key is mime type, Value is a link collector.
	log        ctxd.Logger

	// numWorkers is the number of workers running in parallel to use for crawling. Default value is defaultNumWorkers.
	numWorkers int
	// userAgent is the user agent to disguise when sending request to server. Default value is defaultUserAgent.
	userAgent string
}

// CrawLinks crawls links from http sources.
//
// The crawler will spawn a number of workers to crawl links and close the result channel when all the workers are done.
// In order to stop the crawler, the caller should cancel the context.
//
// See https://pkg.go.dev/context#WithCancel.
func (c HTTPLinkCrawler) CrawLinks(ctx context.Context, sources <-chan string) <-chan LinkCrawlerResult {
	results := make(chan LinkCrawlerResult)
	wg := sync.WaitGroup{}

	wg.Add(c.numWorkers)

	for i := 0; i < c.numWorkers; i++ {
		ctx := ctxd.AddFields(ctx, "crawler.http.worker_id", i)

		go func(ctx context.Context) {
			defer wg.Done()

			c.log.Debug(ctx, "started crawler.http worker")

			for {
				select {
				// Operation canceled.
				case <-ctx.Done():
					c.log.Debug(ctx, "stopped crawler.http worker")

					return

				case source, isClosed := <-sources:
					if !isClosed {
						return
					}

					results <- c.doCrawl(ctx, source)
				}
			}
		}(ctx)
	}

	// Wait for all workers to finish and close the results channel.
	go func() {
		wg.Wait()
		close(results)

		c.log.Debug(ctx, "stopped all crawler.http workers")
	}()

	return results
}

// doCrawl crawls links from a http source.
func (c HTTPLinkCrawler) doCrawl(ctx context.Context, source string) (result LinkCrawlerResult) {
	startTime := time.Now()
	ctx = ctxd.AddFields(ctx, "crawler.http.source", source)

	c.log.Debug(ctx, "started crawling")

	defer func() {
		c.log.Debug(ctx, "finished crawling", "crawler.http.duration", time.Since(startTime).String())
	}()

	var err error

	result = LinkCrawlerResult{Source: source}

	defer func() {
		if err != nil {
			result.Error = err
		}
	}()

	sourceURL, err := parseURL(source)
	if err != nil {
		c.log.Error(ctx, "failed to parse url", "error", err)

		return
	}

	resp, err := c.doRequest(ctx, *sourceURL)
	if err != nil {
		var uErr *url.Error
		if errors.As(err, &uErr) && errors.Is(uErr.Err, context.Canceled) {
			err = ErrOperationCanceled
		}

		return
	}

	defer resp.Body.Close() // nolint: errcheck

	links, err := c.collectLinks(ctx, resp)
	if err != nil {
		return
	}

	result.InternalLinks, result.ExternalLinks = c.sortLinks(ctx, *sourceURL, links)

	return result
}

func (c HTTPLinkCrawler) doRequest(ctx context.Context, sourceURL url.URL) (*http.Response, error) {
	ctx = ctxd.AddFields(ctx,
		"http.url", sourceURL.String(),
		"http.timeout", c.client.Timeout.String(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL.String(), nil)
	if err != nil {
		// This should not happen because the context is not nil and the source URL is valid (parsed in the caller).
		c.log.Error(ctx, "failed to create http request", "error", err)

		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	c.log.Debug(ctx, "send http request",
		"http.user_agent", c.userAgent,
	)

	startTime := time.Now()
	resp, err := c.client.Do(req)
	endTime := time.Now()

	if err != nil {
		c.log.Error(ctx, "failed to send http request", "error", err)

		return nil, fmt.Errorf("failed to send http request: %w", err)
	}

	c.log.Debug(ctx, "received http response", "http.duration", endTime.Sub(startTime).String())

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		c.log.Error(ctx, "unexpected http status code", "status_code", resp.StatusCode)

		return nil, fmt.Errorf("%w: %d", ErrUnexpectedStatusCode, resp.StatusCode)
	}

	return resp, nil
}

// detectContentType detects the content type of the response.
//
// It returns the media type (without the parameters) from the Content-Type in the response headers. If the Content-Type is not set or is set to
// `application/octet-stream`, the function will use http.DetectContentType() to detect the content type. if http.DetectContentType() cannot determine a more
// specific one, it returns `application/octet-stream`.
//
// See https://pkg.go.dev/net/http#DetectContentType.
func (c HTTPLinkCrawler) detectContentType(ctx context.Context, resp *http.Response) (string, error) {
	contentType := resp.Header.Get("Content-Type")

	if contentType != "" {
		contentType, _, _ = mime.ParseMediaType(contentType) // nolint: errcheck // We do not care about the error, it is probably an error after the `;`
	}

	ctx = ctxd.AddFields(ctx, "http.content_type", contentType)

	c.log.Debug(ctx, "parsed content type")

	if contentType != "" && contentType != "application/octet-stream" {
		return contentType, nil
	}

	// Read body to detect the content type.
	// We do not need to read the whole body into memory. In fact, as documented, http.DetectContentType() needs only http.sniffLen bytes. Therefore, the
	// sniffLen should always be equal to http.sniffLen.
	// This is for avoiding the memory waste in case of unsupported content type.
	sniff, err := io.ReadAll(io.LimitReader(resp.Body, int64(sniffLen)))
	if err != nil {
		c.log.Error(ctx, "failed to detect content type", "error", err)

		return "", fmt.Errorf("failed to detect content type: %w", err)
	}

	// Detect content type by using the first http.sniffLen bytes.
	contentType = http.DetectContentType(sniff)
	contentType, _, _ = mime.ParseMediaType(contentType) // nolint: errcheck // We do not care about the error, it is probably an error after the `;`.

	c.log.Debug(ctx, "detected new content type", "http.new_content_type", contentType)

	// Adjust the body reader.
	// We do not need to put a close here because there is another defer in the caller.
	resp.Body = io.NopCloser(io.MultiReader(bytes.NewReader(sniff), resp.Body))

	return contentType, nil
}

// collectLinks collects links from the response by detecting the content type.
func (c HTTPLinkCrawler) collectLinks(ctx context.Context, resp *http.Response) ([]string, error) {
	// Detect the content type of the source.
	contentType, err := c.detectContentType(ctx, resp)
	if err != nil {
		return nil, err
	}

	ctx = ctxd.AddFields(ctx, "http.content_type", contentType)

	linkCollector, ok := c.collectors[contentType]
	if !ok {
		c.log.Error(ctx, "unsupported content type")

		return nil, fmt.Errorf("%w: %s", ErrUnsupportedContentType, contentType)
	}

	ctx = ctxd.AddFields(ctx, "crawler.http.collector", fmt.Sprintf("%T", linkCollector))

	links, err := linkCollector.GetLinks(resp.Body)
	if err != nil {
		c.log.Error(ctx, "failed to get links", "error", err)

		return nil, fmt.Errorf("failed to get links: %w", err)
	}

	c.log.Debug(ctx, "collected links", "crawler.http.num_links", len(links))

	return links, nil
}

// sortLinks sorts the links into internal and external buckets by comparing with the source url.
//
// Links that do not start with the source url are considered external. And internal links will be resolved to absolute URLs.
//
// For example: given a `http://localhost` source
//   - link: .
//     result: http://localhost/
//   - link: /absolute/path/to/file.html
//     result: http://localhost/absolute/path/to/file.html
//   - link: path/to/file.html
//     result: http://localhost/path/to/file.html
//   - link: path/to/file.html#anchor
//     result: http://localhost/path/to/file.html#anchor
func (c HTTPLinkCrawler) sortLinks(ctx context.Context, source url.URL, links []string) ([]string, []string) {
	internalLinks := make([]string, 0, len(links))
	externalLinks := make([]string, 0, len(links))

	for _, link := range links {
		linkURL, err := url.Parse(link)
		if err != nil {
			c.log.Error(ctx, "failed to parse link", "link", link, "error", err)

			continue
		}

		if linkURL.Scheme != "" && linkURL.Scheme != "http" && linkURL.Scheme != "https" {
			c.log.Debug(ctx, "link is not http or https", "link", link)

			continue
		}

		if linkURL.Host != "" && linkURL.Host != source.Host {
			externalLinks = append(externalLinks, link)

			continue
		}

		linkURL = source.ResolveReference(linkURL)

		internalLinks = append(internalLinks, linkURL.String())
	}

	return internalLinks, externalLinks
}

// NewHTTPLinkCrawler creates a new HTTPLinkCrawler for counting links from HTTP sources.
//
// Usage:
//
//	links := make(chan string)
//
//	go func({
//		examples := []string{"localhost", "example.com"}
//
//		for _, link := range examples {
//			links <- link
//		}
//
//		close(links)
//	})
//
//	c := HTTPLinkCrawler()
//
//	for r := range c.CrawLinks(ctx, links) {
//		fmt.Printf("source: %s\nnum internal links: %d\n", r.Source, len(r.InternalLinks))
//	}
func NewHTTPLinkCrawler(opts ...HTTPLinkCrawlerOption) *HTTPLinkCrawler {
	c := &HTTPLinkCrawler{
		client:     &http.Client{}, // Default HTTP Client.
		collectors: make(map[string]collector.LinkCollector),
		log:        ctxd.NoOpLogger{},

		numWorkers: defaultNumWorkers,
		userAgent:  defaultUserAgent,
	}

	for _, opt := range opts {
		opt.applyHTTPLinkCounterOption(c)
	}

	// Safeguard the number of workers.
	if c.numWorkers < 1 {
		c.numWorkers = defaultNumWorkers
	} else if c.numWorkers > maxNumWorkers {
		c.numWorkers = maxNumWorkers
	}

	if c.client.Timeout == 0 {
		c.client.Timeout = defaultTimeout
	}

	return c
}

// HTTPLinkCrawlerOption is option to set up HTTPLinkCrawler.
type HTTPLinkCrawlerOption interface {
	applyHTTPLinkCounterOption(c *HTTPLinkCrawler)
}

type httpLinkCounterOptionFunc func(c *HTTPLinkCrawler)

func (f httpLinkCounterOptionFunc) applyHTTPLinkCounterOption(c *HTTPLinkCrawler) {
	f(c)
}

// WithLogger sets logger for HTTPLinkCrawler.
func WithLogger(l ctxd.Logger) HTTPLinkCrawlerOption {
	return httpLinkCounterOptionFunc(func(c *HTTPLinkCrawler) {
		c.log = l
	})
}

// WithNumWorkers sets number of workers for HTTPLinkCrawler.
func WithNumWorkers(numWorkers int) HTTPLinkCrawlerOption {
	return httpLinkCounterOptionFunc(func(c *HTTPLinkCrawler) {
		c.numWorkers = numWorkers
	})
}

// WithClientTimeout sets timeout for HTTP client.
func WithClientTimeout(d time.Duration) HTTPLinkCrawlerOption {
	return httpLinkCounterOptionFunc(func(c *HTTPLinkCrawler) {
		c.client.Timeout = d
	})
}

// WithLinkCollectors sets link collectors for HTTPLinkCrawler.
func WithLinkCollectors(collectors map[string]collector.LinkCollector) HTTPLinkCrawlerOption {
	return httpLinkCounterOptionFunc(func(c *HTTPLinkCrawler) {
		c.collectors = collectors
	})
}

// WithLinkCollector sets link collector for HTTPLinkCrawler for multiple content types.
func WithLinkCollector(collector collector.LinkCollector, contentTypes ...string) HTTPLinkCrawlerOption {
	return httpLinkCounterOptionFunc(func(c *HTTPLinkCrawler) {
		for _, contentType := range contentTypes {
			c.collectors[contentType] = collector
		}
	})
}

// parseURL parses the url string into an url.URL.
//
// - If the url string does not have a scheme, it will default to https.
// - If the url string is not a valid url, it will return an error.
// - If the url string does not start with http and https, it will return an error.
func parseURL(s string) (*url.URL, error) {
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}

	u, err := url.Parse(s)
	if err != nil {
		return nil, err // nolint: wrapcheck // *url.URL error is meaningful, we do not need to wrap it.
	}

	if u.Host == "" {
		return nil, fmt.Errorf("parse %q: %w", s, ErrMissingHostname)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("parse %q: %w %q", s, ErrUnsupportedScheme, u.Scheme)
	}

	return u, nil
}
