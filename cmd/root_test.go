package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRoot_HelpMentionsBinary(t *testing.T) {
	var buf bytes.Buffer
	root := New()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(buf.String()), "vobiz") {
		t.Fatalf("help missing 'vobiz':\n%s", buf.String())
	}
}

func TestRoot_OutputFlagParses(t *testing.T) {
	globalOutput = ""
	root := New()
	root.SetArgs([]string{"--output", "json"})
	root.RunE = func(*cobra.Command, []string) error { return nil }
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if globalOutput != "json" {
		t.Fatalf("globalOutput = %q", globalOutput)
	}
}
