package main

import (
	"io/fs"
	"os"
	"path/filepath"
)

// skipDirs lists directory names that are never scanned: they cannot hold
// the user's own repositories and may contain hundreds of thousands of files.
var skipDirs = map[string]bool{
	"node_modules": true,
}

// Scan is the outcome of the repository discovery.
type Scan struct {
	Repos     []string         // repositories, in lexical order
	Excluded  []string         // folders skipped because they match an --exclude pattern
	Unmatched []string         // --exclude patterns that matched no folder
	Covered   []CoveredPattern // --exclude patterns redundant with an excluded folder
	Warnings  []string         // folders that could not be read
}

// CoveredPattern is an --exclude pattern that matched nothing because it
// could only match inside a folder that another pattern already excluded.
type CoveredPattern struct {
	Pattern string // e.g. "acme/web"
	Folder  string // slash-separated path of the excluded folder, e.g. "acme"
}

// FindRepositories walks root recursively and returns, in lexical order,
// every directory that contains a ".git" sub-directory.
//
// A repository is not searched any further once found: clones nested in a
// working tree usually belong to other tools (vendored dependencies, package
// caches...), and skipping working trees keeps the scan fast. Directories
// whose ".git" is a file (linked worktrees, submodules) are skipped as well:
// their branches belong to the main repository, which is processed on its
// own. Symbolic links are not followed.
//
// Below the root, a folder whose slash-separated relative path matches an
// exclude pattern is recorded and skipped with its whole subtree, which is
// never read. The root itself is never excluded.
//
// Unreadable directories do not stop the scan: they are returned as warnings.
func FindRepositories(root string, exclude ExcludeList) (*Scan, error) {
	scan := &Scan{}
	used := make(map[string]bool)
	var excludedRels []string // slash-separated paths of scan.Excluded
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == root {
				return walkErr // nothing can be scanned at all
			}
			scan.Warnings = append(scan.Warnings, walkErr.Error())
			return nil // carry on with the rest of the tree
		}
		if !d.IsDir() {
			return nil
		}
		if path != root {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			if rel, err := filepath.Rel(root, path); err == nil {
				rel = filepath.ToSlash(rel)
				if patterns := exclude.Match(rel); len(patterns) > 0 {
					for _, p := range patterns {
						used[p] = true
					}
					scan.Excluded = append(scan.Excluded, path)
					excludedRels = append(excludedRels, rel)
					return filepath.SkipDir
				}
			}
		}
		info, statErr := os.Stat(filepath.Join(path, ".git"))
		if statErr != nil {
			return nil // not a Git working tree: look deeper
		}
		if info.IsDir() {
			scan.Repos = append(scan.Repos, path)
		}
		return filepath.SkipDir
	})
	for _, p := range exclude {
		switch folder, covered := coveringFolder(p, excludedRels); {
		case used[p]:
		case covered:
			scan.Covered = append(scan.Covered, CoveredPattern{Pattern: p, Folder: folder})
		default:
			scan.Unmatched = append(scan.Unmatched, p)
		}
	}
	return scan, err
}
