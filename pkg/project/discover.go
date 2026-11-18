package project

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"serve/pkg/detector"
	"serve/pkg/git"
	"serve/pkg/scanner"
)

// ErrProjectNotFound is returned when a project cannot be resolved.
var ErrProjectNotFound = errors.New("project not found")

// Discover runs full project intelligence for the given path.
// It checks the cache first; refreshes stale entries.
func Discover(path string) (*Project, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	cache, _ := LoadCache()
	if p, ok := cache[absPath]; ok && isFresh(p.DiscoveredAt) {
		return p, nil
	}

	// detector gives us ecosystem info
	det, err := detector.Detect(absPath)
	if err != nil {
		return nil, err
	}

	// git info via pkg/git
	gi := git.Inspect(absPath)

	// scanner for server/process detection only
	si, _ := scanner.Scan(absPath)

	// npm scripts
	scripts := readNPMScripts(filepath.Join(absPath, "package.json"))
	// makefile targets
	for k, v := range readMakeTargets(filepath.Join(absPath, "Makefile")) {
		scripts[k] = v
	}

	proc := "stopped"
	if si != nil && si.ServerStatus != "stopped" && si.ServerStatus != "" {
		proc = "running"
	}

	proj := &Project{
		Name:           filepath.Base(absPath),
		Path:           absPath,
		Runtime:        det.Runtime,
		Language:       det.Language,
		Framework:      det.Framework,
		PackageManager: det.PackageManager,
		BuildSystem:    det.BuildSystem,
		DevCommand:     det.DevCommand,
		BuildCommand:   det.BuildCommand,
		PreviewCommand: det.PreviewCommand,
		Scripts:        scripts,
		ProcessStatus:  proc,
		GitBranch:      gi.Branch,
		GitDirty:       gi.Dirty,
		GitAhead:       gi.Ahead,
		GitBehind:      gi.Behind,
		GitCommit:      gi.Commit,
		DiscoveredAt:   time.Now(),
		UpdatedAt:      time.Now(),
	}

	cache[absPath] = proj
	_ = SaveCache(cache)
	return proj, nil
}

func readNPMScripts(pkgPath string) map[string]string {
	out := make(map[string]string)
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return out
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) == nil {
		for k, v := range pkg.Scripts {
			out[k] = v
		}
	}
	return out
}

func readMakeTargets(mkPath string) map[string]string {
	out := make(map[string]string)
	f, err := os.Open(mkPath)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, ".PHONY") {
			continue
		}
		if idx := strings.Index(line, ":"); idx > 0 {
			target := strings.TrimSpace(line[:idx])
			if target != "" && !strings.Contains(target, " ") {
				out[target] = "make " + target
			}
		}
	}
	return out
}
