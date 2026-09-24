## Why

The workflows pin GitHub Actions by major version (`actions/checkout@v7`, `actions/setup-go@v7`, `goreleaser/goreleaser-action@v7`). New majors (runtime upgrades, deprecations) currently go unnoticed until a workflow breaks. The binary has no Go dependency, so actions are the only external dependencies to track.

## What Changes

- Add `.github/dependabot.yml` for the `github-actions` ecosystem: monthly version updates, all of them grouped in a single pull request, targeting `dev` (never `main`, which only receives merges from `dev`).
- Dependabot pull request titles follow the commit convention (`chore (deps) bump ...`).
- Dependabot security updates are disabled in the repository settings: they would always target `main`, with default titles. Security alerts keep notifying the maintainer, and the floating `@vN` tags already pick up patch releases.
- A new workflow runs GoReleaser in snapshot mode on pull requests that touch the release configuration, so that an action update breaking the release pipeline is caught before it reaches `dev`.
- Document the merge procedure: squash merge, whose single-line message is the pull request title followed by its number.

## Capabilities

### New Capabilities
- `dependency-updates`: automated version update proposals for the GitHub Actions used by the workflows, how they flow into `dev`, and how the release pipeline is exercised before they land.

### Modified Capabilities
<!-- None -->

## Impact

- New files `.github/dependabot.yml` and `.github/workflows/release-check.yml`.
- Repository settings, done by the maintainer in the GitHub UI: squash merging allowed with "Pull request title" as default message, merge commits still allowed (the `main` ruleset requires them), Dependabot security updates disabled.
- `README.md` (Contributing) and `openspec/config.yaml` (git flow: squash-merged Dependabot pull requests are the second way into `dev`).
- Coordinated with `commit-message-check`: only the commits of Dependabot's own pull request branches are exempted from the commit message check; the squash commit landing on `dev` is checked like any other.
