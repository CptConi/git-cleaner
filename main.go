// Command git-cleaner frees disk space and declutters a workstation by
// cleaning up local Git branches in bulk.
//
// It recursively scans a root directory for Git repositories and, in each
// one, prunes the remote-tracking branches whose branch no longer exists on
// the remote, then force-deletes every local branch that is not whitelisted.
// Branches on the remotes are never modified. See README.md for details.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// version is set by release builds (Makefile, GoReleaser):
//
//	go build -ldflags "-X main.version=1.2.3"
//
// When it is empty, appVersion falls back to the module version recorded by
// "go install github.com/CptConi/git-cleaner@v1.2.3".
var version string

// appVersion returns the version of this build.
func appVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}

// defaultKeep is the whitelist used when --keep is not given.
const defaultKeep = "main,master,dev,develop"

// Process exit codes.
const (
	exitOK     = 0 // everything went fine
	exitFailed = 1 // the run completed, but some operations failed
	exitUsage  = 2 // invalid arguments or unusable environment: nothing was done
)

// Options is the configuration built from the command line.
type Options struct {
	Root    string        // absolute directory to scan (symlinks resolved)
	Keep    Whitelist     // local branches that are never deleted
	Exclude ExcludeList   // folders skipped with everything below them
	DryRun  bool          // only report what would be done
	NoFetch bool          // skip the "git fetch --prune" step
	GC      bool          // run "git gc" in each repository after the cleanup
	Jobs    int           // number of repositories processed in parallel
	Timeout time.Duration // time limit of each network operation (0 = none)
	NoColor bool          // never colorize the output
}

// errVersion is returned by parseArgs when --version is requested.
var errVersion = errors.New("version requested")

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the program and returns its exit code.
func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseArgs(args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		fmt.Fprint(stdout, usage())
		return exitOK
	case errors.Is(err, errVersion):
		fmt.Fprintln(stdout, "git-cleaner", appVersion())
		return exitOK
	case err != nil:
		fmt.Fprintf(stderr, "git-cleaner: %v\nRun 'git-cleaner --help' for usage.\n", err)
		return exitUsage
	}

	git, err := NewGit()
	if err != nil {
		fmt.Fprintf(stderr, "git-cleaner: %v\n", err)
		return exitUsage
	}

	// The banner and colors are for interactive terminals: files and pipes get
	// plain text, unless FORCE_COLOR asks otherwise (e.g. "| less -R").
	forced := os.Getenv("FORCE_COLOR") != "" && os.Getenv("FORCE_COLOR") != "0"
	fancy := (forced || stdout == os.Stdout && enableTerminalColors()) && os.Getenv("TERM") != "dumb"
	out := &Printer{
		w:      stdout,
		opts:   opts,
		color:  fancy && !opts.NoColor && os.Getenv("NO_COLOR") == "",
		banner: fancy,
	}
	start := time.Now()

	// Step 1: find the repositories.
	out.Header()
	scan, err := FindRepositories(opts.Root, opts.Exclude)
	if err != nil {
		fmt.Fprintf(stderr, "git-cleaner: cannot scan %s: %v\n", opts.Root, err)
		return exitUsage
	}
	out.ScanResult(len(scan.Repos), scan.Warnings)
	repos := scan.Repos
	if len(repos) == 0 {
		return exitOK
	}

	// Step 2: clean them, printing each result as soon as it is available.
	cleaner := NewCleaner(git, opts)
	summary := Summary{EstimateSupported: cleaner.diskUsage}
	processAll(cleaner, repos, opts.Jobs, func(i int, res *RepoResult) {
		out.Repo(i+1, len(repos), res)
		summary.Add(res)
	})

	// Step 3: final report.
	summary.Elapsed = time.Since(start)
	out.Summary(&summary)
	if summary.Errors > 0 {
		return exitFailed
	}
	return exitOK
}

// processAll cleans the repositories with `jobs` parallel workers. Results
// are passed to report one at a time and in the order of repos, so the output
// stays deterministic whatever the order in which repositories complete.
func processAll(c *Cleaner, repos []string, jobs int, report func(i int, res *RepoResult)) {
	results := make([]chan *RepoResult, len(repos))
	for i := range results {
		results[i] = make(chan *RepoResult, 1) // buffered: workers never wait
	}

	queue := make(chan int)
	go func() {
		for i := range repos {
			queue <- i
		}
		close(queue)
	}()

	for range min(jobs, len(repos)) {
		go func() {
			for i := range queue {
				results[i] <- c.SafeClean(repos[i])
			}
		}()
	}

	for i := range repos {
		report(i, <-results[i])
	}
}

// parseArgs parses and validates the command line.
func parseArgs(args []string) (*Options, error) {
	flags := flag.NewFlagSet("git-cleaner", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // run() prints errors and help itself

	// When adding a flag, document it in usage() too.
	opts := &Options{}
	// --keep and --exclude may be repeated: every occurrence adds its
	// comma-separated patterns (flag.String would keep only the last one).
	var keep, exclude []string
	flags.Func("keep", "", func(v string) error { keep = append(keep, v); return nil })
	flags.Func("exclude", "", func(v string) error { exclude = append(exclude, v); return nil })
	flags.BoolVar(&opts.DryRun, "dry-run", false, "")
	flags.BoolVar(&opts.NoFetch, "no-fetch", false, "")
	flags.BoolVar(&opts.GC, "gc", false, "")
	flags.IntVar(&opts.Jobs, "jobs", defaultJobs(), "")
	flags.IntVar(&opts.Jobs, "j", defaultJobs(), "")
	flags.DurationVar(&opts.Timeout, "timeout", 2*time.Minute, "")
	flags.BoolVar(&opts.NoColor, "no-color", false, "")
	showVersion := flags.Bool("version", false, "")

	positional, err := parseInterspersed(flags, args)
	if err != nil {
		return nil, err
	}
	if *showVersion {
		return nil, errVersion
	}
	switch {
	case len(positional) == 0:
		return nil, errors.New("missing <root-directory> argument")
	case len(positional) > 1:
		return nil, fmt.Errorf("expected a single <root-directory>, got %d arguments: %s "+
			"(separate --keep and --exclude values with commas, not spaces)", len(positional), strings.Join(positional, " "))
	case opts.Jobs < 1:
		return nil, errors.New("--jobs must be at least 1")
	case opts.Timeout < 0:
		return nil, errors.New("--timeout cannot be negative")
	}
	keepList := defaultKeep // only when --keep is absent: --keep "" protects nothing
	if len(keep) > 0 {
		keepList = strings.Join(keep, ",")
	}
	if opts.Keep, err = ParseWhitelist(keepList); err != nil {
		return nil, err
	}
	if opts.Exclude, err = ParseExcludes(strings.Join(exclude, ",")); err != nil {
		return nil, err
	}
	if opts.Root, err = resolveRoot(positional[0]); err != nil {
		return nil, err
	}
	return opts, nil
}

// parseInterspersed parses flags placed anywhere on the command line: the
// flag package stops at the first positional argument, which would silently
// ignore the flag in "git-cleaner ~/Projects --dry-run". Everything after
// "--" is positional.
func parseInterspersed(flags *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := flags.Parse(args); err != nil {
			return nil, err
		}
		rest := flags.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		if consumed := len(args) - len(rest); consumed > 0 && args[consumed-1] == "--" {
			return append(positional, rest...), nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

// resolveRoot turns the <root-directory> argument into an absolute path. It
// expands a leading "~" (cmd.exe and PowerShell do not) and resolves symbolic
// links, because filepath.WalkDir does not descend into a symlinked root.
func resolveRoot(arg string) (string, error) {
	abs, err := filepath.Abs(expandHome(arg))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", fmt.Errorf("directory not found: %s", abs)
	case err != nil:
		return "", err
	case !info.IsDir():
		return "", fmt.Errorf("not a directory: %s", abs)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return abs, nil
}

// expandHome replaces a leading "~" with the home directory of the user.
func expandHome(path string) string {
	isHome := path == "~" || strings.HasPrefix(path, "~/") ||
		(runtime.GOOS == "windows" && strings.HasPrefix(path, `~\`))
	if !isHome {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// defaultJobs returns the default parallelism: one worker per CPU, capped
// at 8 to stay gentle with Git servers during the fetch step.
func defaultJobs() int {
	return max(1, min(runtime.NumCPU(), 8))
}

// usage returns the help text. Keep it in sync with the flags of parseArgs.
func usage() string {
	return fmt.Sprintf(`git-cleaner - bulk cleanup of local Git branches

Usage:
  git-cleaner [options] <root-directory>

Recursively finds the Git repositories under <root-directory> (folders that
contain a .git directory). In each one, it prunes the remote-tracking branches
whose branch was deleted on the remote (git fetch --all --prune), then
force-deletes (git branch -D) every local branch that is not in the keep list
and whose commits are all on a remote: unpushed work is never deleted.
Branches on the remotes are never modified.

Options:
  --keep <list>    Local branches never deleted: comma-separated names or glob
                   patterns (default %q).
                   Quote patterns for the shell: --keep 'main,release/*'
  --dry-run        Show what would be pruned and deleted, change nothing.
  --no-fetch       Skip the fetch/prune step (no network access): branches are
                   then checked against the state of the remotes at the last
                   fetch.
  --gc             Run "git gc" in every repository after the cleanup to free
                   disk space right away. Deleted branches may then become
                   unrecoverable.
  -j, --jobs <n>   Repositories processed in parallel (default %d).
  --timeout <d>    Time limit of each network operation, e.g. 30s or 5m;
                   0 disables it (default 2m).
  --no-color       Disable colors (the NO_COLOR variable is honored too;
                   FORCE_COLOR=1 keeps them when the output is piped).
  --version        Print the version and exit.
  -h, --help       Show this help.

Examples:
  git-cleaner --dry-run ~/Projects
  git-cleaner --keep 'main,develop,release/*' ~/Projects
  git-cleaner --no-fetch --gc ~/Projects

Exit status: 0 on success, 1 if some operations failed, 2 on invalid usage.
`, defaultKeep, defaultJobs())
}
