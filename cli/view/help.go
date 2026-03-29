package view

import (
	"strings"

	"github.com/abdenasser/neohtop-cli/theme"
	"charm.land/lipgloss/v2"
)

// Help renders the keybinding help overlay
type Help struct {
	theme *theme.Theme
}

func NewHelp(th *theme.Theme) *Help {
	return &Help{theme: th}
}

func (h *Help) SetTheme(th *theme.Theme) {
	h.theme = th
}

func (h *Help) Render(width, height int) string {
	th := h.theme

	// Shared styles
	hotkey := lipgloss.NewStyle().Foreground(th.Purple).Bold(true)
	dim := lipgloss.NewStyle().Foreground(th.Overlay0)
	section := lipgloss.NewStyle().Foreground(th.Lavender).Bold(true)
	key := lipgloss.NewStyle().Foreground(th.Peach)
	desc := lipgloss.NewStyle().Foreground(th.Subtext1)
	sep := lipgloss.NewStyle().Foreground(th.Surface1)

	// Title bar
	title := lipgloss.NewStyle().Foreground(th.Purple).Bold(true).Render("Keyboard Reference")
	close := hotkey.Render("esc") + dim.Render(" close")

	contentW := 62
	gap := contentW - lipgloss.Width(title) - lipgloss.Width(close)
	if gap < 1 {
		gap = 1
	}
	titleLine := title + spaces(gap) + close

	bind := func(k, d string) string {
		return key.Render(PadRight(k, 14)) + desc.Render(d)
	}

	divider := sep.Render(strings.Repeat("─", contentW))

	// Two-column layout: left = General/Nav/Actions, right = Sort/Display/Mouse
	leftLines := []string{
		section.Render("⚙️ General"),
		bind("q  Ctrl+c", "Quit"),
		bind("?", "This help"),
		bind("Space", "Pause/resume"),
		bind("s  /", "Search (regex)"),
		bind("Esc", "Close/clear"),
		bind("t", "Themes"),
		"",
		section.Render("🧭 Navigation"),
		bind("↑ ↓  j", "Move selection"),
		bind("PgUp PgDn", "Scroll fast"),
		bind("Home g", "Jump to top"),
		bind("End  G", "Jump to bottom"),
		"",
		section.Render("⚡ Process Actions"),
		bind("i  Enter", "Details"),
		bind("k  x  Del", "Kill"),
		bind("p", "Pin/unpin"),
	}

	rightLines := []string{
		section.Render("🔢 Sorting"),
		bind("0-9", "Sort by col N"),
		"",
		section.Render("🖥️ Display"),
		bind("f", "Filters"),
		bind("c", "Columns"),
		bind("T", "Tree view"),
		bind("r", "Refresh rate"),
		"",
		section.Render("🖱️ Mouse"),
		bind("Click", "Select row"),
		bind("Double-click", "Details"),
		bind("Header click", "Sort"),
		bind("Scroll", "Scroll list"),
	}

	// Pad columns to equal height
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, "")
	}

	// Build two-column rows
	colW := (contentW - 3) / 2 // -3 for " │ " divider
	colSep := dim.Render(" " + IconSep + " ")
	var rows []string
	for i := range leftLines {
		l := leftLines[i]
		r := rightLines[i]
		// Pad each side to colW
		lPad := colW - lipgloss.Width(l)
		if lPad < 0 {
			lPad = 0
		}
		rows = append(rows, l+spaces(lPad)+colSep+r)
	}

	var lines []string
	lines = append(lines, titleLine)
	lines = append(lines, divider)
	lines = append(lines, rows...)

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Surface1).
		Padding(1, 2).
		Width(contentW + 6) // +6 for padding

	rendered := box.Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, rendered)
}
