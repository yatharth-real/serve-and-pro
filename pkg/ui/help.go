package ui

import (
	"fmt"
	"strings"
)

// RenderThemeInfo displays current theme and accent settings along with their configuration sources.
func RenderThemeInfo(tool string, p *Palette) string {
	var sb strings.Builder
	sb.WriteString(p.Header("Theme Configuration (.cc visual system)"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("  Active Theme:   %s  %s\n",
		p.Accent(p.Theme.Name),
		p.Muted(fmt.Sprintf("(source: %s)", p.ThemeSource)),
	))
	sb.WriteString(fmt.Sprintf("  Description:    %s\n", p.Theme.Description))
	sb.WriteString(fmt.Sprintf("  Active Accent:  %s  %s\n",
		p.Accent(p.AccentName),
		p.Muted(fmt.Sprintf("(source: %s)", p.AccentSource)),
	))

	sb.WriteString("\n")
	sb.WriteString(p.Muted("  Shared Config:  " + GetSharedCCPath() + "\n"))
	sb.WriteString(p.Muted("  Tool Config:    " + GetToolPath(tool) + "\n\n"))

	sb.WriteString("Commands:\n")
	sb.WriteString(fmt.Sprintf("  %s theme list            %s\n", p.Accent(tool), p.Muted("List all available themes")))
	sb.WriteString(fmt.Sprintf("  %s theme <name>          %s\n", p.Accent(tool), p.Muted("Set theme for this tool")))
	sb.WriteString(fmt.Sprintf("  %s theme <name> --global %s\n", p.Accent(tool), p.Muted("Set shared .cc global theme")))
	sb.WriteString(fmt.Sprintf("  %s accent list           %s\n", p.Accent(tool), p.Muted("List all supported accent colors")))
	sb.WriteString(fmt.Sprintf("  %s accent <color>        %s\n", p.Accent(tool), p.Muted("Set accent color for this tool")))
	sb.WriteString(fmt.Sprintf("  %s accent <color> --global %s\n", p.Accent(tool), p.Muted("Set shared .cc global accent")))

	return sb.String()
}

// RenderThemeList formats the table of available themes.
func RenderThemeList(activeTheme string, p *Palette) string {
	var sb strings.Builder
	sb.WriteString(p.Header("Available Themes (.cc visual system)"))
	sb.WriteString("\n\n")

	themes := ListThemes()
	maxName := 12
	for _, t := range themes {
		if len(t.Name) > maxName {
			maxName = len(t.Name)
		}
	}

	for _, t := range themes {
		marker := "  "
		nameStr := t.Name
		if strings.EqualFold(t.Name, activeTheme) {
			marker = p.Accent("▶ ")
			nameStr = p.Accent(t.Name)
		}
		sb.WriteString(fmt.Sprintf("%s%-*s  %-8s  %s\n",
			marker,
			maxName+2,
			nameStr,
			p.Muted("["+t.DefaultAccent+"]"),
			t.Description,
		))
	}

	sb.WriteString("\n")
	sb.WriteString(p.Muted("Use 'serve theme <name>' or 'pro theme <name>' to activate.\n"))
	return sb.String()
}

// RenderAccentList formats the list of available accent colors.
func RenderAccentList(activeAccent string, p *Palette) string {
	var sb strings.Builder
	sb.WriteString(p.Header("Available Accent Colors (.cc visual system)"))
	sb.WriteString("\n\n")

	accents := ListAccents()
	for _, a := range accents {
		marker := "  "
		swatch, _ := GetAccentColor(a)
		colorSample := p.Apply(swatch, "██")
		nameStr := a
		if strings.EqualFold(a, activeAccent) {
			marker = p.Accent("▶ ")
			nameStr = p.Accent(a)
		}
		sb.WriteString(fmt.Sprintf("%s%s  %-10s\n", marker, colorSample, nameStr))
	}

	sb.WriteString("\n")
	sb.WriteString(p.Muted("Use 'serve accent <color>' or 'pro accent <color>' to activate.\n"))
	return sb.String()
}

// RenderServeHelp renders the complete theme-aware help screen for Serve and ServeCC.
func RenderServeHelp(appName string, p *Palette) string {
	var sb strings.Builder

	// Banner
	sb.WriteString(RenderBanner(BannerOptions{Command: appName, Palette: p}))

	// Usage
	sb.WriteString(p.Header("USAGE"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s %s\n", p.Accent(appName), p.Muted("[path] [flags]")))
	sb.WriteString(fmt.Sprintf("  %s %s\n\n", p.Accent(appName), p.Muted("<command> [arguments] [flags]")))

	// Primary Commands
	sb.WriteString(p.Header("PRIMARY COMMANDS"))
	sb.WriteString("\n")
	cmds := [][2]string{
		{"detect [path]", "Inspect project and detect runtime, framework, and package manager"},
		{"projects [path]", "Discover runnable applications and workspaces in a monorepo"},
		{"doctor [path]", "Diagnose system runtimes, toolchains, and project readiness"},
		{"deps [path] [flags]", "Inspect dependencies (--check, --outdated)"},
		{"build [path]", "Execute project's production build command"},
		{"preview [path]", "Launch production preview server"},
		{"benchmark [path]", "Measure throughput, TTFB, and latency under concurrency"},
		{"ps", "List running Serve-managed processes"},
		{"ports", "Show currently allocated/listening ports"},
		{"status [name]", "Display real-time status of running project"},
		{"logs [name] [--tail <n>]", "Display logs from a supervised process"},
		{"stop [name|pid]", "Gracefully stop a running Serve process"},
		{"restart [name|pid]", "Restart a running Serve process"},
		{"clean [path]", "Remove common build artifacts and cache directories"},
		{"theme [name|list]", "Manage terminal theme and visual personality"},
		{"accent [color|list]", "Manage primary terminal accent highlight color"},
		{"config [get|set <k> <v>]", "View or update Serve configuration"},
		{"env [path]", "Inspect detected environment variables (.env files)"},
		{"routes [path]", "Inspect proxy and static routes"},
		{"exec [path] <cmd...>", "Execute arbitrary command within project context"},
	}

	for _, c := range cmds {
		sb.WriteString(fmt.Sprintf("  %-26s %s\n", p.Accent(c[0]), c[1]))
	}
	sb.WriteString("\n")

	// Flags
	sb.WriteString(p.Header("FLAGS"))
	sb.WriteString("\n")
	flags := [][2]string{
		{"-p, --port <num|auto>", "Port to listen on (default: 3000, or 'auto')"},
		{"-H, --host <ip>", "Host address to bind to (default: 127.0.0.1)"},
		{"    --lan", "Bind to 0.0.0.0 and display network/LAN address"},
		{"-o, --open", "Open server URL in default browser on startup"},
		{"-s, --https", "Enable HTTPS with local self-signed certificate"},
		{"-w, --watch", "Watch files and enable automatic live reload"},
		{"    --static", "Force native static file serving (bypasses runtime)"},
		{"    --build", "Run build command before serving"},
		{"    --prod", "Run in production/preview mode"},
		{"    --project <name>", "Target specific project/workspace in a monorepo"},
		{"    --proxy <rule>", "Configure reverse proxy rule (e.g. /api=http://localhost:8000)"},
		{"    --theme <name>", "Override visual theme for this invocation"},
		{"    --accent <color>", "Override accent color for this invocation"},
		{"    --no-color", "Disable all ANSI colors and styling"},
		{"-q, --quiet", "Suppress banner and decorative output for scripting"},
		{"-v, --version", "Show version information"},
		{"-h, --help", "Show this help message"},
	}

	for _, f := range flags {
		sb.WriteString(fmt.Sprintf("  %-26s %s\n", p.Accent(f[0]), f[1]))
	}
	sb.WriteString("\n")

	// Examples
	sb.WriteString(p.Header("EXAMPLES"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s .\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s detect ./my-app\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s projects\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s . --project web\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s . --proxy /api=http://localhost:8080\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s --static ./dist\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s benchmark . -c 20 -n 1000\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s theme ocean\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s accent green\n", p.Muted(appName)))
	sb.WriteString("\n")

	return sb.String()
}

// RenderProHelp renders the complete theme-aware help screen for Pro and ProCC.
func RenderProHelp(appName string, p *Palette) string {
	var sb strings.Builder

	// Banner
	sb.WriteString(RenderBanner(BannerOptions{Command: appName, Palette: p}))

	// Usage
	sb.WriteString(p.Header("USAGE"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s %s\n\n", p.Accent(appName), p.Muted("<command> [arguments] [flags]")))

	// Project Management
	sb.WriteString(p.Header("PROJECT MANAGEMENT"))
	sb.WriteString("\n")
	projCmds := [][2]string{
		{"ls [--tag <t>] [--group]", "List all registered projects"},
		{"set [path] [--name <n>]", "Register current or target directory as project (active)"},
		{"use <name>", "Switch the active project"},
		{"current", "Display the currently active project"},
		{"rm <name>", "Remove a project from registry (preserves files on disk)"},
		{"rename <old> <new>", "Rename project alias"},
		{"path [name]", "Print absolute filesystem path to project"},
		{"open [name]", "Open project directory in native system file manager"},
		{"recent", "Show recently accessed projects"},
		{"tags <name> [add|rm <t>]", "View or manage project tags"},
		{"group <name> [group-name]", "Assign or view project group"},
		{"scan [path]", "Automatically scan filesystem for projects"},
		{"tree [name]", "Display structured directory tree of project"},
		{"find <query>", "Search for files across registered projects"},
		{"new --git <url>", "Clone a git repo and register it as a project"},
	}
	for _, c := range projCmds {
		sb.WriteString(fmt.Sprintf("  %-30s %s\n", p.Accent(c[0]), c[1]))
	}
	sb.WriteString("\n")

	// Intelligence
	sb.WriteString(p.Header("INTELLIGENCE"))
	sb.WriteString("\n")
	intelligenceCmds := [][2]string{
		{"dashboard [--json]", "Show project, workspace, git, and service summary"},
		{"overview [name] [--json]", "Detailed project overview with scripts"},
		{"health [name] [--json]", "Check project health (runtime, deps, git, server)"},
		{"info [name]", "Display comprehensive project metadata"},
		{"status [name]", "Display real-time project metrics and server state"},
		{"detect [name]", "Inspect detected ecosystem and framework"},
		{"doctor [name]", "Run diagnostics on project readiness and runtimes"},
	}
	for _, c := range intelligenceCmds {
		sb.WriteString(fmt.Sprintf("  %-30s %s\n", p.Accent(c[0]), c[1]))
	}
	sb.WriteString("\n")

	// Workspace & Orchestration
	sb.WriteString(p.Header("WORKSPACE & ORCHESTRATION"))
	sb.WriteString("\n")
	wsCmds := [][2]string{
		{"workspace [create|use|add|rm|ls]", "Manage named workspace groups"},
		{"task [project] [task]", "List or execute project scripts/tasks"},
		{"git [name] [--json]", "Cross-project git status dashboard"},
		{"deps [project] [outdated]", "Inspect project dependencies"},
		{"exec <project> -- <cmd>", "Execute command in project directory"},
		{"jump <query>", "Fuzzy-search project path by name/tag"},
		{"env [name] [check]", "Inspect .env files (values are redacted)"},
	}
	for _, c := range wsCmds {
		sb.WriteString(fmt.Sprintf("  %-30s %s\n", p.Accent(c[0]), c[1]))
	}
	sb.WriteString("\n")

	// Execution & Runtime
	sb.WriteString(p.Header("EXECUTION & RUNTIME"))
	sb.WriteString("\n")
	execCmds := [][2]string{
		{"serve [name] [flags]", "Launch project dev server using universal adapter"},
		{"dev [name] [flags]", "Alias for 'serve'"},
		{"build [name]", "Execute project's production build command"},
		{"preview [name]", "Launch production preview server"},
		{"stop <name>", "Stop a running project process"},
		{"restart <name>", "Restart a running project process"},
		{"logs <name>", "View logs from project server"},
		{"ports", "List active port allocations"},
		{"run [name] [action]", "Execute a pre-configured action script"},
		{"action add <proj> <a> <c>", "Configure custom action script for project"},
	}
	for _, c := range execCmds {
		sb.WriteString(fmt.Sprintf("  %-30s %s\n", p.Accent(c[0]), c[1]))
	}
	sb.WriteString("\n")

	// Shell Integration
	sb.WriteString(p.Header("SHELL INTEGRATION"))
	sb.WriteString("\n")
	shellCmds := [][2]string{
		{"init <shell>", "Emit shell integration (powershell, bash, zsh, fish)"},
		{"repl", "Start interactive PRO REPL session"},
	}
	for _, c := range shellCmds {
		sb.WriteString(fmt.Sprintf("  %-30s %s\n", p.Accent(c[0]), c[1]))
	}
	sb.WriteString("\n")

	// Maintenance
	sb.WriteString(p.Header("MAINTENANCE"))
	sb.WriteString("\n")
	maintCmds := [][2]string{
		{"gc [--dry-run]", "Remove registry entries for missing projects"},
		{"clean [name] [--dry-run]", "Delete build artifact directories"},
		{"export [--output <file>]", "Export registry to JSON file"},
		{"import <file.json>", "Import projects from JSON file"},
		{"snapshot [name]", "Save or list registry snapshots"},
		{"report [--json]", "PRO diagnostic report"},
		{"theme [name|list]", "Manage terminal theme and visual personality"},
		{"accent [color|list]", "Manage primary terminal accent highlight color"},
	}
	for _, c := range maintCmds {
		sb.WriteString(fmt.Sprintf("  %-30s %s\n", p.Accent(c[0]), c[1]))
	}
	sb.WriteString("\n")

	// Global Flags
	sb.WriteString(p.Header("GLOBAL FLAGS"))
	sb.WriteString("\n")
	flags := [][2]string{
		{"--theme <name>", "Override visual theme for this invocation"},
		{"--accent <color>", "Override accent color for this invocation"},
		{"--no-color", "Disable all ANSI colors and styling"},
		{"-q, --quiet", "Suppress banner and decorative output for scripting"},
		{"--json", "Output as JSON (supported by many commands)"},
		{"-v, --version", "Show version information"},
		{"--git-version", "Print raw version string (for tooling)"},
		{"-h, --help", "Show this help message"},
	}
	for _, f := range flags {
		sb.WriteString(fmt.Sprintf("  %-30s %s\n", p.Accent(f[0]), f[1]))
	}
	sb.WriteString("\n")

	// Examples
	sb.WriteString(p.Header("EXAMPLES"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s ls\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s set ./my-web-app\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s dashboard\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s workspace create myteam\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s git\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s init powershell\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s gc --dry-run\n", p.Muted(appName)))
	sb.WriteString(fmt.Sprintf("  %s theme ocean\n", p.Muted(appName)))
	sb.WriteString("\n")

	return sb.String()
}

