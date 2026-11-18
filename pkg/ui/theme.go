package ui

import (
	"sort"
	"strings"
)

// Theme defines the complete visual personality and color palette for Serve and Pro.
type Theme struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	DefaultAccent string `json:"default_accent"`
	BaseText      string `json:"-"`
	Muted         string `json:"-"`
	Border        string `json:"-"`
	Header        string `json:"-"`
	Success       string `json:"-"`
	Warning       string `json:"-"`
	Error         string `json:"-"`
}

var builtInThemes = map[string]*Theme{
	"default": {
		Name:          "default",
		Description:   "Modern balanced developer palette with cool slate and vibrant accents",
		DefaultAccent: "cyan",
		BaseText:      FgWhite,
		Muted:         FgSlate256,
		Border:        FgSmoke256,
		Header:        Bold + FgBrightWhite,
		Success:       Bold + FgBrightGreen,
		Warning:       Bold + FgBrightYellow,
		Error:         Bold + FgBrightRed,
	},
	"cinematic": {
		Name:          "cinematic",
		Description:   "Dark studio aesthetic with rich copper tones, deep charcoal, and high contrast",
		DefaultAccent: "amber",
		BaseText:      FgBrightWhite,
		Muted:         FgSlate256,
		Border:        FgDark256,
		Header:        Bold + FgAmber256,
		Success:       Bold + FgBrightGreen,
		Warning:       Bold + FgAmber256,
		Error:         Bold + FgBrightRed,
	},
	"ocean": {
		Name:          "ocean",
		Description:   "Deep maritime hues with radiant cyan, teal, and oceanic blues",
		DefaultAccent: "cyan",
		BaseText:      FgBrightWhite,
		Muted:         "\033[38;5;67m",
		Border:        "\033[38;5;24m",
		Header:        Bold + FgTeal256,
		Success:       Bold + FgMint256,
		Warning:       Bold + FgBrightYellow,
		Error:         Bold + FgBrightRed,
	},
	"forest": {
		Name:          "forest",
		Description:   "Natural evergreen palette with vibrant spruce, emerald, and sage",
		DefaultAccent: "green",
		BaseText:      FgBrightWhite,
		Muted:         "\033[38;5;108m",
		Border:        "\033[38;5;29m",
		Header:        Bold + FgForest256,
		Success:       Bold + FgBrightGreen,
		Warning:       Bold + FgAmber256,
		Error:         Bold + FgBrightRed,
	},
	"amber": {
		Name:          "amber",
		Description:   "Warm retro CRT phosphor terminal aesthetic with copper and honey tones",
		DefaultAccent: "amber",
		BaseText:      FgBrightWhite,
		Muted:         "\033[38;5;243m",
		Border:        "\033[38;5;136m",
		Header:        Bold + FgAmber256,
		Success:       Bold + FgBrightGreen,
		Warning:       Bold + FgOrange256,
		Error:         Bold + FgBrightRed,
	},
	"violet": {
		Name:          "violet",
		Description:   "Synthwave twilight aesthetic with neon purple, lavender, and magenta",
		DefaultAccent: "violet",
		BaseText:      FgBrightWhite,
		Muted:         "\033[38;5;103m",
		Border:        "\033[38;5;55m",
		Header:        Bold + FgViolet256,
		Success:       Bold + FgBrightGreen,
		Warning:       Bold + FgBrightYellow,
		Error:         Bold + FgBrightRed,
	},
	"monochrome": {
		Name:          "monochrome",
		Description:   "Pure high-contrast black and white with clean bold typography and zero colored hues",
		DefaultAccent: "white",
		BaseText:      FgWhite,
		Muted:         FgBrightBlack,
		Border:        FgBrightBlack,
		Header:        Bold + FgBrightWhite,
		Success:       Bold + FgWhite,
		Warning:       Bold + FgWhite,
		Error:         Bold + FgWhite,
	},
}

var builtInAccents = map[string]string{
	"cyan":    Bold + FgOcean256,
	"blue":    Bold + FgBrightBlue,
	"green":   Bold + FgForest256,
	"amber":   Bold + FgAmber256,
	"yellow":  Bold + FgBrightYellow,
	"violet":  Bold + FgViolet256,
	"purple":  Bold + FgPurple256,
	"magenta": Bold + FgBrightMagenta,
	"red":     Bold + FgBrightRed,
	"white":   Bold + FgBrightWhite,
}

// GetTheme returns the theme for the given name or the default theme if not found.
func GetTheme(name string) (*Theme, bool) {
	t, ok := builtInThemes[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return builtInThemes["default"], false
	}
	return t, true
}

// GetAccentColor returns the ANSI code for the given accent color name.
func GetAccentColor(accent string) (string, bool) {
	c, ok := builtInAccents[strings.ToLower(strings.TrimSpace(accent))]
	if !ok {
		return builtInAccents["cyan"], false
	}
	return c, true
}

// ListThemes returns all available theme names sorted alphabetically.
func ListThemes() []*Theme {
	list := make([]*Theme, 0, len(builtInThemes))
	for _, t := range builtInThemes {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool {
		// Put "default" first, then alphabetize
		if list[i].Name == "default" {
			return true
		}
		if list[j].Name == "default" {
			return false
		}
		return list[i].Name < list[j].Name
	})
	return list
}

// ListAccents returns all supported accent color names.
func ListAccents() []string {
	keys := make([]string, 0, len(builtInAccents))
	for k := range builtInAccents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
