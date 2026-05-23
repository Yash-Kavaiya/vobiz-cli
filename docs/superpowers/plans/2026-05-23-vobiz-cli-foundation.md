# Vobiz CLI — Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a working `vobiz` binary covering scaffolding, all shared internals (config/auth/output/httpx/paginate/errors), the `auth`, `account`, and `docs` command trees, and a green CI matrix. Subsequent plans add the remaining resource groups (calls, numbers, trunks, applications, whatsapp, partner) and the release pipeline.

**Architecture:** Single Cobra binary in `main.go` → `cmd/*` subcommands. Subcommands only parse flags and call interfaces on `internal/client`, never HTTP directly. `internal/httpx` is the single seam for retries, auth headers, idempotency, and `Retry-After`. `internal/output` renders any rows + column spec as table | json | yaml. Credentials resolve flag > env > profile from `~/.vobiz/config.yaml`.

**Tech Stack:** Go 1.22+, `spf13/cobra`, `gopkg.in/yaml.v3`, `jedib0t/go-pretty/v6/table`, `charmbracelet/glamour`, `google/uuid`, `golang.org/x/term`, `github.com/vobiz-ai/vobiz-go-sdk`.

**Companion spec:** `docs/superpowers/specs/2026-05-23-vobiz-cli-design.md`.

---

## Conventions used throughout

- **Module path placeholder:** the plan uses `github.com/yash-kavaiya/vobiz-cli`. If the eventual GitHub owner differs, do a project-wide rename before Task 1.
- **One commit per task.** Stage explicit file lists, never `git add -A`.
- **TDD:** every code task starts with a failing test, then minimal code, then green test, then commit.
- **No backticks in tests** for shell commands — use Go strings.
- **Windows note:** `chmod 0600` is a no-op on Windows; use `os.Chmod` and skip the mode assertion on `runtime.GOOS == "windows"`.

---

## Task 1: Initialize Go module, .gitignore, LICENSE, README skeleton

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `LICENSE` (MIT)
- Modify: `README.md`

- [ ] **Step 1: Initialize the module**

Run:
```bash
cd c:/Users/yashk/Downloads/vobiz-cli
go mod init github.com/yash-kavaiya/vobiz-cli
```

- [ ] **Step 2: Write .gitignore**

Create `.gitignore`:
```gitignore
# Binaries
/vobiz
/vobiz.exe
/dist/

# Go test artifacts
*.test
*.out
coverage.txt

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
Thumbs.db

# Local config / secrets
.env
.env.local
```

- [ ] **Step 3: Write LICENSE (MIT)**

Create `LICENSE`:
```
MIT License

Copyright (c) 2026 Yash Kavaiya

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

- [ ] **Step 4: Overwrite README.md with a skeleton**

Replace the existing placeholder README:
```markdown
# vobiz-cli

The unofficial-but-friendly terminal interface for the [Vobiz](https://vobiz.ai) programmable-telephony platform.

```bash
vobiz auth login
vobiz account balance
vobiz docs search "sip trunk"
```

Status: under active development. See `docs/superpowers/specs/2026-05-23-vobiz-cli-design.md`.
```

- [ ] **Step 5: Commit**

```bash
git add go.mod .gitignore LICENSE README.md
git commit -m "chore: init Go module, license, gitignore, README skeleton"
```

---

## Task 2: Add `internal/version` package with ldflags-injected metadata

**Files:**
- Create: `internal/version/version.go`
- Create: `internal/version/version_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/version/version_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/version/...
```
Expected: build failure — package does not exist.

- [ ] **Step 3: Write the package**

Create `internal/version/version.go`:
```go
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/version/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/version/
git commit -m "feat(version): add ldflags-injected build metadata"
```

---

## Task 3: Add `internal/errors` typed errors and exit-code mapper

**Files:**
- Create: `internal/errors/errors.go`
- Create: `internal/errors/errors_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/errors/errors_test.go`:
```go
package errors

import (
	"errors"
	"testing"
)

func TestExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"auth", ErrAuth, 1},
		{"not found", ErrNotFound, 1},
		{"validation", ErrValidation, 1},
		{"rate limited", ErrRateLimited, 2},
		{"server", ErrServer, 2},
		{"unknown", errors.New("boom"), 3},
		{"wrapped auth", errWrap("login required", ErrAuth), 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Fatalf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func errWrap(msg string, target error) error {
	return &wrapped{msg: msg, target: target}
}

type wrapped struct {
	msg    string
	target error
}

func (w *wrapped) Error() string { return w.msg }
func (w *wrapped) Unwrap() error { return w.target }
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/errors/...
```
Expected: build failure.

- [ ] **Step 3: Write the package**

Create `internal/errors/errors.go`:
```go
// Package errors defines typed CLI errors and maps them to process exit codes.
package errors

import "errors"

var (
	ErrAuth         = errors.New("authentication error")
	ErrNotFound     = errors.New("not found")
	ErrValidation   = errors.New("validation error")
	ErrRateLimited  = errors.New("rate limited")
	ErrServer       = errors.New("server error")
	ErrInternal     = errors.New("internal error")
)

// ExitCode maps an error (possibly wrapped) to a process exit code:
//
//	0  success
//	1  user error (auth/notfound/validation)
//	2  API error after retries (rate-limited/server/network)
//	3  internal / unknown bug
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	switch {
	case errors.Is(err, ErrAuth),
		errors.Is(err, ErrNotFound),
		errors.Is(err, ErrValidation):
		return 1
	case errors.Is(err, ErrRateLimited),
		errors.Is(err, ErrServer):
		return 2
	default:
		return 3
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/errors/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/errors/
git commit -m "feat(errors): typed CLI errors with exit-code mapping"
```

---

## Task 4: Add `internal/config` for `~/.vobiz/config.yaml` read/write

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Add yaml.v3 dependency**

```bash
go get gopkg.in/yaml.v3
```

- [ ] **Step 2: Write the failing test**

Create `internal/config/config_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveAndLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	in := &File{
		ActiveProfile: "default",
		Profiles: map[string]Profile{
			"default": {AuthID: "AB12", AuthToken: "tok", BaseURL: "https://api.vobiz.ai/api/v1"},
			"staging": {AuthID: "ZZ99", AuthToken: "stg", BaseURL: "https://api.vobiz.ai/api/v1"},
		},
	}

	if err := Save(path, in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
		}
	}

	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.ActiveProfile != "default" {
		t.Fatalf("active = %q", out.ActiveProfile)
	}
	if out.Profiles["staging"].AuthID != "ZZ99" {
		t.Fatalf("staging auth id = %q", out.Profiles["staging"].AuthID)
	}
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.yaml")
	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load missing: unexpected err %v", err)
	}
	if len(out.Profiles) != 0 || out.ActiveProfile != "" {
		t.Fatalf("expected empty file struct, got %+v", out)
	}
}

func TestDefaultPathUsesHomeDir(t *testing.T) {
	t.Setenv("HOME", "/tmp/fakehome")
	t.Setenv("USERPROFILE", `C:\fakehome`) // windows
	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if filepath.Base(got) != "config.yaml" {
		t.Fatalf("DefaultPath = %q", got)
	}
	if filepath.Base(filepath.Dir(got)) != ".vobiz" {
		t.Fatalf("DefaultPath parent = %q", filepath.Dir(got))
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/config/...
```
Expected: build failure.

- [ ] **Step 4: Write the package**

Create `internal/config/config.go`:
```go
// Package config reads and writes ~/.vobiz/config.yaml.
package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Profile struct {
	AuthID    string `yaml:"auth_id"`
	AuthToken string `yaml:"auth_token"`
	BaseURL   string `yaml:"base_url,omitempty"`
}

type File struct {
	ActiveProfile string             `yaml:"active_profile"`
	Profiles      map[string]Profile `yaml:"profiles"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".vobiz", "config.yaml"), nil
}

func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &File{Profiles: map[string]Profile{}}, nil
		}
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	return &f, nil
}

func Save(path string, f *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/config/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat(config): YAML config with named profiles and 0600 mode"
```

---

## Task 5: Add `internal/auth` credential resolver (flag > env > profile)

**Files:**
- Create: `internal/auth/auth.go`
- Create: `internal/auth/auth_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/auth/auth_test.go`:
```go
package auth

import (
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func TestResolve_FlagBeatsEnvBeatsProfile(t *testing.T) {
	cfg := &config.File{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {AuthID: "PROF_ID", AuthToken: "PROF_TOK", BaseURL: "https://api.vobiz.ai/api/v1"},
		},
	}

	t.Run("profile when nothing overrides", func(t *testing.T) {
		got, err := Resolve(Inputs{Config: cfg})
		if err != nil {
			t.Fatal(err)
		}
		if got.AuthID != "PROF_ID" || got.AuthToken != "PROF_TOK" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("env beats profile", func(t *testing.T) {
		got, err := Resolve(Inputs{
			Config: cfg,
			EnvID:  "ENV_ID", EnvToken: "ENV_TOK",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.AuthID != "ENV_ID" || got.AuthToken != "ENV_TOK" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("flag beats env", func(t *testing.T) {
		got, err := Resolve(Inputs{
			Config:   cfg,
			EnvID:    "ENV_ID", EnvToken: "ENV_TOK",
			FlagID:   "FLAG_ID", FlagToken: "FLAG_TOK",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.AuthID != "FLAG_ID" || got.AuthToken != "FLAG_TOK" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("named profile override", func(t *testing.T) {
		cfg.Profiles["staging"] = config.Profile{AuthID: "STG_ID", AuthToken: "STG_TOK"}
		got, err := Resolve(Inputs{Config: cfg, Profile: "staging"})
		if err != nil {
			t.Fatal(err)
		}
		if got.AuthID != "STG_ID" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("missing returns ErrAuth", func(t *testing.T) {
		_, err := Resolve(Inputs{Config: &config.File{Profiles: map[string]config.Profile{}}})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestResolve_DefaultsBaseURL(t *testing.T) {
	cfg := &config.File{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {AuthID: "X", AuthToken: "Y"},
		},
	}
	got, _ := Resolve(Inputs{Config: cfg})
	if got.BaseURL != "https://api.vobiz.ai/api/v1" {
		t.Fatalf("BaseURL = %q", got.BaseURL)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/auth/...
```
Expected: build failure.

- [ ] **Step 3: Write the package**

Create `internal/auth/auth.go`:
```go
// Package auth resolves Vobiz credentials from flags, env vars, and config.
package auth

import (
	"fmt"

	cliErrors "github.com/yash-kavaiya/vobiz-cli/internal/errors"
	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

const DefaultBaseURL = "https://api.vobiz.ai/api/v1"

type Credentials struct {
	AuthID    string
	AuthToken string
	BaseURL   string
	Source    string // "flag" | "env" | "profile:<name>"
}

type Inputs struct {
	Config             *config.File
	Profile            string
	FlagID, FlagToken  string
	EnvID, EnvToken    string
	FlagBaseURL        string
}

func Resolve(in Inputs) (Credentials, error) {
	c := Credentials{BaseURL: DefaultBaseURL}

	switch {
	case in.FlagID != "" && in.FlagToken != "":
		c.AuthID, c.AuthToken, c.Source = in.FlagID, in.FlagToken, "flag"
	case in.EnvID != "" && in.EnvToken != "":
		c.AuthID, c.AuthToken, c.Source = in.EnvID, in.EnvToken, "env"
	default:
		if in.Config == nil {
			return c, fmt.Errorf("%w: no credentials supplied (run 'vobiz auth login')", cliErrors.ErrAuth)
		}
		name := in.Profile
		if name == "" {
			name = in.Config.ActiveProfile
		}
		if name == "" {
			name = "default"
		}
		p, ok := in.Config.Profiles[name]
		if !ok {
			return c, fmt.Errorf("%w: profile %q not found (run 'vobiz auth login --profile %s')", cliErrors.ErrAuth, name, name)
		}
		if p.AuthID == "" || p.AuthToken == "" {
			return c, fmt.Errorf("%w: profile %q is missing auth_id or auth_token", cliErrors.ErrAuth, name)
		}
		c.AuthID, c.AuthToken, c.Source = p.AuthID, p.AuthToken, "profile:"+name
		if p.BaseURL != "" {
			c.BaseURL = p.BaseURL
		}
	}

	if in.FlagBaseURL != "" {
		c.BaseURL = in.FlagBaseURL
	}
	return c, nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/auth/... ./internal/errors/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/
git commit -m "feat(auth): credential resolver with flag>env>profile precedence"
```

---

## Task 6: Add `internal/output` renderers (table | json | yaml)

**Files:**
- Create: `internal/output/output.go`
- Create: `internal/output/output_test.go`

- [ ] **Step 1: Add table dependency**

```bash
go get github.com/jedib0t/go-pretty/v6/table
```

- [ ] **Step 2: Write the failing test**

Create `internal/output/output_test.go`:
```go
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
	if !strings.Contains(buf.String(), "- ID: t1") {
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
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/output/...
```
Expected: build failure.

- [ ] **Step 4: Write the package**

Create `internal/output/output.go`:
```go
// Package output renders any slice/struct as table, JSON, or YAML.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"gopkg.in/yaml.v3"
)

type Format int

const (
	FormatTable Format = iota
	FormatJSON
	FormatYAML
)

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "", "table":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	default:
		return 0, fmt.Errorf("unknown output format %q (use table|json|yaml)", s)
	}
}

type Column struct {
	Header string
	Field  string // struct field name (top-level)
}

// Render writes rows in the chosen format. `rows` must be a slice or a single struct.
func Render(w io.Writer, rows any, cols []Column, f Format) error {
	switch f {
	case FormatJSON:
		return writeJSON(w, rows)
	case FormatYAML:
		return writeYAML(w, rows)
	default:
		return writeTable(w, rows, cols)
	}
}

func writeJSON(w io.Writer, rows any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func writeYAML(w io.Writer, rows any) error {
	b, err := yaml.Marshal(rows)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func writeTable(w io.Writer, rows any, cols []Column) error {
	v := reflect.ValueOf(rows)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		// single struct → wrap in slice for uniform handling
		slice := reflect.MakeSlice(reflect.SliceOf(v.Type()), 1, 1)
		slice.Index(0).Set(v)
		v = slice
	}

	t := table.NewWriter()
	t.SetOutputMirror(w)

	headers := make(table.Row, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}
	t.AppendHeader(headers)

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		row := make(table.Row, len(cols))
		for j, c := range cols {
			fv := item.FieldByName(c.Field)
			if !fv.IsValid() {
				row[j] = ""
				continue
			}
			row[j] = fv.Interface()
		}
		t.AppendRow(row)
	}

	t.Render()
	return nil
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/output/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/output/ go.mod go.sum
git commit -m "feat(output): table/json/yaml renderer with column specs"
```

---

## Task 7: Add `internal/paginate` generic pager

**Files:**
- Create: `internal/paginate/paginate.go`
- Create: `internal/paginate/paginate_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/paginate/paginate_test.go`:
```go
package paginate

import (
	"context"
	"testing"
)

func TestAll_StopsWhenHasMoreFalse(t *testing.T) {
	pages := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7},
	}
	idx := 0
	fetch := func(ctx context.Context, cursor string) (Page[int], error) {
		p := Page[int]{Items: pages[idx], NextCursor: ""}
		idx++
		if idx < len(pages) {
			p.NextCursor = "more"
		}
		return p, nil
	}

	got, err := All(context.Background(), fetch)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{1, 2, 3, 4, 5, 6, 7}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestAll_RespectsLimit(t *testing.T) {
	fetch := func(ctx context.Context, cursor string) (Page[int], error) {
		return Page[int]{Items: []int{1, 2, 3, 4, 5}, NextCursor: "more"}, nil
	}
	got, err := AllN(context.Background(), fetch, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got len %d want 3", len(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/paginate/...
```
Expected: build failure.

- [ ] **Step 3: Write the package**

Create `internal/paginate/paginate.go`:
```go
// Package paginate provides a generic cursor pager.
package paginate

import "context"

type Page[T any] struct {
	Items      []T
	NextCursor string // empty when no more pages
}

type Fetcher[T any] func(ctx context.Context, cursor string) (Page[T], error)

// All fetches every page until NextCursor is empty.
func All[T any](ctx context.Context, fetch Fetcher[T]) ([]T, error) {
	return AllN(ctx, fetch, -1)
}

// AllN fetches pages until either no more pages remain or `limit` items are collected.
// A negative limit means unbounded.
func AllN[T any](ctx context.Context, fetch Fetcher[T], limit int) ([]T, error) {
	var out []T
	cursor := ""
	for {
		p, err := fetch(ctx, cursor)
		if err != nil {
			return nil, err
		}
		out = append(out, p.Items...)
		if limit >= 0 && len(out) >= limit {
			return out[:limit], nil
		}
		if p.NextCursor == "" {
			return out, nil
		}
		cursor = p.NextCursor
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/paginate/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/paginate/
git commit -m "feat(paginate): generic cursor pager with optional limit"
```

---

## Task 8: Add `internal/httpx` retrying HTTP client

**Files:**
- Create: `internal/httpx/idempotency.go`
- Create: `internal/httpx/retry.go`
- Create: `internal/httpx/client.go`
- Create: `internal/httpx/client_test.go`

- [ ] **Step 1: Add uuid dependency**

```bash
go get github.com/google/uuid
```

- [ ] **Step 2: Write the failing test**

Create `internal/httpx/client_test.go`:
```go
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
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/httpx/...
```
Expected: build failure.

- [ ] **Step 4: Write the idempotency helper**

Create `internal/httpx/idempotency.go`:
```go
package httpx

import "github.com/google/uuid"

func newIdempotencyKey() string { return uuid.NewString() }

func isMutation(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}
```

- [ ] **Step 5: Write the retry helper**

Create `internal/httpx/retry.go`:
```go
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
```

- [ ] **Step 6: Write the client**

Create `internal/httpx/client.go`:
```go
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
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
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
```

- [ ] **Step 7: Run tests**

```bash
go test ./internal/httpx/...
```
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/httpx/ go.mod go.sum
git commit -m "feat(httpx): retrying HTTP client with auth headers + idempotency"
```

---

## Task 9: Add `internal/client` resource interfaces and SDK wrapper

**Files:**
- Create: `internal/client/client.go`
- Create: `internal/client/account.go`
- Create: `internal/client/client_test.go`

- [ ] **Step 1: Add the official Vobiz Go SDK**

```bash
go get github.com/vobiz-ai/vobiz-go-sdk
```

If `go get` fails because the upstream module path differs (the docs spell it "Vobiz-Go-SDK"), pin via:
```bash
go mod edit -replace github.com/vobiz-ai/vobiz-go-sdk=github.com/vobiz-ai/Vobiz-Go-SDK@latest
go mod tidy
```

- [ ] **Step 2: Write the failing test**

Create `internal/client/client_test.go`:
```go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
)

func TestAccount_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"auth_id":      "AB12",
			"account_type": "developer",
			"billing_mode": "prepaid",
			"timezone":     "Asia/Kolkata",
			"cash_credits": "25.00",
		})
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Account.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.AuthID != "AB12" || got.AccountType != "developer" || got.CashCredits != "25.00" {
		t.Fatalf("%+v", got)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/client/...
```
Expected: build failure.

- [ ] **Step 4: Write the client struct**

Create `internal/client/client.go`:
```go
// Package client exposes typed resource APIs over the shared httpx client and
// the official Vobiz Go SDK.
package client

import (
	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
	"github.com/yash-kavaiya/vobiz-cli/internal/version"
)

type Client struct {
	HTTP    *httpx.Client
	Account AccountAPI
}

func New(creds auth.Credentials) *Client {
	h := httpx.New(httpx.Config{
		BaseURL:   creds.BaseURL,
		AuthID:    creds.AuthID,
		AuthToken: creds.AuthToken,
		UserAgent: "vobiz-cli/" + version.Version,
	})
	return &Client{
		HTTP:    h,
		Account: &accountAPI{http: h, authID: creds.AuthID},
	}
}
```

- [ ] **Step 5: Write the account resource**

Create `internal/client/account.go`:
```go
package client

import (
	"context"
	"net/http"

	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
)

type Account struct {
	AuthID         string `json:"auth_id"          yaml:"auth_id"`
	AccountType    string `json:"account_type"     yaml:"account_type"`
	BillingMode    string `json:"billing_mode"     yaml:"billing_mode"`
	Timezone       string `json:"timezone"         yaml:"timezone"`
	CashCredits    string `json:"cash_credits"     yaml:"cash_credits"`
	AutoRecharge   bool   `json:"auto_recharge"    yaml:"auto_recharge"`
	ResourceURI    string `json:"resource_uri"     yaml:"resource_uri"`
}

type Transaction struct {
	ID          string `json:"id"           yaml:"id"`
	Amount      string `json:"amount"       yaml:"amount"`
	Description string `json:"description"  yaml:"description"`
	CreatedAt   string `json:"created_at"   yaml:"created_at"`
}

type Concurrency struct {
	Limit   int `json:"limit"    yaml:"limit"`
	Current int `json:"current"  yaml:"current"`
}

type AccountAPI interface {
	Get(ctx context.Context) (*Account, error)
	Balance(ctx context.Context) (string, error)
	Transactions(ctx context.Context, cursor string, limit int) ([]Transaction, string, error)
	Concurrency(ctx context.Context) (*Concurrency, error)
}

type accountAPI struct {
	http   *httpx.Client
	authID string
}

func (a *accountAPI) Get(ctx context.Context) (*Account, error) {
	var out Account
	if err := a.http.DoJSON(ctx, http.MethodGet, "/Account/"+a.authID+"/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *accountAPI) Balance(ctx context.Context) (string, error) {
	acc, err := a.Get(ctx)
	if err != nil {
		return "", err
	}
	return acc.CashCredits, nil
}

func (a *accountAPI) Transactions(ctx context.Context, cursor string, limit int) ([]Transaction, string, error) {
	path := "/Account/" + a.authID + "/Transaction/"
	if cursor != "" {
		path += "?cursor=" + cursor
	}
	var raw struct {
		Objects []Transaction `json:"objects"`
		Meta    struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	if err := a.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, "", err
	}
	return raw.Objects, raw.Meta.Next, nil
}

func (a *accountAPI) Concurrency(ctx context.Context) (*Concurrency, error) {
	var out Concurrency
	if err := a.http.DoJSON(ctx, http.MethodGet, "/Account/"+a.authID+"/Concurrency/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/client/...
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/client/ go.mod go.sum
git commit -m "feat(client): typed Account resource over httpx"
```

---

## Task 10: Scaffold `main.go` and `cmd/root.go` with global flags

**Files:**
- Create: `main.go`
- Create: `cmd/root.go`
- Create: `cmd/root_test.go`

- [ ] **Step 1: Add cobra dependency**

```bash
go get github.com/spf13/cobra
```

- [ ] **Step 2: Write the failing test**

Create `cmd/root_test.go`:
```go
package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRoot_HelpListsTopLevelCommands(t *testing.T) {
	var buf bytes.Buffer
	root := New()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"vobiz", "auth", "account", "docs", "version", "completion"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q:\n%s", want, out)
		}
	}
}

func TestRoot_OutputFlagParses(t *testing.T) {
	root := New()
	root.SetArgs([]string{"--output", "json", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if globalOutput != "json" {
		t.Fatalf("globalOutput = %q", globalOutput)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./cmd/...
```
Expected: build failure.

- [ ] **Step 4: Write `main.go`**

Create `main.go`:
```go
package main

import (
	"fmt"
	"os"

	"github.com/yash-kavaiya/vobiz-cli/cmd"
	cliErrors "github.com/yash-kavaiya/vobiz-cli/internal/errors"
)

func main() {
	root := cmd.New()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(cliErrors.ExitCode(err))
	}
}
```

- [ ] **Step 5: Write `cmd/root.go`**

Create `cmd/root.go`:
```go
// Package cmd wires the Cobra command tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Global flag-backing vars. Tests may read these to assert wiring.
var (
	globalOutput   string
	globalProfile  string
	globalAuthID   string
	globalAuthTok  string
	globalBaseURL  string
	globalVerbose  bool
	globalNoColor  bool
)

// New constructs the root *cobra.Command. Subcommands are added by their
// respective packages' Register functions (see cmd/auth, cmd/account, cmd/docs).
func New() *cobra.Command {
	root := &cobra.Command{
		Use:           "vobiz",
		Short:         "Vobiz CLI — programmable telephony from your terminal",
		Long:          "The unofficial-but-friendly terminal interface for the Vobiz programmable-telephony platform.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	pf := root.PersistentFlags()
	pf.StringVarP(&globalOutput, "output", "o", "table", "output format: table|json|yaml")
	pf.StringVar(&globalProfile, "profile", "", "named profile from ~/.vobiz/config.yaml")
	pf.StringVar(&globalAuthID, "auth-id", "", "override Auth ID (env VOBIZ_AUTH_ID)")
	pf.StringVar(&globalAuthTok, "auth-token", "", "override Auth Token (env VOBIZ_AUTH_TOKEN)")
	pf.StringVar(&globalBaseURL, "base-url", "", "override API base URL")
	pf.BoolVarP(&globalVerbose, "verbose", "v", false, "verbose output")
	pf.BoolVar(&globalNoColor, "no-color", false, "disable color output")

	registerVersion(root)
	registerCompletion(root)
	registerAuth(root)
	registerAccount(root)
	registerDocs(root)

	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	return root
}
```

- [ ] **Step 6: Add temporary registration stubs so the package compiles**

Create `cmd/registrations.go` (this file will be replaced piecewise in later tasks):
```go
package cmd

import "github.com/spf13/cobra"

func registerVersion(_ *cobra.Command)    {}
func registerCompletion(_ *cobra.Command) {}
func registerAuth(_ *cobra.Command)       {}
func registerAccount(_ *cobra.Command)    {}
func registerDocs(_ *cobra.Command)       {}
```

- [ ] **Step 7: Adjust the test — `auth`/`account`/`docs` are not yet registered**

Replace the entire contents of `cmd/root_test.go`:
```go
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
	if !strings.Contains(buf.String(), "vobiz") {
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
```

- [ ] **Step 8: Build and run**

```bash
go build ./...
go test ./cmd/...
```
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add main.go cmd/ go.mod go.sum
git commit -m "feat(cmd): cobra root with global flags and registration stubs"
```

---

## Task 11: Implement `vobiz version` and `vobiz completion`

**Files:**
- Modify: `cmd/registrations.go` (replace `registerVersion`, `registerCompletion`)
- Create: `cmd/version.go`
- Create: `cmd/completion.go`
- Create: `cmd/version_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/version_test.go`:
```go
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
		t.Fatalf("completion bash output unexpected (no 'complete' text): %q", buf.String()[:min(200, len(buf.String()))])
	}
}

func min(a, b int) int { if a < b { return a }; return b }
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/ -run Version
```
Expected: FAIL — `version` subcommand not registered.

- [ ] **Step 3: Write `cmd/version.go`**

Create `cmd/version.go`:
```go
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, and build date",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(version.String())
			return nil
		},
	}
}
```

- [ ] **Step 4: Write `cmd/completion.go`**

Create `cmd/completion.go`:
```go
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion script",
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(out)
			}
			_, _ = os.Stderr.Write(nil) // unreachable
			return nil
		},
	}
}
```

- [ ] **Step 5: Wire the registrations**

Replace `cmd/registrations.go`:
```go
package cmd

import "github.com/spf13/cobra"

func registerVersion(root *cobra.Command)    { root.AddCommand(newVersionCmd()) }
func registerCompletion(root *cobra.Command) { root.AddCommand(newCompletionCmd(root)) }
func registerAuth(_ *cobra.Command)          {}
func registerAccount(_ *cobra.Command)       {}
func registerDocs(_ *cobra.Command)          {}
```

- [ ] **Step 6: Run tests**

```bash
go test ./cmd/...
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/
git commit -m "feat(cmd): version and completion subcommands"
```

---

## Task 12: `vobiz auth login` — interactive credential capture and verification

**Files:**
- Create: `cmd/auth/auth.go`
- Create: `cmd/auth/login.go`
- Create: `cmd/auth/login_test.go`
- Modify: `cmd/registrations.go`

- [ ] **Step 1: Add the terminal masking dependency**

```bash
go get golang.org/x/term
```

- [ ] **Step 2: Write the failing test**

Create `cmd/auth/login_test.go`:
```go
package auth

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

type fakeAccountAPI struct {
	getErr error
	called bool
}

func (f *fakeAccountAPI) Get(_ context.Context) (*client.Account, error) {
	f.called = true
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &client.Account{AuthID: "AB12", AccountType: "developer"}, nil
}
func (f *fakeAccountAPI) Balance(_ context.Context) (string, error) { return "", nil }
func (f *fakeAccountAPI) Transactions(_ context.Context, _ string, _ int) ([]client.Transaction, string, error) {
	return nil, "", nil
}
func (f *fakeAccountAPI) Concurrency(_ context.Context) (*client.Concurrency, error) {
	return nil, nil
}

func TestRunLogin_WritesConfigAndVerifies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	fake := &fakeAccountAPI{}
	var out bytes.Buffer
	err := runLogin(loginInputs{
		ConfigPath:  path,
		Profile:     "default",
		AuthID:      "AB12",
		AuthToken:   "tok",
		BaseURL:     "https://api.vobiz.ai/api/v1",
		Out:         &out,
		VerifyAcct:  func(_ string) accountVerifier { return fake },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !fake.called {
		t.Fatal("verifier not called")
	}
	if !strings.Contains(out.String(), "saved") {
		t.Fatalf("output: %q", out.String())
	}
	f, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Profiles["default"].AuthID != "AB12" {
		t.Fatalf("config: %+v", f)
	}
	if f.ActiveProfile != "default" {
		t.Fatalf("active profile = %q", f.ActiveProfile)
	}
}

func TestRunLogin_VerificationFailureDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	fake := &fakeAccountAPI{getErr: errors.New("401 unauthorized")}
	var out bytes.Buffer
	err := runLogin(loginInputs{
		ConfigPath: path,
		Profile:    "default",
		AuthID:     "AB12",
		AuthToken:  "bad",
		Out:        &out,
		VerifyAcct: func(_ string) accountVerifier { return fake },
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := config.Load(path); statErr != nil {
		// Either non-existent or empty — both fine.
	}
	f, _ := config.Load(path)
	if _, ok := f.Profiles["default"]; ok {
		t.Fatal("profile should not have been written on verification failure")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./cmd/auth/...
```
Expected: build failure.

- [ ] **Step 4: Write the shared package file**

Create `cmd/auth/auth.go`:
```go
// Package auth implements `vobiz auth …` subcommands.
package auth

import "github.com/spf13/cobra"

// Register adds `auth` and its children to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Vobiz credentials and profiles",
	}
	cmd.AddCommand(newLoginCmd())
	parent.AddCommand(cmd)
}
```

- [ ] **Step 5: Write `cmd/auth/login.go`**

Create `cmd/auth/login.go`:
```go
package auth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	cliAuth "github.com/yash-kavaiya/vobiz-cli/internal/auth"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

type accountVerifier interface {
	Get(ctx context.Context) (*client.Account, error)
}

type loginInputs struct {
	ConfigPath string
	Profile    string
	AuthID     string
	AuthToken  string
	BaseURL    string
	Out        io.Writer
	VerifyAcct func(authID string) accountVerifier
}

func newLoginCmd() *cobra.Command {
	var (
		profile string
		baseURL string
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save Vobiz Auth ID + Token to ~/.vobiz/config.yaml",
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, tok, err := promptCredentials(cmd.InOrStdin(), cmd.OutOrStdout())
			if err != nil {
				return err
			}
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runLogin(loginInputs{
				ConfigPath: path,
				Profile:    profile,
				AuthID:     id,
				AuthToken:  tok,
				BaseURL:    baseURL,
				Out:        cmd.OutOrStdout(),
				VerifyAcct: func(authID string) accountVerifier {
					c := client.New(cliAuth.Credentials{AuthID: authID, AuthToken: tok, BaseURL: pickBaseURL(baseURL)})
					return c.Account
				},
			})
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "default", "profile name to save under")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "override base URL")
	return cmd
}

func promptCredentials(in io.Reader, out io.Writer) (string, string, error) {
	fmt.Fprint(out, "Auth ID: ")
	r := bufio.NewReader(in)
	idLine, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", "", err
	}
	id := strings.TrimSpace(idLine)
	if id == "" {
		return "", "", errors.New("Auth ID is required")
	}

	fmt.Fprint(out, "Auth Token: ")
	var tok string
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return "", "", err
		}
		tok = strings.TrimSpace(string(b))
	} else {
		line, err := r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", "", err
		}
		tok = strings.TrimSpace(line)
	}
	if tok == "" {
		return "", "", errors.New("Auth Token is required")
	}
	return id, tok, nil
}

func pickBaseURL(flag string) string {
	if flag != "" {
		return flag
	}
	return cliAuth.DefaultBaseURL
}

func runLogin(in loginInputs) error {
	verifier := in.VerifyAcct(in.AuthID)
	if _, err := verifier.Get(context.Background()); err != nil {
		return fmt.Errorf("credentials rejected by API: %w", err)
	}

	f, err := config.Load(in.ConfigPath)
	if err != nil {
		return err
	}
	if f.Profiles == nil {
		f.Profiles = map[string]config.Profile{}
	}
	name := in.Profile
	if name == "" {
		name = "default"
	}
	f.Profiles[name] = config.Profile{
		AuthID:    in.AuthID,
		AuthToken: in.AuthToken,
		BaseURL:   in.BaseURL,
	}
	if f.ActiveProfile == "" {
		f.ActiveProfile = name
	}
	if err := config.Save(in.ConfigPath, f); err != nil {
		return err
	}
	fmt.Fprintf(in.Out, "Credentials saved to %s (profile %q).\n", in.ConfigPath, name)
	return nil
}
```

- [ ] **Step 6: Update registrations**

Replace the `registerAuth` line in `cmd/registrations.go`:
```go
func registerAuth(root *cobra.Command) { auth.Register(root) }
```

Add the import:
```go
import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/auth"
)
```

- [ ] **Step 7: Run tests**

```bash
go test ./cmd/auth/... ./cmd/...
```
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add cmd/auth/ cmd/registrations.go go.mod go.sum
git commit -m "feat(auth): login command with masked token input and API verification"
```

---

## Task 13: `vobiz auth logout`, `auth status`, `auth profile`

**Files:**
- Create: `cmd/auth/logout.go`
- Create: `cmd/auth/status.go`
- Create: `cmd/auth/profile.go`
- Create: `cmd/auth/logout_test.go`
- Modify: `cmd/auth/auth.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/auth/logout_test.go`:
```go
package auth

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func writeCfg(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	f := &config.File{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {AuthID: "A", AuthToken: "T"},
			"staging": {AuthID: "B", AuthToken: "U"},
		},
	}
	if err := config.Save(path, f); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunLogout_RemovesActiveProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)

	var out bytes.Buffer
	if err := runLogout(path, "", &out); err != nil {
		t.Fatal(err)
	}
	f, _ := config.Load(path)
	if _, ok := f.Profiles["default"]; ok {
		t.Fatal("default profile should be removed")
	}
	if f.ActiveProfile == "default" {
		t.Fatal("active profile should be cleared or rotated")
	}
}

func TestRunLogout_NamedProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)

	if err := runLogout(path, "staging", new(bytes.Buffer)); err != nil {
		t.Fatal(err)
	}
	f, _ := config.Load(path)
	if _, ok := f.Profiles["staging"]; ok {
		t.Fatal("staging profile should be removed")
	}
	if f.ActiveProfile != "default" {
		t.Fatalf("active = %q", f.ActiveProfile)
	}
}

func TestRunStatus_PrintsActiveProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)

	var out bytes.Buffer
	if err := runStatus(path, &out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("default")) {
		t.Fatalf("status: %q", out.String())
	}
}

func TestRunProfileList(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)
	var out bytes.Buffer
	if err := runProfileList(path, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"default", "staging"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("profile list missing %q: %s", want, out.String())
		}
	}
}

func TestRunProfileUse(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)
	if err := runProfileUse(path, "staging"); err != nil {
		t.Fatal(err)
	}
	f, _ := config.Load(path)
	if f.ActiveProfile != "staging" {
		t.Fatalf("active = %q", f.ActiveProfile)
	}
}

func TestRunProfileRm(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir)
	if err := runProfileRm(path, "staging"); err != nil {
		t.Fatal(err)
	}
	f, _ := config.Load(path)
	if _, ok := f.Profiles["staging"]; ok {
		t.Fatal("staging should be removed")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/auth/...
```
Expected: build failure.

- [ ] **Step 3: Write logout**

Create `cmd/auth/logout.go`:
```go
package auth

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func newLogoutCmd() *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove credentials for a profile (default: active profile)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runLogout(path, profile, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "profile to remove (default: active)")
	return cmd
}

func runLogout(path, profile string, out io.Writer) error {
	f, err := config.Load(path)
	if err != nil {
		return err
	}
	name := profile
	if name == "" {
		name = f.ActiveProfile
	}
	if _, ok := f.Profiles[name]; !ok {
		return fmt.Errorf("no profile named %q", name)
	}
	delete(f.Profiles, name)

	if f.ActiveProfile == name {
		f.ActiveProfile = ""
		for k := range f.Profiles {
			f.ActiveProfile = k
			break
		}
	}
	if err := config.Save(path, f); err != nil {
		return err
	}
	fmt.Fprintf(out, "Removed profile %q.\n", name)
	return nil
}
```

- [ ] **Step 4: Write status**

Create `cmd/auth/status.go`:
```go
package auth

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show active profile and stored Auth ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runStatus(path, cmd.OutOrStdout())
		},
	}
}

func runStatus(path string, out io.Writer) error {
	f, err := config.Load(path)
	if err != nil {
		return err
	}
	if f.ActiveProfile == "" {
		fmt.Fprintln(out, "No active profile. Run 'vobiz auth login'.")
		return nil
	}
	p := f.Profiles[f.ActiveProfile]
	fmt.Fprintf(out, "Active profile: %s\nAuth ID:        %s\nBase URL:       %s\nConfig file:    %s\n",
		f.ActiveProfile, p.AuthID, fallback(p.BaseURL, "https://api.vobiz.ai/api/v1"), path)
	return nil
}

func fallback(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
```

- [ ] **Step 5: Write profile**

Create `cmd/auth/profile.go`:
```go
package auth

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "List, switch, or remove named profiles",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runProfileList(path, cmd.OutOrStdout())
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "use <name>",
		Short: "Set the active profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runProfileUse(path, args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a named profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			return runProfileRm(path, args[0])
		},
	})
	return cmd
}

func runProfileList(path string, out io.Writer) error {
	f, err := config.Load(path)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(f.Profiles))
	for k := range f.Profiles {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, n := range names {
		marker := " "
		if n == f.ActiveProfile {
			marker = "*"
		}
		fmt.Fprintf(out, "%s %s\n", marker, n)
	}
	return nil
}

func runProfileUse(path, name string) error {
	f, err := config.Load(path)
	if err != nil {
		return err
	}
	if _, ok := f.Profiles[name]; !ok {
		return fmt.Errorf("no profile named %q", name)
	}
	f.ActiveProfile = name
	return config.Save(path, f)
}

func runProfileRm(path, name string) error {
	f, err := config.Load(path)
	if err != nil {
		return err
	}
	if _, ok := f.Profiles[name]; !ok {
		return fmt.Errorf("no profile named %q", name)
	}
	delete(f.Profiles, name)
	if f.ActiveProfile == name {
		f.ActiveProfile = ""
	}
	return config.Save(path, f)
}
```

- [ ] **Step 6: Wire them into `auth.Register`**

Replace `cmd/auth/auth.go`:
```go
// Package auth implements `vobiz auth …` subcommands.
package auth

import "github.com/spf13/cobra"

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Vobiz credentials and profiles",
	}
	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newProfileCmd())
	parent.AddCommand(cmd)
}
```

- [ ] **Step 7: Run tests**

```bash
go test ./cmd/auth/...
```
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add cmd/auth/
git commit -m "feat(auth): logout, status, and profile {list,use,rm}"
```

---

## Task 14: Account command tree (`get`, `balance`, `transactions`, `concurrency`)

**Files:**
- Create: `cmd/account/account.go`
- Create: `cmd/account/get.go`
- Create: `cmd/account/balance.go`
- Create: `cmd/account/transactions.go`
- Create: `cmd/account/concurrency.go`
- Create: `cmd/account/account_test.go`
- Modify: `cmd/registrations.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/account/account_test.go`:
```go
package account

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

type fakeAccount struct {
	acc          *client.Account
	txs          []client.Transaction
	next         string
	concurrency  *client.Concurrency
	wasCalled    string
}

func (f *fakeAccount) Get(_ context.Context) (*client.Account, error) {
	f.wasCalled = "get"
	return f.acc, nil
}
func (f *fakeAccount) Balance(_ context.Context) (string, error) {
	f.wasCalled = "balance"
	return f.acc.CashCredits, nil
}
func (f *fakeAccount) Transactions(_ context.Context, _ string, _ int) ([]client.Transaction, string, error) {
	f.wasCalled = "transactions"
	return f.txs, f.next, nil
}
func (f *fakeAccount) Concurrency(_ context.Context) (*client.Concurrency, error) {
	f.wasCalled = "concurrency"
	return f.concurrency, nil
}

func TestGet_TableOutput(t *testing.T) {
	f := &fakeAccount{acc: &client.Account{AuthID: "AB12", AccountType: "developer", BillingMode: "prepaid", CashCredits: "25.00", Timezone: "Asia/Kolkata"}}
	var out bytes.Buffer
	if err := runGet(f, &out, "table"); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"AB12", "developer", "prepaid", "25.00"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q in:\n%s", w, out.String())
		}
	}
}

func TestGet_JSONOutput(t *testing.T) {
	f := &fakeAccount{acc: &client.Account{AuthID: "AB12", AccountType: "developer", CashCredits: "1.50"}}
	var out bytes.Buffer
	if err := runGet(f, &out, "json"); err != nil {
		t.Fatal(err)
	}
	var got client.Account
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("not valid json: %v\n%s", err, out.String())
	}
	if got.AuthID != "AB12" || got.CashCredits != "1.50" {
		t.Fatalf("decoded = %+v", got)
	}
}

func TestBalance_Prints(t *testing.T) {
	f := &fakeAccount{acc: &client.Account{CashCredits: "12.34"}}
	var out bytes.Buffer
	if err := runBalance(f, &out); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "12.34" {
		t.Fatalf("balance: %q", out.String())
	}
}

func TestTransactions_Pages(t *testing.T) {
	f := &fakeAccount{txs: []client.Transaction{
		{ID: "1", Amount: "10", Description: "topup", CreatedAt: "2026-05-23"},
		{ID: "2", Amount: "-1", Description: "call", CreatedAt: "2026-05-23"},
	}}
	var out bytes.Buffer
	if err := runTransactions(f, &out, "table", 10, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "topup") {
		t.Fatalf("missing topup:\n%s", out.String())
	}
}

func TestConcurrency_Prints(t *testing.T) {
	f := &fakeAccount{concurrency: &client.Concurrency{Limit: 50, Current: 3}}
	var out bytes.Buffer
	if err := runConcurrency(f, &out, "table"); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"50", "3"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q:\n%s", w, out.String())
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/account/...
```
Expected: build failure.

- [ ] **Step 3: Write the shared package file**

Create `cmd/account/account.go`:
```go
// Package account implements `vobiz account …` subcommands.
package account

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	cliAuth "github.com/yash-kavaiya/vobiz-cli/internal/auth"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

// AccountFactory is replaced in tests; in production it builds a real client.
var AccountFactory = func() (client.AccountAPI, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	creds, err := cliAuth.Resolve(cliAuth.Inputs{Config: cfg})
	if err != nil {
		return nil, err
	}
	return client.New(creds).Account, nil
}

func mustAccount() client.AccountAPI {
	a, err := AccountFactory()
	if err != nil {
		panic(fmt.Sprintf("account factory: %v", err))
	}
	return a
}

// Register adds the `account` subtree.
func Register(parent *cobra.Command, format func() string) {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage your Vobiz account",
	}
	cmd.AddCommand(newGetCmd(format))
	cmd.AddCommand(newBalanceCmd())
	cmd.AddCommand(newTransactionsCmd(format))
	cmd.AddCommand(newConcurrencyCmd(format))
	parent.AddCommand(cmd)
}

// usable from tests — currently unused but kept for future symmetry
var _ context.Context = nil
```

- [ ] **Step 4: Write `get`**

Create `cmd/account/get.go`:
```go
package account

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newGetCmd(format func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show account details",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := AccountFactory()
			if err != nil {
				return err
			}
			return runGet(a, cmd.OutOrStdout(), format())
		},
	}
}

func runGet(api client.AccountAPI, w io.Writer, format string) error {
	acc, err := api.Get(context.Background())
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "AUTH ID", Field: "AuthID"},
		{Header: "TYPE", Field: "AccountType"},
		{Header: "BILLING", Field: "BillingMode"},
		{Header: "CREDITS", Field: "CashCredits"},
		{Header: "TZ", Field: "Timezone"},
	}
	return output.Render(w, []client.Account{*acc}, cols, f)
}
```

- [ ] **Step 5: Write `balance`**

Create `cmd/account/balance.go`:
```go
package account

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

func newBalanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "balance",
		Short: "Show current account balance",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := AccountFactory()
			if err != nil {
				return err
			}
			return runBalance(a, cmd.OutOrStdout())
		},
	}
}

func runBalance(api client.AccountAPI, w io.Writer) error {
	b, err := api.Balance(context.Background())
	if err != nil {
		return err
	}
	fmt.Fprintln(w, b)
	return nil
}
```

- [ ] **Step 6: Write `transactions`**

Create `cmd/account/transactions.go`:
```go
package account

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
	"github.com/yash-kavaiya/vobiz-cli/internal/paginate"
)

func newTransactionsCmd(format func() string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transactions",
		Short: "List account transactions",
	}
	var (
		limit int
		all   bool
	)
	list := &cobra.Command{
		Use:   "list",
		Short: "List transactions (paginated)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := AccountFactory()
			if err != nil {
				return err
			}
			return runTransactions(a, cmd.OutOrStdout(), format(), limit, all)
		},
	}
	list.Flags().IntVar(&limit, "limit", 50, "max number of rows")
	list.Flags().BoolVar(&all, "all", false, "fetch all pages")
	cmd.AddCommand(list)
	return cmd
}

func runTransactions(api client.AccountAPI, w io.Writer, format string, limit int, all bool) error {
	fetch := func(ctx context.Context, cursor string) (paginate.Page[client.Transaction], error) {
		items, next, err := api.Transactions(ctx, cursor, limit)
		if err != nil {
			return paginate.Page[client.Transaction]{}, err
		}
		return paginate.Page[client.Transaction]{Items: items, NextCursor: next}, nil
	}

	var (
		rows []client.Transaction
		err  error
	)
	if all {
		rows, err = paginate.All(context.Background(), fetch)
	} else {
		rows, err = paginate.AllN(context.Background(), fetch, limit)
	}
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "ID", Field: "ID"},
		{Header: "AMOUNT", Field: "Amount"},
		{Header: "DESCRIPTION", Field: "Description"},
		{Header: "DATE", Field: "CreatedAt"},
	}
	return output.Render(w, rows, cols, f)
}
```

- [ ] **Step 7: Write `concurrency`**

Create `cmd/account/concurrency.go`:
```go
package account

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newConcurrencyCmd(format func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "concurrency",
		Short: "Show concurrent-call limits and current usage",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := AccountFactory()
			if err != nil {
				return err
			}
			return runConcurrency(a, cmd.OutOrStdout(), format())
		},
	}
}

func runConcurrency(api client.AccountAPI, w io.Writer, format string) error {
	c, err := api.Concurrency(context.Background())
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "LIMIT", Field: "Limit"},
		{Header: "CURRENT", Field: "Current"},
	}
	return output.Render(w, []client.Concurrency{*c}, cols, f)
}
```

- [ ] **Step 8: Wire into registrations**

Replace `cmd/registrations.go`:
```go
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/account"
	"github.com/yash-kavaiya/vobiz-cli/cmd/auth"
)

func registerVersion(root *cobra.Command)    { root.AddCommand(newVersionCmd()) }
func registerCompletion(root *cobra.Command) { root.AddCommand(newCompletionCmd(root)) }
func registerAuth(root *cobra.Command)       { auth.Register(root) }
func registerAccount(root *cobra.Command)    { account.Register(root, func() string { return globalOutput }) }
func registerDocs(_ *cobra.Command)          {}
```

- [ ] **Step 9: Run tests**

```bash
go test ./...
```
Expected: PASS (excluding any test that needs the real `AccountFactory` — none here).

- [ ] **Step 10: Commit**

```bash
git add cmd/account/ cmd/registrations.go
git commit -m "feat(account): get, balance, transactions, concurrency subcommands"
```

---

## Task 15: `internal/docsmcp` — Streamable-HTTP MCP client for docs.vobiz.ai/mcp

**Files:**
- Create: `internal/docsmcp/docsmcp.go`
- Create: `internal/docsmcp/docsmcp_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/docsmcp/docsmcp_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/docsmcp/...
```
Expected: build failure.

- [ ] **Step 3: Write the client**

Create `internal/docsmcp/docsmcp.go`:
```go
// Package docsmcp is a minimal Streamable-HTTP JSON-RPC client for the public
// Vobiz docs MCP server at https://docs.vobiz.ai/mcp. It supports only the
// `search` and `fetch` tools, which is all the CLI uses.
package docsmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

type Client struct {
	endpoint string
	http     *http.Client
	id       atomic.Int64
}

const DefaultEndpoint = "https://docs.vobiz.ai/mcp"

func New(endpoint string) *Client {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Client{
		endpoint: endpoint,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

type Result struct {
	Title   string `json:"title"`
	Path    string `json:"path"`
	Snippet string `json:"snippet"`
}

func (c *Client) Search(ctx context.Context, query string) ([]Result, error) {
	raw, err := c.callTextTool(ctx, "search", map[string]any{"query": query})
	if err != nil {
		return nil, err
	}
	var payload struct {
		Results []Result `json:"results"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("docsmcp: decode search payload: %w", err)
	}
	return payload.Results, nil
}

func (c *Client) Fetch(ctx context.Context, path string) (string, error) {
	return c.callTextTool(ctx, "fetch", map[string]any{"path": path})
}

// ---- JSON-RPC plumbing ----

type mcpRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int64         `json:"id"`
	Method  string        `json:"method"`
	Params  mcpToolParams `json:"params"`
}

type mcpToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type mcpResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Result  struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) callTextTool(ctx context.Context, name string, args map[string]any) (string, error) {
	body := mcpRequest{
		JSONRPC: "2.0",
		ID:      c.id.Add(1),
		Method:  "tools/call",
		Params:  mcpToolParams{Name: name, Arguments: args},
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("docsmcp: HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var out mcpResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("docsmcp: decode response: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("docsmcp: %s (code %d)", out.Error.Message, out.Error.Code)
	}
	if len(out.Result.Content) == 0 {
		return "", fmt.Errorf("docsmcp: empty content from tool %q", name)
	}
	return out.Result.Content[0].Text, nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/docsmcp/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/docsmcp/
git commit -m "feat(docsmcp): Streamable-HTTP MCP client for docs.vobiz.ai/mcp"
```

---

## Task 16: `vobiz docs search` and `vobiz docs open`

**Files:**
- Create: `cmd/docs/docs.go`
- Create: `cmd/docs/search.go`
- Create: `cmd/docs/open.go`
- Create: `cmd/docs/docs_test.go`
- Modify: `cmd/registrations.go`

- [ ] **Step 1: Add glamour for markdown rendering**

```bash
go get github.com/charmbracelet/glamour
```

- [ ] **Step 2: Write the failing test**

Create `cmd/docs/docs_test.go`:
```go
package docs

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/docsmcp"
)

type fakeMCP struct {
	results []docsmcp.Result
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
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./cmd/docs/...
```
Expected: build failure.

- [ ] **Step 4: Write the shared package file**

Create `cmd/docs/docs.go`:
```go
// Package docs implements `vobiz docs …` subcommands.
package docs

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/docsmcp"
)

// MCP is the interface satisfied by both the real client and test fakes.
type MCP interface {
	Search(ctx context.Context, query string) ([]docsmcp.Result, error)
	Fetch(ctx context.Context, path string) (string, error)
}

// Factory builds the real client; replaced in tests if needed.
var Factory = func() MCP { return docsmcp.New("") }

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Search and read Vobiz documentation in your terminal",
	}
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newOpenCmd())
	parent.AddCommand(cmd)
}
```

- [ ] **Step 5: Write `search`**

Create `cmd/docs/search.go`:
```go
package docs

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search Vobiz docs",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(Factory(), strings.Join(args, " "), cmd.OutOrStdout())
		},
	}
}

func runSearch(m MCP, query string, w io.Writer) error {
	results, err := m.Search(context.Background(), query)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Fprintln(w, "No results.")
		return nil
	}
	for _, r := range results {
		fmt.Fprintf(w, "• %s  [%s]\n  %s\n\n", r.Title, r.Path, r.Snippet)
	}
	return nil
}
```

- [ ] **Step 6: Write `open`**

Create `cmd/docs/open.go`:
```go
package docs

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <path>",
		Short: "Fetch and render a Vobiz docs page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpen(Factory(), args[0], cmd.OutOrStdout())
		},
	}
}

func runOpen(m MCP, path string, w io.Writer) error {
	md, err := m.Fetch(context.Background(), path)
	if err != nil {
		return err
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		// fall back to raw markdown if glamour can't initialize
		fmt.Fprint(w, md)
		return nil
	}
	rendered, err := r.Render(md)
	if err != nil {
		fmt.Fprint(w, md)
		return nil
	}
	fmt.Fprint(w, rendered)
	return nil
}
```

- [ ] **Step 7: Wire into registrations**

Replace `cmd/registrations.go`:
```go
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/account"
	"github.com/yash-kavaiya/vobiz-cli/cmd/auth"
	"github.com/yash-kavaiya/vobiz-cli/cmd/docs"
)

func registerVersion(root *cobra.Command)    { root.AddCommand(newVersionCmd()) }
func registerCompletion(root *cobra.Command) { root.AddCommand(newCompletionCmd(root)) }
func registerAuth(root *cobra.Command)       { auth.Register(root) }
func registerAccount(root *cobra.Command)    { account.Register(root, func() string { return globalOutput }) }
func registerDocs(root *cobra.Command)       { docs.Register(root) }
```

- [ ] **Step 8: Run tests**

```bash
go test ./...
```
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add cmd/docs/ cmd/registrations.go go.mod go.sum
git commit -m "feat(docs): search and open subcommands backed by docs.vobiz.ai/mcp"
```

---

## Task 17: Wire env vars and explicit credential resolution into command runtime

So far `AccountFactory` only resolves from the config file. This task adds env-var and flag override so the global `--auth-id`/`--auth-token`/`--profile`/`--base-url` flags actually do something.

**Files:**
- Modify: `cmd/account/account.go` (replace `AccountFactory`)
- Create: `cmd/runtime/runtime.go`
- Create: `cmd/runtime/runtime_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/runtime/runtime_test.go`:
```go
package runtime

import (
	"path/filepath"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func TestResolveCreds_PrefersEnvOverConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := config.Save(path, &config.File{
		ActiveProfile: "default",
		Profiles:      map[string]config.Profile{"default": {AuthID: "FILE_ID", AuthToken: "FILE_TOK"}},
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VOBIZ_AUTH_ID", "ENV_ID")
	t.Setenv("VOBIZ_AUTH_TOKEN", "ENV_TOK")

	got, err := ResolveCreds(Overrides{ConfigPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if got.AuthID != "ENV_ID" || got.AuthToken != "ENV_TOK" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveCreds_FlagsBeatEverything(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	_ = config.Save(path, &config.File{Profiles: map[string]config.Profile{}})

	t.Setenv("VOBIZ_AUTH_ID", "ENV_ID")
	t.Setenv("VOBIZ_AUTH_TOKEN", "ENV_TOK")

	got, err := ResolveCreds(Overrides{
		ConfigPath: path,
		FlagID:     "FLAG_ID",
		FlagToken:  "FLAG_TOK",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AuthID != "FLAG_ID" {
		t.Fatalf("got %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/runtime/...
```
Expected: build failure.

- [ ] **Step 3: Write the runtime helper**

Create `cmd/runtime/runtime.go`:
```go
// Package runtime resolves credentials and builds typed clients for command implementations.
package runtime

import (
	"os"

	cliAuth "github.com/yash-kavaiya/vobiz-cli/internal/auth"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

type Overrides struct {
	ConfigPath string
	Profile    string
	FlagID     string
	FlagToken  string
	FlagBaseURL string
}

func ResolveCreds(o Overrides) (cliAuth.Credentials, error) {
	path := o.ConfigPath
	if path == "" {
		var err error
		path, err = config.DefaultPath()
		if err != nil {
			return cliAuth.Credentials{}, err
		}
	}
	f, err := config.Load(path)
	if err != nil {
		return cliAuth.Credentials{}, err
	}
	return cliAuth.Resolve(cliAuth.Inputs{
		Config:      f,
		Profile:     o.Profile,
		FlagID:      o.FlagID,
		FlagToken:   o.FlagToken,
		FlagBaseURL: o.FlagBaseURL,
		EnvID:       os.Getenv("VOBIZ_AUTH_ID"),
		EnvToken:    os.Getenv("VOBIZ_AUTH_TOKEN"),
	})
}

func NewClient(o Overrides) (*client.Client, error) {
	creds, err := ResolveCreds(o)
	if err != nil {
		return nil, err
	}
	return client.New(creds), nil
}
```

- [ ] **Step 4: Rewrite `cmd/account/account.go`**

Replace the entire file `cmd/account/account.go` with:
```go
// Package account implements `vobiz account …` subcommands.
package account

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

// Overrides is populated by Register's PersistentPreRunE so that AccountFactory
// can see the global flag values at the time a subcommand runs.
var Overrides runtime.Overrides

var AccountFactory = func() (client.AccountAPI, error) {
	c, err := runtime.NewClient(Overrides)
	if err != nil {
		return nil, err
	}
	return c.Account, nil
}

func mustAccount() client.AccountAPI {
	a, err := AccountFactory()
	if err != nil {
		panic(fmt.Sprintf("account factory: %v", err))
	}
	return a
}

// Register adds `account` and its children to the parent command.
// `format` returns the current value of the global -o flag; `ov` returns
// the current values of the global credential flags.
func Register(parent *cobra.Command, format func() string, ov func() runtime.Overrides) {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage your Vobiz account",
	}
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		Overrides = ov()
		return nil
	}
	cmd.AddCommand(newGetCmd(format))
	cmd.AddCommand(newBalanceCmd())
	cmd.AddCommand(newTransactionsCmd(format))
	cmd.AddCommand(newConcurrencyCmd(format))
	parent.AddCommand(cmd)
}
```

- [ ] **Step 5: Update `cmd/registrations.go` to pass an Overrides accessor**

Replace `cmd/registrations.go`:
```go
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/account"
	"github.com/yash-kavaiya/vobiz-cli/cmd/auth"
	"github.com/yash-kavaiya/vobiz-cli/cmd/docs"
	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
)

func registerVersion(root *cobra.Command)    { root.AddCommand(newVersionCmd()) }
func registerCompletion(root *cobra.Command) { root.AddCommand(newCompletionCmd(root)) }
func registerAuth(root *cobra.Command)       { auth.Register(root) }
func registerAccount(root *cobra.Command) {
	account.Register(
		root,
		func() string { return globalOutput },
		func() runtime.Overrides {
			return runtime.Overrides{
				Profile:     globalProfile,
				FlagID:      globalAuthID,
				FlagToken:   globalAuthTok,
				FlagBaseURL: globalBaseURL,
			}
		},
	)
}
func registerDocs(root *cobra.Command) { docs.Register(root) }
```

- [ ] **Step 6: Run tests**

```bash
go test ./...
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/runtime/ cmd/account/account.go cmd/registrations.go
git commit -m "feat(runtime): credential resolution from flags, env, and profile"
```

---

## Task 18: `golangci-lint` config and GitHub Actions CI

**Files:**
- Create: `.golangci.yaml`
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Add the lint config**

Create `.golangci.yaml`:
```yaml
run:
  timeout: 5m
  go: "1.22"

linters:
  disable-all: true
  enable:
    - govet
    - errcheck
    - staticcheck
    - revive
    - gosec
    - ineffassign
    - unused
    - misspell
    - gofmt

issues:
  exclude-rules:
    - path: _test\.go
      linters: [gosec, errcheck]
```

- [ ] **Step 2: Add the CI workflow**

Create `.github/workflows/ci.yml`:
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ["1.22", "1.23"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}
          cache: true
      - run: go vet ./...
      - run: go test ./... -race -coverprofile=coverage.txt
      - if: matrix.os == 'ubuntu-latest' && matrix.go == '1.23'
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.59
```

- [ ] **Step 3: Local pre-flight**

```bash
go vet ./...
go test ./... -race
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add .golangci.yaml .github/workflows/ci.yml
git commit -m "ci: lint config and matrix test workflow"
```

---

## Task 19: End-to-end smoke test against `httptest` server

This validates that all layers compose: cobra → runtime → client → httpx → renderer.

**Files:**
- Create: `cmd/smoke_test.go`

- [ ] **Step 1: Write the smoke test**

Create `cmd/smoke_test.go`:
```go
package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/config"
)

func TestSmoke_AccountGetEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-ID") != "AB12" || r.Header.Get("X-Auth-Token") != "tok" {
			t.Errorf("auth headers missing: %+v", r.Header)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"auth_id":      "AB12",
			"account_type": "developer",
			"billing_mode": "prepaid",
			"timezone":     "UTC",
			"cash_credits": "100.00",
		})
	}))
	defer srv.Close()

	// Point the CLI at a temp HOME so config.DefaultPath() finds our test file.
	// os.UserHomeDir() honors HOME on POSIX and USERPROFILE on Windows.
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	if err := config.Save(filepath.Join(dir, ".vobiz", "config.yaml"), &config.File{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL},
		},
	}); err != nil {
		t.Fatal(err)
	}

	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"account", "get", "--output", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), `"AuthID": "AB12"`) {
		t.Fatalf("smoke output unexpected:\n%s", buf.String())
	}
}
```

- [ ] **Step 2: Run the smoke test**

```bash
go test ./cmd/ -run Smoke -v
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/smoke_test.go
git commit -m "test(smoke): end-to-end account get via real cobra tree"
```

---

## Task 20: README usage section + first manual verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Build the binary**

```bash
go build -o vobiz ./...
./vobiz version
```
Expected: prints `vobiz dev (commit unknown, built unknown)` or similar.

- [ ] **Step 2: Expand the README**

Append to `README.md`:
````markdown
## Install (from source, for now)

```bash
go install github.com/yash-kavaiya/vobiz-cli@latest
```

## First steps

```bash
vobiz auth login                    # paste your Vobiz Auth ID and Token
vobiz account get
vobiz account balance
vobiz account transactions list --limit 20
vobiz docs search "sip trunk"
vobiz docs open /trunks
```

## Profiles

```bash
vobiz auth login --profile staging
vobiz auth profile list
vobiz auth profile use staging
vobiz --profile staging account balance
```

## Environment variables

| Variable           | Purpose                                |
| ------------------ | -------------------------------------- |
| `VOBIZ_AUTH_ID`    | Override Auth ID (beats config)        |
| `VOBIZ_AUTH_TOKEN` | Override Auth Token (beats config)     |
| `NO_COLOR`         | Disable ANSI color in output           |

## Output formats

```bash
vobiz account get -o table    # default
vobiz account get -o json
vobiz account get -o yaml
```

## Roadmap

See `docs/superpowers/specs/2026-05-23-vobiz-cli-design.md`. Upcoming plans add:

- `calls`, `numbers` (Plan 2)
- `trunks`, `applications` (Plan 3)
- `whatsapp`, `partner` (Plan 4)
- GoReleaser, Homebrew tap, Docker image, install script (Plan 5)
````

- [ ] **Step 3: Manual smoke against the public API (optional, requires real credentials)**

```bash
./vobiz auth login
./vobiz account get
./vobiz docs search "sip trunk"
```

If any step fails, fix and re-test before committing.

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: README usage, profiles, env vars, output formats"
```

---

## Follow-on plans

This plan deliberately stops at "working foundation." The next plans drop straight onto the same layers (`cmd/<resource>/<verb>.go` + an interface added to `internal/client`):

| # | Plan filename | Scope |
|---|---|---|
| 2 | `2026-05-23-vobiz-cli-calls-numbers.md` | `cmd/calls` (make, list, get, recordings list, recordings download — using the official SDK's `calls` package), `cmd/numbers` (list, search, buy, release) |
| 3 | `2026-05-23-vobiz-cli-trunks-applications.md` | `cmd/trunks` (CRUD + credentials/ip-acl/origination), `cmd/applications` (CRUD + attach/detach) — resource paths confirmed at the time of writing from `https://docs.vobiz.ai/llms.txt` |
| 4 | `2026-05-23-vobiz-cli-whatsapp-partner.md` | `cmd/whatsapp` (send {text,media,template}, templates, campaigns, contacts), `cmd/partner` (customers, balance transfer, numbers, cdrs, analytics) |
| 5 | `2026-05-23-vobiz-cli-release.md` | `Dockerfile` (distroless multi-stage), `.goreleaser.yaml` (linux/darwin/windows × amd64/arm64 + Homebrew tap + GHCR Docker), `install.sh`, release workflow on tag `v*` |

Each follow-on plan reuses the patterns established here:

- New resource → add `XAPI` interface + struct in `internal/client/<name>.go`; mock it in `cmd/<name>/<name>_test.go`.
- New verb → one file per verb under `cmd/<name>/`, calling `runtime.NewClient(Overrides)`.
- New paginated endpoint → use `paginate.AllN` like `account/transactions.go`.
- New output → add a `[]Column` slice at the call site; no renderer changes.
- New error class → already covered by `internal/httpx.classify` and `internal/errors.ExitCode`.
