## Why

The workflows pin GitHub Actions by major version (`actions/checkout@v7`, `actions/setup-go@v7`,
`goreleaser/goreleaser-action@v7`). New majors (runtime upgrades, deprecations) currently go unnoticed until a
workflow breaks. The binary has no Go dependency, so actions are the only external dependencies to track.

## What Changes

- Add `.github/dependabot.yml` for the `github-actions` ecosystem: monthly checks, all updates grouped in a
  single pull request, targeting `dev` (never `main`, which only receives merges from `dev`).
- Dependabot pull request titles follow the commit convention (`chore (deps) bump ...`).
- Document how to merge them: squash merge with the pull request title as the single-line commit message.

## Capabilities

### New Capabilities
- `dependency-updates`: automated update proposals for the GitHub Actions used by the workflows, and how they
  flow into `dev`.

### Modified Capabilities
<!-- None -->

## Impact

- New file `.github/dependabot.yml`.
- Repository settings, done by the maintainer in the GitHub UI: allow squash merging with "Pull request title"
  as the default commit message.
- `README.md`: Contributing note on Dependabot pull requests.
- Relies on the `commit-message-check` change exempting `dependabot[bot]` commits.
