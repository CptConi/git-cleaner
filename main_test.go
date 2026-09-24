package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func assertStrings(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func TestParseArgs(t *testing.T) {
	root := t.TempDir()
	wantRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	// Flags are accepted after the positional argument too.
	opts, err := parseArgs([]string{root, "--dry-run", "--keep", " main ,release/*", "-j", "3", "--timeout=10s"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Root != wantRoot || !opts.DryRun || opts.Jobs != 3 || opts.Timeout != 10*time.Second {
		t.Errorf("unexpected options: %+v", opts)
	}
	assertStrings(t, "keep", opts.Keep, []string{"main", "release/*"})

	opts, err = parseArgs([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if opts.DryRun || opts.NoFetch || opts.GC || opts.Jobs != defaultJobs() || opts.Timeout != 2*time.Minute {
		t.Errorf("unexpected defaults: %+v", opts)
	}
	assertStrings(t, "default keep", opts.Keep, []string{"main", "master", "dev", "develop"})

	for _, args := range [][]string{
		{},
		{root, root},
		{"--jobs", "0", root},
		{"--timeout", "-1s", root},
		{"--keep", "main,[", root},
		{"--unknown", root},
		{filepath.Join(root, "missing")},
	} {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%q) succeeded, want an error", args)
		}
	}
	if _, err := parseArgs([]string{"-h"}); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("-h: got %v, want flag.ErrHelp", err)
	}
	if _, err := parseArgs([]string{"--version"}); !errors.Is(err, errVersion) {
		t.Errorf("--version: got %v, want errVersion", err)
	}
}

func TestParseInterspersed(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	dry := flags.Bool("dry-run", false, "")
	got, err := parseInterspersed(flags, []string{"a", "--dry-run", "b", "--", "--c", "d"})
	if err != nil {
		t.Fatal(err)
	}
	if !*dry {
		t.Error("--dry-run placed after a positional argument was ignored")
	}
	assertStrings(t, "positional", got, []string{"a", "b", "--c", "d"})
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	for in, want := range map[string]string{
		"~":          home,
		"~/Projects": filepath.Join(home, "Projects"),
		"~other/x":   "~other/x",
		"relative":   "relative",
	} {
		if got := expandHome(in); got != want {
			t.Errorf("expandHome(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	for n, want := range map[int64]string{
		0: "0 B", 1023: "1023 B", 1024: "1.0 KiB", 1536: "1.5 KiB",
		5 << 20: "5.0 MiB", 3 << 30: "3.0 GiB", -2048: "-2.0 KiB",
	} {
		if got := formatBytes(n); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestGitErrorMessage(t *testing.T) {
	for _, c := range []struct{ stderr, want string }{
		{"warning: redirecting to https://x\nfatal: unable to access 'x': Could not resolve host: example.com\nerror: could not fetch origin\n",
			"unable to access 'x': Could not resolve host: example.com"},
		{"git@github.com: Permission denied (publickey).\r\nfatal: Could not read from remote repository.\n",
			"git@github.com: Permission denied (publickey)."},
		{"error: cannot delete branch 'fix' used by worktree at '/w'\n",
			"cannot delete branch 'fix' used by worktree at '/w'"},
		{"", "git branch -D x: exit status 1"},
	} {
		err := &GitError{Args: []string{"branch", "-D", "x"}, Stderr: c.stderr, Err: errors.New("exit status 1")}
		if got := err.Error(); got != c.want {
			t.Errorf("GitError(%q) = %q, want %q", c.stderr, got, c.want)
		}
	}
}

func TestParseGitVersion(t *testing.T) {
	for out, want := range map[string]gitVersion{
		"git version 2.39.5 (Apple Git-154)\n": {2, 39},
		"git version 2.45.1.windows.1":         {2, 45},
		"git version 3.0.0":                    {3, 0},
	} {
		if got, err := parseGitVersion(out); err != nil || got != want {
			t.Errorf("parseGitVersion(%q) = %v, %v; want %v", out, got, err, want)
		}
	}
	if _, err := parseGitVersion("garbage"); err == nil {
		t.Error("parseGitVersion(garbage) succeeded")
	}
	if !(gitVersion{2, 31}).atLeast(diskUsageGitVersion) || (gitVersion{2, 30}).atLeast(diskUsageGitVersion) ||
		!(gitVersion{3, 0}).atLeast(diskUsageGitVersion) {
		t.Error("gitVersion.atLeast is wrong")
	}
}

func TestGitEnvDropsRepositoryVariables(t *testing.T) {
	t.Setenv("GIT_DIR", "/somewhere/else")
	t.Setenv("GIT_WORK_TREE", "/elsewhere")
	env := gitEnv()
	for _, kv := range env {
		if strings.HasPrefix(kv, "GIT_DIR=") || strings.HasPrefix(kv, "GIT_WORK_TREE=") {
			t.Errorf("%s is passed to git", kv)
		}
	}
	if !slices.Contains(env, "LC_ALL=C") || !slices.Contains(env, "GIT_TERMINAL_PROMPT=0") {
		t.Error("git environment lacks LC_ALL=C or GIT_TERMINAL_PROMPT=0")
	}
}

func TestGitEnvDisablesCredentialManagerPrompts(t *testing.T) {
	t.Setenv("GCM_INTERACTIVE", "") // restored after the test
	os.Unsetenv("GCM_INTERACTIVE")
	if env := gitEnv(); !slices.Contains(env, "GCM_INTERACTIVE=never") {
		t.Error("GCM_INTERACTIVE=never is not set by default")
	}

	t.Setenv("GCM_INTERACTIVE", "auto") // an explicit choice of the user wins
	var values []string
	for _, kv := range gitEnv() {
		if value, ok := strings.CutPrefix(kv, "GCM_INTERACTIVE="); ok {
			values = append(values, value)
		}
	}
	assertStrings(t, "GCM_INTERACTIVE values", values, []string{"auto"})
}
