// Package bridge provides CGo bindings to the Rust core library.
// DEPRECATED: This package is not currently used. NeoHtopCLI uses pure Go
// monitoring via the monitor package instead. Kept for potential future use
// if Rust-based monitoring is needed for platforms where Go's syscall coverage
// is insufficient.
package bridge

/*
#cgo LDFLAGS: -L${SRCDIR}/../../core/target/release -lneohtop_core -lm -lpthread -lresolv
#cgo darwin LDFLAGS: -framework IOKit -framework CoreFoundation
#cgo linux LDFLAGS: -ldl
#include "bridge.h"
#include <stdlib.h>
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// Handle wraps the opaque Rust state pointer
type Handle struct {
	ptr unsafe.Pointer
}

// MonitoringData is the top-level JSON response from Rust
type MonitoringData struct {
	Processes   []ProcessInfo `json:"processes"`
	SystemStats SystemStats   `json:"system_stats"`
}

// ProcessInfo mirrors the Rust ProcessInfo struct (lightweight — no environ)
type ProcessInfo struct {
	PID           uint32    `json:"pid"`
	PPID          uint32    `json:"ppid"`
	Name          string    `json:"name"`
	CPUUsage      float32   `json:"cpu_usage"`
	MemoryUsage   uint64    `json:"memory_usage"`
	Status        string    `json:"status"`
	User          string    `json:"user"`
	Command       string    `json:"command"`
	Threads       *uint32   `json:"threads"`
	Root          string    `json:"root"`
	VirtualMemory uint64    `json:"virtual_memory"`
	StartTime     uint64    `json:"start_time"`
	RunTime       uint64    `json:"run_time"`
	DiskUsage     [2]uint64 `json:"disk_usage"`
	SessionID     *uint32   `json:"session_id"`
}

// ProcessDetail contains expensive fields fetched on-demand
type ProcessDetail struct {
	PID           uint32   `json:"pid"`
	Environ       []string `json:"environ"`
	Root          string   `json:"root"`
	VirtualMemory uint64   `json:"virtual_memory"`
}

// SystemStats mirrors the Rust SystemStats struct
type SystemStats struct {
	CPUUsage       []float32  `json:"cpu_usage"`
	MemoryTotal    uint64     `json:"memory_total"`
	MemoryUsed     uint64     `json:"memory_used"`
	MemoryFree     uint64     `json:"memory_free"`
	MemoryCached   uint64     `json:"memory_cached"`
	Uptime         uint64     `json:"uptime"`
	LoadAvg        [3]float64 `json:"load_avg"`
	NetworkRxBytes uint64     `json:"network_rx_bytes"`
	NetworkTxBytes uint64     `json:"network_tx_bytes"`
	DiskTotalBytes uint64     `json:"disk_total_bytes"`
	DiskUsedBytes  uint64     `json:"disk_used_bytes"`
	DiskFreeBytes  uint64     `json:"disk_free_bytes"`
}

// Init creates a new monitoring handle
func Init() (*Handle, error) {
	ptr := C.neohtop_init()
	if ptr == nil {
		return nil, fmt.Errorf("failed to initialize neohtop core")
	}
	return &Handle{ptr: ptr}, nil
}

// GetProcesses fetches current process list and system stats from Rust
func (h *Handle) GetProcesses() (*MonitoringData, error) {
	cStr := C.neohtop_get_processes(h.ptr)
	if cStr == nil {
		return nil, fmt.Errorf("failed to get processes from core")
	}
	defer C.neohtop_free_string(cStr)

	jsonStr := C.GoString(cStr)

	var data MonitoringData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal process data: %w", err)
	}

	return &data, nil
}

// GetProcessDetail fetches detailed info (environ, etc.) for a single process.
// Called on-demand when user opens process details overlay.
func (h *Handle) GetProcessDetail(pid uint32) (*ProcessDetail, error) {
	cStr := C.neohtop_get_process_detail(h.ptr, C.uint32_t(pid))
	if cStr == nil {
		return nil, fmt.Errorf("process %d not found or detail unavailable", pid)
	}
	defer C.neohtop_free_string(cStr)

	jsonStr := C.GoString(cStr)

	var detail ProcessDetail
	if err := json.Unmarshal([]byte(jsonStr), &detail); err != nil {
		return nil, fmt.Errorf("failed to unmarshal process detail: %w", err)
	}

	return &detail, nil
}

// KillProcess terminates a process by PID
func (h *Handle) KillProcess(pid uint32) bool {
	result := C.neohtop_kill_process(h.ptr, C.uint32_t(pid))
	return result == 1
}

// Destroy releases all resources held by the handle
func (h *Handle) Destroy() {
	if h.ptr != nil {
		C.neohtop_destroy(h.ptr)
		h.ptr = nil
	}
}
