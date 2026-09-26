// Package theme provides embedded ANSI color palettes for the report.
package theme

import (
	"os"
	"strings"
)

// Theme is a named ANSI palette.
type Theme struct {
	Name   string
	Border string // box-drawing lines
	Label  string // field names
	Value  string // field values
	Title  string // centered titles
	Accent string // bars / highlights
	Reset  string
}

const reset = "\033[0m"

// built-in palettes
var palettes = map[string]Theme{
	"default": {
		Name: "default", Border: "\033[38;5;245m", Label: "\033[38;5;255m",
		Value: "\033[38;5;252m", Title: "\033[1;38;5;255m", Accent: "\033[38;5;39m", Reset: reset,
	},
	"usgc": {
		Name: "usgc", Border: "\033[38;5;245m", Label: "\033[38;5;255m",
		Value: "\033[38;5;252m", Title: "\033[1;38;5;255m", Accent: "\033[38;5;39m", Reset: reset,
	},
	"dracula": {
		Name: "dracula", Border: "\033[38;2;98;114;164m", Label: "\033[38;2;255;121;198m",
		Value: "\033[38;2;248;248;242m", Title: "\033[1;38;2;189;147;249m", Accent: "\033[38;2;80;250;123m", Reset: reset,
	},
	"nord": {
		Name: "nord", Border: "\033[38;2;76;86;106m", Label: "\033[38;2;136;192;208m",
		Value: "\033[38;2;216;222;233m", Title: "\033[1;38;2;143;188;187m", Accent: "\033[38;2;163;190;140m", Reset: reset,
	},
	"solarized-dark": {
		Name: "solarized-dark", Border: "\033[38;2;88;110;117m", Label: "\033[38;2;38;139;210m",
		Value: "\033[38;2;147;161;161m", Title: "\033[1;38;2;181;137;0m", Accent: "\033[38;2;133;153;0m", Reset: reset,
	},
	"monokai": {
		Name: "monokai", Border: "\033[38;2;117;113;94m", Label: "\033[38;2;249;38;114m",
		Value: "\033[38;2;248;248;242m", Title: "\033[1;38;2;166;226;46m", Accent: "\033[38;2;102;217;239m", Reset: reset,
	},
}

// Names returns sorted built-in theme names.
func Names() []string {
	return []string{"default", "usgc", "dracula", "nord", "solarized-dark", "monokai"}
}

// Lookup resolves a theme by name (case-insensitive). Unknown → default.
func Lookup(name string) Theme {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		key = "default"
	}
	if t, ok := palettes[key]; ok {
		return t
	}
	return palettes["default"]
}

// Resolve picks theme from flag, then MACHINE_REPORT_THEME, then "default".
// Applies NO_COLOR / --no-color and TTY detection: colorless Theme if disabled.
func Resolve(flagTheme string, noColor bool, isTTY bool) Theme {
	name := flagTheme
	if name == "" {
		name = os.Getenv("MACHINE_REPORT_THEME")
	}
	if name == "" {
		name = "default"
	}
	t := Lookup(name)

	envNoColor := os.Getenv("NO_COLOR") != ""
	if noColor || envNoColor || !isTTY {
		return Colorless(t.Name)
	}
	return t
}

// Colorless returns a theme with empty ANSI codes (same Name).
func Colorless(name string) Theme {
	if name == "" {
		name = "default"
	}
	return Theme{Name: name}
}
