package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/abdenasser/neohtop-cli/model"
	"github.com/abdenasser/neohtop-cli/monitor"

	tea "charm.land/bubbletea/v2"
)

// version is set at build time via -ldflags
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("neohtop-cli %s\n", version)
			os.Exit(0)
		case "--json":
			runJSON()
			os.Exit(0)
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		}
	}

	// Initialize the native Go monitor (reads OS interfaces directly — no FFI)
	mon := monitor.New()
	defer mon.Destroy()

	// Create and run the Bubble Tea program
	app := model.NewApp(mon)
	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runJSON outputs a single snapshot of system stats and processes as JSON to stdout.
// Useful for scripting: neohtop-cli --json | jq '.processes[] | select(.cpu_usage > 5)'
func runJSON() {
	mon := monitor.New()
	defer mon.Destroy()

	// Refresh to collect data
	mon.Refresh()
	procs := mon.Processes()
	stats := mon.Stats()

	type jsonProcess struct {
		PID       uint32  `json:"pid"`
		PPID      uint32  `json:"ppid"`
		Name      string  `json:"name"`
		CPU       float32 `json:"cpu_usage"`
		Memory    uint64  `json:"memory_bytes"`
		Status    string  `json:"status"`
		User      string  `json:"user"`
		Command   string  `json:"command"`
		Threads   *uint32 `json:"threads,omitempty"`
		RunTime   uint64  `json:"runtime_secs"`
		DiskRead  uint64  `json:"disk_read_bytes"`
		DiskWrite uint64  `json:"disk_write_bytes"`
	}

	type jsonStats struct {
		CPUBrand      string    `json:"cpu_brand"`
		CPUUsage      []float32 `json:"cpu_usage_per_core"`
		MemoryTotal   uint64    `json:"memory_total"`
		MemoryUsed    uint64    `json:"memory_used"`
		MemoryFree    uint64    `json:"memory_free"`
		Uptime        uint64    `json:"uptime_secs"`
		LoadAvg       [3]float64 `json:"load_avg"`
		NetworkRx     uint64    `json:"network_rx_bytes"`
		NetworkTx     uint64    `json:"network_tx_bytes"`
		DiskTotal     uint64    `json:"disk_total_bytes"`
		DiskUsed      uint64    `json:"disk_used_bytes"`
		DiskFree      uint64    `json:"disk_free_bytes"`
		Hostname      string    `json:"hostname"`
		OSVersion     string    `json:"os_version"`
		KernelVersion string    `json:"kernel_version"`
		ProcessCount  int       `json:"process_count"`
	}

	type jsonOutput struct {
		Version   string        `json:"version"`
		System    jsonStats     `json:"system"`
		Processes []jsonProcess `json:"processes"`
	}

	jsonProcs := make([]jsonProcess, 0, len(procs))
	for _, p := range procs {
		jp := jsonProcess{
			PID:       p.PID,
			PPID:      p.PPID,
			Name:      p.Name,
			CPU:       p.CPUUsage,
			Memory:    p.MemoryUsage,
			Status:    p.Status,
			User:      p.User,
			Command:   p.Command,
			RunTime:   p.RunTime,
			DiskRead:  p.DiskRead,
			DiskWrite: p.DiskWrite,
		}
		if p.Threads != nil {
			jp.Threads = p.Threads
		}
		jsonProcs = append(jsonProcs, jp)
	}

	output := jsonOutput{
		Version: version,
		System: jsonStats{
			CPUBrand:      stats.CPUBrand,
			CPUUsage:      stats.CPUUsage,
			MemoryTotal:   stats.MemoryTotal,
			MemoryUsed:    stats.MemoryUsed,
			MemoryFree:    stats.MemoryFree,
			Uptime:        stats.Uptime,
			LoadAvg:       stats.LoadAvg,
			NetworkRx:     stats.NetworkRxBytes,
			NetworkTx:     stats.NetworkTxBytes,
			DiskTotal:     stats.DiskTotalBytes,
			DiskUsed:      stats.DiskUsedBytes,
			DiskFree:      stats.DiskFreeBytes,
			Hostname:      stats.Hostname,
			OSVersion:     stats.OSVersion,
			KernelVersion: stats.KernelVersion,
			ProcessCount:  len(procs),
		},
		Processes: jsonProcs,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`neohtop-cli %s — a beautiful terminal process monitor

Usage:
  neohtop-cli            Launch the interactive TUI
  neohtop-cli --json     Output system stats + processes as JSON (pipe to jq)
  neohtop-cli --version  Print version
  neohtop-cli --help     Show this help

Examples:
  neohtop-cli --json | jq '.processes[] | select(.cpu_usage > 5)'
  neohtop-cli --json | jq '.system.memory_used'
  neohtop-cli --json | jq '[.processes[] | {name, cpu: .cpu_usage}] | sort_by(.cpu) | reverse[:10]'
`, version)
}
