## Context

The module has no dependency; the attack surface of the binaries is the Go standard library of the toolchain
that builds them. Releases are built by GoReleaser with `go-version: stable`. CI runs a test matrix on Go 1.22
and stable, aggregated into the `tests passed` check that the rulesets of `dev` and `main` require.

## Goals / Non-Goals

**Goals:**
- Block changes and releases while called code has a known vulnerability.
- Notice vulnerabilities published between releases.
- Define the scan once and reuse it.

**Non-Goals:**
- Scanning with Go 1.22: releases are never built with it, and it would report standard library issues fixed
  in newer toolchains.
- Scanning the GitHub Actions themselves (covered by the `dependabot-actions` change).
- Automatic patch releases when the weekly scan fails.

## Decisions

- **Reusable workflow** `.github/workflows/vulncheck.yml` with `workflow_call`, `schedule` (weekly, Monday
  06:00 UTC) and `workflow_dispatch` triggers. `ci.yml` and `release.yml` call it (`uses:
  ./.github/workflows/vulncheck.yml`). Alternative considered: copying a job into each workflow, which drifts.
- **`go run golang.org/x/vuln/cmd/govulncheck@latest ./...`** on `setup-go` stable: no new action to track and
  always the current scanner. Alternatives considered: `golang/govulncheck-action` (one more action to keep
  updated) or a pinned version (reproducible, but to bump by hand since `go run` pins are not tracked by
  Dependabot). The vulnerability database is online, so results change over time anyway.
- **Only called code fails the job**: govulncheck's default symbol-level mode exits with status 3 when a
  vulnerable function is reachable, and only reports imported but uncalled vulnerabilities.
- **Required through `tests passed`**: `tests-passed` gets `vulncheck` in its needs, so no ruleset change is
  needed. In `release.yml`, the GoReleaser job gets `needs: vulncheck`.
- **Scheduled runs use the default branch** (`main`), which is exactly the released code.

## Risks / Trade-offs

- [A vulnerability without an available fix blocks every merge] → temporary escape hatch: remove `vulncheck`
  from the needs of `tests-passed` in a dedicated change, documented in the README.
- [The weekly run fails while nobody is working on the project] → GitHub emails the maintainer about the
  failure; the fix is usually a patch release built with the latest stable Go.
- [The scan needs network access to vuln.go.dev] → available on GitHub-hosted runners; a transient outage
  fails the job, and re-running it fixes the problem.

## Migration Plan

Additive. The schedule only starts once the workflow is on `main` (next merge of `dev`). Rollback: remove the
calls and the file.

## Open Questions

- None.
