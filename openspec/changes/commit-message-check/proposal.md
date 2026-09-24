## Why

The repository follows a strict commit convention: one single line `<type> (<feature>) <changes>` and never a `Co-Authored-By` trailer. It is only enforced by discipline today, so a multi-line message or an automatic trailer can slip into `dev` and, from there, into `main`.

## What Changes

- A POSIX shell checker (`scripts/check-commit-msg.sh`) defines the rule once. In CI it validates raw commit messages. In the hook it first cleans the message the way git does (comments, scissors line of `git commit -v`) and tolerates `fixup!`, `squash!` and `amend!` commits, which must be squashed before pushing.
- A versioned `commit-msg` hook (`.githooks/commit-msg`) rejects a non-compliant message before the commit is created, once enabled with `git config core.hooksPath .githooks`.
- A CI job `commit-messages` checks every commit that the tested revision would bring to its target branch, merge commits included:
  - feature branches: commits that are not on `dev`;
  - pushes to `dev`: commits that are not on `main`;
  - pull requests: their commits, plus the title of pull requests into `main`, which becomes the merge commit message.
- Only runs triggered by Dependabot on its own pull requests are exempted. A versioned allowlist of commit SHAs provides a recovery path, since `dev` forbids force pushes.
- The `tests passed` check requires every job it aggregates to succeed, so the new job blocks the way to `dev` and `main` through the existing rulesets.
- Repository setting: merge commits default to the pull request title, so merging `dev` into `main` produces a compliant single-line message.

## Capabilities

### New Capabilities
- `commit-conventions`: the commit message format of the repository and how it is enforced locally and in CI.

### Modified Capabilities
<!-- None -->

## Impact

- New files:
  - `scripts/check-commit-msg.sh` and its test script `scripts/check-commit-msg_test.sh`;
  - `.githooks/commit-msg`;
  - `.gitattributes` (LF endings for the scripts and the hook);
  - `.github/commit-check-allowlist`.
- `.github/workflows/ci.yml`:
  - `commit-messages` job and script tests on the OS matrix;
  - generic success check in `tests-passed`;
  - `edited` pull request type.
- Repository settings (maintainer): default message of merge commits = pull request title.
- `README.md` Contributing section: format, allowed types, hook, reverts, merges into `main`. The sentence describing `tests passed` as "the whole CI matrix" also changes.
- `openspec/config.yaml` conventions: allowed types.
- No change to the Go code.
- Coordinated with `dependabot-actions` (exemption by event, squash commits checked) and `govulncheck-ci` (same generic `tests-passed` check).
