package cli

import (
	"io"
	"time"
)

// VerbosityLevel is the verbosity level of the application.
type VerbosityLevel uint

const (
	// VerbosityLevelSilent is the silent verbosity level.
	VerbosityLevelSilent VerbosityLevel = iota
	// VerbosityLevelError is the error verbosity level.
	VerbosityLevelError
	// VerbosityLevelDebug is the warning verbosity level.
	VerbosityLevelDebug
)

// Config is the configuration of the application.
type Config struct {
	OutWriter io.Writer // The stream that will receive the results
	ErrWriter io.Writer // The stream that will receive all the log messages and errors.

	NumWorkers     int            // The number of workers that the crawler could run.
	Timeout        time.Duration  // The timeout of the http client of the crawler.
	PrettyOutput   bool           // Disable JSON prettifier.
	VerbosityLevel VerbosityLevel // The verbosity level of the tool.
}
