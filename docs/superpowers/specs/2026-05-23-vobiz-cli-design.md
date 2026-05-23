# Vobiz CLI — Design

**Date:** 2026-05-23
**Status:** Approved (design phase)
**Author:** Yash Kavaiya
**Target binary:** `vobiz`
**Language:** Go

---

## 1. Purpose

A single Go binary that gives developers a terminal interface to the Vobiz programmable-telephony platform — covering account management, phone numbers, calls, trunks, applications, WhatsApp, the partner/reseller API, and in-terminal documentation search. Mental model: `gh` / `gcloud` for Vobiz.

It is **both** an API management client *and* a docs reader, since the public Vobiz MCP server (`https://docs.vobiz.ai/mcp`) makes the docs side cheap to embed.

## 2. Background

Vobiz exposes a REST API at `https://api.vobiz.ai/api/v1`, authenticated per-request with `X-Auth-ID` + `X-Auth-Token` headers. An official Go SDK already exists at `github.com/vobiz-ai/vobiz-go-sdk` covering calls, conferences, and the VobizXML response builder. A separate Partner API lives under `/partner/api/*` for white-label resellers. A public Streamable-HTTP MCP server exposes `search` and `fetch` over the docs.

There is no first-party CLI today.

## 3. Goals and non-goals

### Goals (v1)

- Manage core programmable-telephony resources: `account`, `numbers`, `calls`, `trunks`, `applications`.
- Send WhatsApp messages, manage templates / campaigns / contacts.
- Drive the Partner API: sub-account create, balance transfer, CDRs, analytics.
- `vobiz docs search|open` backed by the Vobiz docs MCP server.
- Profile-based credential management (`vobiz auth login`), env-var override.
- Output as table (default), JSON, or YAML via `-o`.
- Cross-platform binaries (linux/darwin/windows × amd64/arm64) via GoReleaser, plus Homebrew tap and a distroless Docker image.
- Shell completion for bash, zsh, fish, powershell.

### Non-goals (v1)

- Plugin architecture (`kubectl`-style external sub-binaries).
- TUI dashboard / live-call watcher.
- Webhook-tunnel feature (`stripe listen` equivalent).
- Conference management and VobizXML response builder — already covered by the Go SDK for users who need them programmatically.
- OpenAPI client codegen — Vobiz does not publish a confirmed OpenAPI spec.

## 4. Architectural decisions

| # | Decision | Rationale |
|---|---|---|
| D1 | Wrap the official Go SDK; fall back to a shared raw-HTTP client (`internal/httpx`) for endpoints the SDK does not cover. | Faster to build, picks up SDK fixes, but does not block on SDK gaps for trunks/applications/whatsapp/partner. |
| D2 | Single Cobra binary; one Go package per resource under `cmd/`. | Standard, testable, matches user mental model of `gh` / `gcloud`. |
| D3 | Credentials in `~/.vobiz/config.yaml` (mode 0600) with named profiles; env vars and flags override. | Best balance of UX and CI-friendliness. Keychain can be added later behind a flag without breaking the file format. |
| D4 | Default output is a human table; `-o json|yaml|table` flag. | Matches `aws`/`gcloud` ergonomics; scripts get clean JSON on demand. |
| D5 | `internal/httpx` is the single seam for retries, backoff, `Retry-After`, and `Idempotency-Key`. | One place to fix transport behavior; every resource benefits. |
| D6 | Distribute via GoReleaser → GitHub Releases + Homebrew tap + GHCR Docker image. `go install` also works. | Industry standard for Go CLIs; minimal extra effort. |

## 5. Command surface

```
vobiz
├── auth        login | logout | status | profile { list | use | rm }
├── account     get | balance | transactions list | concurrency
├── numbers     list | search --country | buy <n> | release <n>
├── calls       make | list | get <uuid> | recordings { list | download <uuid> }
├── trunks      list | get | create | update | delete
│               credentials { list | create | delete }
│               ip-acl      { list | create | delete }
│               origination { list | set }
├── applications list | get | create | update | delete | attach | detach
├── whatsapp    send {text|media|template}
│               templates  { list | create | delete }
│               campaigns  { list | create | start | pause }
│               contacts   { list | import <csv> }
├── partner     customers  { list | create | get | suspend }
│               balance    transfer --to --amount
│               numbers    { list | assign }
│               cdrs       list
│               analytics
├── docs        search <query> | open <path|topic>
├── completion  bash | zsh | fish | powershell
└── version
```

**Global flags:** `-o json|yaml|table` (default `table`), `--profile <name>`, `--auth-id`, `--auth-token`, `--no-color`, `-v/--verbose`, `--base-url`.

## 6. Repository layout

```
vobiz-cli/
├── go.mod
├── main.go
├── cmd/
│   ├── root.go
│   ├── auth/   account/   numbers/   calls/
│   ├── trunks/ applications/   whatsapp/   partner/   docs/
│   ├── completion.go
│   └── version.go
├── internal/
│   ├── config/      # ~/.vobiz/config.yaml read/write, profile resolution
│   ├── auth/        # credential precedence resolver
│   ├── client/      # *Client wrapping SDK + httpx; one interface per resource
│   ├── httpx/       # shared retrying HTTP client, auth headers, idempotency
│   ├── output/      # table | json | yaml renderer with column specs
│   ├── paginate/    # generic hasMore pager (--limit, --all)
│   ├── docsmcp/     # Streamable-HTTP MCP client for docs.vobiz.ai/mcp
│   ├── errors/      # typed errors → exit codes
│   └── version/     # ldflags-injected build metadata
├── docs/superpowers/specs/
├── .goreleaser.yaml
├── Dockerfile                            # distroless multi-stage
└── .github/workflows/{ci.yml, release.yml}
```

### Boundaries

- `cmd/*` packages never call HTTP or touch the filesystem directly — they parse flags, call `internal/client`, hand results to `internal/output`.
- `internal/client` exposes one interface per resource (`TrunksAPI`, `ApplicationsAPI`, …) so command tests use fakes.
- `internal/httpx` is the only place that knows about retries, headers, idempotency, request IDs.
- `internal/output` is type-agnostic: column specs are declared per resource at the `cmd/*` layer.

## 7. Data flow

```
user invokes `vobiz trunks list -o json`
        │
        ▼
cobra parses → cmd/trunks/list.go RunE(ctx, args, flags)
        │
        ▼
internal/auth.Resolve(flags, env, profile)
        │  → Credentials{AuthID, AuthToken, BaseURL}
        ▼
internal/client.New(creds)
        │  → *Client { sdk *vobiz.Client; http *httpx.Client }
        ▼
client.Trunks.List(ctx, ListOpts{…})
        │  httpx adds X-Auth-* headers, Idempotency-Key for mutations,
        │  retries 5xx/429 with exponential backoff, honors Retry-After
        ▼
internal/paginate.All(...) if --all, else single page
        │
        ▼
internal/output.Render(w, items, format)
        │
        ▼
exit 0  (or typed error → friendly message + non-zero exit code)
```

## 8. Credentials

### Precedence (highest → lowest)

1. `--auth-id` / `--auth-token` flags
2. `VOBIZ_AUTH_ID` / `VOBIZ_AUTH_TOKEN` env vars
3. Profile selected by `--profile <name>`
4. Active profile from config (default name: `default`)

### Config file (`~/.vobiz/config.yaml`, mode 0600)

```yaml
active_profile: default
profiles:
  default:
    auth_id: AB12CD34
    auth_token: <token>
    base_url: https://api.vobiz.ai/api/v1
  staging:
    auth_id: ...
    auth_token: ...
```

`vobiz auth login` prompts for ID + token (token via masked input), writes the file with 0600, verifies by calling `GET /Account/{id}/`.

## 9. HTTP client behavior (`internal/httpx`)

- Adds `X-Auth-ID`, `X-Auth-Token`, `Accept: application/json`, `Content-Type: application/json` (on POST/PUT/PATCH), `User-Agent: vobiz-cli/<version> (<os>/<arch>)`.
- Generates a per-invocation `Idempotency-Key` UUID for every mutation so retries are safe.
- Retry policy:
  - `429`: read `Retry-After`, sleep, retry up to 3 times.
  - `5xx`, `EOF`, `connection reset`: exponential backoff 1s → 2s → 4s, max 3 retries.
  - `4xx` (not 429): no retry.
- Captures and surfaces `X-Request-Id` on errors when present.

## 10. Error handling

| Exit | Category | Examples | User-visible behavior |
|---|---|---|---|
| 0 | Success | — | Output to stdout |
| 1 | User error | Missing flag, invalid number, no creds, 4xx (401/403/404/422) | One-line message + hint (e.g. `run 'vobiz auth login'`) |
| 2 | API error (after retries) | 429 exhausted, 5xx exhausted, network | Message + request-id if present |
| 3 | Internal | Recovered panic, bug | Generic message; stack with `-v` |

Typed errors live in `internal/errors` (`ErrAuth`, `ErrNotFound`, `ErrValidation`, `ErrRateLimited`, `ErrServer`). `cmd/root.go` sets `SilenceErrors: true` and has a single error-handling sink that prints and selects the exit code. No `os.Exit` calls scattered through subcommands.

## 11. Output (`internal/output`)

- `Format` enum: `Table | JSON | YAML`.
- `Render(w io.Writer, rows any, format Format, columns []Column)`.
- Tables via `go-pretty/v6/table` (compact, color-aware, respects `--no-color` and `NO_COLOR`).
- JSON via `encoding/json` with `MarshalIndent` (two-space).
- YAML via `gopkg.in/yaml.v3`.
- Columns declared at the call site per resource — no per-renderer knowledge of Vobiz types.

## 12. Docs subcommand (`internal/docsmcp`)

- Client for the Vobiz docs MCP server at `https://docs.vobiz.ai/mcp`, transport: Streamable HTTP, no auth.
- Tools wrapped: `search(query)`, `fetch(path)`.
- `vobiz docs search <query>` → renders ranked results (title, path, snippet).
- `vobiz docs open <path|topic>` → fetches the markdown and pretty-prints in the terminal (using `glamour` for ANSI rendering).

## 13. Testing strategy

### Unit (fast, no network)

- `internal/output` — golden-file tests per renderer.
- `internal/auth` — table tests for credential precedence.
- `internal/config` — read/write roundtrip in `t.TempDir()`, verify file mode 0600.
- `internal/httpx` — `httptest.Server` exercising retry, backoff, `Retry-After`, idempotency-key replay, request-id capture.
- `internal/paginate` — fake API returning paged responses with `hasMore`.

### Command tests (`cmd/*`)

- Each subcommand injects a fake `Client` (interface stub), runs `cmd.Execute()` with args, asserts stdout/stderr/exit code. No real HTTP.

### Integration (opt-in, `-tags=integration`)

- Hits a real test account via env-supplied creds; skipped by default; runs nightly via GitHub Actions when `VOBIZ_TEST_AUTH_ID` is set as a repo secret.

### MCP docs client

- `httptest.Server` mocking the Streamable-HTTP MCP endpoint with canned `search`/`fetch` responses.

### Quality bar

- ≥80% coverage on `internal/`, ≥60% on `cmd/`.
- `golangci-lint`: `govet`, `errcheck`, `staticcheck`, `gosec`, `revive`.

## 14. CI/CD & release

### `.github/workflows/ci.yml` (PR + push to main)

- Matrix: `{ubuntu, macos, windows} × {go 1.22, 1.23}`.
- Steps: `go vet`, `golangci-lint`, `go test ./...`, `go build ./...`.

### `.github/workflows/release.yml` (on tag `v*`)

- `goreleaser release --clean`.

### `.goreleaser.yaml`

- Builds: linux/darwin/windows × amd64/arm64.
- Archives: `tar.gz` (unix), `zip` (windows); include `LICENSE`, `README.md`, completion scripts.
- `brews`: pushes formula to `homebrew-tap` repo.
- `dockers`: distroless multi-arch image at `ghcr.io/<owner>/vobiz:{version,latest}`.
- `checksums.txt`; cosign keyless signing optional, easy to add later.
- Version via `-ldflags "-X .../version.Version={{.Version}} -X .../version.Commit={{.ShortCommit}} -X .../version.Date={{.Date}}"`.

### Install paths

```bash
brew install <owner>/tap/vobiz
go install github.com/<owner>/vobiz-cli/cmd/vobiz@latest
curl -fsSL https://raw.githubusercontent.com/<owner>/vobiz-cli/main/install.sh | sh
docker run --rm ghcr.io/<owner>/vobiz:latest version
```

## 15. Dependencies (locked at design time)

| Purpose | Module |
|---|---|
| CLI framework | `github.com/spf13/cobra` |
| Config files | `github.com/spf13/viper` (or std + `yaml.v3`; decide at impl time) |
| YAML | `gopkg.in/yaml.v3` |
| Tables | `github.com/jedib0t/go-pretty/v6` |
| Markdown rendering (docs) | `github.com/charmbracelet/glamour` |
| Masked password input | `golang.org/x/term` |
| UUID (idempotency keys) | `github.com/google/uuid` |
| Official Vobiz SDK | `github.com/vobiz-ai/vobiz-go-sdk` |
| MCP client | `github.com/modelcontextprotocol/go-sdk` (or hand-rolled Streamable-HTTP client if pulling the dep is heavy — decide at impl time) |

## 16. Open questions deferred to implementation

- Exact Partner API field names — confirm from `https://docs.vobiz.ai/llms.txt` at implementation time; design above assumes the paths documented on the partner overview page.
- WhatsApp REST paths — same: `llms.txt` is the source of truth, public docs page does not enumerate endpoints.
- Whether to depend on `viper` (DX) or stay std-only (smaller binary) — fine either way; defer.
- Whether to ship cosign-signed artifacts in v1 or wait for v1.1 — easy to add later.

## 17. Success criteria

- `vobiz auth login` + `vobiz account get` returns balance for a real test account.
- `vobiz calls make --to <number> --answer-url <url>` places a call and prints the resulting `CallUUID`.
- `vobiz trunks list`, `vobiz applications list`, `vobiz numbers list` all render tables on a populated account and `-o json` produces parseable JSON.
- `vobiz whatsapp send text --to <num> --body "hi"` sends a message.
- `vobiz partner customers list` works on a partner-enabled account.
- `vobiz docs search "sip trunk"` returns ranked results from the Vobiz docs MCP.
- `go test ./...` is green; `golangci-lint run` is clean; `goreleaser release --snapshot --clean` produces all artifacts locally.
