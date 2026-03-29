package view

import (
	"fmt"
	"strings"

	"github.com/abdenasser/neohtop-cli/theme"
	"github.com/abdenasser/neohtop-cli/types"
	"charm.land/lipgloss/v2"
)

// Toolbar renders a keyboard shortcut hint bar:
// ╭──────────────────────────────────────────────────────────────────────╮
// │ /Search  f·Filters │ 42/320 procs │ p·Pin i·Info k·Kill  ...       │
// ╰──────────────────────────────────────────────────────────────────────╯
type Toolbar struct {
	theme *theme.Theme
}

func NewToolbar(th *theme.Theme) *Toolbar {
	return &Toolbar{theme: th}
}

func (t *Toolbar) SetTheme(th *theme.Theme) {
	t.theme = th
}

func (t *Toolbar) Render(searchTerm string, searchMode bool, sortCfg types.SortConfig, frozen bool, filtered, total, width, refreshMs int, treeMode bool) string {
	if width < 20 {
		width = 80
	}

	th := t.theme

	// The panel border + padding eats 4 chars (border left + pad + pad + border right)
	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	// ── Style definitions ────────────────────────────────────────────
	numStyle := lipgloss.NewStyle().
		Foreground(th.Lavender).
		Bold(true)

	dim := lipgloss.NewStyle().
		Foreground(th.Overlay0)

	activeStyle := lipgloss.NewStyle().
		Foreground(th.Text).
		Background(th.Surface1).
		Padding(0, 1).
		Bold(true)

	// Button style: pill-shaped with background
	btn := func(text, key string) string {
		pill := lipgloss.NewStyle().
			Background(th.Surface0).
			Foreground(th.Subtext1).
			Padding(0, 1).
			Render(text + " " + lipgloss.NewStyle().Foreground(th.Purple).Bold(true).Render("("+key+")"))
		return pill
	}

	warnBtn := func(text, key string) string {
		pill := lipgloss.NewStyle().
			Background(th.Surface0).
			Foreground(th.Peach).
			Bold(true).
			Padding(0, 1).
			Render(text + " " + lipgloss.NewStyle().Foreground(th.Peach).Bold(true).Render("("+key+")"))
		return pill
	}

	// Emoji-only button for ultra-compact mode
	iconBtn := func(emoji string) string {
		return lipgloss.NewStyle().
			Background(th.Surface0).
			Foreground(th.Subtext1).
			Padding(0, 0).
			Render(emoji)
	}

	// ── Left section: Search + Filters ───────────────────────────────

	var searchBox string
	if searchMode {
		regexHint := lipgloss.NewStyle().Foreground(th.Overlay0).Italic(true)
		input := searchTerm + IconCursor
		if searchTerm == "" {
			input = IconCursor + " " + regexHint.Render("regex: name|pid  ^chrome  \\.log$")
		}
		searchBox = activeStyle.Render(IconSearch + " " + input)
	} else if searchTerm != "" {
		searchBox = activeStyle.Render(IconSearch + " " + searchTerm)
	} else {
		searchBox = btn("🔍 Search", "s")
	}

	filterHint := btn("🧪 Filters", "f")

	left := searchBox + " " + filterHint
	leftLen := lipgloss.Width(left)

	// ── Center: Process count ────────────────────────────────────────

	center := numStyle.Render(fmt.Sprintf("%d", filtered)) +
		dim.Render("/") +
		dim.Render(fmt.Sprintf("%d", total)) +
		dim.Render(" procs")
	centerLen := lipgloss.Width(center)

	// ── Right section: Cols, Pause, Sort, Theme, Help ──

	colHint := btn("📊 Cols", "c")

	var pauseHint string
	if frozen {
		pauseHint = warnBtn("🧊 FROZEN", "Space")
	} else {
		pauseHint = btn("⏸ Pause", "Space")
	}

	// Refresh rate indicator
	rateStr := fmt.Sprintf("%.1fs", float64(refreshMs)/1000.0)
	// Strip trailing 0 for clean display (1.0s → 1s, 0.5s stays)
	if strings.HasSuffix(rateStr, ".0s") {
		rateStr = strings.TrimSuffix(rateStr, ".0s") + "s"
	}
	rateHint := btn(rateStr, "r")

	var treeHint string
	if treeMode {
		treeHint = warnBtn("🌳 Tree", "T")
	} else {
		treeHint = btn("🌳 Tree", "T")
	}

	themeHint := btn("🎨 Theme", "t")
	helpHint := btn("❓ Help", "?")

	right := colHint + " " + treeHint + " " + pauseHint + " " + rateHint + " " + themeHint + " " + helpHint
	rightLen := lipgloss.Width(right)

	// ── Layout assembly ──────────────────────────────────────────────

	totalContent := leftLen + centerLen + rightLen + 4 // 4 = two "  " gaps

	var line string

	if totalContent > innerWidth {
		// Compact: just left + right, no center
		gap := innerWidth - leftLen - rightLen
		if gap < 1 {
			// Super compact: emoji-only buttons
			searchCompact := iconBtn("🔍")
			filterCompact := iconBtn("🧪")
			colCompact := iconBtn("📊")
			pauseCompact := iconBtn("⏸")
			if frozen {
				pauseCompact = lipgloss.NewStyle().Background(th.Surface0).Foreground(th.Peach).Bold(true).Render("🧊")
			}
			themeCompact := iconBtn("🎨")
			helpCompact := iconBtn("❓")
			left = searchCompact + " " + filterCompact
			right = colCompact + " " + pauseCompact + " " + themeCompact + " " + helpCompact
			leftLen = lipgloss.Width(left)
			rightLen = lipgloss.Width(right)
			gap = innerWidth - leftLen - rightLen
			if gap < 1 {
				gap = 1
			}
		}
		line = left + spaces(gap) + right
	} else {
		// Full layout: left ... center ... right
		usedFixed := leftLen + centerLen + rightLen
		remaining := innerWidth - usedFixed
		gap1 := remaining / 2
		gap2 := remaining - gap1
		if gap1 < 0 {
			gap1 = 0
		}
		if gap2 < 0 {
			gap2 = 0
		}
		line = left + spaces(gap1) + center + spaces(gap2) + right
	}

	// Wrap in a bordered panel
	panelStyle := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Surface1).
		Padding(0, 1)

	return panelStyle.Render(line)
}
