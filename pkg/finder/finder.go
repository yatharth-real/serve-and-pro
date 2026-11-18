package finder

import (
	"io/fs"
	"path/filepath"
	"strings"
)

var defaultIgnoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".next":        true,
	".nuxt":        true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	".pro":         true,
}

// FindOptions contains filters for file searching.
type FindOptions struct {
	Extension string
	Directory string
	MaxLimit  int
}

// SearchResult represents a matching file.
type SearchResult struct {
	RelativePath string
	FullPath     string
	IsDir        bool
}

// Search searches for files and directory names matching the query within rootPath.
func Search(rootPath, query string, opts FindOptions) ([]SearchResult, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	if opts.MaxLimit <= 0 {
		opts.MaxLimit = 100
	}

	targetExt := strings.ToLower(strings.TrimPrefix(opts.Extension, "."))
	filterDir := filepath.Clean(opts.Directory)
	if filterDir == "." {
		filterDir = ""
	}

	queryLower := strings.ToLower(query)
	var results []SearchResult

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		name := d.Name()
		if d.IsDir() {
			if defaultIgnoredDirs[name] {
				return filepath.SkipDir
			}
			if path == absRoot {
				return nil
			}
		}

		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return nil
		}

		// Directory filter
		if filterDir != "" {
			if !strings.HasPrefix(strings.ToLower(rel), strings.ToLower(filterDir)) {
				return nil
			}
		}

		// Extension filter (for files)
		if !d.IsDir() && targetExt != "" {
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
			if ext != targetExt {
				return nil
			}
		}

		// Match query in filename or path
		relNorm := filepath.ToSlash(rel)
		if queryLower == "" || strings.Contains(strings.ToLower(relNorm), queryLower) {
			results = append(results, SearchResult{
				RelativePath: relNorm,
				FullPath:     path,
				IsDir:        d.IsDir(),
			})
			if len(results) >= opts.MaxLimit {
				return filepath.SkipAll
			}
		}

		return nil
	})

	return results, err
}
