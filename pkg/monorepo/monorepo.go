package monorepo

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"serve/pkg/detector"
)

// SubProject represents a runnable project discovered in a monorepo.
type SubProject struct {
	Name         string           `json:"name"`
	RelativePath string           `json:"relative_path"`
	FullPath     string           `json:"full_path"`
	Detection    *detector.Result `json:"detection"`
}

// WorkspaceInfo contains monorepo discovery results.
type WorkspaceInfo struct {
	IsMonorepo bool          `json:"is_monorepo"`
	System     string        `json:"system"`
	Projects   []*SubProject `json:"projects"`
}

type packageJSON struct {
	Workspaces any `json:"workspaces"`
}

// DetectWorkspace checks if rootDir is a monorepo and locates runnable sub-projects.
func DetectWorkspace(rootDir string) (*WorkspaceInfo, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	info := &WorkspaceInfo{
		Projects: []*SubProject{},
	}

	system := ""
	var scanDirs []string

	if hasFile(absRoot, "pnpm-workspace.yaml") {
		system = "pnpm workspace"
		scanDirs = parsePnpmWorkspace(filepath.Join(absRoot, "pnpm-workspace.yaml"))
	} else if hasFile(absRoot, "turbo.json") {
		system = "Turborepo"
	} else if hasFile(absRoot, "nx.json") {
		system = "Nx"
	} else if hasFile(absRoot, "lerna.json") {
		system = "Lerna"
	} else if hasFile(absRoot, "go.work") {
		system = "Go workspace"
		scanDirs = parseGoWork(filepath.Join(absRoot, "go.work"))
	} else if isCargoWorkspace(absRoot) {
		system = "Cargo workspace"
	} else {
		// Check package.json workspaces
		pkgPath := filepath.Join(absRoot, "package.json")
		if data, err := os.ReadFile(pkgPath); err == nil {
			var p packageJSON
			if err := json.Unmarshal(data, &p); err == nil && p.Workspaces != nil {
				system = "npm/yarn workspaces"
			}
		}
	}

	if system == "" {
		return info, nil
	}

	info.IsMonorepo = true
	info.System = system

	// Common monorepo directories to search if scanDirs not specified
	if len(scanDirs) == 0 {
		scanDirs = []string{"apps/*", "packages/*", "services/*", "crates/*"}
	}

	for _, pattern := range scanDirs {
		cleanPattern := filepath.Clean(filepath.Join(absRoot, filepath.FromSlash(pattern)))
		matches, err := filepath.Glob(cleanPattern)
		if err != nil {
			continue
		}
		for _, m := range matches {
			s, err := os.Stat(m)
			if err != nil || !s.IsDir() {
				continue
			}
			// Avoid scanning heavy dirs
			base := filepath.Base(m)
			if base == "node_modules" || base == ".git" || base == "dist" || base == "target" {
				continue
			}

			det, err := detector.Detect(m)
			if err == nil && (det.DevCommand != "" || det.IsStatic) {
				rel, _ := filepath.Rel(absRoot, m)
				info.Projects = append(info.Projects, &SubProject{
					Name:         det.ProjectName,
					RelativePath: filepath.ToSlash(rel),
					FullPath:     m,
					Detection:    det,
				})
			}
		}
	}

	return info, nil
}

func hasFile(dir, file string) bool {
	_, err := os.Stat(filepath.Join(dir, file))
	return err == nil
}

func isCargoWorkspace(dir string) bool {
	p := filepath.Join(dir, "Cargo.toml")
	data, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "[workspace]")
}

func parsePnpmWorkspace(file string) []string {
	var dirs []string
	f, err := os.Open(file)
	if err != nil {
		return dirs
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inPackages := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "packages:") {
			inPackages = true
			continue
		}
		if inPackages && strings.HasPrefix(line, "-") {
			entry := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			entry = strings.Trim(entry, `"'`)
			if entry != "" {
				dirs = append(dirs, entry)
			}
		}
	}
	return dirs
}

func parseGoWork(file string) []string {
	var dirs []string
	f, err := os.Open(file)
	if err != nil {
		return dirs
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inUse := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "use (") {
			inUse = true
			continue
		}
		if inUse && line == ")" {
			inUse = false
			continue
		}
		if inUse {
			dirs = append(dirs, strings.Trim(line, `"'`))
		} else if strings.HasPrefix(line, "use ") {
			dirs = append(dirs, strings.Trim(strings.TrimPrefix(line, "use "), `"'`))
		}
	}
	return dirs
}
