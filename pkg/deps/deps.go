package deps

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"serve/pkg/detector"
)

// DependencyInfo holds inspected dependency counts and status.
type DependencyInfo struct {
	PackageManager string
	Lockfile       string
	DepsCount      int
	DevDepsCount   int
	Status         string
	Detail         string
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// Inspect analyzes the project dependencies without modifying them.
func Inspect(projectDir string) (*DependencyInfo, error) {
	det, err := detector.Detect(projectDir)
	if err != nil {
		return nil, err
	}

	info := &DependencyInfo{
		PackageManager: det.PackageManager,
		Status:         "unknown",
	}

	// 1. Node / package.json
	pkgPath := filepath.Join(projectDir, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var p packageJSON
		_ = json.Unmarshal(data, &p)
		info.DepsCount = len(p.Dependencies)
		info.DevDepsCount = len(p.DevDependencies)

		for _, lock := range []string{"pnpm-lock.yaml", "bun.lock", "bun.lockb", "yarn.lock", "package-lock.json"} {
			if _, err := os.Stat(filepath.Join(projectDir, lock)); err == nil {
				info.Lockfile = lock
				break
			}
		}

		if _, err := os.Stat(filepath.Join(projectDir, "node_modules")); err == nil {
			info.Status = "installed (node_modules present)"
		} else {
			info.Status = "missing (run install command)"
		}
		return info, nil
	}

	// 2. Cargo / Cargo.toml
	cargoPath := filepath.Join(projectDir, "Cargo.toml")
	if data, err := os.ReadFile(cargoPath); err == nil {
		lines := strings.Split(string(data), "\n")
		count := 0
		inDeps := false
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if strings.HasPrefix(trimmed, "[dependencies]") {
				inDeps = true
				continue
			}
			if strings.HasPrefix(trimmed, "[") {
				inDeps = false
			}
			if inDeps && strings.Contains(trimmed, "=") {
				count++
			}
		}
		info.DepsCount = count
		if _, err := os.Stat(filepath.Join(projectDir, "Cargo.lock")); err == nil {
			info.Lockfile = "Cargo.lock"
			info.Status = "lockfile present"
		}
		return info, nil
	}

	// 3. Go / go.mod
	goModPath := filepath.Join(projectDir, "go.mod")
	if f, err := os.Open(goModPath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		count := 0
		inRequire := false
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "require (") {
				inRequire = true
				continue
			}
			if inRequire && line == ")" {
				inRequire = false
				continue
			}
			if inRequire || strings.HasPrefix(line, "require ") {
				count++
			}
		}
		info.DepsCount = count
		if _, err := os.Stat(filepath.Join(projectDir, "go.sum")); err == nil {
			info.Lockfile = "go.sum"
			info.Status = "locked"
		}
		return info, nil
	}

	// 4. Python
	reqPath := filepath.Join(projectDir, "requirements.txt")
	if f, err := os.Open(reqPath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		count := 0
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				count++
			}
		}
		info.DepsCount = count
		info.Status = "requirements specified"
		return info, nil
	}

	info.Status = "no standard package manifest detected"
	return info, nil
}

// RunOutdated invokes the underlying package manager's outdated check if installed.
func RunOutdated(projectDir string) error {
	info, err := Inspect(projectDir)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch info.PackageManager {
	case "pnpm":
		cmd = exec.Command("pnpm", "outdated")
	case "yarn":
		cmd = exec.Command("yarn", "outdated")
	case "npm":
		cmd = exec.Command("npm", "outdated")
	case "bun":
		cmd = exec.Command("bun", "outdated")
	case "cargo":
		cmd = exec.Command("cargo", "outdated")
	default:
		return fmt.Errorf("outdated check not supported or no package manager for %q", info.PackageManager)
	}

	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
