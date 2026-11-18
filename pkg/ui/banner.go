package ui

import (
	"fmt"
	"strings"

	"serve/pkg/version"
)

var bannerArt = map[string][]string{
	"serve": {
		"  ██████  ███████ ██████  ██    ██ ███████",
		"  ██      ██      ██   ██ ██    ██ ██     ",
		"  ███████ █████   ██████  ██    ██ █████  ",
		"       ██ ██      ██   ██  ██  ██  ██     ",
		"  ██████  ███████ ██   ██   ████   ███████",
	},
	"servecc": {
		"  ██████  ███████ ██████  ██    ██ ███████   ██████  ██████ ",
		"  ██      ██      ██   ██ ██    ██ ██       ██      ██      ",
		"  ███████ █████   ██████  ██    ██ █████    ██      ██      ",
		"       ██ ██      ██   ██  ██  ██  ██       ██      ██      ",
		"  ██████  ███████ ██   ██   ████   ███████   ██████  ██████ ",
	},
	"pro": {
		"  ██████  ██████   ██████ ",
		"  ██   ██ ██   ██ ██    ██",
		"  ██████  ██████  ██    ██",
		"  ██      ██   ██ ██    ██",
		"  ██      ██   ██  ██████ ",
	},
	"procc": {
		"  ██████  ██████   ██████    ██████  ██████ ",
		"  ██   ██ ██   ██ ██    ██  ██      ██      ",
		"  ██████  ██████  ██    ██  ██      ██      ",
		"  ██      ██   ██ ██    ██  ██      ██      ",
		"  ██      ██   ██  ██████    ██████  ██████ ",
	},
}

var productDescriptors = map[string]string{
	"serve":   "Universal Local Development & Web Serving Orchestrator",
	"servecc": "Universal Local Development & Web Serving Orchestrator (.cc edition)",
	"pro":     "Local Project Management & Workspace Navigation System",
	"procc":   "Local Project Management & Workspace Navigation System (.cc edition)",
}

// BannerOptions configures banner generation.
type BannerOptions struct {
	Command      string
	Palette      *Palette
	Version      string
	Descriptor   string
	ShowMetadata bool
}

// NormalizeCommand canonicalizes the command name to serve, servecc, pro, or procc.
func NormalizeCommand(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	lower = strings.TrimSuffix(lower, ".exe")
	switch {
	case strings.Contains(lower, "servecc"):
		return "servecc"
	case strings.Contains(lower, "serve"):
		return "serve"
	case strings.Contains(lower, "procc"):
		return "procc"
	case strings.Contains(lower, "pro"):
		return "pro"
	default:
		return "serve"
	}
}

// RenderBanner generates the styled, unmistakable command banner for the active product.
func RenderBanner(opts BannerOptions) string {
	p := opts.Palette
	if p == nil {
		p = ResolvePalette(opts.Command, RuntimeOptions{})
	}

	// Suppression check: quiet mode or JSON mode must never emit banner
	if p.Quiet || p.JSON {
		return ""
	}

	cmdKey := NormalizeCommand(opts.Command)
	lines, ok := bannerArt[cmdKey]
	if !ok {
		lines = bannerArt["serve"]
	}

	desc := opts.Descriptor
	if desc == "" {
		desc = productDescriptors[cmdKey]
	}

	ver := opts.Version
	if ver == "" {
		ver = version.Current
	}

	var sb strings.Builder
	sb.WriteString("\n")

	// 1. Render ASCII Art Typography
	for _, l := range lines {
		sb.WriteString(p.Accent(l))
		sb.WriteString("\n")
	}

	// 2. Subtitle / Product Descriptor
	sb.WriteString("\n")
	sb.WriteString("  ")
	sb.WriteString(p.Bold(desc))
	sb.WriteString("\n")

	// 3. Metadata & Rule
	themeName := "default"
	if p.Theme != nil {
		themeName = p.Theme.Name
	}

	meta := fmt.Sprintf("  v%s │ theme: %s (%s) │ .cc visual system",
		ver,
		themeName,
		p.AccentName,
	)

	// Format metadata with subtle accents
	if p.ColorEnabled {
		metaStyled := fmt.Sprintf("  %s %s %s: %s (%s) %s %s",
			p.Bold(fmt.Sprintf("v%s", ver)),
			p.Border("│"),
			p.Muted("theme"),
			p.Accent(themeName),
			p.Accent(p.AccentName),
			p.Border("│"),
			p.Muted(".cc visual system"),
		)
		sb.WriteString(metaStyled)
	} else {
		sb.WriteString(meta)
	}
	sb.WriteString("\n")

	// 4. Accent-infused Divider line
	dividerWidth := 68
	divider := strings.Repeat("─", dividerWidth)
	sb.WriteString("  ")
	sb.WriteString(p.Border(divider))
	sb.WriteString("\n\n")

	return sb.String()
}
