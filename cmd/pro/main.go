package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"serve/pkg/actions"
	"serve/pkg/adapter"
	"serve/pkg/detector"
	"serve/pkg/doctor"
	"serve/pkg/finder"
	"serve/pkg/registry"
	"serve/pkg/scanner"
	"serve/pkg/server"
	"serve/pkg/supervisor"
	"serve/pkg/tree"
	"serve/pkg/ui"
	"serve/pkg/util"
	"serve/pkg/version"
)

func getAppName() string {
	base := strings.ToLower(filepath.Base(os.Args[0]))
	base = strings.TrimSuffix(base, ".exe")
	if strings.Contains(base, "procc") {
		return "procc"
	}
	return "pro"
}

func printUsage(palette *ui.Palette) {
	fmt.Print(ui.RenderProHelp(getAppName(), palette))
}

func extractUIOptions(args []string) (ui.RuntimeOptions, []string) {
	var opts ui.RuntimeOptions
	var remaining []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--no-color":
			opts.NoColor = true
		case arg == "-q" || arg == "--quiet":
			opts.Quiet = true
		case arg == "--json":
			opts.JSON = true
		case arg == "--theme" && i+1 < len(args):
			opts.ThemeOverride = args[i+1]
			i++
		case strings.HasPrefix(arg, "--theme="):
			opts.ThemeOverride = strings.TrimPrefix(arg, "--theme=")
		case arg == "--accent" && i+1 < len(args):
			opts.AccentOverride = args[i+1]
			i++
		case strings.HasPrefix(arg, "--accent="):
			opts.AccentOverride = strings.TrimPrefix(arg, "--accent=")
		default:
			remaining = append(remaining, arg)
		}
	}
	return opts, remaining
}

func extractFlags(args []string, flagsWithArgs map[string]bool) ([]string, []string) {
	var flagList []string
	var posList []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagList = append(flagList, arg)
			parts := strings.SplitN(arg, "=", 2)
			name := strings.TrimLeft(parts[0], "-")
			if len(parts) == 1 && flagsWithArgs[name] && i+1 < len(args) {
				i++
				flagList = append(flagList, args[i])
			}
		} else {
			posList = append(posList, arg)
		}
	}
	return flagList, posList
}

func resolveProject(reg *registry.Registry, name string) (*registry.Project, error) {
	if name == "." {
		name = ""
	}
	if name != "" {
		return reg.Get(name)
	}

	// 1. Check for explicitly set active project
	if proj, err := reg.GetActive(); err == nil && proj != nil {
		return proj, nil
	}

	// 2. Check if current working directory is a registered project
	cwd, err := os.Getwd()
	if err == nil {
		for _, p := range reg.List() {
			if strings.EqualFold(filepath.Clean(p.Path), filepath.Clean(cwd)) {
				return p, nil
			}
		}

		// 3. Fallback to current working directory as an ad-hoc project
		base := filepath.Base(cwd)
		return &registry.Project{
			Name:         base,
			Path:         cwd,
			RegisteredAt: time.Now(),
			LastAccessed: time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("no project specified and no active project is currently set")
}

func main() {
	appName := getAppName()
	uiOpts, remainingArgs := extractUIOptions(os.Args[1:])
	palette := ui.ResolvePalette(appName, uiOpts)

	if len(remainingArgs) < 1 {
		printUsage(palette)
		os.Exit(0)
	}

	cmd := remainingArgs[0]
	args := remainingArgs[1:]

	if cmd == "--help" || cmd == "-h" || cmd == "help" {
		printUsage(palette)
		os.Exit(0)
	}
	if cmd == "--version" || cmd == "-v" || cmd == "version" {
		if !palette.Quiet && !palette.JSON {
			fmt.Print(ui.RenderBanner(ui.BannerOptions{Command: appName, Palette: palette}))
		}
		if palette.JSON {
			fmt.Printf(`{"product":"%s","version":"%s"}`+"\n", appName, version.Current)
		} else {
			fmt.Printf("%s v%s\n", palette.Bold(appName), version.Current)
		}
		os.Exit(0)
	}
	if cmd == "--git-version" {
		fmt.Println(version.Current)
		os.Exit(0)
	}

	if cmd == "theme" {
		handleThemeCommand(appName, palette, args)
		return
	}
	if cmd == "accent" {
		handleAccentCommand(appName, palette, args)
		return
	}

	reg, err := registry.Open("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing registry: %v\n", err)
		os.Exit(1)
	}

	switch cmd {
	case "set":
		handleSet(reg, args)
	case "ls", "list":
		handleList(reg, args)
	case "rm", "remove":
		handleRemove(reg, args)
	case "path":
		handlePath(reg, args)
	case "open":
		handleOpen(reg, args)
	case "serve":
		handleServe(reg, args)
	case "dev":
		handleDev(reg, args)
	case "build":
		handleBuild(reg, args)
	case "preview":
		handlePreview(reg, args)
	case "detect":
		handleDetect(reg, args)
	case "doctor":
		handleDoctor(reg, args)
	case "stop":
		handleStop(reg, args)
	case "restart":
		handleRestart(reg, args)
	case "logs":
		handleLogs(reg, args)
	case "ports":
		handlePorts()
	case "use":
		handleUse(reg, args)
	case "current":
		handleCurrent(reg, args)
	case "rename":
		handleRename(reg, args)
	case "tags", "tag":
		handleTags(reg, args)
	case "group":
		handleGroup(reg, args)
	case "recent":
		handleRecent(reg, args)
	case "info":
		handleInfo(reg, args)
	case "status":
		handleStatus(reg, args)
	case "scan":
		handleScan(reg, args)
	case "tree":
		handleTree(reg, args)
	case "find":
		handleFind(reg, args)
	case "run":
		handleRun(reg, args)
	case "action", "actions":
		handleAction(reg, args)
	case "dashboard":
		handleDashboard(reg, palette, args)
	case "health":
		handleHealth(reg, palette, args)
	case "workspace":
		handleWorkspace(reg, palette, args)
	case "task":
		handleTask(reg, args)
	case "git":
		handleGit(reg, args)
	case "deps":
		handleDeps(reg, args)
	case "overview":
		handleOverview(reg, palette, args)
	case "exec":
		handleExec(reg, args)
	case "init":
		handleInit(args)
	case "repl":
		handleREPL(reg, palette)
	case "jump":
		handleJump(reg, args)
	case "env":
		handleEnv(reg, args)
	case "new":
		handleNew(reg, args)
	case "gc":
		handleGC(reg, args)
	case "export":
		handleExport(reg, args)
	case "import":
		handleImport(reg, args)
	case "report":
		handleReport(reg, args)
	case "clean":
		handleClean(reg, args)
	case "snapshot":
		handleSnapshot(reg, args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %q. Run '%s --help' for available commands.\n", cmd, appName)
		os.Exit(1)
	}
}

func handleThemeCommand(tool string, palette *ui.Palette, args []string) {
	if len(args) == 0 {
		fmt.Print(ui.RenderThemeInfo(tool, palette))
		return
	}

	sub := strings.ToLower(args[0])
	if sub == "list" || sub == "ls" {
		fmt.Print(ui.RenderThemeList(palette.Theme.Name, palette))
		return
	}

	themeName := sub
	theme, ok := ui.GetTheme(themeName)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: Unknown theme %q.\nAvailable themes:\n", themeName)
		for _, t := range ui.ListThemes() {
			fmt.Fprintf(os.Stderr, "  - %-12s %s\n", t.Name, t.Description)
		}
		os.Exit(1)
	}

	global := false
	for _, a := range args[1:] {
		if a == "--global" || a == "-g" {
			global = true
		}
	}

	if err := ui.SaveThemeConfig(tool, global, theme.Name, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving theme configuration: %v\n", err)
		os.Exit(1)
	}

	targetScope := fmt.Sprintf("for %s (~/.%s/theme.json)", tool, tool)
	if global {
		targetScope = "globally for all .cc tools (~/.cc/theme.json)"
	}
	fmt.Printf("%s Activated theme %s %s\n", palette.Success("[OK]"), palette.Accent(theme.Name), targetScope)
}

func handleAccentCommand(tool string, palette *ui.Palette, args []string) {
	if len(args) == 0 {
		fmt.Printf("Active Accent: %s  %s\n\n", palette.Accent(palette.AccentName), palette.Muted(fmt.Sprintf("(source: %s)", palette.AccentSource)))
		fmt.Println("Run 'pro accent list' to view supported accents.")
		return
	}

	sub := strings.ToLower(args[0])
	if sub == "list" || sub == "ls" {
		fmt.Print(ui.RenderAccentList(palette.AccentName, palette))
		return
	}

	accentName := sub
	_, ok := ui.GetAccentColor(accentName)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: Unknown accent color %q.\nSupported accents: %s\n", accentName, strings.Join(ui.ListAccents(), ", "))
		os.Exit(1)
	}

	global := false
	for _, a := range args[1:] {
		if a == "--global" || a == "-g" {
			global = true
		}
	}

	if err := ui.SaveThemeConfig(tool, global, "", accentName); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving accent configuration: %v\n", err)
		os.Exit(1)
	}

	targetScope := fmt.Sprintf("for %s (~/.%s/theme.json)", tool, tool)
	if global {
		targetScope = "globally for all .cc tools (~/.cc/theme.json)"
	}
	fmt.Printf("%s Activated accent %s %s\n", palette.Success("[OK]"), palette.Accent(accentName), targetScope)
}

func handleSet(reg *registry.Registry, args []string) {
	fs := flag.NewFlagSet("set", flag.ContinueOnError)
	var customName string
	fs.StringVar(&customName, "name", "", "Custom alias name for the project")
	fs.StringVar(&customName, "n", "", "Custom alias name (shorthand)")

	flagList, posList := extractFlags(args, map[string]bool{"name": true, "n": true})

	if err := fs.Parse(flagList); err != nil {
		os.Exit(1)
	}

	targetPath := "."
	explicitPath := false
	if len(posList) > 0 {
		targetPath = posList[0]
		explicitPath = true
	}

	proj, err := reg.Add(targetPath, customName)
	if err != nil {
		if !explicitPath {
			absPath, _ := filepath.Abs(targetPath)
			candidateName := customName
			if candidateName == "" && absPath != "" {
				candidateName = filepath.Base(absPath)
			}
			if existing, getErr := reg.Get(candidateName); getErr == nil {
				_ = reg.SetActive(existing.Name)
				fmt.Printf("Project %q already registered -> set as active (%s)\n", existing.Name, existing.Path)
				return
			}
			for _, p := range reg.List() {
				if absPath != "" && strings.EqualFold(filepath.Clean(p.Path), filepath.Clean(absPath)) {
					_ = reg.SetActive(p.Name)
					fmt.Printf("Project %q already registered -> set as active (%s)\n", p.Name, p.Path)
					return
				}
			}
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	_ = reg.SetActive(proj.Name)
	fmt.Printf("Registered project %q -> %s (active)\n", proj.Name, proj.Path)
}

func handleList(reg *registry.Registry, args []string) {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	var tagFilter string
	var grouped bool
	fs.StringVar(&tagFilter, "tag", "", "Filter projects by tag")
	fs.BoolVar(&grouped, "group", false, "Group projects by category")
	fs.BoolVar(&grouped, "grouped", false, "Group projects by category")

	flagList, _ := extractFlags(args, map[string]bool{"tag": true})

	if err := fs.Parse(flagList); err != nil {
		os.Exit(1)
	}

	projects := reg.List()
	if len(projects) == 0 {
		fmt.Printf("No registered projects. Use '%s set <path>' to register one.\n", getAppName())
		return
	}

	activeProj, _ := reg.GetActive()
	activeName := ""
	if activeProj != nil {
		activeName = activeProj.Name
	}

	if tagFilter != "" {
		var filtered []*registry.Project
		for _, p := range projects {
			for _, t := range p.Tags {
				if strings.EqualFold(t, tagFilter) {
					filtered = append(filtered, p)
					break
				}
			}
		}
		projects = filtered
		if len(projects) == 0 {
			fmt.Printf("No projects with tag %q found.\n", tagFilter)
			return
		}
	}

	fmt.Println("Projects")
	fmt.Println()

	maxNameLen := 0
	for _, p := range projects {
		if len(p.Name) > maxNameLen {
			maxNameLen = len(p.Name)
		}
	}
	if maxNameLen < 12 {
		maxNameLen = 12
	}

	if grouped {
		groups := make(map[string][]*registry.Project)
		for _, p := range projects {
			grp := p.Group
			if grp == "" {
				grp = "ungrouped"
			}
			groups[grp] = append(groups[grp], p)
		}

		for grp, projs := range groups {
			fmt.Printf("[%s]\n", grp)
			for _, p := range projs {
				marker := " "
				if p.Name == activeName {
					marker = "*"
				}
				status := ""
				if !p.Exists() {
					status = " [missing]"
				}
				fmt.Printf("  %s %-*s  %s%s\n", marker, maxNameLen, p.Name, p.Path, status)
			}
			fmt.Println()
		}
		return
	}

	for _, p := range projects {
		marker := " "
		if p.Name == activeName {
			marker = "*"
		}
		status := ""
		if !p.Exists() {
			status = " [missing]"
		}
		fmt.Printf("  %s %-*s  %s%s\n", marker, maxNameLen, p.Name, p.Path, status)
	}
}

func handleRemove(reg *registry.Registry, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s rm <name>\n", getAppName())
		os.Exit(1)
	}

	name := args[0]
	if err := reg.Remove(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed project %q from registry (files preserved on disk).\n", name)
}

func handlePath(reg *registry.Registry, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(proj.Path)
}

func handleOpen(reg *registry.Registry, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !proj.Exists() {
		fmt.Fprintf(os.Stderr, "Error: project directory %s does not exist on disk\n", proj.Path)
		os.Exit(1)
	}

	if err := util.OpenFileManager(proj.Path); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file manager: %v\n", err)
		os.Exit(1)
	}
}

func handleServe(reg *registry.Registry, args []string) {
	flagsWithArgs := map[string]bool{
		"port": true, "p": true,
		"host": true, "H": true,
	}
	flagList, posList := extractFlags(args, flagsWithArgs)

	var targetProjectName string
	if len(posList) > 0 {
		targetProjectName = posList[0]
	}

	proj, err := resolveProject(reg, targetProjectName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !proj.Exists() {
		fmt.Fprintf(os.Stderr, "Error: project directory %s does not exist on disk\n", proj.Path)
		os.Exit(1)
	}

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	var (
		port  int
		host  string
		lan   bool
		open  bool
		https bool
		watch bool
	)
	fs.IntVar(&port, "port", 3000, "Port")
	fs.IntVar(&port, "p", 3000, "Port")
	fs.StringVar(&host, "host", "", "Host")
	fs.StringVar(&host, "H", "", "Host")
	fs.BoolVar(&lan, "lan", false, "LAN")
	fs.BoolVar(&open, "open", false, "Open")
	fs.BoolVar(&open, "o", false, "Open")
	fs.BoolVar(&https, "https", false, "HTTPS")
	fs.BoolVar(&https, "s", false, "HTTPS")
	fs.BoolVar(&watch, "watch", false, "Watch")
	fs.BoolVar(&watch, "w", false, "Watch")

	if err := fs.Parse(flagList); err != nil {
		os.Exit(1)
	}

	det, err := detector.Detect(proj.Path)
	if err == nil && !det.IsStatic && det.DevCommand != "" {
		rtCheck := adapter.CheckRuntime(det.Runtime)
		if !rtCheck.Installed {
			fmt.Fprint(os.Stderr, adapter.MissingRuntimeError(det))
			os.Exit(1)
		}

		if !util.CheckPortAvailable(host, port) {
			newPort, err := supervisor.FindAvailablePort(port+1, 50)
			if err == nil {
				fmt.Printf("Port %d is already in use. Trying %d...\n", port, newPort)
				port = newPort
			}
		}

		adaptedCmd, extraEnv := adapter.BuildFrameworkDevCommand(det, port, host)
		fmt.Printf("Serve -> %s (%s)\n", det.Type, det.Runtime)
		fmt.Printf("Command: %s\n", adaptedCmd)
		fmt.Printf("Directory: %s\n\n", proj.Name)

		if err := supervisor.SuperviseForeground(proj.Name, proj.Path, adaptedCmd, port, det.Runtime, det.Framework, extraEnv); err != nil {
			fmt.Fprintf(os.Stderr, "Process exited: %v\n", err)
			os.Exit(1)
		}
		return
	}

	srv, err := server.NewServer(server.Config{
		Target:   proj.Path,
		Host:     host,
		Port:     port,
		AutoPort: true,
		LAN:      lan,
		Open:     open,
		HTTPS:    https,
		Watch:    watch,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleDev(reg *registry.Registry, args []string) {
	handleServe(reg, args)
}

func handleBuild(reg *registry.Registry, args []string) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	det, err := detector.Detect(proj.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if det.BuildCommand == "" {
		fmt.Printf("No build command detected for %q.\n", proj.Name)
		return
	}
	fmt.Printf("Building %s (%s)...\n", proj.Name, det.BuildCommand)
	_ = executeInDir(proj.Path, det.BuildCommand)
}

func handlePreview(reg *registry.Registry, args []string) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	det, err := detector.Detect(proj.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if det.PreviewCommand != "" {
		fmt.Printf("Launching preview for %s (%s)...\n", proj.Name, det.PreviewCommand)
		_ = executeInDir(proj.Path, det.PreviewCommand)
		return
	}
	// Fallback to static serving
	dist := filepath.Join(proj.Path, "dist")
	target := proj.Path
	if _, err := os.Stat(dist); err == nil {
		target = dist
	}
	srv, _ := server.NewServer(server.Config{Target: target, AutoPort: true})
	_ = srv.Start()
}

func handleDetect(reg *registry.Registry, args []string) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	det, err := detector.Detect(proj.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(det.Format())
}

func handleDoctor(reg *registry.Registry, args []string) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	path := "."
	if proj, err := resolveProject(reg, name); err == nil {
		path = proj.Path
	}
	fmt.Print(doctor.RunDiagnostics(path))
}

func handleStop(reg *registry.Registry, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: pro stop <project>")
		os.Exit(1)
	}
	name := args[0]
	if err := supervisor.StopProcess(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Stopped %q.\n", name)
}

func handleRestart(reg *registry.Registry, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: pro restart <project>")
		os.Exit(1)
	}
	name := args[0]
	proc, err := supervisor.FindProcess(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Process %q not found.\n", name)
		os.Exit(1)
	}
	cmdStr := proc.Command
	dir := proc.Directory
	port := proc.Port
	rt := proc.Runtime
	fw := proc.Framework
	id := proc.ID
	_ = supervisor.StopProcess(id)
	fmt.Printf("Restarting %s...\n", id)
	_ = supervisor.SuperviseForeground(id, dir, cmdStr, port, rt, fw, nil)
}

func handleLogs(reg *registry.Registry, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: pro logs <project> [--tail <n>]")
		os.Exit(1)
	}
	name := args[0]
	lines, err := supervisor.ReadLogs(name, 50)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	for _, l := range lines {
		fmt.Println(l)
	}
}

func handlePorts() {
	list, _ := supervisor.ListProcesses()
	fmt.Println("Active Port Allocations")
	fmt.Println("────────────────────────")
	if len(list) == 0 {
		fmt.Println("No ports currently reserved.")
		return
	}
	for _, p := range list {
		fmt.Printf("Port %-6d -> %s (PID %d)\n", p.Port, p.ID, p.PID)
	}
}

func executeInDir(dir, cmdStr string) error {
	var cmd *exec.Cmd
	if os.PathSeparator == '\\' {
		cmd = exec.Command("cmd", "/c", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func handleUse(reg *registry.Registry, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s use <name>\n", getAppName())
		os.Exit(1)
	}

	name := args[0]
	if err := reg.SetActive(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Active project set to %q.\n", name)
}

func handleCurrent(reg *registry.Registry, args []string) {
	proj, err := reg.GetActive()
	if err != nil {
		fmt.Printf("No active project selected. Use '%s use <name>' to select one.\n", getAppName())
		return
	}

	fmt.Println(proj.Name)
	fmt.Println(proj.Path)
}

func handleRename(reg *registry.Registry, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s rename <old-name> <new-name>\n", getAppName())
		os.Exit(1)
	}

	oldName, newName := args[0], args[1]
	if err := reg.Rename(oldName, newName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Renamed project alias %q -> %q (directory unchanged).\n", oldName, newName)
}

func handleTags(reg *registry.Registry, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s tags <name> [add|remove <tag>]\n", getAppName())
		os.Exit(1)
	}

	name := args[0]
	proj, err := reg.Get(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(args) == 1 {
		if len(proj.Tags) == 0 {
			fmt.Printf("Project %q has no tags.\n", name)
			return
		}
		fmt.Printf("Tags for %q: %s\n", name, strings.Join(proj.Tags, ", "))
		return
	}

	action := strings.ToLower(args[1])
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s tags %s [add|remove] <tag>\n", getAppName(), name)
		os.Exit(1)
	}
	tag := args[2]

	switch action {
	case "add":
		if err := reg.AddTag(name, tag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added tag %q to %q.\n", tag, name)
	case "remove", "rm":
		if err := reg.RemoveTag(name, tag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed tag %q from %q.\n", tag, name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown tag action %q. Use 'add' or 'remove'.\n", action)
		os.Exit(1)
	}
}

func handleGroup(reg *registry.Registry, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s group <name> <group>\n", getAppName())
		os.Exit(1)
	}

	name, group := args[0], args[1]
	if err := reg.SetGroup(name, group); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Assigned project %q to group %q.\n", name, group)
}

func handleRecent(reg *registry.Registry, args []string) {
	fs := flag.NewFlagSet("recent", flag.ContinueOnError)
	limit := 10
	fs.IntVar(&limit, "limit", 10, "Maximum number of projects to display")
	fs.IntVar(&limit, "n", 10, "Maximum number of projects to display (shorthand)")

	flagList, _ := extractFlags(args, map[string]bool{"limit": true, "n": true})

	if err := fs.Parse(flagList); err != nil {
		os.Exit(1)
	}

	recent := reg.Recent(limit)
	if len(recent) == 0 {
		fmt.Println("No recently used projects.")
		return
	}

	fmt.Println("Recent Projects")
	fmt.Println()
	for _, p := range recent {
		timeStr := p.LastAccessed.Format("2006-01-02 15:04")
		fmt.Printf("  %-16s %-18s %s\n", p.Name, timeStr, p.Path)
	}
}

func handleInfo(reg *registry.Registry, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !proj.Exists() {
		fmt.Printf("Project:   %s\n", proj.Name)
		fmt.Printf("Path:      %s\n", proj.Path)
		fmt.Println("Status:    Directory missing on disk")
		return
	}

	det, _ := detector.Detect(proj.Path)
	info, err := scanner.Scan(proj.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning project: %v\n", err)
		os.Exit(1)
	}

	gitStr := info.GitStatus
	if info.GitBranch != "" && info.GitBranch != "HEAD" {
		gitStr += fmt.Sprintf(" (%s)", info.GitBranch)
	}

	projectType := det.Type
	if projectType == "" {
		projectType = info.Type
	}

	fmt.Printf("Project:   %s\n", proj.Name)
	fmt.Printf("Path:      %s\n", proj.Path)
	fmt.Printf("Type:      %s\n", projectType)
	if det.Language != "" {
		fmt.Printf("Language:  %s\n", det.Language)
	}
	if det.Runtime != "" {
		fmt.Printf("Runtime:   %s\n", det.Runtime)
	}
	fmt.Printf("Files:     %d\n", info.FileCount)
	fmt.Printf("Git:       %s\n", gitStr)
	fmt.Printf("Server:    %s\n", info.ServerStatus)
}

func handleStatus(reg *registry.Registry, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !proj.Exists() {
		fmt.Printf("Project:   %s\nPath:      %s\nStatus:    Directory missing on disk\n", proj.Name, proj.Path)
		return
	}

	info, err := scanner.Scan(proj.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Project:    %s\n", proj.Name)
	fmt.Printf("Path:       %s\n", proj.Path)
	fmt.Printf("Ecosystem:  %s\n", info.Ecosystem)
	fmt.Printf("Type:       %s\n", info.Type)
	fmt.Printf("Git Branch: %s\n", info.GitBranch)
	fmt.Printf("Git Status: %s\n", info.GitStatus)
	fmt.Printf("Server:     %s\n", info.ServerStatus)
	if len(info.ConfigFiles) > 0 {
		fmt.Printf("Configs:    %s\n", strings.Join(info.ConfigFiles, ", "))
	}
}

func handleScan(reg *registry.Registry, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	proj, err := resolveProject(reg, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !proj.Exists() {
		fmt.Fprintf(os.Stderr, "Error: project path %s does not exist on disk\n", proj.Path)
		os.Exit(1)
	}

	info, err := scanner.Scan(proj.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Project:     %s\n", proj.Name)
	fmt.Printf("Path:        %s\n", proj.Path)
	fmt.Printf("Ecosystem:   %s\n", info.Ecosystem)
	fmt.Printf("Primary:     %s\n", info.Type)
	if len(info.Frameworks) > 0 {
		fmt.Printf("Frameworks:  %s\n", strings.Join(info.Frameworks, ", "))
	}
	fmt.Printf("Total Files: %d\n", info.FileCount)
	if len(info.ConfigFiles) > 0 {
		fmt.Printf("Configs:     %s\n", strings.Join(info.ConfigFiles, ", "))
	}
}

func handleTree(reg *registry.Registry, args []string) {
	flagsWithArgs := map[string]bool{"depth": true, "d": true}
	flagList, posList := extractFlags(args, flagsWithArgs)

	fs := flag.NewFlagSet("tree", flag.ContinueOnError)
	depth := 3
	fs.IntVar(&depth, "depth", 3, "Max directory depth to display")
	fs.IntVar(&depth, "d", 3, "Max directory depth (shorthand)")

	if err := fs.Parse(flagList); err != nil {
		os.Exit(1)
	}

	var targetProjectName string
	if len(posList) > 0 {
		targetProjectName = posList[0]
	}

	proj, err := resolveProject(reg, targetProjectName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !proj.Exists() {
		fmt.Fprintf(os.Stderr, "Error: project path %s does not exist on disk\n", proj.Path)
		os.Exit(1)
	}

	if err := tree.RenderTree(os.Stdout, proj.Path, tree.TreeOptions{MaxDepth: depth}); err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering tree: %v\n", err)
		os.Exit(1)
	}
}

func handleFind(reg *registry.Registry, args []string) {
	flagsWithArgs := map[string]bool{
		"ext": true, "dir": true, "limit": true,
	}
	flagList, posList := extractFlags(args, flagsWithArgs)

	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	var ext string
	var dirFilter string
	var limit int
	fs.StringVar(&ext, "ext", "", "Filter by file extension (e.g. ts, js, html)")
	fs.StringVar(&dirFilter, "dir", "", "Filter by directory subpath")
	fs.IntVar(&limit, "limit", 100, "Maximum results to display")

	if err := fs.Parse(flagList); err != nil {
		os.Exit(1)
	}

	var targetProjectName string
	var query string

	if len(posList) == 1 {
		if _, err := reg.Peek(posList[0]); err == nil {
			targetProjectName = posList[0]
			query = ""
		} else {
			targetProjectName = ""
			query = posList[0]
		}
	} else if len(posList) >= 2 {
		targetProjectName = posList[0]
		query = posList[1]
	}

	proj, err := resolveProject(reg, targetProjectName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !proj.Exists() {
		fmt.Fprintf(os.Stderr, "Error: project path %s does not exist on disk\n", proj.Path)
		os.Exit(1)
	}

	results, err := finder.Search(proj.Path, query, finder.FindOptions{
		Extension: ext,
		Directory: dirFilter,
		MaxLimit:  limit,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching project: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No matching files found.")
		return
	}

	for _, r := range results {
		suffix := ""
		if r.IsDir {
			suffix = "/"
		}
		fmt.Printf("  %s%s\n", r.RelativePath, suffix)
	}
	fmt.Printf("\nFound %d match(es).\n", len(results))
}

func handleRun(reg *registry.Registry, args []string) {
	if len(args) == 0 {
		proj, err := reg.GetActive()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Usage: %s run [name] [action]\n", getAppName())
			os.Exit(1)
		}
		if err := actions.Run(proj.Path, "", nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	targetName := args[0]
	actionName := ""
	var extraArgs []string

	if _, err := reg.Peek(targetName); err == nil {
		if len(args) > 1 {
			actionName = args[1]
			extraArgs = args[2:]
		}
	} else {
		actProj, err := reg.GetActive()
		if err == nil {
			targetName = actProj.Name
			actionName = args[0]
			extraArgs = args[1:]
		}
	}

	proj, err := resolveProject(reg, targetName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !proj.Exists() {
		fmt.Fprintf(os.Stderr, "Error: project path %s does not exist on disk\n", proj.Path)
		os.Exit(1)
	}

	if err := actions.Run(proj.Path, actionName, extraArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleAction(reg *registry.Registry, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s action [add|ls] <project> ...\n", getAppName())
		os.Exit(1)
	}

	sub := args[0]
	switch sub {
	case "add", "set":
		if len(args) < 4 {
			fmt.Fprintf(os.Stderr, "Usage: %s action add <project> <action-name> <command>\n", getAppName())
			os.Exit(1)
		}
		projName := args[1]
		actionName := args[2]
		cmdStr := strings.Join(args[3:], " ")

		proj, err := resolveProject(reg, projName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := actions.SetAction(proj.Path, actionName, cmdStr); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configured action %q for project %q.\n", actionName, proj.Name)

	case "ls", "list":
		var projName string
		if len(args) > 1 {
			projName = args[1]
		}
		proj, err := resolveProject(reg, projName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		names, actionsMap, err := actions.ListActions(proj.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(names) == 0 {
			fmt.Printf("No actions configured for %q.\n", proj.Name)
			return
		}

		fmt.Printf("Actions for %q:\n\n", proj.Name)
		for _, k := range names {
			fmt.Printf("  %-12s %s\n", k, actionsMap[k])
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown action command %q. Use 'add' or 'ls'.\n", sub)
		os.Exit(1)
	}
}

var _ = strconv.Itoa
