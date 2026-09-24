## 1. Reusable scan workflow

- [ ] 1.1 Add `.github/workflows/vulncheck.yml` with `workflow_call`, weekly `schedule` and `workflow_dispatch` triggers, `permissions: contents: read`
- [ ] 1.2 Job `govulncheck`: checkout, `setup-go` with `go-version: stable` (no cache), `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`

## 2. Wiring

- [ ] 2.1 `ci.yml`: add a `vulncheck` job calling the reusable workflow and add it to the needs of `tests-passed`; make `tests-passed` fail when it does not succeed
- [ ] 2.2 `release.yml`: add a `vulncheck` job calling the reusable workflow and make the GoReleaser job depend on it

## 3. Documentation

- [ ] 3.1 README Development section: what is scanned, when, and the escape hatch when a vulnerability has no fix yet

## 4. Verification

- [ ] 4.1 `actionlint` passes on every workflow
- [ ] 4.2 Locally, `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` passes with the current stable Go, and reports standard library vulnerabilities with an old toolchain (`GOTOOLCHAIN=go1.22.0`), proving the job would fail
- [ ] 4.3 `gofmt -l .`, `go vet ./...` (linux, darwin, windows) and `go test ./...` still pass, and a dry run on a sample tree behaves as before
- [ ] 4.4 On the feature branch, CI shows the `vulncheck` job green and `tests passed` waiting for it
