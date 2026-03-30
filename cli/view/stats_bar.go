package view

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/abdenasser/neohtop-cli/theme"
	"github.com/abdenasser/neohtop-cli/types"
	"charm.land/lipgloss/v2"
)

// StatsBar renders system metrics panels using braille dot-matrix bars
// with btop-inspired sparkline history graphs and embedded border titles.
//
// Layout: [CPU (flex 2.5)] [Memory (flex 2)] [Storage+System (flex 1.2)] [Network (flex 1.2)]
type StatsBar struct {
	theme *theme.Theme

	// CPU history — per-core sparkline ring buffers
	cpuHistory    []*SparklineHistory
	cpuAvgHistory *SparklineHistory

	// Network history
	rxHistory *SparklineHistory
	txHistory *SparklineHistory

	// Memory history
	memHistory *SparklineHistory

	// Network session stats
	sessionRxTotal uint64
	sessionTxTotal uint64
	peakRx         uint64
	peakTx         uint64

	// Track sparkline width so we can resize if terminal changes
	lastSparkWidth int
}

const defaultSparkWidth = 40

func NewStatsBar(th *theme.Theme) *StatsBar {
	return &StatsBar{
		theme:          th,
		cpuAvgHistory:  NewSparklineHistory(defaultSparkWidth),
		rxHistory:      NewSparklineHistory(defaultSparkWidth),
		txHistory:      NewSparklineHistory(defaultSparkWidth),
		memHistory:     NewSparklineHistory(defaultSparkWidth),
		lastSparkWidth: defaultSparkWidth,
	}
}

func (s *StatsBar) SetTheme(th *theme.Theme) {
	s.theme = th
}

// RecordStats pushes current values into history buffers. Call this every tick.
func (s *StatsBar) RecordStats(stats types.SystemStats) {
	// Initialize per-core histories on first call or if core count changed
	if len(s.cpuHistory) != len(stats.CPUUsage) {
		s.cpuHistory = make([]*SparklineHistory, len(stats.CPUUsage))
		for i := range s.cpuHistory {
			s.cpuHistory[i] = NewSparklineHistory(s.lastSparkWidth)
		}
	}

	// Record per-core CPU
	for i, usage := range stats.CPUUsage {
		s.cpuHistory[i].Push(float64(usage))
	}

	// Record average
	var avg float64
	for _, c := range stats.CPUUsage {
		avg += float64(c)
	}
	if len(stats.CPUUsage) > 0 {
		avg /= float64(len(stats.CPUUsage))
	}
	s.cpuAvgHistory.Push(avg)

	// Record memory
	memPct := float64(0)
	if stats.MemoryTotal > 0 {
		memPct = float64(stats.MemoryUsed) / float64(stats.MemoryTotal) * 100
	}
	s.memHistory.Push(memPct)

	// Track peaks first (before normalizing)
	if stats.NetworkRxBytes > s.peakRx {
		s.peakRx = stats.NetworkRxBytes
	}
	if stats.NetworkTxBytes > s.peakTx {
		s.peakTx = stats.NetworkTxBytes
	}

	// Record network — normalize relative to peak so sparkline shows variation
	s.rxHistory.Push(normalizeRelative(stats.NetworkRxBytes, s.peakRx))
	s.txHistory.Push(normalizeRelative(stats.NetworkTxBytes, s.peakTx))

	// Track session cumulative totals
	s.sessionRxTotal += stats.NetworkRxBytes
	s.sessionTxTotal += stats.NetworkTxBytes
}

// normalizeRelative maps a value to 0-100 relative to the observed peak.
// This ensures the sparkline shows meaningful vertical variation.
func normalizeRelative(value, peak uint64) float64 {
	if peak == 0 || value == 0 {
		return 0
	}
	return float64(value) / float64(peak) * 100
}

// networkNormalize maps byte rates to 0-100 on a log scale for sparkline display
func networkNormalize(bytes uint64) float64 {
	if bytes == 0 {
		return 0
	}
	// Log scale: 1KB=10, 10KB=25, 100KB=40, 1MB=55, 10MB=70, 100MB=85, 1GB=100
	val := float64(bytes)
	if val < 1024 {
		return val / 1024 * 10
	}
	// Compress into 10-100 range using powers of 1024
	tier := 0.0
	for val > 1024 && tier < 6 {
		val /= 1024
		tier++
	}
	result := 10 + tier*15 + (val/1024)*15
	if result > 100 {
		result = 100
	}
	return result
}

func (s *StatsBar) Render(stats types.SystemStats, width int) string {
	if width < 40 {
		width = 80
	}

	// ── Responsive layout ──────────────────────────────────────────
	// Wide (≥180): 4 panels in a row  [cpu][mem][info][net]
	// Medium (≥90): 2×2 grid          [cpu][mem] / [info][net]
	// Narrow (<90): compact one-liner
	gap := 1
	spacer := strings.Repeat(" ", gap)
	th := s.theme
	borderFg := th.Surface1

	type panelSpec struct {
		render func(types.SystemStats, int) string
		title  string
		icon   string
		accent color.Color
		flex   float64
	}
	allPanels := []panelSpec{
		{s.renderCPU, "cpu", "🚀", th.Purple, 2.5},
		{s.renderMemory, "mem", "💾", th.Blue, 2.0},
		{s.renderInfo, "info", "ℹ️", th.Green, 1.2},
		{s.renderNetwork, "net", "🌐", th.Peach, 1.2},
	}

	// Determine layout mode
	type layoutMode int
	const (
		layoutWide   layoutMode = iota // 4 across
		layoutGrid                     // 2×2
		layoutCompact                  // single line
	)

	mode := layoutWide
	if width < 180 {
		mode = layoutGrid
	}
	if width < 90 {
		mode = layoutCompact
	}

	if mode == layoutCompact {
		return s.renderCompact(stats, width)
	}

	// ── Calculate panel widths based on layout mode ──
	var rows [][]int // indices into allPanels per row

	if mode == layoutWide {
		rows = [][]int{{0, 1, 2, 3}}
	} else {
		// 2×2 grid: top row = cpu + mem, bottom row = info + net
		rows = [][]int{{0, 1}, {2, 3}}
	}

	// Calculate widths for each row
	type sized struct {
		idx   int
		width int
	}
	rowSized := make([][]sized, len(rows))

	for ri, row := range rows {
		numGaps := len(row) - 1
		availWidth := width - (numGaps * gap)
		totalFlex := 0.0
		for _, pi := range row {
			totalFlex += allPanels[pi].flex
		}
		rowSized[ri] = make([]sized, len(row))
		used := 0
		for ci, pi := range row {
			w := int(float64(availWidth) * allPanels[pi].flex / totalFlex)
			if ci == len(row)-1 {
				w = availWidth - used // remainder goes to last panel
			}
			rowSized[ri][ci] = sized{pi, w}
			used += w
		}
	}

	// Resize sparkline histories based on CPU panel width
	cpuW := rowSized[0][0].width
	sparkW := cpuW - 12
	if sparkW < 8 {
		sparkW = 8
	}
	if sparkW != s.lastSparkWidth {
		s.resizeHistories(sparkW)
		s.lastSparkWidth = sparkW
	}

	// ── Render each row of panels ──
	var rowStrings []string

	for _, row := range rowSized {
		// Render panel contents
		rendered := make([]string, len(row))
		maxH := 0
		for ci, s2 := range row {
			p := allPanels[s2.idx]
			content := p.render(stats, s2.width-4) // subtract border+padding
			rendered[ci] = s.renderBtopPanel(content, p.icon, p.title, s2.width, 0, borderFg, p.accent)
			h := lipgloss.Height(rendered[ci])
			if h > maxH {
				maxH = h
			}
		}

		// Re-render with equal height within this row
		for ci, s2 := range row {
			p := allPanels[s2.idx]
			content := p.render(stats, s2.width-4)
			rendered[ci] = s.renderBtopPanel(content, p.icon, p.title, s2.width, maxH-4, borderFg, p.accent)
		}

		// Join panels horizontally
		parts := make([]string, 0, len(rendered)*2-1)
		for i, r := range rendered {
			if i > 0 {
				parts = append(parts, spacer)
			}
			parts = append(parts, r)
		}
		rowStrings = append(rowStrings, lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	}

	panelsBlock := strings.Join(rowStrings, "\n")

	// ── Ghost emoji branding centered above panels with gradient ──
	brandColors := []color.Color{th.Purple, th.Purple, th.Pink, th.Pink, th.Fuchsia, th.Fuchsia, th.Indigo}
	brandText := "NeoHtop"
	var brandChars string
	for i, ch := range brandText {
		c := brandColors[i%len(brandColors)]
		brandChars += lipgloss.NewStyle().Foreground(c).Bold(true).Render(string(ch))
	}
	cli := lipgloss.NewStyle().
		Foreground(th.Overlay0).
		Render(" CLI")
	brandStr := "👻 " + brandChars + cli
	brandW := lipgloss.Width(brandStr)
	pad := (width - brandW) / 2
	if pad < 0 {
		pad = 0
	}
	brandLine := strings.Repeat(" ", pad) + brandStr

	return brandLine + "\n" + panelsBlock
}

// renderBtopPanel creates a panel with the title embedded in the top border
// like btop: ╭─ cpu ──────────────────╮
//
// Width budget:  │ + space + content(contentW) + space + │
//   where contentW = width - 4  (2 border chars + 2 padding chars)
//
// The `content` string must already be rendered at contentW width.
func (s *StatsBar) renderBtopPanel(content, icon, title string, width, height int, borderFg, accent color.Color) string {
	// innerW = space between the two border columns (│)
	innerW := width - 2
	if innerW < 4 {
		innerW = 4
	}
	// contentW = innerW - 2 padding chars (1 each side)
	contentW := innerW - 2

	titleStr := " " + icon + " " + title + " "
	titleRendered := lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Render(titleStr)
	titleW := lipgloss.Width(titleRendered)

	borderStyle := lipgloss.NewStyle().Foreground(borderFg)

	// Top border: ╭─ title ──────╮
	// Total = ╭(1) + ─(1) + title(titleW) + ─×N + ╮(1) = width
	// So N = width - 3 - titleW = innerW - 1 - titleW
	topRulerLen := innerW - 1 - titleW
	if topRulerLen < 1 {
		topRulerLen = 1
	}
	topBorder := borderStyle.Render("╭─") + titleRendered + borderStyle.Render(strings.Repeat("─", topRulerLen)+"╮")

	// Bottom border: ╰──────────────╯
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", innerW) + "╯")

	// Split content into lines and pad/truncate each to exactly contentW
	contentLines := strings.Split(content, "\n")

	// If height is specified, pad or truncate to that many lines
	if height > 0 && len(contentLines) < height {
		for len(contentLines) < height {
			contentLines = append(contentLines, "")
		}
	}

	emptyRow := borderStyle.Render("│") + " " + strings.Repeat(" ", contentW) + " " + borderStyle.Render("│") + "\n"

	var body strings.Builder
	body.WriteString(emptyRow) // top padding
	for _, line := range contentLines {
		lineW := lipgloss.Width(line)
		pad := contentW - lineW
		if pad < 0 {
			pad = 0
		}
		// │ + space(padding) + content + space-fill + space(padding) + │
		body.WriteString(borderStyle.Render("│") + " " + line + strings.Repeat(" ", pad) + " " + borderStyle.Render("│") + "\n")
	}
	body.WriteString(emptyRow) // bottom padding

	return topBorder + "\n" + body.String() + bottomBorder
}

// resizeHistories adjusts sparkline widths when terminal resizes
func (s *StatsBar) resizeHistories(newWidth int) {
	s.cpuAvgHistory.Resize(newWidth)
	s.memHistory.Resize(newWidth)
	s.rxHistory.Resize(newWidth)
	s.txHistory.Resize(newWidth)
	for _, h := range s.cpuHistory {
		h.Resize(newWidth)
	}
}

// renderCompact renders a single-line summary when terminal is narrow
func (s *StatsBar) renderCompact(stats types.SystemStats, width int) string {
	titleStyle := lipgloss.NewStyle().Foreground(s.theme.Purple).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(s.theme.Text)
	labelStyle := lipgloss.NewStyle().Foreground(s.theme.Subtext0)

	var avg float32
	for _, c := range stats.CPUUsage {
		avg += c
	}
	if len(stats.CPUUsage) > 0 {
		avg /= float32(len(stats.CPUUsage))
	}

	memPct := float32(0)
	if stats.MemoryTotal > 0 {
		memPct = float32(stats.MemoryUsed) / float32(stats.MemoryTotal) * 100
	}

	parts := []string{
		titleStyle.Render("🚀 ") + valStyle.Render(fmt.Sprintf("%.0f%%", avg)),
		titleStyle.Render("💾 ") + valStyle.Render(fmt.Sprintf("%.0f%%", memPct)),
		titleStyle.Render("💿 ") + valStyle.Render(FormatBytes(stats.DiskUsedBytes)+"/"+FormatBytes(stats.DiskTotalBytes)),
		titleStyle.Render("🌐 ") + labelStyle.Render("⬇️") + valStyle.Render(FormatBytes(stats.NetworkRxBytes)) + labelStyle.Render(" ⬆️") + valStyle.Render(FormatBytes(stats.NetworkTxBytes)),
		titleStyle.Render("⏱️ ") + valStyle.Render(FormatUptime(stats.Uptime)),
	}

	barStyle := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1)

	return barStyle.Render(strings.Join(parts, "  "))
}

// ── Panel renderers ──────────────────────────────────────────────────

func (s *StatsBar) renderCPU(stats types.SystemStats, panelWidth int) string {
	th := s.theme

	var avg float32
	for _, c := range stats.CPUUsage {
		avg += c
	}
	if len(stats.CPUUsage) > 0 {
		avg /= float32(len(stats.CPUUsage))
	}

	// ── Total CPU sparkline ──
	avgPillFg := s.usageColor(float64(avg))
	avgPill := lipgloss.NewStyle().Foreground(avgPillFg).Bold(true).Render(fmt.Sprintf("%.0f%%", avg))
	totalLabel := lipgloss.NewStyle().Foreground(th.Text).Bold(true).Render("Total")

	sparkW := panelWidth - lipgloss.Width(totalLabel) - lipgloss.Width(avgPill) - 2
	if sparkW < 4 {
		sparkW = 4
	}
	totalSpark := s.cpuAvgHistory.RenderBlockGradient(sparkW, th.Green, th.Yellow, th.Peach, th.Red)
	totalLine := totalLabel + " " + totalSpark + " " + avgPill

	// ── Per-core sparklines ──
	maxCores := len(stats.CPUUsage)
	if maxCores > 16 {
		maxCores = 16
	}

	// 2-column layout for cores
	colWidth := (panelWidth - 1) / 2
	labelW := 3 // "C0 " or "C9 " or "10 "
	pctW := 5   // " 99%"
	coreSparkW := colWidth - labelW - pctW
	if coreSparkW < 4 {
		coreSparkW = 4
	}

	labelStyle := lipgloss.NewStyle().Foreground(th.Overlay0)
	pctStyle := lipgloss.NewStyle().Foreground(th.Subtext0)
	pctHighStyle := lipgloss.NewStyle().Foreground(th.Peach).Bold(true)

	renderCore := func(idx int) string {
		usage := stats.CPUUsage[idx]

		label := labelStyle.Render(fmt.Sprintf("c%d ", idx))

		// Use sparkline history if available, otherwise just current value
		var spark string
		if idx < len(s.cpuHistory) && s.cpuHistory[idx] != nil {
			spark = s.cpuHistory[idx].RenderBlockGradient(coreSparkW, th.Green, th.Yellow, th.Peach, th.Red)
		} else {
			// Fallback: single block
			spark = RenderMini([]float64{float64(usage)}, coreSparkW, th.Purple)
		}

		ps := pctStyle
		if usage > 75 {
			ps = pctHighStyle
		}
		pct := ps.Render(fmt.Sprintf(" %3.0f%%", usage))

		return label + spark + pct
	}

	var lines []string
	lines = append(lines, totalLine)
	lines = append(lines, "") // spacing after total

	for i := 0; i < maxCores; i += 2 {
		col1 := renderCore(i)
		if i+1 < maxCores {
			col2 := renderCore(i + 1)
			lines = append(lines, col1+" "+col2)
		} else {
			lines = append(lines, col1)
		}
	}

	if len(stats.CPUUsage) > maxCores {
		moreStyle := lipgloss.NewStyle().Foreground(th.Overlay0).Italic(true)
		lines = append(lines, moreStyle.Render(fmt.Sprintf("+%d more cores", len(stats.CPUUsage)-maxCores)))
	}

	// Load averages at bottom (like btop)
	lines = append(lines, "") // spacing before load
	dimStyle := lipgloss.NewStyle().Foreground(th.Overlay0)
	loadLine := dimStyle.Render("Load ") +
		s.loadStyle(stats.LoadAvg[0]).Render(fmt.Sprintf("%.1f", stats.LoadAvg[0])) +
		dimStyle.Render(" ") +
		s.loadStyle(stats.LoadAvg[1]).Render(fmt.Sprintf("%.1f", stats.LoadAvg[1])) +
		dimStyle.Render(" ") +
		s.loadStyle(stats.LoadAvg[2]).Render(fmt.Sprintf("%.1f", stats.LoadAvg[2]))
	lines = append(lines, loadLine)

	return strings.Join(lines, "\n")
}

func (s *StatsBar) renderMemory(stats types.SystemStats, panelWidth int) string {
	th := s.theme
	labelStyle := lipgloss.NewStyle().Foreground(th.Overlay0)
	valStyle := lipgloss.NewStyle().Foreground(th.Text)

	memPct := float64(0)
	if stats.MemoryTotal > 0 {
		memPct = float64(stats.MemoryUsed) / float64(stats.MemoryTotal) * 100
	}

	// ── RAM bar + percentage ──
	pctPill := lipgloss.NewStyle().Foreground(s.usageColor(memPct)).Bold(true).Render(fmt.Sprintf("%.0f%%", memPct))
	ramLabel := lipgloss.NewStyle().Foreground(th.Text).Bold(true).Render("RAM")

	barWidth := panelWidth
	if barWidth < 8 {
		barWidth = 8
	}

	// ── Inline stats: Used · Free · Cache ──
	usedPart := lipgloss.NewStyle().Foreground(th.Lavender).Bold(true).Render(FormatMemorySize(stats.MemoryUsed))
	totalPart := valStyle.Render(FormatMemorySize(stats.MemoryTotal))
	dot := labelStyle.Render(" · ")
	freePart := labelStyle.Render("Free ") + valStyle.Render(FormatMemorySize(stats.MemoryFree))
	statsLine := usedPart + labelStyle.Render("/") + totalPart + dot + freePart
	if stats.MemoryCached > 0 {
		cachePart := labelStyle.Render("Cache ") + lipgloss.NewStyle().Foreground(th.Yellow).Render(FormatMemorySize(stats.MemoryCached))
		statsLine += dot + cachePart
	}

	// ── Disk section ──
	diskPct := float64(0)
	if stats.DiskTotalBytes > 0 {
		diskPct = float64(stats.DiskUsedBytes) / float64(stats.DiskTotalBytes) * 100
	}

	diskPill := lipgloss.NewStyle().Foreground(s.usageColor(diskPct)).Bold(true).Render(fmt.Sprintf("%.0f%%", diskPct))
	diskLabel := lipgloss.NewStyle().Foreground(th.Text).Bold(true).Render(IconDisk + " Disk")
	diskRulerW := panelWidth - lipgloss.Width(diskLabel) - lipgloss.Width(diskPill) - 3
	if diskRulerW < 1 {
		diskRulerW = 1
	}
	diskHeader := diskLabel + " " +
		lipgloss.NewStyle().Foreground(th.Surface1).Render(strings.Repeat("─", diskRulerW)) +
		" " + diskPill

	diskBar := s.btopBar(diskPct, barWidth)

	diskUsed := lipgloss.NewStyle().Foreground(th.Lavender).Bold(true).Render(FormatBytes(stats.DiskUsedBytes))
	diskTotal := valStyle.Render(FormatBytes(stats.DiskTotalBytes))
	diskFree := labelStyle.Render("Free ") + valStyle.Render(FormatBytes(stats.DiskFreeBytes))
	diskStats := diskUsed + labelStyle.Render("/") + diskTotal + dot + diskFree

	ramHeader := ramLabel + " " +
		lipgloss.NewStyle().Foreground(th.Surface1).Render(strings.Repeat("─", panelWidth-lipgloss.Width(ramLabel)-lipgloss.Width(pctPill)-3)) +
		" " + pctPill

	lines := []string{
		ramHeader,
		s.btopBar(memPct, barWidth),
		statsLine,
		"",
		diskHeader,
		diskBar,
		diskStats,
	}

	return strings.Join(lines, "\n")
}

func (s *StatsBar) renderInfo(stats types.SystemStats, panelWidth int) string {
	th := s.theme
	labelStyle := lipgloss.NewStyle().Foreground(th.Overlay0)
	valStyle := lipgloss.NewStyle().Foreground(th.Text)
	accentStyle := lipgloss.NewStyle().Foreground(th.Lavender).Bold(true)

	sep := lipgloss.NewStyle().Foreground(th.Surface1).Render(strings.Repeat("─", panelWidth))

	// Hostname
	hostname := stats.Hostname
	if hostname == "" {
		hostname = "—"
	}
	hostLine := labelStyle.Render("Host    ") + accentStyle.Render(hostname)

	// OS version
	osVer := stats.OSVersion
	if osVer == "" {
		osVer = "—"
	}
	osLine := labelStyle.Render("OS      ") + valStyle.Render(osVer)

	// Kernel
	kernel := stats.KernelVersion
	if kernel == "" {
		kernel = "—"
	}
	kernelLine := labelStyle.Render("Kernel  ") + valStyle.Render(kernel)

	// CPU brand
	cpuBrand := stats.CPUBrand
	if cpuBrand == "" {
		cpuBrand = fmt.Sprintf("%d cores", len(stats.CPUUsage))
	}
	// Truncate long CPU names
	maxCPUW := panelWidth - 8
	if len(cpuBrand) > maxCPUW && maxCPUW > 4 {
		cpuBrand = cpuBrand[:maxCPUW-1] + "…"
	}
	cpuLine := labelStyle.Render("CPU     ") + valStyle.Render(cpuBrand)

	// Cores
	coreLine := labelStyle.Render("Cores   ") + valStyle.Render(fmt.Sprintf("%d", len(stats.CPUUsage)))

	// Uptime
	uptimeLine := labelStyle.Render("Uptime  ") + accentStyle.Render(FormatUptime(stats.Uptime))

	// Process count
	procLine := labelStyle.Render("Procs   ") + valStyle.Render(fmt.Sprintf("%d", stats.ProcessCount))

	lines := []string{
		hostLine,
		osLine,
		kernelLine,
		"",
		sep,
		"",
		cpuLine,
		coreLine,
		uptimeLine,
		procLine,
	}

	return strings.Join(lines, "\n")
}

func (s *StatsBar) renderNetwork(stats types.SystemStats, panelWidth int) string {
	th := s.theme

	labelStyle := lipgloss.NewStyle().Foreground(th.Overlay0)
	rxColor := lipgloss.NewStyle().Foreground(th.Green)
	txColor := lipgloss.NewStyle().Foreground(th.Blue)

	// ── RX sparkline ──
	rxLabel := rxColor.Bold(true).Render(IconDownload + " RX")
	rxVal := rxColor.Render(FormatBytes(stats.NetworkRxBytes) + "/s")
	rxSparkW := panelWidth - lipgloss.Width(rxLabel) - lipgloss.Width(rxVal) - 2
	if rxSparkW < 4 {
		rxSparkW = 4
	}
	rxSpark := s.rxHistory.RenderBlock(rxSparkW, th.Green)
	rxLine := rxLabel + " " + rxSpark + " " + rxVal

	// ── TX sparkline ──
	txLabel := txColor.Bold(true).Render(IconUpload + " TX")
	txVal := txColor.Render(FormatBytes(stats.NetworkTxBytes) + "/s")
	txSparkW := panelWidth - lipgloss.Width(txLabel) - lipgloss.Width(txVal) - 2
	if txSparkW < 4 {
		txSparkW = 4
	}
	txSpark := s.txHistory.RenderBlock(txSparkW, th.Blue)
	txLine := txLabel + " " + txSpark + " " + txVal

	// ── Peak & Session stats ──
	sep := lipgloss.NewStyle().Foreground(th.Surface1).Render(strings.Repeat("─", panelWidth))

	peakRx := labelStyle.Render("Peak " + IconDownload + " ") + rxColor.Render(FormatBytes(s.peakRx)+"/s")
	peakTx := labelStyle.Render("Peak " + IconUpload + " ") + txColor.Render(FormatBytes(s.peakTx)+"/s")

	lines := []string{
		rxLine,
		"",
		txLine,
		"",
		sep,
		"",
		peakRx,
		peakTx,
	}

	return strings.Join(lines, "\n")
}

// ── Helpers ──────────────────────────────────────────────────────────

// usageColor returns a color based on usage percentage (green→yellow→peach→red)
func (s *StatsBar) usageColor(pct float64) color.Color {
	switch {
	case pct > 90:
		return s.theme.Red
	case pct > 75:
		return s.theme.Peach
	case pct > 50:
		return s.theme.Yellow
	default:
		return s.theme.Green
	}
}

// loadStyle returns a style colored by load average severity
func (s *StatsBar) loadStyle(load float64) lipgloss.Style {
	switch {
	case load > 4.0:
		return lipgloss.NewStyle().Foreground(s.theme.Red).Bold(true)
	case load > 2.0:
		return lipgloss.NewStyle().Foreground(s.theme.Yellow)
	default:
		return lipgloss.NewStyle().Foreground(s.theme.Text)
	}
}

// btopBar creates a btop-style bar with color based on usage level.
func (s *StatsBar) btopBar(pct float64, width int) string {
	var fillColor, accentColor color.Color
	switch {
	case pct > 90:
		fillColor = s.theme.Red
		accentColor = s.theme.Maroon
	case pct > 75:
		fillColor = s.theme.Peach
		accentColor = s.theme.Yellow
	case pct > 50:
		fillColor = s.theme.Yellow
		accentColor = s.theme.Peach
	default:
		fillColor = s.theme.Purple
		accentColor = s.theme.Pink
	}
	return RenderBar(pct/100.0, BarStyle{
		FillColor:   fillColor,
		AccentColor: accentColor,
		EmptyColor:  s.theme.Surface0,
		Width:       width,
	})
}
