package types

// Process represents a system process (lightweight — no environ)
type Process struct {
	PID           uint32
	PPID          uint32
	Name          string
	CPUUsage      float32
	MemoryUsage   uint64
	Status        string
	User          string
	Command       string
	Threads       *uint32
	Root          string
	VirtualMemory uint64
	StartTime     uint64
	RunTime       uint64
	DiskRead      uint64
	DiskWrite     uint64
	SessionID     *uint32

	// Tree view fields (set by BuildProcessTree)
	TreePrefix string // e.g. "├─ ", "│  └─ "
	TreeDepth  int    // nesting level (0 = root)
}

// ProcessDetail contains expensive fields fetched on-demand
type ProcessDetail struct {
	PID           uint32
	Environ       []string
	Root          string
	VirtualMemory uint64
}

// SystemStats holds system-wide monitoring data
type SystemStats struct {
	CPUBrand       string
	CPUUsage       []float32
	MemoryTotal    uint64
	MemoryUsed     uint64
	MemoryFree     uint64
	MemoryCached   uint64
	Uptime         uint64
	LoadAvg        [3]float64
	NetworkRxBytes uint64
	NetworkTxBytes uint64
	DiskTotalBytes uint64
	DiskUsedBytes  uint64
	DiskFreeBytes  uint64
	Hostname       string
	OSVersion      string
	KernelVersion  string
	ProcessCount   int
}

// SortField represents the field to sort by
type SortField int

const (
	SortByPID SortField = iota
	SortByName
	SortByCPU
	SortByMemory
	SortByStatus
	SortByUser
	SortByCommand
	SortByRunTime
	SortByDisk
	SortByThreads
)

func (s SortField) String() string {
	switch s {
	case SortByPID:
		return "PID"
	case SortByName:
		return "Name"
	case SortByCPU:
		return "CPU%"
	case SortByMemory:
		return "Memory"
	case SortByStatus:
		return "Status"
	case SortByUser:
		return "User"
	case SortByCommand:
		return "Command"
	case SortByRunTime:
		return "Runtime"
	case SortByDisk:
		return "Disk I/O"
	case SortByThreads:
		return "Threads"
	default:
		return "?"
	}
}

// SortDirection represents ascending or descending
type SortDirection int

const (
	SortAsc SortDirection = iota
	SortDesc
)

// SortConfig holds the current sort state
type SortConfig struct {
	Field     SortField
	Direction SortDirection
}

// OverlayType represents which overlay is currently active
type OverlayType int

const (
	OverlayNone OverlayType = iota
	OverlayHelp
	OverlayProcessDetails
	OverlayKillConfirm
	OverlayFilters
	OverlayColumns
	OverlayThemes
)
