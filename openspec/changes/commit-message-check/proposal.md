## Why

The repository follows a strict commit convention: one single line `<type> (<feature>) <changes>` and never a
`Co-Authored-By` trailer. It is only enforced by discipline today, so a multi-line message or an automatic
trailer can slip into `dev` and, from there, into `main`.

## What Changes

- A versioned `commit-msg` hook (`.githooks/commit-msg`) rejects a non-compliant message before the commit is
  created, once enabled with `git config core.hooksPath .githooks`.
- A CI job checks every commit a push or a pull request introduces, and becomes part of the `tests passed`
  check that the `dev` and `main` rulesets require.
- Both rely on a single POSIX shell checker, so the rule is defined once.
- The convention is documented in the README (Contributing).

## Capabilities

### New Capabilities
- `commit-conventions`: the commit message format of the repository and how it is enforced locally and in CI.

### Modified Capabilities
<!-- None -->

## Impact

- New files: `scripts/check-commit-msg.sh`, `.githooks/commit-msg`.
- `.github/workflows/ci.yml`: new `commit-messages` job (full history checkout), added to the needs of
  `tests-passed`.
- `README.md`: Contributing section (format, allowed types, hook activation).
- No change to the Go code.
- Coupling with the `dependabot-actions` change: bot commits need an exemption or a squash merge.
