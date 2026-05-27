// Package httpx is the shared HTTP transport for Vobiz REST calls.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	cliErrors "github.com/yash-kavaiya/vobiz-cli/internal/errors"
)

type Config struct {
	BaseURL     string
	AuthID      string
	AuthToken   string
	UserAgent   string
	MaxRetries  int
	BaseBackoff time.Duration
	HTTPClient  *http.Client
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.BaseBackoff == 0 {
		cfg.BaseBackoff = time.Second
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "vobiz-cli"
	}
	return &Client{cfg: cfg, http: cfg.HTTPClient}
}

func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	u, err := c.resolve(path)
	if err != nil {
		return nil, err
	}

	// Buffer the body so we can replay on retry.
	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, err
		}
	}

	idemKey := ""
	if isMutation(method) {
		idemKey = newIdempotencyKey()
	}

	var lastResp *http.Response
	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Auth-ID", c.cfg.AuthID)
		req.Header.Set("X-Auth-Token", c.cfg.AuthToken)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.cfg.UserAgent)
		if isMutation(method) {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", idemKey)
		}

		resp, err := c.http.Do(req)
		lastResp, lastErr = resp, err

		if !shouldRetry(resp, err) {
			break
		}
		if attempt == c.cfg.MaxRetries {
			break
		}
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff(attempt, c.cfg.BaseBackoff, resp)):
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v", cliErrors.ErrServer, lastErr)
	}
	return classify(lastResp)
}

func (c *Client) DoJSON(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	resp, err := c.Do(ctx, method, path, rdr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil || resp.StatusCode == http.StatusNoContent {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) resolve(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	if c.cfg.BaseURL == "" {
		return "", errors.New("httpx: BaseURL is empty")
	}
	u, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return "", err
	}
	// Split path from query so the ? doesn't get URL-escaped into the path.
	pathOnly, query := path, ""
	if i := strings.IndexByte(path, '?'); i >= 0 {
		pathOnly, query = path[:i], path[i+1:]
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(pathOnly, "/")
	if query != "" {
		u.RawQuery = query
	}
	return u.String(), nil
}

func classify(resp *http.Response) (*http.Response, error) {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	rid := resp.Header.Get("X-Request-Id")
	msg := strings.TrimSpace(string(body))
	switch {
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%w: %s%s", cliErrors.ErrAuth, msg, withReqID(rid))
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("%w: %s%s", cliErrors.ErrNotFound, msg, withReqID(rid))
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: %s%s", cliErrors.ErrRateLimited, msg, withReqID(rid))
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return nil, fmt.Errorf("%w: %s%s", cliErrors.ErrValidation, msg, withReqID(rid))
	default:
		return nil, fmt.Errorf("%w: HTTP %d %s%s", cliErrors.ErrServer, resp.StatusCode, msg, withReqID(rid))
	}
}

func withReqID(id string) string {
	if id == "" {
		return ""
	}
	return " (request-id=" + id + ")"
}
