## 1. Pattern parsing

- [x] 1.1 Extract a shared `parsePatterns(flag, list)` from `ParseWhitelist`: comma splitting, trimming, blank entries ignored, `path.Clean`, `path.Match` validation, errors naming the flag; keep `ParseWhitelist` as a wrapper
- [x] 1.2 For `--exclude`, also reject absolute paths, `..` segments, `**` and backslashes with a message explaining that patterns are relative to the root with `/` separators
- [x] 1.3 Declare `--keep` and `--exclude` with `flag.Func` so that repeated occurrences add up; `--keep` falls back to the default list only when absent; add `Options.Exclude`
- [x] 1.4 Name both flags in the hint printed for extra positional arguments
- [x] 1.5 Unit tests: normalization (`acme/`, `./acme`), each rejection case with exit status 2, repeated `--keep` and `--exclude`, flags accepted after the root argument

## 2. Discovery

- [x] 2.1 Make `FindRepositories` take the exclusion patterns; below the root, check `skipDirs`, then the slash-separated relative path against the patterns (record the folder, return `filepath.SkipDir`), then `.git`
- [x] 2.2 Return the excluded folders along with repositories and warnings; update the caller in `run`
- [x] 2.3 Tests in `scan_test.go`: excluded parent folder with nested repositories, wildcard pattern, single repository, case-sensitive matching, pattern anchored at the root, the root never matched (`*`, `?`, `.`), an unreadable folder inside an excluded one producing no warning, relative paths built with `/` on Windows

## 3. Reporting

- [x] 3.1 Header: `Exclude` line listing the patterns, labels padded to the longest one
- [x] 3.2 `Printer.ScanResult`: excluded folders with OS-native separators, then one warning per pattern that matched no folder
- [x] 3.3 Summary: "Folders excluded" row whenever `--exclude` is given; "No Git repository found outside the N excluded folders." when nothing is left
- [x] 3.4 End-to-end tests: real run next to an excluded clone holding deletable branches and stale remote-tracking branches (references unchanged, no `.git/FETCH_HEAD` created), dry-run output not mentioning its branches, unmatched pattern warning printed before the first repository, everything excluded

## 4. Documentation

- [x] 4.1 `usage()`: `--exclude`, anchored and case-sensitive patterns, parent folders covering their subtree, repeatable flags
- [x] 4.2 README: Features bullet, Usage options table and examples, How it works (Discovery)
- [x] 4.3 `openspec/config.yaml` context and README file table: `whitelist.go` handles the pattern flags

## 5. Verification

- [ ] 5.1 `gofmt -l .` is empty and `go vet ./...` passes for linux, darwin and windows
- [ ] 5.2 `go test -race ./...` passes
- [ ] 5.3 Dry run on a temporary sample tree with nested repositories: `--exclude` of a parent folder, of a single repository, and of a pattern matching nothing behave as specified
- [ ] 5.4 Dry run on the maintainer's `~/Projects` with `--exclude 'acme,personal/portfolio'`: both folders are listed as excluded and no branch of theirs appears in the output
