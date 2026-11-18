package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestRegistry(t *testing.T) (*Registry, string) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "projects.json")
	reg, err := Open(storePath)
	if err != nil {
		t.Fatalf("failed to open registry: %v", err)
	}
	return reg, storePath
}

func TestRegistryAddAndGet(t *testing.T) {
	reg, _ := setupTestRegistry(t)
	projDir := t.TempDir()

	// 1. Add with auto-derived name
	p, err := reg.Add(projDir, "")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if p.Name != filepath.Base(projDir) {
		t.Errorf("expected name %s, got %s", filepath.Base(projDir), p.Name)
	}

	// 2. Duplicate directory path should fail
	_, err = reg.Add(projDir, "different-name")
	if err == nil {
		t.Errorf("expected duplicate path error, got nil")
	}

	// 3. Duplicate name should fail
	anotherDir := t.TempDir()
	_, err = reg.Add(anotherDir, p.Name)
	if err == nil {
		t.Errorf("expected duplicate name error, got nil")
	}

	// 4. Non-existent path should fail
	_, err = reg.Add(filepath.Join(projDir, "does-not-exist"), "fake")
	if err == nil {
		t.Errorf("expected path not found error, got nil")
	}

	// 5. Get project
	got, err := reg.Get(p.Name)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Path != p.Path {
		t.Errorf("expected path %s, got %s", p.Path, got.Path)
	}
}

func TestRegistryRemove(t *testing.T) {
	reg, _ := setupTestRegistry(t)
	projDir := t.TempDir()
	testFile := filepath.Join(projDir, "keep-me.txt")
	_ = os.WriteFile(testFile, []byte("data"), 0644)

	p, err := reg.Add(projDir, "to-remove")
	if err != nil {
		t.Fatal(err)
	}

	// Remove from registry
	if err := reg.Remove(p.Name); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify not in registry
	_, err = reg.Get(p.Name)
	if err == nil {
		t.Errorf("expected project not found after remove")
	}

	// CRITICAL: verify the actual directory and file still exist on disk!
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("FATAL: pro rm deleted actual disk file!")
	}
}

func TestRegistryRename(t *testing.T) {
	reg, _ := setupTestRegistry(t)
	projDir := t.TempDir()

	_, err := reg.Add(projDir, "old-name")
	if err != nil {
		t.Fatal(err)
	}

	if err := reg.Rename("old-name", "new-name"); err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	if _, err := reg.Get("old-name"); err == nil {
		t.Errorf("expected old-name to no longer exist")
	}

	proj, err := reg.Get("new-name")
	if err != nil {
		t.Fatalf("expected new-name to exist: %v", err)
	}
	if proj.Name != "new-name" {
		t.Errorf("expected proj.Name to be new-name")
	}
}

func TestRegistryActiveProject(t *testing.T) {
	reg, _ := setupTestRegistry(t)
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	p1, _ := reg.Add(dir1, "app1")
	p2, _ := reg.Add(dir2, "app2")

	// First added became default active
	act, err := reg.GetActive()
	if err != nil || act.Name != p1.Name {
		t.Errorf("expected app1 to be active, got %v", act)
	}

	// Switch active
	if err := reg.SetActive(p2.Name); err != nil {
		t.Fatalf("SetActive failed: %v", err)
	}

	act, err = reg.GetActive()
	if err != nil || act.Name != p2.Name {
		t.Errorf("expected app2 to be active, got %v", act)
	}
}

func TestRegistryTagsAndGroups(t *testing.T) {
	reg, _ := setupTestRegistry(t)
	dir := t.TempDir()
	_, _ = reg.Add(dir, "myproj")

	// Add tags
	_ = reg.AddTag("myproj", "frontend")
	_ = reg.AddTag("myproj", "experimental")

	proj, _ := reg.Get("myproj")
	if len(proj.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(proj.Tags))
	}

	// Remove tag
	_ = reg.RemoveTag("myproj", "experimental")
	proj, _ = reg.Get("myproj")
	if len(proj.Tags) != 1 || proj.Tags[0] != "frontend" {
		t.Errorf("expected [frontend], got %v", proj.Tags)
	}

	// Group
	_ = reg.SetGroup("myproj", "web")
	proj, _ = reg.Get("myproj")
	if proj.Group != "web" {
		t.Errorf("expected group 'web', got %s", proj.Group)
	}
}

func TestRegistryRecent(t *testing.T) {
	reg, _ := setupTestRegistry(t)
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	_, _ = reg.Add(dir1, "proj1")
	time.Sleep(10 * time.Millisecond)
	_, _ = reg.Add(dir2, "proj2")

	recent := reg.Recent(2)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent, got %d", len(recent))
	}
	if recent[0].Name != "proj2" {
		t.Errorf("expected proj2 to be most recent, got %s", recent[0].Name)
	}

	// Touch proj1
	time.Sleep(10 * time.Millisecond)
	_, _ = reg.Get("proj1")

	recent = reg.Recent(2)
	if recent[0].Name != "proj1" {
		t.Errorf("expected proj1 to be most recent after access, got %s", recent[0].Name)
	}
}

func TestRegistryPersistence(t *testing.T) {
	reg, storePath := setupTestRegistry(t)
	dir := t.TempDir()
	_, _ = reg.Add(dir, "persist-me")
	_ = reg.AddTag("persist-me", "tag1")
	_ = reg.SetGroup("persist-me", "grp1")

	// Re-open from same file
	reg2, err := Open(storePath)
	if err != nil {
		t.Fatalf("re-open failed: %v", err)
	}

	proj, err := reg2.Get("persist-me")
	if err != nil {
		t.Fatalf("failed to find persisted project: %v", err)
	}
	if proj.Group != "grp1" || len(proj.Tags) != 1 || proj.Tags[0] != "tag1" {
		t.Errorf("persisted data mismatch: %+v", proj)
	}
}
