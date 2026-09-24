## Why

On macOS, installing git-cleaner today means either having Go (`go install`) or downloading an archive and
clearing the Gatekeeper quarantine by hand. Homebrew is the usual way to install and update CLI tools there:
`brew install` once, `brew upgrade` afterwards.

## What Changes

- GoReleaser publishes a Homebrew cask for each release into a tap repository `CptConi/homebrew-tap`.
- The cask removes the macOS quarantine attribute after installation, since the binaries are not notarized.
- The README documents `brew install cptconi/tap/git-cleaner` and `brew upgrade`.
- Prerequisites on the maintainer side: create the tap repository and a token allowed to push to it, stored as
  the `HOMEBREW_TAP_TOKEN` secret of git-cleaner.

## Capabilities

### New Capabilities
- `homebrew-distribution`: installing and updating git-cleaner with Homebrew from a dedicated tap, kept in sync
  by the release workflow.

### Modified Capabilities
<!-- None -->

## Impact

- `.goreleaser.yaml`: new `homebrew_casks` section.
- `.github/workflows/release.yml`: passes `HOMEBREW_TAP_TOKEN` to GoReleaser.
- New public repository `CptConi/homebrew-tap` (created by the maintainer) and a repository secret.
- `README.md`: Install section.
- No change to the Go code.
