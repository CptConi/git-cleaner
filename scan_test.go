package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// mkdirs creates the given slash-separated directories under root.
func mkdirs(t *testing.T, root string, dirs ...string) {
	t.Helper()
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// under joins slash-separated paths below root.
func under(root string, rels ...string) []string {
	var paths []string
	for _, rel := range rels {
		paths = append(paths, filepath.Join(root, filepath.FromSlash(rel)))
	}
	return paths
}

// canTestPermissions reports whether unreadable directories can be created:
// not on Windows, and not as root, which ignores permissions.
func canTestPermissions() bool {
	return runtime.GOOS != "windows" && os.Geteuid() != 0
}

// lockDir creates an unreadable directory, made readable again at cleanup so
// that TempDir can remove it.
func lockDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Mkdir(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
}

func TestFindRepositories(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root,
		"alpha/.git",
		"alpha/vendor/lib/.git", // inside a repository: not scanned
		"group/beta/.git",
		"group/notes",           // plain directory
		"node_modules/pkg/.git", // dependency directory: skipped
		"linked/sub/.git",       // inside a linked worktree: not scanned
	)
	// A ".git" file marks a linked worktree or a submodule, not a repository.
	if err := os.WriteFile(filepath.Join(root, "linked", ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if canTestPermissions() {
		lockDir(t, filepath.Join(root, "locked"))
	}

	scan, err := FindRepositories(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertStrings(t, "repositories", scan.Repos, under(root, "alpha", "group/beta"))
	if canTestPermissions() && len(scan.Warnings) != 1 {
		t.Errorf("warnings = %q, want one for the unreadable directory", scan.Warnings)
	}
	if len(scan.Excluded)+len(scan.Unmatched) > 0 {
		t.Errorf("excluded = %q, unmatched = %q; want none without --exclude", scan.Excluded, scan.Unmatched)
	}

	// The root may itself be a repository.
	scan, err = FindRepositories(filepath.Join(root, "alpha"), nil)
	if err != nil {
		t.Fatal(err)
	}
	assertStrings(t, "repositories from a repository root", scan.Repos, under(root, "alpha"))
}

func TestFindRepositoriesWithExclusions(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root,
		"acme/web/app/.git", // nested two levels deep in a client folder
		"acme/api/.git",
		"personal/portfolio/.git",
		"personal/blog/.git",
		"tools/.git",
	)
	all := []string{"acme/api", "acme/web/app", "personal/blog", "personal/portfolio", "tools"}

	for _, c := range []struct {
		name            string
		exclude         ExcludeList
		repos, excluded []string
		unmatched       []string
	}{
		{"parent folder with nested repositories", ExcludeList{"acme"},
			[]string{"personal/blog", "personal/portfolio", "tools"}, []string{"acme"}, nil},
		{"wildcard", ExcludeList{"personal/*"},
			[]string{"acme/api", "acme/web/app", "tools"}, []string{"personal/blog", "personal/portfolio"}, nil},
		{"single repository", ExcludeList{"personal/portfolio"},
			[]string{"acme/api", "acme/web/app", "personal/blog", "tools"}, []string{"personal/portfolio"}, nil},
		{"nested folder, slash-separated on every OS", ExcludeList{"acme/web"},
			[]string{"acme/api", "personal/blog", "personal/portfolio", "tools"}, []string{"acme/web"}, nil},
		{"case-sensitive", ExcludeList{"Acme"}, all, nil, []string{"Acme"}},
		{"anchored at the root", ExcludeList{"portfolio"}, all, nil, []string{"portfolio"}},
		{"duplicate pattern", ExcludeList{"tools", "tools"},
			[]string{"acme/api", "acme/web/app", "personal/blog", "personal/portfolio"}, []string{"tools"}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			scan, err := FindRepositories(root, c.exclude)
			if err != nil {
				t.Fatal(err)
			}
			assertStrings(t, "repositories", scan.Repos, under(root, c.repos...))
			assertStrings(t, "excluded", scan.Excluded, under(root, c.excluded...))
			assertStrings(t, "unmatched", scan.Unmatched, c.unmatched)
		})
	}

	// The root itself is never excluded, whatever the pattern.
	scan, err := FindRepositories(filepath.Join(root, "tools"), ExcludeList{"*", "?", "."})
	if err != nil {
		t.Fatal(err)
	}
	assertStrings(t, "repositories from an excluded-looking root", scan.Repos, under(root, "tools"))
	assertStrings(t, "unmatched", scan.Unmatched, []string{"*", "?", "."})
}

// An excluded folder is not even read: no warning for what it contains.
func TestFindRepositoriesDoesNotWalkExcludedFolders(t *testing.T) {
	if !canTestPermissions() {
		t.Skip("needs a non-root user on a Unix-like system")
	}
	root := t.TempDir()
	mkdirs(t, root, "acme", "tools/.git")
	lockDir(t, filepath.Join(root, "acme", "locked"))

	scan, err := FindRepositories(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Warnings) != 1 {
		t.Fatalf("without --exclude: warnings = %q, want one", scan.Warnings)
	}
	if scan, err = FindRepositories(root, ExcludeList{"acme"}); err != nil {
		t.Fatal(err)
	}
	if len(scan.Warnings) != 0 {
		t.Errorf("with --exclude acme: warnings = %q, want none", scan.Warnings)
	}
}
