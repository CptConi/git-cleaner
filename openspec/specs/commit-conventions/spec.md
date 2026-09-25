## Purpose

The commit message convention of the repository, one single line `<type> (<feature>) <changes>` without body nor trailers, and how it is enforced: an opt-in local `commit-msg` hook, and a CI check, required through `tests passed`, of every commit a revision brings to its target branch and of the title of pull requests into `main`.

## Requirements

### Requirement: Commit message format
A commit message SHALL be exactly one line matching `^(feat|fix|docs|chore|refactor|test|ci|perf|build|revert|release) \([A-Za-z0-9._-]+\) [^[:space:]].*$` once its final newline is removed, so that a message with a body, trailers (including `Co-Authored-By`), a carriage return or an empty description is rejected.

#### Scenario: Compliant message
- **WHEN** a commit message is `feat (exclude-repositories) add --exclude flag, wording for help`
- **THEN** the check accepts it

#### Scenario: Conventional Commits colon
- **WHEN** a commit message is `feat(exclude-repositories): add --exclude flag`
- **THEN** the check rejects it and prints the expected format and the allowed types

#### Scenario: Multi-line message
- **WHEN** a commit message has a second line, such as a `Co-Authored-By:` trailer
- **THEN** the check rejects it

#### Scenario: Unknown type
- **WHEN** a commit message is `wip (exclude-repositories) try something`
- **THEN** the check rejects it and lists the allowed types

### Requirement: Local enforcement before committing
The repository SHALL provide a `commit-msg` hook that cleans the message as git records it (removing the scissors section of `git commit -v` and comment lines with `git stripspace --strip-comments`), accepts messages starting with `fixup!`, `squash!` or `amend!`, runs the check on the result and aborts the commit when it fails.

#### Scenario: Hook enabled and non-compliant message
- **WHEN** a contributor enabled the hooks with `git config core.hooksPath .githooks` and commits with a non-compliant message
- **THEN** the commit is not created and the expected format is printed

#### Scenario: Verbose commit
- **WHEN** a contributor runs `git commit -v` and writes a compliant subject above the scissors line
- **THEN** the hook accepts the commit

#### Scenario: Fixup commit
- **WHEN** a contributor runs `git commit --fixup HEAD`
- **THEN** the hook accepts it, and CI rejects it if it is pushed without being squashed

### Requirement: CI checks every commit a revision brings to its target
CI SHALL check the raw message of every commit, merge commits included, that the tested revision would bring to its target branch: for a push to a branch other than `dev` and `main`, the commits not on `dev`; for a push to `dev`, the commits not on `main`; for a pull request, the commits between its base and its head. Pushes to `main` SHALL NOT be checked, since they only come from merged pull requests.

#### Scenario: Compliant commit on top of a rejected one
- **WHEN** a feature branch holding a non-compliant commit receives a compliant commit on top of it
- **THEN** the job still fails, so the branch cannot be fast-forwarded into `dev`

#### Scenario: Force push after fixing a message
- **WHEN** the author rewrites the non-compliant commit and force-pushes the feature branch
- **THEN** the job checks the rewritten commits and passes

#### Scenario: Merge commit on a feature branch
- **WHEN** a feature branch contains a merge commit with git's default `Merge branch 'dev'` message
- **THEN** the job fails

### Requirement: Pull request titles into main follow the convention
CI SHALL check the title of every pull request into `main` against the commit message format, re-running when the title is edited, because the repository setting makes it the message of the merge commit.

#### Scenario: Pull request into main with a free-form title
- **WHEN** the maintainer opens a pull request from `dev` into `main` titled `Release`
- **THEN** the job fails until the title is changed to a compliant one such as `release (v1.1.0) merge dev into main`

### Requirement: Exemptions are limited to Dependabot runs and an allowlist
CI SHALL skip the commit checks only for runs triggered by Dependabot on its own work (a pull request opened by `dependabot[bot]`, or a push by `dependabot[bot]` to a `dependabot/` branch) and for commits whose SHA is listed in the versioned file `.github/commit-check-allowlist`. The author name of a commit SHALL NOT grant any exemption.

#### Scenario: Dependabot pull request
- **WHEN** Dependabot opens a pull request into `dev` whose commit has a multi-line body
- **THEN** the commit checks are skipped for that pull request

#### Scenario: Squash commit of a Dependabot pull request
- **WHEN** the maintainer squash-merges a Dependabot pull request into `dev` with a multi-line message
- **THEN** the job of the push to `dev` fails, although the commit is authored by `dependabot[bot]`

#### Scenario: Forged author
- **WHEN** a feature branch holds a non-compliant commit whose author name is `dependabot[bot]`
- **THEN** the job fails

#### Scenario: Recovery from a bad commit on dev
- **WHEN** a non-compliant commit reached `dev` and its SHA is added to `.github/commit-check-allowlist`
- **THEN** the job ignores that commit and pull requests from `dev` into `main` can pass again

### Requirement: Required through tests passed
The `tests passed` job SHALL succeed only when every job it aggregates succeeded, so that the commit checks are required by the `dev` and `main` rulesets without changing them.

#### Scenario: Failing commit check
- **WHEN** the `commit-messages` job fails while the test matrix passes
- **THEN** `tests passed` fails
