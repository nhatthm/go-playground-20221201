//go:build testsignal

package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/nhatthm/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/nhatthm/go-playground-20221201/internal/app/cli"
)

func Test_Run_SigTerm(t *testing.T) {
	t.Parallel()

	doneCh := make(chan struct{}, 1)
	syscallCh := make(chan struct{}, 1)

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet("/path1").
			ReturnCode(200).
			Run(func(r *http.Request) ([]byte, error) {
				close(doneCh) // Signal to kill the test process.

				<-syscallCh // Wait until the signal is broadcast.

				return nil, nil
			})
	})(t)

	var (
		code cli.ExitCode
		wg   sync.WaitGroup
	)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	wg.Add(1)

	go func() {
		defer wg.Done()

		counter := 0

		code = cli.Run(cli.Config{
			OutWriter:      outBuf,
			ErrWriter:      errBuf,
			VerbosityLevel: cli.VerbosityLevelError,
			NumWorkers:     1,
		}, readerFunc(func(p []byte) (int, error) {
			counter++

			if counter == 2 {
				<-syscallCh
			} else if counter == 3 {
				return 0, io.EOF
			}

			link := fmt.Sprintf("%s/path%d\n", srv.URL(), counter)

			copy(p[:], link)

			return len(link), nil
		}))
	}()

	<-doneCh // Wait until the http server is serving the 1st request.

	_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	time.Sleep(100 * time.Millisecond) // Sleep to give time for the context to be cancelled.
	close(syscallCh)                   // Signal the reader to send the 2nd link.

	wg.Wait()

	// There should be only one result because the publisher is stopped when the context is canceled.
	expected := fmt.Sprintf(`[{"page_url":"%s/path1","internal_links_num":0,"external_links_num":0,"success":false,"error":"operation canceled"}]`, srv.URL())

	assert.Equal(t, expected, strings.Trim(outBuf.String(), "\r\n"))
	assert.NotEmpty(t, errBuf.String())
	assert.Equal(t, cli.CodeErrOperationCanceled, code)
}
