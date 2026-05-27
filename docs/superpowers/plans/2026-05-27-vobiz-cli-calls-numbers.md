# Vobiz CLI — Plan 2: Calls + Numbers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `vobiz numbers {list,search,buy,release}` and `vobiz calls {make,list,get,recordings list,recordings download}` on top of the foundation merged in Plan 1. Also fold in two carry-over fixes from the foundation code review: single-struct JSON/YAML wrapping and `output.writeTable` silent field-typo swallow.

**Architecture:** Drops onto the patterns established in Plan 1. Each resource adds (a) a typed `*API` interface + struct in `internal/client/`, and (b) one verb-per-file under `cmd/<resource>/`. Subcommands only call `runtime.NewClient(Overrides)`, never HTTP directly. Recording download streams `io.Copy` from response body to file.

**Tech Stack:** Same as Plan 1 — Go 1.24+, spf13/cobra, gopkg.in/yaml.v3, jedib0t/go-pretty, charmbracelet/glamour, google/uuid, golang.org/x/term, vobiz REST API at `https://api.vobiz.ai/api/v1`.

**Companion docs:**
- Design spec: `docs/superpowers/specs/2026-05-23-vobiz-cli-design.md`
- Plan 1 (merged): `docs/superpowers/plans/2026-05-23-vobiz-cli-foundation.md`

---

## Vobiz API surface used in this plan

Confirmed verbatim from `https://docs.vobiz.ai/llms.txt` and the linked pages:

### Calls
| Operation | Method | Path |
|---|---|---|
| Make outbound call | `POST` | `/Account/{auth_id}/Call/` |
| List CDRs | `GET` | `/Account/{auth_id}/cdr` |
| Get one CDR | `GET` | `/Account/{auth_id}/cdr/{call_id}` |
| Recent CDRs | `GET` | `/Account/{auth_id}/cdr/recent` |

CDR list pagination: `page` (default 1), `per_page` (default 20, max 100). Filters: `from_number`, `to_number`, `start_date`, `end_date`, `call_direction`, `min_duration`. Response top-level: `{ account_id, count, data, pagination, summary, success }`. Pagination object: `{ page, per_page, total, pages, has_next, has_prev }`.

### Recordings
| Operation | Method | Path |
|---|---|---|
| List recordings | `GET` | `/Account/{auth_id}/Recording/` |
| Get recording | `GET` | `/Account/{auth_id}/Recording/{recording_id}` |
| Download audio | `GET` | `/Account/{auth_id}/Recording/{recording_id}/download` |

### Phone numbers / DIDs
The docs index lists documentation routes (`/account-phone-number/*`) rather than concrete REST paths for the inventory/purchase endpoints. The REST surface follows Plivo-style conventions:

| Operation | Method | Path |
|---|---|---|
| List owned numbers | `GET` | `/Account/{auth_id}/Number/` |
| Search inventory | `GET` | `/Account/{auth_id}/PhoneNumber/?country_iso=XX` |
| Buy a number | `POST` | `/Account/{auth_id}/AvailablePrefix/{number}/` (or `/Account/{auth_id}/Number/` per inventory) |
| Release a number | `DELETE` | `/Account/{auth_id}/Number/{number}/` |

**Verify-before-merge:** Task 2 includes a hard requirement to confirm these paths against a real account or the up-to-date `llms.txt` index before commit. If the paths differ, update the constants in `internal/client/numbers.go` and `numbers_test.go` and proceed — no other code needs to change because every subcommand goes through the typed interface.

---

## Conventions used throughout

- **Module path:** `github.com/yash-kavaiya/vobiz-cli` (unchanged from Plan 1).
- **One commit per task.** Stage explicit file lists, never `git add -A`.
- **TDD:** every code task starts with a failing test, then minimal code, then green, then commit.
- **Branch:** create `feature/calls-numbers` off `main` before Task 0.
- **Windows note:** `chmod 0600` is a no-op on Windows; tests gate mode assertions on `runtime.GOOS != "windows"`.
- **Go PATH note for subagent shells:** if `go version` fails, run `$env:PATH = "C:\Go\bin;" + $env:PATH` (PowerShell) or `export PATH="/c/Go/bin:$PATH"` (bash).

---

## Task 0: Create branch and fold in two carry-over fixes from the foundation review

**Files:**
- Modify: `internal/output/output.go`
- Modify: `internal/output/output_test.go`
- Modify: `cmd/account/get.go`
- Modify: `cmd/account/concurrency.go`
- Modify: `cmd/account/transactions.go`
- Modify: `cmd/account/account_test.go`

- [ ] **Step 1: Branch off main**

```
git checkout main
git pull --ff-only   # if remote exists
git checkout -b feature/calls-numbers
```

- [ ] **Step 2: Write failing tests for the output fixes**

Append to `internal/output/output_test.go`:
```go
func TestRender_SingleStructEmitsObjectNotArray_JSON(t *testing.T) {
	row := trunk{ID: "t1", Name: "Outbound-A", CPS: 10}
	var buf bytes.Buffer
	if err := Render(&buf, row, cols, FormatJSON); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.HasPrefix(strings.TrimSpace(out), "[") {
		t.Fatalf("single struct should not be wrapped in an array, got:\n%s", out)
	}
	if !strings.Contains(out, `"ID": "t1"`) {
		t.Fatalf("json output unexpected:\n%s", out)
	}
}

func TestRender_SingleStructEmitsObjectNotArray_YAML(t *testing.T) {
	row := trunk{ID: "t1", Name: "Outbound-A", CPS: 10}
	var buf bytes.Buffer
	if err := Render(&buf, row, cols, FormatYAML); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.HasPrefix(strings.TrimSpace(out), "-") {
		t.Fatalf("single struct should not produce a YAML list item, got:\n%s", out)
	}
	if !strings.Contains(out, "id: t1") {
		t.Fatalf("yaml output unexpected:\n%s", out)
	}
}

func TestRender_TypoInColumnFieldReturnsError(t *testing.T) {
	bogus := []Column{{Header: "OOPS", Field: "TypoField"}}
	var buf bytes.Buffer
	err := Render(&buf, rows, bogus, FormatTable)
	if err == nil {
		t.Fatal("expected error for unknown struct field 'TypoField'")
	}
	if !strings.Contains(err.Error(), "TypoField") {
		t.Fatalf("error should mention the bad field name: %v", err)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```
go test ./internal/output/...
```
Expected: 3 new tests FAIL.

- [ ] **Step 4: Fix `Render` to detect singletons and validate column fields**

Replace `Render` and `writeTable` in `internal/output/output.go`:
```go
// Render writes rows in the chosen format. `rows` must be a slice or a single struct.
//
// If `rows` is a single struct (not a slice), JSON/YAML output emits a single
// object — not a one-element array. This matches the ergonomic expectation
// that `vobiz account get -o json | jq '.auth_id'` works without `.[0]`.
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

func writeTable(w io.Writer, rows any, cols []Column) error {
	v := reflect.ValueOf(rows)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		slice := reflect.MakeSlice(reflect.SliceOf(v.Type()), 1, 1)
		slice.Index(0).Set(v)
		v = slice
	}

	// Validate every column field exists on the element type up-front so a typo
	// in a column spec fails loudly instead of producing silent empty cells.
	if v.Len() > 0 {
		elemType := v.Index(0).Type()
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		for _, c := range cols {
			if _, ok := elemType.FieldByName(c.Field); !ok {
				return fmt.Errorf("output: column %q references unknown struct field %q on %s",
					c.Header, c.Field, elemType.Name())
			}
		}
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

Note: `writeJSON` and `writeYAML` already pass `rows` directly to the encoders, so a non-slice value naturally serializes as a single object. The bug was that *callers* (`cmd/account/get.go` etc.) wrapped singletons in a one-element slice before passing.

- [ ] **Step 5: Stop wrapping singletons in `cmd/account`**

Modify the three account commands that wrap single structs.

In `cmd/account/get.go`, replace the `output.Render` call:
```go
return output.Render(w, *acc, cols, f)
```

In `cmd/account/concurrency.go`, replace the `output.Render` call:
```go
return output.Render(w, *c, cols, f)
```

In `cmd/account/transactions.go`, leave it as-is — it's a true list.

- [ ] **Step 6: Update the account tests to match the new shape**

Edit `cmd/account/account_test.go`. Replace `TestGet_JSONOutput`:
```go
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
```

- [ ] **Step 7: Update the smoke test to match the new shape**

Edit `cmd/smoke_test.go`:
```go
	if !strings.Contains(buf.String(), `"auth_id": "AB12"`) {
		t.Fatalf("smoke output unexpected:\n%s", buf.String())
	}
```
already passes because the API JSON response still uses snake_case. But the runtime test was happening against the *wrapped* form `[{...}]`. After Step 5, the output will be `{...}`. The existing assertion (`contains "auth_id": "AB12"`) still passes either way. No change needed — just re-run to confirm.

- [ ] **Step 8: Run all tests, expect green**

```
go test ./...
```
Expected: PASS across all packages.

- [ ] **Step 9: Commit**

```
git add internal/output/ cmd/account/get.go cmd/account/concurrency.go cmd/account/account_test.go
git commit -m "fix(output): emit object (not array) for single struct, fail loudly on bad column field"
```

---

## Task 1: Add `internal/client/numbers.go` with typed `NumbersAPI`

**Files:**
- Create: `internal/client/numbers.go`
- Create: `internal/client/numbers_test.go`
- Modify: `internal/client/client.go`

- [ ] **Step 1: Write the failing test**

Create `internal/client/numbers_test.go`:
```go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
)

func TestNumbers_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/Number/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"objects": []map[string]any{
				{"number": "+14155551212", "country": "US", "monthly_rental_rate": "1.00"},
			},
			"meta": map[string]any{"next": ""},
		})
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	items, next, err := c.Numbers.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Number != "+14155551212" {
		t.Fatalf("got %+v", items)
	}
	if next != "" {
		t.Fatalf("next = %q", next)
	}
}

func TestNumbers_SearchInventory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/Account/AB12/PhoneNumber/") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("country_iso") != "IN" {
			t.Errorf("country_iso = %q", r.URL.Query().Get("country_iso"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"objects": []map[string]any{
				{"number": "+919999999999", "country": "IN", "setup_rate": "0", "monthly_rental_rate": "0.80"},
			},
		})
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Numbers.SearchInventory(context.Background(), "IN")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Number != "+919999999999" {
		t.Fatalf("got %+v", got)
	}
}

func TestNumbers_Buy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q", r.Method)
		}
		if !strings.Contains(r.URL.Path, "AvailablePrefix") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Errorf("missing idempotency key")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"message":"created","numbers":["+14155551212"]}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	if err := c.Numbers.Buy(context.Background(), "+14155551212"); err != nil {
		t.Fatal(err)
	}
}

func TestNumbers_Release(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q", r.Method)
		}
		if r.URL.Path != "/Account/AB12/Number/+14155551212/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	if err := c.Numbers.Release(context.Background(), "+14155551212"); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```
go test ./internal/client/...
```
Expected: build failure (`c.Numbers undefined`).

- [ ] **Step 3: Write the package**

Create `internal/client/numbers.go`:
```go
package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
)

type Number struct {
	Number            string `json:"number"               yaml:"number"`
	Country           string `json:"country"              yaml:"country"`
	NumberType        string `json:"number_type"          yaml:"number_type,omitempty"`
	MonthlyRentalRate string `json:"monthly_rental_rate"  yaml:"monthly_rental_rate"`
	SetupRate         string `json:"setup_rate,omitempty" yaml:"setup_rate,omitempty"`
	Application       string `json:"application,omitempty" yaml:"application,omitempty"`
	AddedOn           string `json:"added_on,omitempty"    yaml:"added_on,omitempty"`
}

type NumbersAPI interface {
	List(ctx context.Context, cursor string) ([]Number, string, error)
	SearchInventory(ctx context.Context, countryISO string) ([]Number, error)
	Buy(ctx context.Context, number string) error
	Release(ctx context.Context, number string) error
}

type numbersAPI struct {
	http   *httpx.Client
	authID string
}

func (n *numbersAPI) List(ctx context.Context, cursor string) ([]Number, string, error) {
	path := "/Account/" + n.authID + "/Number/"
	if cursor != "" {
		path += "?cursor=" + url.QueryEscape(cursor)
	}
	var raw struct {
		Objects []Number `json:"objects"`
		Meta    struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	if err := n.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, "", err
	}
	return raw.Objects, raw.Meta.Next, nil
}

func (n *numbersAPI) SearchInventory(ctx context.Context, countryISO string) ([]Number, error) {
	q := url.Values{}
	if countryISO != "" {
		q.Set("country_iso", countryISO)
	}
	path := "/Account/" + n.authID + "/PhoneNumber/"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var raw struct {
		Objects []Number `json:"objects"`
	}
	if err := n.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return raw.Objects, nil
}

func (n *numbersAPI) Buy(ctx context.Context, number string) error {
	path := "/Account/" + n.authID + "/AvailablePrefix/" + url.PathEscape(number) + "/"
	return n.http.DoJSON(ctx, http.MethodPost, path, struct{}{}, nil)
}

func (n *numbersAPI) Release(ctx context.Context, number string) error {
	path := "/Account/" + n.authID + "/Number/" + url.PathEscape(number) + "/"
	return n.http.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}
```

- [ ] **Step 4: Wire Numbers into the Client struct**

Modify `internal/client/client.go` to add a `Numbers` field and construct it:
```go
type Client struct {
	HTTP    *httpx.Client
	Account AccountAPI
	Numbers NumbersAPI
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
		Numbers: &numbersAPI{http: h, authID: creds.AuthID},
	}
}
```

- [ ] **Step 5: Run tests**

```
go test ./internal/client/...
```
Expected: PASS.

- [ ] **Step 6: Verify the REST paths against the live API or `llms.txt`**

This task assumes the paths from §"Vobiz API surface used in this plan" above. Before committing, fetch `https://docs.vobiz.ai/llms.txt` and search for "PhoneNumber" / "AvailablePrefix" / "Number/". If any verified path differs from the constants in `numbers.go`, update the constants and re-run tests. Document any divergence in the commit body.

- [ ] **Step 7: Commit**

```
git add internal/client/numbers.go internal/client/numbers_test.go internal/client/client.go
git commit -m "feat(client): typed Numbers API (list, search inventory, buy, release)"
```

---

## Task 2: `vobiz numbers list | search` (read-only)

**Files:**
- Create: `cmd/numbers/numbers.go`
- Create: `cmd/numbers/list.go`
- Create: `cmd/numbers/search.go`
- Create: `cmd/numbers/numbers_test.go`
- Modify: `cmd/registrations.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/numbers/numbers_test.go`:
```go
package numbers

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

type fakeNumbers struct {
	owned  []client.Number
	inv    []client.Number
	bought string
	released string
}

func (f *fakeNumbers) List(_ context.Context, _ string) ([]client.Number, string, error) {
	return f.owned, "", nil
}
func (f *fakeNumbers) SearchInventory(_ context.Context, _ string) ([]client.Number, error) {
	return f.inv, nil
}
func (f *fakeNumbers) Buy(_ context.Context, n string) error     { f.bought = n; return nil }
func (f *fakeNumbers) Release(_ context.Context, n string) error { f.released = n; return nil }

func TestList_RendersOwned(t *testing.T) {
	f := &fakeNumbers{owned: []client.Number{
		{Number: "+14155551212", Country: "US", MonthlyRentalRate: "1.00"},
		{Number: "+919999999999", Country: "IN", MonthlyRentalRate: "0.80"},
	}}
	var out bytes.Buffer
	if err := runList(f, &out, "table", 50, false); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"+14155551212", "+919999999999", "US", "IN"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q:\n%s", w, out.String())
		}
	}
}

func TestSearch_RendersInventory(t *testing.T) {
	f := &fakeNumbers{inv: []client.Number{
		{Number: "+12025550100", Country: "US", SetupRate: "0", MonthlyRentalRate: "1.00"},
	}}
	var out bytes.Buffer
	if err := runSearch(f, &out, "table", "US"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "+12025550100") {
		t.Fatalf("missing number:\n%s", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./cmd/numbers/...
```
Expected: build failure.

- [ ] **Step 3: Write the package files**

Create `cmd/numbers/numbers.go`:
```go
// Package numbers implements `vobiz numbers …` subcommands.
package numbers

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

var Overrides runtime.Overrides

var NumbersFactory = func() (client.NumbersAPI, error) {
	c, err := runtime.NewClient(Overrides)
	if err != nil {
		return nil, err
	}
	return c.Numbers, nil
}

func Register(parent *cobra.Command, format func() string, ov func() runtime.Overrides) {
	cmd := &cobra.Command{
		Use:   "numbers",
		Short: "Manage owned phone numbers and search inventory",
	}
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		Overrides = ov()
		return nil
	}
	cmd.AddCommand(newListCmd(format))
	cmd.AddCommand(newSearchCmd(format))
	cmd.AddCommand(newBuyCmd())
	cmd.AddCommand(newReleaseCmd())
	parent.AddCommand(cmd)
}
```

Create `cmd/numbers/list.go`:
```go
package numbers

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
	"github.com/yash-kavaiya/vobiz-cli/internal/paginate"
)

func newListCmd(format func() string) *cobra.Command {
	var (
		limit int
		all   bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List owned phone numbers (paginated)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := NumbersFactory()
			if err != nil {
				return err
			}
			return runList(a, cmd.OutOrStdout(), format(), limit, all)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "max number of rows")
	cmd.Flags().BoolVar(&all, "all", false, "fetch all pages")
	return cmd
}

func runList(api client.NumbersAPI, w io.Writer, format string, limit int, all bool) error {
	fetch := func(ctx context.Context, cursor string) (paginate.Page[client.Number], error) {
		items, next, err := api.List(ctx, cursor)
		if err != nil {
			return paginate.Page[client.Number]{}, err
		}
		return paginate.Page[client.Number]{Items: items, NextCursor: next}, nil
	}
	var (
		rows []client.Number
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
		{Header: "NUMBER", Field: "Number"},
		{Header: "COUNTRY", Field: "Country"},
		{Header: "TYPE", Field: "NumberType"},
		{Header: "MONTHLY", Field: "MonthlyRentalRate"},
		{Header: "APPLICATION", Field: "Application"},
	}
	return output.Render(w, rows, cols, f)
}
```

Create `cmd/numbers/search.go`:
```go
package numbers

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newSearchCmd(format func() string) *cobra.Command {
	var country string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search available numbers in inventory (filter by ISO country code)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := NumbersFactory()
			if err != nil {
				return err
			}
			return runSearch(a, cmd.OutOrStdout(), format(), country)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "ISO country code (e.g. US, IN, GB)")
	return cmd
}

func runSearch(api client.NumbersAPI, w io.Writer, format, country string) error {
	rows, err := api.SearchInventory(context.Background(), country)
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "NUMBER", Field: "Number"},
		{Header: "COUNTRY", Field: "Country"},
		{Header: "SETUP", Field: "SetupRate"},
		{Header: "MONTHLY", Field: "MonthlyRentalRate"},
	}
	return output.Render(w, rows, cols, f)
}
```

Also create stubs for `buy.go` and `release.go` (full implementation in Task 3 — but the package must compile because `numbers.go` references `newBuyCmd` and `newReleaseCmd`):

Create `cmd/numbers/buy.go`:
```go
package numbers

import "github.com/spf13/cobra"

func newBuyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "buy <number>",
		Short: "Buy a phone number from inventory (implemented in Task 3)",
		Hidden: true,
	}
}
```

Create `cmd/numbers/release.go`:
```go
package numbers

import "github.com/spf13/cobra"

func newReleaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "release <number>",
		Short: "Release a number back to inventory (implemented in Task 3)",
		Hidden: true,
	}
}
```

- [ ] **Step 4: Wire into `cmd/registrations.go`**

Edit `cmd/registrations.go` — add the `numbers` import and a `registerNumbers` call. Replace the file with:
```go
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/account"
	"github.com/yash-kavaiya/vobiz-cli/cmd/auth"
	"github.com/yash-kavaiya/vobiz-cli/cmd/docs"
	"github.com/yash-kavaiya/vobiz-cli/cmd/numbers"
	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
)

func registerVersion(root *cobra.Command)    { root.AddCommand(newVersionCmd()) }
func registerCompletion(root *cobra.Command) { root.AddCommand(newCompletionCmd(root)) }
func registerAuth(root *cobra.Command)       { auth.Register(root) }
func registerAccount(root *cobra.Command) {
	account.Register(root, func() string { return globalOutput }, ovFn)
}
func registerNumbers(root *cobra.Command) {
	numbers.Register(root, func() string { return globalOutput }, ovFn)
}
func registerDocs(root *cobra.Command) { docs.Register(root) }

func ovFn() runtime.Overrides {
	return runtime.Overrides{
		Profile:     globalProfile,
		FlagID:      globalAuthID,
		FlagToken:   globalAuthTok,
		FlagBaseURL: globalBaseURL,
	}
}
```

Also edit `cmd/root.go` to call `registerNumbers(root)` after `registerAccount(root)`:
```go
	registerAccount(root)
	registerNumbers(root)
	registerDocs(root)
```

- [ ] **Step 5: Run tests**

```
go test ./cmd/numbers/... ./cmd/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add cmd/numbers/ cmd/registrations.go cmd/root.go
git commit -m "feat(numbers): list and search subcommands"
```

---

## Task 3: `vobiz numbers buy | release` (mutations)

**Files:**
- Modify: `cmd/numbers/buy.go` (replace stub)
- Modify: `cmd/numbers/release.go` (replace stub)
- Modify: `cmd/numbers/numbers_test.go` (add tests)

- [ ] **Step 1: Add failing tests**

Append to `cmd/numbers/numbers_test.go`:
```go
func TestBuy_CallsAPI(t *testing.T) {
	f := &fakeNumbers{}
	var out bytes.Buffer
	if err := runBuy(f, &out, "+14155551212"); err != nil {
		t.Fatal(err)
	}
	if f.bought != "+14155551212" {
		t.Fatalf("bought = %q", f.bought)
	}
	if !strings.Contains(out.String(), "+14155551212") {
		t.Fatalf("output missing number:\n%s", out.String())
	}
}

func TestRelease_CallsAPI(t *testing.T) {
	f := &fakeNumbers{}
	var out bytes.Buffer
	if err := runRelease(f, &out, "+14155551212"); err != nil {
		t.Fatal(err)
	}
	if f.released != "+14155551212" {
		t.Fatalf("released = %q", f.released)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./cmd/numbers/...
```
Expected: build failure.

- [ ] **Step 3: Replace `buy.go`**

```go
package numbers

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

func newBuyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "buy <number>",
		Short: "Buy a phone number from inventory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := NumbersFactory()
			if err != nil {
				return err
			}
			return runBuy(a, cmd.OutOrStdout(), args[0])
		},
	}
}

func runBuy(api client.NumbersAPI, w io.Writer, number string) error {
	if err := api.Buy(context.Background(), number); err != nil {
		return err
	}
	fmt.Fprintf(w, "Purchased %s.\n", number)
	return nil
}
```

- [ ] **Step 4: Replace `release.go`**

```go
package numbers

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

func newReleaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "release <number>",
		Short: "Release a number back to inventory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := NumbersFactory()
			if err != nil {
				return err
			}
			return runRelease(a, cmd.OutOrStdout(), args[0])
		},
	}
}

func runRelease(api client.NumbersAPI, w io.Writer, number string) error {
	if err := api.Release(context.Background(), number); err != nil {
		return err
	}
	fmt.Fprintf(w, "Released %s.\n", number)
	return nil
}
```

- [ ] **Step 5: Run tests**

```
go test ./cmd/numbers/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add cmd/numbers/buy.go cmd/numbers/release.go cmd/numbers/numbers_test.go
git commit -m "feat(numbers): buy and release subcommands"
```

---

## Task 4: Add `internal/client/calls.go` with typed `CallsAPI`

**Files:**
- Create: `internal/client/calls.go`
- Create: `internal/client/calls_test.go`
- Modify: `internal/client/client.go`

- [ ] **Step 1: Write the failing test**

Create `internal/client/calls_test.go`:
```go
package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
)

func TestCalls_Make(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/Call/" || r.Method != http.MethodPost {
			t.Errorf("path/method = %q/%q", r.URL.Path, r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if req["from"] != "+14150000000" || req["to"] != "+14155551212" {
			t.Errorf("from/to wrong: %+v", req)
		}
		if req["answer_url"] != "https://example.com/ans" {
			t.Errorf("answer_url wrong: %+v", req)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"api_id":"a1","message":"call submitted","request_uuid":"uuid-1"}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Calls.Make(context.Background(), MakeCallParams{
		From: "+14150000000", To: "+14155551212", AnswerURL: "https://example.com/ans",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.RequestUUID != "uuid-1" {
		t.Fatalf("got %+v", got)
	}
}

func TestCalls_ListCDR_Pagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cdr") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("page = %q", r.URL.Query().Get("page"))
		}
		_, _ = w.Write([]byte(`{
		  "data":[{"uuid":"c1","caller_id_number":"+1","destination_number":"+2","duration":30,"billsec":25,"cost":"0.10","call_direction":"outbound","hangup_cause":"NORMAL_CLEARING"}],
		  "pagination":{"page":2,"per_page":20,"total":50,"pages":3,"has_next":true,"has_prev":true},
		  "success":true
		}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	rows, page, err := c.Calls.ListCDR(context.Background(), CDRListOpts{Page: 2, PerPage: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].UUID != "c1" {
		t.Fatalf("rows: %+v", rows)
	}
	if !page.HasNext || page.Total != 50 {
		t.Fatalf("page: %+v", page)
	}
}

func TestCalls_GetCDR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cdr/c1") {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"uuid":"c1","caller_id_number":"+1","destination_number":"+2","duration":30,"cost":"0.10"}}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Calls.GetCDR(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "c1" {
		t.Fatalf("got %+v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```
go test ./internal/client/...
```
Expected: build failure.

- [ ] **Step 3: Write the package**

Create `internal/client/calls.go`:
```go
package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
)

type MakeCallParams struct {
	From                 string `json:"from"`
	To                   string `json:"to"`
	AnswerURL            string `json:"answer_url"`
	AnswerMethod         string `json:"answer_method,omitempty"`
	RingURL              string `json:"ring_url,omitempty"`
	RingMethod           string `json:"ring_method,omitempty"`
	HangupURL            string `json:"hangup_url,omitempty"`
	HangupMethod         string `json:"hangup_method,omitempty"`
	FallbackURL          string `json:"fallback_url,omitempty"`
	FallbackMethod       string `json:"fallback_method,omitempty"`
	MachineDetection     string `json:"machine_detection,omitempty"`
	MachineDetectionTime int    `json:"machine_detection_time,omitempty"`
	CallerName           string `json:"caller_name,omitempty"`
	SendDigits           string `json:"send_digits,omitempty"`
	TimeLimit            int    `json:"time_limit,omitempty"`
}

type MakeCallResponse struct {
	APIID       string `json:"api_id"        yaml:"api_id"`
	Message     string `json:"message"       yaml:"message"`
	RequestUUID string `json:"request_uuid"  yaml:"request_uuid"`
}

// CDR is a Call Detail Record. The Vobiz API returns 40+ fields per record;
// these are the most useful for terminal display. Additional fields are
// preserved in `Extra` so JSON/YAML output round-trips correctly.
type CDR struct {
	UUID              string `json:"uuid"               yaml:"uuid"`
	CallerIDNumber    string `json:"caller_id_number"   yaml:"caller_id_number"`
	DestinationNumber string `json:"destination_number" yaml:"destination_number"`
	CallDirection     string `json:"call_direction"     yaml:"call_direction"`
	Duration          int    `json:"duration"           yaml:"duration"`
	BillSec           int    `json:"billsec"            yaml:"billsec"`
	Cost              string `json:"cost"               yaml:"cost"`
	HangupCause       string `json:"hangup_cause"       yaml:"hangup_cause"`
	StartTime         string `json:"start_stamp,omitempty"  yaml:"start_stamp,omitempty"`
	EndTime           string `json:"end_stamp,omitempty"    yaml:"end_stamp,omitempty"`
	MOS               string `json:"mos,omitempty"          yaml:"mos,omitempty"`
}

type Pagination struct {
	Page    int  `json:"page"      yaml:"page"`
	PerPage int  `json:"per_page"  yaml:"per_page"`
	Total   int  `json:"total"     yaml:"total"`
	Pages   int  `json:"pages"     yaml:"pages"`
	HasNext bool `json:"has_next"  yaml:"has_next"`
	HasPrev bool `json:"has_prev"  yaml:"has_prev"`
}

type CDRListOpts struct {
	Page          int
	PerPage       int
	FromNumber    string
	ToNumber      string
	StartDate     string // ISO date
	EndDate       string
	CallDirection string // "inbound" | "outbound"
	MinDuration   int    // seconds
}

type CallsAPI interface {
	Make(ctx context.Context, p MakeCallParams) (*MakeCallResponse, error)
	ListCDR(ctx context.Context, opts CDRListOpts) ([]CDR, Pagination, error)
	GetCDR(ctx context.Context, callID string) (*CDR, error)
}

type callsAPI struct {
	http   *httpx.Client
	authID string
}

func (c *callsAPI) Make(ctx context.Context, p MakeCallParams) (*MakeCallResponse, error) {
	var out MakeCallResponse
	if err := c.http.DoJSON(ctx, http.MethodPost, "/Account/"+c.authID+"/Call/", p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *callsAPI) ListCDR(ctx context.Context, opts CDRListOpts) ([]CDR, Pagination, error) {
	q := url.Values{}
	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PerPage > 0 {
		q.Set("per_page", strconv.Itoa(opts.PerPage))
	}
	if opts.FromNumber != "" {
		q.Set("from_number", opts.FromNumber)
	}
	if opts.ToNumber != "" {
		q.Set("to_number", opts.ToNumber)
	}
	if opts.StartDate != "" {
		q.Set("start_date", opts.StartDate)
	}
	if opts.EndDate != "" {
		q.Set("end_date", opts.EndDate)
	}
	if opts.CallDirection != "" {
		q.Set("call_direction", opts.CallDirection)
	}
	if opts.MinDuration > 0 {
		q.Set("min_duration", strconv.Itoa(opts.MinDuration))
	}
	path := "/Account/" + c.authID + "/cdr"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var raw struct {
		Data       []CDR      `json:"data"`
		Pagination Pagination `json:"pagination"`
	}
	if err := c.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, Pagination{}, err
	}
	return raw.Data, raw.Pagination, nil
}

func (c *callsAPI) GetCDR(ctx context.Context, callID string) (*CDR, error) {
	var raw struct {
		Data CDR `json:"data"`
	}
	path := "/Account/" + c.authID + "/cdr/" + url.PathEscape(callID)
	if err := c.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return &raw.Data, nil
}
```

- [ ] **Step 4: Wire `Calls` into the Client struct**

Modify `internal/client/client.go`:
```go
type Client struct {
	HTTP    *httpx.Client
	Account AccountAPI
	Numbers NumbersAPI
	Calls   CallsAPI
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
		Numbers: &numbersAPI{http: h, authID: creds.AuthID},
		Calls:   &callsAPI{http: h, authID: creds.AuthID},
	}
}
```

- [ ] **Step 5: Run tests**

```
go test ./internal/client/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add internal/client/calls.go internal/client/calls_test.go internal/client/client.go
git commit -m "feat(client): typed Calls API (make, list CDRs, get CDR)"
```

---

## Task 5: `vobiz calls make`

**Files:**
- Create: `cmd/calls/calls.go`
- Create: `cmd/calls/make.go`
- Create: `cmd/calls/calls_test.go`
- Modify: `cmd/registrations.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/calls/calls_test.go`:
```go
package calls

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

type fakeCalls struct {
	madeWith client.MakeCallParams
	cdr      *client.CDR
	cdrs     []client.CDR
	pag      client.Pagination
}

func (f *fakeCalls) Make(_ context.Context, p client.MakeCallParams) (*client.MakeCallResponse, error) {
	f.madeWith = p
	return &client.MakeCallResponse{APIID: "a1", Message: "submitted", RequestUUID: "uuid-1"}, nil
}
func (f *fakeCalls) ListCDR(_ context.Context, _ client.CDRListOpts) ([]client.CDR, client.Pagination, error) {
	return f.cdrs, f.pag, nil
}
func (f *fakeCalls) GetCDR(_ context.Context, _ string) (*client.CDR, error) {
	return f.cdr, nil
}

func TestMake_PassesParams_AndPrintsRequestUUID(t *testing.T) {
	f := &fakeCalls{}
	var out bytes.Buffer
	if err := runMake(f, &out, "+14150000000", "+14155551212", "https://x/ans", makeFlags{}); err != nil {
		t.Fatal(err)
	}
	if f.madeWith.From != "+14150000000" || f.madeWith.To != "+14155551212" || f.madeWith.AnswerURL != "https://x/ans" {
		t.Fatalf("params: %+v", f.madeWith)
	}
	if !strings.Contains(out.String(), "uuid-1") {
		t.Fatalf("output: %q", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./cmd/calls/...
```
Expected: build failure.

- [ ] **Step 3: Write `cmd/calls/calls.go`**

```go
// Package calls implements `vobiz calls …` subcommands.
package calls

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

var Overrides runtime.Overrides

var CallsFactory = func() (client.CallsAPI, error) {
	c, err := runtime.NewClient(Overrides)
	if err != nil {
		return nil, err
	}
	return c.Calls, nil
}

func Register(parent *cobra.Command, format func() string, ov func() runtime.Overrides) {
	cmd := &cobra.Command{
		Use:   "calls",
		Short: "Make outbound calls and inspect call records",
	}
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		Overrides = ov()
		return nil
	}
	cmd.AddCommand(newMakeCmd())
	parent.AddCommand(cmd)
}
```

- [ ] **Step 4: Write `cmd/calls/make.go`**

```go
package calls

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

type makeFlags struct {
	AnswerMethod         string
	RingURL              string
	HangupURL            string
	FallbackURL          string
	MachineDetection     string
	MachineDetectionTime int
	CallerName           string
	TimeLimit            int
}

func newMakeCmd() *cobra.Command {
	var (
		from, to, ans string
		flags         makeFlags
	)
	cmd := &cobra.Command{
		Use:   "make",
		Short: "Place an outbound call",
		Long:  "Place an outbound call. The Vobiz API will request --answer-url when the callee picks up; that URL must return valid Vobiz XML.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := CallsFactory()
			if err != nil {
				return err
			}
			return runMake(a, cmd.OutOrStdout(), from, to, ans, flags)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "caller ID in E.164 (required)")
	cmd.Flags().StringVar(&to, "to", "", "destination number(s) in E.164, separated by '<' for fan-out (required)")
	cmd.Flags().StringVar(&ans, "answer-url", "", "URL returning Vobiz XML when the call connects (required)")
	cmd.Flags().StringVar(&flags.AnswerMethod, "answer-method", "", "HTTP verb for --answer-url (default POST)")
	cmd.Flags().StringVar(&flags.RingURL, "ring-url", "", "URL notified when the call starts ringing")
	cmd.Flags().StringVar(&flags.HangupURL, "hangup-url", "", "URL notified when the call hangs up")
	cmd.Flags().StringVar(&flags.FallbackURL, "fallback-url", "", "URL invoked if --answer-url fails")
	cmd.Flags().StringVar(&flags.MachineDetection, "machine-detection", "", "answering-machine detection: true|hangup")
	cmd.Flags().IntVar(&flags.MachineDetectionTime, "machine-detection-time", 0, "ms to wait for AMD (2000–10000)")
	cmd.Flags().StringVar(&flags.CallerName, "caller-name", "", "caller display name (max 50 chars)")
	cmd.Flags().IntVar(&flags.TimeLimit, "time-limit", 0, "max call duration in seconds (default 14400)")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("answer-url")
	return cmd
}

func runMake(api client.CallsAPI, w io.Writer, from, to, ans string, f makeFlags) error {
	resp, err := api.Make(context.Background(), client.MakeCallParams{
		From: from, To: to, AnswerURL: ans,
		AnswerMethod:         f.AnswerMethod,
		RingURL:              f.RingURL,
		HangupURL:            f.HangupURL,
		FallbackURL:          f.FallbackURL,
		MachineDetection:     f.MachineDetection,
		MachineDetectionTime: f.MachineDetectionTime,
		CallerName:           f.CallerName,
		TimeLimit:            f.TimeLimit,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\nRequestUUID: %s\nAPIID:       %s\n", resp.Message, resp.RequestUUID, resp.APIID)
	return nil
}
```

- [ ] **Step 5: Wire into registrations**

Edit `cmd/registrations.go`:
```go
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/account"
	"github.com/yash-kavaiya/vobiz-cli/cmd/auth"
	"github.com/yash-kavaiya/vobiz-cli/cmd/calls"
	"github.com/yash-kavaiya/vobiz-cli/cmd/docs"
	"github.com/yash-kavaiya/vobiz-cli/cmd/numbers"
	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
)

func registerVersion(root *cobra.Command)    { root.AddCommand(newVersionCmd()) }
func registerCompletion(root *cobra.Command) { root.AddCommand(newCompletionCmd(root)) }
func registerAuth(root *cobra.Command)       { auth.Register(root) }
func registerAccount(root *cobra.Command)    { account.Register(root, func() string { return globalOutput }, ovFn) }
func registerNumbers(root *cobra.Command)    { numbers.Register(root, func() string { return globalOutput }, ovFn) }
func registerCalls(root *cobra.Command)      { calls.Register(root, func() string { return globalOutput }, ovFn) }
func registerDocs(root *cobra.Command)       { docs.Register(root) }

func ovFn() runtime.Overrides {
	return runtime.Overrides{
		Profile:     globalProfile,
		FlagID:      globalAuthID,
		FlagToken:   globalAuthTok,
		FlagBaseURL: globalBaseURL,
	}
}
```

Edit `cmd/root.go` — add `registerCalls(root)` after `registerNumbers(root)`:
```go
	registerAccount(root)
	registerNumbers(root)
	registerCalls(root)
	registerDocs(root)
```

- [ ] **Step 6: Run tests**

```
go test ./cmd/calls/... ./cmd/...
```
Expected: PASS.

- [ ] **Step 7: Commit**

```
git add cmd/calls/ cmd/registrations.go cmd/root.go
git commit -m "feat(calls): make subcommand with full POST /Call/ flag surface"
```

---

## Task 6: `vobiz calls list | get` (CDRs)

**Files:**
- Create: `cmd/calls/list.go`
- Create: `cmd/calls/get.go`
- Modify: `cmd/calls/calls.go` (add `newListCmd`, `newGetCmd` to Register)
- Modify: `cmd/calls/calls_test.go` (add tests)

- [ ] **Step 1: Add failing tests**

Append to `cmd/calls/calls_test.go`:
```go
func TestList_TableOutput(t *testing.T) {
	f := &fakeCalls{
		cdrs: []client.CDR{
			{UUID: "c1", CallerIDNumber: "+1", DestinationNumber: "+2", Duration: 30, BillSec: 25, Cost: "0.10", HangupCause: "NORMAL_CLEARING", CallDirection: "outbound"},
		},
		pag: client.Pagination{Page: 1, PerPage: 20, Total: 1, Pages: 1},
	}
	var out bytes.Buffer
	if err := runList(f, &out, "table", listFlags{Page: 1, PerPage: 20}); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"c1", "+1", "+2", "30", "NORMAL_CLEARING"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q:\n%s", w, out.String())
		}
	}
}

func TestGet_Prints(t *testing.T) {
	f := &fakeCalls{cdr: &client.CDR{UUID: "c1", Duration: 42, Cost: "0.05"}}
	var out bytes.Buffer
	if err := runGet(f, &out, "table", "c1"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "c1") {
		t.Fatalf("missing uuid:\n%s", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./cmd/calls/...
```
Expected: build failure.

- [ ] **Step 3: Write `cmd/calls/list.go`**

```go
package calls

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

type listFlags struct {
	Page          int
	PerPage       int
	FromNumber    string
	ToNumber      string
	StartDate     string
	EndDate       string
	CallDirection string
	MinDuration   int
}

func newListCmd(format func() string) *cobra.Command {
	var f listFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List call detail records (CDRs)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := CallsFactory()
			if err != nil {
				return err
			}
			return runList(a, cmd.OutOrStdout(), format(), f)
		},
	}
	cmd.Flags().IntVar(&f.Page, "page", 1, "page number")
	cmd.Flags().IntVar(&f.PerPage, "per-page", 20, "rows per page (max 100)")
	cmd.Flags().StringVar(&f.FromNumber, "from", "", "filter by caller ID number")
	cmd.Flags().StringVar(&f.ToNumber, "to", "", "filter by destination number")
	cmd.Flags().StringVar(&f.StartDate, "start", "", "ISO date — only calls on or after")
	cmd.Flags().StringVar(&f.EndDate, "end", "", "ISO date — only calls on or before")
	cmd.Flags().StringVar(&f.CallDirection, "direction", "", "inbound | outbound")
	cmd.Flags().IntVar(&f.MinDuration, "min-duration", 0, "only calls longer than N seconds")
	return cmd
}

func runList(api client.CallsAPI, w io.Writer, format string, f listFlags) error {
	rows, _, err := api.ListCDR(context.Background(), client.CDRListOpts{
		Page: f.Page, PerPage: f.PerPage,
		FromNumber: f.FromNumber, ToNumber: f.ToNumber,
		StartDate: f.StartDate, EndDate: f.EndDate,
		CallDirection: f.CallDirection, MinDuration: f.MinDuration,
	})
	if err != nil {
		return err
	}
	fmt, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "UUID", Field: "UUID"},
		{Header: "FROM", Field: "CallerIDNumber"},
		{Header: "TO", Field: "DestinationNumber"},
		{Header: "DIR", Field: "CallDirection"},
		{Header: "DUR", Field: "Duration"},
		{Header: "BILL", Field: "BillSec"},
		{Header: "COST", Field: "Cost"},
		{Header: "HANGUP", Field: "HangupCause"},
	}
	return output.Render(w, rows, cols, fmt)
}
```

- [ ] **Step 4: Write `cmd/calls/get.go`**

```go
package calls

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newGetCmd(format func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <call-uuid>",
		Short: "Get a single call detail record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := CallsFactory()
			if err != nil {
				return err
			}
			return runGet(a, cmd.OutOrStdout(), format(), args[0])
		},
	}
}

func runGet(api client.CallsAPI, w io.Writer, format, callID string) error {
	cdr, err := api.GetCDR(context.Background(), callID)
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "UUID", Field: "UUID"},
		{Header: "FROM", Field: "CallerIDNumber"},
		{Header: "TO", Field: "DestinationNumber"},
		{Header: "DIR", Field: "CallDirection"},
		{Header: "DURATION", Field: "Duration"},
		{Header: "BILLSEC", Field: "BillSec"},
		{Header: "COST", Field: "Cost"},
		{Header: "HANGUP", Field: "HangupCause"},
		{Header: "MOS", Field: "MOS"},
	}
	return output.Render(w, *cdr, cols, f)
}
```

- [ ] **Step 5: Register list & get in `cmd/calls/calls.go`**

Replace the `Register` body to add the new subcommands:
```go
func Register(parent *cobra.Command, format func() string, ov func() runtime.Overrides) {
	cmd := &cobra.Command{
		Use:   "calls",
		Short: "Make outbound calls and inspect call records",
	}
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		Overrides = ov()
		return nil
	}
	cmd.AddCommand(newMakeCmd())
	cmd.AddCommand(newListCmd(format))
	cmd.AddCommand(newGetCmd(format))
	parent.AddCommand(cmd)
}
```

- [ ] **Step 6: Run tests**

```
go test ./cmd/calls/...
```
Expected: PASS.

- [ ] **Step 7: Commit**

```
git add cmd/calls/list.go cmd/calls/get.go cmd/calls/calls.go cmd/calls/calls_test.go
git commit -m "feat(calls): list and get subcommands (CDRs)"
```

---

## Task 7: Add `internal/client/recordings.go` with typed `RecordingsAPI`

**Files:**
- Create: `internal/client/recordings.go`
- Create: `internal/client/recordings_test.go`
- Modify: `internal/client/client.go`

The download method streams the response body so large recordings don't load into memory.

- [ ] **Step 1: Write the failing test**

Create `internal/client/recordings_test.go`:
```go
package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
)

func TestRecordings_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/Recording/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
		  "objects":[{"recording_id":"r1","call_uuid":"c1","duration":12,"recording_format":"mp3","resource_uri":"/Recording/r1"}],
		  "meta":{"next":""}
		}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	rows, _, err := c.Recordings.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].RecordingID != "r1" {
		t.Fatalf("rows: %+v", rows)
	}
}

func TestRecordings_Download_StreamsBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/Recording/r1/download") {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("\x00ID3FAKE_MP3_BYTES"))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	var buf bytes.Buffer
	if err := c.Recordings.Download(context.Background(), "r1", &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("FAKE_MP3_BYTES")) {
		t.Fatalf("download body: %q", buf.String())
	}
}

// Compile-time guarantee that the interface is implemented.
var _ io.Writer = (*bytes.Buffer)(nil)
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/client/...
```
Expected: build failure.

- [ ] **Step 3: Write the package**

Create `internal/client/recordings.go`:
```go
package client

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
)

type Recording struct {
	RecordingID     string `json:"recording_id"     yaml:"recording_id"`
	CallUUID        string `json:"call_uuid"        yaml:"call_uuid"`
	Duration        int    `json:"duration"         yaml:"duration"`
	RecordingFormat string `json:"recording_format" yaml:"recording_format"`
	RecordingURL    string `json:"recording_url,omitempty" yaml:"recording_url,omitempty"`
	ResourceURI     string `json:"resource_uri"     yaml:"resource_uri"`
	AddedOn         string `json:"added_on,omitempty" yaml:"added_on,omitempty"`
}

type RecordingsAPI interface {
	List(ctx context.Context, cursor string) ([]Recording, string, error)
	Get(ctx context.Context, recordingID string) (*Recording, error)
	Download(ctx context.Context, recordingID string, dst io.Writer) error
}

type recordingsAPI struct {
	http   *httpx.Client
	authID string
}

func (r *recordingsAPI) List(ctx context.Context, cursor string) ([]Recording, string, error) {
	path := "/Account/" + r.authID + "/Recording/"
	if cursor != "" {
		path += "?cursor=" + url.QueryEscape(cursor)
	}
	var raw struct {
		Objects []Recording `json:"objects"`
		Meta    struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	if err := r.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, "", err
	}
	return raw.Objects, raw.Meta.Next, nil
}

func (r *recordingsAPI) Get(ctx context.Context, recordingID string) (*Recording, error) {
	var out Recording
	path := "/Account/" + r.authID + "/Recording/" + url.PathEscape(recordingID)
	if err := r.http.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *recordingsAPI) Download(ctx context.Context, recordingID string, dst io.Writer) error {
	path := "/Account/" + r.authID + "/Recording/" + url.PathEscape(recordingID) + "/download"
	resp, err := r.http.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(dst, resp.Body)
	return err
}
```

- [ ] **Step 4: Wire `Recordings` into the Client struct**

Modify `internal/client/client.go`:
```go
type Client struct {
	HTTP       *httpx.Client
	Account    AccountAPI
	Numbers    NumbersAPI
	Calls      CallsAPI
	Recordings RecordingsAPI
}

func New(creds auth.Credentials) *Client {
	h := httpx.New(httpx.Config{
		BaseURL:   creds.BaseURL,
		AuthID:    creds.AuthID,
		AuthToken: creds.AuthToken,
		UserAgent: "vobiz-cli/" + version.Version,
	})
	return &Client{
		HTTP:       h,
		Account:    &accountAPI{http: h, authID: creds.AuthID},
		Numbers:    &numbersAPI{http: h, authID: creds.AuthID},
		Calls:      &callsAPI{http: h, authID: creds.AuthID},
		Recordings: &recordingsAPI{http: h, authID: creds.AuthID},
	}
}
```

- [ ] **Step 5: Run tests**

```
go test ./internal/client/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add internal/client/recordings.go internal/client/recordings_test.go internal/client/client.go
git commit -m "feat(client): typed Recordings API (list, get, streaming download)"
```

---

## Task 8: `vobiz calls recordings list | download`

**Files:**
- Create: `cmd/calls/recordings.go`
- Modify: `cmd/calls/calls.go` (add `newRecordingsCmd` to Register)
- Modify: `cmd/calls/calls_test.go` (add tests)

- [ ] **Step 1: Add failing tests**

Append to `cmd/calls/calls_test.go`:
```go
type fakeRecordings struct {
	rows         []client.Recording
	downloadedID string
}

func (f *fakeRecordings) List(_ context.Context, _ string) ([]client.Recording, string, error) {
	return f.rows, "", nil
}
func (f *fakeRecordings) Get(_ context.Context, _ string) (*client.Recording, error) {
	return nil, nil
}
func (f *fakeRecordings) Download(_ context.Context, id string, dst io.Writer) error {
	f.downloadedID = id
	_, err := dst.Write([]byte("FAKE_MP3"))
	return err
}

func TestRecordings_List_Renders(t *testing.T) {
	f := &fakeRecordings{rows: []client.Recording{
		{RecordingID: "r1", CallUUID: "c1", Duration: 12, RecordingFormat: "mp3"},
	}}
	var out bytes.Buffer
	if err := runRecordingsList(f, &out, "table", 50, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "r1") {
		t.Fatalf("missing recording id:\n%s", out.String())
	}
}

func TestRecordings_Download_WritesFile(t *testing.T) {
	f := &fakeRecordings{}
	dir := t.TempDir()
	dst := dir + "/r1.mp3"
	if err := runRecordingsDownload(f, "r1", dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "FAKE_MP3" {
		t.Fatalf("body: %q", b)
	}
	if f.downloadedID != "r1" {
		t.Fatalf("downloaded id = %q", f.downloadedID)
	}
}
```

(Add `"io"` and `"os"` to the imports at the top of the file.)

- [ ] **Step 2: Run test to verify it fails**

```
go test ./cmd/calls/...
```
Expected: build failure.

- [ ] **Step 3: Write `cmd/calls/recordings.go`**

```go
package calls

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
	"github.com/yash-kavaiya/vobiz-cli/internal/paginate"
)

var RecordingsFactory = func() (client.RecordingsAPI, error) {
	c, err := runtime.NewClient(Overrides)
	if err != nil {
		return nil, err
	}
	return c.Recordings, nil
}

func newRecordingsCmd(format func() string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recordings",
		Short: "List and download call recordings",
	}
	cmd.AddCommand(newRecordingsListCmd(format))
	cmd.AddCommand(newRecordingsDownloadCmd())
	return cmd
}

func newRecordingsListCmd(format func() string) *cobra.Command {
	var (
		limit int
		all   bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recordings (paginated)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := RecordingsFactory()
			if err != nil {
				return err
			}
			return runRecordingsList(a, cmd.OutOrStdout(), format(), limit, all)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "max number of rows")
	cmd.Flags().BoolVar(&all, "all", false, "fetch all pages")
	return cmd
}

func runRecordingsList(api client.RecordingsAPI, w io.Writer, format string, limit int, all bool) error {
	fetch := func(ctx context.Context, cursor string) (paginate.Page[client.Recording], error) {
		items, next, err := api.List(ctx, cursor)
		if err != nil {
			return paginate.Page[client.Recording]{}, err
		}
		return paginate.Page[client.Recording]{Items: items, NextCursor: next}, nil
	}
	var (
		rows []client.Recording
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
		{Header: "ID", Field: "RecordingID"},
		{Header: "CALL UUID", Field: "CallUUID"},
		{Header: "DURATION", Field: "Duration"},
		{Header: "FORMAT", Field: "RecordingFormat"},
		{Header: "ADDED ON", Field: "AddedOn"},
	}
	return output.Render(w, rows, cols, f)
}

func newRecordingsDownloadCmd() *cobra.Command {
	var dest string
	cmd := &cobra.Command{
		Use:   "download <recording-id>",
		Short: "Download a recording's audio file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := RecordingsFactory()
			if err != nil {
				return err
			}
			out := dest
			if out == "" {
				out = args[0] + ".mp3"
			}
			return runRecordingsDownload(a, args[0], out)
		},
	}
	cmd.Flags().StringVarP(&dest, "output-file", "f", "", "destination path (default <recording-id>.mp3)")
	return cmd
}

func runRecordingsDownload(api client.RecordingsAPI, recordingID, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := api.Download(context.Background(), recordingID, f); err != nil {
		_ = os.Remove(dest)
		return err
	}
	fmt.Fprintf(os.Stderr, "Wrote %s.\n", dest)
	return nil
}
```

- [ ] **Step 4: Register recordings under calls**

Edit `cmd/calls/calls.go`:
```go
func Register(parent *cobra.Command, format func() string, ov func() runtime.Overrides) {
	cmd := &cobra.Command{
		Use:   "calls",
		Short: "Make outbound calls, inspect call records, and manage recordings",
	}
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		Overrides = ov()
		return nil
	}
	cmd.AddCommand(newMakeCmd())
	cmd.AddCommand(newListCmd(format))
	cmd.AddCommand(newGetCmd(format))
	cmd.AddCommand(newRecordingsCmd(format))
	parent.AddCommand(cmd)
}
```

- [ ] **Step 5: Run tests**

```
go test ./cmd/calls/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add cmd/calls/recordings.go cmd/calls/calls.go cmd/calls/calls_test.go
git commit -m "feat(calls): recordings list and download subcommands"
```

---

## Task 9: Extend the end-to-end smoke test to cover `numbers list` and `calls list`

**Files:**
- Modify: `cmd/smoke_test.go`

- [ ] **Step 1: Add the new smoke test**

Append to `cmd/smoke_test.go`:
```go
func TestSmoke_NumbersListEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-ID") != "AB12" {
			t.Errorf("missing auth header: %+v", r.Header)
		}
		_, _ = w.Write([]byte(`{"objects":[{"number":"+14155551212","country":"US","monthly_rental_rate":"1.00"}],"meta":{"next":""}}`))
	}))
	defer srv.Close()

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
	root.SetArgs([]string{"numbers", "list", "--output", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), `"+14155551212"`) {
		t.Fatalf("smoke output unexpected:\n%s", buf.String())
	}
}

func TestSmoke_CallsListEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"uuid":"c1","caller_id_number":"+1","destination_number":"+2","duration":10,"cost":"0.01","hangup_cause":"NORMAL_CLEARING"}],"pagination":{"page":1,"per_page":20,"total":1,"pages":1},"success":true}`))
	}))
	defer srv.Close()

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
	root.SetArgs([]string{"calls", "list", "--output", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), `"c1"`) {
		t.Fatalf("smoke output unexpected:\n%s", buf.String())
	}
}
```

- [ ] **Step 2: Run the smoke tests**

```
go test ./cmd/ -run Smoke -v
```
Expected: 3 smoke tests PASS (the original `TestSmoke_AccountGetEndToEnd` plus the two new ones).

- [ ] **Step 3: Commit**

```
git add cmd/smoke_test.go
git commit -m "test(smoke): cover numbers list and calls list end-to-end"
```

---

## Task 10: README update + manual verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Build the binary and exercise the new commands**

```
go build -o vobiz .
./vobiz --help            # confirm `numbers` and `calls` appear
./vobiz numbers --help    # confirm `list`, `search`, `buy`, `release`
./vobiz calls --help      # confirm `make`, `list`, `get`, `recordings`
./vobiz calls recordings --help
```

Expected: all subcommands listed in their parents' help output. No build errors. Delete the binary afterwards (`rm vobiz` / `rm vobiz.exe`).

- [ ] **Step 2: Extend the README "First steps" section**

In `README.md`, replace the `## First steps` block with:
````markdown
## First steps

```bash
vobiz auth login                                # paste your Vobiz Auth ID and Token
vobiz account get
vobiz account balance
vobiz account transactions list --limit 20

vobiz numbers list
vobiz numbers search --country US
vobiz numbers buy +14155551212
vobiz numbers release +14155551212

vobiz calls make --from +14150000000 --to +14155551212 \
                 --answer-url https://example.com/answer.xml
vobiz calls list --direction outbound --per-page 20
vobiz calls get <call-uuid>
vobiz calls recordings list
vobiz calls recordings download <recording-id> -f recording.mp3

vobiz docs search "sip trunk"
vobiz docs open /trunks
```
````

Also update the Roadmap section to reflect that Plan 2 has landed:
````markdown
## Roadmap

See `docs/superpowers/specs/2026-05-23-vobiz-cli-design.md`. Upcoming plans add:

- ~~`calls`, `numbers` (Plan 2)~~ ✅ shipped
- `trunks`, `applications` (Plan 3)
- `whatsapp`, `partner` (Plan 4)
- GoReleaser, Homebrew tap, Docker image, install script (Plan 5)
````

- [ ] **Step 3: Optional — exercise against a real account**

If you have a Vobiz test account with credentials in `~/.vobiz/config.yaml`:

```
./vobiz numbers list
./vobiz numbers search --country IN
./vobiz calls list --per-page 5
```

If any of these fail with a 404 or 422, the assumed REST path for that resource (see §"Vobiz API surface used in this plan" above) is wrong. Fix the path constant in `internal/client/<resource>.go`, update the corresponding test, and re-run before declaring Plan 2 done. Commit the fix separately with a message like `fix(numbers): correct REST path discovered against live API`.

- [ ] **Step 4: Commit**

```
git add README.md
git commit -m "docs: README usage for numbers and calls"
```

---

## Follow-on plans

| # | Plan filename | Scope |
|---|---|---|
| 3 | `YYYY-MM-DD-vobiz-cli-trunks-applications.md` | `cmd/trunks` (CRUD + credentials/ip-acl/origination), `cmd/applications` (CRUD + attach/detach) |
| 4 | `YYYY-MM-DD-vobiz-cli-whatsapp-partner.md` | `cmd/whatsapp` (send {text,media,template}, templates, campaigns, contacts), `cmd/partner` (customers, balance transfer, numbers, cdrs, analytics) |
| 5 | `YYYY-MM-DD-vobiz-cli-release.md` | `Dockerfile` (distroless multi-stage), `.goreleaser.yaml` (linux/darwin/windows × amd64/arm64 + Homebrew tap + GHCR Docker), `install.sh`, release workflow on tag `v*` |

Each follow-on plan reuses the patterns established by Plan 1 and Plan 2:

- New resource → add `XAPI` interface + struct in `internal/client/<name>.go`; mock it in `cmd/<name>/<name>_test.go`.
- New verb → one file per verb under `cmd/<name>/`, calling `runtime.NewClient(Overrides)`.
- New paginated endpoint → use `paginate.AllN` like `cmd/account/transactions.go` and `cmd/numbers/list.go`.
- New output → add a `[]Column` slice at the call site; `Render` validates field names and emits a single object (not an array) for non-slice inputs.
- New error class → already covered by `internal/httpx.classify` and `internal/errors.ExitCode`.
- New mutation → `httpx` auto-generates an `Idempotency-Key`; flag this in the command's `--help` if it's destructive.
