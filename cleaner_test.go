package main

// Integration tests: they drive the real git executable on repositories
// created in temporary directories.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// isolateGit makes git ignore the user and system configuration and gives
// it an identity, so that the tests behave the same on every machine.
func isolateGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	home := t.TempDir()
	for name, value := range map[string]string{
		"HOME":                home,
		"USERPROFILE":         home,
		"XDG_CONFIG_HOME":     home,
		"GIT_CONFIG_GLOBAL":   filepath.Join(home, ".gitconfig"),
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME":     "Test",
		"GIT_AUTHOR_EMAIL":    "test@example.com",
		"GIT_COMMITTER_NAME":  "Test",
		"GIT_COMMITTER_EMAIL": "test@example.com",
	} {
		t.Setenv(name, value)
	}
}

// gitT runs a git command of a test setup and returns its trimmed output.
func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// fixture builds this layout in a temporary root and returns root and work:
//
//	remote.git/  bare repository acting as the "origin" remote
//	wt/          linked worktree of work, on branch feature-c
//	work/        repository with the local branches:
//	    main       whitelisted
//	    develop    whitelisted
//	    feature-a  pushed, merged into main, then deleted on the remote
//	    feature-b  never pushed: holds one commit found nowhere else
//	    feature-c  checked out in the wt/ worktree
//	    fix        checked out in work/ (HEAD)
func fixture(t *testing.T) (root, work string) {
	t.Helper()
	isolateGit(t)
	root = t.TempDir()
	remote := filepath.Join(root, "remote.git")
	work = filepath.Join(root, "work")
	commit := func(file string) {
		if err := os.WriteFile(filepath.Join(work, file), []byte(file+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitT(t, work, "add", file)
		gitT(t, work, "commit", "-q", "-m", "add "+file)
	}

	gitT(t, root, "init", "-q", "--bare", remote)
	gitT(t, root, "init", "-q", work)
	gitT(t, work, "symbolic-ref", "HEAD", "refs/heads/main")
	gitT(t, work, "remote", "add", "origin", remote)
	commit("a")
	gitT(t, work, "push", "-q", "origin", "main")

	gitT(t, work, "switch", "-q", "-c", "feature-a")
	commit("b")
	gitT(t, work, "push", "-q", "origin", "feature-a")
	gitT(t, work, "switch", "-q", "main")
	gitT(t, work, "merge", "-q", "--ff-only", "feature-a")
	gitT(t, work, "push", "-q", "origin", "main")
	gitT(t, remote, "branch", "-D", "feature-a") // deleted on the remote only

	gitT(t, work, "switch", "-q", "-c", "feature-b")
	commit("c")
	gitT(t, work, "switch", "-q", "main")
	gitT(t, work, "branch", "develop")
	gitT(t, work, "branch", "feature-c")
	gitT(t, work, "worktree", "add", "-q", filepath.Join(root, "wt"), "feature-c")
	gitT(t, work, "switch", "-q", "-c", "fix")
	return root, work
}

func newTestCleaner(t *testing.T, dryRun bool) *Cleaner {
	t.Helper()
	git, err := NewGit()
	if err != nil {
		t.Fatal(err)
	}
	keep, err := ParseWhitelist(defaultKeep)
	if err != nil {
		t.Fatal(err)
	}
	return NewCleaner(git, &Options{Keep: keep, DryRun: dryRun, Jobs: 1, Timeout: time.Minute})
}

func branchNames(branches []BranchInfo) []string {
	var names []string
	for _, b := range branches {
		names = append(names, b.Name)
	}
	return names
}

func skippedNames(skipped []SkippedBranch, failed bool) []string {
	var names []string
	for _, s := range skipped {
		if s.Failed == failed {
			names = append(names, s.Name)
		}
	}
	return names
}

func localBranches(t *testing.T, repo string) []string {
	return strings.Fields(gitT(t, repo, "for-each-ref", "--format=%(refname:short)", "refs/heads"))
}

// checkPlan verifies the decisions taken on the fixture, which are the same
// in dry-run and real mode.
func checkPlan(t *testing.T, c *Cleaner, res *RepoResult) {
	t.Helper()
	if len(res.Errors) > 0 {
		t.Fatalf("unexpected errors: %q", res.Errors)
	}
	assertStrings(t, "pruned", res.Pruned, []string{"origin/feature-a"})
	assertStrings(t, "deleted", branchNames(res.Deleted), []string{"feature-a", "feature-b"})
	assertStrings(t, "kept", res.Kept, []string{"develop", "main"})
	assertStrings(t, "skipped", skippedNames(res.Skipped, false), []string{"feature-c", "fix"})
	// feature-a was merged into main: nothing is lost with it. The commit of
	// feature-b exists nowhere else.
	if len(res.Deleted) == 2 {
		if got := []int{res.Deleted[0].Unique, res.Deleted[1].Unique}; !slices.Equal(got, []int{0, 1}) {
			t.Errorf("unique commits = %v, want [0 1]", got)
		}
	}
	if c.diskUsage && res.Reclaimable <= 0 {
		t.Errorf("reclaimable = %d, want > 0", res.Reclaimable)
	}
}

func TestCleanDryRunChangesNothing(t *testing.T) {
	_, work := fixture(t)
	before := gitT(t, work, "for-each-ref")
	c := newTestCleaner(t, true)
	checkPlan(t, c, c.Clean(work))
	if after := gitT(t, work, "for-each-ref"); after != before {
		t.Errorf("dry run modified references:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestCleanDeletesAndPrunes(t *testing.T) {
	_, work := fixture(t)
	c := newTestCleaner(t, false)
	checkPlan(t, c, c.Clean(work))
	assertStrings(t, "remaining branches", localBranches(t, work), []string{"develop", "feature-c", "fix", "main"})
	assertStrings(t, "remaining remote-tracking refs",
		strings.Fields(gitT(t, work, "for-each-ref", "--format=%(refname:short)", "refs/remotes")),
		[]string{"origin/main"})
}

func TestCleanNoFetchLeavesRemoteRefs(t *testing.T) {
	_, work := fixture(t)
	c := newTestCleaner(t, false)
	c.opts.NoFetch = true
	res := c.Clean(work)
	if len(res.Errors)+len(res.Pruned) > 0 {
		t.Errorf("errors = %q, pruned = %q; want none", res.Errors, res.Pruned)
	}
	assertStrings(t, "deleted", branchNames(res.Deleted), []string{"feature-a", "feature-b"})
	gitT(t, work, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/feature-a")
}

func TestCleanGoesOnAfterDeletionFailure(t *testing.T) {
	_, work := fixture(t)
	gitDir := filepath.Join(work, ".git")
	if _, err := os.Stat(filepath.Join(gitDir, "reftable")); err == nil {
		t.Skip("the lock-file trick requires the 'files' reference backend")
	}
	// A leftover lock file makes "git branch -D feature-a" fail.
	if err := os.WriteFile(filepath.Join(gitDir, "refs", "heads", "feature-a.lock"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	res := newTestCleaner(t, false).Clean(work)
	assertStrings(t, "deleted", branchNames(res.Deleted), []string{"feature-b"})
	assertStrings(t, "failed", skippedNames(res.Skipped, true), []string{"feature-a"})
	assertStrings(t, "remaining branches", localBranches(t, work), []string{"develop", "feature-a", "feature-c", "fix", "main"})
}

// A GIT_DIR inherited from the environment must not redirect the cleanup to
// another repository.
func TestCleanIgnoresGitDirEnvironment(t *testing.T) {
	_, work := fixture(t)
	decoy := filepath.Join(t.TempDir(), "decoy")
	gitT(t, "", "init", "-q", decoy)
	gitT(t, decoy, "commit", "-q", "--allow-empty", "-m", "init")
	gitT(t, decoy, "branch", "precious")
	t.Setenv("GIT_DIR", filepath.Join(decoy, ".git"))

	c := newTestCleaner(t, false)
	checkPlan(t, c, c.Clean(work))
	gitT(t, decoy, "rev-parse", "--verify", "--quiet", "refs/heads/precious")
}

func TestRunEndToEnd(t *testing.T) {
	root, work := fixture(t)
	var out, errOut bytes.Buffer
	if code := run([]string{root, "--dry-run"}, &out, &errOut); code != exitOK {
		t.Fatalf("dry run: exit code %d\nstdout:\n%s\nstderr:\n%s", code, &out, &errOut)
	}
	for _, want := range []string{
		"Found 1 Git repository.",
		"would prune  origin/feature-a",
		"would delete feature-b",
		"1 unique commit",
		"skipped      fix: currently checked out (HEAD)",
		"Summary (dry run - nothing was changed)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("dry-run output lacks %q:\n%s", want, &out)
		}
	}
	if got := localBranches(t, work); len(got) != 6 {
		t.Fatalf("dry run changed the branches: %q", got)
	}

	out.Reset()
	if code := run([]string{"--gc", root}, &out, &errOut); code != exitOK {
		t.Fatalf("real run: exit code %d\nstdout:\n%s\nstderr:\n%s", code, &out, &errOut)
	}
	for _, want := range []string{"pruned       origin/feature-a", "deleted      feature-b", "Disk space freed by git gc"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, &out)
		}
	}
	assertStrings(t, "remaining branches", localBranches(t, work), []string{"develop", "feature-c", "fix", "main"})
}
