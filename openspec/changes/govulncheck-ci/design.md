## Context

The module has no dependency, so the attack surface of the binaries is the Go standard library of the toolchain that builds them. Releases are built by GoReleaser with `go-version: stable`, for darwin, linux and windows with `CGO_ENABLED=0`. CI runs a test matrix on Go 1.22 and stable, aggregated into the `tests passed` check that the rulesets of `dev` and `main` require. Go publishes standard library vulnerabilities together with the release that fixes them, so a scan with the current stable toolchain already contains the fix on the day the vulnerability becomes known.

## Goals / Non-Goals

**Goals:**
- Block changes and releases while the code reaches a known vulnerability.
- Detect, between releases, that the published binaries were built with an affected toolchain.
- Cover every release platform.
- Keep an escape hatch that does not switch detection off.

**Non-Goals:**
- Scanning with Go 1.22: releases are never built with it.
- Scanning the GitHub Actions themselves (covered by the `dependabot-actions` change).
- Automatic patch releases when the weekly scan fails.
- Binary mode on the release archives: on stripped binaries it only has module-level precision and reports every vulnerability of the toolchain (46 findings against 1 in source mode in the review).

## Decisions

- **One script, `scripts/vulncheck.sh`**:
  - It installs govulncheck once with `go install golang.org/x/vuln/cmd/govulncheck@latest` into a temporary `GOBIN`.
  - It then runs it for `GOOS` in `linux darwin windows`, with `CGO_ENABLED=0`, `-format json` and `-show verbose`.
  - It must not use `GOOS=… go run …`, which would cross-compile govulncheck itself.
  - The JSON output is filtered with `jq`: the findings whose trace reaches a function are the reached vulnerabilities, others are informational. Their IDs, minus the allowlist, decide the result.
  - JSON mode exits 0 even with findings, so a non-zero exit always means a tool or network error. These are retried up to 3 times with a pause, and findings never are.
  - The script prints the findings in a readable form before failing.
- **Allowlist** `.github/vulncheck-allowlist`, one OSV ID per line (`GO-2026-1234  # reason, date`). govulncheck itself cannot silence findings, and removing the job from `tests-passed` would switch detection off and still leave releases blocked.
- **`vulncheck.yml`**, reusable (`workflow_call`) and manually triggerable, scans the checked-out source with `setup-go` stable. `ci.yml` calls it as job `vulncheck`; `release.yml` calls it and the GoReleaser job gets `needs: vulncheck`, so the scan uses the same stable toolchain as the build a moment later.
- **`vulncheck-release.yml`**, weekly (`cron: "17 6 * * 1"`, off the start of the hour when GitHub may drop queued scheduled jobs) and `workflow_dispatch`:
  - it finds the latest release with `gh release view --json tagName` and exits successfully when there is none;
  - it downloads the `linux_amd64` archive and reads the Go version of the binary with `go version` (build information survives `-s -w`);
  - it checks out the tag;
  - it installs govulncheck with stable, then sets up that Go version and runs the script with `GOTOOLCHAIN=local`, so that the analysis uses the standard library the published binaries embed.
- **Generic `tests-passed`** (shared with `commit-message-check`): it fails unless every value of `join(needs.*.result, ' ')` is `success`; whichever change lands first introduces it, the other only adds its job to `needs`.
- **Permissions**: `contents: read` in the scan workflows; the weekly one also reads releases with the default token.

## Risks / Trade-offs

- [GitHub disables scheduled workflows of public repositories after 60 days without activity] → the README says how to re-enable it from the Actions tab; a manual run is part of the release checklist.
- [Scheduled failure notifications go to the user who created or last changed the schedule, if Actions notifications are enabled] → the README asks the maintainer to keep them enabled.
- [Right after a Go security release, `stable` in `setup-go` may still resolve to the previous version for a few hours] → the scan fails meanwhile; re-running later fixes it, and the allowlist covers longer delays.
- [`@latest` depends on proxy.golang.org as well as vuln.go.dev] → both are retried as tool errors.
- [govulncheck@latest requires a recent Go to build (v1.8.0 needs Go 1.26)] → it is always built with stable, even when analyzing with an older release toolchain.

## Migration Plan

Additive. The weekly and manual workflows only run once the files are on `main` (next merge of `dev`); until the first release, the weekly run succeeds without scanning. Rollback: remove the calls and the files.

## Open Questions

- None.
