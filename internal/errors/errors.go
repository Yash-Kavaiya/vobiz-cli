// Package errors defines typed CLI errors and maps them to process exit codes.
package errors

import "errors"

var (
	ErrAuth        = errors.New("authentication error")
	ErrNotFound    = errors.New("not found")
	ErrValidation  = errors.New("validation error")
	ErrRateLimited = errors.New("rate limited")
	ErrServer      = errors.New("server error")
	ErrInternal    = errors.New("internal error")
)

// ExitCode maps an error (possibly wrapped) to a process exit code:
//
//	0  success
//	1  user error (auth/notfound/validation)
//	2  API error after retries (rate-limited/server/network)
//	3  internal / unknown bug
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	switch {
	case errors.Is(err, ErrAuth),
		errors.Is(err, ErrNotFound),
		errors.Is(err, ErrValidation):
		return 1
	case errors.Is(err, ErrRateLimited),
		errors.Is(err, ErrServer):
		return 2
	default:
		return 3
	}
}
