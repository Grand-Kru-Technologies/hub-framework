package util

import "errors"

// ErrAllRetriesFailed marks that ExecuteWithBackoff exhausted every retry attempt.
// The underlying failure is wrapped with %w so errors.Unwrap still works.
var ErrAllRetriesFailed = errors.New("all retry attempts failed")
