package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectNextJsWithPnpm(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"name":"my-next-app","scripts":{"dev":"next dev","build":"next build"},"dependencies":{"next":"^14.0.0","react":"^18.0.0"}}`
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644)
	_ = os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: 5.4"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0644)

	res, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if res.Type != "Next.js" {
		t.Errorf("expected Next.js, got %s", res.Type)
	}
	if res.PackageManager != "pnpm" {
		t.Errorf("expected pnpm, got %s", res.PackageManager)
	}
	if res.Language != "TypeScript" {
		t.Errorf("expected TypeScript, got %s", res.Language)
	}
	if res.DevCommand != "pnpm dev" {
		t.Errorf("expected 'pnpm dev', got %s", res.DevCommand)
	}
}

func TestDetectViteReactBun(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"name":"bun-vite-app","scripts":{"dev":"vite","build":"vite build"},"devDependencies":{"vite":"^5.0.0"},"dependencies":{"react":"^18.0.0"}}`
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644)
	_ = os.WriteFile(filepath.Join(dir, "bun.lock"), []byte(""), 0644)

	res, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	if res.Type != "React + Vite" {
		t.Errorf("expected React + Vite, got %s", res.Type)
	}
	if res.PackageManager != "bun" {
		t.Errorf("expected bun, got %s", res.PackageManager)
	}
	if res.DevCommand != "bun run dev" {
		t.Errorf("expected 'bun run dev', got %s", res.DevCommand)
	}
}

func TestDetectDjango(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "manage.py"), []byte("#!/usr/bin/env python"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("django>=4.0"), 0644)

	res, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	if res.Type != "Django" {
		t.Errorf("expected Django, got %s", res.Type)
	}
	if res.DevCommand != "python manage.py runserver" {
		t.Errorf("expected 'python manage.py runserver', got %s", res.DevCommand)
	}
}

func TestDetectFastAPI(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "main.py"), []byte("from fastapi import FastAPI\napp = FastAPI()"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("fastapi\nuvicorn"), 0644)

	res, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	if res.Type != "FastAPI" {
		t.Errorf("expected FastAPI, got %s", res.Type)
	}
	if res.DevCommand != "uvicorn main:app --reload" {
		t.Errorf("expected uvicorn, got %s", res.DevCommand)
	}
}

func TestDetectGoAndRust(t *testing.T) {
	dirGo := t.TempDir()
	_ = os.WriteFile(filepath.Join(dirGo, "go.mod"), []byte("module testapp\ngo 1.22\nrequire github.com/gin-gonic/gin v1.9.0\n"), 0644)

	resGo, err := Detect(dirGo)
	if err != nil {
		t.Fatal(err)
	}
	if resGo.Type != "Go (Gin)" {
		t.Errorf("expected Go (Gin), got %s", resGo.Type)
	}

	dirRust := t.TempDir()
	_ = os.WriteFile(filepath.Join(dirRust, "Cargo.toml"), []byte("[package]\nname = \"app\"\n[dependencies]\naxum = \"0.7\"\n"), 0644)

	resRust, err := Detect(dirRust)
	if err != nil {
		t.Fatal(err)
	}
	if resRust.Type != "Rust (Axum)" {
		t.Errorf("expected Rust (Axum), got %s", resRust.Type)
	}
}

func TestDetectStaticAndWasm(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Hello</h1>"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "module.wasm"), []byte("\x00asm\x01\x00\x00\x00"), 0644)

	res, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsStatic {
		t.Errorf("expected IsStatic to be true")
	}
	if !res.WasmTarget {
		t.Errorf("expected WasmTarget to be true")
	}
}
