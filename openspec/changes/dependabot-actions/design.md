## Context

The workflows (`ci.yml`, `branch-policy.yml`, `release.yml`) use three actions pinned by major version. The Go module has no dependency. `main` only accepts pull requests from `dev` ("source is dev" check) merged with a merge commit (the only method its ruleset allows). `dev` requires the `tests passed` check and has no pull_request rule, since features reach it by fast-forward pushes. Commit messages must be single-line `<type> (<feature>) <changes>`. `release.yml` only runs on `v*` tags, so nothing exercises it before a release.

## Goals / Non-Goals

**Goals:**
- Be told about action updates without watching release pages.
- Keep the git flow: updates land in `dev` first, with CI.
- Keep `dev` history compliant with the commit convention.
- Catch an update that breaks the release pipeline before it is merged.

**Non-Goals:**
- Go module updates: there is no dependency (`gomod` can be added if one appears).
- Updating the GoReleaser version itself: the action installs `~> v2`, i.e. the latest v2 release.
- Auto-merging update pull requests.
- Composite actions under `.github/actions/`: `directory: "/"` only covers `.github/workflows` and a root `action.yml`; `directories` will be needed if the repository adds some.

## Decisions

- **`target-branch: dev`**, for version updates. Dependabot raises security update pull requests against the default branch only, ignoring `target-branch` and the commit message customization, so they would target `main` and fail "source is dev". **Security updates are disabled** in the repository settings instead: Dependabot alerts still notify the maintainer, and the floating `@vN` tags already receive patch releases.
- **Monthly schedule, one group for everything** (`groups: { actions: { patterns: ["*"] } }`): actions change rarely, and one pull request a month is easy to review. If one action of the group breaks, the maintainer can comment `@dependabot ignore <action> major version` to exclude it from the group. Alternative considered: weekly and ungrouped, which is noisier for three actions.
- **`commit-message.prefix: "chore (deps) "` with a trailing space**: Dependabot inserts a colon after a prefix ending with a letter, a digit, `)` or `]`, unless the value ends with whitespace. The prefix also applies to grouped pull request titles, which become `chore (deps) bump <action> from X to Y in the actions group` (one update) or `chore (deps) bump the actions group with N updates` (several).
- **Squash merge with "Pull request title" as the default message** (repository setting): the default message would copy Dependabot's multi-line commit body. With the setting, the squash commit message is `<title> (#N)` with an empty body, which matches the commit convention. "Allow merge commits" stays enabled, because the `main` ruleset only allows merge commits and conflicting repository settings would block merges into `main`.
- **No technical guard for the merge method into `dev`**: merge methods can only be restricted through a pull_request rule, which would also force pull requests for the fast-forward feature pushes. Instead, the commit message check of `commit-message-check` runs on the push to `dev` that the merge creates: a merge commit bringing Dependabot's multi-line commits fails it. Only the commits of Dependabot's own pull request branches are exempted, not every commit authored by `dependabot[bot]` (the squash commit is authored by the pull request author, i.e. `dependabot[bot]`).
- **Release snapshot check** (`.github/workflows/release-check.yml`): on pull requests whose paths include `.github/workflows/release.yml` or `.goreleaser.yaml`, run the same GoReleaser action with `release --snapshot --clean`. It is not a required check: a path-filtered workflow reports nothing on other pull requests, which would leave a required check pending forever.
- **Fixes to a broken update go through a feature branch**, not onto the Dependabot branch: Dependabot stops rebasing a pull request once other commits are pushed to it.

## Risks / Trade-offs

- [The maintainer merges a Dependabot pull request with a merge commit] → the commit message check of the push to `dev` fails and flags it; the procedure is in the README.
- [A new major version changes inputs and breaks a workflow] → CI and, for the release pipeline, the snapshot check run on the update pull request before anything reaches `dev`.
- [A security fix of an action is only picked up at the next monthly run or patch tag] → Dependabot alerts notify the maintainer, who can bump it manually through a feature branch.
- [GoReleaser needs a secret in snapshot mode once the `homebrew-tap` change lands] → the snapshot job sets a placeholder value for `HOMEBREW_TAP_DEPLOY_KEY`, since nothing is published.

## Migration Plan

Add the files on a feature branch, merge into `dev`. Dependabot reads its configuration from the default branch only (`main`), so updates start minutes after the next merge of `dev` into `main`. Rollback: delete the files and re-enable security updates if wanted.

## Open Questions

- None: restricting pull request merges into `dev` to squash merges was considered and rejected (see Decisions).
