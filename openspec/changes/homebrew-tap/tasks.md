## 1. Setup (maintainer, GitHub UI)

- [ ] 1.1 Create the public repository `CptConi/homebrew-tap` with a README explaining `brew install cptconi/tap/git-cleaner`
- [ ] 1.2 Generate an SSH key pair; add the public key to the tap as a deploy key with write access
- [ ] 1.3 Create the `release` environment in git-cleaner, restricted to `v*` tags, and store the private key in it as the secret `HOMEBREW_TAP_DEPLOY_KEY`
- [ ] 1.4 Add `.github/rulesets/tags.json` (creation, update and deletion of `refs/tags/v*` restricted, repository admins as bypass actors) and import it

## 2. GoReleaser and release workflow

- [ ] 2.1 Add a `homebrew_casks` entry to `.goreleaser.yaml`:
  - `repository` (owner, name, `git.url`, `private_key` from the environment);
  - `homepage` and `description`;
  - `commit_author` (maintainer noreply identity);
  - commit message template `chore (homebrew) update git-cleaner to {{ .Tag }}`;
  - `skip_upload: auto`.
- [ ] 2.2 Emit the macOS quarantine removal as declared `postflight_steps` through `custom_block`; fall back to `hooks.post.install` only if the steps cannot be expressed, and then record the 2027-12-11 deadline in a follow-up change
- [ ] 2.3 In `release.yml`, run the GoReleaser job in the `release` environment, pass `HOMEBREW_TAP_DEPLOY_KEY`, and upload `dist/homebrew/Casks/git-cleaner.rb` as an artifact when the job fails
- [ ] 2.4 If the `dependabot-actions` release snapshot check exists, give it a placeholder `HOMEBREW_TAP_DEPLOY_KEY`

## 3. Documentation

- [ ] 3.1 README Install: `brew install cptconi/tap/git-cleaner`, `brew update && brew upgrade git-cleaner`, macOS and Linux where Homebrew supports casks
- [ ] 3.2 README Development: release credentials (deploy key, environment, tag ruleset) and the recovery procedure when the tap update fails
- [ ] 3.3 `openspec/config.yaml` context, "Releases" line: the Homebrew cask and its tap

## 4. Verification

- [ ] 4.1 `goreleaser check` and `goreleaser release --snapshot --clean` pass locally, and `dist/homebrew/Casks/git-cleaner.rb` holds darwin and linux entries for both architectures and the post-install steps
- [ ] 4.2 `go run github.com/rhysd/actionlint/cmd/actionlint@latest` passes; `gofmt -l .`, `make vet` and `go test ./...` still pass (no Go code changes)
- [ ] 4.3 After the next release, on macOS:
  - `brew install cptconi/tap/git-cleaner` prints no deprecation warning;
  - `git-cleaner --version` and `git-cleaner --dry-run` on a sample tree run without a Gatekeeper prompt.
- [ ] 4.4 After the next release, in `ghcr.io/homebrew/brew`: `brew install cptconi/tap/git-cleaner` and `git-cleaner --version` work
- [ ] 4.5 After the release that follows, `brew update && brew upgrade git-cleaner` installs it
