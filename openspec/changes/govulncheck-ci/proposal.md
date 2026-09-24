## Why

git-cleaner has no third-party dependency, but its release binaries embed the Go standard library of the
toolchain that built them, and Go regularly publishes security fixes for it (`net/http`, `os/exec`,
`crypto/*`...). Nothing tells the maintainer today that the published binaries call vulnerable code, nor stops
a release built with an affected toolchain.

## What Changes

- A reusable workflow runs `govulncheck ./...` with the latest stable Go, the toolchain used for releases.
- CI calls it on every push and pull request, and its result becomes part of the `tests passed` check, so a
  known vulnerability in called code blocks the way to `dev` and `main`.
- The same scan runs every week on `main` (and on demand), to catch vulnerabilities published after the last
  release.
- The release workflow runs it before GoReleaser: binaries are not published while a known vulnerability
  affects them.

## Capabilities

### New Capabilities
- `vulnerability-scanning`: scanning the code and its Go toolchain for known vulnerabilities in CI, on a
  schedule and before releases.

### Modified Capabilities
<!-- None -->

## Impact

- New file `.github/workflows/vulncheck.yml` (reusable, scheduled, manually triggerable).
- `.github/workflows/ci.yml`: new `vulncheck` job, added to the needs of `tests-passed`.
- `.github/workflows/release.yml`: GoReleaser job depends on the scan.
- `README.md`: Development section.
- No change to the Go code, no dependency added to `go.mod` (govulncheck is run with `go run`).
