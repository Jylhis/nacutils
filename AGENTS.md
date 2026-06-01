# Agents

This file tells AI coding agents how to work in this repository.

## Canon

Engineering norms for this repo live in two files:

- [`ENGINEERING_PRINCIPLES.md`](./ENGINEERING_PRINCIPLES.md) — foundational beliefs
- [`WAY_OF_WORKING.md`](./WAY_OF_WORKING.md) — day-to-day process (branches, CI, releases)

Read both before making changes.

## Quick rules

- Run `just lint && just test` before opening any PR.
- PR titles follow Conventional Commits (`feat:`, `fix:`, `ci:`, `docs:`, `chore:`, `refactor:`, `perf:`, `test:`, `build:`, `revert:`).
- Never commit secrets, credentials, or compiled binaries (`bin/`, `dist/`, the root `nacmail` file).
- Keep commits small and logical. One concept per commit.
- Commit co-author line for Paperclip runs: `Co-Authored-By: Paperclip <noreply@paperclip.ing>`

## CI

GitHub Actions (`.github/workflows/`) runs on every push and PR:

| Workflow | Trigger | What it does |
|----------|---------|--------------|
| `ci.yml` | push/PR to `main` | lint, test (with race detector + coverage), gitleaks scan |
| `pr-title.yml` | PR open/edit | enforces Conventional Commits on PR title |
| `release.yml` | push `v*` tag | GoReleaser → GitHub Releases |

## Dev environment

```bash
just dev    # enters the Nix/devenv shell with Go, just, golangci-lint, goreleaser
just        # list all recipes
```

## Ownership

Assigned agent: FoundingEngineer (Paperclip agent `399c7834-c9c8-4cee-b00f-78d96714321b`).
Escalate blockers via Paperclip issue comments on the active task.
