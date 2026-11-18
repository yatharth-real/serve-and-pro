package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileConfig holds serialized theme and accent preferences.
type FileConfig struct {
	Theme  string `json:"theme,omitempty"`
	Accent string `json:"accent,omitempty"`
}

// RuntimeOptions captures invocation-level visual overrides.
type RuntimeOptions struct {
	ThemeOverride  string
	AccentOverride string
	NoColor        bool
	Quiet          bool
	JSON           bool
}

// Palette provides active styling and color formatting based on resolved theme and terminal mode.
type Palette struct {
	Theme         *Theme
	AccentName    string
	AccentCode    string
	ColorEnabled  bool
	Quiet         bool
	JSON          bool
	ThemeSource   string
	AccentSource  string
	ConfiguredDir string
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// GetSharedCCPath returns the path to ~/.cc/theme.json
func GetSharedCCPath() string {
	home := getHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".cc", "theme.json")
}

// GetToolPath returns the path to ~/.<tool>/theme.json
func GetToolPath(tool string) string {
	home := getHomeDir()
	if home == "" {
		return ""
	}
	toolDir := strings.ToLower(strings.TrimSpace(tool))
	if strings.Contains(toolDir, "pro") {
		toolDir = "pro"
	} else {
		toolDir = "serve"
	}
	return filepath.Join(home, "."+toolDir, "theme.json")
}

func readFileConfig(path string) (*FileConfig, bool) {
	if path == "" {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, false
	}
	return &cfg, true
}

// ResolvePalette computes the active Palette using strict deterministic precedence:
// 1. CLI flag override
// 2. Tool-specific config (~/.serve/theme.json or ~/.pro/theme.json)
// 3. Shared ~/.cc/theme.json config
// 4. Built-in default
func ResolvePalette(tool string, opts RuntimeOptions) *Palette {
	themeName := "default"
	accentName := ""
	themeSource := "default"
	accentSource := "default"

	// 3. Shared .cc config
	if ccCfg, ok := readFileConfig(GetSharedCCPath()); ok {
		if ccCfg.Theme != "" {
			themeName = ccCfg.Theme
			themeSource = "~/.cc/theme.json"
		}
		if ccCfg.Accent != "" {
			accentName = ccCfg.Accent
			accentSource = "~/.cc/theme.json"
		}
	}

	// 2. Tool-specific config
	toolPath := GetToolPath(tool)
	if toolCfg, ok := readFileConfig(toolPath); ok {
		if toolCfg.Theme != "" {
			themeName = toolCfg.Theme
			themeSource = fmt.Sprintf("~/.%s/theme.json", filepath.Base(filepath.Dir(toolPath))[1:])
		}
		if toolCfg.Accent != "" {
			accentName = toolCfg.Accent
			accentSource = fmt.Sprintf("~/.%s/theme.json", filepath.Base(filepath.Dir(toolPath))[1:])
		}
	}

	// 1. CLI override
	if opts.ThemeOverride != "" {
		themeName = opts.ThemeOverride
		themeSource = "flag (--theme)"
	}
	if opts.AccentOverride != "" {
		accentName = opts.AccentOverride
		accentSource = "flag (--accent)"
	}

	theme, _ := GetTheme(themeName)
	if accentName == "" {
		accentName = theme.DefaultAccent
	}

	accentCode, _ := GetAccentColor(accentName)

	colorEnabled := true
	if opts.NoColor || opts.JSON || !SupportsColor() {
		colorEnabled = false
	}

	return &Palette{
		Theme:        theme,
		AccentName:   accentName,
		AccentCode:   accentCode,
		ColorEnabled: colorEnabled,
		Quiet:        opts.Quiet,
		JSON:         opts.JSON,
		ThemeSource:  themeSource,
		AccentSource: accentSource,
	}
}

// SaveThemeConfig persists theme and/or accent preference to either tool-specific or shared .cc config.
func SaveThemeConfig(tool string, global bool, theme, accent string) error {
	var targetFile string
	if global {
		targetFile = GetSharedCCPath()
	} else {
		targetFile = GetToolPath(tool)
	}

	if targetFile == "" {
		return fmt.Errorf("unable to determine home directory for configuration")
	}

	if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	cfg := &FileConfig{}
	if existing, ok := readFileConfig(targetFile); ok {
		cfg = existing
	}

	if theme != "" {
		cfg.Theme = strings.ToLower(strings.TrimSpace(theme))
	}
	if accent != "" {
		cfg.Accent = strings.ToLower(strings.TrimSpace(accent))
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(targetFile, data, 0644)
}

// Style helpers

func (p *Palette) Apply(ansiCode, text string) string {
	if !p.ColorEnabled || ansiCode == "" || text == "" {
		return text
	}
	return ansiCode + text + Reset
}

func (p *Palette) Accent(text string) string {
	return p.Apply(p.AccentCode, text)
}

func (p *Palette) Header(text string) string {
	if p.Theme != nil && p.Theme.Header != "" {
		return p.Apply(p.Theme.Header, text)
	}
	return p.Apply(Bold+FgBrightWhite, text)
}

func (p *Palette) Muted(text string) string {
	if p.Theme != nil && p.Theme.Muted != "" {
		return p.Apply(p.Theme.Muted, text)
	}
	return p.Apply(Dim, text)
}

func (p *Palette) Border(text string) string {
	if p.Theme != nil && p.Theme.Border != "" {
		return p.Apply(p.Theme.Border, text)
	}
	return p.Apply(Dim, text)
}

func (p *Palette) Success(text string) string {
	if p.Theme != nil && p.Theme.Success != "" {
		return p.Apply(p.Theme.Success, text)
	}
	return p.Apply(Bold+FgBrightGreen, text)
}

func (p *Palette) Warning(text string) string {
	if p.Theme != nil && p.Theme.Warning != "" {
		return p.Apply(p.Theme.Warning, text)
	}
	return p.Apply(Bold+FgBrightYellow, text)
}

func (p *Palette) Error(text string) string {
	if p.Theme != nil && p.Theme.Error != "" {
		return p.Apply(p.Theme.Error, text)
	}
	return p.Apply(Bold+FgBrightRed, text)
}

func (p *Palette) Bold(text string) string {
	return p.Apply(Bold, text)
}
