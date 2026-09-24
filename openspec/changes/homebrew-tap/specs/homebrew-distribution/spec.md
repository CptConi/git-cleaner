## ADDED Requirements

### Requirement: Install with Homebrew
Each published release SHALL be installable on macOS with `brew install cptconi/tap/git-cleaner`, on Apple
Silicon and Intel machines, and the installed `git-cleaner --version` SHALL print the released version.

#### Scenario: Fresh install on macOS
- **WHEN** a user runs `brew install cptconi/tap/git-cleaner` after the release of `v1.2.0`
- **THEN** `git-cleaner --version` prints `git-cleaner 1.2.0`

#### Scenario: Upgrade
- **WHEN** release `v1.3.0` is published and the user runs `brew upgrade git-cleaner`
- **THEN** version `1.3.0` replaces `1.2.0`

### Requirement: Installed binary runs without Gatekeeper prompt
The cask SHALL remove the `com.apple.quarantine` attribute from the installed binary after installation, since
the binaries are neither signed nor notarized.

#### Scenario: First launch after install
- **WHEN** a user runs `git-cleaner --help` right after installing it with Homebrew
- **THEN** macOS does not block the binary and the help is printed

### Requirement: Tap kept in sync by the release workflow
The release workflow SHALL update the cask in `CptConi/homebrew-tap` for every published release, with a
single-line commit message following the `<type> (<feature>) <changes>` convention.

#### Scenario: New release published
- **WHEN** the release workflow publishes `v1.2.0`
- **THEN** the tap repository receives a commit `chore (homebrew) update git-cleaner to v1.2.0` pointing the
  cask to the `v1.2.0` archives and checksums

#### Scenario: Missing token
- **WHEN** the `HOMEBREW_TAP_TOKEN` secret is missing or cannot push to the tap
- **THEN** the release workflow fails, and the maintainer sees that the tap was not updated
