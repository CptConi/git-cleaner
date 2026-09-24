## ADDED Requirements

### Requirement: Exclude folders by relative path pattern
The CLI SHALL accept an `--exclude` option holding comma-separated glob patterns (`path.Match` syntax, where
`*` does not match `/`). Each pattern SHALL be matched against the path of every folder met during discovery,
relative to the scanned root and written with `/` separators on every operating system. Blank entries SHALL be
ignored and matching SHALL be case-sensitive. A pattern SHALL never match the scanned root itself.

#### Scenario: Excluding a folder by name
- **WHEN** the user runs `git-cleaner --exclude 'acme' ~/Projects` and `~/Projects/acme` holds several repositories,
  some of them nested in sub-folders
- **THEN** none of the repositories under `~/Projects/acme` is processed, and every other repository is

#### Scenario: Excluding with a wildcard
- **WHEN** the user runs `git-cleaner --exclude 'personal/*' ~/Projects`
- **THEN** every folder directly under `~/Projects/personal`, and everything below them, is excluded

#### Scenario: Excluding a single repository
- **WHEN** the user runs `git-cleaner --exclude 'personal/portfolio' ~/Projects` and `~/Projects/personal/portfolio` is a
  repository
- **THEN** that repository is not processed while its sibling repositories are

#### Scenario: Same patterns on Windows
- **WHEN** the user runs `git-cleaner --exclude 'personal/portfolio' C:\Projects` on Windows
- **THEN** `C:\Projects\personal\portfolio` is excluded

### Requirement: Excluded folders are left untouched
The CLI SHALL NOT walk an excluded folder, nor fetch, prune or delete anything in the repositories it contains,
in dry-run mode as in a real run.

#### Scenario: Real run next to an excluded repository
- **WHEN** the user runs `git-cleaner --exclude 'personal/portfolio' ~/Projects` without `--dry-run`
- **THEN** the references of `~/Projects/personal/portfolio`, local and remote-tracking, are identical before and after
  the run

### Requirement: Exclusions are reported
The CLI SHALL list the excluded folders after the scan and SHALL show their number in the final summary, so
that an exclusion is never silent.

#### Scenario: Scan output with exclusions
- **WHEN** a run excludes the folders `acme` and `personal/portfolio`
- **THEN** the scan output lists both folders and the summary reports 2 excluded folders

#### Scenario: Pattern that matches nothing
- **WHEN** the user passes `--exclude 'Nope'` and no folder is called `Nope`
- **THEN** the run proceeds normally and the summary reports 0 excluded folders

### Requirement: Invalid exclusion patterns are rejected
The CLI SHALL reject a malformed `--exclude` pattern before scanning, with an error message naming the pattern
and exit status 2.

#### Scenario: Malformed pattern
- **WHEN** the user runs `git-cleaner --exclude 'acme,[oops' ~/Projects`
- **THEN** git-cleaner prints an error about `[oops`, scans nothing and exits with status 2
