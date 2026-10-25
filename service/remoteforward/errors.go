package remoteforward

import (
	"errors"
	"fmt"
)

var (
	ErrMalformedJSON    = errors.New("malformed JSON response body")
	ErrNonJSONErrorBody = errors.New("upstream error response is not JSON")
)

// UpstreamHTTPError is returned when the upstream responds with a non-2xx status
// and a JSON object body.
type UpstreamHTTPError struct {
	StatusCode int
	Body       []byte
	Payload    map[string]interface{}
}

func (e *UpstreamHTTPError) Error() string {
	return fmt.Sprintf("upstream returned HTTP %d with JSON error body", e.StatusCode)
}

// NonJSONErrorResponse is returned when the upstream responds with a non-2xx status
// and a body that is not a JSON object.
type NonJSONErrorResponse struct {
	StatusCode int
	Body       []byte
}

func (e *NonJSONErrorResponse) Error() string {
	return fmt.Sprintf("upstream returned HTTP %d with non-JSON error body", e.StatusCode)
}
