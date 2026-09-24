## 1. Checker

- [ ] 1.1 Write `scripts/check-commit-msg.sh <message-file>`: strip git comment lines and trailing blank lines, require exactly one line matching the pattern, print the expected format and the allowed types on failure
- [ ] 1.2 Add a test script (`scripts/check-commit-msg_test.sh`) covering compliant messages, the Conventional Commits colon, multi-line messages with a `Co-Authored-By` trailer, unknown types and empty descriptions

## 2. Local hook

- [ ] 2.1 Add `.githooks/commit-msg` calling the checker (executable bit set in git)
- [ ] 2.2 Check that it works from Git Bash on Windows (no bash-only syntax, LF line endings enforced with `.gitattributes`)

## 3. CI

- [ ] 3.1 Add a `commit-messages` job to `ci.yml` (checkout with `fetch-depth: 0`) that computes the range for pushes, new branches and pull requests, skips `dependabot[bot]` commits and runs the checker on each commit
- [ ] 3.2 Add the job to the `needs` of `tests-passed` and make `tests-passed` fail when it fails
- [ ] 3.3 Run the script tests in CI

## 4. Documentation

- [ ] 4.1 README Contributing: format, allowed types, `git config core.hooksPath .githooks`, and the one-line message to type when merging `dev` into `main`

## 5. Verification

- [ ] 5.1 `actionlint` passes on the workflows
- [ ] 5.2 The script tests pass locally; `gofmt -l .`, `go vet ./...` and `go test ./...` still pass
- [ ] 5.3 Push a feature branch with one compliant and one non-compliant commit (then drop it) to see the job fail and `tests passed` fail with it
