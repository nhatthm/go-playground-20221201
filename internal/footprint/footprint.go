package footprint

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/bool64/ctxd"
)

const reportInterval = 100 * time.Millisecond

// Track tracks the resources usage and write to log.
func Track(ctx context.Context, log ctxd.Logger) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-time.After(reportInterval):
			// See: https://golang.org/pkg/runtime/#MemStats
			var m runtime.MemStats

			runtime.ReadMemStats(&m)

			log.Debug(ctx, "memory usage",
				"alloc_mb", formatB(m.Alloc),
				"total_alloc_mb", formatB(m.TotalAlloc),
				"sys_mb", formatB(m.TotalAlloc),
				"num_gc", m.NumGC,
			)
		}
	}
}

func formatB(b uint64) string {
	return fmt.Sprintf("%dMiB", b/1024/1024) // nolint: gomnd // bytes conversion.
}
