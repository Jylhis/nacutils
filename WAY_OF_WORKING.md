# Way of Working

How the Jylhis engineering team operates day-to-day on this repository.

## Branches and PRs

- `main` is always deployable. Direct pushes are blocked.
- All changes land via pull request. One approval required before merge.
- PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `ci:`, `docs:`, `chore:`, etc.
- Keep PRs focused. One logical change per PR; large changes should be split into stacked PRs.

## CI gates (all must be green to merge)

| Check | Tool |
|-------|------|
| Lint | `golangci-lint` via `just lint` |
| Tests | `go test -race ./...` via `just test` |
| PR title | Conventional Commits via `action-semantic-pull-request` |
| Secret scan | `gitleaks detect` |
| Dependabot | weekly Go module + Actions updates |

Run `just lint && just test` locally before opening a PR.

## Commit messages

Commit messages follow Conventional Commits. The PR title is what ends up in the changelog, so it matters most.

```
feat: add nacmail rm command
fix: handle missing mailbox dir on first send
docs: add dogfood section to README
ci: pin gitleaks to v8.24.3
chore: tidy go.mod
```

## Releases

Tag a semver tag to publish binaries:

```bash
git tag v0.1.0 && git push origin v0.1.0
```

GoReleaser (`.github/workflows/release.yml`) publishes linux/darwin × amd64/arm64 tarballs to GitHub Releases automatically.

## Dependencies

Keep them minimal. Add a new dependency only when writing the equivalent from scratch is a genuine maintenance burden. Prefer the standard library.

## Code review

- Reviewers look for correctness, security, and clarity — not style (that's the linter's job).
- Author responds to every comment, even if just to explain why it was not addressed.
- Approve when you'd be comfortable being on-call for the change.
