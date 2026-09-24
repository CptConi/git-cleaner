## 1. Configuration

- [ ] 1.1 Add `.github/dependabot.yml`: `github-actions` ecosystem, directory `/`, monthly schedule, `target-branch: dev`, a single group for every action, `commit-message.prefix: "chore (deps) "`
- [ ] 1.2 Validate the file against the Dependabot options reference (keys, trailing-space prefix)

## 2. Repository settings (maintainer, GitHub UI)

- [ ] 2.1 Allow squash merging with "Pull request title" as the default commit message
- [ ] 2.2 Decide on the open question: restrict pull request merges into `dev` to squash merges in the `dev` ruleset

## 3. Documentation

- [ ] 3.1 README Contributing: Dependabot pull requests target `dev` and are squash-merged with their title

## 4. Verification

- [ ] 4.1 `actionlint` still passes and CI is green on the feature branch
- [ ] 4.2 After the next merge of `dev` into `main`, check in the Insights > Dependency graph > Dependabot tab that the configuration is picked up without errors
