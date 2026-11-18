package tree

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTreeRendering(t *testing.T) {
	dir := t.TempDir()

	// Create structure:
	// src/index.ts
	// node_modules/foo/bar.js (should be ignored)
	// package.json
	_ = os.Mkdir(filepath.Join(dir, "src"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "src", "index.ts"), []byte("console.log(1)"), 0644)

	_ = os.MkdirAll(filepath.Join(dir, "node_modules", "foo"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "node_modules", "foo", "bar.js"), []byte("ignore"), 0644)

	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)

	var buf bytes.Buffer
	err := RenderTree(&buf, dir, TreeOptions{MaxDepth: 2})
	if err != nil {
		t.Fatalf("RenderTree failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "src/") {
		t.Errorf("expected src/ in tree, got:\n%s", out)
	}
	if !strings.Contains(out, "index.ts") {
		t.Errorf("expected index.ts in tree, got:\n%s", out)
	}
	if strings.Contains(out, "node_modules") {
		t.Errorf("node_modules should be ignored, but was in tree:\n%s", out)
	}
}
