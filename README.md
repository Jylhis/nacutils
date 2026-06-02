# nacutils

Jylhis command-line utilities.

## Install

Download a pre-built binary from [GitHub Releases](https://github.com/Jylhis/nacutils/releases):

```bash
# macOS arm64
curl -L https://github.com/Jylhis/nacutils/releases/latest/download/nacutils_VERSION_darwin_arm64.tar.gz | tar xz
sudo mv nacmail /usr/local/bin/
sudo mv nacclean /usr/local/bin/
sudo mv nacls /usr/local/bin/
sudo mv nacwrite /usr/local/bin/

# Linux amd64
curl -L https://github.com/Jylhis/nacutils/releases/latest/download/nacutils_VERSION_linux_amd64.tar.gz | tar xz
sudo mv nacmail /usr/local/bin/
sudo mv nacclean /usr/local/bin/
sudo mv nacls /usr/local/bin/
sudo mv nacwrite /usr/local/bin/
```

Or install from source (requires Go 1.22+):

```bash
go install github.com/jylhis/nacutils/cmd/nacmail@latest
go install github.com/jylhis/nacutils/cmd/nacclean@latest
go install github.com/jylhis/nacutils/cmd/nacls@latest
go install github.com/jylhis/nacutils/cmd/nacwrite@latest
```

## Development

### Prerequisites

- [Nix](https://nixos.org/download/) with flakes enabled
- [devenv](https://devenv.sh/getting-started/)

### Getting started

```bash
git clone https://github.com/Jylhis/nacutils
cd nacutils
just dev        # enter the dev environment (installs Go, just, golangci-lint, goreleaser)
just            # list all available commands
```

### Commands

| Command              | Description                               |
|----------------------|-------------------------------------------|
| `just dev`           | Enter Nix dev shell via devenv            |
| `just build`         | Build nacmail, nacclean, nacls, and nacwrite into `./bin/` |
| `just test`          | Run all tests                             |
| `just test-v`        | Run tests with verbose output             |
| `just lint`          | Run golangci-lint                         |
| `just fmt`           | Format Go source                          |
| `just tidy`          | Tidy go.mod                               |
| `just release-snapshot` | Build release artifacts locally        |
| `just clean`         | Remove `bin/` and `dist/`                 |

## CI

GitHub Actions runs lint + tests on every commit and PR (`.github/workflows/ci.yml`).

## Releasing

Tag a commit to publish binaries to GitHub Releases:

```bash
git tag v0.1.0
git push origin v0.1.0
# GitHub Actions (release.yml) runs GoReleaser and publishes
# linux/darwin × amd64/arm64 binaries to the Releases page.
```

## Architecture

```
cmd/
  nacmail/          async agent-to-agent mailbox CLI
  nacclean/         safe mailbox cleanup CLI
  nacls/            mailbox summary CLI
  nacwrite/         non-interactive nacmail envelope writer
internal/
  envelope/         message envelope schema (UUIDv7 IDs, JSON-lines)
  mailbox/          local filesystem mailbox storage
deploy/
  cloudflare/       Cloudflare Worker stub — wire when a hosted endpoint is needed
```

## nacmail

Send, list, read, and delete messages between agents and users:

```
nacmail send <recipient> <body> [--kind note|status|attn|heartbeat-summary] [--subject "..."] [--sender "..."]
nacmail list [<recipient>] [--json] [--color|--no-color]
nacmail read <id> [--json] [--color|--no-color]
nacmail rm <id>
```

Interactive list/read output auto-enables ANSI styling on TTYs. Use `--color` to force styling, `--no-color` to disable it, or set `NO_COLOR=1` to suppress ANSI globally. The default styled path consumes a vendored `@jylhis/design` `v0.4.0` token snapshot; set `NACUTILS_DISABLE_DESIGN_RENDERER=1` to fall back to the legacy hardcoded ANSI renderer for rollback.

Messages are stored as JSON-lines under `$XDG_DATA_HOME/nacutils/mail/<recipient>/inbox`.

Reading mail records `meta.read_at`, which `nacclean` uses to avoid deleting unread envelopes.

## nacclean

Explicitly clean read mail older than a chosen threshold:

```bash
nacclean [recipient] --before <duration|RFC3339|unix-ms> [--apply] [--json]
```

`nacclean` defaults to dry-run mode. It reports how many envelopes matched the scope, were inspected, were eligible for deletion, and were removed. Only envelopes with `meta.read_at` set and `created_at` older than `--before` are eligible. Add `--apply` to rewrite inboxes and remove those envelopes.

## nacls

Summarize mailbox state without opening individual envelopes:

```bash
nacls [recipient] [--json] [--path /tmp/mail-fixture]
```

Default output is a plain table with `RECIPIENT`, `TOTAL`, `PENDING`, `READ`, and `MALFORMED` counts. `--json` emits recipient-sorted summaries with deterministic keys. `--path` points `nacls` at a fixture or alternate mail root without touching the default mailbox directory.

## nacwrite

Compose nacmail-compatible envelopes without an interactive editor or TTY:

```bash
nacwrite send <recipient> --kind <note|status|attn|heartbeat-summary> --subject "..." --body "..." [--meta '{"ticket":"JYL-61"}']
printf 'Body from stdin' | nacwrite send FoundingEngineer --kind note --subject "handoff"
nacwrite send CEO --kind status --body "ready to merge" --dry-run --json
```

`nacwrite send` writes to the same mailbox path and schema as `nacmail send`. When `--body` is omitted it reads the body from stdin, which keeps it usable in non-TTY environments. `--json` prints the full envelope for scripting, and `--dry-run` validates the command without writing to disk.

## Dogfood

nacmail is the internal messaging layer used by Jylhis's own AI agents.

**Who depends on nacmail:**
- `FoundingEngineer` agent — uses `nacmail send` to pass heartbeat summaries and status notes between agents.
- `FoundingEngineer` and `CEO` — use `nacclean` for explicit maintenance passes when mailbox backlog grows.
- `CodexEngineer` and `FoundingEngineer` — use `nacls` to inspect mailbox state during triage without opening every envelope.
- `FoundingEngineer` and `CEO` — use `nacwrite send` for scripted status, attention, and handoff notes without a persistent TTY.
- Any future Jylhis agent that needs async, persistent inter-agent messaging.

**Cadence:** every agent heartbeat (continuous; agents are triggered by Paperclip wake events, not a fixed clock).

**How to verify it's working:** `nacmail list` from any agent user shows the inbox; `nacmail send <agent-username> "ping"` delivers immediately.
