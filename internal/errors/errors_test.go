package errors

import (
	"errors"
	"testing"
)

func TestExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"auth", ErrAuth, 1},
		{"not found", ErrNotFound, 1},
		{"validation", ErrValidation, 1},
		{"rate limited", ErrRateLimited, 2},
		{"server", ErrServer, 2},
		{"unknown", errors.New("boom"), 3},
		{"wrapped auth", errWrap("login required", ErrAuth), 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Fatalf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func errWrap(msg string, target error) error {
	return &wrapped{msg: msg, target: target}
}

type wrapped struct {
	msg    string
	target error
}

func (w *wrapped) Error() string { return w.msg }
func (w *wrapped) Unwrap() error { return w.target }
