//go:build !testsignal

package cli_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nhatthm/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/nhatthm/go-playground-20221201/internal/app/cli"
)

func Test_Run_Error_NoInputSource(t *testing.T) {
	t.Parallel()

	outBuf := new(safeBuffer)
	errBuf := new(safeBuffer)

	code := cli.Run(cli.Config{
		OutWriter: outBuf,
		ErrWriter: errBuf,
	})

	expectedError := "no input source\n"

	assert.Empty(t, outBuf.String())
	assert.Equal(t, cli.CodeErrNoInputSource, code)
	assert.Equal(t, expectedError, errBuf.String())
}

func Test_Run_Error_NumWorkers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		numWorkers    int
		expectedError string
	}{
		{
			scenario:      "negative",
			numWorkers:    -1,
			expectedError: "number of workers must be greater than 0",
		},
		{
			scenario:      "zero",
			numWorkers:    0,
			expectedError: "number of workers must be greater than 0",
		},
		{
			scenario:      "too many",
			numWorkers:    25,
			expectedError: "maximum workers is 24",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			outBuf := new(safeBuffer)
			errBuf := new(safeBuffer)

			code := cli.Run(cli.Config{
				OutWriter:  outBuf,
				ErrWriter:  errBuf,
				NumWorkers: tc.numWorkers,
			}, []string{""})

			assert.Empty(t, outBuf.String())
			assert.Equal(t, tc.expectedError, strings.Trim(errBuf.String(), "\n"))
			assert.Equal(t, cli.CodeErrBadArgs, code)
		})
	}
}

func Test_Run_BufferedOutput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario       string
		prettyOutput   bool
		expectedOutput string
	}{
		{
			scenario:     "pretty",
			prettyOutput: true,
			expectedOutput: `[
  {
    "page_url": "[server]/path1",
    "internal_links_num": 1,
    "external_links_num": 0,
    "success": true,
    "error": null
  },
  {
    "page_url": "[server]/path2",
    "internal_links_num": 1,
    "external_links_num": 0,
    "success": true,
    "error": null
  }
]
`,
		},
		{
			scenario:     "no pretty",
			prettyOutput: false,
			expectedOutput: `[{"page_url":"[server]/path1","internal_links_num":1,"external_links_num":0,"success":true,"error":null},{"page_url":"[server]/path2","internal_links_num":1,"external_links_num":0,"success":true,"error":null}]
`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			send1stResp := make(chan struct{}, 1)
			send2ndResp := make(chan struct{}, 1)

			srv := httpmock.New(func(s *httpmock.Server) {
				s.ExpectGet("/path1").
					ReturnCode(httpmock.StatusOK).
					Run(func(r *http.Request) ([]byte, error) {
						defer close(send1stResp)

						return []byte(`<a href="/path1">Example</a>`), nil
					})

				s.ExpectGet("/path2").
					ReturnCode(httpmock.StatusOK).
					Run(func(r *http.Request) ([]byte, error) {
						<-send2ndResp

						return []byte(`<a href="/path2">Example</a>`), nil
					})
			})(t)

			outBuf := new(safeBuffer)
			errBuf := new(safeBuffer)

			var (
				code cli.ExitCode
				wg   sync.WaitGroup
			)

			wg.Add(1)

			go func() {
				defer wg.Done()

				code = cli.Run(cli.Config{
					OutWriter:      outBuf,
					ErrWriter:      errBuf,
					PrettyOutput:   tc.prettyOutput,
					NumWorkers:     1,
					VerbosityLevel: cli.VerbosityLevelError,
				}, srvRequests(srv, 2))
			}()

			<-send1stResp

			assert.Empty(t, outBuf.String())
			assert.Empty(t, errBuf.String())

			close(send2ndResp)

			wg.Wait()

			expected := strings.ReplaceAll(tc.expectedOutput, "[server]", srv.URL())

			assert.Equal(t, expected, outBuf.String())
			assert.Empty(t, errBuf.String())
			assert.Equal(t, cli.CodeOK, code)
		})
	}
}

func Test_Run_BufferedOutput_Error(t *testing.T) {
	t.Parallel()

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet("/path1").
			ReturnCode(httpmock.StatusOK).
			Return(`<a href="/path1">Example</a>`)
	})(t)

	errBuf := new(safeBuffer)

	code := cli.Run(cli.Config{
		OutWriter: writerFunc(func([]byte) (int, error) {
			return 0, errors.New("write error")
		}),
		ErrWriter:      errBuf,
		NumWorkers:     1,
		VerbosityLevel: cli.VerbosityLevelError,
	}, srvRequests(srv, 1))

	expectedError := `failed to encode report	{"error": "write error"}`

	assert.Contains(t, errBuf.String(), expectedError)
	assert.Equal(t, cli.CodeErrOutput, code)
}

func Test_Run_UnbufferedOutput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario       string
		prettyOutput   bool
		expectedOutput string
	}{
		{
			scenario:     "pretty",
			prettyOutput: true,
			expectedOutput: `[
  {
    "page_url": "[server]/path1",
    "internal_links_num": 1,
    "external_links_num": 0,
    "success": true,
    "error": null
  },
  {
    "page_url": "[server]/path2",
    "internal_links_num": 1,
    "external_links_num": 0,
    "success": true,
    "error": null
  }
]`,
		},
		{
			scenario:       "no pretty",
			prettyOutput:   false,
			expectedOutput: `[{"page_url":"[server]/path1","internal_links_num":1,"external_links_num":0,"success":true,"error":null},{"page_url":"[server]/path2","internal_links_num":1,"external_links_num":0,"success":true,"error":null}]`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			send1stResp := make(chan struct{}, 1)
			send2ndResp := make(chan struct{}, 1)

			srv := httpmock.New(func(s *httpmock.Server) {
				s.ExpectGet("/path1").
					ReturnCode(httpmock.StatusOK).
					Run(func(r *http.Request) ([]byte, error) {
						defer close(send1stResp)

						return []byte(`<a href="/path1">Example</a>`), nil
					})

				s.ExpectGet("/path2").
					ReturnCode(httpmock.StatusOK).
					Run(func(r *http.Request) ([]byte, error) {
						<-send2ndResp

						return []byte(`<a href="/path2">Example</a>`), nil
					})
			})(t)

			outBuf := new(safeBuffer)
			errBuf := new(safeBuffer)

			var (
				code cli.ExitCode
				wg   sync.WaitGroup
			)

			wg.Add(1)

			go func() {
				defer wg.Done()

				code = cli.Run(cli.Config{
					OutWriter:    outBuf,
					ErrWriter:    errBuf,
					PrettyOutput: tc.prettyOutput,
					NumWorkers:   1,
				}, srvRequests(srv, 2))
			}()

			<-send1stResp

			time.Sleep(50 * time.Millisecond)

			assert.NotEmpty(t, outBuf.String())
			assert.Empty(t, errBuf.String())

			close(send2ndResp)

			wg.Wait()

			expected := strings.ReplaceAll(tc.expectedOutput, "[server]", srv.URL())

			assert.Equal(t, expected, strings.Trim(outBuf.String(), "\n"))
			assert.Empty(t, errBuf.String())
			assert.Equal(t, cli.CodeOK, code)
		})
	}
}

func Test_Run_UnbufferedOutput_CouldNotWriteOpenBracket(t *testing.T) {
	t.Parallel()

	outBuf := new(safeBuffer)
	outW := writerFunc(func(p []byte) (int, error) {
		if bytes.Contains(p, []byte("[")) && !bytes.Contains(p, []byte("could not")) {
			return 0, errors.New("write error")
		}

		return outBuf.Write(p)
	})

	code := cli.Run(cli.Config{
		OutWriter:  outW,
		ErrWriter:  outW,
		NumWorkers: 1,
	}, strings.NewReader(""))

	expected := `could not write [ to output: write error`

	assert.Equal(t, expected, strings.Trim(outBuf.String(), "\n"))
	assert.Equal(t, cli.CodeErrOutput, code)
}

func Test_Run_UnbufferedOutput_CouldNotWriteCloseBracket(t *testing.T) {
	t.Parallel()

	outBuf := new(safeBuffer)
	outW := writerFunc(func(p []byte) (int, error) {
		if bytes.Contains(p, []byte("]")) && !bytes.Contains(p, []byte("could not")) {
			return 0, errors.New("write error")
		}

		return outBuf.Write(p)
	})

	code := cli.Run(cli.Config{
		OutWriter:  outW,
		ErrWriter:  outW,
		NumWorkers: 1,
	}, strings.NewReader(""))

	expected := `[could not write ] to output: write error`

	assert.Equal(t, expected, strings.Trim(outBuf.String(), "\n"))
	assert.Equal(t, cli.CodeErrOutput, code)
}

func Test_Run_UnbufferedOutput_CouldNotWriteResult(t *testing.T) {
	t.Parallel()

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet("/path1").
			ReturnCode(httpmock.StatusOK).
			Return(`<a href="/path1">Example</a>`)
	})(t)

	outBuf := new(safeBuffer)
	outW := writerFunc(func(p []byte) (int, error) {
		if !bytes.Contains(p, []byte("[")) && !bytes.Contains(p, []byte("]")) && !bytes.Contains(p, []byte("could not")) {
			return 0, errors.New("write error")
		}

		return outBuf.Write(p)
	})

	code := cli.Run(cli.Config{
		OutWriter:  outW,
		ErrWriter:  outW,
		NumWorkers: 1,
	}, srvRequests(srv, 1))

	expected := fmt.Sprintf(`[could not write "%s/path1" report: write error`, srv.URL())

	assert.Equal(t, expected, strings.Trim(outBuf.String(), "\n"))
	assert.Equal(t, cli.CodeErrOutput, code)
}

func Test_Run_InputFile_ErrorNotFound(t *testing.T) {
	t.Parallel()

	outBuf := new(safeBuffer)
	errBuf := new(safeBuffer)

	code := cli.Run(cli.Config{
		OutWriter:  outBuf,
		ErrWriter:  errBuf,
		NumWorkers: 1,
	}, "file-not-found")

	expectedError := "could not open input file: open file-not-found: no such file or directory\n"

	assert.Empty(t, outBuf.String())
	assert.Equal(t, expectedError, errBuf.String())
	assert.Equal(t, cli.CodeErrOpenInputSource, code)
}

func Test_Run_InputFile_Success(t *testing.T) {
	t.Parallel()

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet("/path1").
			ReturnCode(httpmock.StatusOK).
			Return(`<a href="/path1">Example</a>`)
	})(t)

	inputFile := t.TempDir() + "/input.txt"

	err := os.WriteFile(inputFile, []byte(srv.URL()+"/path1"), 0o644) // nolint: gosec
	if err != nil {
		t.Errorf("could not prepare input file: %v", err)

		return
	}

	outBuf := new(safeBuffer)
	errBuf := new(safeBuffer)

	code := cli.Run(cli.Config{
		OutWriter:  outBuf,
		ErrWriter:  errBuf,
		NumWorkers: 1,
	}, inputFile)

	expected := fmt.Sprintf(`[{"page_url":"%s/path1","internal_links_num":1,"external_links_num":0,"success":true,"error":null}]`, srv.URL())

	assert.Equal(t, expected, strings.Trim(outBuf.String(), "\n"))
	assert.Empty(t, errBuf.String())
	assert.Equal(t, cli.CodeOK, code)
}

func Test_Run_RequestError(t *testing.T) {
	t.Parallel()

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet("/path1").
			ReturnCode(httpmock.StatusForbidden)
	})(t)

	outBuf := new(safeBuffer)
	errBuf := new(safeBuffer)

	code := cli.Run(cli.Config{
		OutWriter:      outBuf,
		ErrWriter:      errBuf,
		NumWorkers:     1,
		VerbosityLevel: cli.VerbosityLevelError,
	}, []string{srv.URL() + "/path1"})

	expected := fmt.Sprintf(`[{"page_url":"%s/path1","internal_links_num":0,"external_links_num":0,"success":false,"error":"unexpected status code: 403"}]`, srv.URL())
	expectedError := fmt.Sprintf(`unexpected http status code	{"status_code": 403, "crawler.http.worker_id": 0, "crawler.http.source": "%s/path1", "http.url": "%s/path1", "http.timeout": "30s"}`, srv.URL(), srv.URL())

	assert.Equal(t, expected, strings.Trim(outBuf.String(), "\n"))
	assert.Contains(t, strings.Trim(errBuf.String(), "\n"), expectedError)
	assert.Equal(t, cli.CodeOK, code)
}

func Test_Run_MultipleSources_Unsupported(t *testing.T) {
	t.Parallel()

	outBuf := new(safeBuffer)
	errBuf := new(safeBuffer)

	cfg := cli.Config{
		OutWriter:      outBuf,
		ErrWriter:      errBuf,
		NumWorkers:     1,
		VerbosityLevel: cli.VerbosityLevelDebug,
	}

	code := cli.Run(cfg,
		[]string{},           // This is ignored because it is empty.
		"",                   // This is ignored because it is empty.
		nil,                  // This is ignored because it is nil.
		(io.ReadCloser)(nil), // This is ignored because it is nil.
		(io.Reader)(nil),     // This is ignored because it is nil.
		2,                    // This is not a supported source.
	)

	expectedError := `unsupported input source: int`

	assert.Empty(t, outBuf.String())
	assert.Equal(t, expectedError, strings.Trim(errBuf.String(), "\n"))
	assert.Equal(t, cli.CodeErrUnsupportedInputSource, code)
}

func Test_Run_CouldNotReadFromSource(t *testing.T) {
	t.Parallel()

	outBuf := new(safeBuffer)
	errBuf := new(safeBuffer)

	code := cli.Run(cli.Config{
		OutWriter:      outBuf,
		ErrWriter:      errBuf,
		NumWorkers:     1,
		VerbosityLevel: cli.VerbosityLevelError,
	}, readerFunc(func([]byte) (n int, err error) {
		return 0, errors.New("read error")
	}))

	expected := "[]\n"
	expectedError := `could not read input for publishing	{"error": "read error"}`

	assert.Equal(t, expected, outBuf.String())
	assert.Contains(t, errBuf.String(), expectedError)
	assert.Equal(t, cli.CodeOK, code)
}

func Test_Run_Debug(t *testing.T) {
	t.Parallel()

	srv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectGet("/path1").
			ReturnCode(httpmock.StatusOK).
			Return(`<a href="/path1">Example</a>`)
	})(t)

	outBuf := new(safeBuffer)
	errBuf := new(safeBuffer)

	code := cli.Run(cli.Config{
		OutWriter:      outBuf,
		ErrWriter:      errBuf,
		NumWorkers:     1,
		VerbosityLevel: cli.VerbosityLevelDebug,
	}, srvRequests(srv, 1))

	expected := fmt.Sprintf(`[{"page_url":"%s/path1","internal_links_num":1,"external_links_num":0,"success":true,"error":null}]`, srv.URL())

	assert.Equal(t, expected, strings.Trim(outBuf.String(), "\n"))
	assert.Equal(t, cli.CodeOK, code)

	expectedLogLines := map[string][]string{
		"DEBUG": {
			"started buffered publisher",
			"publishing source",
			"started crawler.http worker",
			"started crawling",
			"send http request",
			"received http response",
			"parsed content type",
			"collected links",
			"finished crawling",
			"stopped all crawler.http workers",
			"received result",
		},
	}

	actualLogLinesCount := 0
	expectedLogLinesCount := 0
	hasError := false

	// nolint: ifshort // The scanner will read and reset the errBuf. This is a test, so it is fine to skip the tee.
	fullLog := errBuf.String()
	scanner := bufio.NewScanner(errBuf)

	actualLogLines := make(map[string]map[string]struct{})

	for scanner.Scan() {
		actualLogLinesCount++

		actualCols := strings.Split(scanner.Text(), "\t")
		actualLevel, actualMsg := actualCols[1], actualCols[3]

		if _, ok := actualLogLines[actualLevel]; !ok {
			actualLogLines[actualLevel] = make(map[string]struct{})
		}

		actualLogLines[actualLevel][actualMsg] = struct{}{}
	}

	for level, expectedMsgs := range expectedLogLines {
		for _, expectedMsg := range expectedMsgs {
			expectedLogLinesCount++

			if _, ok := actualLogLines[level][expectedMsg]; !ok {
				hasError = true

				t.Errorf("missing log line for level %s: %q", level, expectedMsg)
			}
		}
	}

	assert.Equal(t, expectedLogLinesCount, actualLogLinesCount)

	if hasError {
		t.Logf("full log:\n%s", fullLog)
	}
}
