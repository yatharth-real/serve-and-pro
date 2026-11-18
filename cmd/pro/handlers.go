package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"serve/pkg/detector"
	"serve/pkg/git"
	"serve/pkg/project"
	"serve/pkg/registry"
	"serve/pkg/supervisor"
	"serve/pkg/ui"
	"serve/pkg/version"
	"serve/pkg/workspace"
)

// ─── dashboard ────────────────────────────────────────────────────────────────

func handleDashboard(reg *registry.Registry, palette *ui.Palette, args []string) {
	var jsonMode bool
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
		}
	}

	projects := reg.List()
	activeProj, _ := reg.GetActive()

	var activeInfo *project.Project
	if activeProj != nil {
		activeInfo, _ = project.Discover(activeProj.Path)
	}

	procs, _ := supervisor.ListProcesses()

	ws, _ := workspace.Open("")
	activeWS := ws.Active()

	// Count modified projects
	modified := 0
	running := len(procs)
	for _, p := range projects {
		gi := git.Inspect(p.Path)
		if gi.Dirty {
			modified++
		}
	}

	recent := reg.Recent(5)

	if jsonMode {
		type dashJSON struct {
			Version       string `json:"version"`
			Projects      int    `json:"projects"`
			Running       int    `json:"running"`
			Modified      int    `json:"modified"`
			ActiveProject string `json:"active_project,omitempty"`
			Workspace     string `json:"workspace,omitempty"`
		}
		d := dashJSON{
			Version:  version.Current,
			Projects: len(projects),
			Running:  running,
			Modified: modified,
		}
		if activeProj != nil {
			d.ActiveProject = activeProj.Name
		}
		if activeWS != nil {
			d.Workspace = activeWS.Name
		}
		ui.PrintJSON(d)
		return
	}

	// Human output
	fmt.Printf("  PRO  v%s\n\n", version.Current)

	// Active project
	if activeProj != nil {
		fmt.Println(palette.Header("ACTIVE"))
		fmt.Printf("  %s\n", palette.Bold(activeProj.Name))
		fmt.Printf("  %s\n\n", palette.Muted(activeProj.Path))

		if activeInfo != nil {
			fmt.Println(palette.Header("PROJECT"))
			if activeInfo.Language != "" || activeInfo.Runtime != "" {
				fmt.Printf("  %s / %s\n", activeInfo.Language, activeInfo.Runtime)
			}
			if activeInfo.PackageManager != "" && activeInfo.PackageManager != "None" {
				fmt.Printf("  %s\n", activeInfo.PackageManager)
			}
			if activeInfo.GitBranch != "" {
				dirtyStr := ""
				if activeInfo.GitDirty {
					dirtyStr = " (dirty)"
				}
				fmt.Printf("  Git: %s%s\n", activeInfo.GitBranch, dirtyStr)
			}
			fmt.Println()
		}
	} else {
		fmt.Println(palette.Muted("  No active project. Run 'pro set .' to register the current directory."))
		fmt.Println()
	}

	// Workspace
	if activeWS != nil {
		fmt.Println(palette.Header("WORKSPACE"))
		fmt.Printf("  %s  (%d projects)\n\n", palette.Accent(activeWS.Name), len(activeWS.Projects))
	}

	// Services
	if len(procs) > 0 {
		fmt.Println(palette.Header("SERVICES"))
		for _, p := range procs {
			fmt.Printf("  %-20s :%d   %s\n", p.ID, p.Port, palette.Success("RUNNING"))
		}
		fmt.Println()
	}

	// Summary
	fmt.Println(palette.Header("WORKSPACE OVERVIEW"))
	fmt.Printf("  Projects:   %d\n", len(projects))
	fmt.Printf("  Running:    %d\n", running)
	fmt.Printf("  Modified:   %d\n", modified)
	fmt.Println()

	// Recent
	if len(recent) > 0 {
		fmt.Println(palette.Header("RECENT"))
		for _, p := range recent {
			fmt.Printf("  %s\n", p.Name)
		}
		fmt.Println()
	}
}

// ─── health ───────────────────────────────────────────────────────────────────

func handleHealth(reg *registry.Registry, palette *ui.Palette, args []string) {
	var name string
	var jsonMode bool
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
		} else if !strings.HasPrefix(a, "-") {
			name = a
		}
	}

	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	det, _ := detector.Detect(proj.Path)
	gi := git.Inspect(proj.Path)
	procs, _ := supervisor.ListProcesses()

	var lines []ui.StatusLine

	// Runtime check
	if det != nil && det.Runtime != "" && det.Runtime != "Browser" && det.Runtime != "Browser / Native" {
		lines = append(lines, ui.StatusLine{Label: "Runtime", Severity: ui.SeverityOK, Message: det.Runtime})
	} else {
		lines = append(lines, ui.StatusLine{Label: "Runtime", Severity: ui.SeverityInfo, Message: "No runtime required (static)"})
	}

	// Dependencies
	nodeModules := filepath.Join(proj.Path, "node_modules")
	pkgJSON := filepath.Join(proj.Path, "package.json")
	if _, err := os.Stat(pkgJSON); err == nil {
		if _, err := os.Stat(nodeModules); err != nil {
			lines = append(lines, ui.StatusLine{Label: "Dependencies", Severity: ui.SeverityWarn, Message: "node_modules missing — run install"})
		} else {
			lines = append(lines, ui.StatusLine{Label: "Dependencies", Severity: ui.SeverityOK, Message: "installed"})
		}
	}

	// Git
	if gi.Branch != "" {
		if gi.Dirty {
			lines = append(lines, ui.StatusLine{Label: "Git", Severity: ui.SeverityWarn, Message: fmt.Sprintf("%s — %d uncommitted change(s)", gi.Branch, gi.Changed)})
		} else {
			lines = append(lines, ui.StatusLine{Label: "Git", Severity: ui.SeverityOK, Message: fmt.Sprintf("%s — clean", gi.Branch)})
		}
	} else {
		lines = append(lines, ui.StatusLine{Label: "Git", Severity: ui.SeverityInfo, Message: "not a git repository"})
	}

	// Dev server
	serverRunning := false
	for _, p := range procs {
		if strings.EqualFold(p.ID, proj.Name) {
			serverRunning = true
		}
	}
	if serverRunning {
		lines = append(lines, ui.StatusLine{Label: "Dev Server", Severity: ui.SeverityOK, Message: "running"})
	} else {
		lines = append(lines, ui.StatusLine{Label: "Dev Server", Severity: ui.SeverityInfo, Message: "stopped"})
	}

	// Filesystem
	if _, err := os.Stat(proj.Path); err != nil {
		lines = append(lines, ui.StatusLine{Label: "Filesystem", Severity: ui.SeverityError, Message: "project directory missing"})
	} else {
		lines = append(lines, ui.StatusLine{Label: "Filesystem", Severity: ui.SeverityOK, Message: "OK"})
	}

	if jsonMode {
		ui.PrintJSON(lines)
		return
	}

	fmt.Printf("\nProject Health — %s\n\n", proj.Name)
	ui.PrintStatusLines(lines, palette)
	fmt.Println()
}

// ─── workspace ────────────────────────────────────────────────────────────────

func handleWorkspace(reg *registry.Registry, palette *ui.Palette, args []string) {
	ws, err := workspace.Open("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening workspace store: %v\n", err)
		os.Exit(1)
	}

	if len(args) == 0 {
		// list
		printWorkspaceList(ws, palette)
		return
	}

	sub := args[0]
	switch sub {
	case "ls", "list":
		printWorkspaceList(ws, palette)
	case "create":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: pro workspace create <name>")
			os.Exit(1)
		}
		w, err := ws.Create(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Workspace %q created.\n", w.Name)
	case "add":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: pro workspace add <workspace> <project>")
			os.Exit(1)
		}
		if err := ws.Add(args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added %q to workspace %q.\n", args[2], args[1])
	case "rm", "remove":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: pro workspace rm <workspace> <project>")
			os.Exit(1)
		}
		if err := ws.Remove(args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed %q from workspace %q.\n", args[2], args[1])
	case "use":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: pro workspace use <name>")
			os.Exit(1)
		}
		if err := ws.Use(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Workspace %q not found. Create it first with 'pro workspace create %s'.\n", args[1], args[1])
			os.Exit(1)
		}
		fmt.Printf("Active workspace set to %q.\n", args[1])
	case "status":
		active := ws.Active()
		if active == nil {
			fmt.Println("No active workspace. Use 'pro workspace use <name>'.")
			return
		}
		fmt.Printf("Workspace: %s\n\n", active.Name)
		for _, p := range active.Projects {
			fmt.Printf("  %s\n", p)
		}
	case "start":
		active := ws.Active()
		if active == nil {
			fmt.Fprintln(os.Stderr, "No active workspace. Use 'pro workspace use <name>'.")
			os.Exit(1)
		}
		fmt.Printf("Starting workspace %q (%d projects)...\n", active.Name, len(active.Projects))
		for _, alias := range active.Projects {
			p, err := reg.Peek(alias)
			if err != nil {
				fmt.Printf("  [SKIP] %s — not found in registry\n", alias)
				continue
			}
			fmt.Printf("  [%s] → %s\n", alias, p.Path)
			handleServe(reg, []string{alias})
		}
	case "stop":
		active := ws.Active()
		if active == nil {
			fmt.Fprintln(os.Stderr, "No active workspace.")
			os.Exit(1)
		}
		for _, alias := range active.Projects {
			_ = supervisor.StopProcess(alias)
			fmt.Printf("Stopped %s\n", alias)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown workspace sub-command %q. Use ls, create, add, rm, use, status, start, stop.\n", sub)
		os.Exit(1)
	}
}

func printWorkspaceList(ws *workspace.Store, palette *ui.Palette) {
	list := ws.List()
	if len(list) == 0 {
		fmt.Println("No workspaces. Use 'pro workspace create <name>'.")
		return
	}
	activeName := ""
	if a := ws.Active(); a != nil {
		activeName = a.Name
	}
	fmt.Println("Workspaces")
	fmt.Println()
	for _, w := range list {
		marker := " "
		if w.Name == activeName {
			marker = "*"
		}
		fmt.Printf("  %s %-20s  %d project(s)\n", marker, palette.Accent(w.Name), len(w.Projects))
	}
}

// ─── task ─────────────────────────────────────────────────────────────────────

func handleTask(reg *registry.Registry, args []string) {
	// pro task [project] [task]
	var projName, taskName string
	if len(args) >= 1 {
		projName = args[0]
	}
	if len(args) >= 2 {
		taskName = args[1]
	}

	proj, err := resolveProject(reg, projName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	pi, _ := project.Discover(proj.Path)
	var scripts map[string]string
	if pi != nil {
		scripts = pi.Scripts
	}

	if taskName == "" {
		// list tasks
		if len(scripts) == 0 {
			fmt.Printf("No tasks found for %q.\n", proj.Name)
			return
		}
		fmt.Printf("Tasks for %s\n\n", proj.Name)
		keys := make([]string, 0, len(scripts))
		for k := range scripts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %-20s %s\n", k, scripts[k])
		}
		return
	}

	// execute task
	cmdStr, ok := scripts[taskName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Task %q not found. Run 'pro task %s' to list available tasks.\n", taskName, proj.Name)
		os.Exit(1)
	}

	fmt.Printf("Running task [%s]: %s\n\n", taskName, cmdStr)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}
	cmd.Dir = proj.Path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Task failed: %v\n", err)
		os.Exit(cmd.ProcessState.ExitCode())
	}
}

// ─── git dashboard ────────────────────────────────────────────────────────────

func handleGit(reg *registry.Registry, args []string) {
	var jsonMode bool
	var name string
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
		} else if !strings.HasPrefix(a, "-") {
			name = a
		}
	}

	var targets []*registry.Project
	if name != "" {
		p, err := resolveProject(reg, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		targets = []*registry.Project{p}
	} else {
		targets = reg.List()
	}

	type row struct {
		Name   string `json:"name"`
		Branch string `json:"branch"`
		Dirty  bool   `json:"dirty"`
		Ahead  int    `json:"ahead"`
		Behind int    `json:"behind"`
		Commit string `json:"commit"`
	}

	var rows []row
	for _, p := range targets {
		gi := git.Inspect(p.Path)
		rows = append(rows, row{
			Name:   p.Name,
			Branch: gi.Branch,
			Dirty:  gi.Dirty,
			Ahead:  gi.Ahead,
			Behind: gi.Behind,
			Commit: gi.Commit,
		})
	}

	if jsonMode {
		ui.PrintJSON(rows)
		return
	}

	if len(rows) == 0 {
		fmt.Println("No projects registered.")
		return
	}

	fmt.Println("Git Dashboard")
	fmt.Println()
	fmt.Printf("  %-20s %-16s %-10s %s\n", "PROJECT", "BRANCH", "STATUS", "SYNC")
	fmt.Printf("  %s\n", strings.Repeat("─", 58))
	for _, r := range rows {
		if r.Branch == "" {
			fmt.Printf("  %-20s %-16s %s\n", r.Name, "—", "not a git repo")
			continue
		}
		status := "clean"
		if r.Dirty {
			status = "modified"
		}
		sync := ""
		if r.Ahead > 0 {
			sync += fmt.Sprintf("↑%d ", r.Ahead)
		}
		if r.Behind > 0 {
			sync += fmt.Sprintf("↓%d", r.Behind)
		}
		if sync == "" {
			sync = "—"
		}
		fmt.Printf("  %-20s %-16s %-10s %s\n", r.Name, r.Branch, status, sync)
	}
	fmt.Println()
}

// ─── deps ─────────────────────────────────────────────────────────────────────

func handleDeps(reg *registry.Registry, args []string) {
	// sub-commands: outdated, check, or just list
	var sub, name string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			if sub == "" {
				sub = a
			} else if name == "" {
				name = a
			}
		}
	}

	// Check if sub is actually a project name (not a known subcommand)
	knownSubs := map[string]bool{"outdated": true, "check": true, "list": true, "ls": true}
	if !knownSubs[sub] && sub != "" {
		name = sub
		sub = "list"
	}

	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	det, _ := detector.Detect(proj.Path)

	pm := "npm"
	if det != nil && det.PackageManager != "" && det.PackageManager != "None" {
		pm = strings.ToLower(det.PackageManager)
	}

	switch sub {
	case "outdated":
		fmt.Printf("Checking outdated dependencies for %s (using %s)...\n\n", proj.Name, pm)
		var cmd *exec.Cmd
		switch pm {
		case "pnpm":
			cmd = exec.Command("pnpm", "outdated")
		case "yarn":
			cmd = exec.Command("yarn", "outdated")
		case "bun":
			cmd = exec.Command("bun", "outdated")
		case "cargo":
			cmd = exec.Command("cargo", "outdated")
		default:
			cmd = exec.Command("npm", "outdated")
		}
		cmd.Dir = proj.Path
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	default:
		// List installed
		pkgJSON := filepath.Join(proj.Path, "package.json")
		if _, err := os.Stat(pkgJSON); err == nil {
			data, _ := os.ReadFile(pkgJSON)
			var pkg struct {
				Dependencies    map[string]string `json:"dependencies"`
				DevDependencies map[string]string `json:"devDependencies"`
			}
			if json.Unmarshal(data, &pkg) == nil {
				fmt.Printf("Dependencies for %s\n\n", proj.Name)
				fmt.Printf("  Production:    %d\n", len(pkg.Dependencies))
				fmt.Printf("  Development:   %d\n", len(pkg.DevDependencies))
				return
			}
		}
		fmt.Printf("No dependency information available for %s (%s).\n", proj.Name, pm)
	}
}

// ─── overview ─────────────────────────────────────────────────────────────────

func handleOverview(reg *registry.Registry, palette *ui.Palette, args []string) {
	var name string
	var jsonMode bool
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
		} else if !strings.HasPrefix(a, "-") {
			name = a
		}
	}

	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	pi, _ := project.Discover(proj.Path)
	gi := git.Inspect(proj.Path)
	procs, _ := supervisor.ListProcesses()

	serverStatus := "stopped"
	for _, p := range procs {
		if strings.EqualFold(p.ID, proj.Name) {
			serverStatus = fmt.Sprintf("running (:%d)", p.Port)
		}
	}

	if jsonMode {
		type overviewJSON struct {
			Name      string            `json:"name"`
			Path      string            `json:"path"`
			Framework string            `json:"framework,omitempty"`
			Language  string            `json:"language,omitempty"`
			Runtime   string            `json:"runtime,omitempty"`
			Manager   string            `json:"package_manager,omitempty"`
			GitBranch string            `json:"git_branch,omitempty"`
			GitDirty  bool              `json:"git_dirty"`
			Scripts   map[string]string `json:"scripts,omitempty"`
			Server    string            `json:"server"`
		}
		ov := overviewJSON{
			Name:      proj.Name,
			Path:      proj.Path,
			Server:    serverStatus,
			GitBranch: gi.Branch,
			GitDirty:  gi.Dirty,
		}
		if pi != nil {
			ov.Framework = pi.Framework
			ov.Language = pi.Language
			ov.Runtime = pi.Runtime
			ov.Manager = pi.PackageManager
			ov.Scripts = pi.Scripts
		}
		ui.PrintJSON(ov)
		return
	}

	fmt.Printf("\n  %s\n\n", palette.Bold(proj.Name))
	rows := [][2]string{
		{"Path", proj.Path},
	}
	if pi != nil {
		if pi.Framework != "" && pi.Framework != "None" {
			rows = append(rows, [2]string{"Framework", fmt.Sprintf("%s / %s", pi.Language, pi.Framework)})
		} else if pi.Language != "" {
			rows = append(rows, [2]string{"Language", pi.Language})
		}
		if pi.PackageManager != "" && pi.PackageManager != "None" {
			rows = append(rows, [2]string{"Manager", pi.PackageManager})
		}
		if pi.Runtime != "" {
			rows = append(rows, [2]string{"Runtime", pi.Runtime})
		}
	}
	if gi.Branch != "" {
		rows = append(rows, [2]string{"Git", gi.Branch})
		if gi.Dirty {
			rows = append(rows, [2]string{"Status", fmt.Sprintf("%d uncommitted change(s)", gi.Changed)})
		} else {
			rows = append(rows, [2]string{"Status", "clean"})
		}
	}
	rows = append(rows, [2]string{"Server", serverStatus})

	ui.PrintTable(rows)

	if pi != nil && len(pi.Scripts) > 0 {
		fmt.Println()
		fmt.Println("  Scripts")
		keys := make([]string, 0, len(pi.Scripts))
		for k := range pi.Scripts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("    %s\n", k)
		}
	}
	fmt.Println()
}

// ─── exec ─────────────────────────────────────────────────────────────────────

func handleExec(reg *registry.Registry, args []string) {
	// pro exec <project> -- <command...>
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: pro exec <project> -- <command>")
		os.Exit(1)
	}
	projName := args[0]
	// find '--' separator
	sepIdx := -1
	for i, a := range args {
		if a == "--" {
			sepIdx = i
			break
		}
	}
	if sepIdx < 0 || sepIdx+1 >= len(args) {
		fmt.Fprintln(os.Stderr, "Usage: pro exec <project> -- <command>")
		os.Exit(1)
	}

	proj, err := resolveProject(reg, projName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cmdParts := args[sepIdx+1:]
	var cmd *exec.Cmd
	if len(cmdParts) == 1 {
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", cmdParts[0])
		} else {
			cmd = exec.Command("sh", "-c", cmdParts[0])
		}
	} else {
		cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
	}
	cmd.Dir = proj.Path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ─── init (shell integration) ─────────────────────────────────────────────────

func handleInit(args []string) {
	shell := "powershell"
	if len(args) > 0 {
		shell = strings.ToLower(args[0])
	}

	switch shell {
	case "powershell", "pwsh":
		// Note: the backtick in the comment is a PowerShell escape, must be in a regular string
		fmt.Println("# PRO shell integration for PowerShell")
		fmt.Println("# Add this to your $PROFILE by running:")
		fmt.Println("#   Add-Content $PROFILE \"`nInvoke-Expression (pro init powershell)\"")
		fmt.Println("")
		fmt.Println("function proc {")
		fmt.Println("    param([string]$ProjectName)")
		fmt.Println("    if (-not $ProjectName) {")
		fmt.Println("        pro ls")
		fmt.Println("        return")
		fmt.Println("    }")
		fmt.Println("    $path = pro path $ProjectName 2>&1")
		fmt.Println("    if ($LASTEXITCODE -eq 0 -and $path) {")
		fmt.Println("        Set-Location $path")
		fmt.Println("    } else {")
		fmt.Println("        Write-Error \"Project '$ProjectName' not found.\"")
		fmt.Println("    }")
		fmt.Println("}")
	case "bash":
		fmt.Println("# PRO shell integration for Bash")
		fmt.Println("# Add to ~/.bashrc: eval \"$(pro init bash)\"")
		fmt.Println("")
		fmt.Println("proc() {")
		fmt.Println("    local path")
		fmt.Println("    path=$(pro path \"$1\" 2>/dev/null)")
		fmt.Println("    if [ $? -eq 0 ] && [ -n \"$path\" ]; then")
		fmt.Println("        cd \"$path\"")
		fmt.Println("    else")
		fmt.Println("        echo \"Project '$1' not found.\" >&2")
		fmt.Println("        return 1")
		fmt.Println("    fi")
		fmt.Println("}")
	case "zsh":
		fmt.Println("# PRO shell integration for Zsh")
		fmt.Println("# Add to ~/.zshrc: eval \"$(pro init zsh)\"")
		fmt.Println("")
		fmt.Println("proc() {")
		fmt.Println("    local path")
		fmt.Println("    path=$(pro path \"$1\" 2>/dev/null)")
		fmt.Println("    if [ $? -eq 0 ] && [ -n \"$path\" ]; then")
		fmt.Println("        cd \"$path\"")
		fmt.Println("    else")
		fmt.Println("        echo \"Project '$1' not found.\" >&2")
		fmt.Println("        return 1")
		fmt.Println("    fi")
		fmt.Println("}")
	case "fish":
		fmt.Println("# PRO shell integration for Fish")
		fmt.Println("# Add to ~/.config/fish/config.fish: pro init fish | source")
		fmt.Println("")
		fmt.Println("function proc")
		fmt.Println("    set path (pro path $argv[1] 2>/dev/null)")
		fmt.Println("    if test $status -eq 0; and test -n \"$path\"")
		fmt.Println("        cd $path")
		fmt.Println("    else")
		fmt.Println("        echo \"Project '$argv[1]' not found.\" >&2")
		fmt.Println("    end")
		fmt.Println("end")
	default:
		fmt.Fprintf(os.Stderr, "Unknown shell %q. Supported: powershell, bash, zsh, fish\n", shell)
		os.Exit(1)
	}
}

// ─── repl ─────────────────────────────────────────────────────────────────────

func handleREPL(reg *registry.Registry, palette *ui.Palette) {
	activeName := ""
	if a, err := reg.GetActive(); err == nil {
		activeName = a.Name
	}

	fmt.Printf("PRO REPL v%s  (type 'exit' or Ctrl+C to quit)\n\n", version.Current)
	sc := bufio.NewScanner(os.Stdin)
	for {
		if activeName != "" {
			fmt.Printf("pro [%s]> ", activeName)
		} else {
			fmt.Print("pro> ")
		}

		if !sc.Scan() {
			break
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]
		replArgs := parts[1:]

		switch cmd {
		case "exit", "quit", "q":
			fmt.Println("Goodbye.")
			return
		case "use":
			handleUse(reg, replArgs)
			if len(replArgs) > 0 {
				activeName = replArgs[0]
			}
		case "status":
			handleStatus(reg, replArgs)
		case "dashboard":
			handleDashboard(reg, palette, replArgs)
		case "dev", "serve":
			handleServe(reg, replArgs)
		case "workspace":
			handleWorkspace(reg, palette, replArgs)
		case "task":
			handleTask(reg, replArgs)
		case "git":
			handleGit(reg, replArgs)
		case "ls":
			handleList(reg, replArgs)
		case "help", "?":
			fmt.Println("Commands: use, status, dashboard, dev, workspace, task, git, ls, exit")
		default:
			fmt.Fprintf(os.Stderr, "Unknown command %q. Type 'help' for available commands.\n", cmd)
		}
	}
}

// ─── jump ─────────────────────────────────────────────────────────────────────

func handleJump(reg *registry.Registry, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: pro jump <query>")
		os.Exit(1)
	}
	query := strings.ToLower(strings.Join(args, " "))

	projects := reg.List()
	var matches []*registry.Project
	for _, p := range projects {
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Path), query) {
			matches = append(matches, p)
		}
	}
	// also check tags
	for _, p := range projects {
		alreadyIn := false
		for _, m := range matches {
			if m.Name == p.Name {
				alreadyIn = true
				break
			}
		}
		if alreadyIn {
			continue
		}
		for _, t := range p.Tags {
			if strings.Contains(strings.ToLower(t), query) {
				matches = append(matches, p)
				break
			}
		}
	}

	if len(matches) == 0 {
		fmt.Printf("No projects matching %q.\n", query)
		return
	}
	if len(matches) == 1 {
		// Unique match — print path (can be used with proc in shell)
		fmt.Println(matches[0].Path)
		return
	}
	fmt.Printf("Multiple matches for %q:\n\n", query)
	for _, m := range matches {
		fmt.Printf("  %-20s %s\n", m.Name, m.Path)
	}
}

// ─── env ──────────────────────────────────────────────────────────────────────

func handleEnv(reg *registry.Registry, args []string) {
	var name string
	var checkMode bool
	for _, a := range args {
		if a == "check" {
			checkMode = true
		} else if !strings.HasPrefix(a, "-") {
			name = a
		}
	}

	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	envFiles := []string{".env", ".env.local", ".env.development", ".env.production", ".env.test"}

	fmt.Printf("Environment — %s\n\n", proj.Name)

	found := false
	for _, ef := range envFiles {
		path := filepath.Join(proj.Path, ef)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		found = true
		fmt.Printf("  %s\n", ef)
		lines := strings.Split(string(data), "\n")
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l == "" || strings.HasPrefix(l, "#") {
				continue
			}
			parts := strings.SplitN(l, "=", 2)
			key := parts[0]
			if checkMode {
				if len(parts) == 2 && parts[1] != "" {
					fmt.Printf("    %-30s PRESENT\n", key)
				} else {
					fmt.Printf("    %-30s MISSING\n", key)
				}
			} else {
				fmt.Printf("    %-30s REDACTED\n", key)
			}
		}
		fmt.Println()
	}
	if !found {
		fmt.Println("  No .env files found.")
	}
}

// ─── new ──────────────────────────────────────────────────────────────────────

func handleNew(reg *registry.Registry, args []string) {
	// pro new --git <url> [--name <alias>]
	var gitURL, customName string
	for i, a := range args {
		switch a {
		case "--git":
			if i+1 < len(args) {
				gitURL = args[i+1]
			}
		case "--name":
			if i+1 < len(args) {
				customName = args[i+1]
			}
		}
	}

	if gitURL == "" {
		fmt.Fprintln(os.Stderr, "Usage: pro new --git <url> [--name <alias>]")
		os.Exit(1)
	}

	cwd, _ := os.Getwd()
	parts := strings.Split(strings.TrimSuffix(gitURL, ".git"), "/")
	repoName := parts[len(parts)-1]
	dest := filepath.Join(cwd, repoName)

	fmt.Printf("Cloning %s...\n", gitURL)
	cloneCmd := exec.Command("git", "clone", gitURL, dest)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Clone failed: %v\n", err)
		os.Exit(1)
	}

	alias := customName
	if alias == "" {
		alias = repoName
	}
	p, err := reg.Add(dest, alias)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Clone succeeded but registration failed: %v\n", err)
		os.Exit(1)
	}
	_ = reg.SetActive(p.Name)

	det, _ := detector.Detect(dest)
	fmt.Printf("\nProject cloned.\n")
	fmt.Printf("Registered: %s\n", p.Name)
	fmt.Printf("Path:       %s\n", p.Path)
	if det != nil && det.Language != "" {
		fmt.Printf("Detected:   %s / %s / %s\n", det.Language, det.Runtime, det.PackageManager)
	}
}

// ─── gc ───────────────────────────────────────────────────────────────────────

func handleGC(reg *registry.Registry, args []string) {
	dryRun := false
	for _, a := range args {
		if a == "--dry-run" || a == "-n" {
			dryRun = true
		}
	}

	projects := reg.List()
	var missing []*registry.Project
	for _, p := range projects {
		if !p.Exists() {
			missing = append(missing, p)
		}
	}

	if len(missing) == 0 {
		fmt.Println("All registered projects exist on disk. Nothing to collect.")
		return
	}

	fmt.Println("Missing Projects")
	fmt.Println()
	for _, p := range missing {
		fmt.Printf("  %-20s %s\n", p.Name, p.Path)
	}
	fmt.Println()

	if dryRun {
		fmt.Printf("(dry run) Would remove %d project(s) from registry.\n", len(missing))
		return
	}

	for _, p := range missing {
		_ = reg.Remove(p.Name)
	}
	fmt.Printf("Removed %d project(s) from registry (files were already gone from disk).\n", len(missing))
}

// ─── export ───────────────────────────────────────────────────────────────────

func handleExport(reg *registry.Registry, args []string) {
	outFile := ""
	for i, a := range args {
		if a == "--output" || a == "-o" {
			if i+1 < len(args) {
				outFile = args[i+1]
			}
		}
	}

	type exportedProject struct {
		Name  string   `json:"name"`
		Path  string   `json:"path"`
		Group string   `json:"group,omitempty"`
		Tags  []string `json:"tags,omitempty"`
	}

	projects := reg.List()
	var exported []exportedProject
	for _, p := range projects {
		exported = append(exported, exportedProject{
			Name:  p.Name,
			Path:  p.Path,
			Group: p.Group,
			Tags:  p.Tags,
		})
	}

	data, _ := json.MarshalIndent(exported, "", "  ")
	if outFile != "" {
		if err := os.WriteFile(outFile, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Exported %d project(s) to %s\n", len(exported), outFile)
	} else {
		fmt.Println(string(data))
	}
}

// ─── import ───────────────────────────────────────────────────────────────────

func handleImport(reg *registry.Registry, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: pro import <file.json>")
		os.Exit(1)
	}

	type importedProject struct {
		Name  string   `json:"name"`
		Path  string   `json:"path"`
		Group string   `json:"group,omitempty"`
		Tags  []string `json:"tags,omitempty"`
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var projects []importedProject
	if err := json.Unmarshal(data, &projects); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid import file: %v\n", err)
		os.Exit(1)
	}

	imported, skipped := 0, 0
	for _, p := range projects {
		if _, err := os.Stat(p.Path); err != nil {
			fmt.Printf("  [SKIP] %s — path does not exist: %s\n", p.Name, p.Path)
			skipped++
			continue
		}
		rp, err := reg.Add(p.Path, p.Name)
		if err != nil {
			fmt.Printf("  [SKIP] %s — %v\n", p.Name, err)
			skipped++
			continue
		}
		if p.Group != "" {
			_ = reg.SetGroup(rp.Name, p.Group)
		}
		for _, t := range p.Tags {
			_ = reg.AddTag(rp.Name, t)
		}
		fmt.Printf("  [OK] %s — %s\n", rp.Name, rp.Path)
		imported++
	}
	fmt.Printf("\nImported: %d, Skipped: %d\n", imported, skipped)
}

// ─── report ───────────────────────────────────────────────────────────────────

func handleReport(reg *registry.Registry, args []string) {
	var jsonMode bool
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
		}
	}

	procs, _ := supervisor.ListProcesses()
	projects := reg.List()

	type reportData struct {
		PROVersion   string   `json:"pro_version"`
		GOOS         string   `json:"os"`
		GOARCH       string   `json:"arch"`
		Projects     int      `json:"projects"`
		Running      int      `json:"running"`
		ProjectNames []string `json:"project_names"`
	}

	names := make([]string, 0, len(projects))
	for _, p := range projects {
		names = append(names, p.Name)
	}

	d := reportData{
		PROVersion:   version.Current,
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		Projects:     len(projects),
		Running:      len(procs),
		ProjectNames: names,
	}

	if jsonMode {
		ui.PrintJSON(d)
		return
	}

	fmt.Println("PRO Diagnostic Report")
	fmt.Println()
	fmt.Printf("  PRO Version: %s\n", d.PROVersion)
	fmt.Printf("  OS:          %s/%s\n", d.GOOS, d.GOARCH)
	fmt.Printf("  Projects:    %d\n", d.Projects)
	fmt.Printf("  Running:     %d\n", d.Running)
	if len(names) > 0 {
		fmt.Printf("  Registry:    %s\n", strings.Join(names, ", "))
	}
	fmt.Println()
}

// ─── clean ────────────────────────────────────────────────────────────────────

func handleClean(reg *registry.Registry, args []string) {
	dryRun := false
	var name string
	for _, a := range args {
		if a == "--dry-run" || a == "-n" {
			dryRun = true
		} else if !strings.HasPrefix(a, "-") {
			name = a
		}
	}

	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	artifactDirs := []string{
		"node_modules", "dist", "build", ".next", ".nuxt",
		"target", ".venv", "__pycache__", ".gradle", "zig-cache",
		"out", ".svelte-kit", ".turbo",
	}

	var toDelete []string
	for _, d := range artifactDirs {
		path := filepath.Join(proj.Path, d)
		if _, err := os.Stat(path); err == nil {
			toDelete = append(toDelete, path)
		}
	}

	if len(toDelete) == 0 {
		fmt.Printf("No artifact directories found in %s.\n", proj.Name)
		return
	}

	fmt.Printf("Artifact directories in %s:\n\n", proj.Name)
	for _, p := range toDelete {
		info, _ := getDirSize(p)
		fmt.Printf("  %s  (%s)\n", filepath.Base(p), info)
	}
	fmt.Println()

	if dryRun {
		fmt.Printf("(dry run) Would delete %d artifact director(ies). Run without --dry-run to proceed.\n", len(toDelete))
		return
	}

	fmt.Printf("Delete %d director(ies)? [y/N] ", len(toDelete))
	var confirm string
	fmt.Scanln(&confirm)
	if !strings.EqualFold(confirm, "y") {
		fmt.Println("Aborted.")
		return
	}
	for _, p := range toDelete {
		if err := os.RemoveAll(p); err != nil {
			fmt.Fprintf(os.Stderr, "  [ERROR] %s: %v\n", filepath.Base(p), err)
		} else {
			fmt.Printf("  Deleted %s\n", filepath.Base(p))
		}
	}
}

func getDirSize(path string) (string, error) {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, _ := d.Info()
			if info != nil {
				total += info.Size()
			}
		}
		return nil
	})
	if total < 1024*1024 {
		return fmt.Sprintf("%dKB", total/1024), nil
	}
	return fmt.Sprintf("%.1fMB", float64(total)/(1024*1024)), nil
}

// ─── snapshot ─────────────────────────────────────────────────────────────────

func handleSnapshot(reg *registry.Registry, args []string) {
	sub := ""
	var name string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			if sub == "" {
				sub = a
			} else {
				name = a
			}
		}
	}

	home, _ := os.UserHomeDir()
	snapshotDir := filepath.Join(home, ".pro", "snapshots")
	_ = os.MkdirAll(snapshotDir, 0o755)

	switch sub {
	case "ls", "list", "":
		entries, _ := os.ReadDir(snapshotDir)
		if len(entries) == 0 {
			fmt.Println("No snapshots. Run 'pro snapshot' to create one.")
			return
		}
		fmt.Println("Snapshots")
		fmt.Println()
		for _, e := range entries {
			fmt.Printf("  %s\n", e.Name())
		}
		return
	default:
		// treat sub as snapshot name or create new
		if name == "" {
			name = sub
			sub = "create"
		}
	}

	switch sub {
	case "create":
		snapName := name
		if snapName == "" {
			snapName = time.Now().Format("2006-01-02T15-04-05")
		}
		type snapProject struct {
			Name  string   `json:"name"`
			Path  string   `json:"path"`
			Group string   `json:"group,omitempty"`
			Tags  []string `json:"tags,omitempty"`
		}
		var projects []snapProject
		for _, p := range reg.List() {
			projects = append(projects, snapProject{
				Name: p.Name, Path: p.Path, Group: p.Group, Tags: p.Tags,
			})
		}
		data, _ := json.MarshalIndent(projects, "", "  ")
		dest := filepath.Join(snapshotDir, snapName+".json")
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Snapshot saved: %s\n", dest)
	default:
		fmt.Fprintf(os.Stderr, "Unknown snapshot command %q.\n", sub)
		os.Exit(1)
	}
}
