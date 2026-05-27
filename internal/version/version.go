// Package version holds build-time metadata injected via -ldflags.
package version

import "fmt"

var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

func String() string {
	commit := Commit
	if commit == "" {
		commit = "unknown"
	}
	date := Date
	if date == "" {
		date = "unknown"
	}
	return fmt.Sprintf("vobiz %s (commit %s, built %s)", Version, commit, date)
}
