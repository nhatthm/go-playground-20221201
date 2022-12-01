package cli

import (
	"bufio"
	"context"
	"io"

	"github.com/bool64/ctxd"
)

// sourcePublisher is a function that reads from a source and publishes the results to a channel.
type sourcePublisher func(ctx context.Context, source io.Reader) <-chan string

// bufferedSourcePublisher creates a new source publisher that reads from a source and publishes the results to a buffered channel.
//
// The buffer size is double the number of workers. This is a fair balance between resource saturation and performance.
func bufferedSourcePublisher(numWorkers int, log ctxd.Logger) sourcePublisher {
	return func(ctx context.Context, source io.Reader) <-chan string {
		bufSize := numWorkers * 2 // nolint: gomnd // Buffer size is double the number of workers.
		linksCh := make(chan string, bufSize)

		log.Debug(ctx, "started buffered publisher", "buffer_size", bufSize)

		go func() {
			defer close(linksCh)

			s := bufio.NewScanner(source)

		process:
			for {
				select {
				case <-ctx.Done():
					log.Debug(ctx, "buffered publisher stopped")

					return

				default:
					if !s.Scan() {
						break process
					}

					link := s.Text()

					log.Debug(ctx, "publishing source", "source", link)

					linksCh <- link
				}
			}

			if err := s.Err(); err != nil {
				log.Error(ctx, "could not read input for publishing", "error", err)
			}
		}()

		return linksCh
	}
}
