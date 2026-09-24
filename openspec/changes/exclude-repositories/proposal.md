## Why

Some repositories must not be touched at all, even when every branch they hold is pushed: active client
projects or a personal portfolio whose local branches the maintainer wants to keep at hand. `--keep` only
matches branch names, so today the only workaround is to run git-cleaner on several narrower roots.

## What Changes

- New `--exclude <patterns>` option: comma-separated glob patterns matched against the path of each folder
  relative to the scanned root, with `/` separators on every OS (e.g. `--exclude 'acme,personal/portfolio'`).
- A folder that matches a pattern is skipped during discovery together with everything below it: its
  repositories are neither fetched, pruned nor cleaned, and nested folders are not even walked.
- The scan result lists the excluded folders and the summary counts them, so an exclusion is never silent.
- Invalid patterns are rejected at startup with exit status 2, like invalid `--keep` patterns.

## Capabilities

### New Capabilities
- `repository-exclusion`: excluding folders, and the repositories they contain, from a run by relative path
  pattern.

### Modified Capabilities
<!-- None: no baseline spec exists yet for repository discovery. -->

## Impact

- `main.go`: new flag, `Options.Exclude`, help text.
- `scan.go`: `FindRepositories` takes the exclusion patterns and returns the excluded folders.
- `report.go`: scan result and summary lines for excluded folders.
- `README.md`: options table and examples.
- Tests: `scan_test.go` (matching, pruned walk, Windows separators), `main_test.go` (flag parsing and
  validation), end-to-end run.
- No change to the per-repository cleanup pipeline, no new dependency.
