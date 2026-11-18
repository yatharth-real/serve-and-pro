package scanner

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ProjectInfo contains comprehensive local inspection data about a project.
type ProjectInfo struct {
	Path         string   `json:"path"`
	Type         string   `json:"type"`
	Ecosystem    string   `json:"ecosystem"`
	Frameworks   []string `json:"frameworks"`
	FileCount    int      `json:"file_count"`
	GitStatus    string   `json:"git_status"`
	GitBranch    string   `json:"git_branch"`
	GitChanged   int      `json:"git_changed"`
	ServerStatus string   `json:"server_status"`
	ConfigFiles  []string `json:"config_files"`
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// Scan inspects a project directory and gathers local metadata.
func Scan(dirPath string) (*ProjectInfo, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("directory does not exist: %s", absPath)
	}

	res := &ProjectInfo{
		Path: absPath,
	}

	// 1. Count files
	fileCount := 0
	_ = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" ||
				name == "dist" || name == "target" || name == "__pycache__" || name == ".venv" {
				return filepath.SkipDir
			}
			return nil
		}
		fileCount++
		return nil
	})
	res.FileCount = fileCount

	// 2. Detect Technology & Frameworks
	detectTech(absPath, res)

	// 3. Git Status
	detectGit(absPath, res)

	// 4. Server Status
	detectServer(res)

	// 5. Config files
	detectConfigFiles(absPath, res)

	return res, nil
}

func detectTech(dir string, res *ProjectInfo) {
	var frameworks []string
	ecosystem := "Unknown"
	primaryType := "Unknown"

	// Node.js ecosystems
	pkgPath := filepath.Join(dir, "package.json")
	if pkgBytes, err := os.ReadFile(pkgPath); err == nil {
		ecosystem = "Node"
		primaryType = "Node"

		var pkg packageJSON
		if err := json.Unmarshal(pkgBytes, &pkg); err == nil {
			has := func(dep string) bool {
				_, inDep := pkg.Dependencies[dep]
				_, inDev := pkg.DevDependencies[dep]
				return inDep || inDev
			}

			if has("next") {
				frameworks = append(frameworks, "Next.js")
				primaryType = "Next.js"
			} else if has("astro") {
				frameworks = append(frameworks, "Astro")
				primaryType = "Astro"
			} else if has("svelte") || has("@sveltejs/kit") {
				frameworks = append(frameworks, "Svelte")
				primaryType = "Svelte"
			} else if has("vite") {
				frameworks = append(frameworks, "Vite")
				if has("react") {
					frameworks = append(frameworks, "React")
					primaryType = "React (Vite)"
				} else if has("vue") {
					frameworks = append(frameworks, "Vue")
					primaryType = "Vue (Vite)"
				} else {
					primaryType = "Vite"
				}
			} else if has("react") {
				frameworks = append(frameworks, "React")
				primaryType = "React"
			} else if has("vue") {
				frameworks = append(frameworks, "Vue")
				primaryType = "Vue"
			} else if has("@angular/core") {
				frameworks = append(frameworks, "Angular")
				primaryType = "Angular"
			} else if has("express") {
				frameworks = append(frameworks, "Express")
				primaryType = "Express"
			} else if has("fastify") {
				frameworks = append(frameworks, "Fastify")
				primaryType = "Fastify"
			} else if has("@nestjs/core") {
				frameworks = append(frameworks, "NestJS")
				primaryType = "NestJS"
			}

			if has("tailwindcss") {
				frameworks = append(frameworks, "Tailwind CSS")
			}
			if has("typescript") {
				frameworks = append(frameworks, "TypeScript")
			}
		}
	}

	// Rust
	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
		ecosystem = "Rust"
		if primaryType == "Unknown" {
			primaryType = "Rust"
		}
		frameworks = append(frameworks, "Cargo")
	}

	// Go
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		ecosystem = "Go"
		if primaryType == "Unknown" {
			primaryType = "Go"
		}
	}

	// Python
	pyMarkers := []string{"pyproject.toml", "requirements.txt", "Pipfile", "setup.py"}
	for _, marker := range pyMarkers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			ecosystem = "Python"
			if primaryType == "Unknown" {
				primaryType = "Python"
			}
			// Look for django, flask, fastapi
			if reqBytes, err := os.ReadFile(filepath.Join(dir, marker)); err == nil {
				content := strings.ToLower(string(reqBytes))
				if strings.Contains(content, "django") {
					frameworks = append(frameworks, "Django")
					primaryType = "Django"
				} else if strings.Contains(content, "fastapi") {
					frameworks = append(frameworks, "FastAPI")
					primaryType = "FastAPI"
				} else if strings.Contains(content, "flask") {
					frameworks = append(frameworks, "Flask")
					primaryType = "Flask"
				}
			}
			break
		}
	}

	// Static HTML
	if primaryType == "Unknown" {
		if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
			ecosystem = "Web"
			primaryType = "Static HTML"
		}
	}

	res.Ecosystem = ecosystem
	res.Type = primaryType
	res.Frameworks = frameworks
}

func detectGit(dir string, res *ProjectInfo) {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		res.GitStatus = "not a git repository"
		return
	}

	// Branch
	branchCmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	branchOut, err := branchCmd.Output()
	if err == nil {
		res.GitBranch = strings.TrimSpace(string(branchOut))
	}
	if res.GitBranch == "" {
		res.GitBranch = "HEAD"
	}

	// Status
	statusCmd := exec.Command("git", "-C", dir, "status", "--porcelain")
	statusOut, err := statusCmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(statusOut)), "\n")
		count := 0
		for _, l := range lines {
			if strings.TrimSpace(l) != "" {
				count++
			}
		}
		res.GitChanged = count
		if count == 0 {
			res.GitStatus = "clean"
		} else {
			res.GitStatus = fmt.Sprintf("%d modified file(s)", count)
		}
	} else {
		res.GitStatus = "git error"
	}
}

func detectServer(res *ProjectInfo) {
	ports := []int{3000, 5173, 8080, 8000, 4321, 3001, 8081}
	for _, p := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", p), 30*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			res.ServerStatus = fmt.Sprintf("running (http://localhost:%d)", p)
			return
		}
	}
	res.ServerStatus = "stopped"
}

func detectConfigFiles(dir string, res *ProjectInfo) {
	knownConfigs := []string{
		"package.json", "tsconfig.json", "vite.config.ts", "vite.config.js",
		"next.config.js", "next.config.mjs", "next.config.ts",
		"astro.config.mjs", "svelte.config.js",
		"Cargo.toml", "go.mod", "pyproject.toml", "requirements.txt",
		"Dockerfile", "docker-compose.yml", ".env",
	}

	var found []string
	for _, cfg := range knownConfigs {
		if _, err := os.Stat(filepath.Join(dir, cfg)); err == nil {
			found = append(found, cfg)
		}
	}
	res.ConfigFiles = found
}
