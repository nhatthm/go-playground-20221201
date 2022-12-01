package collector_test

type errorReader struct {
	err error
}

func (e errorReader) Read([]byte) (int, error) {
	return 0, e.err
}

func newErrorReader(err error) errorReader {
	return errorReader{err: err}
}
