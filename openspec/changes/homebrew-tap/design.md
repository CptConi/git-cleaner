## Context

Releases are built by GoReleaser v2 (`.goreleaser.yaml`, `release.yml`) on `v*` tags:
- tar.gz archives for darwin and linux, zip for windows, for amd64 and arm64;
- checksums.

The binaries are not signed: a browser download is quarantined by Gatekeeper, which the README tells users to clear with `xattr`. GoReleaser deprecated formulas built from binaries (`brews`) in favor of casks (`homebrew_casks`). A GoReleaser run publishes the GitHub release first (draft, assets, publish) and pushes the cask afterwards. Its `hooks.post.install` option generates a Ruby `postflight` block.

Homebrew 7.0.0 (2026-09-13) changes the context in three ways:
- it deprecates cask `*flight` Ruby blocks in favor of declared `*_steps`, with warnings for third-party taps until 2027-12-11;
- it supports casks on Linux (since 4.5);
- it moves Intel Macs to its lowest support tier, and Homebrew stops running on them on 2027-09-01.

The only rulesets today protect `main` and `dev`: any `v*` tag triggers a release.

## Goals / Non-Goals

**Goals:**
- `brew install` and `brew update && brew upgrade` on macOS, and on Linux where Homebrew supports casks.
- No manual step per release once the setup exists, and a documented recovery when the tap update fails.
- No Gatekeeper prompt and no deprecation warning for Homebrew users.
- Release credentials usable only by release tag runs.

**Non-Goals:**
- Signing and notarizing the binaries (paid Apple Developer account).
- Submitting to homebrew-core (needs notability and a formula built from source).
- Windows package managers (winget, Scoop): possible follow-up change.

## Decisions

- **`homebrew_casks` rather than `brews`**: formulas generated from binaries are deprecated in GoReleaser v2; casks are the supported path, and they include the linux archives with `on_linux` blocks, so Linux support comes without extra configuration.
- **Dedicated tap `CptConi/homebrew-tap`**:
  - `brew install cptconi/tap/git-cleaner` taps it automatically, trusts it for that item and finds the cask without `--cask`.
  - Homebrew lowercases the name, and GitHub resolves it case-insensitively.
  - Alternative considered: a tap inside the git-cleaner repository, which works with the two-argument `brew tap`. It was rejected because it adds a `brew tap` step for users, and the release job would have to push to the protected `main`.
  - The tap repository gets a README.
- **Declared post-install steps for the quarantine**, emitted through the GoReleaser `custom_block`:

  ```ruby
  postflight_steps do
    on_macos do
      run "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "{{staged_path}}/git-cleaner"]
    end
  end
  ```

  This avoids `hooks.post.install`, which produces a deprecated `postflight` block: warnings on every install, errors with `HOMEBREW_DEVELOPER`, broken installs after 2027-12-11. The exact step syntax is checked on a real install. If it cannot be expressed through `custom_block`, the fallback is `hooks.post.install`, with a follow-up change due before 2027-12-11.
- **Deploy key instead of a personal access token**: GoReleaser pushes over SSH (`repository.git.url: git@github.com:CptConi/homebrew-tap.git`, `private_key: "{{ .Env.HOMEBREW_TAP_DEPLOY_KEY }}"`). A deploy key with write access covers the tap only, does not expire, and is not revoked for inactivity (a personal access token is revoked after a year without use, which is likely with infrequent releases). `GITHUB_TOKEN` cannot push to another repository.
- **`release` environment and tag ruleset**:
  - The secret lives in a GitHub environment whose deployment rule only allows `v*` tags, and the GoReleaser job declares `environment: release`.
  - `.github/rulesets/tags.json` restricts creating, updating and deleting `refs/tags/v*` to the repository admin role, i.e. the maintainer.
  - Together they make the path "tag → release → tap" require the maintainer.
- **Stable releases only**: `skip_upload: auto` leaves the cask untouched for prerelease tags such as `v1.3.0-rc1`.
- **Commit identity**: `commit_msg_template: "chore (homebrew) update git-cleaner to {{ .Tag }}"`, with `commit_author` set to the maintainer's noreply identity (`Nicolas Renard`, `66407565+CptConi@users.noreply.github.com`). This is the same identity as the rest of the project; the default GoReleaser bot is a third-party GitHub account.
- **No `license` key**: GoReleaser ignores it for casks.

## Risks / Trade-offs

- [Removing the quarantine bypasses a macOS protection] → only for the binary of this cask. Homebrew checks the sha256 written in the cask, which proves integrity against the release but not who produced it: the real trust boundary is write access to the tap and to the release pipeline, hence the deploy key scope, the `release` environment and the tag ruleset.
- [The tap push fails after the release is published (key removed, GitHub outage)] → the job fails and uploads `dist/homebrew/Casks/git-cleaner.rb` as an artifact (`if: failure()`). The maintainer commits it to the tap by hand. Re-running the job would fail earlier on the already uploaded assets; deleting the GitHub release, keeping the tag, is the other way out.
- [`brew upgrade` alone may not see a new release for up to 24 hours (Homebrew auto-update interval)] → the README says `brew update && brew upgrade git-cleaner`.
- [Linux support for casks is recent in Homebrew] → smoke test in `ghcr.io/homebrew/brew` after each release; the README presents it as supported where Homebrew supports casks, with `go install` and the archives as alternatives.
- [Intel Macs] → supported for as long as Homebrew runs on them (until 2027-09-01).

## Migration Plan

1. The maintainer:
   - creates `CptConi/homebrew-tap` with a README;
   - generates an SSH key pair and adds the public key as a write deploy key of the tap;
   - creates the `release` environment limited to `v*` tags, holding the private key as `HOMEBREW_TAP_DEPLOY_KEY`;
   - imports `.github/rulesets/tags.json`.
2. Merge the change through `dev` into `main`, then push the next tag: the first cask is created.
3. Rollback: remove the `homebrew_casks` section and the environment; the tap repository can be archived.

## Open Questions

- None.
