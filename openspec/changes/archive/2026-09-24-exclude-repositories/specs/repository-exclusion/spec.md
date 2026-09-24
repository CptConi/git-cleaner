## ADDED Requirements

### Requirement: Exclude folders by relative path pattern
The CLI SHALL accept an `--exclude` option holding comma-separated glob patterns (`path.Match` syntax, where `*` does not match `/`). Each pattern SHALL be normalized with `path.Clean`, SHALL be anchored at the scanned root, and SHALL be matched against the path of every folder met during discovery, relative to that root and written with `/` separators on every operating system. Blank entries SHALL be ignored, matching SHALL be case-sensitive, and a pattern SHALL never match the scanned root itself.

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

#### Scenario: Normalized pattern
- **WHEN** the user runs `git-cleaner --exclude 'acme/' ~/Projects` or `git-cleaner --exclude './acme' ~/Projects`
- **THEN** `~/Projects/acme` is excluded, as with `--exclude 'acme'`

#### Scenario: Patterns are anchored at the root
- **WHEN** the user runs `git-cleaner --exclude 'portfolio' ~/Projects` and the only folder with that name is
  `~/Projects/personal/portfolio`
- **THEN** nothing is excluded and a warning says that the pattern matched no folder

#### Scenario: Same patterns on Windows
- **WHEN** the user runs `git-cleaner --exclude "personal/portfolio" C:\Projects` on Windows
- **THEN** `C:\Projects\personal\portfolio` is excluded

### Requirement: Pattern flags can be repeated
The `--keep` and `--exclude` options SHALL accept several occurrences, and the patterns of every occurrence SHALL add up.

#### Scenario: Repeated --exclude
- **WHEN** the user runs `git-cleaner --exclude 'acme' --exclude 'personal/portfolio' ~/Projects`
- **THEN** both `~/Projects/acme` and `~/Projects/personal/portfolio` are excluded

#### Scenario: Repeated --keep
- **WHEN** the user runs `git-cleaner --keep main --keep develop ~/Projects`
- **THEN** both `main` and `develop` are protected

### Requirement: Excluded folders are left untouched
The CLI SHALL NOT walk an excluded folder, nor fetch, prune or delete anything in the repositories it contains, in dry-run mode as in a real run.

#### Scenario: Real run next to an excluded repository
- **WHEN** the user runs `git-cleaner --exclude 'personal/portfolio' ~/Projects` without `--dry-run`, and
  `~/Projects/personal/portfolio` holds pushed branches outside the keep list and stale remote-tracking branches
- **THEN** its references, local and remote-tracking, are identical before and after the run, and no fetch ran
  in it

#### Scenario: Dry run next to an excluded repository
- **WHEN** the user runs `git-cleaner --dry-run --exclude 'personal/portfolio' ~/Projects`
- **THEN** the output mentions no branch of `~/Projects/personal/portfolio`

#### Scenario: Unreadable folder inside an excluded folder
- **WHEN** an excluded folder contains a sub-folder that cannot be read
- **THEN** no warning is printed about that sub-folder

### Requirement: Exclusions are reported
The CLI SHALL list the `--exclude` patterns in the run header and the excluded folders after the scan. It SHALL warn once, before processing any repository, about every distinct pattern that matched no folder, and SHALL report a pattern that could only match inside an already excluded folder as covered by that folder instead. Whenever `--exclude` is given, the final summary SHALL show the number of excluded folders, and a run left with no repository to process SHALL say how many folders were excluded.

#### Scenario: Scan output with exclusions
- **WHEN** a run excludes the folders `acme` and `personal/portfolio`
- **THEN** the header lists both patterns, the scan output lists both folders, and the summary reports 2
  excluded folders

#### Scenario: Pattern that matches nothing
- **WHEN** the user passes `--exclude 'Acme'` and the folder is called `acme`
- **THEN** a warning saying that `Acme` matched no folder is printed before any repository is processed, and
  the summary reports 0 excluded folders

#### Scenario: Pattern covered by an excluded folder
- **WHEN** the user passes `--exclude 'acme,acme/web'`
- **THEN** `acme` is excluded and a warning says that `acme/web` is already covered by the exclusion of `acme`,
  instead of saying that it matched no folder

#### Scenario: Duplicate patterns
- **WHEN** the user passes `--exclude 'nope,nope'` and no folder is called `nope`
- **THEN** a single warning is printed for `nope`

#### Scenario: Everything excluded
- **WHEN** every repository under the root is inside an excluded folder
- **THEN** git-cleaner says that no repository was found outside the excluded folders, gives their number and
  exits with status 0

### Requirement: Invalid exclusion patterns are rejected
The CLI SHALL reject, before scanning, any `--exclude` pattern that is a malformed glob, an absolute path, contains a `..` segment, contains `**` or contains a backslash, with an error message naming the pattern and exit status 2.

#### Scenario: Malformed pattern
- **WHEN** the user runs `git-cleaner --exclude 'acme,[oops' ~/Projects`
- **THEN** git-cleaner prints an error about `[oops`, scans nothing and exits with status 2

#### Scenario: Absolute path given as pattern
- **WHEN** the user runs `git-cleaner --exclude ~/Projects/acme ~/Projects`, which the shell expands to an absolute path
- **THEN** git-cleaner explains that patterns are relative to the root, scans nothing and exits with status 2

#### Scenario: Windows separator
- **WHEN** the user runs `git-cleaner --exclude "personal\portfolio" C:\Projects`
- **THEN** git-cleaner asks for `/` separators, scans nothing and exits with status 2
