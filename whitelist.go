package main

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Whitelist is the list of local branches that must never be deleted. Each
// entry is an exact branch name or a glob pattern (path.Match syntax): "*"
// matches any run of characters except "/", so "release/*" protects
// "release/1.0" but neither "release/1.0/hotfix" nor "releases".
type Whitelist []string

// ParseWhitelist parses a comma-separated list of --keep patterns such as
// "main, master,release/*". Blank entries are ignored; an empty list protects
// no branch.
func ParseWhitelist(list string) (Whitelist, error) {
	patterns, err := parsePatterns("--keep", list, nil)
	return Whitelist(patterns), err
}

// Matches reports whether branch is protected by the whitelist.
func (w Whitelist) Matches(branch string) bool {
	for _, pattern := range w {
		if ok, _ := path.Match(pattern, branch); ok {
			return true
		}
	}
	return false
}

// String returns the whitelist in a human-readable form.
func (w Whitelist) String() string {
	if len(w) == 0 {
		return "(none)"
	}
	return strings.Join(w, ", ")
}

// ExcludeList is the list of --exclude patterns: folders, relative to the
// scanned root and written with "/" separators, that are skipped together with
// everything below them. Patterns use path.Match syntax and are anchored at the
// root: "acme" excludes <root>/acme, not <root>/clients/acme.
type ExcludeList []string

// ParseExcludes parses a comma-separated list of --exclude patterns such as
// "acme,personal/portfolio". On top of the checks of --keep, it rejects the
// patterns that could never match a folder path relative to the root.
func ParseExcludes(list string) (ExcludeList, error) {
	patterns, err := parsePatterns("--exclude", list, checkExcludePattern)
	return ExcludeList(patterns), err
}

// Match returns the patterns that match rel, the slash-separated path of a
// folder relative to the scanned root (all of them, so that none is later
// reported as matching nothing).
func (e ExcludeList) Match(rel string) []string {
	var matched []string
	for _, p := range e {
		if ok, _ := path.Match(p, rel); ok {
			matched = append(matched, p)
		}
	}
	return matched
}

// drivePrefix matches Windows drive letters such as "C:" or "c:".
var drivePrefix = regexp.MustCompile(`^[A-Za-z]:`)

// checkExcludePattern rejects the --exclude patterns that can never match a
// folder path relative to the root, which would otherwise exclude nothing
// without a word.
func checkExcludePattern(raw string) error {
	switch {
	case strings.Contains(raw, `\`):
		return errors.New(`contains a backslash: use "/" as the separator on every OS`)
	case strings.HasPrefix(raw, "/") || drivePrefix.MatchString(raw):
		return errors.New(`is an absolute path: patterns are relative to <root-directory>, e.g. "acme" or "personal/portfolio"`)
	case strings.Contains(raw, "**"):
		return errors.New(`contains "**", which is not supported: a folder pattern already covers everything below it`)
	case raw == ".." || strings.HasPrefix(raw, "../") || strings.HasSuffix(raw, "/..") || strings.Contains(raw, "/../"):
		return errors.New(`contains "..": patterns are relative to <root-directory> and cannot leave it`)
	}
	return nil
}

// parsePatterns splits a comma-separated list of glob patterns: blank entries
// are ignored, each entry is trimmed, checked by check (if not nil),
// normalized with path.Clean ("acme/" and "./acme" become "acme") and
// validated with path.Match. Errors name the flag and the pattern.
func parsePatterns(flag, list string, check func(raw string) error) ([]string, error) {
	var patterns []string
	for _, item := range strings.Split(list, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if check != nil {
			if err := check(item); err != nil {
				return nil, fmt.Errorf("invalid %s pattern \"%s\": %w", flag, item, err)
			}
		}
		item = path.Clean(item)
		// Matching against an empty name only validates the pattern syntax.
		if _, err := path.Match(item, ""); err != nil {
			return nil, fmt.Errorf("invalid %s pattern \"%s\": %w", flag, item, err)
		}
		patterns = append(patterns, item)
	}
	return patterns, nil
}
