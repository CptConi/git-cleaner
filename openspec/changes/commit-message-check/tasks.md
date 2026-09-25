## 1. Checker

- [x] 1.1 Write `scripts/check-commit-msg.sh`: raw mode for CI (exactly one line, pattern with `[^[:space:]]`, `LC_ALL=C`, line count correct without a trailing newline) and a hook mode that removes the scissors section, runs `git stripspace --strip-comments` and accepts `fixup!`, `squash!` and `amend!`; print the expected format and the allowed types on failure
- [x] 1.2 Write `scripts/check-commit-msg_test.sh`: compliant messages, Dependabot titles with ` (#N)`, the Conventional Commits colon, multi-line messages with a `Co-Authored-By` trailer, unknown types, empty descriptions, a tab or `\r` after the parenthesis, no trailing newline, `git commit -v` scissors content and fixup prefixes in hook mode
- [x] 1.3 Add `.gitattributes` forcing LF endings on `scripts/*.sh` and `.githooks/*`

## 2. Local hook

- [x] 2.1 Add `.githooks/commit-msg`, calling the checker in hook mode through `sh "$(git rev-parse --show-toplevel)/scripts/check-commit-msg.sh"`, with its executable bit recorded (`git update-index --chmod=+x`)

## 3. CI

- [x] 3.1 Add a `commit-messages` job to `ci.yml` (checkout with `fetch-depth: 0`): compute the range from the tested revision (`origin/dev..HEAD`, `origin/main..HEAD` on `dev`, `base.sha..head.sha` for pull requests, nothing on `main`), include merge commits, skip Dependabot's own runs by event identity inside the step, ignore the SHAs of `.github/commit-check-allowlist`, and emit one `::error::` annotation per failing commit
- [x] 3.2 Check the title of pull requests into `main` in the same job, and add `edited` to the `pull_request` types
- [x] 3.3 Add `.github/commit-check-allowlist` (empty, with a header comment)
- [x] 3.4 Run the script tests on the ubuntu, macos and windows runners (Git Bash)
- [x] 3.5 Make `tests-passed` generic: add `commit-messages` to its needs and fail unless every value of `join(needs.*.result, ' ')` is `success`; update the comment and step name that mention only the test matrix

## 4. Repository settings (maintainer, GitHub UI)

- [ ] 4.1 Set "Allow merge commits" to use the pull request title as the default message

## 5. Documentation

- [x] 5.1 README Contributing:
  - the format and the allowed types;
  - `git config core.hooksPath .githooks` and what it replaces;
  - rebasing feature branches instead of merging `dev` into them;
  - reverts with `git revert --no-commit`;
  - the pull request title of merges into `main`;
  - the allowlist;
  - `tests passed` no longer being only the test matrix.
- [x] 5.2 `openspec/config.yaml` conventions: list the allowed types

## 6. Verification

- [x] 6.1 `go run github.com/rhysd/actionlint/cmd/actionlint@latest` passes on the workflows; the script tests pass locally and in CI on the three OS
- [x] 6.2 `gofmt -l .` is empty, `make vet` and `go test ./...` still pass
- [x] 6.3 On a throwaway branch (then deleted), check the job:
  - a non-compliant commit fails;
  - a compliant commit on top of it still fails;
  - a force push of the fixed commit passes;
  - a merge commit with git's default message fails.
- [ ] 6.4 After merging into `dev`: the push to `dev` passes, and a draft pull request `dev` → `main` with a free-form title fails until the title complies
