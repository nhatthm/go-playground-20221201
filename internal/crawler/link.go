package crawler

import (
	"context"
	"time"
)

const (
	// ErrOperationCanceled indicates that the operation was canceled.
	ErrOperationCanceled = Error("operation canceled")
)

const (
	// defaultNumWorkers is the default value for number of workers.
	defaultNumWorkers = 10
	// maxNumWorkers is the limitation for number of workers to avoid resource saturation.
	maxNumWorkers = 24

	// defaultTimeout is the default timeout for requesting an url.
	defaultTimeout = 30 * time.Second
)

// LinkCrawlerResult is the result of LinkCrawler.
type LinkCrawlerResult struct {
	Source        string
	InternalLinks []string
	ExternalLinks []string
	Error         error
}

// LinkCrawler counts links from multiple sources.
type LinkCrawler interface {
	CrawLinks(ctx context.Context, sources <-chan string) <-chan LinkCrawlerResult
}
