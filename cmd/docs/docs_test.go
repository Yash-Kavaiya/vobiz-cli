package docs

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/docsmcp"
)

type fakeMCP struct {
	results  []docsmcp.Result
	markdown string
}

func (f *fakeMCP) Search(_ context.Context, _ string) ([]docsmcp.Result, error) {
	return f.results, nil
}
func (f *fakeMCP) Fetch(_ context.Context, _ string) (string, error) {
	return f.markdown, nil
}

func TestSearch_RendersResults(t *testing.T) {
	m := &fakeMCP{results: []docsmcp.Result{
		{Title: "Trunks", Path: "/trunks", Snippet: "SIP trunks"},
		{Title: "Apps", Path: "/applications", Snippet: "XML apps"},
	}}
	var out bytes.Buffer
	if err := runSearch(m, "anything", &out); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"Trunks", "/trunks", "Apps", "/applications"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q:\n%s", w, out.String())
		}
	}
}

func TestOpen_RendersMarkdown(t *testing.T) {
	m := &fakeMCP{markdown: "# Hello\n\nBody."}
	var out bytes.Buffer
	if err := runOpen(m, "/anything", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Hello") {
		t.Fatalf("output missing 'Hello':\n%s", out.String())
	}
}
