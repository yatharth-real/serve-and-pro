package doctor

import (
	"fmt"
	"os"
	"strings"

	"serve/pkg/adapter"
	"serve/pkg/detector"
)

// CheckResult represents a toolchain check result.
type CheckResult struct {
	Tool      string
	Installed bool
	Version   string
	Path      string
}

// RunDiagnostics performs the environment and project diagnostics.
func RunDiagnostics(projectDir string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Serve Doctor\n\n")

	// 1. Toolchain availability
	tools := []string{
		"node", "pnpm", "bun", "yarn", "npm", "deno",
		"python", "go", "cargo", "php", "ruby", "dotnet", "java", "elixir",
	}

	fmt.Fprintf(&sb, "Runtimes & Toolchains:\n")
	foundCount := 0
	for _, t := range tools {
		chk := adapter.CheckRuntime(t)
		if chk.Installed {
			foundCount++
			fmt.Fprintf(&sb, "  %-12s [OK]        %s\n", chk.Name, chk.Version)
		}
	}
	if foundCount == 0 {
		fmt.Fprintf(&sb, "  No external runtimes detected in PATH (Native static serving only).\n")
	}

	fmt.Fprintf(&sb, "\nProject Context:\n")
	if projectDir == "" {
		projectDir = "."
	}

	det, err := detector.Detect(projectDir)
	if err != nil {
		fmt.Fprintf(&sb, "  Path: %s (Error inspecting directory: %v)\n", projectDir, err)
		return sb.String()
	}

	abs, _ := os.Getwd()
	fmt.Fprintf(&sb, "  Path:       %s\n", abs)
	fmt.Fprintf(&sb, "  Type:       %s\n", det.Type)
	fmt.Fprintf(&sb, "  Language:   %s\n", det.Language)

	if det.IsStatic {
		fmt.Fprintf(&sb, "  Runtime:    Browser / Native Serve (Zero runtime dependencies)\n")
		fmt.Fprintf(&sb, "  Status:     READY (Static serving)\n")
	} else {
		fmt.Fprintf(&sb, "  Runtime:    %s\n", det.Runtime)
		rtCheck := adapter.CheckRuntime(det.Runtime)
		if rtCheck.Installed {
			fmt.Fprintf(&sb, "  Status:     READY (Runtime verified: %s)\n", rtCheck.Version)
		} else {
			fmt.Fprintf(&sb, "  Status:     MISSING RUNTIME (%s is not in PATH)\n", det.Runtime)
			fmt.Fprintf(&sb, "  Remedy:     Install %s or run 'serve --static .'\n", det.Runtime)
		}
	}

	return sb.String()
}
