## ADDED Requirements

### Requirement: Monthly update proposals for GitHub Actions
The repository SHALL configure Dependabot to check the GitHub Actions used by `.github/workflows` every month
and to group all available updates into a single pull request.

#### Scenario: New major version of an action
- **WHEN** a new major version of `actions/checkout` is released
- **THEN** at the next monthly check Dependabot opens one pull request bumping every outdated action

### Requirement: Update pull requests target dev
Dependabot pull requests SHALL target the `dev` branch, so that they go through the same CI checks as any
feature and only reach `main` with the next merge of `dev`.

#### Scenario: Update pull request base branch
- **WHEN** Dependabot opens an update pull request
- **THEN** its base branch is `dev`

### Requirement: Update commits follow the commit convention
Dependabot pull request titles SHALL follow the `<type> (<feature>) <changes>` convention with the
`chore (deps)` prefix, and they SHALL be merged with a squash merge whose single-line message is the pull
request title.

#### Scenario: Title of an update pull request
- **WHEN** Dependabot proposes to bump `actions/setup-go` from 7 to 8
- **THEN** the pull request title starts with `chore (deps) ` without a colon after the prefix

#### Scenario: Merging an update
- **WHEN** the maintainer merges a Dependabot pull request into `dev`
- **THEN** `dev` receives one commit whose message is the pull request title on a single line
