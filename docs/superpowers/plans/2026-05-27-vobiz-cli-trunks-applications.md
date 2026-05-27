# Vobiz CLI — Plan 3: Trunks + Applications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `vobiz trunks {list,get,create,update,delete}` and `vobiz applications {list,get,create,update,delete,attach,detach}` on top of Plans 1 + 2. Trunk sub-resources (credentials, IP-ACL, origination URIs) are explicitly deferred to a follow-on plan once their REST paths are confirmed against a live account — the public docs reference them by name but don't enumerate concrete API routes.

**Architecture:** Same pattern Plan 2 established. Each resource adds (a) a typed `*API` interface + struct in `internal/client/`, and (b) one verb-per-file under `cmd/<resource>/`. Subcommands route through `runtime.NewClient(Overrides)`.

**Tech Stack:** Unchanged from Plan 2 — Go 1.24+, spf13/cobra, gopkg.in/yaml.v3, jedib0t/go-pretty, charmbracelet/glamour, google/uuid, golang.org/x/term.

**Companion docs:**
- Design spec: `docs/superpowers/specs/2026-05-23-vobiz-cli-design.md`
- Plan 1 (merged): `docs/superpowers/plans/2026-05-23-vobiz-cli-foundation.md`
- Plan 2 (merged): `docs/superpowers/plans/2026-05-27-vobiz-cli-calls-numbers.md`

---

## Vobiz API surface used in this plan

### Applications
Confirmed verbatim from `https://docs.vobiz.ai/applications/create-application`:

| Operation | Method | Path |
|---|---|---|
| Create application | `POST` | `/Account/{auth_id}/Application/` |

Inferred from Plivo-style conventions (the docs index lists doc routes like `/applications/retrieve-application` rather than REST paths — verify against live API or `llms.txt` before merge):

| Operation | Method | Path |
|---|---|---|
| List applications | `GET` | `/Account/{auth_id}/Application/` |
| Retrieve application | `GET` | `/Account/{auth_id}/Application/{app_id}/` |
| Update application | `POST` | `/Account/{auth_id}/Application/{app_id}/` |
| Delete application | `DELETE` | `/Account/{auth_id}/Application/{app_id}/` |
| Attach number to app | `POST` | `/Account/{auth_id}/Number/{number}/` (body: `{"app_id": "..."}`) |
| Detach number from app | `POST` | `/Account/{auth_id}/Number/{number}/` (body: `{"app_id": ""}`) |

**Application Object fields** (verbatim from the create-application page):
`app_name`, `answer_url`, `answer_method`, `hangup_url`, `hangup_method`, `fallback_answer_url`, `fallback_method`, `message_url`, `message_method`, `default_number_app`, `default_endpoint_app`, `subaccount`, `application_type`, `default_app`, `enabled`, `log_incoming_messages`, `public_uri`, `sip_transfer_method`, `sip_transfer_url`, `sip_uri`. Response: `{api_id, app_id, message}`.

### Trunks
**Trunk Object fields** (verbatim from `https://docs.vobiz.ai/trunks/trunk-object`):
`trunk_id`, `account_id`, `name`, `trunk_domain`, `trunk_status`, `secure`, `trunk_direction`, `concurrent_calls_limit` (default 10), `cps_limit` (default 2), `description`, `transport` (`udp`|`tcp`), `recording`, `enable_transcription`, `pii_redaction`, `webhook_method`, `recording_webhook_enabled`, `credential_uuid`, `primary_uri_uuid`, `inbound_destination`, `created_at`, `updated_at`.

Inferred CRUD paths (verify before merge):

| Operation | Method | Path |
|---|---|---|
| List trunks | `GET` | `/Account/{auth_id}/Trunk/` |
| Retrieve trunk | `GET` | `/Account/{auth_id}/Trunk/{trunk_id}/` |
| Create trunk | `POST` | `/Account/{auth_id}/Trunk/` |
| Update trunk | `POST` | `/Account/{auth_id}/Trunk/{trunk_id}/` |
| Delete trunk | `DELETE` | `/Account/{auth_id}/Trunk/{trunk_id}/` |

**Verify-before-merge:** Task 1 includes a hard requirement to confirm the trunk and application paths against `https://docs.vobiz.ai/llms.txt` (search for "Trunk" / "Application") or a real account. If anything differs, update the path constants in `internal/client/trunks.go` / `applications.go` and the corresponding tests — no other code changes.

### Deferred to a follow-on plan (3.5)
- Trunk **credentials** sub-resource (`/Account/{id}/Credential/`?)
- Trunk **IP ACL** sub-resource
- Trunk **origination URI** sub-resource
- Trunk **webhook** management

These are listed in the design spec §5 but their REST paths can't be confirmed from the current docs index. Implement once verified against a real account.

---

## Conventions used throughout

- **Module path:** `github.com/yash-kavaiya/vobiz-cli` (unchanged).
- **One commit per task.** Stage explicit file lists.
- **TDD:** failing test first, minimal impl, green, commit.
- **Branch:** `feature/trunks-applications` off `main` before Task 1.
- **Windows note:** mode-bit assertions stay gated on `runtime.GOOS != "windows"`.
- **Go PATH note for subagent shells:** if `go version` fails, run `$env:PATH = "C:\Go\bin;" + $env:PATH` (PowerShell) or `export PATH="/c/Go/bin:$PATH"` (bash).

---

## Task 1: Create branch + add `internal/client/trunks.go`

**Files:**
- Create: `internal/client/trunks.go`
- Create: `internal/client/trunks_test.go`
- Modify: `internal/client/client.go`

- [ ] **Step 1: Branch off main**

```
git checkout main
git pull --ff-only
git checkout -b feature/trunks-applications
```

- [ ] **Step 2: Write the failing test**

Create `internal/client/trunks_test.go`:
```go
package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/auth"
)

func TestTrunks_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/Trunk/" || r.Method != http.MethodGet {
			t.Errorf("path/method = %q/%q", r.URL.Path, r.Method)
		}
		_, _ = w.Write([]byte(`{
		  "objects":[{"trunk_id":"t1","name":"Outbound-A","trunk_direction":"outbound","cps_limit":10,"concurrent_calls_limit":50}],
		  "meta":{"next":""}
		}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	rows, _, err := c.Trunks.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].TrunkID != "t1" || rows[0].Name != "Outbound-A" {
		t.Fatalf("rows: %+v", rows)
	}
}

func TestTrunks_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/Trunk/t1/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"trunk_id":"t1","name":"Outbound-A","trunk_direction":"outbound","cps_limit":10}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Trunks.Get(context.Background(), "t1")
	if err != nil {
		t.Fatal(err)
	}
	if got.TrunkID != "t1" {
		t.Fatalf("got %+v", got)
	}
}

func TestTrunks_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/Account/AB12/Trunk/" {
			t.Errorf("path/method = %q/%q", r.URL.Path, r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if req["name"] != "New-Outbound" || req["trunk_direction"] != "outbound" {
			t.Errorf("body wrong: %+v", req)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"trunk_id":"t99","name":"New-Outbound","trunk_direction":"outbound"}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Trunks.Create(context.Background(), TrunkParams{
		Name: "New-Outbound", TrunkDirection: "outbound",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.TrunkID != "t99" {
		t.Fatalf("got %+v", got)
	}
}

func TestTrunks_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/Account/AB12/Trunk/t1/" {
			t.Errorf("path/method = %q/%q", r.URL.Path, r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if req["cps_limit"] != float64(25) {
			t.Errorf("body wrong: %+v", req)
		}
		_, _ = w.Write([]byte(`{"trunk_id":"t1","name":"Outbound-A","cps_limit":25}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Trunks.Update(context.Background(), "t1", TrunkParams{CPSLimit: 25})
	if err != nil {
		t.Fatal(err)
	}
	if got.CPSLimit != 25 {
		t.Fatalf("got %+v", got)
	}
}

func TestTrunks_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/Account/AB12/Trunk/t1/" {
			t.Errorf("path/method = %q/%q", r.URL.Path, r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	if err := c.Trunks.Delete(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```
go test ./internal/client/...
```
Expected: build failure (`c.Trunks undefined`).

- [ ] **Step 4: Write the package**

Create `internal/client/trunks.go`:
```go
package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
)

type Trunk struct {
	TrunkID                 string `json:"trunk_id"                yaml:"trunk_id"`
	AccountID               string `json:"account_id,omitempty"    yaml:"account_id,omitempty"`
	Name                    string `json:"name"                    yaml:"name"`
	TrunkDomain             string `json:"trunk_domain,omitempty"  yaml:"trunk_domain,omitempty"`
	TrunkStatus             string `json:"trunk_status,omitempty"  yaml:"trunk_status,omitempty"`
	Secure                  bool   `json:"secure,omitempty"        yaml:"secure,omitempty"`
	TrunkDirection          string `json:"trunk_direction"         yaml:"trunk_direction"`
	ConcurrentCallsLimit    int    `json:"concurrent_calls_limit,omitempty" yaml:"concurrent_calls_limit,omitempty"`
	CPSLimit                int    `json:"cps_limit,omitempty"     yaml:"cps_limit,omitempty"`
	Description             string `json:"description,omitempty"   yaml:"description,omitempty"`
	Transport               string `json:"transport,omitempty"     yaml:"transport,omitempty"`
	Recording               bool   `json:"recording,omitempty"     yaml:"recording,omitempty"`
	EnableTranscription     bool   `json:"enable_transcription,omitempty" yaml:"enable_transcription,omitempty"`
	PIIRedaction            bool   `json:"pii_redaction,omitempty" yaml:"pii_redaction,omitempty"`
	WebhookMethod           string `json:"webhook_method,omitempty" yaml:"webhook_method,omitempty"`
	RecordingWebhookEnabled bool   `json:"recording_webhook_enabled,omitempty" yaml:"recording_webhook_enabled,omitempty"`
	CredentialUUID          string `json:"credential_uuid,omitempty" yaml:"credential_uuid,omitempty"`
	PrimaryURIUUID          string `json:"primary_uri_uuid,omitempty" yaml:"primary_uri_uuid,omitempty"`
	InboundDestination      string `json:"inbound_destination,omitempty" yaml:"inbound_destination,omitempty"`
	CreatedAt               string `json:"created_at,omitempty"    yaml:"created_at,omitempty"`
	UpdatedAt               string `json:"updated_at,omitempty"    yaml:"updated_at,omitempty"`
}

// TrunkParams is the request body shape for Create/Update. Zero-valued fields
// are omitted via omitempty so partial updates only send what changed.
type TrunkParams struct {
	Name                 string `json:"name,omitempty"`
	TrunkDirection       string `json:"trunk_direction,omitempty"`
	Description          string `json:"description,omitempty"`
	Secure               bool   `json:"secure,omitempty"`
	Transport            string `json:"transport,omitempty"`
	ConcurrentCallsLimit int    `json:"concurrent_calls_limit,omitempty"`
	CPSLimit             int    `json:"cps_limit,omitempty"`
	Recording            bool   `json:"recording,omitempty"`
	EnableTranscription  bool   `json:"enable_transcription,omitempty"`
	PIIRedaction         bool   `json:"pii_redaction,omitempty"`
	InboundDestination   string `json:"inbound_destination,omitempty"`
}

type TrunksAPI interface {
	List(ctx context.Context, cursor string) ([]Trunk, string, error)
	Get(ctx context.Context, trunkID string) (*Trunk, error)
	Create(ctx context.Context, p TrunkParams) (*Trunk, error)
	Update(ctx context.Context, trunkID string, p TrunkParams) (*Trunk, error)
	Delete(ctx context.Context, trunkID string) error
}

type trunksAPI struct {
	http   *httpx.Client
	authID string
}

func (t *trunksAPI) base() string { return "/Account/" + t.authID + "/Trunk/" }

func (t *trunksAPI) List(ctx context.Context, cursor string) ([]Trunk, string, error) {
	path := t.base()
	if cursor != "" {
		path += "?cursor=" + url.QueryEscape(cursor)
	}
	var raw struct {
		Objects []Trunk `json:"objects"`
		Meta    struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	if err := t.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, "", err
	}
	return raw.Objects, raw.Meta.Next, nil
}

func (t *trunksAPI) Get(ctx context.Context, trunkID string) (*Trunk, error) {
	var out Trunk
	if err := t.http.DoJSON(ctx, http.MethodGet, t.base()+url.PathEscape(trunkID)+"/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *trunksAPI) Create(ctx context.Context, p TrunkParams) (*Trunk, error) {
	var out Trunk
	if err := t.http.DoJSON(ctx, http.MethodPost, t.base(), p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *trunksAPI) Update(ctx context.Context, trunkID string, p TrunkParams) (*Trunk, error) {
	var out Trunk
	if err := t.http.DoJSON(ctx, http.MethodPost, t.base()+url.PathEscape(trunkID)+"/", p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (t *trunksAPI) Delete(ctx context.Context, trunkID string) error {
	return t.http.DoJSON(ctx, http.MethodDelete, t.base()+url.PathEscape(trunkID)+"/", nil, nil)
}
```

- [ ] **Step 5: Wire `Trunks` into the Client struct**

Modify `internal/client/client.go` to add `Trunks TrunksAPI` and the `&trunksAPI{...}` constructor entry (follow the existing pattern for Account/Numbers/Calls/Recordings).

- [ ] **Step 6: Verify paths against `llms.txt`**

Fetch `https://docs.vobiz.ai/llms.txt` and grep for "Trunk". If the live REST surface differs from `/Account/{id}/Trunk/`, update the `t.base()` constant and re-run tests. Note any deviation in the commit body.

- [ ] **Step 7: Run tests**

```
go test ./internal/client/...
```
Expected: PASS.

- [ ] **Step 8: Commit**

```
git add internal/client/trunks.go internal/client/trunks_test.go internal/client/client.go
git commit -m "feat(client): typed Trunks API (list, get, create, update, delete)"
```

---

## Task 2: `vobiz trunks {list,get,create,update,delete}`

**Files:**
- Create: `cmd/trunks/trunks.go`
- Create: `cmd/trunks/list.go`
- Create: `cmd/trunks/get.go`
- Create: `cmd/trunks/create.go`
- Create: `cmd/trunks/update.go`
- Create: `cmd/trunks/delete.go`
- Create: `cmd/trunks/trunks_test.go`
- Modify: `cmd/registrations.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/trunks/trunks_test.go`:
```go
package trunks

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

type fakeTrunks struct {
	rows      []client.Trunk
	one       *client.Trunk
	created   client.TrunkParams
	updatedID string
	deletedID string
}

func (f *fakeTrunks) List(_ context.Context, _ string) ([]client.Trunk, string, error) {
	return f.rows, "", nil
}
func (f *fakeTrunks) Get(_ context.Context, _ string) (*client.Trunk, error) { return f.one, nil }
func (f *fakeTrunks) Create(_ context.Context, p client.TrunkParams) (*client.Trunk, error) {
	f.created = p
	return &client.Trunk{TrunkID: "new", Name: p.Name, TrunkDirection: p.TrunkDirection}, nil
}
func (f *fakeTrunks) Update(_ context.Context, id string, _ client.TrunkParams) (*client.Trunk, error) {
	f.updatedID = id
	return f.one, nil
}
func (f *fakeTrunks) Delete(_ context.Context, id string) error { f.deletedID = id; return nil }

func TestList_Renders(t *testing.T) {
	f := &fakeTrunks{rows: []client.Trunk{
		{TrunkID: "t1", Name: "Outbound-A", TrunkDirection: "outbound", CPSLimit: 10},
	}}
	var out bytes.Buffer
	if err := runList(f, &out, "table", 50, false); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"t1", "Outbound-A", "outbound", "10"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q:\n%s", w, out.String())
		}
	}
}

func TestGet_Renders(t *testing.T) {
	f := &fakeTrunks{one: &client.Trunk{TrunkID: "t1", Name: "Outbound-A", TrunkDomain: "abc.sip.vobiz.ai"}}
	var out bytes.Buffer
	if err := runGet(f, &out, "table", "t1"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "abc.sip.vobiz.ai") {
		t.Fatalf("missing domain:\n%s", out.String())
	}
}

func TestCreate_PassesParams(t *testing.T) {
	f := &fakeTrunks{}
	var out bytes.Buffer
	if err := runCreate(f, &out, "table", createFlags{Name: "Foo", Direction: "outbound", CPSLimit: 5}); err != nil {
		t.Fatal(err)
	}
	if f.created.Name != "Foo" || f.created.TrunkDirection != "outbound" || f.created.CPSLimit != 5 {
		t.Fatalf("created = %+v", f.created)
	}
}

func TestUpdate_CallsAPI(t *testing.T) {
	f := &fakeTrunks{one: &client.Trunk{TrunkID: "t1"}}
	var out bytes.Buffer
	if err := runUpdate(f, &out, "table", "t1", createFlags{CPSLimit: 99}); err != nil {
		t.Fatal(err)
	}
	if f.updatedID != "t1" {
		t.Fatalf("updatedID = %q", f.updatedID)
	}
}

func TestDelete_CallsAPI(t *testing.T) {
	f := &fakeTrunks{}
	var out bytes.Buffer
	if err := runDelete(f, &out, "t1"); err != nil {
		t.Fatal(err)
	}
	if f.deletedID != "t1" {
		t.Fatalf("deletedID = %q", f.deletedID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./cmd/trunks/...
```
Expected: build failure.

- [ ] **Step 3: Write the package files**

Create `cmd/trunks/trunks.go`:
```go
// Package trunks implements `vobiz trunks …` subcommands.
package trunks

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

var Overrides runtime.Overrides

var TrunksFactory = func() (client.TrunksAPI, error) {
	c, err := runtime.NewClient(Overrides)
	if err != nil {
		return nil, err
	}
	return c.Trunks, nil
}

func Register(parent *cobra.Command, format func() string, ov func() runtime.Overrides) {
	cmd := &cobra.Command{
		Use:   "trunks",
		Short: "Manage SIP trunks",
	}
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		Overrides = ov()
		return nil
	}
	cmd.AddCommand(newListCmd(format))
	cmd.AddCommand(newGetCmd(format))
	cmd.AddCommand(newCreateCmd(format))
	cmd.AddCommand(newUpdateCmd(format))
	cmd.AddCommand(newDeleteCmd())
	parent.AddCommand(cmd)
}
```

Create `cmd/trunks/list.go`:
```go
package trunks

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
		Short: "List SIP trunks (paginated)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := TrunksFactory()
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

func runList(api client.TrunksAPI, w io.Writer, format string, limit int, all bool) error {
	fetch := func(ctx context.Context, cursor string) (paginate.Page[client.Trunk], error) {
		items, next, err := api.List(ctx, cursor)
		if err != nil {
			return paginate.Page[client.Trunk]{}, err
		}
		return paginate.Page[client.Trunk]{Items: items, NextCursor: next}, nil
	}
	var (
		rows []client.Trunk
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
		{Header: "ID", Field: "TrunkID"},
		{Header: "NAME", Field: "Name"},
		{Header: "DIR", Field: "TrunkDirection"},
		{Header: "CPS", Field: "CPSLimit"},
		{Header: "CONCURRENT", Field: "ConcurrentCallsLimit"},
		{Header: "STATUS", Field: "TrunkStatus"},
	}
	return output.Render(w, rows, cols, f)
}
```

Create `cmd/trunks/get.go`:
```go
package trunks

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newGetCmd(format func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <trunk-id>",
		Short: "Show a single trunk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := TrunksFactory()
			if err != nil {
				return err
			}
			return runGet(a, cmd.OutOrStdout(), format(), args[0])
		},
	}
}

func runGet(api client.TrunksAPI, w io.Writer, format, id string) error {
	tr, err := api.Get(context.Background(), id)
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "ID", Field: "TrunkID"},
		{Header: "NAME", Field: "Name"},
		{Header: "DIR", Field: "TrunkDirection"},
		{Header: "DOMAIN", Field: "TrunkDomain"},
		{Header: "CPS", Field: "CPSLimit"},
		{Header: "CONCURRENT", Field: "ConcurrentCallsLimit"},
		{Header: "TRANSPORT", Field: "Transport"},
		{Header: "STATUS", Field: "TrunkStatus"},
	}
	return output.Render(w, *tr, cols, f)
}
```

Create `cmd/trunks/create.go`:
```go
package trunks

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

// createFlags is shared between create and update so the same field set
// is available to both commands.
type createFlags struct {
	Name                 string
	Direction            string
	Description          string
	Secure               bool
	Transport            string
	ConcurrentCallsLimit int
	CPSLimit             int
	Recording            bool
	EnableTranscription  bool
	PIIRedaction         bool
	InboundDestination   string
}

func (f createFlags) toParams() client.TrunkParams {
	return client.TrunkParams{
		Name:                 f.Name,
		TrunkDirection:       f.Direction,
		Description:          f.Description,
		Secure:               f.Secure,
		Transport:            f.Transport,
		ConcurrentCallsLimit: f.ConcurrentCallsLimit,
		CPSLimit:             f.CPSLimit,
		Recording:            f.Recording,
		EnableTranscription:  f.EnableTranscription,
		PIIRedaction:         f.PIIRedaction,
		InboundDestination:   f.InboundDestination,
	}
}

func addCreateFlags(cmd *cobra.Command, f *createFlags) {
	cmd.Flags().StringVar(&f.Name, "name", "", "trunk name (max 255 chars)")
	cmd.Flags().StringVar(&f.Direction, "direction", "", "inbound | outbound | both")
	cmd.Flags().StringVar(&f.Description, "description", "", "free-text description")
	cmd.Flags().BoolVar(&f.Secure, "secure", false, "enable TLS/SRTP")
	cmd.Flags().StringVar(&f.Transport, "transport", "", "SIP transport: udp | tcp")
	cmd.Flags().IntVar(&f.ConcurrentCallsLimit, "concurrent-calls-limit", 0, "max simultaneous calls")
	cmd.Flags().IntVar(&f.CPSLimit, "cps-limit", 0, "calls-per-second limit")
	cmd.Flags().BoolVar(&f.Recording, "recording", false, "enable call recording")
	cmd.Flags().BoolVar(&f.EnableTranscription, "enable-transcription", false, "enable transcription")
	cmd.Flags().BoolVar(&f.PIIRedaction, "pii-redaction", false, "redact PII from transcripts")
	cmd.Flags().StringVar(&f.InboundDestination, "inbound-destination", "", "destination for inbound routing")
}

func newCreateCmd(format func() string) *cobra.Command {
	var f createFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new SIP trunk",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := TrunksFactory()
			if err != nil {
				return err
			}
			return runCreate(a, cmd.OutOrStdout(), format(), f)
		},
	}
	addCreateFlags(cmd, &f)
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("direction")
	return cmd
}

func runCreate(api client.TrunksAPI, w io.Writer, format string, f createFlags) error {
	tr, err := api.Create(context.Background(), f.toParams())
	if err != nil {
		return err
	}
	fm, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "ID", Field: "TrunkID"},
		{Header: "NAME", Field: "Name"},
		{Header: "DIR", Field: "TrunkDirection"},
		{Header: "DOMAIN", Field: "TrunkDomain"},
	}
	fmt.Fprintln(w, "Created.")
	return output.Render(w, *tr, cols, fm)
}
```

Create `cmd/trunks/update.go`:
```go
package trunks

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newUpdateCmd(format func() string) *cobra.Command {
	var f createFlags
	cmd := &cobra.Command{
		Use:   "update <trunk-id>",
		Short: "Update an existing SIP trunk (only specified flags are sent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := TrunksFactory()
			if err != nil {
				return err
			}
			return runUpdate(a, cmd.OutOrStdout(), format(), args[0], f)
		},
	}
	addCreateFlags(cmd, &f)
	return cmd
}

func runUpdate(api client.TrunksAPI, w io.Writer, format, id string, f createFlags) error {
	tr, err := api.Update(context.Background(), id, f.toParams())
	if err != nil {
		return err
	}
	fm, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "ID", Field: "TrunkID"},
		{Header: "NAME", Field: "Name"},
		{Header: "DIR", Field: "TrunkDirection"},
		{Header: "CPS", Field: "CPSLimit"},
		{Header: "CONCURRENT", Field: "ConcurrentCallsLimit"},
	}
	return output.Render(w, *tr, cols, fm)
}
```

Create `cmd/trunks/delete.go`:
```go
package trunks

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <trunk-id>",
		Short: "Delete a SIP trunk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := TrunksFactory()
			if err != nil {
				return err
			}
			return runDelete(a, cmd.OutOrStdout(), args[0])
		},
	}
}

func runDelete(api client.TrunksAPI, w io.Writer, id string) error {
	if err := api.Delete(context.Background(), id); err != nil {
		return err
	}
	fmt.Fprintf(w, "Deleted trunk %s.\n", id)
	return nil
}
```

- [ ] **Step 4: Wire into `cmd/registrations.go` and `cmd/root.go`**

In `cmd/registrations.go` add the `trunks` import and a `registerTrunks` function (matching the existing `registerCalls` shape).

In `cmd/root.go` add `registerTrunks(root)` after `registerCalls(root)`.

- [ ] **Step 5: Run tests**

```
go test ./cmd/trunks/... ./cmd/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add cmd/trunks/ cmd/registrations.go cmd/root.go
git commit -m "feat(trunks): list, get, create, update, delete subcommands"
```

---

## Task 3: Add `internal/client/applications.go`

**Files:**
- Create: `internal/client/applications.go`
- Create: `internal/client/applications_test.go`
- Modify: `internal/client/client.go`

- [ ] **Step 1: Write the failing test**

Create `internal/client/applications_test.go`:
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

func TestApplications_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/Application/" || r.Method != http.MethodGet {
			t.Errorf("path/method = %q/%q", r.URL.Path, r.Method)
		}
		_, _ = w.Write([]byte(`{
		  "objects":[{"app_id":"a1","app_name":"main-flow","answer_url":"https://x/ans","enabled":true}],
		  "meta":{"next":""}
		}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	rows, _, err := c.Applications.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].AppID != "a1" {
		t.Fatalf("rows: %+v", rows)
	}
}

func TestApplications_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Account/AB12/Application/a1/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"app_id":"a1","app_name":"main-flow","answer_url":"https://x/ans"}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Applications.Get(context.Background(), "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.AppName != "main-flow" {
		t.Fatalf("got %+v", got)
	}
}

func TestApplications_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/Account/AB12/Application/" {
			t.Errorf("path/method = %q/%q", r.URL.Path, r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if req["app_name"] != "new-app" {
			t.Errorf("body wrong: %+v", req)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"api_id":"ai1","app_id":"new","message":"created"}`))
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	got, err := c.Applications.Create(context.Background(), AppParams{
		AppName: "new-app", AnswerURL: "https://x/ans",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AppID != "new" {
		t.Fatalf("got %+v", got)
	}
}

func TestApplications_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/Account/AB12/Application/a1/" {
			t.Errorf("path/method = %q/%q", r.URL.Path, r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	if err := c.Applications.Delete(context.Background(), "a1"); err != nil {
		t.Fatal(err)
	}
}

func TestApplications_AttachNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/Account/AB12/Number/") {
			t.Errorf("path = %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if req["app_id"] != "a1" {
			t.Errorf("body wrong: %+v", req)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	if err := c.Applications.AttachNumber(context.Background(), "a1", "+14155551212"); err != nil {
		t.Fatal(err)
	}
}

func TestApplications_DetachNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		// Detach is implemented as "clear the app_id binding"
		if req["app_id"] != "" {
			t.Errorf("detach should send empty app_id, got: %+v", req)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(auth.Credentials{AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL})
	if err := c.Applications.DetachNumber(context.Background(), "+14155551212"); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/client/...
```
Expected: build failure.

- [ ] **Step 3: Write the package**

Create `internal/client/applications.go`:
```go
package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/yash-kavaiya/vobiz-cli/internal/httpx"
)

type Application struct {
	APIID               string `json:"api_id,omitempty"               yaml:"api_id,omitempty"`
	AppID               string `json:"app_id"                         yaml:"app_id"`
	AppName             string `json:"app_name"                       yaml:"app_name"`
	AnswerURL           string `json:"answer_url"                     yaml:"answer_url"`
	AnswerMethod        string `json:"answer_method,omitempty"        yaml:"answer_method,omitempty"`
	HangupURL           string `json:"hangup_url,omitempty"           yaml:"hangup_url,omitempty"`
	HangupMethod        string `json:"hangup_method,omitempty"        yaml:"hangup_method,omitempty"`
	FallbackAnswerURL   string `json:"fallback_answer_url,omitempty"  yaml:"fallback_answer_url,omitempty"`
	FallbackMethod      string `json:"fallback_method,omitempty"      yaml:"fallback_method,omitempty"`
	MessageURL          string `json:"message_url,omitempty"          yaml:"message_url,omitempty"`
	MessageMethod       string `json:"message_method,omitempty"       yaml:"message_method,omitempty"`
	DefaultNumberApp    bool   `json:"default_number_app,omitempty"   yaml:"default_number_app,omitempty"`
	DefaultEndpointApp  bool   `json:"default_endpoint_app,omitempty" yaml:"default_endpoint_app,omitempty"`
	Subaccount          string `json:"subaccount,omitempty"           yaml:"subaccount,omitempty"`
	ApplicationType     string `json:"application_type,omitempty"     yaml:"application_type,omitempty"`
	DefaultApp          bool   `json:"default_app,omitempty"          yaml:"default_app,omitempty"`
	Enabled             bool   `json:"enabled,omitempty"              yaml:"enabled,omitempty"`
	LogIncomingMessages bool   `json:"log_incoming_messages,omitempty" yaml:"log_incoming_messages,omitempty"`
	PublicURI           bool   `json:"public_uri,omitempty"           yaml:"public_uri,omitempty"`
	SIPTransferMethod   string `json:"sip_transfer_method,omitempty"  yaml:"sip_transfer_method,omitempty"`
	SIPTransferURL      string `json:"sip_transfer_url,omitempty"     yaml:"sip_transfer_url,omitempty"`
	SIPURI              string `json:"sip_uri,omitempty"              yaml:"sip_uri,omitempty"`
	Message             string `json:"message,omitempty"              yaml:"message,omitempty"`
}

// AppParams is the request body for Create/Update.
type AppParams struct {
	AppName             string `json:"app_name,omitempty"`
	AnswerURL           string `json:"answer_url,omitempty"`
	AnswerMethod        string `json:"answer_method,omitempty"`
	HangupURL           string `json:"hangup_url,omitempty"`
	HangupMethod        string `json:"hangup_method,omitempty"`
	FallbackAnswerURL   string `json:"fallback_answer_url,omitempty"`
	FallbackMethod      string `json:"fallback_method,omitempty"`
	MessageURL          string `json:"message_url,omitempty"`
	MessageMethod       string `json:"message_method,omitempty"`
	DefaultNumberApp    bool   `json:"default_number_app,omitempty"`
	DefaultEndpointApp  bool   `json:"default_endpoint_app,omitempty"`
	Subaccount          string `json:"subaccount,omitempty"`
	ApplicationType     string `json:"application_type,omitempty"`
	DefaultApp          bool   `json:"default_app,omitempty"`
	Enabled             bool   `json:"enabled,omitempty"`
	LogIncomingMessages bool   `json:"log_incoming_messages,omitempty"`
	PublicURI           bool   `json:"public_uri,omitempty"`
	SIPTransferMethod   string `json:"sip_transfer_method,omitempty"`
	SIPTransferURL      string `json:"sip_transfer_url,omitempty"`
	SIPURI              string `json:"sip_uri,omitempty"`
}

type ApplicationsAPI interface {
	List(ctx context.Context, cursor string) ([]Application, string, error)
	Get(ctx context.Context, appID string) (*Application, error)
	Create(ctx context.Context, p AppParams) (*Application, error)
	Update(ctx context.Context, appID string, p AppParams) (*Application, error)
	Delete(ctx context.Context, appID string) error
	AttachNumber(ctx context.Context, appID, number string) error
	DetachNumber(ctx context.Context, number string) error
}

type applicationsAPI struct {
	http   *httpx.Client
	authID string
}

func (a *applicationsAPI) base() string  { return "/Account/" + a.authID + "/Application/" }
func (a *applicationsAPI) numURL(n string) string {
	return "/Account/" + a.authID + "/Number/" + url.PathEscape(n) + "/"
}

func (a *applicationsAPI) List(ctx context.Context, cursor string) ([]Application, string, error) {
	path := a.base()
	if cursor != "" {
		path += "?cursor=" + url.QueryEscape(cursor)
	}
	var raw struct {
		Objects []Application `json:"objects"`
		Meta    struct {
			Next string `json:"next"`
		} `json:"meta"`
	}
	if err := a.http.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, "", err
	}
	return raw.Objects, raw.Meta.Next, nil
}

func (a *applicationsAPI) Get(ctx context.Context, appID string) (*Application, error) {
	var out Application
	if err := a.http.DoJSON(ctx, http.MethodGet, a.base()+url.PathEscape(appID)+"/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *applicationsAPI) Create(ctx context.Context, p AppParams) (*Application, error) {
	var out Application
	if err := a.http.DoJSON(ctx, http.MethodPost, a.base(), p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *applicationsAPI) Update(ctx context.Context, appID string, p AppParams) (*Application, error) {
	var out Application
	if err := a.http.DoJSON(ctx, http.MethodPost, a.base()+url.PathEscape(appID)+"/", p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *applicationsAPI) Delete(ctx context.Context, appID string) error {
	return a.http.DoJSON(ctx, http.MethodDelete, a.base()+url.PathEscape(appID)+"/", nil, nil)
}

// AttachNumber binds a phone number to an application by POSTing the app_id
// onto the Number resource. The convention follows Plivo: numbers carry their
// application binding directly.
func (a *applicationsAPI) AttachNumber(ctx context.Context, appID, number string) error {
	return a.http.DoJSON(ctx, http.MethodPost, a.numURL(number),
		map[string]any{"app_id": appID}, nil)
}

// DetachNumber clears the application binding on a number.
func (a *applicationsAPI) DetachNumber(ctx context.Context, number string) error {
	return a.http.DoJSON(ctx, http.MethodPost, a.numURL(number),
		map[string]any{"app_id": ""}, nil)
}
```

- [ ] **Step 4: Wire `Applications` into the Client struct**

Modify `internal/client/client.go` — add `Applications ApplicationsAPI` and the `&applicationsAPI{...}` constructor entry. Pattern matches Trunks.

- [ ] **Step 5: Run tests**

```
go test ./internal/client/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add internal/client/applications.go internal/client/applications_test.go internal/client/client.go
git commit -m "feat(client): typed Applications API (CRUD + attach/detach number)"
```

---

## Task 4: `vobiz applications {list,get,create,update,delete,attach,detach}`

**Files:**
- Create: `cmd/applications/applications.go`
- Create: `cmd/applications/list.go`
- Create: `cmd/applications/get.go`
- Create: `cmd/applications/create.go`
- Create: `cmd/applications/update.go`
- Create: `cmd/applications/delete.go`
- Create: `cmd/applications/attach.go`
- Create: `cmd/applications/detach.go`
- Create: `cmd/applications/applications_test.go`
- Modify: `cmd/registrations.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/applications/applications_test.go`:
```go
package applications

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

type fakeApps struct {
	rows       []client.Application
	one        *client.Application
	created    client.AppParams
	updatedID  string
	deletedID  string
	attachedTo string
	attachedNo string
	detachedNo string
}

func (f *fakeApps) List(_ context.Context, _ string) ([]client.Application, string, error) {
	return f.rows, "", nil
}
func (f *fakeApps) Get(_ context.Context, _ string) (*client.Application, error) { return f.one, nil }
func (f *fakeApps) Create(_ context.Context, p client.AppParams) (*client.Application, error) {
	f.created = p
	return &client.Application{AppID: "new", AppName: p.AppName, AnswerURL: p.AnswerURL}, nil
}
func (f *fakeApps) Update(_ context.Context, id string, _ client.AppParams) (*client.Application, error) {
	f.updatedID = id
	return f.one, nil
}
func (f *fakeApps) Delete(_ context.Context, id string) error { f.deletedID = id; return nil }
func (f *fakeApps) AttachNumber(_ context.Context, appID, n string) error {
	f.attachedTo, f.attachedNo = appID, n
	return nil
}
func (f *fakeApps) DetachNumber(_ context.Context, n string) error { f.detachedNo = n; return nil }

func TestList_Renders(t *testing.T) {
	f := &fakeApps{rows: []client.Application{
		{AppID: "a1", AppName: "main", AnswerURL: "https://x/ans", Enabled: true},
	}}
	var out bytes.Buffer
	if err := runList(f, &out, "table", 50, false); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"a1", "main", "https://x/ans"} {
		if !strings.Contains(out.String(), w) {
			t.Fatalf("missing %q:\n%s", w, out.String())
		}
	}
}

func TestCreate_PassesParams(t *testing.T) {
	f := &fakeApps{}
	var out bytes.Buffer
	if err := runCreate(f, &out, "table", createFlags{AppName: "new-app", AnswerURL: "https://x/ans"}); err != nil {
		t.Fatal(err)
	}
	if f.created.AppName != "new-app" || f.created.AnswerURL != "https://x/ans" {
		t.Fatalf("created = %+v", f.created)
	}
}

func TestAttach_CallsAPI(t *testing.T) {
	f := &fakeApps{}
	var out bytes.Buffer
	if err := runAttach(f, &out, "a1", "+14155551212"); err != nil {
		t.Fatal(err)
	}
	if f.attachedTo != "a1" || f.attachedNo != "+14155551212" {
		t.Fatalf("attached = %s/%s", f.attachedTo, f.attachedNo)
	}
}

func TestDetach_CallsAPI(t *testing.T) {
	f := &fakeApps{}
	var out bytes.Buffer
	if err := runDetach(f, &out, "+14155551212"); err != nil {
		t.Fatal(err)
	}
	if f.detachedNo != "+14155551212" {
		t.Fatalf("detached = %q", f.detachedNo)
	}
}

func TestDelete_CallsAPI(t *testing.T) {
	f := &fakeApps{}
	var out bytes.Buffer
	if err := runDelete(f, &out, "a1"); err != nil {
		t.Fatal(err)
	}
	if f.deletedID != "a1" {
		t.Fatalf("deletedID = %q", f.deletedID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./cmd/applications/...
```
Expected: build failure.

- [ ] **Step 3: Write the package**

Create `cmd/applications/applications.go`:
```go
// Package applications implements `vobiz applications …` subcommands.
package applications

import (
	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/cmd/runtime"
	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

var Overrides runtime.Overrides

var AppsFactory = func() (client.ApplicationsAPI, error) {
	c, err := runtime.NewClient(Overrides)
	if err != nil {
		return nil, err
	}
	return c.Applications, nil
}

func Register(parent *cobra.Command, format func() string, ov func() runtime.Overrides) {
	cmd := &cobra.Command{
		Use:   "applications",
		Short: "Manage Vobiz applications (XML answer-flow definitions)",
	}
	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		Overrides = ov()
		return nil
	}
	cmd.AddCommand(newListCmd(format))
	cmd.AddCommand(newGetCmd(format))
	cmd.AddCommand(newCreateCmd(format))
	cmd.AddCommand(newUpdateCmd(format))
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newAttachCmd())
	cmd.AddCommand(newDetachCmd())
	parent.AddCommand(cmd)
}
```

Create `cmd/applications/list.go`:
```go
package applications

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
		Short: "List applications (paginated)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := AppsFactory()
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

func runList(api client.ApplicationsAPI, w io.Writer, format string, limit int, all bool) error {
	fetch := func(ctx context.Context, cursor string) (paginate.Page[client.Application], error) {
		items, next, err := api.List(ctx, cursor)
		if err != nil {
			return paginate.Page[client.Application]{}, err
		}
		return paginate.Page[client.Application]{Items: items, NextCursor: next}, nil
	}
	var (
		rows []client.Application
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
		{Header: "APP ID", Field: "AppID"},
		{Header: "NAME", Field: "AppName"},
		{Header: "ANSWER URL", Field: "AnswerURL"},
		{Header: "ENABLED", Field: "Enabled"},
	}
	return output.Render(w, rows, cols, f)
}
```

Create `cmd/applications/get.go`:
```go
package applications

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newGetCmd(format func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <app-id>",
		Short: "Show a single application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := AppsFactory()
			if err != nil {
				return err
			}
			return runGet(a, cmd.OutOrStdout(), format(), args[0])
		},
	}
}

func runGet(api client.ApplicationsAPI, w io.Writer, format, id string) error {
	app, err := api.Get(context.Background(), id)
	if err != nil {
		return err
	}
	f, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "APP ID", Field: "AppID"},
		{Header: "NAME", Field: "AppName"},
		{Header: "ANSWER URL", Field: "AnswerURL"},
		{Header: "HANGUP URL", Field: "HangupURL"},
		{Header: "MESSAGE URL", Field: "MessageURL"},
		{Header: "ENABLED", Field: "Enabled"},
	}
	return output.Render(w, *app, cols, f)
}
```

Create `cmd/applications/create.go`:
```go
package applications

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

type createFlags struct {
	AppName             string
	AnswerURL           string
	AnswerMethod        string
	HangupURL           string
	HangupMethod        string
	FallbackAnswerURL   string
	MessageURL          string
	MessageMethod       string
	Enabled             bool
	DefaultNumberApp    bool
	DefaultEndpointApp  bool
	LogIncomingMessages bool
	PublicURI           bool
	ApplicationType     string
	Subaccount          string
}

func (f createFlags) toParams() client.AppParams {
	return client.AppParams{
		AppName:             f.AppName,
		AnswerURL:           f.AnswerURL,
		AnswerMethod:        f.AnswerMethod,
		HangupURL:           f.HangupURL,
		HangupMethod:        f.HangupMethod,
		FallbackAnswerURL:   f.FallbackAnswerURL,
		MessageURL:          f.MessageURL,
		MessageMethod:       f.MessageMethod,
		Enabled:             f.Enabled,
		DefaultNumberApp:    f.DefaultNumberApp,
		DefaultEndpointApp:  f.DefaultEndpointApp,
		LogIncomingMessages: f.LogIncomingMessages,
		PublicURI:           f.PublicURI,
		ApplicationType:     f.ApplicationType,
		Subaccount:          f.Subaccount,
	}
}

func addCreateFlags(cmd *cobra.Command, f *createFlags) {
	cmd.Flags().StringVar(&f.AppName, "name", "", "application name")
	cmd.Flags().StringVar(&f.AnswerURL, "answer-url", "", "URL returning Vobiz XML when calls arrive")
	cmd.Flags().StringVar(&f.AnswerMethod, "answer-method", "", "HTTP verb for --answer-url")
	cmd.Flags().StringVar(&f.HangupURL, "hangup-url", "", "URL notified when a call hangs up")
	cmd.Flags().StringVar(&f.HangupMethod, "hangup-method", "", "HTTP verb for --hangup-url")
	cmd.Flags().StringVar(&f.FallbackAnswerURL, "fallback-answer-url", "", "backup URL if --answer-url fails")
	cmd.Flags().StringVar(&f.MessageURL, "message-url", "", "URL for incoming SMS")
	cmd.Flags().StringVar(&f.MessageMethod, "message-method", "", "HTTP verb for --message-url")
	cmd.Flags().BoolVar(&f.Enabled, "enabled", false, "enable the application")
	cmd.Flags().BoolVar(&f.DefaultNumberApp, "default-number-app", false, "use as default for any new number")
	cmd.Flags().BoolVar(&f.DefaultEndpointApp, "default-endpoint-app", false, "use as default for any new endpoint")
	cmd.Flags().BoolVar(&f.LogIncomingMessages, "log-incoming-messages", false, "log inbound messages")
	cmd.Flags().BoolVar(&f.PublicURI, "public-uri", false, "expose a public SIP URI")
	cmd.Flags().StringVar(&f.ApplicationType, "application-type", "", "application classifier")
	cmd.Flags().StringVar(&f.Subaccount, "subaccount", "", "subaccount auth ID to scope this app to")
}

func newCreateCmd(format func() string) *cobra.Command {
	var f createFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new application",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := AppsFactory()
			if err != nil {
				return err
			}
			return runCreate(a, cmd.OutOrStdout(), format(), f)
		},
	}
	addCreateFlags(cmd, &f)
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("answer-url")
	return cmd
}

func runCreate(api client.ApplicationsAPI, w io.Writer, format string, f createFlags) error {
	app, err := api.Create(context.Background(), f.toParams())
	if err != nil {
		return err
	}
	fm, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "APP ID", Field: "AppID"},
		{Header: "NAME", Field: "AppName"},
		{Header: "ANSWER URL", Field: "AnswerURL"},
	}
	fmt.Fprintln(w, "Created.")
	return output.Render(w, *app, cols, fm)
}
```

Create `cmd/applications/update.go`:
```go
package applications

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
	"github.com/yash-kavaiya/vobiz-cli/internal/output"
)

func newUpdateCmd(format func() string) *cobra.Command {
	var f createFlags
	cmd := &cobra.Command{
		Use:   "update <app-id>",
		Short: "Update an application (only specified flags are sent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := AppsFactory()
			if err != nil {
				return err
			}
			return runUpdate(a, cmd.OutOrStdout(), format(), args[0], f)
		},
	}
	addCreateFlags(cmd, &f)
	return cmd
}

func runUpdate(api client.ApplicationsAPI, w io.Writer, format, id string, f createFlags) error {
	app, err := api.Update(context.Background(), id, f.toParams())
	if err != nil {
		return err
	}
	fm, err := output.ParseFormat(format)
	if err != nil {
		return err
	}
	cols := []output.Column{
		{Header: "APP ID", Field: "AppID"},
		{Header: "NAME", Field: "AppName"},
		{Header: "ANSWER URL", Field: "AnswerURL"},
		{Header: "ENABLED", Field: "Enabled"},
	}
	return output.Render(w, *app, cols, fm)
}
```

Create `cmd/applications/delete.go`:
```go
package applications

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <app-id>",
		Short: "Delete an application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := AppsFactory()
			if err != nil {
				return err
			}
			return runDelete(a, cmd.OutOrStdout(), args[0])
		},
	}
}

func runDelete(api client.ApplicationsAPI, w io.Writer, id string) error {
	if err := api.Delete(context.Background(), id); err != nil {
		return err
	}
	fmt.Fprintf(w, "Deleted application %s.\n", id)
	return nil
}
```

Create `cmd/applications/attach.go`:
```go
package applications

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

func newAttachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attach <app-id> <number>",
		Short: "Bind a phone number to an application",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := AppsFactory()
			if err != nil {
				return err
			}
			return runAttach(a, cmd.OutOrStdout(), args[0], args[1])
		},
	}
}

func runAttach(api client.ApplicationsAPI, w io.Writer, appID, number string) error {
	if err := api.AttachNumber(context.Background(), appID, number); err != nil {
		return err
	}
	fmt.Fprintf(w, "Attached %s to application %s.\n", number, appID)
	return nil
}
```

Create `cmd/applications/detach.go`:
```go
package applications

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yash-kavaiya/vobiz-cli/internal/client"
)

func newDetachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detach <number>",
		Short: "Clear the application binding on a number",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := AppsFactory()
			if err != nil {
				return err
			}
			return runDetach(a, cmd.OutOrStdout(), args[0])
		},
	}
}

func runDetach(api client.ApplicationsAPI, w io.Writer, number string) error {
	if err := api.DetachNumber(context.Background(), number); err != nil {
		return err
	}
	fmt.Fprintf(w, "Detached application from %s.\n", number)
	return nil
}
```

- [ ] **Step 4: Wire into registrations**

Add `applications` import to `cmd/registrations.go` and a `registerApplications` function. Add the corresponding `registerApplications(root)` call in `cmd/root.go` after `registerTrunks(root)`.

- [ ] **Step 5: Run tests**

```
go test ./cmd/applications/... ./cmd/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```
git add cmd/applications/ cmd/registrations.go cmd/root.go
git commit -m "feat(applications): CRUD + attach/detach number subcommands"
```

---

## Task 5: Extend smoke test for trunks + applications

**Files:**
- Modify: `cmd/smoke_test.go`

- [ ] **Step 1: Add the new smoke tests**

Append to `cmd/smoke_test.go`:
```go
func TestSmoke_TrunksListEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"objects":[{"trunk_id":"t1","name":"Outbound-A","trunk_direction":"outbound","cps_limit":10}],"meta":{"next":""}}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	if err := config.Save(filepath.Join(dir, ".vobiz", "config.yaml"), &config.File{
		ActiveProfile: "default",
		Profiles:      map[string]config.Profile{"default": {AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL}},
	}); err != nil {
		t.Fatal(err)
	}

	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"trunks", "list", "--output", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), `"trunk_id": "t1"`) {
		t.Fatalf("smoke output unexpected:\n%s", buf.String())
	}
}

func TestSmoke_ApplicationsListEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"objects":[{"app_id":"a1","app_name":"main","answer_url":"https://x/ans","enabled":true}],"meta":{"next":""}}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	if err := config.Save(filepath.Join(dir, ".vobiz", "config.yaml"), &config.File{
		ActiveProfile: "default",
		Profiles:      map[string]config.Profile{"default": {AuthID: "AB12", AuthToken: "tok", BaseURL: srv.URL}},
	}); err != nil {
		t.Fatal(err)
	}

	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"applications", "list", "--output", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), `"app_id": "a1"`) {
		t.Fatalf("smoke output unexpected:\n%s", buf.String())
	}
}
```

- [ ] **Step 2: Run the smoke tests**

```
go test ./cmd/ -run Smoke -v
```
Expected: 5 smoke tests PASS (Account/Numbers/Calls/Trunks/Applications).

- [ ] **Step 3: Commit**

```
git add cmd/smoke_test.go
git commit -m "test(smoke): cover trunks list and applications list end-to-end"
```

---

## Task 6: README update + manual verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Build and exercise the new commands**

```
go build -o vobiz .
./vobiz --help            # confirm `trunks` and `applications` appear
./vobiz trunks --help     # confirm list/get/create/update/delete
./vobiz applications --help  # confirm CRUD + attach/detach
```

Delete the binary afterwards (`rm vobiz` / `rm vobiz.exe`).

- [ ] **Step 2: Extend the README "First steps" section**

Add to `README.md` after the calls block:
````markdown
vobiz trunks list
vobiz trunks create --name Outbound-A --direction outbound --cps-limit 10
vobiz trunks get <trunk-id>
vobiz trunks update <trunk-id> --cps-limit 20
vobiz trunks delete <trunk-id>

vobiz applications list
vobiz applications create --name main --answer-url https://example.com/answer.xml --enabled
vobiz applications get <app-id>
vobiz applications attach <app-id> +14155551212
vobiz applications detach +14155551212
vobiz applications delete <app-id>
````

Update the Roadmap section:
```
- ~~`trunks`, `applications` (Plan 3)~~ shipped
```

- [ ] **Step 3: Optional — exercise against a real account**

```
./vobiz trunks list
./vobiz applications list
```

If any 404 or 422, the inferred REST path is wrong. Fix the constant in `internal/client/trunks.go` or `applications.go`, update the corresponding tests, and re-run before merging.

- [ ] **Step 4: Commit**

```
git add README.md
git commit -m "docs: README usage for trunks and applications"
```

---

## Follow-on plans

| # | Plan filename | Scope |
|---|---|---|
| 3.5 | `YYYY-MM-DD-vobiz-cli-trunk-subresources.md` | Trunk credentials, IP ACLs, origination URIs, and webhook management — REST paths verified against a live account first |
| 4 | `YYYY-MM-DD-vobiz-cli-whatsapp-partner.md` | `cmd/whatsapp` (send {text,media,template}, templates, campaigns, contacts), `cmd/partner` (customers, balance transfer, numbers, cdrs, analytics) |
| 5 | `YYYY-MM-DD-vobiz-cli-release.md` | `Dockerfile` (distroless multi-stage), `.goreleaser.yaml` (linux/darwin/windows × amd64/arm64 + Homebrew tap + GHCR Docker), `install.sh`, release workflow on tag `v*` |

Each plan continues the established patterns: new resource → `internal/client/<name>.go` typed API + tests; new verb → one file per verb under `cmd/<name>/`; new column spec at the call site; mutations get free Idempotency-Key headers and retry replay safety from `internal/httpx`.
