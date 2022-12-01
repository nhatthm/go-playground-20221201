//go:build !windows && !testsignal

// This test required a piped stdin, we cannot do that with windows.

package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/nhatthm/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/nhatthm/go-playground-20221201/internal/app/cli"
)

func Test_Run_InputPipe_Success(t *testing.T) {
	t.Parallel()

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet("/path1").
			ReturnCode(200).
			Return(`<a href="/path1">Example</a>`)
	})(t)

	pipeFile := filepath.Clean(t.TempDir() + "/pipe")

	// Create a pipe.
	if err := syscall.Mkfifo(pipeFile, 0o666); err != nil {
		t.Fatalf("could not create pipe: %v", err)

		return
	}

	// Write to pipe.
	go func() {
		f, err := os.OpenFile(pipeFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o777) // nolint: gosec
		if err != nil {
			t.Errorf("could not open pipe for writing: %v", err)

			return
		}

		defer func() {
			_ = f.Close() // nolint: errcheck
		}()

		err = os.WriteFile(pipeFile, []byte(srv.URL()+"/path1"), 0o644) // nolint: gosec
		if err != nil {
			t.Errorf("could not write pipe data: %v", err)

			return
		}
	}()

	// Open pipe.
	f, err := os.OpenFile(pipeFile, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		t.Errorf("could not open pipe: %v", err)

		return
	}

	defer func() {
		_ = f.Close() // nolint: errcheck
	}()

	outBuf := new(safeBuffer)
	errBuf := new(safeBuffer)

	code := cli.Run(cli.Config{
		OutWriter:  outBuf,
		ErrWriter:  errBuf,
		NumWorkers: 1,
	}, f)

	expected := fmt.Sprintf(`[{"page_url":"%s/path1","internal_links_num":1,"external_links_num":0,"success":true,"error":null}]`, srv.URL())

	assert.Equal(t, expected, strings.Trim(outBuf.String(), "\n"))
	assert.Empty(t, errBuf.String())
	assert.Equal(t, cli.CodeOK, code)
}
