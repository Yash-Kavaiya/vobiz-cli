package httpx

import "github.com/google/uuid"

func newIdempotencyKey() string { return uuid.NewString() }

func isMutation(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}
