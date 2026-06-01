# nacutils

Jylhis command-line utilities.

## Install

Download a pre-built binary from [GitHub Releases](https://github.com/Jylhis/nacutils/releases):

```bash
# macOS arm64
curl -L https://github.com/Jylhis/nacutils/releases/latest/download/nacutils_VERSION_darwin_arm64.tar.gz | tar xz
sudo mv nacmail /usr/local/bin/

# Linux amd64
curl -L https://github.com/Jylhis/nacutils/releases/latest/download/nacutils_VERSION_linux_amd64.tar.gz | tar xz
sudo mv nacmail /usr/local/bin/
```

Or install from source (requires Go 1.22+):

```bash
go install github.com/jylhis/nacutils/cmd/nacmail@latest
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
| `just build`         | Build nacmail into `./bin/`               |
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

Interactive list/read output auto-enables ANSI styling on TTYs. Use `--color` to force styling, `--no-color` to disable it, or set `NO_COLOR=1` to suppress ANSI globally.

Messages are stored as JSON-lines under `$XDG_DATA_HOME/nacutils/mail/<recipient>/inbox`.

## Dogfood

nacmail is the internal messaging layer used by Jylhis's own AI agents.

**Who depends on nacmail:**
- `FoundingEngineer` agent — uses `nacmail send` to pass heartbeat summaries and status notes between agents.
- Any future Jylhis agent that needs async, persistent inter-agent messaging.

**Cadence:** every agent heartbeat (continuous; agents are triggered by Paperclip wake events, not a fixed clock).

**How to verify it's working:** `nacmail list` from any agent user shows the inbox; `nacmail send <agent-username> "ping"` delivers immediately.
