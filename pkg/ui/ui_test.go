package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"serve/pkg/version"
)

func TestBannerProductIdentity(t *testing.T) {
	products := []struct {
		cmd        string
		wantSubstr string
	}{
		{"serve", "Universal Local Development & Web Serving Orchestrator"},
		{"servecc", "Universal Local Development & Web Serving Orchestrator (.cc edition)"},
		{"pro", "Local Project Management & Workspace Navigation System"},
		{"procc", "Local Project Management & Workspace Navigation System (.cc edition)"},
	}

	for _, p := range products {
		t.Run(p.cmd, func(t *testing.T) {
			palette := ResolvePalette(p.cmd, RuntimeOptions{NoColor: true})
			banner := RenderBanner(BannerOptions{Command: p.cmd, Palette: palette})

			if !strings.Contains(banner, p.wantSubstr) {
				t.Errorf("banner for %s does not contain expected descriptor %q.\nBanner:\n%s", p.cmd, p.wantSubstr, banner)
			}

			// Ensure version synchronization
			if !strings.Contains(banner, version.Current) {
				t.Errorf("banner for %s does not contain version.Current %q.\nBanner:\n%s", p.cmd, version.Current, banner)
			}
		})
	}
}

func TestBannerNoColor(t *testing.T) {
	palette := ResolvePalette("serve", RuntimeOptions{NoColor: true})
	banner := RenderBanner(BannerOptions{Command: "serve", Palette: palette})

	if strings.Contains(banner, "\033[") {
		t.Errorf("expected no ANSI escape sequences in --no-color mode, but found some.\nBanner:\n%s", banner)
	}

	stripped := StripANSI(banner)
	if banner != stripped {
		t.Errorf("expected banner in --no-color mode to match StripANSI output")
	}
}

func TestBannerQuiet(t *testing.T) {
	palette := ResolvePalette("serve", RuntimeOptions{Quiet: true})
	banner := RenderBanner(BannerOptions{Command: "serve", Palette: palette})

	if banner != "" {
		t.Errorf("expected empty banner in --quiet mode, got:\n%s", banner)
	}
}

func TestBannerJSONMode(t *testing.T) {
	palette := ResolvePalette("serve", RuntimeOptions{JSON: true})
	banner := RenderBanner(BannerOptions{Command: "serve", Palette: palette})

	if banner != "" {
		t.Errorf("expected empty banner in --json mode, got:\n%s", banner)
	}
}

func TestAllBuiltInThemes(t *testing.T) {
	themeNames := []string{"default", "cinematic", "ocean", "forest", "amber", "violet", "monochrome"}

	for _, name := range themeNames {
		t.Run(name, func(t *testing.T) {
			theme, ok := GetTheme(name)
			if !ok {
				t.Fatalf("built-in theme %q not found", name)
			}
			if theme.Name != name {
				t.Errorf("theme.Name = %q, want %q", theme.Name, name)
			}

			palette := ResolvePalette("serve", RuntimeOptions{ThemeOverride: name})
			if palette.Theme.Name != name {
				t.Errorf("palette.Theme.Name = %q, want %q", palette.Theme.Name, name)
			}

			banner := RenderBanner(BannerOptions{Command: "serve", Palette: palette})
			if !strings.Contains(banner, name) {
				t.Errorf("banner should mention active theme name %q", name)
			}
		})
	}
}

func TestAllAccents(t *testing.T) {
	accents := ListAccents()
	if len(accents) < 7 {
		t.Errorf("expected at least 7 accents, got %d", len(accents))
	}

	for _, acc := range accents {
		t.Run(acc, func(t *testing.T) {
			palette := ResolvePalette("serve", RuntimeOptions{AccentOverride: acc})
			if palette.AccentName != acc {
				t.Errorf("palette.AccentName = %q, want %q", palette.AccentName, acc)
			}

			banner := RenderBanner(BannerOptions{Command: "serve", Palette: palette})
			if !strings.Contains(banner, acc) {
				t.Errorf("banner should mention active accent name %q", acc)
			}
		})
	}
}

func TestInvalidThemeAndAccentFallback(t *testing.T) {
	theme, ok := GetTheme("non_existent_theme_xyz")
	if ok {
		t.Errorf("expected non-existent theme to return ok=false")
	}
	if theme == nil || theme.Name != "default" {
		t.Errorf("expected fallback to default theme, got %v", theme)
	}

	code, ok := GetAccentColor("non_existent_color_xyz")
	if ok {
		t.Errorf("expected non-existent accent to return ok=false")
	}
	if code == "" {
		t.Errorf("expected fallback accent code, got empty string")
	}
}

func TestConfigPrecedence(t *testing.T) {
	// Setup isolated temp home directory
	tmpHome := t.TempDir()
	origHome := os.Getenv("USERPROFILE")
	if origHome == "" {
		origHome = os.Getenv("HOME")
	}
	os.Setenv("USERPROFILE", tmpHome)
	os.Setenv("HOME", tmpHome)
	defer func() {
		os.Setenv("USERPROFILE", origHome)
		os.Setenv("HOME", origHome)
	}()

	// 1. Initially should be built-in default
	p1 := ResolvePalette("serve", RuntimeOptions{})
	if p1.Theme.Name != "default" || p1.ThemeSource != "default" {
		t.Errorf("p1: want default/default, got %s/%s", p1.Theme.Name, p1.ThemeSource)
	}

	// 2. Set shared .cc config
	ccDir := filepath.Join(tmpHome, ".cc")
	_ = os.MkdirAll(ccDir, 0755)
	_ = os.WriteFile(filepath.Join(ccDir, "theme.json"), []byte(`{"theme": "ocean", "accent": "green"}`), 0644)

	p2 := ResolvePalette("serve", RuntimeOptions{})
	if p2.Theme.Name != "ocean" || !strings.Contains(p2.ThemeSource, ".cc") {
		t.Errorf("p2: want ocean from .cc, got %s/%s", p2.Theme.Name, p2.ThemeSource)
	}
	if p2.AccentName != "green" || !strings.Contains(p2.AccentSource, ".cc") {
		t.Errorf("p2: want green from .cc, got %s/%s", p2.AccentName, p2.AccentSource)
	}

	// 3. Set tool-specific config (~/.serve/theme.json)
	serveDir := filepath.Join(tmpHome, ".serve")
	_ = os.MkdirAll(serveDir, 0755)
	_ = os.WriteFile(filepath.Join(serveDir, "theme.json"), []byte(`{"theme": "forest", "accent": "amber"}`), 0644)

	p3 := ResolvePalette("serve", RuntimeOptions{})
	if p3.Theme.Name != "forest" || !strings.Contains(p3.ThemeSource, ".serve") {
		t.Errorf("p3: want forest from .serve, got %s/%s", p3.Theme.Name, p3.ThemeSource)
	}
	if p3.AccentName != "amber" || !strings.Contains(p3.AccentSource, ".serve") {
		t.Errorf("p3: want amber from .serve, got %s/%s", p3.AccentName, p3.AccentSource)
	}

	// Note that Pro should still pick up the shared .cc config because ~/.pro has no override
	pPro := ResolvePalette("pro", RuntimeOptions{})
	if pPro.Theme.Name != "ocean" {
		t.Errorf("pPro: want ocean from .cc fallback, got %s", pPro.Theme.Name)
	}

	// 4. CLI flag override wins over all
	p4 := ResolvePalette("serve", RuntimeOptions{
		ThemeOverride:  "cinematic",
		AccentOverride: "red",
	})
	if p4.Theme.Name != "cinematic" || !strings.Contains(p4.ThemeSource, "flag") {
		t.Errorf("p4: want cinematic from flag, got %s/%s", p4.Theme.Name, p4.ThemeSource)
	}
	if p4.AccentName != "red" || !strings.Contains(p4.AccentSource, "flag") {
		t.Errorf("p4: want red from flag, got %s/%s", p4.AccentName, p4.AccentSource)
	}
}

func TestHelpRenderers(t *testing.T) {
	p := ResolvePalette("serve", RuntimeOptions{NoColor: true})

	serveHelp := RenderServeHelp("serve", p)
	if !strings.Contains(serveHelp, "PRIMARY COMMANDS") || !strings.Contains(serveHelp, "FLAGS") {
		t.Errorf("RenderServeHelp missing expected sections")
	}

	proHelp := RenderProHelp("pro", p)
	if !strings.Contains(proHelp, "PROJECT MANAGEMENT") || !strings.Contains(proHelp, "EXECUTION & RUNTIME") {
		t.Errorf("RenderProHelp missing expected sections")
	}

	themeList := RenderThemeList("ocean", p)
	if !strings.Contains(themeList, "Available Themes") || !strings.Contains(themeList, "ocean") {
		t.Errorf("RenderThemeList missing expected content")
	}

	accentList := RenderAccentList("cyan", p)
	if !strings.Contains(accentList, "Available Accent Colors") || !strings.Contains(accentList, "cyan") {
		t.Errorf("RenderAccentList missing expected content")
	}
}
