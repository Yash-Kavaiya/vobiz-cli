package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cliErrors "github.com/yash-kavaiya/vobiz-cli/internal/errors"
)

func TestDo_SendsAuthHeaders(t *testing.T) {
	var gotID, gotTok, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = r.Header.Get("X-Auth-ID")
		gotTok = r.Header.Get("X-Auth-Token")
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AuthID: "AB", AuthToken: "TK", UserAgent: "vobiz-cli/test"})
	resp, err := c.Do(context.Background(), http.MethodGet, "/anything", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotID != "AB" || gotTok != "TK" || gotUA != "vobiz-cli/test" {
		t.Fatalf("headers: %q %q %q", gotID, gotTok, gotUA)
	}
}

func TestDo_RetriesOn5xxThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AuthID: "x", AuthToken: "y", MaxRetries: 3, BaseBackoff: time.Millisecond})
	resp, err := c.Do(context.Background(), http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || calls != 3 {
		t.Fatalf("status=%d calls=%d", resp.StatusCode, calls)
	}
}

func TestDo_HonorsRetryAfterOn429(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AuthID: "x", AuthToken: "y", MaxRetries: 2, BaseBackoff: time.Millisecond})
	resp, err := c.Do(context.Background(), http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || calls != 2 {
		t.Fatalf("status=%d calls=%d", resp.StatusCode, calls)
	}
}

func TestDo_Returns4xxImmediately(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(401)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AuthID: "x", AuthToken: "y", MaxRetries: 3, BaseBackoff: time.Millisecond})
	_, err := c.Do(context.Background(), http.MethodGet, "/", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, cliErrors.ErrAuth) {
		t.Fatalf("want ErrAuth, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestDo_GeneratesIdempotencyKeyOnMutations(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Idempotency-Key")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AuthID: "x", AuthToken: "y"})
	resp, err := c.Do(context.Background(), http.MethodPost, "/", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got == "" {
		t.Fatal("Idempotency-Key not set on POST")
	}
}

func TestDo_NoIdempotencyKeyOnGET(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Idempotency-Key")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AuthID: "x", AuthToken: "y"})
	resp, err := c.Do(context.Background(), http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got != "" {
		t.Fatalf("GET should not have Idempotency-Key, got %q", got)
	}
}

func TestDoJSON_Decodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"name":"hello"}`)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AuthID: "x", AuthToken: "y"})
	var out struct {
		Name string `json:"name"`
	}
	if err := c.DoJSON(context.Background(), http.MethodGet, "/", nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "hello" {
		t.Fatalf("name = %q", out.Name)
	}
}
