package detector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"serve/pkg/config"
)

// Result represents the universal detection result for a project.
type Result struct {
	ProjectName    string `json:"project_name"`
	Type           string `json:"type"`
	Language       string `json:"language"`
	Framework      string `json:"framework"`
	PackageManager string `json:"package_manager"`
	Runtime        string `json:"runtime"`
	DevServer      string `json:"dev_server"`
	Entry          string `json:"entry"`
	DevCommand     string `json:"dev_command"`
	BuildCommand   string `json:"build_command"`
	PreviewCommand string `json:"preview_command"`
	BuildSystem    string `json:"build_system"`
	Confidence     int    `json:"confidence"`
	IsStatic       bool   `json:"is_static"`
	WasmTarget     bool   `json:"wasm_target"`
	IsProfiled     bool   `json:"is_profiled"`
}

type packageJSON struct {
	Name            string            `json:"name"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// Detect analyzes the project directory and returns a comprehensive Result.
func Detect(dirPath string) (*Result, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		// Single file target
		res := &Result{
			ProjectName:    filepath.Base(absPath),
			Type:           "Single File",
			Language:       detectLanguageByExt(filepath.Ext(absPath)),
			Framework:      "None",
			PackageManager: "None",
			Runtime:        "Browser / Native",
			DevServer:      "Native Serve",
			Entry:          filepath.Base(absPath),
			Confidence:     100,
			IsStatic:       true,
		}
		if strings.HasSuffix(strings.ToLower(absPath), ".wasm") {
			res.Type = "WebAssembly Binary"
			res.WasmTarget = true
			res.Language = "WebAssembly"
		}
		return res, nil
	}

	name := filepath.Base(absPath)
	res := &Result{
		ProjectName: name,
		Confidence:  50,
	}

	// 1. Check for Node / JS / TS ecosystem
	if hasFile(absPath, "package.json") {
		detectNode(absPath, res)
	} else if hasFile(absPath, "deno.json") || hasFile(absPath, "deno.jsonc") {
		detectDeno(absPath, res)
	} else if hasFile(absPath, "Cargo.toml") {
		detectRust(absPath, res)
	} else if hasFile(absPath, "go.mod") {
		detectGo(absPath, res)
	} else if hasAnyFile(absPath, "pyproject.toml", "requirements.txt", "Pipfile", "setup.py", "manage.py") {
		detectPython(absPath, res)
	} else if hasFile(absPath, "composer.json") {
		detectPHP(absPath, res)
	} else if hasFile(absPath, "Gemfile") {
		detectRuby(absPath, res)
	} else if hasAnyFile(absPath, "pom.xml", "build.gradle", "build.gradle.kts") {
		detectJVM(absPath, res)
	} else if hasGlob(absPath, "*.csproj") || hasGlob(absPath, "*.fsproj") {
		detectDotNet(absPath, res)
	} else if hasFile(absPath, "mix.exs") {
		detectElixir(absPath, res)
	} else if hasFile(absPath, "index.html") {
		res.Type = "Static HTML"
		res.Language = "HTML / CSS / JS"
		res.Framework = "Static"
		res.PackageManager = "None"
		res.Runtime = "Browser"
		res.DevServer = "Native Serve"
		res.Entry = "index.html"
		res.Confidence = 95
		res.IsStatic = true
	} else {
		// Generic or unknown
		res.Type = "Generic Directory"
		res.Language = "Unknown"
		res.Framework = "None"
		res.PackageManager = "None"
		res.Runtime = "Browser / Native"
		res.DevServer = "Native Serve"
		res.Confidence = 40
		res.IsStatic = true
	}

	// Check for WebAssembly build artifacts or definitions
	if hasGlob(absPath, "*.wasm") || hasFile(absPath, "wasm-pack.toml") {
		res.WasmTarget = true
		if res.Type == "Static HTML" || res.Type == "Generic Directory" {
			res.Type = "WebAssembly Project"
			res.Language = "WebAssembly"
		}
	}

	// 2. Overlay .serve/config project profile overrides if present
	if prof, ok, _ := config.LoadProjectProfile(absPath); ok {
		res.IsProfiled = true
		if prof.Runtime != "" {
			res.Runtime = prof.Runtime
		}
		if prof.Framework != "" {
			res.Framework = prof.Framework
			res.Type = prof.Framework
		}
		if prof.DevCommand != "" {
			res.DevCommand = prof.DevCommand
			res.IsStatic = false
		}
		if prof.BuildCommand != "" {
			res.BuildCommand = prof.BuildCommand
		}
		if prof.PreviewCommand != "" {
			res.PreviewCommand = prof.PreviewCommand
		}
		res.Confidence = 100
	}

	return res, nil
}

func detectNode(dir string, res *Result) {
	pkgBytes, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return
	}

	var pkg packageJSON
	_ = json.Unmarshal(pkgBytes, &pkg)
	if pkg.Name != "" {
		res.ProjectName = pkg.Name
	}

	// Detect Package Manager & Lockfile
	pkgMgr := "npm"
	if hasFile(dir, "pnpm-lock.yaml") {
		pkgMgr = "pnpm"
	} else if hasAnyFile(dir, "bun.lock", "bun.lockb") {
		pkgMgr = "bun"
	} else if hasFile(dir, "yarn.lock") {
		pkgMgr = "yarn"
	} else if hasFile(dir, "package-lock.json") {
		pkgMgr = "npm"
	}

	res.PackageManager = pkgMgr
	res.Runtime = "Node.js"
	if pkgMgr == "bun" {
		res.Runtime = "Bun"
	}

	// Language detection
	isTS := hasAnyFile(dir, "tsconfig.json", "tsconfig.node.json")
	if isTS {
		res.Language = "TypeScript"
	} else {
		res.Language = "JavaScript"
	}

	res.Entry = "package.json"
	res.Confidence = 90

	hasDep := func(name string) bool {
		_, inDep := pkg.Dependencies[name]
		_, inDev := pkg.DevDependencies[name]
		return inDep || inDev
	}

	// Detect Framework
	switch {
	case hasDep("next"):
		res.Framework = "Next.js"
		res.Type = "Next.js"
		res.DevServer = "Next.js"
		res.BuildSystem = "Next CLI"
		res.Confidence = 98

	case hasDep("astro"):
		res.Framework = "Astro"
		res.Type = "Astro"
		res.DevServer = "Astro"
		res.BuildSystem = "Astro CLI"
		res.Confidence = 98

	case hasDep("@sveltejs/kit"):
		res.Framework = "SvelteKit"
		res.Type = "SvelteKit"
		res.DevServer = "Vite"
		res.BuildSystem = "Vite"
		res.Confidence = 98

	case hasDep("svelte"):
		res.Framework = "Svelte"
		res.Type = "Svelte"
		res.DevServer = "Vite"
		res.BuildSystem = "Vite"
		res.Confidence = 95

	case hasDep("nuxt"):
		res.Framework = "Nuxt"
		res.Type = "Nuxt"
		res.DevServer = "Nuxt / Vite"
		res.BuildSystem = "Nuxt CLI"
		res.Confidence = 98

	case hasDep("@remix-run/react") || hasDep("@remix-run/node"):
		res.Framework = "Remix"
		res.Type = "Remix"
		res.DevServer = "Remix"
		res.BuildSystem = "Vite"
		res.Confidence = 98

	case hasDep("@tanstack/react-start") || hasDep("@tanstack/start"):
		res.Framework = "TanStack Start"
		res.Type = "TanStack Start"
		res.DevServer = "Vite"
		res.BuildSystem = "Vite"
		res.Confidence = 98

	case hasDep("gatsby"):
		res.Framework = "Gatsby"
		res.Type = "Gatsby"
		res.DevServer = "Gatsby"
		res.BuildSystem = "Gatsby CLI"
		res.Confidence = 98

	case hasDep("vite"):
		res.DevServer = "Vite"
		res.BuildSystem = "Vite"
		if hasDep("react") {
			res.Framework = "React"
			res.Type = "React + Vite"
		} else if hasDep("vue") {
			res.Framework = "Vue"
			res.Type = "Vue + Vite"
		} else if hasDep("solid-js") {
			res.Framework = "SolidJS"
			res.Type = "SolidJS + Vite"
		} else if hasDep("preact") {
			res.Framework = "Preact"
			res.Type = "Preact + Vite"
		} else if hasDep("@builder.io/qwik") {
			res.Framework = "Qwik"
			res.Type = "Qwik + Vite"
		} else {
			res.Framework = "Vite"
			res.Type = "Vite Project"
		}
		res.Confidence = 96

	case hasDep("react"):
		res.Framework = "React"
		res.Type = "React"
		res.DevServer = "React Scripts"
		res.Confidence = 92

	case hasDep("vue"):
		res.Framework = "Vue"
		res.Type = "Vue"
		res.DevServer = "Vue CLI"
		res.Confidence = 92

	case hasDep("@angular/core") || hasFile(dir, "angular.json"):
		res.Framework = "Angular"
		res.Type = "Angular"
		res.DevServer = "Angular CLI"
		res.BuildSystem = "Angular CLI"
		res.Confidence = 98

	case hasDep("@analogjs/router"):
		res.Framework = "Analog"
		res.Type = "Analog (Angular SSR)"
		res.DevServer = "Vite"
		res.Confidence = 98

	case hasDep("lit"):
		res.Framework = "Lit"
		res.Type = "Lit Component Project"
		res.DevServer = "Vite / WDS"
		res.Confidence = 90

	case hasDep("express"):
		res.Framework = "Express"
		res.Type = "Node (Express)"
		res.DevServer = "Node / Nodemon"
		res.Confidence = 92

	case hasDep("fastify"):
		res.Framework = "Fastify"
		res.Type = "Node (Fastify)"
		res.DevServer = "Fastify CLI / Node"
		res.Confidence = 92

	case hasDep("@nestjs/core"):
		res.Framework = "NestJS"
		res.Type = "NestJS"
		res.DevServer = "Nest CLI"
		res.Confidence = 98

	default:
		res.Framework = "Node.js"
		res.Type = "Node Application"
		res.DevServer = "Node"
		res.Confidence = 85
	}

	// Commands derivation from package.json scripts
	runCmd := func(sub string) string {
		switch pkgMgr {
		case "pnpm":
			return "pnpm " + sub
		case "yarn":
			return "yarn " + sub
		case "bun":
			return "bun run " + sub
		default:
			return "npm run " + sub
		}
	}

	if _, ok := pkg.Scripts["dev"]; ok {
		res.DevCommand = runCmd("dev")
	} else if _, ok := pkg.Scripts["start"]; ok {
		res.DevCommand = runCmd("start")
	} else if _, ok := pkg.Scripts["serve"]; ok {
		res.DevCommand = runCmd("serve")
	}

	if _, ok := pkg.Scripts["build"]; ok {
		res.BuildCommand = runCmd("build")
	}

	if _, ok := pkg.Scripts["preview"]; ok {
		res.PreviewCommand = runCmd("preview")
	} else if _, ok := pkg.Scripts["start"]; ok && res.DevCommand != runCmd("start") {
		res.PreviewCommand = runCmd("start")
	}
}

func detectDeno(dir string, res *Result) {
	res.Language = "TypeScript"
	res.Runtime = "Deno"
	res.PackageManager = "Deno"
	res.DevServer = "Deno"
	res.Type = "Deno Application"
	res.Entry = "deno.json"
	res.DevCommand = "deno task dev"
	res.Confidence = 95
}

func detectRust(dir string, res *Result) {
	res.Language = "Rust"
	res.Runtime = "Rust"
	res.PackageManager = "Cargo"
	res.BuildSystem = "Cargo"
	res.Entry = "Cargo.toml"
	res.Confidence = 95

	cargoBytes, _ := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	cargoStr := string(cargoBytes)

	switch {
	case strings.Contains(cargoStr, "axum"):
		res.Framework = "Axum"
		res.Type = "Rust (Axum)"
	case strings.Contains(cargoStr, "actix-web"):
		res.Framework = "Actix Web"
		res.Type = "Rust (Actix Web)"
	case strings.Contains(cargoStr, "rocket"):
		res.Framework = "Rocket"
		res.Type = "Rust (Rocket)"
	case strings.Contains(cargoStr, "wasm-bindgen"):
		res.Framework = "Wasm-Bindgen"
		res.Type = "Rust (WebAssembly)"
		res.WasmTarget = true
	default:
		res.Framework = "Cargo"
		res.Type = "Rust Application"
	}

	res.DevCommand = "cargo run"
	res.BuildCommand = "cargo build --release"
	res.DevServer = "Cargo"
}

func detectGo(dir string, res *Result) {
	res.Language = "Go"
	res.Runtime = "Go"
	res.PackageManager = "Go Modules"
	res.BuildSystem = "go build"
	res.Entry = "go.mod"
	res.Confidence = 95

	modBytes, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	modStr := string(modBytes)

	switch {
	case strings.Contains(modStr, "github.com/gin-gonic/gin"):
		res.Framework = "Gin"
		res.Type = "Go (Gin)"
	case strings.Contains(modStr, "github.com/labstack/echo"):
		res.Framework = "Echo"
		res.Type = "Go (Echo)"
	case strings.Contains(modStr, "github.com/gofiber/fiber"):
		res.Framework = "Fiber"
		res.Type = "Go (Fiber)"
	default:
		res.Framework = "Go standard library"
		res.Type = "Go Application"
	}

	res.DevCommand = "go run ."
	res.BuildCommand = "go build ."
	res.DevServer = "Go"
}

func detectPython(dir string, res *Result) {
	res.Language = "Python"
	res.Runtime = "Python"
	res.BuildSystem = "pip"
	res.PackageManager = "pip"

	if hasFile(dir, "poetry.lock") || hasFile(dir, "pyproject.toml") {
		res.PackageManager = "poetry"
	} else if hasFile(dir, "Pipfile") || hasFile(dir, "Pipfile.lock") {
		res.PackageManager = "pipenv"
	}

	// Search for framework indicators
	reqText := ""
	for _, f := range []string{"requirements.txt", "pyproject.toml", "Pipfile"} {
		if b, err := os.ReadFile(filepath.Join(dir, f)); err == nil {
			reqText += " " + strings.ToLower(string(b))
		}
	}

	hasManagePy := hasFile(dir, "manage.py")

	switch {
	case hasManagePy || strings.Contains(reqText, "django"):
		res.Framework = "Django"
		res.Type = "Django"
		res.DevServer = "manage.py"
		res.DevCommand = "python manage.py runserver"
		res.Entry = "manage.py"
		res.Confidence = 96

	case strings.Contains(reqText, "fastapi"):
		res.Framework = "FastAPI"
		res.Type = "FastAPI"
		res.DevServer = "Uvicorn"
		res.DevCommand = "uvicorn main:app --reload"
		res.Entry = "main.py"
		res.Confidence = 95

	case strings.Contains(reqText, "flask"):
		res.Framework = "Flask"
		res.Type = "Flask"
		res.DevServer = "Flask"
		res.DevCommand = "flask run"
		res.Entry = "app.py"
		res.Confidence = 95

	default:
		res.Framework = "Python"
		res.Type = "Python Application"
		res.DevServer = "Python"
		res.DevCommand = "python app.py"
		res.Entry = "app.py"
		res.Confidence = 85
	}
}

func detectPHP(dir string, res *Result) {
	res.Language = "PHP"
	res.Runtime = "PHP"
	res.PackageManager = "composer"
	res.Entry = "composer.json"
	res.Confidence = 95

	compBytes, _ := os.ReadFile(filepath.Join(dir, "composer.json"))
	compStr := strings.ToLower(string(compBytes))

	if strings.Contains(compStr, "laravel/framework") || hasFile(dir, "artisan") {
		res.Framework = "Laravel"
		res.Type = "Laravel"
		res.DevServer = "Artisan"
		res.DevCommand = "php artisan serve"
		res.Entry = "artisan"
	} else if strings.Contains(compStr, "symfony/framework-bundle") {
		res.Framework = "Symfony"
		res.Type = "Symfony"
		res.DevServer = "Symfony CLI"
		res.DevCommand = "symfony server:start"
	} else {
		res.Framework = "PHP"
		res.Type = "PHP Application"
		res.DevServer = "PHP Built-in"
		res.DevCommand = "php -S 127.0.0.1:8000"
	}
}

func detectRuby(dir string, res *Result) {
	res.Language = "Ruby"
	res.Runtime = "Ruby"
	res.PackageManager = "bundler"
	res.Entry = "Gemfile"
	res.Confidence = 95

	gemBytes, _ := os.ReadFile(filepath.Join(dir, "Gemfile"))
	gemStr := strings.ToLower(string(gemBytes))

	if strings.Contains(gemStr, "rails") {
		res.Framework = "Rails"
		res.Type = "Ruby on Rails"
		res.DevServer = "Puma / Rails"
		res.DevCommand = "bin/rails server"
	} else if strings.Contains(gemStr, "sinatra") {
		res.Framework = "Sinatra"
		res.Type = "Sinatra"
		res.DevServer = "Rack"
		res.DevCommand = "bundle exec ruby app.rb"
	} else {
		res.Framework = "Ruby"
		res.Type = "Ruby Application"
		res.DevServer = "Ruby"
		res.DevCommand = "bundle exec rackup"
	}
}

func detectJVM(dir string, res *Result) {
	res.Runtime = "Java / JVM"
	res.Confidence = 95

	if hasFile(dir, "pom.xml") {
		res.PackageManager = "Maven"
		res.BuildSystem = "mvn"
		res.Entry = "pom.xml"
	} else {
		res.PackageManager = "Gradle"
		res.BuildSystem = "gradle"
		res.Entry = "build.gradle"
	}

	if hasFile(dir, "build.gradle.kts") {
		res.Language = "Kotlin"
	} else {
		res.Language = "Java"
	}

	// Check Spring Boot, Quarkus, Micronaut
	text := ""
	for _, f := range []string{"pom.xml", "build.gradle", "build.gradle.kts"} {
		if b, err := os.ReadFile(filepath.Join(dir, f)); err == nil {
			text += " " + strings.ToLower(string(b))
		}
	}

	if strings.Contains(text, "spring-boot") {
		res.Framework = "Spring Boot"
		res.Type = "Spring Boot (" + res.Language + ")"
		if res.PackageManager == "Maven" {
			res.DevCommand = "mvn spring-boot:run"
		} else {
			res.DevCommand = "./gradlew bootRun"
		}
	} else if strings.Contains(text, "quarkus") {
		res.Framework = "Quarkus"
		res.Type = "Quarkus"
		res.DevCommand = "mvn quarkus:dev"
	} else if strings.Contains(text, "micronaut") {
		res.Framework = "Micronaut"
		res.Type = "Micronaut"
		res.DevCommand = "mvn mn:run"
	} else {
		res.Framework = "JVM"
		res.Type = "Java/Kotlin Application"
		res.DevCommand = "mvn exec:java"
	}
}

func detectDotNet(dir string, res *Result) {
	res.Runtime = ".NET"
	res.PackageManager = "dotnet / NuGet"
	res.BuildSystem = "dotnet"
	res.DevCommand = "dotnet watch run"
	res.BuildCommand = "dotnet build"
	res.Confidence = 95

	if hasGlob(dir, "*.fsproj") {
		res.Language = "F#"
		res.Type = "F# (.NET)"
	} else {
		res.Language = "C#"
		res.Type = "C# (ASP.NET Core)"
	}
	res.Framework = "ASP.NET Core"
	res.Entry = "*.csproj"
}

func detectElixir(dir string, res *Result) {
	res.Language = "Elixir"
	res.Runtime = "Elixir / BEAM"
	res.PackageManager = "Mix"
	res.BuildSystem = "mix"
	res.Entry = "mix.exs"
	res.Confidence = 95

	mixBytes, _ := os.ReadFile(filepath.Join(dir, "mix.exs"))
	if strings.Contains(string(mixBytes), ":phoenix") {
		res.Framework = "Phoenix"
		res.Type = "Phoenix (Elixir)"
		res.DevServer = "Phoenix Endpoint"
		res.DevCommand = "mix phx.server"
	} else {
		res.Framework = "Mix"
		res.Type = "Elixir Application"
		res.DevCommand = "iex -S mix"
	}
}

// Format formats the Result for clean human-readable CLI display.
func (r *Result) Format() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Project:         %s\n", r.ProjectName)
	fmt.Fprintf(&sb, "Type:            %s\n", r.Type)
	fmt.Fprintf(&sb, "Language:        %s\n", r.Language)
	if r.Framework != "" && r.Framework != "None" {
		fmt.Fprintf(&sb, "Framework:       %s\n", r.Framework)
	}
	if r.PackageManager != "" && r.PackageManager != "None" {
		fmt.Fprintf(&sb, "Package Manager: %s\n", r.PackageManager)
	}
	if r.Runtime != "" {
		fmt.Fprintf(&sb, "Runtime:         %s\n", r.Runtime)
	}
	if r.DevServer != "" {
		fmt.Fprintf(&sb, "Dev Server:      %s\n", r.DevServer)
	}
	if r.Entry != "" {
		fmt.Fprintf(&sb, "Entry:           %s\n", r.Entry)
	}
	if r.DevCommand != "" {
		fmt.Fprintf(&sb, "Dev Command:     %s\n", r.DevCommand)
	}
	if r.BuildCommand != "" {
		fmt.Fprintf(&sb, "Build Command:   %s\n", r.BuildCommand)
	}
	if r.PreviewCommand != "" {
		fmt.Fprintf(&sb, "Preview Command: %s\n", r.PreviewCommand)
	}
	if r.WasmTarget {
		fmt.Fprintf(&sb, "WebAssembly:     Detected (Browser target)\n")
	}
	if r.IsProfiled {
		fmt.Fprintf(&sb, "Profile:         .serve/config (Active override)\n")
	}
	fmt.Fprintf(&sb, "Confidence:      %d%%\n", r.Confidence)
	return sb.String()
}

func hasFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func hasAnyFile(dir string, names ...string) bool {
	for _, n := range names {
		if hasFile(dir, n) {
			return true
		}
	}
	return false
}

func hasGlob(dir, pattern string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, pattern))
	return len(matches) > 0
}

func detectLanguageByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".html", ".htm":
		return "HTML"
	case ".css":
		return "CSS"
	case ".js", ".mjs", ".cjs":
		return "JavaScript"
	case ".ts", ".mts":
		return "TypeScript"
	case ".wasm":
		return "WebAssembly"
	case ".json":
		return "JSON"
	default:
		return "Plain Text / Static"
	}
}
