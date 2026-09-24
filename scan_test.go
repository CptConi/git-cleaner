package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindRepositories(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		"alpha/.git",
		"alpha/vendor/lib/.git", // inside a repository: not scanned
		"group/beta/.git",
		"group/notes",           // plain directory
		"node_modules/pkg/.git", // dependency directory: skipped
		"linked/sub/.git",       // inside a linked worktree: not scanned
	} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A ".git" file marks a linked worktree or a submodule, not a repository.
	if err := os.WriteFile(filepath.Join(root, "linked", ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	checkUnreadable := runtime.GOOS != "windows" && os.Geteuid() != 0 // root ignores permissions
	if checkUnreadable {
		locked := filepath.Join(root, "locked")
		if err := os.Mkdir(locked, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(locked, 0o755) }) // let TempDir remove it
	}

	repos, warnings, err := FindRepositories(root)
	if err != nil {
		t.Fatal(err)
	}
	assertStrings(t, "repositories", repos, []string{
		filepath.Join(root, "alpha"),
		filepath.Join(root, "group", "beta"),
	})
	if checkUnreadable && len(warnings) != 1 {
		t.Errorf("warnings = %q, want one for the unreadable directory", warnings)
	}

	// The root may itself be a repository.
	repos, _, err = FindRepositories(filepath.Join(root, "alpha"))
	if err != nil {
		t.Fatal(err)
	}
	assertStrings(t, "repositories from a repository root", repos, []string{filepath.Join(root, "alpha")})
}
