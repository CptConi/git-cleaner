## Context

The convention is `<type> (<feature>) <comma-separated changes>` on a single line, without `Co-Authored-By` (see `openspec/config.yaml`). Rulesets require the `tests passed` check on `dev` and on `main`. On `main`, a pull request from `dev` merged with a merge commit is required. On `dev`, only the tested tip of a push counts, and force pushes and deletions are forbidden. Commits reach `dev` by fast-forward from feature branches whose CI already ran. Dependabot pull requests (change `dependabot-actions`) are squash-merged into `dev`; GitHub makes the pull request author, `dependabot[bot]`, the author of the squash commit.

## Goals / Non-Goals

**Goals:**
- One definition of the rule, used by a local hook and by CI.
- Blocking in CI through the existing required `tests passed` check, with no hole: the check of a tip covers every commit it would bring.
- Clear error messages showing the expected format.
- A way out when a bad commit reaches `dev` anyway.

**Non-Goals:**
- Rewriting existing history (it already complies).
- Enforcing a maximum length.
- Protecting the checker against a pull request that weakens it: a pull request runs its own version of the script, and reviewing changes to `scripts/` is the maintainer's job.

## Decisions

- **POSIX `sh` checker** (`scripts/check-commit-msg.sh`), run as `sh "$(git rev-parse --show-toplevel)/scripts/check-commit-msg.sh"` (no executable bit needed). It uses `LC_ALL=C` and `grep -E`, and works with BSD tools on macOS, GNU tools on Linux and Git Bash on Windows. Alternative considered: a Go program, which would need `go run` in the hook (slow, Go required).
- **Pattern** `^(feat|fix|docs|chore|refactor|test|ci|perf|build|revert|release) \([A-Za-z0-9._-]+\) [^[:space:]].*$`:
  - `[^[:space:]]` rejects a tab or a carriage return as the first character of the description.
  - The raw message, final newline removed, must hold exactly one line. Lines are counted in a way that also works without a trailing newline (not with `wc -l` alone).
  - `release` covers the merge of `dev` into `main`.
- **Two modes, one rule.**
  - CI checks the raw message (`git log -1 --format=%B <sha>`).
  - The hook first reproduces git's cleanup: `sed '/^# -\{24\} >8 -\{24\}$/,$d' | git stripspace --strip-comments` removes the `git commit -v` diff and the comment lines, and honors `core.commentChar`. It also accepts `fixup!`, `squash!` and `amend!` prefixes, which CI rejects if they are pushed unsquashed.
  - A message kept with comments by `git commit -m ... -m "# x"` passes the hook but fails in CI: the hook is a convenience, CI is authoritative.
- **Commit range derived from the tested revision, not from the event** (full-history checkout, `fetch-depth: 0`):
  - push to a branch other than `dev` and `main`: `origin/dev..HEAD`;
  - push to `dev`: `origin/main..HEAD`, which gives the same verdict as the `dev` → `main` pull request run on that SHA;
  - pull request: `base.sha..head.sha`;
  - push to `main`: skipped (only merged pull requests reach it).

  Alternative rejected: `before..after`. A compliant commit pushed on top of a rejected one would turn the tip green, and after a force push `before` is not fetched, so the range fails.
- **Merge commits are checked** (no `--no-merges`): a merge commit on a feature branch would otherwise reach `dev` unchecked by fast-forward. Feature branches are rebased on `dev` rather than merged, as the README explains.
- **Pull request title of pull requests into `main`** is checked as well, with `edited` added to the `pull_request` types of `ci.yml`: the repository setting "Allow merge commits → Pull request title" makes it the merge commit message.
- **Exemptions by event identity**:
  - skip the commit checks when `github.event.pull_request.user.login == 'dependabot[bot]'`;
  - skip them for pushes by `github.actor == 'dependabot[bot]'` to `dependabot/**`.

  Commit author names are never trusted: they can be forged, and the squash commit of a Dependabot pull request is authored by `dependabot[bot]`, yet it must be checked on the push to `dev`. `git rev-list --author` is not used anyway, since its argument is a regex (`[bot]` is a character class). The skip happens inside the job step, never through a job-level `if:`, because a skipped job reports success.
- **Recovery allowlist** `.github/commit-check-allowlist`: one full SHA per line, with a comment explaining why. `dev` forbids force pushes, so a bad commit that reached it (for instance created by GitHub) would otherwise make every `dev` → `main` pull request fail forever.
- **Generic `tests-passed`**: it checks `RESULTS: ${{ join(needs.*.result, ' ') }}` and fails unless every value is `success`. Adding a job then only means adding it to `needs`. The `govulncheck-ci` change relies on the same check; whichever change lands first introduces it.
- **Failure output**: one `::error::` annotation per failing commit, with newlines escaped as `%0A`, plus the expected format and the allowed types.
- **Hook activation stays opt-in** (`core.hooksPath` cannot be set by a clone). The README warns that it replaces `.git/hooks` and any global `core.hooksPath` for this repository.

## Risks / Trade-offs

- [`git revert` does not run the `commit-msg` hook and writes `Revert "..."` with a body] → CI rejects it; the README shows `git revert --no-commit <sha>` followed by a compliant `git commit -m`.
- [The GitHub merge dialog lets the maintainer edit the merge commit message] → the default comes from the checked pull request title; a hand-edited message that breaks the rule is only caught on the next push to `dev`, and fixed through the allowlist.
- [Contributors forget to enable the hook] → CI still blocks the way to `dev` and `main`; the hook only saves a round trip.
- [The allowed type list may miss a type the maintainer wants] → single place to edit in the checker, covered by the script tests.

## Migration Plan

Additive. Existing commits comply; the ranges only look at commits not yet on `dev` or `main`. Rollback: remove the job from the needs of `tests-passed`.

## Open Questions

- Is the list of allowed types right, in particular `release` for merges into `main`?
