## ADDED Requirements

### Requirement: Monthly version updates for GitHub Actions
The repository SHALL configure Dependabot version updates for the GitHub Actions used by `.github/workflows`, checked every month, with every available update grouped into a single pull request.

#### Scenario: New major version of an action
- **WHEN** a new major version of `actions/checkout` is released
- **THEN** at the next monthly check Dependabot opens one pull request bumping every outdated action

### Requirement: Version update pull requests target dev
Dependabot version update pull requests SHALL target the `dev` branch, so that they go through the same CI checks as any feature and only reach `main` with the next merge of `dev`, and Dependabot security updates SHALL be disabled because they always target the default branch.

#### Scenario: Update pull request base branch
- **WHEN** Dependabot opens a version update pull request
- **THEN** its base branch is `dev`

#### Scenario: Security advisory for an action
- **WHEN** a security advisory is published for an action used by the workflows
- **THEN** no pull request is opened against `main` and the maintainer is notified by a Dependabot alert

### Requirement: Update pull request titles follow the commit convention
Dependabot pull request titles SHALL start with the `chore (deps) ` prefix, without a colon, so that they follow the `<type> (<feature>) <changes>` convention.

#### Scenario: Single update in the group
- **WHEN** Dependabot proposes to bump only `actions/setup-go` from 7 to 8
- **THEN** the pull request title is `chore (deps) bump actions/setup-go from 7 to 8 in the actions group`

#### Scenario: Several updates in the group
- **WHEN** Dependabot proposes to bump three actions
- **THEN** the pull request title is `chore (deps) bump the actions group with 3 updates`

### Requirement: Update pull requests are squash-merged into dev
The maintainer SHALL merge Dependabot pull requests into `dev` with a squash merge whose message is the pull request title followed by its number, on a single line.

#### Scenario: Merging an update
- **WHEN** the maintainer merges a Dependabot pull request into `dev`
- **THEN** `dev` receives one commit whose message is the pull request title followed by ` (#N)`, on a single line, and the commit message check of the push to `dev` passes

### Requirement: Release pipeline exercised before an update lands
CI SHALL run GoReleaser in snapshot mode on every pull request that changes `.github/workflows/release.yml` or `.goreleaser.yaml`, so that an action update breaking the release pipeline fails before being merged.

#### Scenario: Breaking goreleaser-action major version
- **WHEN** a Dependabot pull request bumps `goreleaser/goreleaser-action` to a major version whose inputs changed
- **THEN** the release snapshot check fails on the pull request
