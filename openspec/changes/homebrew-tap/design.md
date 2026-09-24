## Context

Releases are built by GoReleaser (`.goreleaser.yaml`, `release.yml`) on `v*` tags: tar.gz archives for
darwin/linux, zip for windows, amd64 and arm64, plus checksums. The binaries are not signed: a browser download
is quarantined by Gatekeeper, which the README tells users to clear with `xattr`. GoReleaser deprecated
Homebrew formulas (`brews`) in favor of casks (`homebrew_casks`) for prebuilt binaries.

## Goals / Non-Goals

**Goals:**
- `brew install` / `brew upgrade` on macOS, Apple Silicon and Intel.
- Zero manual step per release once the prerequisites exist.
- No Gatekeeper prompt for Homebrew users.

**Non-Goals:**
- Signing and notarizing the binaries (paid Apple Developer account).
- Linux support through Homebrew: casks target macOS; Linux users keep `go install` and the release archives.
- Submitting to homebrew-core (requires notability and a formula built from source).
- Windows package managers (winget, Scoop): possible follow-up change.

## Decisions

- **`homebrew_casks` rather than `brews`**: formulas generated from binaries are deprecated in GoReleaser v2;
  casks are the supported path.
- **Dedicated tap `CptConi/homebrew-tap`**: `brew install cptconi/tap/git-cleaner` taps it automatically.
  Alternative considered: a tap inside the git-cleaner repository, which Homebrew does not support (a tap is a
  repository named `homebrew-<name>`).
- **Post-install hook** (from the GoReleaser documentation) that runs
  `xattr -dr com.apple.quarantine #{staged_path}/git-cleaner` on macOS, instead of asking every user to do it.
- **Token**: a fine-grained personal access token restricted to `CptConi/homebrew-tap` with
  "Contents: read and write", stored as the `HOMEBREW_TAP_TOKEN` secret and exposed to the GoReleaser step
  only. The default `GITHUB_TOKEN` cannot push to another repository.
- **Commit message template** `chore (homebrew) update git-cleaner to {{ .Tag }}` to keep the convention in the
  tap too; commit author left to the GoReleaser bot, so tap updates are clearly automated.

## Risks / Trade-offs

- [Removing the quarantine bypasses a macOS protection] → only for the binary installed by the cask, which
  comes from the maintainer's own release and is checked against the release checksums by Homebrew.
- [The token expires (fine-grained tokens have a lifetime)] → the release workflow fails loudly; renew it
  and re-run the job; the expiry date goes into the maintainer's reminders.
- [The tap is updated but the release failed later] → GoReleaser publishes the cask after the release
  artifacts, so a cask never points to missing archives.

## Migration Plan

1. The maintainer creates the empty public repository `CptConi/homebrew-tap`, the token and the secret.
2. Merge the change through `dev` into `main`, then publish the next tag: the first cask is created.
3. Rollback: remove the `homebrew_casks` section; the tap repository can be archived.

## Open Questions

- Should the tap commits be authored by the maintainer (noreply address) rather than the GoReleaser bot?
