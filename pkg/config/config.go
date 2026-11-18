package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// ProjectProfile represents local configuration stored in `.serve/config` or `.serve/config.json`.
type ProjectProfile struct {
	Runtime         string            `json:"runtime,omitempty"`
	Framework       string            `json:"framework,omitempty"`
	DevCommand      string            `json:"dev_command,omitempty"`
	BuildCommand    string            `json:"build_command,omitempty"`
	StartCommand    string            `json:"start_command,omitempty"`
	PreviewCommand  string            `json:"preview_command,omitempty"`
	Port            int               `json:"port,omitempty"`
	Host            string            `json:"host,omitempty"`
	Environment     map[string]string `json:"environment,omitempty"`
	WatchPaths      []string          `json:"watch_paths,omitempty"`
	IgnorePaths     []string          `json:"ignore_paths,omitempty"`
	Proxy           map[string]string `json:"proxy,omitempty"`
	StaticDirectory string            `json:"static_directory,omitempty"`
}

// GlobalConfig represents system-wide user preferences in $LOCALAPPDATA/serve/config.json.
type GlobalConfig struct {
	DefaultPort int               `json:"default_port"`
	DefaultHost string            `json:"default_host"`
	AutoPort    bool              `json:"auto_port"`
	Settings    map[string]string `json:"settings"`
}

var (
	globalMu sync.RWMutex
)

// GlobalConfigPath returns the path to global config.json
func GlobalConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "serve", "config.json")
}

// LoadGlobalConfig loads global settings or defaults.
func LoadGlobalConfig() (*GlobalConfig, error) {
	globalMu.RLock()
	defer globalMu.RUnlock()

	p := GlobalConfigPath()
	cfg := &GlobalConfig{
		DefaultPort: 3000,
		DefaultHost: "127.0.0.1",
		AutoPort:    true,
		Settings:    make(map[string]string),
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	_ = json.Unmarshal(data, cfg)
	if cfg.Settings == nil {
		cfg.Settings = make(map[string]string)
	}
	return cfg, nil
}

// SaveGlobalConfig saves global settings.
func SaveGlobalConfig(cfg *GlobalConfig) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	p := GlobalConfigPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// LoadProjectProfile checks for .serve/config or .serve/config.json in projectDir.
func LoadProjectProfile(projectDir string) (*ProjectProfile, bool, error) {
	jsonPath := filepath.Join(projectDir, ".serve", "config.json")
	if data, err := os.ReadFile(jsonPath); err == nil {
		var prof ProjectProfile
		if err := json.Unmarshal(data, &prof); err == nil {
			return &prof, true, nil
		}
	}

	textPath := filepath.Join(projectDir, ".serve", "config")
	f, err := os.Open(textPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer f.Close()

	prof := &ProjectProfile{
		Environment: make(map[string]string),
		Proxy:       make(map[string]string),
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		rawKey := strings.TrimSpace(parts[0])
		k := strings.ToLower(rawKey)
		v := strings.TrimSpace(parts[1])
		v = strings.Trim(v, `"'`)

		switch k {
		case "runtime":
			prof.Runtime = v
		case "framework":
			prof.Framework = v
		case "dev", "dev_command", "dev command":
			prof.DevCommand = v
		case "build", "build_command", "build command":
			prof.BuildCommand = v
		case "start", "start_command", "start command":
			prof.StartCommand = v
		case "preview", "preview_command", "preview command":
			prof.PreviewCommand = v
		case "port":
			if p, err := strconv.Atoi(v); err == nil {
				prof.Port = p
			}
		case "host":
			prof.Host = v
		case "static", "static_directory", "static directory":
			prof.StaticDirectory = v
		case "watch", "watch_paths", "watch paths":
			for _, wp := range strings.Split(v, ",") {
				if t := strings.TrimSpace(wp); t != "" {
					prof.WatchPaths = append(prof.WatchPaths, t)
				}
			}
		case "ignore", "ignore_paths", "ignore paths":
			for _, ip := range strings.Split(v, ",") {
				if t := strings.TrimSpace(ip); t != "" {
					prof.IgnorePaths = append(prof.IgnorePaths, t)
				}
			}
		case "proxy":
			pxParts := strings.SplitN(v, "=", 2)
			if len(pxParts) == 2 {
				prof.Proxy[strings.TrimSpace(pxParts[0])] = strings.TrimSpace(pxParts[1])
			}
		default:
			if strings.HasPrefix(k, "env.") {
				envKey := rawKey[4:]
				prof.Environment[envKey] = v
			}
		}
	}

	return prof, true, nil
}

// SaveProjectProfile saves a profile to .serve/config.json
func SaveProjectProfile(projectDir string, prof *ProjectProfile) error {
	dir := filepath.Join(projectDir, ".serve")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(prof, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
}

// FormatProfile formats profile details for display
func (p *ProjectProfile) Format() string {
	var sb strings.Builder
	if p.Runtime != "" {
		fmt.Fprintf(&sb, "Runtime:          %s\n", p.Runtime)
	}
	if p.Framework != "" {
		fmt.Fprintf(&sb, "Framework:        %s\n", p.Framework)
	}
	if p.DevCommand != "" {
		fmt.Fprintf(&sb, "Dev Command:      %s\n", p.DevCommand)
	}
	if p.BuildCommand != "" {
		fmt.Fprintf(&sb, "Build Command:    %s\n", p.BuildCommand)
	}
	if p.PreviewCommand != "" {
		fmt.Fprintf(&sb, "Preview Command:  %s\n", p.PreviewCommand)
	}
	if p.Port > 0 {
		fmt.Fprintf(&sb, "Port:             %d\n", p.Port)
	}
	if p.Host != "" {
		fmt.Fprintf(&sb, "Host:             %s\n", p.Host)
	}
	if p.StaticDirectory != "" {
		fmt.Fprintf(&sb, "Static Directory: %s\n", p.StaticDirectory)
	}
	if len(p.Proxy) > 0 {
		var prx []string
		for k, v := range p.Proxy {
			prx = append(prx, fmt.Sprintf("%s -> %s", k, v))
		}
		fmt.Fprintf(&sb, "Proxy Rules:      %s\n", strings.Join(prx, ", "))
	}
	return sb.String()
}
