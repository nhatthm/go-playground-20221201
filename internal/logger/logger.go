package logger

import (
	"github.com/bool64/ctxd"
	"github.com/bool64/zapctxd"
)

// NewLogger initiates a new contextualized zap logger.
func NewLogger(cfg Config) *zapctxd.Logger {
	zCfg := zapctxd.Config{
		Level:   cfg.Level,
		DevMode: true,
		FieldNames: ctxd.FieldNames{
			Timestamp: "timestamp",
			Message:   "message",
		},
		Output:    cfg.Output,
		StripTime: cfg.StripTime,
	}

	l := zapctxd.New(zCfg)

	return l
}
