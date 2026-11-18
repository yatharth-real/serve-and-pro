# Serve & Pro

<img width="736" height="372" alt="image" src="https://github.com/user-attachments/assets/5e85cd45-cfce-4114-ae12-bc2865260194" />
<img width="764" height="249" alt="image" src="https://github.com/user-attachments/assets/029195ec-5f3f-4852-aac6-37060445f3f2" />


**Serve** is a fast, lightweight, universal local development and web serving orchestrator built from scratch in Go.

It consists of two unified command groups (available via four permanent global commands):
* `serve` & `servecc` — Universal local web serving and development orchestrator. Inspects your project, detects its ecosystem/framework, selects the appropriate runtime pipeline, supervises processes, handles reverse proxying, and serves static files with sub-millisecond latency.
* `pro` & `procc` — Local project control center and navigation system. Maintains a fast local registry, tracks aliases, tags, groups, inspects Git status, renders directory trees, searches files, and manages project lifecycle.

---

## Key Highlights

- **Universal Project Detection**: Automatically detects 20+ frontend and backend frameworks and ecosystems (React, Next.js, Vite, Vue, Nuxt, Svelte, Astro, Angular, Remix, SolidJS, Django, FastAPI, Flask, Laravel, Rails, Go, Rust, ASP.NET Core, Spring Boot, Phoenix, and WebAssembly).
- **Zero Cloud & Zero Telemetry**: 100% private, local-first. No remote API calls, no accounts, no analytics.
- **Lightweight Orchestrator**: Uses your existing local toolchains (Node, Bun, Deno, Python, Go, Cargo, etc.) rather than bundling bloated runtime environments inside the binary.
- **Process Supervisor**: Clean process tree management, signal forwarding, log streaming, and zero orphaned child processes on Ctrl+C.
- **Massive Project Optimization**: Lazy directory traversal, zero-copy streaming, bounded 64KB buffers, HTTP Range requests, ETags, and strict directory ignore rules (`node_modules`, `.git`, `dist`, `target`, `__pycache__`, etc.).
- **Strict Security**: Path traversal protection, Windows Alternate Data Stream defense (`::$DATA`), and automatic protection of sensitive files (`.env`, `*.pem`, `*.key`, `id_rsa`).
- **Four Permanent Global Commands**: `serve`, `servecc`, `pro`, and `procc`.

---

## Installation & Build

### Prerequisites
- Go 1.22+ (to compile from source)

### Building the Binaries
```bash
go build -ldflags="-s -w" -o serve.exe ./cmd/serve
go build -ldflags="-s -w" -o pro.exe ./cmd/pro
```

### Global Windows Installation
To install globally across all PowerShell and terminal sessions:

```powershell
.\install.ps1
```

This installs four permanent globally accessible commands:
* `serve` & `servecc`: Local development orchestrator & web server
* `pro` & `procc`: Project control center and navigation system

To uninstall:
```powershell
.\uninstall.ps1
```

---

## 1. Serve: Universal Web Serving & Orchestration

Run in any project directory:

```powershell
serve .
```

Serve inspects the project. If it is a framework project (e.g. Next.js, Vite, Django), it validates the required runtime and launches the development pipeline. If it is a static project or if `--static` is passed, it launches the high-performance native HTTP server.

### Primary Commands

| Command | Description |
|---------|-------------|
| `serve .` | Automatically detect project and launch dev environment or static server |
| `serve detect [path]` | Deeply analyze project ecosystem, framework, language, package manager, and entry |
| `serve doctor [path]` | Check system toolchain availability and project readiness |
| `serve deps [path]` | Inspect dependencies, lockfiles, and counts (`--check`, `--outdated`) |
| `serve build [path]` | Execute the detected production build command |
| `serve preview [path]` | Launch production build preview server |
| `serve benchmark [path]` | Measure server throughput (req/sec), TTFB, and memory usage |
| `serve ps` | List active Serve-supervised processes |
| `serve ports` | Display all currently allocated/listening ports |
| `serve status [name]` | Inspect running status of a project |
| `serve logs [name]` | View recent logs from a supervised process (`--tail <n>`) |
| `serve stop [name\|pid]` | Terminate a running process tree cleanly |
| `serve restart [name\|pid]` | Restart a project dev server |
| `serve clean [path]` | Clean standard build and cache directories (`dist`, `build`, `.next`, etc.) |
| `serve config` | View or modify global Serve settings (`get <k>`, `set <k> <v>`) |
| `serve env [path]` | Inspect local environment configuration files safely |
| `serve routes [path]` | Display configured proxy routes and static endpoints |
| `serve exec [path] <cmd...>` | Execute arbitrary command inside project directory |

### Core Serving Flags

| Flag | Shorthand | Description |
|------|-----------|-------------|
| `--port <num\|auto>` | `-p` | Set TCP port (default: 3000, or `auto` for next open port) |
| `--host <ip>` | `-H` | Bind to a specific interface (default: 127.0.0.1) |
| `--lan` | | Bind to 0.0.0.0 and display network/LAN address for mobile testing |
| `--open` | `-o` | Automatically open default browser on launch |
| `--https` | `-s` | Enable TLS with in-memory self-signed certificate |
| `--watch` | `-w` | Enable file watching with automatic browser live reload |
| `--static` | | Force native static file serving (bypasses framework dev command) |
| `--build` | | Run build command prior to serving |
| `--prod` | | Run in production/preview mode |
| `--proxy <rule>` | | Reverse proxy route (e.g. `/api=http://localhost:8000`) |
| `--version` | `-v` | Show version information |
| `--help` | `-h` | Show help text |

---

## 2. Project Profiles (`.serve/config`)

You can override automatic detection or configure custom parameters by adding `.serve/config` (or `.serve/config.json`) to your project root:

```ini
# .serve/config
runtime = Node.js
framework = Next.js
dev_command = pnpm dev
build_command = pnpm build
port = 4000
proxy = /api=http://localhost:8080
static_directory = dist
```

---

## 3. Pro: Project Control Center

`pro` manages and navigates your project registry.

### Core Commands
* `pro set <path> [--name <alias>]` — Register a project
* `pro ls [--group] [--tag <tag>]` — List registered projects
* `pro rm <name>` — Remove project from registry (*never touches disk files*)
* `pro path <name>` — Output absolute path (`cd $(pro path myapp)`)
* `pro open [name]` — Open in Windows Explorer / OS file manager
* `pro serve [name] [flags]` — Serve project using the Serve engine

### Universal Lifecycle Commands
* `pro dev [name]` — Launch development server
* `pro build [name]` — Run production build
* `pro preview [name]` — Launch production preview
* `pro detect [name]` — Run universal detector on project
* `pro doctor [name]` — Run environment diagnostics
* `pro stop [name]` — Terminate running project process
* `pro restart [name]` — Restart project process
* `pro logs [name]` — View project process logs
* `pro ports` — Display active port allocations

### Navigation & Discovery
* `pro use <name>` / `pro current` — Set and show active project
* `pro info [name]` — Project summary (type, runtime, files, git status)
* `pro tree [name] [--depth N]` — Unicode directory tree
* `pro find [name] <query> [--ext ext]` — Fast file search
* `pro tags [name] [add|rm <tag>]` — Tag-based organization
* `pro group [name] <group>` — Group-based organization
* `pro recent` — Recently accessed projects

---

## 4. Reverse Proxying

Serve includes a built-in reverse proxy with WebSocket support:

```powershell
serve . --proxy /api=http://localhost:8000
```

Requests to `/api/...` are proxied upstream to `http://localhost:8000/...`, while static files and framework routes continue to be served locally.

---

## 5. Automated Test Suite

Run all tests:

```powershell
go test -v ./...
```

The test suite exercises:
* Project detection across 20+ frameworks and runtimes
* System runtime checking and diagnostic output
* Static file serving, index detection, range requests, and ETag generation
* Secret file protection (`.env`, `*.key`, `*.pem`)
* Reverse proxy forwarding
* Process supervisor, port allocation, and registration
* Directory tree rendering and file finding
* Project profiles (`.serve/config`)
* Full end-to-end integration workflows
