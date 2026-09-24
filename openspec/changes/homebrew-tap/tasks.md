## 1. Prerequisites (maintainer, GitHub UI)

- [ ] 1.1 Create the empty public repository `CptConi/homebrew-tap`
- [ ] 1.2 Create a fine-grained token limited to `CptConi/homebrew-tap` with "Contents: read and write", note its expiry date
- [ ] 1.3 Add it to git-cleaner as the Actions secret `HOMEBREW_TAP_TOKEN`

## 2. GoReleaser

- [ ] 2.1 Add a `homebrew_casks` entry to `.goreleaser.yaml`: repository `CptConi/homebrew-tap` with the token from the environment, homepage, description, MIT license, commit message template `chore (homebrew) update git-cleaner to {{ .Tag }}`
- [ ] 2.2 Add the post-install hook removing `com.apple.quarantine` from `#{staged_path}/git-cleaner` on macOS
- [ ] 2.3 Pass `HOMEBREW_TAP_TOKEN` to the GoReleaser step in `release.yml`

## 3. Documentation

- [ ] 3.1 README Install section: `brew install cptconi/tap/git-cleaner`, `brew upgrade git-cleaner`, and that Homebrew is for macOS

## 4. Verification

- [ ] 4.1 `goreleaser check` and `goreleaser release --snapshot --clean` pass locally, and the generated cask in `dist/homebrew/` holds both architectures and the hook
- [ ] 4.2 `actionlint` passes; `gofmt -l .`, `go vet ./...` (linux, darwin, windows) and `go test ./...` still pass
- [ ] 4.3 After the next release: `brew install cptconi/tap/git-cleaner` on macOS, then `git-cleaner --version` and a dry run on a sample tree work without a Gatekeeper prompt
