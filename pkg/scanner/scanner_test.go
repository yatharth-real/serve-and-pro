package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanStaticHTML(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Static</h1>"), 0644)

	info, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if info.Type != "Static HTML" {
		t.Errorf("expected 'Static HTML', got %q", info.Type)
	}
	if info.FileCount != 1 {
		t.Errorf("expected 1 file, got %d", info.FileCount)
	}
}

func TestScanNodeAndFrameworks(t *testing.T) {
	// 1. Next.js
	nextDir := t.TempDir()
	pkgNext := `{"dependencies": {"next": "^14.0.0", "react": "^18.0.0"}}`
	_ = os.WriteFile(filepath.Join(nextDir, "package.json"), []byte(pkgNext), 0644)

	infoNext, err := Scan(nextDir)
	if err != nil {
		t.Fatal(err)
	}
	if infoNext.Type != "Next.js" {
		t.Errorf("expected Next.js, got %q", infoNext.Type)
	}

	// 2. Vite + React
	viteDir := t.TempDir()
	pkgVite := `{"devDependencies": {"vite": "^5.0.0"}, "dependencies": {"react": "^18.0.0"}}`
	_ = os.WriteFile(filepath.Join(viteDir, "package.json"), []byte(pkgVite), 0644)

	infoVite, err := Scan(viteDir)
	if err != nil {
		t.Fatal(err)
	}
	if infoVite.Type != "React (Vite)" {
		t.Errorf("expected React (Vite), got %q", infoVite.Type)
	}

	// 3. Astro
	astroDir := t.TempDir()
	pkgAstro := `{"dependencies": {"astro": "^4.0.0"}}`
	_ = os.WriteFile(filepath.Join(astroDir, "package.json"), []byte(pkgAstro), 0644)

	infoAstro, err := Scan(astroDir)
	if err != nil {
		t.Fatal(err)
	}
	if infoAstro.Type != "Astro" {
		t.Errorf("expected Astro, got %q", infoAstro.Type)
	}
}

func TestScanGoAndRust(t *testing.T) {
	goDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module testapp\ngo 1.22\n"), 0644)
	infoGo, err := Scan(goDir)
	if err != nil {
		t.Fatal(err)
	}
	if infoGo.Type != "Go" {
		t.Errorf("expected Go, got %q", infoGo.Type)
	}

	rustDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(rustDir, "Cargo.toml"), []byte("[package]\nname = \"test\"\nversion = \"0.1.0\"\n"), 0644)
	infoRust, err := Scan(rustDir)
	if err != nil {
		t.Fatal(err)
	}
	if infoRust.Type != "Rust" {
		t.Errorf("expected Rust, got %q", infoRust.Type)
	}
}
