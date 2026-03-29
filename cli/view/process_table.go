package view

import (
	"fmt"
	"strings"

	"github.com/abdenasser/neohtop-cli/config"
	"github.com/abdenasser/neohtop-cli/theme"
	"github.com/abdenasser/neohtop-cli/types"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// columnSpec defines min width, ideal width, and growth priority per column.
// Higher priority columns get extra space first.
type columnSpec struct {
	Min      int // absolute minimum (header + a few chars)
	Ideal    int // comfortable width for typical content
	Priority int // 0=fixed, 1=low grow, 2=medium grow, 3=high grow
}

var columnSpecs = map[string]columnSpec{
	"pid":     {Min: 10, Ideal: 12, Priority: 0},     // "1·PID ⇅" needs ~10
	"name":    {Min: 12, Ideal: 22, Priority: 3},     // icon + process names
	"cpu":     {Min: 16, Ideal: 18, Priority: 0},     // "3·CPU% ⇅" needs ~11, + mini-bar
	"memory":  {Min: 13, Ideal: 14, Priority: 1},     // "4·Memory ⇅" needs ~13
	"status":  {Min: 13, Ideal: 14, Priority: 0},     // "5·Status ⇅" needs ~13
	"user":    {Min: 11, Ideal: 16, Priority: 2},     // "6·User ⇅" needs ~11
	"command": {Min: 15, Ideal: 40, Priority: 3},     // commands can be very long
	"threads": {Min: 10, Ideal: 12, Priority: 0},     // "0·Thr ⇅" needs ~10
	"runtime": {Min: 14, Ideal: 14, Priority: 0},     // "8·Runtime ⇅" needs ~14
	"disk":    {Min: 15, Ideal: 18, Priority: 1},     // "9·Disk I/O ⇅" needs ~15
}

// Column headers
var columnHeaders = map[string]string{
	"pid":     "PID",
	"name":    "Name",
	"cpu":     "CPU%",
	"memory":  "Memory",
	"status":  "Status",
	"user":    "User",
	"command": "Command",
	"threads": "Thr",
	"runtime": "Runtime",
	"disk":    "Disk I/O",
}

// columnSortKeys maps column names to their keyboard shortcut for sorting.
var columnSortKeys = map[string]string{
	"pid":     "1",
	"name":    "2",
	"cpu":     "3",
	"memory":  "4",
	"status":  "5",
	"user":    "6",
	"command": "7",
	"runtime": "8",
	"disk":    "9",
	"threads": "0",
}

// ProcessTable renders the process list using Charm's lipgloss/table
type ProcessTable struct {
	theme      *theme.Theme
	cfg        *config.Config
	width      int
	height     int
	searchTerm string
	treeMode   bool

	// Cached column widths — only recomputed when width or columns change
	cachedColWidths []int
	cachedWidth     int
	cachedNumCols   int
}

func NewProcessTable(th *theme.Theme, cfg *config.Config) *ProcessTable {
	return &ProcessTable{theme: th, cfg: cfg, width: 120, height: 30}
}

func (pt *ProcessTable) SetTheme(th *theme.Theme) {
	pt.theme = th
}

func (pt *ProcessTable) SetSize(w, h int) {
	pt.width = w
	pt.height = h
}

func (pt *ProcessTable) SetSearchTerm(term string) {
	pt.searchTerm = term
}

func (pt *ProcessTable) SetTreeMode(enabled bool) {
	pt.treeMode = enabled
}

// getColWidths returns cached column widths, recomputing only when width or columns change
func (pt *ProcessTable) getColWidths() []int {
	numCols := len(pt.cfg.Columns)
	if pt.cachedColWidths != nil && pt.cachedWidth == pt.width && pt.cachedNumCols == numCols {
		return pt.cachedColWidths
	}

	overhead := 2 + (numCols - 1) + numCols*2
	availableForContent := pt.width - overhead
	if availableForContent < numCols*4 {
		availableForContent = numCols * 4
	}
	pt.cachedColWidths = pt.computeColumnWidths(availableForContent)
	pt.cachedWidth = pt.width
	pt.cachedNumCols = numCols
	return pt.cachedColWidths
}

func (pt *ProcessTable) Render(processes []types.Process, cursor, scrollOffset int, sortCfg types.SortConfig, pinned map[string]bool) string {
	if len(processes) == 0 {
		ghost := lipgloss.NewStyle().Foreground(pt.theme.Overlay0).Render("👻")
		msg := lipgloss.NewStyle().Foreground(pt.theme.Overlay0).Italic(true).Render("No processes match your filters")
		hint := lipgloss.NewStyle().Foreground(pt.theme.Overlay0).Render("Try adjusting search or filter criteria")
		empty := ghost + "\n\n" + msg + "\n" + hint
		box := lipgloss.NewStyle().Padding(3, 4).Width(pt.width).Align(lipgloss.Center)
		return box.Render(empty)
	}

	// Calculate visible range
	visibleCount := pt.height - 4 // account for table borders + header
	if visibleCount < 1 {
		visibleCount = 10
	}

	endIdx := scrollOffset + visibleCount
	if endIdx > len(processes) {
		endIdx = len(processes)
	}

	visibleProcs := processes[scrollOffset:endIdx]

	// Build header row with sort indicators
	headers := pt.buildHeaders(sortCfg)

	// Use cached column widths (recomputed only on resize)
	colWidths := pt.getColWidths()

	// Build data rows (pass colWidths so truncation matches column sizes)
	rows := make([][]string, 0, len(visibleProcs))
	for i, p := range visibleProcs {
		globalIdx := scrollOffset + i
		isPinned := pinned[p.Command]
		row := pt.buildRow(p, isPinned, globalIdx, colWidths)
		rows = append(rows, row)
	}

	// Create the Charm table with precise column widths that fill terminal width
	t := table.New().
		Width(pt.width).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(pt.theme.Surface1)).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Base cell style
			s := lipgloss.NewStyle().Padding(0, 1)

			// Apply column width
			if col < len(colWidths) {
				s = s.Width(colWidths[col])
			}

			// Header row
			if row == table.HeaderRow {
				return s.
					Foreground(pt.theme.Overlay1).
					Bold(true).
					Align(lipgloss.Center)
			}

			// Data rows
			if row < 0 || row >= len(visibleProcs) {
				return s
			}

			p := visibleProcs[row]
			globalIdx := scrollOffset + row
			isSelected := globalIdx == cursor
			isPinned := pinned[p.Command]

			// Selected + pinned
			if isSelected && isPinned {
				s = s.Background(pt.theme.Surface1).
					Foreground(pt.theme.Lavender).
					Bold(true)
				return s
			}

			// Selected row
			if isSelected {
				s = s.Background(pt.theme.Surface1).
					Foreground(pt.theme.Text).
					Bold(true)
				return s
			}

			// Pinned row
			if isPinned {
				s = s.Background(pt.theme.Surface0).
					Foreground(pt.theme.Lavender)
				return s
			}

			// High CPU warning
			if p.CPUUsage > 80 {
				s = s.Foreground(pt.theme.Maroon)
				return s
			}

			// Alternating row colors
			if row%2 == 0 {
				s = s.Foreground(pt.theme.Text)
			} else {
				s = s.Foreground(pt.theme.Subtext1)
			}

			// Column-specific coloring
			colName := pt.colNameAt(col)
			switch colName {
			case "cpu":
				switch {
				case p.CPUUsage > 80:
					s = s.Foreground(pt.theme.Red)
				case p.CPUUsage > 50:
					s = s.Foreground(pt.theme.Yellow)
				case p.CPUUsage > 20:
					s = s.Foreground(pt.theme.Peach)
				}
			case "memory":
				memMB := float64(p.MemoryUsage) / (1024 * 1024)
				switch {
				case memMB > 1024:
					s = s.Foreground(pt.theme.Red)
				case memMB > 512:
					s = s.Foreground(pt.theme.Yellow)
				case memMB > 256:
					s = s.Foreground(pt.theme.Peach)
				}
			case "status":
				switch p.Status {
				case "Running":
					s = s.Foreground(pt.theme.Green)
				case "Sleeping", "Idle":
					s = s.Foreground(pt.theme.Blue)
				case "Stopped":
					s = s.Foreground(pt.theme.Red)
				case "Zombie":
					s = s.Foreground(pt.theme.Yellow)
				}
			}

			return s
		}).
		Headers(headers...).
		Rows(rows...)

	return t.String()
}

// highlightMatch wraps matching substrings with accent coloring
func (pt *ProcessTable) highlightMatch(s string) string {
	if pt.searchTerm == "" {
		return s
	}
	term := strings.ToLower(pt.searchTerm)
	lower := strings.ToLower(s)
	idx := strings.Index(lower, term)
	if idx < 0 {
		return s
	}
	// Highlight the match
	hl := lipgloss.NewStyle().
		Background(pt.theme.Surface1).
		Foreground(pt.theme.Yellow).
		Bold(true)
	before := s[:idx]
	match := s[idx : idx+len(pt.searchTerm)]
	after := s[idx+len(pt.searchTerm):]
	return before + hl.Render(match) + after
}

// buildHeaders creates header labels with sort indicators
func (pt *ProcessTable) buildHeaders(sortCfg types.SortConfig) []string {
	headers := make([]string, 0, len(pt.cfg.Columns))

	for _, col := range pt.cfg.Columns {
		label := columnHeaders[col]
		key := columnSortKeys[col]
		field := colToSortField(col)

		// Build header: "key·Label ▲/▼" with colored key
		keyStyle := lipgloss.NewStyle().Foreground(pt.theme.Purple).Bold(true)
		dotStyle := lipgloss.NewStyle().Foreground(pt.theme.Overlay0)
		h := keyStyle.Render(key) + dotStyle.Render("·") + label

		if field == sortCfg.Field {
			if sortCfg.Direction == types.SortAsc {
				h += " " + IconSortAsc
			} else {
				h += " " + IconSortDesc
			}
		} else {
			h += " " + IconSortNone
		}

		headers = append(headers, h)
	}

	return headers
}

// buildRow creates a string slice for a single process row.
// colWidths provides the computed width for each column so truncation
// matches the actual available space (no wrapping, no waste).
func (pt *ProcessTable) buildRow(p types.Process, isPinned bool, globalIdx int, colWidths []int) []string {
	row := make([]string, 0, len(pt.cfg.Columns))

	for i, col := range pt.cfg.Columns {
		w := 20 // fallback
		if i < len(colWidths) {
			w = colWidths[i] - 2 // subtract cell padding (Padding(0,1) = 2 chars)
			if w < 4 {
				w = 4
			}
		}
		switch col {
		case "pid":
			row = append(row, fmt.Sprintf("%d", p.PID))
		case "name":
			icon := ProcessIcon(p.Name)
			name := p.Name

			// Tree prefix (e.g. "├─ ", "│  └─ ")
			treeStr := ""
			if pt.treeMode && p.TreePrefix != "" {
				treeStyle := lipgloss.NewStyle().Foreground(pt.theme.Surface2)
				treeStr = treeStyle.Render(p.TreePrefix)
			}
			treePrefixW := lipgloss.Width(treeStr)

			prefix := ""
			availW := w - treePrefixW
			if availW < 4 {
				availW = 4
			}
			if isPinned {
				prefix = "📌"
				if availW > 5 {
					name = Truncate(name, availW-4) // icon(2) + pin(2)
				}
			} else if availW > 3 {
				name = Truncate(name, availW-2)
			} else {
				name = Truncate(name, availW)
			}
			name = pt.highlightMatch(name)
			if isPinned {
				row = append(row, treeStr+prefix+icon+" "+name)
			} else if availW > 3 {
				row = append(row, treeStr+icon+" "+name)
			} else {
				row = append(row, treeStr+name)
			}
		case "cpu":
			pct := FormatPercentage(p.CPUUsage)
			barW := 5
			filled := int(p.CPUUsage / 100.0 * float32(barW*8))
			fullBlocks := filled / 8
			remainder := filled % 8
			blocks := []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
			bar := ""
			for j := 0; j < fullBlocks && j < barW; j++ {
				bar += "█"
			}
			if fullBlocks < barW && remainder > 0 {
				bar += blocks[remainder]
			}
			// Pad the bar to barW using visual width
			for lipgloss.Width(bar) < barW {
				bar += " "
			}
			row = append(row, pct+" "+bar)
		case "memory":
			row = append(row, FormatBytes(p.MemoryUsage))
		case "status":
			row = append(row, p.Status)
		case "user":
			row = append(row, Truncate(p.User, w))
		case "command":
			cmd := Truncate(p.Command, w)
			cmd = pt.highlightMatch(cmd)
			row = append(row, cmd)
		case "threads":
			if p.Threads != nil {
				row = append(row, fmt.Sprintf("%d", *p.Threads))
			} else {
				row = append(row, "-")
			}
		case "runtime":
			row = append(row, FormatRuntime(p.RunTime))
		case "disk":
			disk := FormatBytesCompact(p.DiskRead) + "/" + FormatBytesCompact(p.DiskWrite)
			row = append(row, Truncate(disk, w))
		default:
			row = append(row, "")
		}
	}

	return row
}

// computeColumnWidths distributes available width across columns using a
// priority-based algorithm:
//  1. Every column starts at its minimum width.
//  2. Remaining space is distributed in priority rounds (3→2→1→0),
//     growing each eligible column toward its ideal width.
//  3. Any leftover after ideals are met is spread evenly across
//     high-priority (3) columns so wide content (names, commands) gets room.
func (pt *ProcessTable) computeColumnWidths(availableWidth int) []int {
	numCols := len(pt.cfg.Columns)
	widths := make([]int, numCols)

	// Step 1: assign minimums
	usedWidth := 0
	for i, col := range pt.cfg.Columns {
		spec := columnSpecs[col]
		widths[i] = spec.Min
		usedWidth += spec.Min
	}

	remaining := availableWidth - usedWidth
	if remaining <= 0 {
		return widths
	}

	// Step 2: grow columns toward ideal, priority 3 first, then 2, 1, 0
	for pri := 3; pri >= 0; pri-- {
		if remaining <= 0 {
			break
		}
		for i, col := range pt.cfg.Columns {
			if remaining <= 0 {
				break
			}
			spec := columnSpecs[col]
			if spec.Priority != pri {
				continue
			}
			grow := spec.Ideal - widths[i]
			if grow <= 0 {
				continue
			}
			if grow > remaining {
				grow = remaining
			}
			widths[i] += grow
			remaining -= grow
		}
	}

	// Step 3: distribute any leftover evenly among high-priority (>=2) columns
	if remaining > 0 {
		growable := make([]int, 0)
		for i, col := range pt.cfg.Columns {
			spec := columnSpecs[col]
			if spec.Priority >= 2 {
				growable = append(growable, i)
			}
		}
		if len(growable) == 0 {
			// fallback: grow all data columns
			for i := range pt.cfg.Columns {
				growable = append(growable, i)
			}
		}
		perCol := remaining / len(growable)
		extra := remaining % len(growable)
		for j, idx := range growable {
			bonus := perCol
			if j < extra {
				bonus++
			}
			widths[idx] += bonus
		}
	}

	return widths
}

// colNameAt returns the column name at a given index
func (pt *ProcessTable) colNameAt(col int) string {
	if col < len(pt.cfg.Columns) {
		return pt.cfg.Columns[col]
	}
	return ""
}

// ColumnHitZones returns the X ranges for each column, for mouse click mapping.
// Accounts for lipgloss/table border and padding.
func (pt *ProcessTable) ColumnHitZones() []ColumnZone {
	colWidths := pt.getColWidths()

	var zones []ColumnZone
	pos := 1 // start after left border

	for i, col := range pt.cfg.Columns {
		w := colWidths[i] + 2 // cell width + padding
		zones = append(zones, ColumnZone{
			StartX: pos,
			EndX:   pos + w,
			Field:  colToSortField(col),
		})
		pos += w + 1 // +1 for column separator
	}
	return zones
}

// ColumnZone represents a clickable column header region
type ColumnZone struct {
	StartX int
	EndX   int
	Field  types.SortField
}

func colToSortField(col string) types.SortField {
	switch col {
	case "pid":
		return types.SortByPID
	case "name":
		return types.SortByName
	case "cpu":
		return types.SortByCPU
	case "memory":
		return types.SortByMemory
	case "status":
		return types.SortByStatus
	case "user":
		return types.SortByUser
	case "command":
		return types.SortByCommand
	case "threads":
		return types.SortByThreads
	case "runtime":
		return types.SortByRunTime
	case "disk":
		return types.SortByDisk
	}
	return types.SortByPID
}

