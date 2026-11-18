package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectProfileText(t *testing.T) {
	dir := t.TempDir()
	serveDir := filepath.Join(dir, ".serve")
	_ = os.Mkdir(serveDir, 0755)

	content := `
# Serve config
runtime = Node.js
framework = Next.js
dev_command = pnpm dev
port = 4000
proxy = /api=http://localhost:8080
env.API_KEY = secret123
`
	_ = os.WriteFile(filepath.Join(serveDir, "config"), []byte(content), 0644)

	prof, ok, err := LoadProjectProfile(dir)
	if err != nil {
		t.Fatalf("LoadProjectProfile failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected profile to be found")
	}

	if prof.Runtime != "Node.js" {
		t.Errorf("expected Node.js, got %s", prof.Runtime)
	}
	if prof.Framework != "Next.js" {
		t.Errorf("expected Next.js, got %s", prof.Framework)
	}
	if prof.DevCommand != "pnpm dev" {
		t.Errorf("expected 'pnpm dev', got %s", prof.DevCommand)
	}
	if prof.Port != 4000 {
		t.Errorf("expected port 4000, got %d", prof.Port)
	}
	if prof.Proxy["/api"] != "http://localhost:8080" {
		t.Errorf("expected proxy /api -> http://localhost:8080, got %v", prof.Proxy)
	}
	if prof.Environment["API_KEY"] != "secret123" {
		t.Errorf("expected API_KEY=secret123, got %v", prof.Environment)
	}
}

func TestProjectProfileJSON(t *testing.T) {
	dir := t.TempDir()
	profIn := &ProjectProfile{
		Runtime:      "Python",
		Framework:    "FastAPI",
		DevCommand:   "uvicorn main:app --reload",
		BuildCommand: "",
		Port:         8000,
	}

	if err := SaveProjectProfile(dir, profIn); err != nil {
		t.Fatalf("SaveProjectProfile failed: %v", err)
	}

	profOut, ok, err := LoadProjectProfile(dir)
	if err != nil || !ok {
		t.Fatalf("LoadProjectProfile failed: ok=%v, err=%v", ok, err)
	}

	if profOut.Framework != "FastAPI" || profOut.Port != 8000 {
		t.Errorf("mismatched profile output: %+v", profOut)
	}
}
