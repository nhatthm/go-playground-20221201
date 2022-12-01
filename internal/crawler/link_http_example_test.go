//go:build !testsignal

package crawler_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nhatthm/httpmock"

	"github.com/nhatthm/go-playground-20221201/internal/collector"
	"github.com/nhatthm/go-playground-20221201/internal/crawler"
)

func ExampleNewHTTPLinkCrawler() {
	srv1 := httpmock.MockServer(func(s *httpmock.Server) {
		s.ExpectGet("/").
			ReturnHeader("Content-Type", "text/html").
			ReturnCode(httpmock.StatusOK).
			Return([]byte(`
				<a href="server1.html">link to server1.html</a>
				<a href="https://example.com">link to example</a>
			`))
	})

	srv2 := httpmock.MockServer(func(s *httpmock.Server) {
		s.ExpectGet("/").
			After(100*time.Millisecond).
			ReturnHeader("Content-Type", "text/plain").
			ReturnCode(httpmock.StatusOK).
			Return([]byte(`
				https://example.com
			`))
	})

	// Create a new crawler.
	c := crawler.NewHTTPLinkCrawler(
		crawler.WithNumWorkers(24),
		crawler.WithLinkCollectors(map[string]collector.LinkCollector{
			"text/html":  collector.NewHTMLLinkCollector(),
			"text/plain": collector.NewTextLinkCollector(),
		}),
		crawler.WithLinkCollector(collector.NewJSONLinkCollector(), "application/json", "text/x-json"),
	)

	links := make(chan string)

	go func() {
		examples := []string{
			srv1.URL() + "/",
			srv2.URL() + "/",
		}

		for _, link := range examples {
			links <- link
		}

		close(links)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	replacer := strings.NewReplacer(
		srv1.URL(), "http://server1",
		srv2.URL(), "http://server2",
	)

	for r := range c.CrawLinks(ctx, links) {
		if r.Error != nil {
			panic(r.Error)
		}

		r.Source = replacer.Replace(r.Source)

		fmt.Printf("source: %s\nnum internal links: %d\nnum external links: %d\n", r.Source, len(r.InternalLinks), len(r.ExternalLinks))
	}

	// Output:
	// source: http://server1/
	// num internal links: 1
	// num external links: 1
	// source: http://server2/
	// num internal links: 0
	// num external links: 1
}
