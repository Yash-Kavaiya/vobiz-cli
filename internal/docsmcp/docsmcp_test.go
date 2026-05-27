package docsmcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newMockServer(t *testing.T, handler func(req mcpRequest, w http.ResponseWriter)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req mcpRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		handler(req, w)
	}))
}

func TestSearch_ParsesResults(t *testing.T) {
	srv := newMockServer(t, func(req mcpRequest, w http.ResponseWriter) {
		if req.Method != "tools/call" || req.Params.Name != "search" {
			t.Fatalf("unexpected: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "jsonrpc":"2.0","id":1,
		  "result":{
		    "content":[
		      {"type":"text","text":"{\"results\":[{\"title\":\"Trunks\",\"path\":\"/trunks\",\"snippet\":\"SIP trunks…\"}]}"}
		    ]
		  }
		}`))
	})
	defer srv.Close()

	c := New(srv.URL)
	got, err := c.Search(context.Background(), "trunks")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "Trunks" || got[0].Path != "/trunks" {
		t.Fatalf("got %+v", got)
	}
}

func TestFetch_ReturnsMarkdown(t *testing.T) {
	srv := newMockServer(t, func(req mcpRequest, w http.ResponseWriter) {
		if req.Params.Name != "fetch" {
			t.Fatalf("unexpected name: %q", req.Params.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "jsonrpc":"2.0","id":1,
		  "result":{"content":[{"type":"text","text":"# Heading\n\nbody."}]}
		}`))
	})
	defer srv.Close()

	md, err := New(srv.URL).Fetch(context.Background(), "/trunks")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "# Heading") {
		t.Fatalf("markdown: %q", md)
	}
}
