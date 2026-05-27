package version

import "testing"

func TestStringIncludesAllFields(t *testing.T) {
	Version = "1.2.3"
	Commit = "abc1234"
	Date = "2026-05-23"

	got := String()
	want := "vobiz 1.2.3 (commit abc1234, built 2026-05-23)"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestStringDefaultsForDevBuild(t *testing.T) {
	Version = "dev"
	Commit = ""
	Date = ""

	got := String()
	want := "vobiz dev (commit unknown, built unknown)"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
