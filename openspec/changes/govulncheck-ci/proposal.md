## Why

git-cleaner has no third-party dependency, but its release binaries embed the Go standard library of the toolchain that built them, and Go regularly publishes security fixes for it (`os/exec`, `path/filepath`, `net/*`...). Nothing tells the maintainer today that the code calls vulnerable functions, nor that already published binaries were built with an affected toolchain, nor stops a release in that state.

## What Changes

- A shared script (`scripts/vulncheck.sh`) runs govulncheck for each release platform (linux, darwin, windows, cgo disabled). It fails on vulnerable functions that the code reaches, except for the vulnerability IDs listed in `.github/vulncheck-allowlist`, and retries tool or network errors.
- CI runs it on every push and pull request with the latest stable Go, through a reusable workflow; the result becomes part of the `tests passed` check.
- The release workflow runs it before GoReleaser, with the toolchain that builds the release: binaries are not published while a known vulnerability affects them.
- A weekly workflow (also triggered manually) scans the latest published release: the source of the latest `v*` tag, analyzed with the Go version that built its binaries, so that a vulnerability fixed by a newer Go toolchain is reported while the published binaries remain affected.

## Capabilities

### New Capabilities
- `vulnerability-scanning`: scanning the code and the Go toolchain used for releases for known vulnerabilities, in CI, before releases and weekly on the published release.

### Modified Capabilities
<!-- None -->

## Impact

- New files:
  - `scripts/vulncheck.sh`;
  - `.github/vulncheck-allowlist`;
  - `.github/workflows/vulncheck.yml` (reusable scan of the current source);
  - `.github/workflows/vulncheck-release.yml` (weekly and manual scan of the latest release).
- `.github/workflows/ci.yml`: new `vulncheck` job, added to the needs of the generic `tests-passed` check (shared with `commit-message-check`).
- `.github/workflows/release.yml`: GoReleaser depends on the scan.
- `README.md`: Development section (what is scanned, the allowlist, re-enabling the weekly workflow).
- No change to the Go code, no dependency added to `go.mod`.
