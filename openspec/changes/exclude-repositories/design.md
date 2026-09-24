## Context

Discovery lives in `scan.go`: `FindRepositories(root)` walks the tree with `filepath.WalkDir` in lexical order,
skips `node_modules`, and stops descending at the first folder holding a `.git` entry (a directory is a
repository; a file marks a linked worktree or a submodule). Its single caller is `run` in `main.go`, which
returns right after printing "No Git repository found." when the list is empty, without a summary. `--keep`
(`whitelist.go`) parses comma-separated `path.Match` patterns and validates them at startup (exit status 2), but
is declared with `flag.String`, so a repeated `--keep` silently keeps only its last value. The maintainer needs
to protect whole client folders (`~/Projects/acme`, which nests repositories two levels deep) and single
repositories (`~/Projects/personal/portfolio`).

## Goals / Non-Goals

**Goals:**
- Exclude folders, and everything below them, by pattern relative to the root.
- Same pattern syntax as `--keep`, identical on every OS, with the typing mistakes that would silently match
  nothing either normalized, rejected or reported before anything happens.
- Repeatable pattern flags for `--keep` and `--exclude`.

**Non-Goals:**
- Excluding individual branches per repository (`--keep` covers branch names).
- gitignore-style name matching at any depth (`portfolio` matching `personal/portfolio`): patterns are anchored
  at the root, which keeps them predictable.
- `**` wildcards: a pattern naming a parent folder already covers its whole subtree, and `path.Match` has no
  `**`, so it is rejected rather than silently acting as `*`.
- A configuration file persisting `--keep` / `--exclude` (possible follow-up change).

## Decisions

- **Match folders during the walk, not repositories after it.** In the `WalkDir` callback, for every
  directory other than the root, the checks run in this order: `skipDirs` (`node_modules`), then the exclusion
  patterns on the slash-separated relative path (`filepath.Rel` + `filepath.ToSlash`), then the `.git` stat.
  On an exclusion match, the callback records the folder and returns `filepath.SkipDir`, so nested
  repositories are covered and excluded trees are never read (no permission warnings from them). Alternative
  considered: filtering the returned repositories by their parent folders, rejected because it still walks
  the excluded trees.
- **Normalization and rejection at parse time.** Each pattern goes through `path.Clean` (removes `./`,
  trailing `/`, `//`). Absolute paths (`/…`, `C:…`, `\\…`), `..` segments, `**` and `\` are rejected with
  exit status 2 and a message explaining that patterns are relative to the root with `/` separators.
  Converting absolute paths to relative ones was rejected: the root is symlink-resolved (`/tmp` becomes
  `/private/tmp`), so the conversion would be fragile.
- **Repeatable flags** through `flag.Func` for `--keep` and `--exclude`, each occurrence appending its
  comma-separated patterns. `--keep` keeps its default list only when the flag is absent. A shared
  `parsePatterns(flag, list)` does the splitting, trimming, cleaning and validation, and names the flag in its
  errors. `ParseWhitelist` stays as a thin wrapper, so existing callers and tests keep working.
- **Case-sensitive matching everywhere**, like `--keep`, so that a pattern behaves the same on Linux and on
  case-insensitive macOS or Windows file systems (`WalkDir` reports names as stored on disk). The warning for
  unmatched patterns catches case mistakes.
- **Reporting.**
  - `FindRepositories` returns the excluded folders along with repositories and warnings.
  - The header gains an `Exclude` line (labels padded to the longest one).
  - `Printer.ScanResult` prints `excluded  <folders>` with the same OS-native separators as repository
    paths, then a warning for each pattern that matched no folder. The scan completes before any repository
    is processed, so these lines always come first.
  - The summary gains a "Folders excluded" row, printed whenever `--exclude` was given, even at 0. It counts
    matched folders, including ones that hold no repository: nested repositories are not walked, so they
    cannot be counted.
  - With no repository left, the message becomes "No Git repository found outside the N excluded folders."
- **Error hint.** The hint printed for extra positional arguments ("separate --keep values with commas")
  names both flags.

## Risks / Trade-offs

- [A pattern typed with the wrong case matches nothing on macOS/Windows] → warning before processing,
  documented in the help text.
- [`*` not crossing `/` surprises users expecting `acme/*` to reach `acme/web/app`] → it does,
  because `acme/web` matches and its subtree is skipped; documented with examples in the README.
- [Repeating `--keep` now adds up instead of overriding] → the previous behavior was a silent trap, not a
  feature; mentioned in the help text.
- [The signature change of `FindRepositories` touches callers and tests] → single caller (`run`), tests
  updated in the same change.

## Migration Plan

Additive for `--exclude`; without it, discovery and output are unchanged. `--keep` changes only when it is
repeated. Rollback is reverting the change.

## Open Questions

- None blocking. A later change may add a configuration file so that `--keep` and `--exclude` do not have to
  be typed on every run.
