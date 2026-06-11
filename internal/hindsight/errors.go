package hindsight

import (
	stderrors "errors"
	"fmt"
)

// StatusError is returned by the HTTP transport for non-2xx responses. It
// preserves the request method/path, the HTTP status code, and a bounded prefix
// of the response body so callers can classify failures without substring
// matching.
type StatusError struct {
	Method string
	Path   string
	Code   int
	Body   string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%s %s: status %d: %s", e.Method, e.Path, e.Code, e.Body)
}

// IsUnsupported reports whether err is (or wraps) a StatusError carrying a 404
// or 405 status, i.e. the server does not expose the requested endpoint.
func IsUnsupported(err error) bool {
	var se *StatusError
	if stderrors.As(err, &se) {
		return se.Code == 404 || se.Code == 405
	}
	return false
}
