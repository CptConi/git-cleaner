## Why

Installing git-cleaner today means either having Go (`go install`) or downloading an archive and, on macOS, clearing the Gatekeeper quarantine by hand. Homebrew is the usual way to install and update CLI tools on macOS, and it also runs on Linux: `brew install` once, `brew upgrade` afterwards.

## What Changes

- GoReleaser publishes a Homebrew cask for each stable release (not for prerelease tags) into a tap repository `CptConi/homebrew-tap`, with a single-line commit following the commit convention.
- On macOS, the cask removes the quarantine attribute from the installed binary through declared post-install steps (Homebrew 7 `*_steps`), since the binaries are not notarized. It does not use the Ruby `postflight` blocks that Homebrew 7 deprecates.
- The cask also installs on Linux, where Homebrew supports casks; this is smoke-tested in the official Homebrew container image.
- Release credentials are narrowed:
  - GoReleaser pushes to the tap with a deploy key that only has write access to the tap;
  - the key is stored in a GitHub environment that only `v*` tag runs can use;
  - a tag ruleset restricts the creation of `v*` tags to the maintainer.
- When the tap update fails after the release is published, the generated cask is kept as a workflow artifact, so the tap can be updated by hand.
- The README documents `brew install cptconi/tap/git-cleaner` and `brew update && brew upgrade git-cleaner`.

## Capabilities

### New Capabilities
- `homebrew-distribution`: installing and updating git-cleaner with Homebrew from a dedicated tap, kept in sync by a protected release workflow.

### Modified Capabilities
<!-- None -->

## Impact

- `.goreleaser.yaml`: new `homebrew_casks` section.
- `.github/workflows/release.yml`:
  - `release` environment;
  - deploy key given to GoReleaser;
  - cask artifact kept on failure.
- New file `.github/rulesets/tags.json`, to import in the repository settings.
- Maintainer setup:
  - public repository `CptConi/homebrew-tap` with a README;
  - deploy key with write access;
  - `release` environment holding the `HOMEBREW_TAP_DEPLOY_KEY` secret.
- `README.md` (Install, Development) and the "Releases" line of the `openspec/config.yaml` context.
- If `dependabot-actions` lands first, its release snapshot check needs a placeholder for the deploy key variable.
- No change to the Go code.
