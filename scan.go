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
// Unreadable directories do not stop the scan: they are returned as warnings.
func FindRepositories(root string) (repos, warnings []string, err error) {
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == root {
				return walkErr // nothing can be scanned at all
			}
			warnings = append(warnings, walkErr.Error())
			return nil // carry on with the rest of the tree
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		info, statErr := os.Stat(filepath.Join(path, ".git"))
		if statErr != nil {
			return nil // not a Git working tree: look deeper
		}
		if info.IsDir() {
			repos = append(repos, path)
		}
		return filepath.SkipDir
	})
	return repos, warnings, err
}
