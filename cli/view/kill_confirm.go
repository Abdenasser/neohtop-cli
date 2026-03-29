package view

import (
	"fmt"
	"strings"

	"github.com/abdenasser/neohtop-cli/theme"
	"github.com/abdenasser/neohtop-cli/types"
	"charm.land/lipgloss/v2"
)

// KillConfirm renders a minimal kill confirmation dialog
type KillConfirm struct {
	theme *theme.Theme
}

func NewKillConfirm(th *theme.Theme) *KillConfirm {
	return &KillConfirm{theme: th}
}

func (kc *KillConfirm) SetTheme(th *theme.Theme) {
	kc.theme = th
}

func (kc *KillConfirm) Render(p types.Process, width, height int) string {
	th := kc.theme

	hotkey := lipgloss.NewStyle().Foreground(th.Purple).Bold(true)
	dim := lipgloss.NewStyle().Foreground(th.Overlay0)
	danger := lipgloss.NewStyle().Foreground(th.Red).Bold(true)
	sep := lipgloss.NewStyle().Foreground(th.Surface1)
	label := lipgloss.NewStyle().Foreground(th.Subtext0)
	val := lipgloss.NewStyle().Foreground(th.Text)

	contentW := 44

	// Title
	title := danger.Render("☠️  KILL PROCESS")
	close := hotkey.Render("esc") + dim.Render(" cancel")
	gap := contentW - lipgloss.Width(title) - lipgloss.Width(close)
	if gap < 1 {
		gap = 1
	}
	titleLine := title + spaces(gap) + close

	divider := sep.Render(strings.Repeat("─", contentW))

	// Warning message
	warnMsg := danger.Render("⚠️  This will forcefully terminate the process!")

	// Process info
	icon := ProcessIcon(p.Name)
	nameLine := val.Render(icon+" ") + lipgloss.NewStyle().Foreground(th.Text).Bold(true).Render(Truncate(p.Name, contentW-4))
	pidLine := label.Render("PID ") + val.Render(fmt.Sprintf("%d", p.PID))
	cmdLine := label.Render("Cmd ") + dim.Render(Truncate(p.Command, contentW-6))

	// Footer hotkeys
	yHint := danger.Render("y") + dim.Render("/") + danger.Render("Enter") + dim.Render(" confirm")
	nHint := hotkey.Render("n") + dim.Render("/") + hotkey.Render("esc") + dim.Render(" cancel")

	lines := []string{
		titleLine,
		divider,
		"",
		warnMsg,
		"",
		nameLine,
		pidLine,
		cmdLine,
		"",
		divider,
		yHint + "    " + nHint,
	}

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Red).
		Padding(1, 2).
		Width(contentW + 6)

	rendered := box.Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, rendered)
}
