package monorepo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectWorkspace_Pnpm(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pnpm-workspace-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write pnpm-workspace.yaml
	pnpmYaml := `packages:
  - 'apps/*'
  - 'packages/*'
`
	if err := os.WriteFile(filepath.Join(tempDir, "pnpm-workspace.yaml"), []byte(pnpmYaml), 0644); err != nil {
		t.Fatal(err)
	}

	// Create apps/web with package.json
	webDir := filepath.Join(tempDir, "apps", "web")
	_ = os.MkdirAll(webDir, 0755)
	webPkg := `{"name": "@repo/web", "scripts": {"dev": "vite"}}`
	if err := os.WriteFile(filepath.Join(webDir, "package.json"), []byte(webPkg), 0644); err != nil {
		t.Fatal(err)
	}

	// Create apps/docs with package.json
	docsDir := filepath.Join(tempDir, "apps", "docs")
	_ = os.MkdirAll(docsDir, 0755)
	docsPkg := `{"name": "@repo/docs", "scripts": {"dev": "next dev"}}`
	if err := os.WriteFile(filepath.Join(docsDir, "package.json"), []byte(docsPkg), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := DetectWorkspace(tempDir)
	if err != nil {
		t.Fatalf("DetectWorkspace failed: %v", err)
	}
	if !ws.IsMonorepo {
		t.Fatalf("expected IsMonorepo to be true")
	}
	if len(ws.Projects) != 2 {
		t.Fatalf("expected 2 discovered projects, got %d", len(ws.Projects))
	}
}

func TestDetectWorkspace_GoWork(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go-work-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	goWork := `go 1.22
use (
	./service-a
	./service-b
)
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.work"), []byte(goWork), 0644); err != nil {
		t.Fatal(err)
	}

	// Create service-a with main.go
	servA := filepath.Join(tempDir, "service-a")
	_ = os.MkdirAll(servA, 0755)
	_ = os.WriteFile(filepath.Join(servA, "go.mod"), []byte("module service-a\n"), 0644)
	_ = os.WriteFile(filepath.Join(servA, "main.go"), []byte("package main\nfunc main() {}\n"), 0644)

	ws, err := DetectWorkspace(tempDir)
	if err != nil {
		t.Fatalf("DetectWorkspace failed: %v", err)
	}
	if !ws.IsMonorepo {
		t.Fatalf("expected IsMonorepo to be true")
	}
	if len(ws.Projects) != 1 {
		t.Fatalf("expected 1 discovered project, got %d", len(ws.Projects))
	}
	if ws.Projects[0].Name != "service-a" {
		t.Errorf("expected project name service-a, got %s", ws.Projects[0].Name)
	}
}

func TestDetectWorkspace_Cargo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cargo-ws-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cargoToml := `[workspace]
members = [
    "crates/server",
]
`
	if err := os.WriteFile(filepath.Join(tempDir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		t.Fatal(err)
	}

	crateDir := filepath.Join(tempDir, "crates", "server")
	_ = os.MkdirAll(crateDir, 0755)
	_ = os.WriteFile(filepath.Join(crateDir, "Cargo.toml"), []byte("[package]\nname = \"server\"\n"), 0644)

	ws, err := DetectWorkspace(tempDir)
	if err != nil {
		t.Fatalf("DetectWorkspace failed: %v", err)
	}
	if !ws.IsMonorepo {
		t.Fatalf("expected IsMonorepo to be true")
	}
	if len(ws.Projects) != 1 {
		t.Fatalf("expected 1 discovered project, got %d", len(ws.Projects))
	}
}
