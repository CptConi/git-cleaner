## 1. Pattern parsing

- [ ] 1.1 Extract the comma splitting, trimming and `path.Match` validation of `ParseWhitelist` into a shared helper whose error names the flag (`--keep` or `--exclude`)
- [ ] 1.2 Add the `--exclude` flag and `Options.Exclude`; reject malformed patterns with exit status 2
- [ ] 1.3 Unit tests: parsing, blank entries, invalid pattern message naming `--exclude`, flag accepted after the root argument

## 2. Discovery

- [ ] 2.1 Make `FindRepositories` take the exclusion patterns: match each folder below the root on its slash-separated relative path, record it and return `filepath.SkipDir`
- [ ] 2.2 Return the excluded folders along with repositories and warnings; update the caller in `run`
- [ ] 2.3 Tests in `scan_test.go`: excluded parent folder with nested repositories, wildcard pattern, single repository, pattern matching nothing, the root never matched, relative paths built with `/` on Windows

## 3. Reporting

- [ ] 3.1 Print the excluded folders in `Printer.ScanResult` and add a "Folders excluded" line to the summary
- [ ] 3.2 End-to-end test: a real run next to an excluded repository leaves its references unchanged and reports it

## 4. Documentation

- [ ] 4.1 Document `--exclude` in `usage()` (case-sensitive matching, parent folders cover their subtree) and in the README options table, with examples

## 5. Verification

- [ ] 5.1 `gofmt -l .` is empty and `go vet ./...` passes for linux, darwin and windows
- [ ] 5.2 `go test -race ./...` passes
- [ ] 5.3 Dry run on a sample tree: `git-cleaner --dry-run --exclude 'acme,personal/portfolio' ~/Projects` lists both folders as excluded and processes the others
