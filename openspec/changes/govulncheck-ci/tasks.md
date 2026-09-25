## 1. Scan script

- [x] 1.1 Write `scripts/vulncheck.sh`:
  - install govulncheck with `go install` into a temporary `GOBIN`, unless `GOVULNCHECK` names a prebuilt binary;
  - loop over `GOOS` linux, darwin and windows with `CGO_ENABLED=0` and `-format json -scan symbol` (govulncheck rejects `-show` with JSON output, which already holds every finding), on the module given as argument, the repository by default;
  - keep the findings whose trace reaches a function, minus the IDs of `.github/vulncheck-allowlist`;
  - print them and fail when some remain, with exit status 3 (other failures are errors);
  - retry non-zero exits up to 3 times.
- [x] 1.2 Add `.github/vulncheck-allowlist` (empty, with a header comment describing the format)

## 2. Workflows

- [x] 2.1 Add `.github/workflows/vulncheck.yml` (`workflow_call`, `workflow_dispatch`, `permissions: contents: read`): checkout, `setup-go` stable without cache, run the script
- [x] 2.2 `ci.yml`: add a `vulncheck` job calling it and add it to the needs of `tests-passed`, whose check fails unless every value of `join(needs.*.result, ' ')` is `success` (introduce this generic check if `commit-message-check` has not yet)
- [x] 2.3 `release.yml`: add a `vulncheck` job calling the reusable workflow and make the GoReleaser job depend on it
- [x] 2.4 Add `.github/workflows/vulncheck-release.yml`:
  - triggers: `cron: "17 6 * * 1"` and `workflow_dispatch`;
  - latest release through `gh release view`, and success without scanning when there is none;
  - Go version read with `go version` from the `linux_amd64` archive;
  - checkout of the default branch, for the script and the allowlist (`v0.1.0` predates them), and of the tag into `release/`;
  - govulncheck installed with stable, then run with the release's Go version and `GOTOOLCHAIN=local`;
  - on exit status 3, an annotation saying that a new release is needed.

## 3. Documentation

- [x] 3.1 README Development section:
  - what is scanned and when;
  - the allowlist;
  - re-enabling the weekly workflow after 60 days of inactivity;
  - keeping Actions notifications enabled;
  - `tests passed` including the scan.

## 4. Verification

- [x] 4.1 `go run github.com/rhysd/actionlint/cmd/actionlint@latest` passes on every workflow
- [x] 4.2 Locally:
  - `scripts/vulncheck.sh` passes with the current stable Go;
  - it fails, listing the reached standard library vulnerabilities, when run with Go 1.22.0 as the analysis toolchain (govulncheck built with stable, `PATH="$(GOTOOLCHAIN=go1.22.0 go env GOROOT)/bin:$PATH" GOTOOLCHAIN=local`);
  - adding those IDs to the allowlist makes it pass.
- [x] 4.3 `gofmt -l .`, `make vet` and `go test ./...` still pass (no Go code changes, so no dry run is needed)
- [x] 4.4 On the feature branch, CI shows the `vulncheck` job green and `tests passed` waiting for it
- [ ] 4.5 After the next merge of `dev` into `main`, trigger `vulncheck-release.yml` manually: it succeeds, scanning the latest release or reporting that none exists
