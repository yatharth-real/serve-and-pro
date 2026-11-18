package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"serve/pkg/proxy"
)

func createTempDirWithFiles(t *testing.T) string {
	dir := t.TempDir()

	// index.html
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Hello Serve</h1>"), 0644); err != nil {
		t.Fatal(err)
	}

	// style.css
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte("body { color: red; }"), 0644); err != nil {
		t.Fatal(err)
	}

	// data.json
	if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"status":"ok"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// .env (secret file)
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=12345"), 0644); err != nil {
		t.Fatal(err)
	}

	// app.wasm
	if err := os.WriteFile(filepath.Join(dir, "app.wasm"), []byte("\x00asm\x01\x00\x00\x00"), 0644); err != nil {
		t.Fatal(err)
	}

	// subfolder without index
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "item.txt"), []byte("sub item content"), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestDirectoryServingAndIndex(t *testing.T) {
	dir := createTempDirWithFiles(t)
	srv, err := NewServer(Config{Target: dir, Silent: true})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Hello Serve") {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestSecretProtection(t *testing.T) {
	dir := createTempDirWithFiles(t)
	srv, err := NewServer(Config{Target: dir, Silent: true})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Try fetching .env
	resp, err := http.Get(ts.URL + "/.env")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for .env, got %d", resp.StatusCode)
	}
}

func TestProxyIntegration(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"upstream":"backend-ok"}`))
	}))
	defer backend.Close()

	pm, err := proxy.NewManager(map[string]string{
		"/api": backend.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	dir := createTempDirWithFiles(t)
	srv, err := NewServer(Config{
		Target:       dir,
		Silent:       true,
		ProxyManager: pm,
	})
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/users")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from proxy, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "backend-ok") {
		t.Errorf("expected backend response, got %s", string(body))
	}
}
