package output

import (
	"bytes"
	"strings"
	"testing"
)

type trunk struct {
	ID   string
	Name string
	CPS  int
}

var cols = []Column{
	{Header: "ID", Field: "ID"},
	{Header: "NAME", Field: "Name"},
	{Header: "CPS", Field: "CPS"},
}

var rows = []trunk{
	{"t1", "Outbound-A", 10},
	{"t2", "Outbound-B", 25},
}

func TestRenderTable(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, rows, cols, FormatTable); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"ID", "NAME", "CPS", "Outbound-A", "Outbound-B", "10", "25"} {
		if !strings.Contains(out, want) {
			t.Fatalf("table output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, rows, cols, FormatJSON); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"ID": "t1"`) || !strings.Contains(out, `"Name": "Outbound-A"`) {
		t.Fatalf("json output unexpected:\n%s", out)
	}
}

func TestRenderYAML(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, rows, cols, FormatYAML); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "- id: t1") {
		t.Fatalf("yaml output unexpected:\n%s", buf.String())
	}
}

func TestParseFormat(t *testing.T) {
	for _, s := range []string{"table", "TABLE", "Table"} {
		f, err := ParseFormat(s)
		if err != nil || f != FormatTable {
			t.Fatalf("ParseFormat(%q) = (%v,%v)", s, f, err)
		}
	}
	if _, err := ParseFormat("xml"); err == nil {
		t.Fatal("expected error for xml")
	}
}
