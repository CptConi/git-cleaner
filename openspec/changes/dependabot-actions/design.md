## Context

The workflows (`ci.yml`, `branch-policy.yml`, `release.yml`) use three actions pinned by major version. The Go
module has no dependency. `main` only accepts pull requests from `dev` ("source is dev" check), and `dev`
requires the `tests passed` check. Commit messages must be single-line `<type> (<feature>) <changes>`.

## Goals / Non-Goals

**Goals:**
- Be told about action updates without watching release pages.
- Keep the git flow: updates land in `dev` first, with CI.
- Keep `dev` history compliant with the commit convention.

**Non-Goals:**
- Go module updates: there is no dependency (`gomod` can be added if one appears).
- Updating the GoReleaser version itself: the action installs `~> v2`, i.e. the latest v2 release.
- Auto-merging update pull requests.

## Decisions

- **`target-branch: dev`**: a pull request into `main` would fail the "source is dev" check anyway.
- **Monthly schedule, one group for everything** (`groups: { actions: { patterns: ["*"] } }`): actions change
  rarely, and one pull request a month is easy to review. Alternative considered: weekly and ungrouped, which
  is noisier for three actions.
- **`commit-message.prefix: "chore (deps) "` with a trailing space**: Dependabot inserts a colon after a prefix
  ending with a letter, digit, `)` or `]` unless the value ends with whitespace (Dependabot options reference),
  so titles become `chore (deps) bump ...`.
- **Squash merge with "Pull request title" as default message** (repository setting): Dependabot commits carry
  multi-line bodies and trailers, which would break the convention if merged as they are. The squash commit
  message becomes `chore (deps) bump ... (#N)`, which matches the pattern. The `commit-message-check` change
  exempts `dependabot[bot]` commits inside the pull request itself.

## Risks / Trade-offs

- [The maintainer merges with a merge commit instead of a squash] → the multi-line Dependabot commits reach
  `dev`; mitigation: restrict the dev ruleset or the repository settings to squash merges for pull requests,
  and document the procedure.
- [A new major version changes inputs and breaks a workflow] → CI runs on the update pull request before
  anything reaches `dev`.

## Migration Plan

Add the file on a feature branch, merge into `dev`: Dependabot picks the configuration from the default
branch only (`main`), so updates start after the next merge of `dev` into `main`. Rollback: delete the file.

## Open Questions

- Should the `dev` ruleset be extended to allow only squash merges for pull requests, to make the procedure
  impossible to get wrong?
