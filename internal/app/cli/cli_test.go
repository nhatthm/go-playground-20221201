package cli_test

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/nhatthm/httpmock"
)

// Mock interfaces for testing.

type readerFunc func(p []byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) {
	return f(p)
}

type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) {
	return f(p)
}

type safeBuffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

func (s *safeBuffer) Read(p []byte) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.buffer.Read(p) // nolint: wrapcheck
}

func (s *safeBuffer) Write(p []byte) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.buffer.Write(p) // nolint: wrapcheck
}

func (s *safeBuffer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.buffer.String()
}

// srvRequests generates a list of server urls for testing.
func srvRequests(srv *httpmock.Server, numRequests int) []string {
	result := make([]string, numRequests)

	for i := 0; i < numRequests; i++ {
		result[i] = fmt.Sprintf("%s/path%d", srv.URL(), i+1)
	}

	return result
}
