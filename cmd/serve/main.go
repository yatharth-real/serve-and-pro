package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"serve/pkg/adapter"
	"serve/pkg/benchmark"
	"serve/pkg/config"
	"serve/pkg/deps"
	"serve/pkg/detector"
	"serve/pkg/doctor"
	"serve/pkg/monorepo"
	"serve/pkg/proxy"
	"serve/pkg/server"
	"serve/pkg/supervisor"
	"serve/pkg/ui"
	"serve/pkg/util"
	"serve/pkg/version"
)

func getAppName() string {
	base := strings.ToLower(filepath.Base(os.Args[0]))
	base = strings.TrimSuffix(base, ".exe")
	if strings.Contains(base, "servecc") {
		return "servecc"
	}
	return "serve"
}

func printUsage(palette *ui.Palette) {
	fmt.Print(ui.RenderServeHelp(getAppName(), palette))
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

func main() {
	appName := getAppName()

	uiOpts, remainingArgs := extractUIOptions(os.Args[1:])
	palette := ui.ResolvePalette(appName, uiOpts)

	if len(remainingArgs) == 0 {
		runDefaultServe(".", nil, palette)
		return
	}

	firstArg := remainingArgs[0]

	// Check global flags or subcommands
	if firstArg == "-v" || firstArg == "--version" || firstArg == "version" {
		if !palette.Quiet && !palette.JSON {
			fmt.Print(ui.RenderBanner(ui.BannerOptions{Command: appName, Palette: palette}))
		}
		if palette.JSON {
			fmt.Printf(`{"product":"%s","version":"%s"}`+"\n", appName, version.Current)
		} else {
			fmt.Printf("%s v%s\n", palette.Bold(appName), version.Current)
		}
		return
	}
	if firstArg == "-h" || firstArg == "--help" || firstArg == "help" {
		printUsage(palette)
		return
	}

	subcommands := map[string]bool{
		"detect": true, "projects": true, "doctor": true, "deps": true, "build": true,
		"preview": true, "benchmark": true, "ps": true, "ports": true,
		"status": true, "logs": true, "stop": true, "restart": true,
		"clean": true, "config": true, "env": true, "routes": true,
		"exec": true, "run": true, "theme": true, "accent": true,
	}

	if subcommands[firstArg] {
		handleSubcommand(firstArg, remainingArgs[1:], palette)
		return
	}

	// Otherwise, it is `serve [path] [flags...]`
	flagsWithArgs := map[string]bool{
		"port":    true, "p": true,
		"host":    true, "H": true,
		"proxy":   true,
		"project": true,
	}

	flagList, posList := extractFlags(remainingArgs, flagsWithArgs)
	target := "."
	if len(posList) > 0 {
		target = posList[0]
	}

	runDefaultServe(target, flagList, palette)
}

func handleSubcommand(cmd string, args []string, palette *ui.Palette) {
	switch cmd {
	case "detect":
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		res, err := detector.Detect(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting project: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(res.Format())

	case "projects":
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		ws, err := monorepo.DetectWorkspace(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting workspace: %v\n", err)
			os.Exit(1)
		}
		if !ws.IsMonorepo {
			fmt.Println("No monorepo workspace configuration detected in", target)
			return
		}
		fmt.Printf("Monorepo System: %s\n", ws.System)
		fmt.Printf("Discovered %d runnable project(s):\n\n", len(ws.Projects))
		fmt.Printf("%-20s %-16s %-16s %s\n", "NAME", "FRAMEWORK", "RUNTIME", "PATH")
		fmt.Println("──────────────────────────────────────────────────────────────────────────")
		for _, p := range ws.Projects {
			fw := p.Detection.Framework
			if fw == "" {
				fw = p.Detection.Type
			}
			fmt.Printf("%-20s %-16s %-16s %s\n", p.Name, fw, p.Detection.Runtime, p.RelativePath)
		}
		fmt.Println()

	case "doctor":
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		fmt.Print(doctor.RunDiagnostics(target))

	case "deps":
		target := "."
		var check, outdated bool
		for _, a := range args {
			if a == "--check" {
				check = true
			} else if a == "--outdated" {
				outdated = true
			} else if !strings.HasPrefix(a, "-") {
				target = a
			}
		}

		if outdated {
			if err := deps.RunOutdated(target); err != nil {
				fmt.Fprintf(os.Stderr, "Error running outdated check: %v\n", err)
				os.Exit(1)
			}
			return
		}

		info, err := deps.Inspect(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error inspecting dependencies: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Package Manager:  %s\n", info.PackageManager)
		if info.Lockfile != "" {
			fmt.Printf("Lockfile:         %s\n", info.Lockfile)
		}
		fmt.Printf("Dependencies:     %d\n", info.DepsCount)
		fmt.Printf("Dev Dependencies: %d\n", info.DevDepsCount)
		fmt.Printf("Status:           %s\n", info.Status)

		if check && info.Status == "missing (run install command)" {
			os.Exit(1)
		}

	case "build":
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		det, err := detector.Detect(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if det.BuildCommand == "" {
			fmt.Println("No build command detected for this project.")
			return
		}
		fmt.Printf("Executing build: %s\n", det.BuildCommand)
		if err := executeCommand(target, det.BuildCommand); err != nil {
			fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
			os.Exit(1)
		}

	case "preview":
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		det, err := detector.Detect(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if det.PreviewCommand != "" {
			fmt.Printf("Executing preview: %s\n", det.PreviewCommand)
			if err := executeCommand(target, det.PreviewCommand); err != nil {
				fmt.Fprintf(os.Stderr, "Preview failed: %v\n", err)
				os.Exit(1)
			}
			return
		}
		// Fallback to static serving dist or target
		distDir := filepath.Join(target, "dist")
		if _, err := os.Stat(distDir); err == nil {
			target = distDir
		}
		runDefaultServe(target, []string{"--static"}, palette)

	case "benchmark":
		target := "."
		concurrency := 10
		requests := 500
		for i := 0; i < len(args); i++ {
			if args[i] == "-c" || args[i] == "--concurrency" {
				if i+1 < len(args) {
					concurrency, _ = strconv.Atoi(args[i+1])
					i++
				}
			} else if args[i] == "-n" || args[i] == "--requests" {
				if i+1 < len(args) {
					requests, _ = strconv.Atoi(args[i+1])
					i++
				}
			} else if !strings.HasPrefix(args[i], "-") {
				target = args[i]
			}
		}
		fmt.Printf("Benchmarking %s (%d requests, %d concurrency)...\n", target, requests, concurrency)
		rep, err := benchmark.RunBenchmark(target, concurrency, requests)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Benchmark failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(rep.Format())

	case "ps":
		list, err := supervisor.ListProcesses()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("No active Serve processes.")
			return
		}
		fmt.Printf("%-16s %-8s %-6s %-16s %s\n", "NAME", "PID", "PORT", "RUNTIME", "COMMAND")
		fmt.Println("───────────────────────────────────────────────────────────────────")
		for _, p := range list {
			fmt.Printf("%-16s %-8d %-6d %-16s %s\n", p.ID, p.PID, p.Port, p.Runtime, p.Command)
		}

	case "ports":
		list, err := supervisor.ListProcesses()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Serve Port Allocations")
		fmt.Println("────────────────────────")
		if len(list) == 0 {
			fmt.Println("No ports currently reserved by active Serve processes.")
			return
		}
		for _, p := range list {
			fmt.Printf("Port %-6d -> %s (PID %d)\n", p.Port, p.ID, p.PID)
		}

	case "status":
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		if name == "" {
			handleSubcommand("ps", nil, palette)
			return
		}
		proc, err := supervisor.FindProcess(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Process %q not found.\n", name)
			os.Exit(1)
		}
		fmt.Printf("Name:       %s\n", proc.ID)
		fmt.Printf("PID:        %d\n", proc.PID)
		fmt.Printf("Port:       %d\n", proc.Port)
		fmt.Printf("Command:    %s\n", proc.Command)
		fmt.Printf("Directory:  %s\n", proc.Directory)
		fmt.Printf("Runtime:    %s\n", proc.Runtime)
		fmt.Printf("Started:    %s\n", proc.StartedAt.Format("2006-01-02 15:04:05"))

	case "logs":
		if len(args) < 1 {
			fmt.Println("Usage: serve logs <name|pid> [--tail <n>]")
			os.Exit(1)
		}
		name := args[0]
		tail := 50
		for i := 1; i < len(args); i++ {
			if (args[i] == "--tail" || args[i] == "-n") && i+1 < len(args) {
				tail, _ = strconv.Atoi(args[i+1])
			}
		}
		lines, err := supervisor.ReadLogs(name, tail)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, l := range lines {
			fmt.Println(l)
		}

	case "stop":
		if len(args) < 1 {
			fmt.Println("Usage: serve stop <name|pid>")
			os.Exit(1)
		}
		name := args[0]
		if err := supervisor.StopProcess(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping process: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Stopped process %q.\n", name)

	case "restart":
		if len(args) < 1 {
			fmt.Println("Usage: serve restart <name|pid>")
			os.Exit(1)
		}
		proc, err := supervisor.FindProcess(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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

	case "clean":
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		cleanDirs := []string{
			"dist", "build", "target", ".cache", ".next", ".nuxt",
			".svelte-kit", "coverage", "__pycache__",
		}
		cleaned := 0
		for _, d := range cleanDirs {
			p := filepath.Join(target, d)
			if info, err := os.Stat(p); err == nil && info.IsDir() {
				_ = os.RemoveAll(p)
				fmt.Printf("Removed: %s\n", p)
				cleaned++
			}
		}
		if cleaned == 0 {
			fmt.Println("No standard build or cache directories found to clean.")
		} else {
			fmt.Printf("Cleaned %d directory(s).\n", cleaned)
		}

	case "config":
		if len(args) == 0 {
			cfg, _ := config.LoadGlobalConfig()
			fmt.Printf("Default Port: %d\n", cfg.DefaultPort)
			fmt.Printf("Default Host: %s\n", cfg.DefaultHost)
			fmt.Printf("Auto Port:    %v\n", cfg.AutoPort)
			return
		}
		sub := args[0]
		cfg, _ := config.LoadGlobalConfig()
		if sub == "get" && len(args) > 1 {
			k := args[1]
			fmt.Println(cfg.Settings[k])
		} else if sub == "set" && len(args) > 2 {
			cfg.Settings[args[1]] = args[2]
			_ = config.SaveGlobalConfig(cfg)
			fmt.Printf("Set %s = %s\n", args[1], args[2])
		}

	case "env":
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		envFiles := []string{".env", ".env.local", ".env.development"}
		found := false
		for _, ef := range envFiles {
			p := filepath.Join(target, ef)
			if data, err := os.ReadFile(p); err == nil {
				found = true
				fmt.Printf("[%s]\n", ef)
				lines := strings.Split(string(data), "\n")
				for _, l := range lines {
					trimmed := strings.TrimSpace(l)
					if trimmed == "" || strings.HasPrefix(trimmed, "#") {
						continue
					}
					parts := strings.SplitN(trimmed, "=", 2)
					if len(parts) == 2 {
						fmt.Printf("  %-20s = [set]\n", parts[0])
					}
				}
				fmt.Println()
			}
		}
		if !found {
			fmt.Println("No local .env files detected.")
		}

	case "routes":
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		prof, ok, _ := config.LoadProjectProfile(target)
		if ok && len(prof.Proxy) > 0 {
			fmt.Println("Configured Proxy Routes (.serve/config):")
			for k, v := range prof.Proxy {
				fmt.Printf("  %-16s -> %s\n", k, v)
			}
		} else {
			fmt.Println("No custom proxy routes configured. Serving files from root directory.")
		}

	case "exec", "run":
		if len(args) < 1 {
			fmt.Println("Usage: serve exec [path] <command...>")
			os.Exit(1)
		}
		target := "."
		cmdIdx := 0
		if !strings.Contains(args[0], " ") && !isCommandName(args[0]) {
			target = args[0]
			cmdIdx = 1
		}
		cmdStr := strings.Join(args[cmdIdx:], " ")
		_ = executeCommand(target, cmdStr)

	case "theme":
		handleThemeCommand(getAppName(), palette, args)

	case "accent":
		handleAccentCommand(getAppName(), palette, args)

	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %q. Run 'serve --help' for details.\n", cmd)
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
		fmt.Println("Run 'serve accent list' to view supported accents.")
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

func isCommandName(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func executeCommand(dir, cmdStr string) error {
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

func runDefaultServe(target string, flagList []string, palette *ui.Palette) {
	if palette == nil {
		palette = ui.ResolvePalette(getAppName(), ui.RuntimeOptions{})
	}
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	var (
		portRaw string
		host    string
		lan     bool
		open    bool
		https   bool
		watch   bool
		static  bool
		build   bool
		prod        bool
		proxyS      string
		projectName string
	)

	fs.StringVar(&portRaw, "port", "3000", "Port to listen on or 'auto'")
	fs.StringVar(&portRaw, "p", "3000", "Port (shorthand)")
	fs.StringVar(&host, "host", "", "Host address")
	fs.StringVar(&host, "H", "", "Host (shorthand)")
	fs.BoolVar(&lan, "lan", false, "Bind to 0.0.0.0 and display LAN address")
	fs.BoolVar(&open, "open", false, "Open browser on start")
	fs.BoolVar(&open, "o", false, "Open browser (shorthand)")
	fs.BoolVar(&https, "https", false, "Enable HTTPS with self-signed certificate")
	fs.BoolVar(&https, "s", false, "Enable HTTPS (shorthand)")
	fs.BoolVar(&watch, "watch", false, "Watch files and live reload")
	fs.BoolVar(&watch, "w", false, "Watch files (shorthand)")
	fs.BoolVar(&static, "static", false, "Force native static file serving")
	fs.BoolVar(&build, "build", false, "Run build command before serving")
	fs.BoolVar(&prod, "prod", false, "Run in production/preview mode")
	fs.StringVar(&proxyS, "proxy", "", "Reverse proxy rules (e.g. /api=http://localhost:8000)")
	fs.StringVar(&projectName, "project", "", "Target specific project/workspace in a monorepo")

	_ = fs.Parse(flagList)

	port := 3000
	autoPort := false
	if strings.ToLower(portRaw) == "auto" {
		autoPort = true
	} else if p, err := strconv.Atoi(portRaw); err == nil && p > 0 {
		port = p
	}

	det, err := detector.Detect(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !palette.Quiet && !palette.JSON {
		fmt.Print(ui.RenderBanner(ui.BannerOptions{Command: getAppName(), Palette: palette}))
	}

	// Check for monorepo if target wasn't explicitly a subproject
	ws, _ := monorepo.DetectWorkspace(target)
	if ws != nil && ws.IsMonorepo && len(ws.Projects) > 0 {
		var selected *monorepo.SubProject
		if projectName != "" {
			for _, p := range ws.Projects {
				if strings.EqualFold(p.Name, projectName) || strings.EqualFold(p.RelativePath, projectName) || strings.Contains(strings.ToLower(p.RelativePath), strings.ToLower(projectName)) {
					selected = p
					break
				}
			}
			if selected == nil {
				fmt.Fprintf(os.Stderr, "Project %q not found in %s workspace.\nAvailable projects:\n", projectName, ws.System)
				for _, p := range ws.Projects {
					fmt.Fprintf(os.Stderr, "  - %s (%s)\n", p.Name, p.RelativePath)
				}
				os.Exit(1)
			}
		} else if len(ws.Projects) == 1 {
			selected = ws.Projects[0]
		} else {
			fmt.Printf("Monorepo detected (%s) with %d runnable projects:\n\n", ws.System, len(ws.Projects))
			for i, p := range ws.Projects {
				fmt.Printf("  %d. %-16s %-14s (%s)\n", i+1, p.Name, p.Detection.Type, p.RelativePath)
			}
			fmt.Printf("\nUse 'serve . --project <name>' or 'serve <path>' to select an application.\n")
			selected = ws.Projects[0]
			fmt.Printf("Defaulting to: %s\n\n", selected.Name)
		}

		if selected != nil {
			target = selected.FullPath
			det = selected.Detection
		}
	}

	// If build flag passed and project has a build command
	if build && det.BuildCommand != "" {
		fmt.Printf("Building project: %s\n", det.BuildCommand)
		if err := executeCommand(target, det.BuildCommand); err != nil {
			fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Parse proxy rules from flag and profile
	proxyRules := make(map[string]string)
	if proxyS != "" {
		rules, err := proxy.ParseProxyRules([]string{proxyS})
		if err == nil {
			for k, v := range rules {
				proxyRules[k] = v
			}
		}
	}

	if prof, ok, _ := config.LoadProjectProfile(target); ok {
		for k, v := range prof.Proxy {
			proxyRules[k] = v
		}
	}

	var proxyMgr *proxy.Manager
	if len(proxyRules) > 0 {
		proxyMgr, _ = proxy.NewManager(proxyRules)
	}

	// Decision: Framework Orchestration vs Native Static Serving
	if !static && !det.IsStatic && det.DevCommand != "" && !prod {
		// Framework development server execution
		rtCheck := adapter.CheckRuntime(det.Runtime)
		if !rtCheck.Installed {
			fmt.Fprint(os.Stderr, adapter.MissingRuntimeError(det))
			os.Exit(1)
		}

		// Ensure port available
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
		fmt.Printf("Directory: %s\n\n", det.ProjectName)

		if err := supervisor.SuperviseForeground(det.ProjectName, target, adaptedCmd, port, det.Runtime, det.Framework, extraEnv); err != nil {
			fmt.Fprintf(os.Stderr, "Process exited: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Native Static File Server
	serveTarget := target
	if prof, ok, _ := config.LoadProjectProfile(target); ok && prof.StaticDirectory != "" {
		serveTarget = filepath.Join(target, prof.StaticDirectory)
	} else if prod && hasDir(target, "dist") {
		serveTarget = filepath.Join(target, "dist")
	} else if prod && hasDir(target, "build") {
		serveTarget = filepath.Join(target, "build")
	}

	srv, err := server.NewServer(server.Config{
		Target:       serveTarget,
		Host:         host,
		Port:         port,
		AutoPort:     autoPort || port == 3000,
		LAN:          lan,
		Open:         open,
		HTTPS:        https,
		Watch:        watch,
		ProxyManager: proxyMgr,
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

func hasDir(root, sub string) bool {
	info, err := os.Stat(filepath.Join(root, sub))
	return err == nil && info.IsDir()
}
