## Context

Discovery lives in `scan.go`: `FindRepositories(root)` walks the tree with `filepath.WalkDir`, skips
`node_modules`, stops at the first folder holding a `.git` directory and returns the repositories in lexical
order. `--keep` (`whitelist.go`) already parses comma-separated `path.Match` patterns, validates them at startup
and exits with status 2 on error. The maintainer needs to protect whole client folders (`~/Projects/acme`, which
nests repositories two levels deep) and single repositories (`~/Projects/personal/portfolio`).

## Goals / Non-Goals

**Goals:**
- Exclude folders, and everything below them, by pattern relative to the root.
- Same pattern syntax and validation as `--keep`, identical on every OS.
- Excluded folders are visible in the output.

**Non-Goals:**
- Excluding individual branches per repository (`--keep` covers branch names).
- A configuration file persisting `--keep` / `--exclude` (possible follow-up change).
- `**` recursive wildcards: matching folders as the walk descends already makes a single pattern cover a
  whole subtree.

## Decisions

- **Match folders during the walk, not repositories after it.** In the `WalkDir` callback, every directory
  below the root has its relative path (`filepath.Rel` + `filepath.ToSlash`) checked against the patterns; on
  a match the callback records it and returns `filepath.SkipDir`. A pattern naming a parent folder thus covers
  nested repositories, and excluded trees are never walked (faster, no permission warnings from them).
  Alternative considered: filtering the returned repository list, which needs `**` to cover nested
  repositories and still walks the excluded trees.
- **Reuse the whitelist parser.** `ParseWhitelist` already does comma splitting, trimming and pattern
  validation. It becomes a generic `parsePatterns(flag, list)` used by both flags, so that error messages name
  the right flag. The `Whitelist` type keeps its `Matches` method; exclusions use a small `patternList` with
  the same semantics.
- **Case-sensitive matching everywhere**, like `--keep`, so that a pattern behaves the same on Linux and on
  case-insensitive macOS or Windows file systems (`WalkDir` reports names as stored on disk).
- **Reporting.** `FindRepositories` returns the excluded folders in addition to repositories and warnings;
  `Printer.ScanResult` prints them after the warnings (`excluded  acme, personal/portfolio`), and `Summary`
  gains an `Excluded` counter printed as "Folders excluded".

## Risks / Trade-offs

- [A pattern typed with the wrong case silently matches nothing on macOS/Windows] → the summary shows 0
  excluded folders and the help text states that matching is case-sensitive.
- [`*` not crossing `/` surprises users expecting `acme/*` to reach `acme/web/app`] → it does,
  because `acme/web` matches and its subtree is skipped; documented with examples in the README.
- [The signature change of `FindRepositories` touches callers and tests] → single caller (`run`), tests
  updated in the same change.

## Migration Plan

Purely additive: without `--exclude`, discovery is unchanged. Rollback is reverting the change.

## Open Questions

- None blocking. A later change may add a configuration file so that `--keep` and `--exclude` do not have to
  be typed on every run.
