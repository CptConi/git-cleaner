## Why

Some repositories must not be touched at all, even when every branch they hold is pushed: active client
projects or a personal portfolio whose local branches the maintainer wants to keep at hand. `--keep` only
matches branch names, so today the only workaround is to run git-cleaner on several narrower roots.

While sharing the pattern parser, this change also fixes a trap of the current `--keep`: when the flag is
repeated (`--keep main --keep develop`), only the last value is kept, silently.

## What Changes

- New `--exclude <patterns>` option: comma-separated glob patterns matched against the path of each folder
  relative to the scanned root, with `/` separators on every OS (e.g. `--exclude 'acme,personal/portfolio'`).
  Patterns are anchored at the root and normalized (`acme/` and `./acme` mean `acme`).
- A folder that matches a pattern is skipped during discovery together with everything below it: its
  repositories are neither fetched, pruned nor cleaned, and its subtree is not even walked.
- Exclusions are never silent: the run header lists the patterns, the scan result lists the excluded folders,
  a pattern that matched no folder triggers a warning before any repository is processed, and the summary counts
  the excluded folders (also when no repository is left to process).
- Patterns that cannot match are rejected at startup with exit status 2: malformed globs, absolute paths, `..`
  segments, `**` and backslashes.
- `--keep` and `--exclude` can be repeated; the values of every occurrence add up (**behavior fix** for
  `--keep`, which kept only its last occurrence).

## Capabilities

### New Capabilities
- `repository-exclusion`: excluding folders, and the repositories they contain, from a run by relative path
  pattern, and repeating the pattern flags.

### Modified Capabilities
<!-- None: no baseline spec exists yet for repository discovery or the keep list. -->

## Impact

- `main.go`: new flag, `Options.Exclude`, repeatable pattern flags, help text, the hint printed for extra
  positional arguments, the "no repository found" path.
- `whitelist.go` / `whitelist_test.go`: shared pattern parsing (`ParseWhitelist` kept as a wrapper).
- `scan.go`: `FindRepositories` takes the exclusion patterns and returns the excluded folders.
- `report.go`: header, scan result, warnings and summary lines.
- `README.md` (Features, Usage options, How it works) and the `openspec/config.yaml` context.
- Tests: `scan_test.go`, `whitelist_test.go`, `main_test.go`, end-to-end tests in `cleaner_test.go`.
- No change to the per-repository cleanup pipeline, no new dependency.
