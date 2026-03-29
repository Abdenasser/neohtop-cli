//go:build linux

package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

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

	// Collect load average
	m.collectLoadAverage(&stats)

	// Collect system info
	m.collectSystemInfo(&stats)

	return stats
}

// collectCPUUsage reads /proc/stat and calculates per-core CPU percentage
func (m *Monitor) collectCPUUsage(elapsed float64) []float32 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return []float32{}
	}
	defer file.Close()

	var cpuUsages []float32
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// cpu line is aggregate; cpuN lines are per-core
		if fields[0] == "cpu" {
			continue // skip aggregate for now
		}

		// Parse per-core CPU line: cpuN user nice system idle iowait irq softirq steal
		user, _ := strconv.ParseUint(fields[1], 10, 64)
		nice, _ := strconv.ParseUint(fields[2], 10, 64)
		system, _ := strconv.ParseUint(fields[3], 10, 64)
		idle, _ := strconv.ParseUint(fields[4], 10, 64)
		iowait := uint64(0)
		if len(fields) > 5 {
			iowait, _ = strconv.ParseUint(fields[5], 10, 64)
		}

		cpu := cpuTimes{
			User:   user,
			System: system,
			Idle:   idle,
			Nice:   nice,
			IOWait: iowait,
		}

		// Find index of this CPU in prevCPUTimes
		coreIdx := len(cpuUsages)
		usage := float32(0.0)

		if coreIdx < len(m.prevCPUTimes) {
			prev := m.prevCPUTimes[coreIdx]
			totalDelta := (cpu.User + cpu.System + cpu.Idle + cpu.Nice + cpu.IOWait) -
				(prev.User + prev.System + prev.Idle + prev.Nice + prev.IOWait)
			busyDelta := (cpu.User + cpu.System + cpu.Nice) -
				(prev.User + prev.System + prev.Nice)

			if totalDelta > 0 {
				usage = float32(busyDelta) / float32(totalDelta) * 100.0
			}
		}

		cpuUsages = append(cpuUsages, usage)
	}

	// Store current CPU times for next refresh
	var newCPUTimes []cpuTimes
	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		if fields[0] == "cpu" {
			continue
		}

		user, _ := strconv.ParseUint(fields[1], 10, 64)
		nice, _ := strconv.ParseUint(fields[2], 10, 64)
		system, _ := strconv.ParseUint(fields[3], 10, 64)
		idle, _ := strconv.ParseUint(fields[4], 10, 64)
		iowait := uint64(0)
		if len(fields) > 5 {
			iowait, _ = strconv.ParseUint(fields[5], 10, 64)
		}

		newCPUTimes = append(newCPUTimes, cpuTimes{
			User:   user,
			System: system,
			Idle:   idle,
			Nice:   nice,
			IOWait: iowait,
		})
	}

	m.prevCPUTimes = newCPUTimes
	return cpuUsages
}

// collectMemoryStats reads /proc/meminfo and populates memory fields
func (m *Monitor) collectMemoryStats(stats *SystemStats) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer file.Close()

	var (
		memTotal      uint64
		memFree       uint64
		memAvailable  uint64
		buffers       uint64
		cached        uint64
		sReclaimable  uint64
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSuffix(parts[0], ":")
		value, _ := strconv.ParseUint(parts[1], 10, 64)

		// Convert from kB to bytes
		value *= 1024

		switch key {
		case "MemTotal":
			memTotal = value
		case "MemFree":
			memFree = value
		case "MemAvailable":
			memAvailable = value
		case "Buffers":
			buffers = value
		case "Cached":
			cached = value
		case "SReclaimable":
			sReclaimable = value
		}
	}

	stats.MemoryTotal = memTotal
	// MemAvailable is the best estimate of free memory for applications
	if memAvailable > 0 {
		stats.MemoryFree = memAvailable
	} else {
		stats.MemoryFree = memFree
	}

	// Used = Total - Available (or fallback to manual calc)
	if memAvailable > 0 {
		stats.MemoryUsed = memTotal - memAvailable
	} else {
		used := memTotal
		if memFree < used {
			used -= memFree
		}
		if buffers < used {
			used -= buffers
		}
		if cached < used {
			used -= cached
		}
		if sReclaimable < used {
			used -= sReclaimable
		}
		stats.MemoryUsed = used
	}
	stats.MemoryCached = cached
}

// collectDiskStats reads filesystem stats for the root mount
func (m *Monitor) collectDiskStats(stats *SystemStats) {
	var statfs unix.Statfs_t
	err := unix.Statfs("/", &statfs)
	if err != nil {
		return
	}

	// Total blocks and block size
	blockSize := uint64(statfs.Bsize)
	totalBlocks := statfs.Blocks
	freeBlocks := statfs.Bfree

	stats.DiskTotalBytes = uint64(totalBlocks) * blockSize
	stats.DiskFreeBytes = uint64(freeBlocks) * blockSize
	stats.DiskUsedBytes = stats.DiskTotalBytes - stats.DiskFreeBytes
}

// collectNetworkStats reads /proc/net/dev and calculates network rates
func (m *Monitor) collectNetworkStats(stats *SystemStats, elapsed float64) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return
	}
	defer file.Close()

	var rxBytes, txBytes uint64

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		// Skip first two header lines
		if lineNum <= 2 {
			continue
		}

		line := scanner.Text()
		// Format: "iface: rx_bytes rx_packets ... tx_bytes tx_packets ..."
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == ':' || r == ' '
		})

		if len(parts) < 10 {
			continue
		}

		iface := parts[0]
		// Skip loopback
		if iface == "lo" {
			continue
		}

		rx, err1 := strconv.ParseUint(parts[1], 10, 64)
		tx, err2 := strconv.ParseUint(parts[9], 10, 64)

		if err1 == nil {
			rxBytes += rx
		}
		if err2 == nil {
			txBytes += tx
		}
	}

	// Calculate rates from delta
	stats.NetworkRxBytes = 0
	stats.NetworkTxBytes = 0

	if m.prevNet.RxBytes > 0 && rxBytes >= m.prevNet.RxBytes {
		delta := rxBytes - m.prevNet.RxBytes
		stats.NetworkRxBytes = uint64(float64(delta) / elapsed)
	}

	if m.prevNet.TxBytes > 0 && txBytes >= m.prevNet.TxBytes {
		delta := txBytes - m.prevNet.TxBytes
		stats.NetworkTxBytes = uint64(float64(delta) / elapsed)
	}

	// Store for next refresh
	m.prevNet = netCounters{
		RxBytes: rxBytes,
		TxBytes: txBytes,
	}
}

// collectUptime reads /proc/uptime
func (m *Monitor) collectUptime(stats *SystemStats) {
	file, err := os.Open("/proc/uptime")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) > 0 {
			// Parse uptime in seconds as float, convert to uint64
			var uptime float64
			fmt.Sscanf(parts[0], "%f", &uptime)
			stats.Uptime = uint64(uptime)
		}
	}
}

// collectLoadAverage reads /proc/loadavg
func (m *Monitor) collectLoadAverage(stats *SystemStats) {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) >= 3 {
			fmt.Sscanf(parts[0], "%f", &stats.LoadAvg[0])
			fmt.Sscanf(parts[1], "%f", &stats.LoadAvg[1])
			fmt.Sscanf(parts[2], "%f", &stats.LoadAvg[2])
		}
	}
}

// collectSystemInfo gathers hostname, OS, kernel on Linux
func (m *Monitor) collectSystemInfo(stats *SystemStats) {
	// Hostname
	if data, err := os.ReadFile("/proc/sys/kernel/hostname"); err == nil {
		stats.Hostname = strings.TrimSpace(string(data))
	}

	// Kernel version
	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		stats.KernelVersion = strings.TrimSpace(string(data))
	}

	// OS version from /etc/os-release
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				val := strings.TrimPrefix(line, "PRETTY_NAME=")
				val = strings.Trim(val, "\"")
				stats.OSVersion = val
				break
			}
		}
	}

	// CPU brand from /proc/cpuinfo
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					stats.CPUBrand = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}
}
