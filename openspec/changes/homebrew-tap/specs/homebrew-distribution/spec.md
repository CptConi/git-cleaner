## ADDED Requirements

### Requirement: Install and upgrade with Homebrew on macOS
Each stable release SHALL be installable on macOS with `brew install cptconi/tap/git-cleaner`, on Apple Silicon and on Intel machines for as long as Homebrew runs on them, and the installed `git-cleaner --version` SHALL print the released version.

#### Scenario: Fresh install on macOS
- **WHEN** a user runs `brew install cptconi/tap/git-cleaner` after the release of `v1.2.0`
- **THEN** `git-cleaner --version` prints `git-cleaner 1.2.0`

#### Scenario: Upgrade
- **WHEN** release `v1.3.0` is published and the user runs `brew update && brew upgrade git-cleaner`
- **THEN** version `1.3.0` replaces `1.2.0`

### Requirement: Install with Homebrew on Linux
The cask SHALL also provide the Linux archives, so that `brew install cptconi/tap/git-cleaner` works where Homebrew supports casks on Linux, as verified in the official Homebrew container image.

#### Scenario: Install in the Homebrew container
- **WHEN** `brew install cptconi/tap/git-cleaner` runs in `ghcr.io/homebrew/brew` after a release
- **THEN** `git-cleaner --version` prints the released version

### Requirement: Installed binary runs without Gatekeeper prompt
On macOS, the cask SHALL remove the `com.apple.quarantine` attribute from the installed binary through declared post-install steps, without triggering any Homebrew deprecation warning, since the binaries are neither signed nor notarized.

#### Scenario: First launch after install
- **WHEN** a user runs `git-cleaner --help` right after installing it with Homebrew on macOS
- **THEN** macOS does not block the binary and the help is printed

#### Scenario: No deprecation warning
- **WHEN** a user installs or upgrades the cask with a current Homebrew
- **THEN** no deprecation warning about the tap is printed

### Requirement: Tap kept in sync for stable releases
The release workflow SHALL update the cask in `CptConi/homebrew-tap` for every stable release, and not for prerelease tags, with a single-line commit message following the `<type> (<feature>) <changes>` convention, authored with the maintainer's noreply address.

#### Scenario: New release published
- **WHEN** the release workflow publishes `v1.2.0`
- **THEN** the tap repository receives a commit `chore (homebrew) update git-cleaner to v1.2.0` pointing the cask to the `v1.2.0` archives and checksums

#### Scenario: Prerelease tag
- **WHEN** the tag `v1.3.0-rc1` is released
- **THEN** the cask in the tap is left unchanged

### Requirement: Release credentials are restricted to release tags
The credential used to push to the tap SHALL be a deploy key with write access to the tap only, stored in a GitHub environment usable only by runs of `v*` tags, and a tag ruleset SHALL allow only the maintainer to create, update or delete `v*` tags.

#### Scenario: Tag pushed by someone else
- **WHEN** a collaborator who is not the maintainer pushes a `v*` tag
- **THEN** GitHub rejects the push and no release runs

#### Scenario: Workflow run from a branch
- **WHEN** a workflow runs on a branch instead of a `v*` tag
- **THEN** it cannot read the deploy key

### Requirement: Recoverable tap update failure
When the tap cannot be updated after the release was published, the release workflow SHALL fail and SHALL keep the generated cask file as a workflow artifact.

#### Scenario: Deploy key removed from the tap
- **WHEN** the release workflow publishes `v1.2.0` but the push to the tap is refused
- **THEN** the GitHub release exists, the workflow fails, and its artifacts contain `git-cleaner.rb` ready to be committed to the tap by hand
