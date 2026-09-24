package main

import (
	"strings"
	"testing"
)

func TestParseExcludes(t *testing.T) {
	e, err := ParseExcludes(" acme/ , ./personal//portfolio,,clients/* ")
	if err != nil {
		t.Fatal(err)
	}
	assertStrings(t, "normalized patterns", e, []string{"acme", "personal/portfolio", "clients/*"})

	// Duplicates, once normalized, are dropped for both flags.
	if e, err = ParseExcludes("acme, acme/ ,./acme,tools"); err != nil {
		t.Fatal(err)
	}
	assertStrings(t, "deduplicated --exclude", e, []string{"acme", "tools"})
	w, err := ParseWhitelist("main,develop,main")
	if err != nil {
		t.Fatal(err)
	}
	assertStrings(t, "deduplicated --keep", w, []string{"main", "develop"})

	for _, raw := range []string{
		"[oops",              // malformed glob
		"/Users/me/acme",     // absolute path, e.g. an expanded ~/Projects/acme
		"C:/Projects/acme",   // Windows drive
		`personal\portfolio`, // backslash separator
		"../acme",            // leaving the root
		"acme/../x",          // ".." segment hidden by path.Clean
		"**/app",             // unsupported recursive wildcard
	} {
		_, err := ParseExcludes("ok," + raw)
		if err == nil || !strings.Contains(err.Error(), "--exclude") || !strings.Contains(err.Error(), raw) {
			t.Errorf("ParseExcludes(%q) = %v, want an error naming --exclude and the pattern", raw, err)
		}
	}
}

func TestCoveringFolder(t *testing.T) {
	excluded := []string{"acme", "personal/portfolio"}
	for pattern, want := range map[string]string{
		"acme/web":                "acme",
		"acme/*":                  "acme",
		"*/web":                   "acme", // only candidates inside acme remain
		"personal/portfolio/site": "personal/portfolio",
		"personal/blog":           "", // sibling of an excluded folder: really unmatched
		"acme":                    "", // same depth: not covered, it would have matched
		"tools":                   "",
	} {
		got, ok := coveringFolder(pattern, excluded)
		if got != want || ok != (want != "") {
			t.Errorf("coveringFolder(%q) = %q, %v; want %q", pattern, got, ok, want)
		}
	}
}

func TestExcludeListMatch(t *testing.T) {
	e := ExcludeList{"acme", "personal/*", "personal/portfolio"}
	for rel, want := range map[string][]string{
		"acme":               {"acme"},
		"acme/web":           nil, // only the folder itself: the walk skips its subtree
		"personal/portfolio": {"personal/*", "personal/portfolio"},
		"personal":           nil,
		"Acme":               nil, // case-sensitive
		"clients/acme":       nil, // anchored at the root
	} {
		assertStrings(t, "Match("+rel+")", e.Match(rel), want)
	}
}

func TestWhitelist(t *testing.T) {
	w, err := ParseWhitelist(" main, ,release/*,hotfix-? ")
	if err != nil {
		t.Fatal(err)
	}
	assertStrings(t, "entries", w, []string{"main", "release/*", "hotfix-?"})

	for branch, want := range map[string]bool{
		"main":              true,
		"Main":              false, // branch names are case-sensitive
		"release/1.0":       true,
		"release/1.0/fix":   false, // "*" does not match "/"
		"releases":          false,
		"hotfix-1":          true,
		"hotfix-12":         false,
		"feature/new-login": false,
	} {
		if got := w.Matches(branch); got != want {
			t.Errorf("Matches(%q) = %v, want %v", branch, got, want)
		}
	}

	if _, err := ParseWhitelist("main,[oops"); err == nil || !strings.Contains(err.Error(), "--keep") {
		t.Errorf("invalid pattern: got %v, want an error naming --keep", err)
	}
	if w, _ := ParseWhitelist(""); len(w) != 0 || w.Matches("main") || w.String() != "(none)" {
		t.Errorf("empty whitelist = %q", w)
	}
}
