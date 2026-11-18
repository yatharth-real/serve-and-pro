package deps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDepsInspectNode(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"name":"test-deps","dependencies":{"react":"^18.0.0","next":"^14.0.0"},"devDependencies":{"typescript":"^5.0.0"}}`
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644)
	_ = os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0644)

	info, err := Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if info.PackageManager != "pnpm" {
		t.Errorf("expected pnpm, got %s", info.PackageManager)
	}
	if info.Lockfile != "pnpm-lock.yaml" {
		t.Errorf("expected pnpm-lock.yaml, got %s", info.Lockfile)
	}
	if info.DepsCount != 2 {
		t.Errorf("expected 2 dependencies, got %d", info.DepsCount)
	}
	if info.DevDepsCount != 1 {
		t.Errorf("expected 1 dev dependency, got %d", info.DevDepsCount)
	}
}
