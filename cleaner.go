package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// BranchInfo describes a local branch removed by the cleanup.
type BranchInfo struct {
	Name   string // short name, e.g. "feature/login"
	SHA    string // commit the branch pointed to (what is needed to restore it)
	Unique int    // commits found on no surviving ref (-1 = unknown)
}

// SkippedBranch is a local branch that was not deleted.
type SkippedBranch struct {
	Name   string
	Reason string
	Failed bool // git refused the deletion: counted as an error
}

// RepoResult describes what happened (or would happen, in dry-run mode) in
// one repository.
type RepoResult struct {
	Path        string
	Pruned      []string        // remote-tracking branches removed, e.g. "origin/feature-x"
	Deleted     []BranchInfo    // local branches removed
	Skipped     []SkippedBranch // local branches left in place, with the reason
	Kept        []string        // local branches protected by the whitelist
	Errors      []string        // failed operations (the cleanup went on)
	Reclaimable int64           // estimated bytes Git can free afterwards (-1 = unknown)
	GCFreed     int64           // bytes actually freed by "git gc"
	GCDone      bool            // "git gc" ran successfully
}

// Cleaner applies the cleanup to repositories. It is safe for concurrent use.
type Cleaner struct {
	git       *Git
	opts      *Options
	diskUsage bool          // git supports "rev-list --disk-usage"
	gcSlot    chan struct{} // serializes "git gc" runs
}

// NewCleaner returns a Cleaner that runs git through git with options opts.
func NewCleaner(git *Git, opts *Options) *Cleaner {
	return &Cleaner{
		git:       git,
		opts:      opts,
		diskUsage: git.version.atLeast(diskUsageGitVersion),
		// "git gc" already uses every CPU core, and a lot of memory on big
		// repositories: running several at once would only thrash the machine.
		gcSlot: make(chan struct{}, 1),
	}
}

// SafeClean is Clean, except that an unexpected panic (a bug) is reported as
// an error of that repository instead of crashing the whole run.
func (c *Cleaner) SafeClean(path string) (res *RepoResult) {
	defer func() {
		if r := recover(); r != nil {
			res = &RepoResult{Path: path, Reclaimable: -1, Errors: []string{fmt.Sprintf("internal error: %v", r)}}
		}
	}()
	return c.Clean(path)
}

// Clean processes the repository whose working tree is path:
//
//  1. prune the remote-tracking branches deleted on the remote;
//  2. sort the local branches into kept, skipped and to delete;
//  3. delete them with "git branch -D", going on after any failure;
//  4. estimate the disk space held only by the removed references;
//  5. optionally run "git gc" and measure the space it really freed.
//
// In dry-run mode, steps 1 and 3 are only simulated and step 5 is skipped.
func (c *Cleaner) Clean(path string) *RepoResult {
	res := &RepoResult{Path: path, Reclaimable: -1}
	repo := &Repo{git: c.git, Path: path}

	remotes, err := repo.Remotes()
	if err != nil {
		res.Errors = append(res.Errors, "not a usable Git repository: "+err.Error())
		return res
	}

	// Step 1, real run: fetch with --prune. The remote-tracking refs are
	// listed beforehand to find out which ones the prune removed.
	fetch := !c.opts.NoFetch && len(remotes) > 0
	var before []Ref
	if fetch && !c.opts.DryRun {
		if before, err = repo.Refs("refs/remotes"); err == nil {
			err = repo.FetchPrune(c.opts.Timeout)
		}
		if err != nil {
			// Not fatal: local branches are cleaned up all the same.
			res.Errors = append(res.Errors, "fetch --prune failed: "+err.Error())
		}
	}

	refs, err := repo.Refs()
	if err != nil {
		res.Errors = append(res.Errors, "cannot list branches: "+err.Error())
		return res
	}

	// Step 1, continued: find the pruned (or, in dry-run, prunable) refs.
	var pruned []Ref
	if fetch {
		if c.opts.DryRun {
			pruned = c.staleRemoteRefs(repo, remotes, refs, res)
		} else {
			pruned = missingRefs(before, refs)
		}
	}

	// Step 2: sort the local branches. "gone" collects every reference the
	// cleanup removes, to know which ones survive it.
	gone := make(map[string]bool)
	for _, ref := range pruned {
		gone[ref.Name] = true
		res.Pruned = append(res.Pruned, strings.TrimPrefix(ref.Name, "refs/remotes/"))
	}
	var candidates []Ref
	for _, ref := range refs {
		name, isBranch := strings.CutPrefix(ref.Name, "refs/heads/")
		if !isBranch || ref.Symbolic {
			continue
		}
		switch {
		case c.opts.Keep.Matches(name):
			res.Kept = append(res.Kept, name)
		case ref.Current:
			// Git cannot delete the branch HEAD is on: warn and go on.
			res.Skipped = append(res.Skipped, SkippedBranch{Name: name, Reason: "currently checked out (HEAD)"})
		case ref.Worktree != "":
			res.Skipped = append(res.Skipped, SkippedBranch{Name: name, Reason: "checked out in worktree " + ref.Worktree})
		default:
			candidates = append(candidates, ref)
			gone[ref.Name] = true
		}
	}
	keep := survivingObjects(refs, gone, repo.HeadCommit())

	// Step 3: delete the candidates (dry-run: only list them). "Unique"
	// commits are those that only the reflog will still reference afterwards.
	for _, ref := range candidates {
		branch := BranchInfo{Name: strings.TrimPrefix(ref.Name, "refs/heads/"), SHA: ref.SHA, Unique: -1}
		if n, err := repo.CountUnique(ref.SHA, keep); err == nil {
			branch.Unique = n
		}
		if !c.opts.DryRun {
			if err := repo.DeleteBranch(branch.Name); err != nil {
				res.Skipped = append(res.Skipped, SkippedBranch{Name: branch.Name, Reason: err.Error(), Failed: true})
				keep = append(keep, ref.SHA) // the branch survives
				continue
			}
		}
		res.Deleted = append(res.Deleted, branch)
	}

	// Step 4: deleting a reference frees (almost) nothing by itself: Git's
	// garbage collection deletes the objects later, once no reflog entry
	// references them. Estimate what it will be able to free.
	if c.diskUsage {
		var tips []string
		for _, b := range res.Deleted {
			tips = append(tips, b.SHA)
		}
		for _, ref := range pruned {
			if ref.SHA != "" {
				tips = append(tips, ref.SHA)
			}
		}
		res.Reclaimable = 0
		if len(tips) > 0 {
			if res.Reclaimable, err = repo.DiskUsage(tips, keep); err != nil {
				res.Reclaimable = -1
			}
		}
	}

	// Step 5: garbage-collect to give the space back right away, measuring
	// the size of the .git directory before and after.
	if c.opts.GC && !c.opts.DryRun {
		c.gcSlot <- struct{}{}
		gitDir := filepath.Join(path, ".git")
		sizeBefore := dirSize(gitDir)
		err := repo.GC()
		freed := sizeBefore - dirSize(gitDir)
		<-c.gcSlot
		if err != nil {
			res.Errors = append(res.Errors, "git gc failed: "+err.Error())
		} else {
			res.GCDone, res.GCFreed = true, freed
		}
	}
	return res
}

// staleRemoteRefs asks every remote which remote-tracking refs a prune would
// delete (dry-run mode) and returns them, resolved against refs.
func (c *Cleaner) staleRemoteRefs(repo *Repo, remotes []string, refs []Ref, res *RepoResult) []Ref {
	byName := make(map[string]Ref, len(refs))
	for _, ref := range refs {
		byName[ref.Name] = ref
	}
	var stale []Ref
	for _, remote := range remotes {
		names, err := repo.StaleRemoteRefs(remote, c.opts.Timeout)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("cannot query remote %q: %v", remote, err))
			continue
		}
		for _, name := range names {
			ref, ok := byName[name]
			if !ok {
				ref = Ref{Name: name} // unknown object: ignored by the estimate
			}
			stale = append(stale, ref)
		}
	}
	return stale
}

// missingRefs returns the references of before that are absent from after.
func missingRefs(before, after []Ref) []Ref {
	present := make(map[string]bool, len(after))
	for _, ref := range after {
		present[ref.Name] = true
	}
	var missing []Ref
	for _, ref := range before {
		if !present[ref.Name] && !ref.Symbolic {
			missing = append(missing, ref)
		}
	}
	return missing
}

// survivingObjects returns the distinct objects pointed to by the references
// that the cleanup leaves in place, plus HEAD (which may be detached).
func survivingObjects(refs []Ref, gone map[string]bool, head string) []string {
	seen := make(map[string]bool)
	var objects []string
	add := func(sha string) {
		if sha != "" && !seen[sha] {
			seen[sha] = true
			objects = append(objects, sha)
		}
	}
	for _, ref := range refs {
		if !gone[ref.Name] && !ref.Symbolic {
			add(ref.SHA)
		}
	}
	add(head)
	return objects
}

// dirSize returns the disk space used by the regular files under dir.
// Unreadable entries are ignored: the result is only used for reporting.
func dirSize(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				total += allocatedSize(info)
			}
		}
		return nil
	})
	return total
}
