## ADDED Requirements

### Requirement: Commit message format
A commit message SHALL be exactly one line matching `<type> (<feature>) <changes>`, where `<type>` is one of
`feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `perf`, `build`, `revert` or `release`, `<feature>` is
made of letters, digits, `.`, `_` or `-`, and `<changes>` is a non-empty description. A message with a body or
trailers, including `Co-Authored-By`, SHALL be rejected.

#### Scenario: Compliant message
- **WHEN** a commit message is `feat (exclude-repositories) add --exclude flag, wording for help`
- **THEN** the check accepts it

#### Scenario: Conventional Commits colon
- **WHEN** a commit message is `feat(exclude-repositories): add --exclude flag`
- **THEN** the check rejects it and prints the expected format

#### Scenario: Multi-line message
- **WHEN** a commit message has a second line, such as a `Co-Authored-By:` trailer
- **THEN** the check rejects it

#### Scenario: Unknown type
- **WHEN** a commit message is `wip (exclude-repositories) try something`
- **THEN** the check rejects it and lists the allowed types

### Requirement: Local enforcement before committing
The repository SHALL provide a `commit-msg` hook that runs the check and aborts the commit when the message is
not compliant. Git comment lines (starting with `#`) and trailing blank lines added by git SHALL be ignored.

#### Scenario: Hook enabled
- **WHEN** a contributor enabled the hooks with `git config core.hooksPath .githooks` and commits with a
  non-compliant message
- **THEN** the commit is not created and the expected format is printed

### Requirement: CI enforcement of pushed commits
CI SHALL check every commit introduced by a push or by a pull request, except commits authored by
`dependabot[bot]`, and the result SHALL be part of the `tests passed` status.

#### Scenario: Non-compliant commit pushed on a feature branch
- **WHEN** a feature branch receives a commit with a multi-line message
- **THEN** the `commit-messages` job fails and `tests passed` fails with it, so the commit cannot reach `dev`

#### Scenario: New branch
- **WHEN** a branch is pushed for the first time
- **THEN** only the commits that are not already on `dev` are checked
