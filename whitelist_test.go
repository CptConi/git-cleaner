package main

import "testing"

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

	if _, err := ParseWhitelist("main,[oops"); err == nil {
		t.Error("an invalid pattern was accepted")
	}
	if w, _ := ParseWhitelist(""); len(w) != 0 || w.Matches("main") || w.String() != "(none)" {
		t.Errorf("empty whitelist = %q", w)
	}
}
