package httpx

import (
	"net/http"
	"strconv"
	"time"
)

func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return resp.StatusCode >= 500 && resp.StatusCode <= 599
}

func backoff(attempt int, base time.Duration, resp *http.Response) time.Duration {
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				return time.Duration(secs) * time.Second
			}
		}
	}
	d := base << attempt
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}
