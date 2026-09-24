package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// This file holds everything that talks to Git. git-cleaner drives the git
// executable instead of using a Go implementation of Git, so that the user's
// own setup (credential helpers, SSH configuration, proxies, hooks...) keeps
// working exactly as on the command line.

// gitVersion is a Git release number (major.minor).
type gitVersion struct{ major, minor int }

var (
	minGitVersion       = gitVersion{2, 23} // for-each-ref %(worktreepath)
	diskUsageGitVersion = gitVersion{2, 31} // rev-list --disk-usage
)

func (v gitVersion) atLeast(o gitVersion) bool {
	return v.major > o.major || (v.major == o.major && v.minor >= o.minor)
}

func (v gitVersion) String() string { return fmt.Sprintf("%d.%d", v.major, v.minor) }

// parseGitVersion extracts the version from the output of "git version",
// e.g. "git version 2.39.5 (Apple Git-154)" or "git version 2.45.1.windows.1".
func parseGitVersion(out string) (gitVersion, error) {
	if fields := strings.Fields(out); len(fields) >= 3 {
		if parts := strings.SplitN(fields[2], ".", 3); len(parts) >= 2 {
			major, err1 := strconv.Atoi(parts[0])
			minor, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil {
				return gitVersion{major, minor}, nil
			}
		}
	}
	return gitVersion{}, fmt.Errorf("cannot parse git version from %q", strings.TrimSpace(out))
}

// repoEnvVars are the environment variables that point git to another
// repository than the one of its working directory (the repository-related
// part of "git rev-parse --local-env-vars", plus GIT_NAMESPACE). They are
// removed from the environment of every git command: when git-cleaner is run
// as "git --git-dir=x cleaner" for instance, GIT_DIR would otherwise make
// every command operate on repository x.
var repoEnvVars = []string{
	"GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_COMMON_DIR", "GIT_DIR",
	"GIT_GRAFT_FILE", "GIT_IMPLICIT_WORK_TREE", "GIT_INDEX_FILE",
	"GIT_NAMESPACE", "GIT_NO_REPLACE_OBJECTS", "GIT_OBJECT_DIRECTORY",
	"GIT_PREFIX", "GIT_REPLACE_REF_BASE", "GIT_SHALLOW_FILE", "GIT_WORK_TREE",
}

// Git runs git commands. It is safe for concurrent use.
type Git struct {
	path    string     // absolute path of the git executable
	env     []string   // environment of every git process
	version gitVersion // version of the git executable
}

// NewGit locates the git executable and checks that it is recent enough.
func NewGit() (*Git, error) {
	path, err := exec.LookPath("git")
	if err != nil {
		return nil, errors.New("git not found in PATH: install Git first")
	}
	g := &Git{path: path, env: gitEnv()}
	out, err := g.exec(command{}, "version")
	if err != nil {
		return nil, fmt.Errorf("cannot run %s: %w", path, err)
	}
	if g.version, err = parseGitVersion(out); err != nil {
		return nil, err
	}
	if !g.version.atLeast(minGitVersion) {
		return nil, fmt.Errorf("git %s is too old: git-cleaner requires git %s or later", g.version, minGitVersion)
	}
	return g, nil
}

// gitEnv returns the environment given to every git command.
func gitEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if !isRepoEnvVar(name) {
			env = append(env, kv)
		}
	}
	env = append(env,
		"LC_ALL=C",              // untranslated messages: some of them are parsed
		"GIT_TERMINAL_PROMPT=0", // fail rather than wait for a password prompt
	)
	// Git Credential Manager can also open sign-in windows (browser, dialogs),
	// which GIT_TERMINAL_PROMPT does not cover: with repositories processed in
	// parallel, several could pop up at once. Disable them, unless the user
	// configured GCM_INTERACTIVE explicitly.
	if _, set := os.LookupEnv("GCM_INTERACTIVE"); !set {
		env = append(env, "GCM_INTERACTIVE=never")
	}
	return env
}

func isRepoEnvVar(name string) bool {
	for _, v := range repoEnvVars {
		if strings.EqualFold(name, v) { // names are case-insensitive on Windows
			return true
		}
	}
	return false
}

// command holds the execution settings of one git invocation.
type command struct {
	dir     string        // working directory ("" = current directory)
	stdin   string        // data sent to git's standard input
	timeout time.Duration // maximum duration (0 = unlimited)
}

// exec runs git with args and returns its standard output.
func (g *Git) exec(c command, args ...string) (string, error) {
	ctx := context.Background()
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, g.path, args...)
	cmd.Dir = c.dir
	cmd.Env = g.env
	if c.stdin != "" {
		cmd.Stdin = strings.NewReader(c.stdin)
	} // otherwise stdin is the null device: git can never wait for input
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// On timeout, first ask git to stop so that it removes its ".lock" files
	// (a plain kill would leave them behind and block the repository); it is
	// killed if still running WaitDelay later. Windows has no SIGTERM.
	cmd.Cancel = func() error {
		if runtime.GOOS == "windows" {
			return cmd.Process.Kill()
		}
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 5 * time.Second

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("timed out after %s (see --timeout)", c.timeout)
		}
		return "", &GitError{Args: args, Stderr: stderr.String(), Err: err}
	}
	return stdout.String(), nil
}

// GitError reports a git command that exited with a non-zero status.
type GitError struct {
	Args   []string // git arguments
	Stderr string   // error output of git
	Err    error    // underlying *exec.ExitError
}

// Error returns git's own diagnostic, far more useful than "exit status 128":
// the first meaningful line of its error output, without the "fatal: " or
// "error: " prefix.
func (e *GitError) Error() string {
	for _, line := range strings.Split(e.Stderr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "warning: ") || strings.HasPrefix(line, "hint: ") {
			continue
		}
		for _, prefix := range []string{"fatal: ", "error: "} {
			line = strings.TrimPrefix(line, prefix)
		}
		return line
	}
	return fmt.Sprintf("git %s: %v", strings.Join(e.Args, " "), e.Err)
}

func (e *GitError) Unwrap() error { return e.Err }

// Repo runs git commands inside one repository.
type Repo struct {
	git  *Git
	Path string // working tree root (the directory that contains .git)
}

func (r *Repo) run(args ...string) (string, error) {
	return r.git.exec(command{dir: r.Path}, args...)
}

// Ref is a Git reference as listed by "git for-each-ref".
type Ref struct {
	Name     string // full name, e.g. "refs/heads/feature/x"
	SHA      string // object ID the reference points to
	Symbolic bool   // symbolic reference, e.g. refs/remotes/origin/HEAD
	Current  bool   // local branch checked out in the main working tree
	Worktree string // path of the worktree where the branch is checked out
}

// refFormat separates the fields with NUL bytes (%00), which can appear
// neither in reference names nor in paths.
const refFormat = "%(objectname)%00%(refname)%00%(symref)%00%(HEAD)%00%(worktreepath)"

// Refs lists the references located under one of prefixes (e.g.
// "refs/remotes"), or all references when no prefix is given.
func (r *Repo) Refs(prefixes ...string) ([]Ref, error) {
	out, err := r.run(append([]string{"for-each-ref", "--format=" + refFormat}, prefixes...)...)
	if err != nil {
		return nil, err
	}
	var refs []Ref
	for _, line := range splitLines(out) {
		f := strings.Split(line, "\x00")
		if len(f) != 5 {
			continue
		}
		refs = append(refs, Ref{SHA: f[0], Name: f[1], Symbolic: f[2] != "", Current: f[3] == "*", Worktree: f[4]})
	}
	return refs, nil
}

// Remotes returns the names of the configured remotes.
func (r *Repo) Remotes() ([]string, error) {
	out, err := r.run("remote")
	return splitLines(out), err
}

// HeadCommit returns the commit HEAD points to, "" if there is none yet.
func (r *Repo) HeadCommit() string {
	out, err := r.run("rev-parse", "--verify", "--quiet", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// FetchPrune runs "git fetch --all --prune": it updates the remote-tracking
// branches of every remote and deletes those whose branch no longer exists
// on the remote. The automatic background "git gc" that fetch may start is
// disabled (through configuration keys that any Git version accepts): it
// would compete with the branch deletions that follow.
func (r *Repo) FetchPrune(timeout time.Duration) error {
	_, err := r.git.exec(command{dir: r.Path, timeout: timeout},
		"-c", "gc.auto=0", "-c", "maintenance.auto=false",
		"fetch", "--all", "--prune", "--quiet")
	return err
}

// StaleRemoteRefs returns the full names of the remote-tracking refs of
// remote that "git fetch --prune" would delete, without changing anything:
// "git remote prune --dry-run" only asks the remote for its branch list.
func (r *Repo) StaleRemoteRefs(remote string, timeout time.Duration) ([]string, error) {
	out, err := r.git.exec(command{dir: r.Path, timeout: timeout},
		"remote", "prune", "--dry-run", "--", remote)
	if err != nil {
		return nil, err
	}
	// Relevant lines look like " * [would prune] origin/feature-x" (LC_ALL=C
	// guarantees an untranslated output).
	var refs []string
	for _, line := range splitLines(out) {
		name, ok := strings.CutPrefix(strings.TrimSpace(line), "* [would prune] ")
		if !ok {
			continue
		}
		if !strings.HasPrefix(name, "refs/") {
			name = "refs/remotes/" + name // git abbreviates refs/remotes/<name>
		}
		refs = append(refs, name)
	}
	return refs, nil
}

// DeleteBranch force-deletes a local branch, even if it is not merged
// ("git branch -D"). Git refuses to delete a branch checked out in a worktree.
func (r *Repo) DeleteBranch(name string) error {
	_, err := r.run("branch", "-D", "--", name)
	return err
}

// CountUnique counts the commits reachable from tip but from none of keep.
func (r *Repo) CountUnique(tip string, keep []string) (int, error) {
	out, err := r.git.exec(command{dir: r.Path, stdin: revisions([]string{tip}, keep)},
		"rev-list", "--count", "--stdin")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(out))
}

// DiskUsage returns the size on disk, in bytes, of the objects (commits,
// trees, blobs) reachable from tips but from none of keep. --missing=allow-any
// stops partial clones from downloading the objects they lack.
func (r *Repo) DiskUsage(tips, keep []string) (int64, error) {
	out, err := r.git.exec(command{dir: r.Path, stdin: revisions(tips, keep)},
		"rev-list", "--objects", "--disk-usage", "--missing=allow-any", "--stdin")
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(out), 10, 64)
}

// GC runs "git gc": it packs the objects and deletes the unreachable ones
// that are old enough, following Git's expiry rules (reflogs are honored).
func (r *Repo) GC() error {
	_, err := r.run("gc", "--quiet")
	return err
}

// revisions builds the input of "git rev-list --stdin": one object per line,
// "^" marking the excluded ones. Passing them on stdin rather than as
// arguments avoids command-line length limits (32 KB on Windows) in
// repositories with thousands of references.
func revisions(include, exclude []string) string {
	var b strings.Builder
	for _, sha := range include {
		b.WriteString(sha + "\n")
	}
	for _, sha := range exclude {
		b.WriteString("^" + sha + "\n")
	}
	return b.String()
}

// splitLines splits command output into non-empty lines.
func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimRight(line, "\r"); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
