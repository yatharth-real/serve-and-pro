package adapter

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"serve/pkg/detector"
)

// RuntimeCheck represents the status of a system toolchain or runtime.
type RuntimeCheck struct {
	Name      string
	Installed bool
	Path      string
	Version   string
}

// CheckRuntime checks whether a given runtime executable is available in PATH.
func CheckRuntime(runtimeName string) RuntimeCheck {
	binaryMap := map[string][]string{
		"node.js": {"node"},
		"node":    {"node"},
		"bun":     {"bun"},
		"deno":    {"deno"},
		"pnpm":    {"pnpm"},
		"yarn":    {"yarn"},
		"npm":     {"npm"},
		"python":  {"python", "python3", "py", "uv"},
		"go":      {"go"},
		"rust":    {"cargo", "rustc"},
		"cargo":   {"cargo"},
		"php":     {"php"},
		"ruby":    {"ruby"},
		".net":    {"dotnet"},
		"dotnet":  {"dotnet"},
		"java":    {"java"},
		"jvm":     {"java"},
		"elixir":  {"mix", "elixir"},
	}

	bins, ok := binaryMap[strings.ToLower(runtimeName)]
	if !ok {
		bins = []string{strings.ToLower(runtimeName)}
	}

	for _, bin := range bins {
		p, err := exec.LookPath(bin)
		if err == nil {
			ver := getVersion(bin)
			return RuntimeCheck{
				Name:      runtimeName,
				Installed: true,
				Path:      p,
				Version:   ver,
			}
		}
	}

	return RuntimeCheck{
		Name:      runtimeName,
		Installed: false,
	}
}

func getVersion(bin string) string {
	cmd := exec.Command(bin, "--version")
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 {
			return strings.TrimSpace(lines[0])
		}
	}
	return "available"
}

// BuildFrameworkDevCommand adapts the development command to pass port and host appropriately for the detected framework.
func BuildFrameworkDevCommand(det *detector.Result, port int, host string) (string, []string) {
	if port <= 0 {
		port = 3000
	}
	if host == "" {
		host = "127.0.0.1"
	}

	portStr := strconv.Itoa(port)
	baseCmd := det.DevCommand
	envVars := []string{
		fmt.Sprintf("PORT=%s", portStr),
		fmt.Sprintf("HOST=%s", host),
	}

	fw := strings.ToLower(det.Framework)
	typ := strings.ToLower(det.Type)

	// Framework-specific argument customization
	switch {
	case strings.Contains(fw, "vite") || strings.Contains(typ, "vite"):
		// For npm/pnpm/yarn scripts: add -- --port <port> --host <host>
		if strings.HasPrefix(baseCmd, "npm ") || strings.HasPrefix(baseCmd, "pnpm ") || strings.HasPrefix(baseCmd, "yarn ") || strings.HasPrefix(baseCmd, "bun ") {
			baseCmd += fmt.Sprintf(" -- --port %s --host %s", portStr, host)
		} else {
			baseCmd += fmt.Sprintf(" --port %s --host %s", portStr, host)
		}

	case strings.Contains(fw, "next") || strings.Contains(typ, "next"):
		if strings.HasPrefix(baseCmd, "npm ") || strings.HasPrefix(baseCmd, "pnpm ") || strings.HasPrefix(baseCmd, "yarn ") || strings.HasPrefix(baseCmd, "bun ") {
			baseCmd += fmt.Sprintf(" -- -p %s -H %s", portStr, host)
		} else {
			baseCmd += fmt.Sprintf(" -p %s -H %s", portStr, host)
		}

	case strings.Contains(fw, "nuxt") || strings.Contains(typ, "nuxt"):
		if strings.HasPrefix(baseCmd, "npm ") || strings.HasPrefix(baseCmd, "pnpm ") || strings.HasPrefix(baseCmd, "yarn ") || strings.HasPrefix(baseCmd, "bun ") {
			baseCmd += fmt.Sprintf(" -- --port %s --host %s", portStr, host)
		} else {
			baseCmd += fmt.Sprintf(" --port %s --host %s", portStr, host)
		}

	case strings.Contains(fw, "astro"):
		if strings.HasPrefix(baseCmd, "npm ") || strings.HasPrefix(baseCmd, "pnpm ") || strings.HasPrefix(baseCmd, "yarn ") || strings.HasPrefix(baseCmd, "bun ") {
			baseCmd += fmt.Sprintf(" -- --port %s --host %s", portStr, host)
		} else {
			baseCmd += fmt.Sprintf(" --port %s --host %s", portStr, host)
		}

	case strings.Contains(fw, "angular"):
		if strings.HasPrefix(baseCmd, "npm ") || strings.HasPrefix(baseCmd, "pnpm ") || strings.HasPrefix(baseCmd, "yarn ") || strings.HasPrefix(baseCmd, "bun ") {
			baseCmd += fmt.Sprintf(" -- --port %s --host %s", portStr, host)
		} else {
			baseCmd += fmt.Sprintf(" --port %s --host %s", portStr, host)
		}

	case strings.Contains(fw, "django"):
		baseCmd = fmt.Sprintf("python manage.py runserver %s:%s", host, portStr)

	case strings.Contains(fw, "fastapi"):
		baseCmd = fmt.Sprintf("uvicorn main:app --reload --port %s --host %s", portStr, host)

	case strings.Contains(fw, "flask"):
		baseCmd = fmt.Sprintf("flask run --port %s --host %s", portStr, host)
		envVars = append(envVars, fmt.Sprintf("FLASK_RUN_PORT=%s", portStr))

	case strings.Contains(fw, "laravel"):
		baseCmd = fmt.Sprintf("php artisan serve --port=%s --host=%s", portStr, host)

	case strings.Contains(typ, "php"):
		baseCmd = fmt.Sprintf("php -S %s:%s", host, portStr)

	case strings.Contains(fw, "rails"):
		baseCmd = fmt.Sprintf("bin/rails server -p %s -b %s", portStr, host)

	case strings.Contains(fw, "spring boot"):
		if strings.HasPrefix(baseCmd, "mvn") {
			baseCmd += fmt.Sprintf(" -Dspring-boot.run.arguments=--server.port=%s", portStr)
		} else {
			baseCmd += fmt.Sprintf(" --args='--server.port=%s'", portStr)
		}

	case strings.Contains(typ, "asp.net") || strings.Contains(fw, "asp.net") || strings.Contains(fw, "aspnet"):
		envVars = append(envVars, fmt.Sprintf("ASPNETCORE_URLS=http://%s:%s", host, portStr))

	case strings.Contains(fw, "rocket"):
		envVars = append(envVars, fmt.Sprintf("ROCKET_PORT=%s", portStr))
	}

	return baseCmd, envVars
}

// MissingRuntimeError formats a clean, honest error message when a required runtime is not found.
func MissingRuntimeError(d *detector.Result) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Detected:         %s\n", d.Type)
	fmt.Fprintf(&sb, "Required runtime: %s\n", d.Runtime)
	if d.PackageManager != "" && d.PackageManager != "None" {
		fmt.Fprintf(&sb, "Package manager:  %s\n", d.PackageManager)
	}
	fmt.Fprintf(&sb, "Status:           NOT FOUND in system PATH\n\n")
	fmt.Fprintf(&sb, "Install %s to run the development server.\n", d.Runtime)
	fmt.Fprintf(&sb, "Alternatively, to serve built static assets directly, run:\n")
	fmt.Fprintf(&sb, "  serve --static .\n")
	return sb.String()
}
