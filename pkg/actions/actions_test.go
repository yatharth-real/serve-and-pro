package actions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActionsLoadAndSave(t *testing.T) {
	dir := t.TempDir()

	// Initial empty
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.Actions) != 0 {
		t.Fatalf("expected 0 actions, got %d", len(cfg.Actions))
	}

	// Set action
	if err := SetAction(dir, "dev", "echo running dev"); err != nil {
		t.Fatalf("SetAction failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, ".pro", "actions.json")); os.IsNotExist(err) {
		t.Fatalf(".pro/actions.json was not created")
	}

	// List actions
	names, actionsMap, err := ListActions(dir)
	if err != nil {
		t.Fatalf("ListActions failed: %v", err)
	}
	if len(names) != 1 || names[0] != "dev" {
		t.Errorf("expected ['dev'], got %v", names)
	}
	if actionsMap["dev"] != "echo running dev" {
		t.Errorf("expected 'echo running dev', got %q", actionsMap["dev"])
	}
}

func TestActionsRun(t *testing.T) {
	dir := t.TempDir()
	_ = SetAction(dir, "test", "echo test-success")

	err := Run(dir, "test", nil)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}

	// Non-existent action
	err = Run(dir, "nonexistent", nil)
	if err == nil {
		t.Errorf("expected error for nonexistent action")
	}
}
