package finder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearch(t *testing.T) {
	dir := t.TempDir()

	_ = os.Mkdir(filepath.Join(dir, "src"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "src", "auth.ts"), []byte("auth code"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "src", "user.ts"), []byte("user code"), 0644)

	_ = os.Mkdir(filepath.Join(dir, "tests"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "tests", "auth_test.go"), []byte("test code"), 0644)

	// Search "auth"
	results, err := Search(dir, "auth", FindOptions{})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'auth', got %d", len(results))
	}

	// Filter by extension "ts"
	resultsTS, err := Search(dir, "auth", FindOptions{Extension: "ts"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resultsTS) != 1 || resultsTS[0].RelativePath != "src/auth.ts" {
		t.Errorf("expected 1 result 'src/auth.ts', got %+v", resultsTS)
	}
}
