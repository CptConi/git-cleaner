## Context

The convention is `<type> (<feature>) <comma-separated changes>` on a single line, without `Co-Authored-By`
(see `openspec/config.yaml`). Rulesets already require the `tests passed` check on `dev` (for pushed commits)
and on `main` (for pull requests from `dev`). Commits reach `dev` by fast-forward from feature branches whose CI
already ran.

## Goals / Non-Goals

**Goals:**
- One definition of the rule, used by a local hook and by CI.
- Blocking in CI through the existing required `tests passed` check.
- Clear error messages showing the expected format.

**Non-Goals:**
- Rewriting existing history (it already complies).
- Checking merge commit messages written in the GitHub UI when merging `dev` into `main`: they are created
  at merge time, after the checks ran (see Risks).
- Enforcing a maximum length.

## Decisions

- **POSIX `sh` + `grep -E` checker** (`scripts/check-commit-msg.sh <file>`): runs on the Ubuntu runners and in
  the hook on macOS, Linux and Git Bash for Windows, without adding a toolchain. Alternative considered: a Go
  program, which would need `go run` in the hook (slow, requires Go for every contributor).
- **Pattern**: `^(feat|fix|docs|chore|refactor|test|ci|perf|build|revert|release) \([A-Za-z0-9._-]+\) [^ ].*$`,
  and the stripped message must hold exactly one line. `release` covers the merge of `dev` into `main`.
- **Commit range in CI**: `git rev-list --no-merges <base>..<head>` with a full-history checkout. Pull requests
  use `base.sha..head.sha`. Pushes use `before..after`, and `origin/dev..after` for new branches (where
  `before` is all zeros). Each failing commit is reported with a GitHub `::error` annotation.
- **Dependabot exemption** by author (`dependabot[bot]`): its commits carry multi-line bodies and only reach
  `dev` through a squash merge whose single-line message the maintainer writes (see the `dependabot-actions`
  change).
- **Hook activation stays opt-in** (`core.hooksPath` cannot be set by a clone); documented in the README.

## Risks / Trade-offs

- [A GitHub merge commit (dev → main) keeps the default multi-line message] → documented in the README: edit
  the message to one line when merging, e.g. `release (v1.1.0) merge dev into main`.
- [Contributors forget to enable the hook] → CI still blocks the push from reaching `dev`/`main`; the hook only
  saves a round-trip.
- [The allowed type list may miss a type the maintainer wants] → single place to edit in the checker, covered
  by tests of the script.

## Migration Plan

Additive. Existing commits comply; the check only looks at new ranges. Rollback: remove the job from the needs
of `tests-passed`.

## Open Questions

- Is the list of allowed types right (in particular `release` for merges into `main`)?
