package main

import (
	"fmt"
	"path"
	"strings"
)

// Whitelist is the list of local branches that must never be deleted. Each
// entry is an exact branch name or a glob pattern (path.Match syntax): "*"
// matches any run of characters except "/", so "release/*" protects
// "release/1.0" but neither "release/1.0/hotfix" nor "releases".
type Whitelist []string

// ParseWhitelist parses a comma-separated list such as "main, master,release/*".
// Blank entries are ignored; an empty list protects no branch.
func ParseWhitelist(list string) (Whitelist, error) {
	var w Whitelist
	for _, item := range strings.Split(list, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		// Matching against an empty name only validates the pattern syntax.
		if _, err := path.Match(item, ""); err != nil {
			return nil, fmt.Errorf("invalid --keep pattern %q: %w", item, err)
		}
		w = append(w, item)
	}
	return w, nil
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
