## 1. Configuration

- [ ] 1.1 Add `.github/dependabot.yml`: `github-actions` ecosystem, directory `/`, monthly schedule, `target-branch: dev`, a single group `actions` with `patterns: ["*"]`, `commit-message.prefix: "chore (deps) "`
- [ ] 1.2 Validate it with `check-jsonschema --builtin-schema vendor.dependabot .github/dependabot.yml`

## 2. Release snapshot check

- [ ] 2.1 Add `.github/workflows/release-check.yml`: on pull requests changing `.github/workflows/release.yml` or `.goreleaser.yaml`, run the GoReleaser action with `release --snapshot --clean` (read-only permissions, not a required check)

## 3. Repository settings (maintainer, GitHub UI)

- [ ] 3.1 Allow squash merging with "Pull request title" as the default commit message, and keep merge commits allowed
- [ ] 3.2 Disable Dependabot security updates (keep Dependabot alerts enabled)

## 4. Documentation

- [ ] 4.1 README Contributing: Dependabot pull requests target `dev` and are squash-merged with their title; fixes go through a feature branch; `@dependabot ignore <action> major version` for a breaking action
- [ ] 4.2 `openspec/config.yaml` git flow: squash-merged Dependabot pull requests are the second way into `dev`

## 5. Verification

- [ ] 5.1 `actionlint` passes on the workflows and `check-jsonschema` on `dependabot.yml`
- [ ] 5.2 `gofmt -l .`, `go vet ./...` (linux, darwin, windows) and `go test ./...` still pass; a dry run on a sample tree behaves as before
- [ ] 5.3 A pull request touching `.goreleaser.yaml` shows the release snapshot check green
- [ ] 5.4 After the next merge of `dev` into `main`: the Dependabot tab (Insights > Dependency graph) shows no configuration error; on the first update pull request, the base is `dev`, the title matches the commit regex, nothing was opened against `main`, and after the squash merge `git log -1 --format=%B origin/dev` holds a single line
