package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ANSI SGR color codes used by the report.
const (
	bold   = "1"
	dim    = "2"
	red    = "31"
	green  = "32"
	yellow = "33"
	cyan   = "36"
)

// Printer renders the progress and the final report. It is only used from
// the main goroutine.
type Printer struct {
	w      io.Writer
	opts   *Options
	color  bool // emit ANSI color sequences
	banner bool // start with the ASCII-art banner (see banner.go)
}

func (p *Printer) paint(code, s string) string {
	if !p.color || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// verb picks the wording of an action: done in a real run, planned in dry-run.
func (p *Printer) verb(done, planned string) string {
	if p.opts.DryRun {
		return planned
	}
	return done
}

// Header prints the run configuration, before the scan starts.
func (p *Printer) Header() {
	mode := p.paint(red, "LIVE - local branches outside the keep list are force-deleted")
	if p.opts.DryRun {
		mode = p.paint(cyan, "DRY RUN - nothing will be changed")
	}
	fetch := "git fetch --all --prune"
	if p.opts.NoFetch {
		fetch = "skipped (--no-fetch)"
	}
	if p.banner {
		p.Banner()
	} else {
		fmt.Fprintf(p.w, "%s %s\n", p.paint(bold, "git-cleaner"), appVersion())
	}
	field := func(label, value string) { fmt.Fprintf(p.w, "  %-7s %s\n", label, value) } // "Exclude" is the longest label
	field("Mode", mode)
	field("Root", p.opts.Root)
	field("Keep", p.opts.Keep.String())
	if len(p.opts.Exclude) > 0 {
		field("Exclude", strings.Join(p.opts.Exclude, ", "))
	}
	field("Fetch", fetch)
	if p.opts.GC {
		field("GC", p.verb("git gc after cleanup (deleted branches may become unrecoverable)", "skipped in dry-run"))
	}
	fmt.Fprintln(p.w, "\nScanning...")
}

// ScanResult prints the outcome of the repository discovery. The scan is over
// before any repository is processed, so exclusions and the patterns that
// matched nothing are always reported first.
func (p *Printer) ScanResult(s *Scan) {
	for _, w := range s.Warnings {
		fmt.Fprintf(p.w, "%s %s\n", p.paint(yellow, "warning"), w)
	}
	if len(s.Excluded) > 0 {
		names := make([]string, len(s.Excluded))
		for i, dir := range s.Excluded {
			names[i] = p.displayPath(dir)
		}
		fmt.Fprintf(p.w, "%s %s\n", p.paint(cyan, "excluded"), strings.Join(names, ", "))
	}
	for _, c := range s.Covered {
		fmt.Fprintf(p.w, "%s --exclude pattern \"%s\" is already covered by the exclusion of %s\n",
			p.paint(yellow, "warning"), c.Pattern, filepath.FromSlash(c.Folder))
	}
	for _, pattern := range s.Unmatched {
		fmt.Fprintf(p.w, "%s --exclude pattern \"%s\" matched no folder (patterns are case-sensitive and relative to the root)\n",
			p.paint(yellow, "warning"), pattern)
	}
	excluded := len(s.Excluded)
	switch {
	case len(s.Repos) > 0:
		fmt.Fprintf(p.w, "Found %d Git %s.\n\n", len(s.Repos), plural(len(s.Repos), "repository", "repositories"))
	case len(p.opts.Exclude) == 0:
		fmt.Fprintln(p.w, "No Git repository found.")
	case excluded == 0:
		fmt.Fprintln(p.w, "No Git repository found (no folder excluded).")
	default:
		fmt.Fprintf(p.w, "No Git repository found outside the %d excluded %s.\n", excluded, plural(excluded, "folder", "folders"))
	}
}

// Repo prints the result of one repository; index is 1-based.
func (p *Printer) Repo(index, total int, r *RepoResult) {
	var b strings.Builder
	counter := p.paint(dim, fmt.Sprintf("[%*d/%d]", len(strconv.Itoa(total)), index, total))
	name := p.displayPath(r.Path)
	if len(r.Errors)+len(r.Pruned)+len(r.Deleted)+len(r.Skipped) == 0 && !r.GCDone {
		fmt.Fprintf(&b, "%s %s %s\n", counter, name, p.paint(dim, "- nothing to clean"))
		io.WriteString(p.w, b.String())
		return
	}

	fmt.Fprintf(&b, "%s %s\n", counter, p.paint(bold, name))
	line := func(color, label, text string) {
		// Pad before coloring: escape sequences would break the alignment.
		fmt.Fprintf(&b, "    %s %s\n", p.paint(color, fmt.Sprintf("%-12s", label)), text)
	}
	for _, msg := range r.Errors {
		line(red, "error", msg)
	}
	for _, ref := range r.Pruned {
		line(green, p.verb("pruned", "would prune"), ref)
	}
	width := 0
	for _, br := range r.Deleted {
		width = max(width, min(utf8.RuneCountInString(br.Name), 50))
	}
	for _, br := range r.Deleted {
		line(green, p.verb("deleted", "would delete"), fmt.Sprintf("%-*s  %s", width, br.Name, p.paint(dim, shortSHA(br.SHA))))
	}
	for _, s := range r.Skipped {
		color := yellow
		if s.Kind == SkipFailed {
			color = red
		}
		line(color, "skipped", s.Name+": "+s.Reason)
	}
	if len(r.Kept) > 0 {
		line(dim, "kept", strings.Join(r.Kept, ", "))
	}
	if r.GCDone {
		line(cyan, "gc", "freed "+gcOutcome(r.GCFreed))
	}
	b.WriteString("\n")
	io.WriteString(p.w, b.String())
}

// displayPath shows repositories relative to the scanned root.
func (p *Printer) displayPath(path string) string {
	if rel, err := filepath.Rel(p.opts.Root, path); err == nil && rel != "." {
		return rel
	}
	return path
}

// Summary aggregates the results of all repositories.
type Summary struct {
	Repos, ReposWithErrors int
	Excluded               int // folders skipped because of --exclude
	Pruned, Deleted, Kept  int
	CheckedOut, Unpushed   int // branches skipped, by reason
	Failed                 int // branches whose check or deletion failed
	Errors                 int
	Reclaimable            int64 // estimated bytes, summed over known estimates
	EstimateUnknown        int   // repositories without an estimate
	EstimateSupported      bool  // git is recent enough to estimate
	GCRuns                 int
	GCFreed                int64
	Elapsed                time.Duration
}

// Add accounts for the result of one repository.
func (s *Summary) Add(r *RepoResult) {
	s.Repos++
	s.Pruned += len(r.Pruned)
	s.Deleted += len(r.Deleted)
	s.Kept += len(r.Kept)
	errs := len(r.Errors)
	for _, sk := range r.Skipped {
		switch sk.Kind {
		case SkipCheckedOut:
			s.CheckedOut++
		case SkipUnpushed:
			s.Unpushed++
		default:
			s.Failed++
			errs++
		}
	}
	s.Errors += errs
	if errs > 0 {
		s.ReposWithErrors++
	}
	if r.Reclaimable >= 0 {
		s.Reclaimable += r.Reclaimable
	} else {
		s.EstimateUnknown++
	}
	if r.GCDone {
		s.GCRuns++
		s.GCFreed += r.GCFreed
	}
}

// Summary prints the final report.
func (p *Printer) Summary(s *Summary) {
	rule := p.paint(dim, strings.Repeat("-", 64))
	title := "Summary"
	if p.opts.DryRun {
		title += " (dry run - nothing was changed)"
	}
	fmt.Fprintln(p.w, rule)
	fmt.Fprintln(p.w, p.paint(bold, title))
	row := func(label, value string) { fmt.Fprintf(p.w, "  %-32s %s\n", label, value) }

	row("Repositories scanned", strconv.Itoa(s.Repos))
	if len(p.opts.Exclude) > 0 {
		row("Folders excluded", strconv.Itoa(s.Excluded))
	}
	if !p.opts.NoFetch {
		row(p.verb("Remote-tracking refs pruned", "Remote-tracking refs to prune"), strconv.Itoa(s.Pruned))
	}
	row(p.verb("Local branches deleted", "Local branches to delete"), strconv.Itoa(s.Deleted))
	row("Branches kept (whitelist)", strconv.Itoa(s.Kept))
	if s.Unpushed > 0 {
		row("Branches kept (unpushed commits)", p.paint(yellow, strconv.Itoa(s.Unpushed)))
	}
	if s.CheckedOut > 0 {
		row("Branches skipped (checked out)", strconv.Itoa(s.CheckedOut))
	}
	if s.Errors > 0 {
		row("Errors", p.paint(red, fmt.Sprintf("%d in %d %s (see above)",
			s.Errors, s.ReposWithErrors, plural(s.ReposWithErrors, "repository", "repositories"))))
	}
	switch {
	case !s.EstimateSupported:
		row("Estimated reclaimable space", "n/a (requires git "+diskUsageGitVersion.String()+" or later)")
	case s.EstimateUnknown > 0:
		row("Estimated reclaimable space", fmt.Sprintf("%s (unknown for %d %s)", formatBytes(s.Reclaimable),
			s.EstimateUnknown, plural(s.EstimateUnknown, "repository", "repositories")))
	default:
		row("Estimated reclaimable space", formatBytes(s.Reclaimable))
	}
	if s.GCRuns > 0 {
		row("Disk space freed by git gc", gcOutcome(s.GCFreed))
	}
	row("Elapsed", s.Elapsed.Round(100*time.Millisecond).String())
	fmt.Fprintln(p.w, rule)

	// Hints that help interpret the numbers above.
	switch {
	case s.Reclaimable > 0 && s.GCRuns == 0:
		fmt.Fprintln(p.w, p.paint(dim, "Deleting refs frees no disk space by itself: Git's garbage collection reclaims it once\n"+
			"reflogs no longer reference the objects (about 30 days by default). --gc runs it now."))
	case s.Reclaimable > 0:
		fmt.Fprintln(p.w, p.paint(dim, "git gc keeps the objects that reflogs still reference (about 30 days by default):\n"+
			"a later gc reclaims them. See README to reclaim them immediately."))
	}
	switch {
	case p.opts.DryRun && s.Deleted+s.Pruned > 0:
		fmt.Fprintln(p.w, p.paint(dim, "Run the same command without --dry-run to apply these changes."))
	case !p.opts.DryRun && s.Deleted > 0:
		fmt.Fprintln(p.w, p.paint(dim, "Restore a deleted branch with: git -C <repository> branch <name> <sha>"))
	}
}

// gcOutcome describes the space freed by "git gc" in .git directories.
// Small repositories may grow: gc writes a commit-graph and pack index
// files, and can only drop objects that nothing references any more.
func gcOutcome(freed int64) string {
	if freed >= 0 {
		return formatBytes(freed)
	}
	return "nothing (.git grew by " + formatBytes(-freed) + ")"
}

// shortSHA abbreviates an object ID for display.
func shortSHA(sha string) string {
	return sha[:min(len(sha), 10)]
}

// formatBytes formats a size with binary units (1 KiB = 1024 bytes).
func formatBytes(n int64) string {
	const unit = 1024
	abs := n
	if abs < 0 {
		abs = -abs
	}
	if abs < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := abs / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
