package tree

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
	".idea":        true,
	".vscode":      true,
}

// TreeOptions controls the directory tree rendering.
type TreeOptions struct {
	MaxDepth int
}

// RenderTree writes an ASCII/Unicode directory tree to the provided writer.
func RenderTree(w io.Writer, rootPath string, opts TreeOptions) error {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absRoot)
	}

	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 3
	}

	ignores := loadGitIgnore(absRoot)

	fmt.Fprintln(w, filepath.Base(absRoot))
	return walkTree(w, absRoot, "", 1, opts.MaxDepth, ignores)
}

func walkTree(w io.Writer, currentDir, prefix string, currentDepth, maxDepth int, ignores map[string]bool) error {
	if currentDepth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return nil
	}

	// Filter out ignored entries
	var validEntries []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if defaultIgnoredDirs[name] {
			continue
		}
		if ignores[name] {
			continue
		}
		validEntries = append(validEntries, e)
	}

	// Sort: dirs first, then files
	sort.Slice(validEntries, func(i, j int) bool {
		if validEntries[i].IsDir() != validEntries[j].IsDir() {
			return validEntries[i].IsDir()
		}
		return strings.ToLower(validEntries[i].Name()) < strings.ToLower(validEntries[j].Name())
	})

	total := len(validEntries)
	for i, entry := range validEntries {
		isLast := i == total-1
		connector := "├── "
		nextPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			nextPrefix = prefix + "    "
		}

		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}

		fmt.Fprintf(w, "%s%s%s\n", prefix, connector, name)

		if entry.IsDir() {
			subDir := filepath.Join(currentDir, entry.Name())
			_ = walkTree(w, subDir, nextPrefix, currentDepth+1, maxDepth, ignores)
		}
	}

	return nil
}

func loadGitIgnore(root string) map[string]bool {
	ignores := make(map[string]bool)
	file, err := os.Open(filepath.Join(root, ".gitignore"))
	if err != nil {
		return ignores
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "/")
		line = strings.TrimSuffix(line, "/")
		if !strings.Contains(line, "/") && !strings.Contains(line, "*") {
			ignores[line] = true
		}
	}
	return ignores
}
