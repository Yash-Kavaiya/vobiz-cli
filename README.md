# vobiz-cli

The unofficial-but-friendly terminal interface for the [Vobiz](https://vobiz.ai) programmable-telephony platform.

```bash
vobiz auth login
vobiz account balance
vobiz docs search "sip trunk"
```

Status: under active development. See `docs/superpowers/specs/2026-05-23-vobiz-cli-design.md`.

## Install (from source, for now)

```bash
go install github.com/yash-kavaiya/vobiz-cli@latest
```

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

- ~~`calls`, `numbers` (Plan 2)~~ shipped
- `trunks`, `applications` (Plan 3)
- `whatsapp`, `partner` (Plan 4)
- GoReleaser, Homebrew tap, Docker image, install script (Plan 5)
