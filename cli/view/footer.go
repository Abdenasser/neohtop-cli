package view

import (
	"fmt"
	"strings"

	"github.com/abdenasser/neohtop-cli/theme"
	"github.com/abdenasser/neohtop-cli/types"
	"charm.land/lipgloss/v2"
)

type Footer struct {
	theme *theme.Theme
}

func NewFooter(th *theme.Theme) *Footer {
	return &Footer{theme: th}
}

func (f *Footer) SetTheme(th *theme.Theme) {
	f.theme = th
}

func (f *Footer) Render(stats types.SystemStats, selectedPID int, selectedName string, hasSelection, isPinned bool, width int) string {
	th := f.theme
	dim := lipgloss.NewStyle().Foreground(th.Overlay0)
	val := lipgloss.NewStyle().Foreground(th.Subtext1)
	accent := lipgloss.NewStyle().Foreground(th.Purple)
	sep := dim.Render(" · ")

	// Left: hostname + OS
	left := ""
	if stats.Hostname != "" {
		left += accent.Render("🏠 " + stats.Hostname)
	}
	if stats.OSVersion != "" {
		left += sep + val.Render(stats.OSVersion)
	}
	if stats.KernelVersion != "" {
		left += sep + val.Render(stats.KernelVersion)
	}

	// Right: selected process info + action buttons
	var right string
	if hasSelection && selectedPID > 0 {
		pidInfo := accent.Render("▸ ") + val.Render(fmt.Sprintf("PID %d", selectedPID)) + sep + val.Render(selectedName)

		// Action buttons for selected process
		btnStyle := lipgloss.NewStyle().
			Background(th.Surface0).
			Foreground(th.Subtext1).
			Padding(0, 1)
		keyStyle := lipgloss.NewStyle().Foreground(th.Purple).Bold(true)
		dangerKeyStyle := lipgloss.NewStyle().Foreground(th.Red).Bold(true)

		var pinBtn string
		if isPinned {
			pinBtn = btnStyle.Render("📌 Unpin " + keyStyle.Render("(u)"))
		} else {
			pinBtn = btnStyle.Render("📌 Pin " + keyStyle.Render("(p)"))
		}
		infoBtn := btnStyle.Render("ℹ️ Info " + keyStyle.Render("(i)"))
		killBtn := lipgloss.NewStyle().
			Background(th.Surface0).
			Foreground(th.Red).
			Padding(0, 1).
			Render("☠️ Kill " + dangerKeyStyle.Render("(k)"))

		right = pidInfo + "  " + pinBtn + " " + infoBtn + " " + killBtn
	} else {
		right = val.Render(fmt.Sprintf("%d processes", stats.ProcessCount))
	}

	// Layout
	innerW := width - 4
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := innerW - leftW - rightW
	if gap < 1 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + right

	style := lipgloss.NewStyle().
		Foreground(th.Subtext0).
		Padding(0, 1).
		Width(width)

	return style.Render(line)
}
