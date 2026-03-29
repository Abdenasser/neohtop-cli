//go:build windows

package monitor

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	// Windows API DLLs
	kernel32Dll = windows.NewLazySystemDLL("kernel32.dll")
	ntdll       = windows.NewLazySystemDLL("ntdll.dll")

	// Kernel32 functions
	procGlobalMemoryStatusEx   = kernel32Dll.NewProc("GlobalMemoryStatusEx")
	procGetDiskFreeSpaceEx     = kernel32Dll.NewProc("GetDiskFreeSpaceExW")
	procGetComputerNameEx      = kernel32Dll.NewProc("GetComputerNameExW")
	procGetVersionEx           = kernel32Dll.NewProc("GetVersionExW")
	procGetSystemInfo          = kernel32Dll.NewProc("GetSystemInfo")
	procGetLogicalProcessorInformationEx = kernel32Dll.NewProc("GetLogicalProcessorInformationEx")

	// NTDLL functions for processor info
	procNtQuerySystemInformation = ntdll.NewProc("NtQuerySystemInformation")
)

// MEMORYSTATUSEX structure
type MEMORYSTATUSEX struct {
	DwLength                uint32
	DwMemoryLoad            uint32
	UllTotalPhys            uint64
	UllAvailPhys            uint64
	UllTotalPageFile        uint64
	UllAvailPageFile        uint64
	UllTotalVirtual         uint64
	UllAvailVirtual         uint64
	UllAvailExtendedVirtual uint64
}

// SYSTEM_INFO structure
type SYSTEM_INFO struct {
	WProcessorArchitecture     uint16
	WReserved                  uint16
	DwPageSize                 uint32
	LpMinimumApplicationAddress uintptr
	LpMaximumApplicationAddress uintptr
	DwActiveProcessorMask      uintptr
	DwNumberOfProcessors       uint32
	DwProcessorType            uint32
	DwAllocationGranularity    uint32
	WProcessorLevel            uint16
	WProcessorRevision         uint16
}

// SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION for NtQuerySystemInformation
type SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION struct {
	IdleTime   int64
	KernelTime int64
	UserTime   int64
	DpcTime    int64
	InterruptTime int64
	InterruptCount uint32
}

// collectSystemStats gathers CPU, memory, disk, network, uptime, and load average stats
func (m *Monitor) collectSystemStats(elapsed float64) SystemStats {
	stats := SystemStats{}

	// Collect CPU usage per core
	stats.CPUUsage = m.collectCPUUsage(elapsed)

	// Collect memory stats
	m.collectMemoryStats(&stats)

	// Collect disk stats
	m.collectDiskStats(&stats)

	// Collect network stats
	m.collectNetworkStats(&stats, elapsed)

	// Collect uptime
	m.collectUptime(&stats)

	// Collect system info
	m.collectSystemInfo(&stats)

	return stats
}

// collectCPUUsage gets per-core CPU usage on Windows
func (m *Monitor) collectCPUUsage(elapsed float64) []float32 {
	var cpuUsages []float32

	// Use NtQuerySystemInformation to get processor times
	// SystemProcessorPerformanceInformation = 8
	const systemProcessorPerformanceInformation = 8

	// First, get the number of processors
	sysInfo := SYSTEM_INFO{}
	procGetSystemInfo.Call(uintptr(unsafe.Pointer(&sysInfo)))
	numProcessors := int(sysInfo.DwNumberOfProcessors)

	if numProcessors <= 0 {
		numProcessors = 1
	}

	// Allocate buffer for processor info
	bufferSize := uintptr(unsafe.Sizeof(SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION{})) * uintptr(numProcessors)
	buffer := make([]byte, bufferSize)

	// Query system information
	var returnLength uint32
	status, _, _ := procNtQuerySystemInformation.Call(
		uintptr(systemProcessorPerformanceInformation),
		uintptr(unsafe.Pointer(&buffer[0])),
		bufferSize,
		uintptr(unsafe.Pointer(&returnLength)),
	)

	// Status 0 = success in NTSTATUS
	if status != 0 {
		// Fallback: return single CPU usage
		return []float32{0}
	}

	// Parse the processor performance information
	perfInfoSize := unsafe.Sizeof(SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION{})
	for i := 0; i < numProcessors; i++ {
		offset := uintptr(i) * perfInfoSize
		if offset+perfInfoSize > bufferSize {
			break
		}

		perfInfo := (*SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION)(unsafe.Pointer(&buffer[offset]))

		// Calculate CPU usage from delta
		usage := float32(0.0)

		if i < len(m.prevCPUTimes) {
			prev := m.prevCPUTimes[i]

			// Total time = IdleTime + KernelTime + UserTime
			totalTime := perfInfo.IdleTime + perfInfo.KernelTime + perfInfo.UserTime
			busyTime := perfInfo.KernelTime + perfInfo.UserTime

			prevTotalTime := int64(prev.Idle + prev.System + prev.User)
			prevBusyTime := int64(prev.System + prev.User)

			timeDelta := totalTime - prevTotalTime
			busyDelta := busyTime - prevBusyTime

			if timeDelta > 0 {
				usage = float32(busyDelta) / float32(timeDelta) * 100.0
			}
		}

		cpuUsages = append(cpuUsages, usage)
	}

	// Store current values for next refresh
	var newCPUTimes []cpuTimes
	for i := 0; i < numProcessors && i*int(perfInfoSize) < len(buffer); i++ {
		offset := uintptr(i) * perfInfoSize
		if offset+perfInfoSize > bufferSize {
			break
		}

		perfInfo := (*SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION)(unsafe.Pointer(&buffer[offset]))
		newCPUTimes = append(newCPUTimes, cpuTimes{
			User:   uint64(perfInfo.UserTime),
			System: uint64(perfInfo.KernelTime),
			Idle:   uint64(perfInfo.IdleTime),
		})
	}

	m.prevCPUTimes = newCPUTimes

	return cpuUsages
}

// collectMemoryStats gathers memory information
func (m *Monitor) collectMemoryStats(stats *SystemStats) {
	memStatus := MEMORYSTATUSEX{
		DwLength: uint32(unsafe.Sizeof(MEMORYSTATUSEX{})),
	}

	ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		// Failed to get memory status
		return
	}

	stats.MemoryTotal = memStatus.UllTotalPhys
	stats.MemoryFree = memStatus.UllAvailPhys
	stats.MemoryUsed = stats.MemoryTotal - stats.MemoryFree
	stats.MemoryCached = 0 // Windows doesn't expose cache the same way as Linux
}

// collectDiskStats gathers disk space information for the root drive
func (m *Monitor) collectDiskStats(stats *SystemStats) {
	// Get disk space for C:\ drive (or the system drive)
	systemDrive := "C:\\"

	// Try to get the actual system drive
	if envDrive := os.Getenv("SystemDrive"); envDrive != "" {
		systemDrive = envDrive + "\\"
	}

	// Convert to UTF-16
	pathPtr, err := windows.UTF16PtrFromString(systemDrive)
	if err != nil {
		return
	}

	var (
		totalBytes  uint64
		freeBytes   uint64
		availBytes  uint64
	)

	ret, _, _ := procGetDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&availBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&freeBytes)),
	)

	if ret != 0 {
		stats.DiskTotalBytes = totalBytes
		stats.DiskFreeBytes = freeBytes
		stats.DiskUsedBytes = totalBytes - freeBytes
	}
}

// collectNetworkStats gathers network interface statistics
// Windows network stats collection is complex; for simplicity, we'll use a basic approach
func (m *Monitor) collectNetworkStats(stats *SystemStats, elapsed float64) {
	// Getting detailed network stats on Windows requires WMI or specialized APIs
	// For now, we'll return zeroes or cached values
	// A more complete implementation would use GetIfTable2 or WMI

	// If we had previous values, calculate rates
	if m.prevNet.RxBytes > 0 || m.prevNet.TxBytes > 0 {
		// Keep the same values (simplified)
		stats.NetworkRxBytes = m.prevNet.RxBytes
		stats.NetworkTxBytes = m.prevNet.TxBytes
	}

	// Store for next refresh (would be updated with actual values in a full implementation)
	m.prevNet = netCounters{
		RxBytes: stats.NetworkRxBytes,
		TxBytes: stats.NetworkTxBytes,
	}
}

// collectUptime gets system uptime in seconds
func (m *Monitor) collectUptime(stats *SystemStats) {
	// Use GetTickCount64 which returns milliseconds since boot
	getTickCount64 := kernel32Dll.NewProc("GetTickCount64")

	ret, _, _ := getTickCount64.Call()
	ticksMs := int64(ret)

	// Convert to seconds
	stats.Uptime = uint64(ticksMs / 1000)
}

// collectSystemInfo gathers hostname, OS version, and CPU info
func (m *Monitor) collectSystemInfo(stats *SystemStats) {
	// Hostname
	hostname, err := os.Hostname()
	if err == nil {
		stats.Hostname = hostname
	}

	// OS Version using GetVersionEx
	osvi := struct {
		OSVersionInfoSize uint32
		MajorVersion      uint32
		MinorVersion      uint32
		BuildNumber       uint32
		PlatformID        uint32
		CSDVersion        [128]uint16
	}{
		OSVersionInfoSize: uint32(unsafe.Sizeof(struct {
			OSVersionInfoSize uint32
			MajorVersion      uint32
			MinorVersion      uint32
			BuildNumber       uint32
			PlatformID        uint32
			CSDVersion        [128]uint16
		}{})),
	}

	ret, _, _ := procGetVersionEx.Call(uintptr(unsafe.Pointer(&osvi)))
	if ret != 0 {
		// Format Windows version
		stats.OSVersion = fmt.Sprintf("Windows %d.%d", osvi.MajorVersion, osvi.MinorVersion)
		stats.KernelVersion = fmt.Sprintf("%d", osvi.BuildNumber)
	}

	// Get CPU information from registry or WMI
	// For simplicity, we'll try to read from the registry
	stats.CPUBrand = getProcessorName()

	// Get process count from tasklist or WMI (simplified)
	stats.ProcessCount = getProcessCount()
}

// getProcessorName retrieves the processor name from the registry
func getProcessorName() string {
	var key windows.Handle
	err := windows.RegOpenKeyEx(
		windows.HKEY_LOCAL_MACHINE,
		windows.StringToUTF16Ptr("HARDWARE\\DESCRIPTION\\System\\CentralProcessor\\0"),
		0,
		windows.KEY_READ,
		&key,
	)
	if err != nil {
		return "Unknown Processor"
	}
	defer windows.RegCloseKey(key)

	// Read the ProcessorNameString value
	buf := make([]uint16, 256)
	bufLen := uint32(len(buf) * 2)

	err = windows.RegQueryValueEx(
		key,
		windows.StringToUTF16Ptr("ProcessorNameString"),
		nil,
		nil,
		(*byte)(unsafe.Pointer(&buf[0])),
		&bufLen,
	)

	if err != nil {
		return "Unknown Processor"
	}

	return windows.UTF16ToString(buf)
}

// getProcessCount returns the number of running processes
// This is a simplified implementation; a full implementation would enumerate all processes
func getProcessCount() int {
	// Use CreateToolhelp32Snapshot to count processes
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(handle)

	count := 0
	entry := PROCESSENTRY32{Size: uint32(unsafe.Sizeof(PROCESSENTRY32{}))}

	// Process32First
	procGetFirstProcess := kernel32Dll.NewProc("Process32FirstW")
	procGetNextProcess := kernel32Dll.NewProc("Process32NextW")

	ret, _, _ := procGetFirstProcess.Call(uintptr(handle), uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return 0
	}

	for {
		if entry.ProcessID != 0 {
			count++
		}

		ret, _, _ := procGetNextProcess.Call(uintptr(handle), uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return count
}
