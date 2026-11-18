package git

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Info holds git state for a repository.
type Info struct {
	Branch  string
	Commit  string
	Dirty   bool
	Ahead   int
	Behind  int
	Changed int
}

// Inspect returns git state for dir. Returns empty Info if not a git repo.
func Inspect(dir string) Info {
	var info Info
	if !isRepo(dir) {
		return info
	}
	info.Branch = runGit(dir, "branch", "--show-current")
	if info.Branch == "" {
		info.Branch = "HEAD"
	}
	info.Commit = runGit(dir, "rev-parse", "--short", "HEAD")

	porcelain := runGit(dir, "status", "--porcelain")
	if porcelain != "" {
		lines := strings.Split(strings.TrimSpace(porcelain), "\n")
		count := 0
		for _, l := range lines {
			if strings.TrimSpace(l) != "" {
				count++
			}
		}
		info.Changed = count
		info.Dirty = count > 0
	}

	// ahead/behind
	abBuf := runGit(dir, "rev-list", "--left-right", "--count", "@{u}...HEAD")
	parts := strings.Fields(abBuf)
	if len(parts) == 2 {
		info.Behind, _ = strconv.Atoi(parts[0])
		info.Ahead, _ = strconv.Atoi(parts[1])
	}

	return info
}

func isRepo(dir string) bool {
	_, err := exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Output()
	return err == nil
}

func runGit(dir string, args ...string) string {
	full := append([]string{"-C", filepath.Clean(dir)}, args...)
	out, err := exec.Command("git", full...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
