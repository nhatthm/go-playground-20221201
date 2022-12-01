//go:build !testsignal

package logger_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/bool64/ctxd"
	"github.com/stretchr/testify/assert"

	"github.com/nhatthm/go-playground-20221201/internal/logger"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)

	l := logger.NewLogger(logger.Config{
		Output:    buf,
		Level:     logger.ErrorLevel,
		StripTime: true,
	})

	ctx := ctxd.AddFields(context.Background(), "key", "value")

	l.Debug(ctx, "this should not be written to the buffer")
	l.Info(ctx, "this should not be written to the buffer")
	l.Warn(ctx, "this should not be written to the buffer")
	l.Error(ctx, "this should be written to the buffer", "key2", "value2")

	expected := `this should be written to the buffer	{"key2": "value2", "key": "value"}`
	notExpected := `this should not be written to the buffer`

	assert.Contains(t, buf.String(), expected)
	assert.NotContains(t, buf.String(), notExpected)
}
