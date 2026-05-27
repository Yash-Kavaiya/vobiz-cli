package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var buf bytes.Buffer
	root := New()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "vobiz") {
		t.Fatalf("version output unexpected: %q", buf.String())
	}
}

func TestCompletionBash(t *testing.T) {
	var buf bytes.Buffer
	root := New()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"completion", "bash"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "complete") {
		head := buf.String()
		if len(head) > 200 {
			head = head[:200]
		}
		t.Fatalf("completion bash output unexpected (no 'complete' text): %q", head)
	}
}
