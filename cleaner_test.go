package main

// Integration tests: they drive the real git executable on repositories
// created in temporary directories.

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
//	    feature-a  pushed, merged into main, deleted on the remote  -> deleted
//	    feature-b  never pushed                                     -> kept
//	    feature-c  checked out in the wt/ worktree                  -> skipped
//	    feature-d  pushed, still on the remote                      -> deleted
//	    feature-e  pushed, then one more local commit               -> kept
//	    feature-f  pushed, deleted on the remote without a merge    -> kept
//	    fix        checked out in work/ (HEAD)                      -> skipped
//
// origin/feature-g is also left behind by a branch that was pushed, deleted
// locally, then deleted on the remote: pruning it frees its commit.
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
	branch := func(name string, files ...string) {
		gitT(t, work, "switch", "-q", "-c", name, "main")
		for _, f := range files {
			commit(f)
		}
	}

	gitT(t, root, "init", "-q", "--bare", remote)
	gitT(t, root, "init", "-q", work)
	gitT(t, work, "symbolic-ref", "HEAD", "refs/heads/main")
	gitT(t, work, "remote", "add", "origin", remote)
	commit("a")
	gitT(t, work, "push", "-q", "origin", "main")

	branch("feature-a", "b")
	gitT(t, work, "push", "-q", "origin", "feature-a")
	gitT(t, work, "switch", "-q", "main")
	gitT(t, work, "merge", "-q", "--ff-only", "feature-a")
	gitT(t, work, "push", "-q", "origin", "main")
	gitT(t, remote, "branch", "-D", "feature-a")

	branch("feature-b", "c")

	branch("feature-d", "d")
	gitT(t, work, "push", "-q", "origin", "feature-d")

	branch("feature-e", "e1")
	gitT(t, work, "push", "-q", "origin", "feature-e")
	commit("e2")

	branch("feature-f", "f")
	gitT(t, work, "push", "-q", "origin", "feature-f")
	gitT(t, remote, "branch", "-D", "feature-f")

	branch("feature-g", "g")
	gitT(t, work, "push", "-q", "origin", "feature-g")
	gitT(t, work, "switch", "-q", "main")
	gitT(t, work, "branch", "-q", "-D", "feature-g")
	gitT(t, remote, "branch", "-D", "feature-g")

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

func skippedNames(skipped []SkippedBranch, kind SkipKind) []string {
	var names []string
	for _, s := range skipped {
		if s.Kind == kind {
			names = append(names, s.Name)
		}
	}
	return names
}

func localBranches(t *testing.T, repo string) []string {
	return strings.Fields(gitT(t, repo, "for-each-ref", "--format=%(refname:short)", "refs/heads"))
}

func remoteBranches(t *testing.T, repo string) []string {
	return strings.Fields(gitT(t, repo, "for-each-ref", "--format=%(refname:short)", "refs/remotes"))
}

// checkPlan verifies the decisions taken on the fixture, which are the same
// in dry-run and real mode.
func checkPlan(t *testing.T, c *Cleaner, res *RepoResult) {
	t.Helper()
	if len(res.Errors) > 0 {
		t.Fatalf("unexpected errors: %q", res.Errors)
	}
	assertStrings(t, "pruned", res.Pruned, []string{"origin/feature-a", "origin/feature-f", "origin/feature-g"})
	assertStrings(t, "deleted", branchNames(res.Deleted), []string{"feature-a", "feature-d"})
	assertStrings(t, "kept", res.Kept, []string{"develop", "main"})
	assertStrings(t, "kept as unpushed", skippedNames(res.Skipped, SkipUnpushed), []string{"feature-b", "feature-e", "feature-f"})
	assertStrings(t, "checked out", skippedNames(res.Skipped, SkipCheckedOut), []string{"feature-c", "fix"})
	for _, s := range res.Skipped {
		if s.Kind == SkipUnpushed && s.Reason != "1 commit not on any remote" {
			t.Errorf("%s: reason %q", s.Name, s.Reason)
		}
	}
	if c.diskUsage && res.Reclaimable <= 0 { // the commit of origin/feature-g
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

func TestCleanDeletesOnlyPushedBranches(t *testing.T) {
	_, work := fixture(t)
	c := newTestCleaner(t, false)
	checkPlan(t, c, c.Clean(work))
	assertStrings(t, "remaining branches", localBranches(t, work),
		[]string{"develop", "feature-b", "feature-c", "feature-e", "feature-f", "fix", "main"})
	assertStrings(t, "remaining remote-tracking refs", remoteBranches(t, work),
		[]string{"origin/feature-d", "origin/feature-e", "origin/main"})
}

// Without fetching, git-cleaner trusts the remote-tracking branches of the
// last fetch: origin/feature-f still exists locally, so feature-f counts as
// pushed.
func TestCleanNoFetchTrustsLastFetch(t *testing.T) {
	_, work := fixture(t)
	c := newTestCleaner(t, false)
	c.opts.NoFetch = true
	res := c.Clean(work)
	if len(res.Errors)+len(res.Pruned) > 0 {
		t.Errorf("errors = %q, pruned = %q; want none", res.Errors, res.Pruned)
	}
	assertStrings(t, "deleted", branchNames(res.Deleted), []string{"feature-a", "feature-d", "feature-f"})
	assertStrings(t, "kept as unpushed", skippedNames(res.Skipped, SkipUnpushed), []string{"feature-b", "feature-e"})
	gitT(t, work, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/feature-f")
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
	assertStrings(t, "deleted", branchNames(res.Deleted), []string{"feature-d"})
	assertStrings(t, "failed", skippedNames(res.Skipped, SkipFailed), []string{"feature-a"})
	assertStrings(t, "remaining branches", localBranches(t, work),
		[]string{"develop", "feature-a", "feature-b", "feature-c", "feature-e", "feature-f", "fix", "main"})
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
	before := localBranches(t, work)
	var out, errOut bytes.Buffer
	if code := run([]string{root, "--dry-run"}, &out, &errOut); code != exitOK {
		t.Fatalf("dry run: exit code %d\nstdout:\n%s\nstderr:\n%s", code, &out, &errOut)
	}
	for _, want := range []string{
		"Found 1 Git repository.",
		"would prune  origin/feature-a",
		"would delete feature-d",
		"skipped      feature-b: 1 commit not on any remote",
		"skipped      fix: currently checked out (HEAD)",
		"Branches kept (unpushed commits)",
		"Summary (dry run - nothing was changed)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("dry-run output lacks %q:\n%s", want, &out)
		}
	}
	assertStrings(t, "branches after the dry run", localBranches(t, work), before)

	out.Reset()
	if code := run([]string{"--gc", root}, &out, &errOut); code != exitOK {
		t.Fatalf("real run: exit code %d\nstdout:\n%s\nstderr:\n%s", code, &out, &errOut)
	}
	for _, want := range []string{"pruned       origin/feature-a", "deleted      feature-d", "Disk space freed by git gc"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, &out)
		}
	}
	assertStrings(t, "remaining branches", localBranches(t, work),
		[]string{"develop", "feature-b", "feature-c", "feature-e", "feature-f", "fix", "main"})
}

// A clone under personal/portfolio, next to the fixture, holds a branch that
// would be deleted and a remote-tracking branch that would be pruned:
// excluding it must leave it untouched, in dry-run mode as in a real run.
func TestRunWithExclusions(t *testing.T) {
	root, work := fixture(t)
	portfolio := filepath.Join(root, "personal", "portfolio")
	gitT(t, root, "clone", "-q", "-b", "main", filepath.Join(root, "remote.git"), portfolio)
	gitT(t, portfolio, "branch", "feature-x", "origin/main")             // pushed: deletable
	gitT(t, portfolio, "update-ref", "refs/remotes/origin/gone", "HEAD") // not on the remote: prunable
	fetchHead := filepath.Join(portfolio, ".git", "FETCH_HEAD")
	os.Remove(fetchHead)
	before := gitT(t, portfolio, "for-each-ref")
	shown := filepath.Join("personal", "portfolio") // the output uses OS-native separators

	var out, errOut bytes.Buffer
	runOK := func(args ...string) string {
		t.Helper()
		out.Reset()
		errOut.Reset()
		if code := run(args, &out, &errOut); code != exitOK {
			t.Fatalf("run %q: exit code %d\nstdout:\n%s\nstderr:\n%s", args, code, &out, &errOut)
		}
		return out.String()
	}
	contains := func(got string, want ...string) {
		t.Helper()
		for _, w := range want {
			if !strings.Contains(got, w) {
				t.Errorf("output lacks %q:\n%s", w, got)
			}
		}
	}

	// Dry run: reported as excluded, none of its branches mentioned.
	got := runOK("--dry-run", "--exclude", "personal/portfolio", root)
	contains(got, "Exclude personal/portfolio", "excluded "+shown, "would delete feature-d")
	if !regexp.MustCompile(`Folders excluded +1\n`).MatchString(got) {
		t.Errorf("summary lacks 1 excluded folder:\n%s", got)
	}
	if strings.Contains(got, "feature-x") || strings.Contains(got, "origin/gone") {
		t.Errorf("dry run mentions a branch of the excluded clone:\n%s", got)
	}

	// A pattern that matches nothing is reported before the first repository.
	got = runOK("--dry-run", "--exclude", "Nope", root)
	warn := strings.Index(got, `--exclude pattern "Nope" matched no folder`)
	if first := strings.Index(got, "[1/"); warn < 0 || first < warn {
		t.Errorf("no warning before the first repository:\n%s", got)
	}
	if !regexp.MustCompile(`Folders excluded +0\n`).MatchString(got) {
		t.Errorf("summary lacks 0 excluded folders:\n%s", got)
	}

	// Everything excluded: nothing is processed, the exclusions are counted.
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	folders := 0
	for _, e := range entries {
		if e.IsDir() {
			folders++
		}
	}
	got = runOK("--exclude", "*", root)
	contains(got, fmt.Sprintf("No Git repository found outside the %d excluded folders.", folders))

	// Real run: the clone keeps every reference and is not even fetched.
	got = runOK("--exclude", "personal/portfolio", root)
	contains(got, "deleted      feature-d")
	if after := gitT(t, portfolio, "for-each-ref"); after != before {
		t.Errorf("the excluded clone was modified:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if _, err := os.Stat(fetchHead); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the excluded clone was fetched (FETCH_HEAD: %v)", err)
	}
	gitT(t, work, "rev-parse", "--verify", "--quiet", "refs/heads/feature-b") // unpushed work still kept
}
