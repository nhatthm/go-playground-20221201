package crawler

var _ error = (*Error)(nil)

// Error is a crawler error.
type Error string

// Error implements the error interface.
func (e Error) Error() string {
	return string(e)
}
