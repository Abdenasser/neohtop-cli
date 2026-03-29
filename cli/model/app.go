package model

import (
	"time"

	"github.com/abdenasser/neohtop-cli/config"
	"github.com/abdenasser/neohtop-cli/filter"
	"github.com/abdenasser/neohtop-cli/monitor"
	"github.com/abdenasser/neohtop-cli/theme"
	"github.com/abdenasser/neohtop-cli/types"
	"github.com/abdenasser/neohtop-cli/view"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Messages
type TickMsg time.Time
type ProcessDataMsg struct {
	Processes []types.Process
	Stats     types.SystemStats
	Err       error
}
type KillResultMsg struct {
	PID     uint32
	Success bool
}
type ProcessDetailMsg struct {
	Detail *types.ProcessDetail
	Err    error
}

// App is the main Bubble Tea model
type App struct {
	// Data — uses the native Go monitor (no FFI, no JSON)
	mon           *monitor.Monitor
	processes     []types.Process
	filteredProcs []types.Process
	systemStats   types.SystemStats
	ready         bool
	err           error

	// UI state
	width           int
	height          int
	cursor          int
	scrollOffset    int
	searchTerm      string
	searchMode      bool
	isFrozen        bool
	treeMode        bool
	activeOverlay   types.OverlayType
	selectedProcess *types.Process
	selectedDetail  *types.ProcessDetail // lazy-loaded on overlay open
	lastClickTime   time.Time
	lastClickRow    int

	// Layout measurements (set each frame in View)
	tableStartLine int

	// Configuration
	sortConfig      types.SortConfig
	filterConfig    filter.Config
	pinnedProcesses map[string]bool
	cfg             *config.Config
	theme           *theme.Theme

	// Sub-views
	statsBar     *view.StatsBar
	processTable *view.ProcessTable
	toolbar      *view.Toolbar
	helpView     *view.Help
	detailsView  *view.ProcessDetails
	killConfirm  *view.KillConfirm
	filterPanel  *view.FilterPanel
	columnPanel  *view.ColumnPanel
	themePanel   *view.ThemePanel
	footer       *view.Footer

	// Panel state
	panelLine int
}

// NewApp creates a new application model
func NewApp(mon *monitor.Monitor) *App {
	cfg := config.Load()
	th := theme.GetTheme(cfg.Theme)

	return &App{
		mon:             mon,
		sortConfig:      types.SortConfig{Field: types.SortByCPU, Direction: types.SortDesc},
		filterConfig:    filter.NewConfig(),
		pinnedProcesses: make(map[string]bool),
		cfg:             cfg,
		theme:           th,
		statsBar:        view.NewStatsBar(th),
		processTable:    view.NewProcessTable(th, cfg),
		toolbar:         view.NewToolbar(th),
		helpView:        view.NewHelp(th),
		detailsView:     view.NewProcessDetails(th),
		killConfirm:     view.NewKillConfirm(th),
		filterPanel:     view.NewFilterPanel(th),
		columnPanel:     view.NewColumnPanel(th),
		themePanel:      view.NewThemePanel(th),
		footer:          view.NewFooter(th),
	}
}

// Init is called when the program starts
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.fetchProcesses(),
		a.tickCmd(),
	)
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case TickMsg:
		if a.isFrozen {
			return a, a.tickCmd()
		}
		return a, tea.Batch(a.fetchProcesses(), a.tickCmd())

	case ProcessDataMsg:
		if msg.Err != nil {
			a.err = msg.Err
			return a, nil
		}
		a.processes = msg.Processes
		a.systemStats = msg.Stats
		a.ready = true
		a.statsBar.RecordStats(msg.Stats) // push into sparkline histories
		a.applyFiltersAndSort()
		return a, nil

	case KillResultMsg:
		a.activeOverlay = types.OverlayNone
		a.selectedProcess = nil
		a.selectedDetail = nil
		return a, nil

	case ProcessDetailMsg:
		if msg.Err == nil && msg.Detail != nil {
			a.selectedDetail = msg.Detail
		}
		return a, nil

	// Bubble Tea v2: KeyPressMsg replaces KeyMsg
	case tea.KeyPressMsg:
		return a.handleKeyPress(msg)

	// Bubble Tea v2: Mouse events are split into separate types
	case tea.MouseClickMsg:
		return a.handleMouseClick(msg.X, msg.Y, msg.Button)

	case tea.MouseWheelMsg:
		return a.handleMouseWheel(msg)

	case tea.MouseMotionMsg:
		return a.handleMouseMotion(msg)
	}

	return a, nil
}

// newView creates a tea.View with AltScreen and MouseMode set.
// In Bubble Tea v2, these are View fields instead of program options.
func (a *App) newView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// View renders the UI — layout: StatsBar → ToolBar → ProcessTable
func (a *App) View() tea.View {
	if !a.ready {
		loading := lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7")).Bold(true).Render("👻 NeoHtop CLI")
		dots := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).Render("Loading system data...")
		return a.newView("\n\n" + lipgloss.PlaceHorizontal(a.width, lipgloss.Center, loading) + "\n" + lipgloss.PlaceHorizontal(a.width, lipgloss.Center, dots))
	}

	if a.err != nil {
		return a.newView("\n  Error: " + a.err.Error())
	}

	// Render overlay if active
	switch a.activeOverlay {
	case types.OverlayHelp:
		return a.newView(a.helpView.Render(a.width, a.height))
	case types.OverlayProcessDetails:
		if a.selectedProcess != nil {
			return a.newView(a.detailsView.Render(*a.selectedProcess, a.selectedDetail, a.processes, a.width, a.height))
		}
	case types.OverlayKillConfirm:
		if a.selectedProcess != nil {
			return a.newView(a.killConfirm.Render(*a.selectedProcess, a.width, a.height))
		}
	case types.OverlayFilters:
		return a.newView(a.filterPanel.Render(a.filterConfig, a.panelLine, a.width, a.height))
	case types.OverlayColumns:
		return a.newView(a.columnPanel.Render(a.cfg.Columns, a.panelLine, a.width, a.height))
	case types.OverlayThemes:
		return a.newView(a.themePanel.Render(a.theme.Name, a.panelLine, a.width, a.height))
	}

	// === Main layout: StatsBar → ToolBar → ProcessTable ===
	statsStr := a.statsBar.Render(a.systemStats, a.width)

	toolbarStr := a.toolbar.Render(a.searchTerm, a.searchMode, a.sortConfig, a.isFrozen, len(a.filteredProcs), len(a.processes), a.width, a.cfg.RefreshRate, a.treeMode)

	// Footer — includes selected process info + pin/info/kill actions
	procs := a.pageProcesses()
	hasSelection := a.cursor >= 0 && a.cursor < len(procs)
	isPinned := false
	selectedPID := 0
	selectedName := ""
	if hasSelection {
		isPinned = a.pinnedProcesses[procs[a.cursor].Command]
		selectedPID = int(procs[a.cursor].PID)
		selectedName = procs[a.cursor].Name
	}
	footerStr := a.footer.Render(a.systemStats, selectedPID, selectedName, hasSelection, isPinned, a.width)

	// Measure Y lines for mouse mapping
	headerCombined := lipgloss.JoinVertical(lipgloss.Left, statsStr, toolbarStr)
	headerHeight := lipgloss.Height(headerCombined)
	a.tableStartLine = headerHeight

	remainingHeight := a.height - headerHeight - 1
	if remainingHeight < 5 {
		remainingHeight = 5
	}
	a.processTable.SetSize(a.width, remainingHeight)

	a.processTable.SetSearchTerm(a.searchTerm)
	a.processTable.SetTreeMode(a.treeMode)
	tableStr := a.processTable.Render(a.filteredProcs, a.cursor, a.scrollOffset, a.sortConfig, a.pinnedProcesses)

	return a.newView(lipgloss.JoinVertical(lipgloss.Left, statsStr, toolbarStr, tableStr, footerStr))
}

// handleKeyPress processes keyboard input (Bubble Tea v2: KeyPressMsg)
func (a *App) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// In search mode, handle text input
	if a.searchMode {
		switch msg.String() {
		case "esc":
			a.searchMode = false
			a.searchTerm = ""
			a.applyFiltersAndSort()
		case "enter":
			a.searchMode = false
		case "backspace":
			if len(a.searchTerm) > 0 {
				a.searchTerm = a.searchTerm[:len(a.searchTerm)-1]
				a.applyFiltersAndSort()
			}
		default:
			if len(msg.String()) == 1 {
				a.searchTerm += msg.String()
				a.applyFiltersAndSort()
			}
		}
		return a, nil
	}

	// Overlay-specific keys
	if a.activeOverlay != types.OverlayNone {
		switch a.activeOverlay {
		case types.OverlayKillConfirm:
			switch msg.String() {
			case "y", "enter":
				if a.selectedProcess != nil {
					return a, a.killProcess(a.selectedProcess.PID)
				}
			case "n", "esc", "q":
				a.activeOverlay = types.OverlayNone
				a.selectedProcess = nil
			}

		case types.OverlayProcessDetails:
			switch msg.String() {
			case "esc", "q":
				a.activeOverlay = types.OverlayNone
				a.selectedProcess = nil
				a.selectedDetail = nil
			case "x", "delete":
				if a.selectedProcess != nil {
					a.activeOverlay = types.OverlayKillConfirm
				}
			case "p":
				if a.selectedProcess != nil {
					cmd := a.selectedProcess.Command
					if a.pinnedProcesses[cmd] {
						delete(a.pinnedProcesses, cmd)
					} else {
						a.pinnedProcesses[cmd] = true
					}
					a.applyFiltersAndSort()
				}
			}

		case types.OverlayFilters:
			switch msg.String() {
			case "esc", "q":
				a.activeOverlay = types.OverlayNone
			case "up", "k", "shift+tab":
				if a.panelLine > 0 {
					a.panelLine--
				}
			case "down", "j", "tab":
				if a.panelLine < 3 {
					a.panelLine++
				}
			case "enter", " ", "space":
				a.toggleFilterLine(a.panelLine)
			case "right", "l":
				a.adjustFilterOperator(a.panelLine, 1)
			case "left", "h":
				a.adjustFilterOperator(a.panelLine, -1)
			case "+", "=":
				a.adjustFilterValue(a.panelLine, 10)
			case "-":
				a.adjustFilterValue(a.panelLine, -10)
			}

		case types.OverlayColumns:
			switch msg.String() {
			case "esc", "q":
				a.activeOverlay = types.OverlayNone
			case "up", "k":
				if a.panelLine > 0 {
					a.panelLine--
				}
			case "down", "j":
				if a.panelLine < len(view.AllColumns)-1 {
					a.panelLine++
				}
			case "enter", " ", "space":
				a.toggleColumn(a.panelLine)
			}

		case types.OverlayThemes:
			names := theme.ThemeNames()
			switch msg.String() {
			case "esc", "q":
				a.activeOverlay = types.OverlayNone
			case "up", "k":
				if a.panelLine > 0 {
					a.panelLine--
				}
			case "down", "j":
				if a.panelLine < len(names)-1 {
					a.panelLine++
				}
			case "enter", " ", "space":
				a.applyTheme(names[a.panelLine])
			}

		default:
			switch msg.String() {
			case "esc", "q":
				a.activeOverlay = types.OverlayNone
				a.selectedProcess = nil
				a.selectedDetail = nil
			}
		}
		return a, nil
	}

	// Global keys
	switch msg.String() {
	case "q", "ctrl+c":
		return a, tea.Quit

	case "?":
		a.activeOverlay = types.OverlayHelp
	case "/", "s":
		a.searchMode = true
	case " ", "space":
		a.isFrozen = !a.isFrozen
	case "f":
		a.activeOverlay = types.OverlayFilters
		a.panelLine = 0
	case "c":
		a.activeOverlay = types.OverlayColumns
		a.panelLine = 0
	case "t":
		a.activeOverlay = types.OverlayThemes
		// Set panelLine to current theme index
		names := theme.ThemeNames()
		for i, name := range names {
			if name == a.theme.Name {
				a.panelLine = i
				break
			}
		}

	// Navigation
	case "up":
		a.moveCursor(-1)
	case "down", "j":
		a.moveCursor(1)
	case "pgup":
		a.moveCursor(-a.visibleRows())
	case "pgdown":
		a.moveCursor(a.visibleRows())
	case "home", "g":
		a.cursor = 0
		a.scrollOffset = 0
	case "end", "G":
		a.cursor = len(a.pageProcesses()) - 1
		if a.cursor < 0 {
			a.cursor = 0
		}

	// Process actions (match shortcut hints on selected row)
	case "enter", "i":
		procs := a.pageProcesses()
		if a.cursor >= 0 && a.cursor < len(procs) {
			p := procs[a.cursor]
			a.selectedProcess = &p
			a.selectedDetail = nil
			a.activeOverlay = types.OverlayProcessDetails
			return a, a.fetchProcessDetail(p.PID)
		}
	case "k", "x", "delete":
		procs := a.pageProcesses()
		if a.cursor >= 0 && a.cursor < len(procs) {
			p := procs[a.cursor]
			a.selectedProcess = &p
			a.activeOverlay = types.OverlayKillConfirm
		}
	case "p", "u":
		procs := a.pageProcesses()
		if a.cursor >= 0 && a.cursor < len(procs) {
			cmd := procs[a.cursor].Command
			if a.pinnedProcesses[cmd] {
				delete(a.pinnedProcesses, cmd)
			} else {
				a.pinnedProcesses[cmd] = true
			}
			a.applyFiltersAndSort()
		}

	// Sort keys
	case "1":
		a.toggleSort(types.SortByPID)
	case "2":
		a.toggleSort(types.SortByName)
	case "3":
		a.toggleSort(types.SortByCPU)
	case "4":
		a.toggleSort(types.SortByMemory)
	case "5":
		a.toggleSort(types.SortByStatus)
	case "6":
		a.toggleSort(types.SortByUser)
	case "7":
		a.toggleSort(types.SortByCommand)
	case "8":
		a.toggleSort(types.SortByRunTime)
	case "9":
		a.toggleSort(types.SortByDisk)
	case "0":
		a.toggleSort(types.SortByThreads)
	case "T":
		a.treeMode = !a.treeMode
		a.applyFiltersAndSort()
	// Cycle refresh rate
	case "r":
		rates := []int{1000, 2000, 3000, 5000, 500}
		current := a.cfg.RefreshRate
		next := rates[0]
		for i, r := range rates {
			if r == current {
				next = rates[(i+1)%len(rates)]
				break
			}
		}
		a.cfg.RefreshRate = next
	}

	return a, nil
}

// handleMouseClick handles left-click events (Bubble Tea v2: MouseClickMsg)
func (a *App) handleMouseClick(x, y int, button tea.MouseButton) (tea.Model, tea.Cmd) {
	// Dismiss overlays on any click
	if a.activeOverlay != types.OverlayNone {
		a.activeOverlay = types.OverlayNone
		a.selectedProcess = nil
		a.selectedDetail = nil
		return a, nil
	}

	// Only handle left clicks
	if button != tea.MouseLeft {
		return a, nil
	}

	// Table header area (first 2 lines of table: header + border)
	tableHeaderY := a.tableStartLine + 1 // +1 for top border
	tableDataY := tableHeaderY + 2       // header row + separator

	// Click on table header → sort by column
	if y == tableHeaderY {
		zones := a.processTable.ColumnHitZones()
		for _, z := range zones {
			if x >= z.StartX && x < z.EndX {
				a.toggleSort(z.Field)
				break
			}
		}
		return a, nil
	}

	// Click on table data row → select; double-click → details
	if y >= tableDataY {
		row := (y - tableDataY) + a.scrollOffset
		procs := a.pageProcesses()
		if row >= 0 && row < len(procs) {
			now := time.Now()
			if row == a.lastClickRow && now.Sub(a.lastClickTime) < 400*time.Millisecond {
				p := procs[row]
				a.selectedProcess = &p
				a.selectedDetail = nil
				a.activeOverlay = types.OverlayProcessDetails
				a.lastClickTime = time.Time{}
				return a, a.fetchProcessDetail(p.PID)
			}
			a.cursor = row
			a.lastClickRow = row
			a.lastClickTime = now
		}
		return a, nil
	}

	return a, nil
}

// handleMouseWheel handles scroll events (Bubble Tea v2: MouseWheelMsg)
func (a *App) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if a.activeOverlay != types.OverlayNone {
		return a, nil
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		a.moveCursor(-3)
	case tea.MouseWheelDown:
		a.moveCursor(3)
	}

	return a, nil
}

// handleMouseMotion handles hover events (Bubble Tea v2: MouseMotionMsg)
func (a *App) handleMouseMotion(msg tea.MouseMotionMsg) (tea.Model, tea.Cmd) {
	if a.activeOverlay != types.OverlayNone {
		return a, nil
	}

	row := a.mouseYToRow(msg.Y)
	if row >= 0 {
		a.cursor = row
	}

	return a, nil
}

// mouseYToRow converts a Y coordinate to a process row index
// With lipgloss/table: border(1) + header(1) + border(1) + data rows
func (a *App) mouseYToRow(y int) int {
	tableDataY := a.tableStartLine + 3 // top border + header + separator

	if y < tableDataY {
		return -1
	}

	row := (y - tableDataY) + a.scrollOffset
	procs := a.pageProcesses()
	if row >= 0 && row < len(procs) {
		return row
	}
	return -1
}

// Commands

// fetchProcesses calls the native Go monitor directly — no FFI, no JSON.
func (a *App) fetchProcesses() tea.Cmd {
	return func() tea.Msg {
		// Refresh monitor state (direct syscalls)
		a.mon.Refresh()

		// Convert monitor types to app types (zero-copy struct assignment)
		monProcs := a.mon.Processes()
		procs := make([]types.Process, len(monProcs))
		for i, p := range monProcs {
			procs[i] = types.Process{
				PID:           p.PID,
				PPID:          p.PPID,
				Name:          p.Name,
				CPUUsage:      p.CPUUsage,
				MemoryUsage:   p.MemoryUsage,
				Status:        p.Status,
				User:          p.User,
				Command:       p.Command,
				Threads:       p.Threads,
				Root:          p.Root,
				VirtualMemory: p.VirtualMemory,
				StartTime:     p.StartTime,
				RunTime:       p.RunTime,
				DiskRead:      p.DiskRead,
				DiskWrite:     p.DiskWrite,
				SessionID:     p.SessionID,
			}
		}

		monStats := a.mon.Stats()
		stats := types.SystemStats{
			CPUBrand:       monStats.CPUBrand,
			CPUUsage:       monStats.CPUUsage,
			MemoryTotal:    monStats.MemoryTotal,
			MemoryUsed:     monStats.MemoryUsed,
			MemoryFree:     monStats.MemoryFree,
			MemoryCached:   monStats.MemoryCached,
			Uptime:         monStats.Uptime,
			LoadAvg:        monStats.LoadAvg,
			NetworkRxBytes: monStats.NetworkRxBytes,
			NetworkTxBytes: monStats.NetworkTxBytes,
			DiskTotalBytes: monStats.DiskTotalBytes,
			DiskUsedBytes:  monStats.DiskUsedBytes,
			DiskFreeBytes:  monStats.DiskFreeBytes,
			Hostname:       monStats.Hostname,
			OSVersion:      monStats.OSVersion,
			KernelVersion:  monStats.KernelVersion,
			ProcessCount:   len(procs),
		}

		return ProcessDataMsg{Processes: procs, Stats: stats}
	}
}

func (a *App) fetchProcessDetail(pid uint32) tea.Cmd {
	return func() tea.Msg {
		detail := a.mon.GetProcessDetail(pid)
		if detail == nil {
			return ProcessDetailMsg{Err: nil}
		}
		return ProcessDetailMsg{Detail: &types.ProcessDetail{
			PID:           detail.PID,
			Environ:       detail.Environ,
			Root:          detail.Root,
			VirtualMemory: detail.VirtualMemory,
		}}
	}
}

func (a *App) tickCmd() tea.Cmd {
	return tea.Tick(time.Duration(a.cfg.RefreshRate)*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (a *App) killProcess(pid uint32) tea.Cmd {
	return func() tea.Msg {
		success := a.mon.KillProcess(pid)
		return KillResultMsg{PID: pid, Success: success}
	}
}

// Helpers
func (a *App) applyFiltersAndSort() {
	filtered := filter.FilterProcesses(a.processes, a.searchTerm, a.filterConfig)
	sorted := filter.SortProcesses(filtered, a.sortConfig, a.pinnedProcesses)
	if a.treeMode {
		a.filteredProcs = filter.BuildProcessTree(sorted)
	} else {
		a.filteredProcs = sorted
	}

	pageProcs := a.pageProcesses()
	if a.cursor >= len(pageProcs) {
		a.cursor = len(pageProcs) - 1
		if a.cursor < 0 {
			a.cursor = 0
		}
	}
}

func (a *App) toggleSort(field types.SortField) {
	if a.sortConfig.Field == field {
		if a.sortConfig.Direction == types.SortAsc {
			a.sortConfig.Direction = types.SortDesc
		} else {
			a.sortConfig.Direction = types.SortAsc
		}
	} else {
		a.sortConfig.Field = field
		a.sortConfig.Direction = types.SortDesc
	}
	a.applyFiltersAndSort()
}

func (a *App) moveCursor(delta int) {
	procs := a.pageProcesses()
	a.cursor += delta
	if a.cursor < 0 {
		a.cursor = 0
	}
	if a.cursor >= len(procs) {
		a.cursor = len(procs) - 1
		if a.cursor < 0 {
			a.cursor = 0
		}
	}

	visible := a.visibleRows()
	if a.cursor < a.scrollOffset {
		a.scrollOffset = a.cursor
	}
	if a.cursor >= a.scrollOffset+visible {
		a.scrollOffset = a.cursor - visible + 1
	}
}

func (a *App) pageProcesses() []types.Process {
	return a.filteredProcs
}

func (a *App) tableHeight() int {
	h := a.height - a.tableStartLine
	if h < 5 {
		return 5
	}
	return h
}

func (a *App) visibleRows() int {
	h := a.tableHeight()
	if h < 1 {
		return 10
	}
	// With lipgloss/table, each row is 1 line (borders handled by table)
	return h - 4 // subtract borders + header
}

func (a *App) cycleTheme() {
	names := theme.ThemeNames()
	for i, name := range names {
		if name == a.theme.Name {
			next := (i + 1) % len(names)
			a.theme = theme.GetTheme(names[next])
			a.cfg.Theme = names[next]
			a.statsBar.SetTheme(a.theme)
			a.processTable.SetTheme(a.theme)
			a.toolbar.SetTheme(a.theme)
			a.helpView.SetTheme(a.theme)
			a.detailsView.SetTheme(a.theme)
			a.killConfirm.SetTheme(a.theme)
			a.filterPanel.SetTheme(a.theme)
			a.columnPanel.SetTheme(a.theme)
			config.Save(a.cfg)
			return
		}
	}
}

func (a *App) applyTheme(name string) {
	a.theme = theme.GetTheme(name)
	a.cfg.Theme = name
	a.statsBar.SetTheme(a.theme)
	a.processTable.SetTheme(a.theme)
	a.toolbar.SetTheme(a.theme)
	a.helpView.SetTheme(a.theme)
	a.detailsView.SetTheme(a.theme)
	a.killConfirm.SetTheme(a.theme)
	a.filterPanel.SetTheme(a.theme)
	a.columnPanel.SetTheme(a.theme)
	a.themePanel.SetTheme(a.theme)
	a.footer.SetTheme(a.theme)
	config.Save(a.cfg)
}

// Filter panel helpers
func (a *App) toggleFilterLine(line int) {
	switch line {
	case 0:
		a.filterConfig.CPU.Enabled = !a.filterConfig.CPU.Enabled
	case 1:
		a.filterConfig.RAM.Enabled = !a.filterConfig.RAM.Enabled
	case 2:
		a.filterConfig.Runtime.Enabled = !a.filterConfig.Runtime.Enabled
	case 3:
		a.filterConfig.Status.Enabled = !a.filterConfig.Status.Enabled
		if a.filterConfig.Status.Enabled && len(a.filterConfig.Status.Values) == 0 {
			a.filterConfig.Status.Values = []string{"Running"}
		}
	}
	a.applyFiltersAndSort()
}

var operators = []string{">", "<", "=", ">=", "<="}

func (a *App) adjustFilterOperator(line, delta int) {
	getOp := func(f *filter.NumericFilter) {
		for i, op := range operators {
			if op == f.Operator {
				next := (i + delta + len(operators)) % len(operators)
				f.Operator = operators[next]
				return
			}
		}
		f.Operator = ">"
	}

	switch line {
	case 0:
		getOp(&a.filterConfig.CPU)
	case 1:
		getOp(&a.filterConfig.RAM)
	case 2:
		getOp(&a.filterConfig.Runtime)
	case 3:
		statuses := []string{"Running", "Sleeping", "Idle", "Unknown"}
		if len(a.filterConfig.Status.Values) == 0 {
			a.filterConfig.Status.Values = []string{"Running"}
		} else {
			current := a.filterConfig.Status.Values[0]
			for i, s := range statuses {
				if s == current {
					next := (i + delta + len(statuses)) % len(statuses)
					a.filterConfig.Status.Values = []string{statuses[next]}
					break
				}
			}
		}
	}
	a.applyFiltersAndSort()
}

func (a *App) adjustFilterValue(line int, delta float64) {
	adjust := func(f *filter.NumericFilter) {
		f.Value += delta
		if f.Value < 0 {
			f.Value = 0
		}
	}

	switch line {
	case 0:
		adjust(&a.filterConfig.CPU)
	case 1:
		adjust(&a.filterConfig.RAM)
	case 2:
		adjust(&a.filterConfig.Runtime)
	}
	a.applyFiltersAndSort()
}

// Column panel helpers
func (a *App) toggleColumn(line int) {
	if line < 0 || line >= len(view.AllColumns) {
		return
	}
	col := view.AllColumns[line]
	if col.Required {
		return
	}

	for i, c := range a.cfg.Columns {
		if c == col.ID {
			a.cfg.Columns = append(a.cfg.Columns[:i], a.cfg.Columns[i+1:]...)
			config.Save(a.cfg)
			return
		}
	}

	newCols := make([]string, 0, len(a.cfg.Columns)+1)
	inserted := false
	for _, ac := range view.AllColumns {
		for _, c := range a.cfg.Columns {
			if c == ac.ID {
				newCols = append(newCols, c)
			}
		}
		if ac.ID == col.ID && !inserted {
			newCols = append(newCols, col.ID)
			inserted = true
		}
	}
	seen := make(map[string]bool)
	deduped := make([]string, 0, len(newCols))
	for _, c := range newCols {
		if !seen[c] {
			seen[c] = true
			deduped = append(deduped, c)
		}
	}
	a.cfg.Columns = deduped
	config.Save(a.cfg)
}
