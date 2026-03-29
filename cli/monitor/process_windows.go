//go:build windows

package monitor

import (
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	// Cache for SID to username lookups
	sidCache = make(map[string]string)
	sidMutex sync.Mutex

	// Windows API imports
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	advapi32 = windows.NewLazySystemDLL("advapi32.dll")
	psapi    = windows.NewLazySystemDLL("psapi.dll")

	// Kernel32 functions
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")
	procGetProcessTimes          = kernel32.NewProc("GetProcessTimes")
	procTerminateProcess         = kernel32.NewProc("TerminateProcess")
	procGetTickCount64           = kernel32.NewProc("GetTickCount64")

	// Psapi functions
	procGetProcessMemoryInfo = psapi.NewProc("GetProcessMemoryInfo")
)

// PROCESSENTRY32 from Windows API
type PROCESSENTRY32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [windows.MAX_PATH]uint16
}

// PROCESS_MEMORY_COUNTERS structure for GetProcessMemoryInfo
type PROCESS_MEMORY_COUNTERS struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

// collectProcesses enumerates all running processes on Windows
func (m *Monitor) collectProcesses(elapsed float64) []ProcessInfo {
	var processes []ProcessInfo

	// Create a snapshot of all processes
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return processes
	}
	defer windows.CloseHandle(handle)

	// Get system boot time for StartTime calculation
	bootTime := getWindowsBootTime()

	// Iterate through processes
	entry := PROCESSENTRY32{Size: uint32(unsafe.Sizeof(PROCESSENTRY32{}))}
	ret, _, _ := procProcess32First.Call(uintptr(handle), uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return processes
	}

	for {
		pid := entry.ProcessID

		// Skip the idle process
		if pid == 0 {
			goto nextProcess
		}

		// Open the process to get more information
		procHandle, err := windows.OpenProcess(
			windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ,
			false,
			pid,
		)
		if err != nil {
			goto nextProcess
		}

		threadCount := entry.Threads
		proc := ProcessInfo{
			PID:     pid,
			PPID:    entry.ParentProcessID,
			Name:    windows.UTF16ToString(entry.ExeFile[:]),
			Threads: &threadCount,
		}

		// Get process times for CPU usage calculation
		var creationTime, exitTime, kernelTime, userTime windows.Filetime
		ret, _, _ := procGetProcessTimes.Call(
			uintptr(procHandle),
			uintptr(unsafe.Pointer(&creationTime)),
			uintptr(unsafe.Pointer(&exitTime)),
			uintptr(unsafe.Pointer(&kernelTime)),
			uintptr(unsafe.Pointer(&userTime)),
		)

		if ret != 0 {
			// Convert Filetime to 100-nanosecond intervals (Windows uses this natively)
			userNano := userTime.Nanoseconds()
			kernelNano := kernelTime.Nanoseconds()

			// Calculate CPU usage from delta
			proc.CPUUsage = calculateWindowsCPUUsage(m, pid, userNano, kernelNano, elapsed)

			// Calculate start time (CreationTime is in Filetime format)
			if bootTime > 0 {
				creationUnix := filetime2Unix(creationTime)
				proc.StartTime = creationUnix
				if creationUnix > 0 {
					now := uint64(time.Now().Unix())
					if now > creationUnix {
						proc.RunTime = now - creationUnix
					}
				}
			}
		}

		// Get memory usage
		proc.MemoryUsage = getProcessMemoryUsage(procHandle)

		// Get command line from process (basic, using ExeFile)
		proc.Command = windows.UTF16ToString(entry.ExeFile[:])

		// Get username from process token
		proc.User = getProcessUser(procHandle)

		// Status is always "Running" on Windows (no good way to get state without WMI)
		proc.Status = "Running"

		windows.CloseHandle(procHandle)
		processes = append(processes, proc)

	nextProcess:
		ret, _, _ = procProcess32Next.Call(uintptr(handle), uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return processes
}

// getProcessDetail returns detailed information for a single process
func (m *Monitor) getProcessDetail(pid uint32) *ProcessDetail {
	detail := &ProcessDetail{
		PID: pid,
	}

	procHandle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ,
		false,
		pid,
	)
	if err != nil {
		return detail
	}
	defer windows.CloseHandle(procHandle)

	// Get virtual memory (working set size)
	detail.VirtualMemory = getProcessMemoryUsage(procHandle)

	// Environ is not easily accessible on Windows without external tools
	// For now, we'll return an empty list
	detail.Environ = []string{}

	return detail
}

// killProcess terminates a process on Windows
func (m *Monitor) killProcess(pid uint32) bool {
	procHandle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(procHandle)

	// TerminateProcess with exit code 1
	ret, _, _ := procTerminateProcess.Call(uintptr(procHandle), uintptr(1))
	return ret != 0
}

// Helper functions

// calculateWindowsCPUUsage computes per-process CPU percentage
func calculateWindowsCPUUsage(m *Monitor, pid uint32, userNano int64, kernelNano int64, elapsed float64) float32 {
	// Total CPU time in nanoseconds
	totalNano := userNano + kernelNano

	// Get previous values
	prev, hasPrev := m.prevProcTimes[pid]
	prevTotal := int64(prev.User + prev.System)

	cpuUsage := float32(0.0)

	if hasPrev && totalNano > prevTotal && elapsed > 0 {
		// Delta in nanoseconds
		deltaNano := totalNano - prevTotal
		// Convert to seconds
		deltaSeconds := float64(deltaNano) / 1e9
		// CPU usage: (deltaSeconds / elapsed) * 100
		// Multiply by number of cores to get per-core percentage
		numCores := float64(len(m.prevCPUTimes))
		if numCores == 0 {
			numCores = 1
		}
		cpuUsage = float32((deltaSeconds / elapsed) * 100.0 / numCores)
	}

	// Store for next refresh
	m.prevProcTimes[pid] = procTimes{
		User:   uint64(userNano),
		System: uint64(kernelNano),
	}

	return cpuUsage
}

// getProcessMemoryUsage gets the working set size (RSS equivalent)
func getProcessMemoryUsage(procHandle windows.Handle) uint64 {
	memCounters := PROCESS_MEMORY_COUNTERS{
		CB: uint32(unsafe.Sizeof(PROCESS_MEMORY_COUNTERS{})),
	}

	ret, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(procHandle),
		uintptr(unsafe.Pointer(&memCounters)),
		uintptr(memCounters.CB),
	)

	if ret != 0 {
		return uint64(memCounters.WorkingSetSize)
	}

	return 0
}

// getProcessUser retrieves the username for the process owner
func getProcessUser(procHandle windows.Handle) string {
	var tokenHandle windows.Token

	// Open process token
	err := windows.OpenProcessToken(procHandle, windows.TOKEN_QUERY, &tokenHandle)
	if err != nil {
		return "Unknown"
	}
	defer tokenHandle.Close()

	// Get the user SID
	owner, err := tokenHandle.GetTokenUser()
	if err != nil {
		return "Unknown"
	}

	if owner == nil || owner.User.Sid == nil {
		return "Unknown"
	}

	// Convert SID to string for caching
	sidStr := owner.User.Sid.String()

	// Check cache first
	sidMutex.Lock()
	if username, ok := sidCache[sidStr]; ok {
		sidMutex.Unlock()
		return username
	}
	sidMutex.Unlock()

	// Look up the account name
	account, domain, _, err := owner.User.Sid.LookupAccount("")
	if err != nil {
		// Fallback to SID string
		sidMutex.Lock()
		sidCache[sidStr] = sidStr
		sidMutex.Unlock()
		return sidStr
	}

	username := account
	if domain != "" {
		username = domain + "\\" + account
	}

	// Cache the result
	sidMutex.Lock()
	sidCache[sidStr] = username
	sidMutex.Unlock()

	return username
}

// getWindowsBootTime gets the system boot time in Unix timestamp
func getWindowsBootTime() uint64 {
	// GetTickCount64 returns milliseconds since last boot
	ret, _, _ := procGetTickCount64.Call()
	ticksMs := int64(ret)

	// Current time in seconds
	now := int64(time.Now().Unix())

	// Boot time = now - (ticks / 1000)
	bootTime := now - (ticksMs / 1000)

	if bootTime > 0 {
		return uint64(bootTime)
	}

	return 0
}

// filetime2Unix converts Windows FILETIME to Unix timestamp
func filetime2Unix(ft windows.Filetime) uint64 {
	// FILETIME is in 100-nanosecond intervals since 1601-01-01
	// Unix epoch is 1970-01-01
	// Difference is 116444736000000000 (in 100-ns intervals)

	const (
		epochDiff = 116444736000000000 // 100-ns intervals between 1601 and 1970
		intervals = 10000000            // 100-ns intervals per second
	)

	ft100ns := int64(ft.HighDateTime)<<32 | int64(ft.LowDateTime)

	if ft100ns < epochDiff {
		return 0
	}

	return uint64((ft100ns - epochDiff) / intervals)
}
