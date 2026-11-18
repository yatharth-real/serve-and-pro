package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"serve/pkg/ui"
)

var (
	serveBinPath string
	proBinPath   string
)

func TestMain(m *testing.M) {
	// Build binaries before tests
	tempDir, err := os.MkdirTemp("", "serve-build-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	serveBinPath = filepath.Join(tempDir, "serve.exe")
	proBinPath = filepath.Join(tempDir, "pro.exe")

	cmdServe := exec.Command("go", "build", "-o", serveBinPath, "../cmd/serve")
	if out, err := cmdServe.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build serve: %v\nOutput: %s\n", err, string(out))
		os.Exit(1)
	}

	cmdPro := exec.Command("go", "build", "-o", proBinPath, "../cmd/pro")
	if out, err := cmdPro.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build pro: %v\nOutput: %s\n", err, string(out))
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func runCLI(bin string, configPath string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	if configPath != "" {
		cmd.Env = append(os.Environ(), "SERVE_CONFIG_PATH="+configPath)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestEndToEndProWorkflow(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "projects.json")

	// 1. Create a dummy project directory
	projDir := filepath.Join(t.TempDir(), "my-awesome-app")
	_ = os.MkdirAll(filepath.Join(projDir, "src"), 0755)
	_ = os.WriteFile(filepath.Join(projDir, "index.html"), []byte("<h1>Awesome</h1>"), 0644)
	_ = os.WriteFile(filepath.Join(projDir, "src", "main.ts"), []byte("console.log('main')"), 0644)
	_ = os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{"name":"app","dependencies":{"vite":"^5.0.0","react":"^18.0.0"}}`), 0644)

	// 2. pro set
	out, err := runCLI(proBinPath, cfgFile, "set", projDir, "--name", "coolapp")
	if err != nil {
		t.Fatalf("pro set failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Registered project \"coolapp\"") {
		t.Errorf("unexpected output from pro set: %s", out)
	}

	// 3. Prevent duplicate set
	out, err = runCLI(proBinPath, cfgFile, "set", projDir)
	if err == nil {
		t.Errorf("expected duplicate path error, got success: %s", out)
	}

	// 4. pro current (should be coolapp since it was first added)
	out, err = runCLI(proBinPath, cfgFile, "current")
	if err != nil {
		t.Fatalf("pro current failed: %v", err)
	}
	if !strings.Contains(out, "coolapp") {
		t.Errorf("expected coolapp as active: %s", out)
	}

	// 5. pro path
	out, err = runCLI(proBinPath, cfgFile, "path", "coolapp")
	if err != nil {
		t.Fatalf("pro path failed: %v", err)
	}
	if !strings.Contains(strings.ToLower(strings.TrimSpace(out)), strings.ToLower(projDir)) {
		t.Errorf("expected path %s, got: %s", projDir, out)
	}

	// 6. pro tags
	_, err = runCLI(proBinPath, cfgFile, "tags", "coolapp", "add", "frontend")
	if err != nil {
		t.Fatalf("pro tags add failed: %v", err)
	}
	out, err = runCLI(proBinPath, cfgFile, "tags", "coolapp")
	if err != nil {
		t.Fatalf("pro tags failed: %v", err)
	}
	if !strings.Contains(out, "frontend") {
		t.Errorf("expected tag frontend: %s", out)
	}

	// 7. pro group
	_, err = runCLI(proBinPath, cfgFile, "group", "coolapp", "web")
	if err != nil {
		t.Fatalf("pro group failed: %v", err)
	}

	// 8. pro ls
	out, err = runCLI(proBinPath, cfgFile, "ls")
	if err != nil {
		t.Fatalf("pro ls failed: %v", err)
	}
	if !strings.Contains(out, "coolapp") {
		t.Errorf("expected coolapp in pro ls: %s", out)
	}

	// 9. pro ls --group
	out, err = runCLI(proBinPath, cfgFile, "ls", "--group")
	if err != nil {
		t.Fatalf("pro ls --group failed: %v", err)
	}
	if !strings.Contains(out, "[web]") || !strings.Contains(out, "coolapp") {
		t.Errorf("expected grouped output: %s", out)
	}

	// 10. pro info
	out, err = runCLI(proBinPath, cfgFile, "info", "coolapp")
	if err != nil {
		t.Fatalf("pro info failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "React (Vite)") && !strings.Contains(out, "Vite") {
		t.Errorf("expected Vite in info, got: %s", out)
	}

	// 11. pro scan
	out, err = runCLI(proBinPath, cfgFile, "scan", "coolapp")
	if err != nil {
		t.Fatalf("pro scan failed: %v", err)
	}
	if !strings.Contains(out, "Node") {
		t.Errorf("expected Node ecosystem in scan, got: %s", out)
	}

	// 12. pro tree
	out, err = runCLI(proBinPath, cfgFile, "tree", "coolapp", "--depth", "2")
	if err != nil {
		t.Fatalf("pro tree failed: %v", err)
	}
	if !strings.Contains(out, "src/") || !strings.Contains(out, "package.json") {
		t.Errorf("unexpected tree output: %s", out)
	}

	// 13. pro find
	out, err = runCLI(proBinPath, cfgFile, "find", "coolapp", "main")
	if err != nil {
		t.Fatalf("pro find failed: %v", err)
	}
	if !strings.Contains(out, "src/main.ts") {
		t.Errorf("expected src/main.ts in find output: %s", out)
	}

	// 14. pro action add & pro action ls & pro run
	_, err = runCLI(proBinPath, cfgFile, "action", "add", "coolapp", "greet", "echo hello-from-action")
	if err != nil {
		t.Fatalf("pro action add failed: %v", err)
	}
	out, err = runCLI(proBinPath, cfgFile, "action", "ls", "coolapp")
	if err != nil {
		t.Fatalf("pro action ls failed: %v", err)
	}
	if !strings.Contains(out, "greet") {
		t.Errorf("expected greet in action ls: %s", out)
	}
	out, err = runCLI(proBinPath, cfgFile, "run", "coolapp", "greet")
	if err != nil {
		t.Fatalf("pro run failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "hello-from-action") {
		t.Errorf("expected output from run, got: %s", out)
	}

	// 15. pro rename
	out, err = runCLI(proBinPath, cfgFile, "rename", "coolapp", "renamedapp")
	if err != nil {
		t.Fatalf("pro rename failed: %v, out: %s", err, out)
	}
	out, err = runCLI(proBinPath, cfgFile, "current")
	if !strings.Contains(out, "renamedapp") {
		t.Errorf("expected renamedapp as current, got: %s", out)
	}

	// 16. pro rm
	out, err = runCLI(proBinPath, cfgFile, "rm", "renamedapp")
	if err != nil {
		t.Fatalf("pro rm failed: %v, out: %s", err, out)
	}
	// Verify directory still exists on disk
	if _, err := os.Stat(projDir); os.IsNotExist(err) {
		t.Fatalf("FATAL: pro rm deleted project directory from disk!")
	}
}

func TestEndToEndServeExecution(t *testing.T) {
	// Create test web directory
	webDir := filepath.Join(t.TempDir(), "website")
	_ = os.MkdirAll(webDir, 0755)
	_ = os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<h1>Hello World</h1>"), 0644)
	_ = os.WriteFile(filepath.Join(webDir, "style.css"), []byte("body { background: black; }"), 0644)

	port := 39182
	cmd := exec.Command(serveBinPath, webDir, "--port", fmt.Sprintf("%d", port))
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start serve: %v", err)
	}

	defer func() {
		_ = cmd.Process.Kill()
	}()

	// Wait for server to bind
	time.Sleep(300 * time.Millisecond)

	// Fetch index.html
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		t.Fatalf("failed to reach server: %v\nOutput: %s", err, outBuf.String())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Hello World") {
		t.Errorf("unexpected body: %s", string(body))
	}

	// Fetch style.css
	respCSS, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/style.css", port))
	if err != nil {
		t.Fatalf("failed to reach server: %v", err)
	}
	defer respCSS.Body.Close()
	if !strings.Contains(respCSS.Header.Get("Content-Type"), "text/css") {
		t.Errorf("expected text/css, got %s", respCSS.Header.Get("Content-Type"))
	}

	// Test Path Traversal Protection
	respBad, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/../../secret.txt", port))
	if err == nil {
		respBad.Body.Close()
		if respBad.StatusCode == http.StatusOK {
			t.Errorf("path traversal succeeded with 200 OK!")
		}
	}
}

func TestEndToEndProServeIntegration(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "projects.json")
	webDir := filepath.Join(t.TempDir(), "webproject")
	_ = os.MkdirAll(webDir, 0755)
	_ = os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<h1>Pro Serve Works</h1>"), 0644)

	// Register project
	_, err := runCLI(proBinPath, cfgFile, "set", webDir, "--name", "servetest")
	if err != nil {
		t.Fatalf("pro set failed: %v", err)
	}

	port := 39183
	cmd := exec.Command(proBinPath, "serve", "servetest", "--port", fmt.Sprintf("%d", port))
	cmd.Env = append(os.Environ(), "SERVE_CONFIG_PATH="+cfgFile)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start pro serve: %v", err)
	}

	defer func() {
		_ = cmd.Process.Kill()
	}()

	time.Sleep(300 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		t.Fatalf("failed to reach pro serve: %v\nOutput: %s", err, outBuf.String())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Pro Serve Works") {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestUniversalCommands(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "projects.json")
	dir := filepath.Join(t.TempDir(), "react-vite-app")
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test-vite","dependencies":{"react":"^18.0.0"},"devDependencies":{"vite":"^5.0.0"}}`), 0644)
	_ = os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>React Vite</h1>"), 0644)

	// 1. serve detect
	out, err := runCLI(serveBinPath, cfgFile, "detect", dir)
	if err != nil {
		t.Fatalf("serve detect failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "React + Vite") || !strings.Contains(out, "pnpm") {
		t.Errorf("unexpected detect output:\n%s", out)
	}

	// 2. serve doctor
	out, err = runCLI(serveBinPath, cfgFile, "doctor", dir)
	if err != nil {
		t.Fatalf("serve doctor failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Serve Doctor") || !strings.Contains(out, "Project Context:") {
		t.Errorf("unexpected doctor output:\n%s", out)
	}

	// 3. serve deps
	out, err = runCLI(serveBinPath, cfgFile, "deps", dir)
	if err != nil {
		t.Fatalf("serve deps failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Dependencies:") || !strings.Contains(out, "pnpm") {
		t.Errorf("unexpected deps output:\n%s", out)
	}

	// 4. serve benchmark
	out, err = runCLI(serveBinPath, cfgFile, "benchmark", dir, "-c", "2", "-n", "10")
	if err != nil {
		t.Fatalf("serve benchmark failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Serve Benchmark Results") || !strings.Contains(out, "Throughput:") {
		t.Errorf("unexpected benchmark output:\n%s", out)
	}

	// 5. pro detect
	_, err = runCLI(proBinPath, cfgFile, "set", dir, "--name", "viteapp")
	if err != nil {
		t.Fatalf("pro set failed: %v", err)
	}

	out, err = runCLI(proBinPath, cfgFile, "detect", "viteapp")
	if err != nil {
		t.Fatalf("pro detect failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "React + Vite") {
		t.Errorf("unexpected pro detect output:\n%s", out)
	}

	// 6. pro doctor
	out, err = runCLI(proBinPath, cfgFile, "doctor", "viteapp")
	if err != nil {
		t.Fatalf("pro doctor failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Serve Doctor") {
		t.Errorf("unexpected pro doctor output:\n%s", out)
	}
}

func TestFixturesDetectionAndMonorepoProjects(t *testing.T) {
	fixturesDir, err := filepath.Abs("../fixtures")
	if err != nil {
		t.Fatalf("failed to resolve fixtures dir: %v", err)
	}

	// 1. Static fixture detection
	out, err := runCLI(serveBinPath, "", "detect", filepath.Join(fixturesDir, "static"))
	if err != nil {
		t.Fatalf("detect static failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Static HTML") {
		t.Errorf("expected static detection, got:\n%s", out)
	}

	// 2. Vite fixture detection
	out, err = runCLI(serveBinPath, "", "detect", filepath.Join(fixturesDir, "vite"))
	if err != nil {
		t.Fatalf("detect vite failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Vite") {
		t.Errorf("expected Vite detection, got:\n%s", out)
	}

	// 3. Next.js fixture detection
	out, err = runCLI(serveBinPath, "", "detect", filepath.Join(fixturesDir, "next"))
	if err != nil {
		t.Fatalf("detect next failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Next.js") {
		t.Errorf("expected Next.js detection, got:\n%s", out)
	}

	// 4. FastAPI fixture detection
	out, err = runCLI(serveBinPath, "", "detect", filepath.Join(fixturesDir, "fastapi"))
	if err != nil {
		t.Fatalf("detect fastapi failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "FastAPI") {
		t.Errorf("expected FastAPI detection, got:\n%s", out)
	}

	// 5. Monorepo projects command
	out, err = runCLI(serveBinPath, "", "projects", filepath.Join(fixturesDir, "monorepo"))
	if err != nil {
		t.Fatalf("serve projects failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Monorepo System: pnpm workspace") {
		t.Errorf("expected pnpm workspace header, got:\n%s", out)
	}
	if !strings.Contains(out, "@monorepo/web") || !strings.Contains(out, "@monorepo/docs") {
		t.Errorf("expected @monorepo/web and @monorepo/docs listed, got:\n%s", out)
	}
}

func TestBannerAndThemeCLI(t *testing.T) {
	// 1. serve --version
	out, err := runCLI(serveBinPath, "", "--version")
	if err != nil {
		t.Fatalf("serve --version failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Universal Local Development & Web Serving Orchestrator") {
		t.Errorf("expected Serve descriptor in banner output:\n%s", out)
	}

	// 2. serve --help
	out, err = runCLI(serveBinPath, "", "--help")
	if err != nil {
		t.Fatalf("serve --help failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "PRIMARY COMMANDS") || !strings.Contains(out, "theme [name|list]") {
		t.Errorf("expected PRIMARY COMMANDS and theme command in help output:\n%s", out)
	}

	// 3. pro --version
	out, err = runCLI(proBinPath, "", "--version")
	if err != nil {
		t.Fatalf("pro --version failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Local Project Management & Workspace Navigation System") {
		t.Errorf("expected Pro descriptor in banner output:\n%s", out)
	}

	// 4. pro --help
	out, err = runCLI(proBinPath, "", "--help")
	if err != nil {
		t.Fatalf("pro --help failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "PROJECT MANAGEMENT") || !strings.Contains(out, "theme [name|list]") {
		t.Errorf("expected PROJECT MANAGEMENT and theme command in pro help output:\n%s", out)
	}

	// 5. serve --no-color --version
	out, err = runCLI(serveBinPath, "", "--no-color", "--version")
	if err != nil {
		t.Fatalf("serve --no-color --version failed: %v, out: %s", err, out)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI codes with --no-color flag:\n%s", out)
	}

	// 6. serve --quiet --version
	out, err = runCLI(serveBinPath, "", "--quiet", "--version")
	if err != nil {
		t.Fatalf("serve --quiet --version failed: %v, out: %s", err, out)
	}
	cleanOut := ui.StripANSI(out)
	if strings.Contains(cleanOut, "Universal Local Development") || strings.Contains(cleanOut, "theme:") {
		t.Errorf("expected quiet mode to suppress banner:\n%s", out)
	}
	if !strings.Contains(cleanOut, "serve v") {
		t.Errorf("expected version output in quiet mode:\n%s", out)
	}

	// 7. serve theme list & accent list
	out, err = runCLI(serveBinPath, "", "theme", "list")
	if err != nil {
		t.Fatalf("serve theme list failed: %v, out: %s", err, out)
	}
	cleanOut = ui.StripANSI(out)
	if !strings.Contains(cleanOut, "Available Themes") || !strings.Contains(cleanOut, "ocean") {
		t.Errorf("expected Available Themes in theme list output:\n%s", out)
	}

	out, err = runCLI(serveBinPath, "", "accent", "list")
	if err != nil {
		t.Fatalf("serve accent list failed: %v, out: %s", err, out)
	}
	cleanOut = ui.StripANSI(out)
	if !strings.Contains(cleanOut, "Available Accent Colors") || !strings.Contains(cleanOut, "green") {
		t.Errorf("expected Available Accent Colors in accent list output:\n%s", out)
	}

	// 8. Dynamic theme and accent overrides via flags
	out, err = runCLI(serveBinPath, "", "--theme", "ocean", "--accent", "green", "--version")
	if err != nil {
		t.Fatalf("serve theme override failed: %v, out: %s", err, out)
	}
	cleanOut = ui.StripANSI(out)
	if !strings.Contains(cleanOut, "theme: ocean (green)") {
		t.Errorf("expected 'theme: ocean (green)' in banner metadata:\n%s", out)
	}
}


