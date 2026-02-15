package tui

import "sort"

const (
	ThemeIDBlackbird    = "blackbird"
	ThemeIDHighContrast = "high-contrast"
)

var builtInThemes = map[string]Theme{
	ThemeIDBlackbird: {
		ID: ThemeIDBlackbird,
		Colors: ColorTokens{
			TextPrimary:   "15",
			TextMuted:     "240",
			TextOnAccent:  "15",
			Accent:        "69",
			Info:          "39",
			Success:       "42",
			Warning:       "214",
			Danger:        "196",
			Surface:       "240",
			SurfaceActive: "69",
			SurfaceMuted:  "236",
			Border:        "240",
			BorderActive:  "69",
		},
	},
	ThemeIDHighContrast: {
		ID: ThemeIDHighContrast,
		Colors: ColorTokens{
			TextPrimary:   "15",
			TextMuted:     "250",
			TextOnAccent:  "16",
			Accent:        "51",
			Info:          "87",
			Success:       "118",
			Warning:       "226",
			Danger:        "196",
			Surface:       "16",
			SurfaceActive: "255",
			SurfaceMuted:  "238",
			Border:        "255",
			BorderActive:  "226",
		},
	},
}

// BuiltInThemeIDs returns the built-in theme IDs in deterministic alphabetical order.
func BuiltInThemeIDs() []string {
	ids := make([]string, 0, len(builtInThemes))
	for id := range builtInThemes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ResolveTheme returns a built-in theme for a known theme ID.
// Unknown theme IDs fall back to the blackbird theme.
func ResolveTheme(themeID string) Theme {
	if theme, ok := builtInThemes[themeID]; ok {
		return theme
	}
	return builtInThemes[ThemeIDBlackbird]
}
