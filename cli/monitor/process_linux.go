//go:build linux

package monitor

import (
	"bufio"
	"bytes"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var (
	uidCache = make(map[uint32]string)
	uidMutex sync.Mutex
)

// collectProcesses scans /proc and returns current process list
func (m *Monitor) collectProcesses(elapsed float64) []ProcessInfo {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return []ProcessInfo{}
	}

	var processes []ProcessInfo
	bootTime := getBootTime()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			continue
		}

		pidUint32 := uint32(pid)
		stat := readProcStat(pidUint32)
		if stat == nil {
			continue
		}

		proc := ProcessInfo{
			PID:    pidUint32,
			PPID:   stat.ppid,
			Name:   stat.comm,
			Status: parseStatus(stat.state),
			Threads: &stat.numThreads,
		}

		// CPU usage calculation
		proc.CPUUsage = calculateCPUUsage(m, pidUint32, stat, elapsed)

		// Memory: RSS in bytes = rss_pages * page_size
		pageSize := os.Getpagesize()
		proc.MemoryUsage = uint64(stat.rss) * uint64(pageSize)
		proc.VirtualMemory = stat.vsize

		// Command line
		proc.Command = readCmdline(pidUint32)

		// User
		proc.User = lookupUID(stat.uid)

		// Root symlink
		proc.Root = readRoot(pidUint32)

		// Start time calculation
		if bootTime > 0 {
			ticksPerSec := uint64(100) // Most Linux systems use 100 Hz
			startTimeTicks := stat.starttime
			proc.StartTime = bootTime + (startTimeTicks / ticksPerSec)
			if stat.currentTime > startTimeTicks {
				proc.RunTime = (stat.currentTime - startTimeTicks) / ticksPerSec
			}
		}

		// Disk I/O
		proc.DiskRead, proc.DiskWrite = calculateDiskIO(m, pidUint32, elapsed)

		// Session ID
		proc.SessionID = &stat.sid

		processes = append(processes, proc)
	}

	return processes
}

// procStat holds parsed /proc/[pid]/stat fields
type procStat struct {
	pid        uint32
	comm       string
	state      byte
	ppid       uint32
	uid        uint32
	numThreads uint32
	vsize      uint64
	rss        uint64
	utime      uint64
	stime      uint64
	starttime  uint64
	currentTime uint64
	sid        uint32
}

// readProcStat parses /proc/[pid]/stat and /proc/[pid]/status
func readProcStat(pid uint32) *procStat {
	statPath := filepath.Join("/proc", strconv.FormatUint(uint64(pid), 10), "stat")
	data, err := os.ReadFile(statPath)
	if err != nil {
		return nil
	}

	// Parse stat file: find the last closing paren to handle comm with spaces/parens
	lastParen := bytes.LastIndexByte(data, ')')
	if lastParen < 0 {
		return nil
	}

	// Extract comm (between first '(' and last ')')
	firstParen := bytes.IndexByte(data, '(')
	if firstParen < 0 {
		return nil
	}

	comm := string(data[firstParen+1 : lastParen])

	// Parse fields after the last paren
	fields := strings.Fields(string(data[lastParen+1:]))
	if len(fields) < 21 {
		return nil
	}

	stat := &procStat{
		pid:  pid,
		comm: comm,
	}

	if len(fields) > 0 {
		stat.state = fields[0][0]
	}
	if len(fields) > 1 {
		v, _ := strconv.ParseUint(fields[1], 10, 32)
		stat.ppid = uint32(v)
	}
	if len(fields) > 1 {
		v, _ := strconv.ParseUint(fields[1], 10, 32)
		stat.ppid = uint32(v)
	}
	if len(fields) > 10 {
		v, _ := strconv.ParseUint(fields[10], 10, 32)
		stat.numThreads = uint32(v)
	}
	if len(fields) > 19 {
		stat.vsize, _ = strconv.ParseUint(fields[19], 10, 64)
	}
	if len(fields) > 20 {
		stat.rss, _ = strconv.ParseUint(fields[20], 10, 64)
	}

	// utime and stime (fields 11 and 12 in zero-indexed stat fields after comm)
	if len(fields) > 11 {
		stat.utime, _ = strconv.ParseUint(fields[11], 10, 64)
	}
	if len(fields) > 12 {
		stat.stime, _ = strconv.ParseUint(fields[12], 10, 64)
	}

	// starttime (field 19 in zero-indexed, which is fields[18])
	if len(fields) > 18 {
		stat.starttime, _ = strconv.ParseUint(fields[18], 10, 64)
	}

	// Get current time in ticks
	stat.currentTime = getTicksSinceBoot()

	// Read UID and SID from /proc/[pid]/status
	statusPath := filepath.Join("/proc", strconv.FormatUint(uint64(pid), 10), "status")
	statusData, err := os.ReadFile(statusPath)
	if err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(statusData))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Uid:") {
				parts := strings.Fields(line)
				if len(parts) > 1 {
					v, _ := strconv.ParseUint(parts[1], 10, 32)
					stat.uid = uint32(v)
				}
			}
			if strings.HasPrefix(line, "Session:") {
				parts := strings.Fields(line)
				if len(parts) > 1 {
					v, _ := strconv.ParseUint(parts[1], 10, 32)
					stat.sid = uint32(v)
				}
			}
		}
	}

	return stat
}

// readCmdline reads the full command line from /proc/[pid]/cmdline
func readCmdline(pid uint32) string {
	cmdlinePath := filepath.Join("/proc", strconv.FormatUint(uint64(pid), 10), "cmdline")
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return ""
	}

	// cmdline is null-separated; join with spaces
	parts := bytes.Split(data, []byte{0})
	var cmd []string
	for _, part := range parts {
		if len(part) > 0 {
			cmd = append(cmd, string(part))
		}
	}
	return strings.Join(cmd, " ")
}

// readRoot reads the /proc/[pid]/root symlink
func readRoot(pid uint32) string {
	rootPath := filepath.Join("/proc", strconv.FormatUint(uint64(pid), 10), "root")
	target, err := os.Readlink(rootPath)
	if err != nil {
		return ""
	}
	return target
}

// lookupUID maps UID to username with caching
func lookupUID(uid uint32) string {
	uidMutex.Lock()
	if name, ok := uidCache[uid]; ok {
		uidMutex.Unlock()
		return name
	}
	uidMutex.Unlock()

	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	name := ""
	if err == nil {
		name = u.Username
	} else {
		name = strconv.FormatUint(uint64(uid), 10)
	}

	uidMutex.Lock()
	uidCache[uid] = name
	uidMutex.Unlock()

	return name
}

// parseStatus converts stat state to readable status
func parseStatus(state byte) string {
	switch state {
	case 'R':
		return "Running"
	case 'S':
		return "Sleeping"
	case 'D':
		return "Sleeping" // Uninterruptible sleep
	case 'I':
		return "Idle"
	case 'T':
		return "Stopped"
	case 'Z':
		return "Zombie"
	default:
		return "Unknown"
	}
}

// calculateCPUUsage computes per-process CPU percentage
func calculateCPUUsage(m *Monitor, pid uint32, stat *procStat, elapsed float64) float32 {
	totalTime := stat.utime + stat.stime

	prev, hasPrev := m.prevProcTimes[pid]
	prevTotal := prev.User + prev.System

	cpuUsage := float32(0.0)

	if hasPrev && totalTime > prevTotal && elapsed > 0 {
		ticksPerSec := float64(100) // Most Linux systems use 100 Hz
		deltaTicks := float64(totalTime - prevTotal)
		totalTicks := ticksPerSec * elapsed * float64(len(m.prevCPUTimes))
		if totalTicks > 0 {
			cpuUsage = float32((deltaTicks / totalTicks) * 100.0)
		}
	}

	// Store for next refresh
	m.prevProcTimes[pid] = procTimes{
		User:   stat.utime,
		System: stat.stime,
	}

	return cpuUsage
}

// calculateDiskIO computes per-process disk I/O rates
func calculateDiskIO(m *Monitor, pid uint32, elapsed float64) (uint64, uint64) {
	ioPath := filepath.Join("/proc", strconv.FormatUint(uint64(pid), 10), "io")
	data, err := os.ReadFile(ioPath)
	if err != nil {
		// Gracefully handle permission denied or file not found
		return 0, 0
	}

	var readBytes, writeBytes uint64

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		switch parts[0] {
		case "read_bytes:":
			readBytes, _ = strconv.ParseUint(parts[1], 10, 64)
		case "write_bytes:":
			writeBytes, _ = strconv.ParseUint(parts[1], 10, 64)
		}
	}

	// Calculate rates from delta
	diskRead := uint64(0)
	diskWrite := uint64(0)

	prev, hasPrev := m.prevProcDisk[pid]
	if hasPrev && elapsed > 0 {
		if readBytes >= prev.Read {
			delta := readBytes - prev.Read
			diskRead = uint64(float64(delta) / elapsed)
		}
		if writeBytes >= prev.Write {
			delta := writeBytes - prev.Write
			diskWrite = uint64(float64(delta) / elapsed)
		}
	}

	// Store for next refresh
	m.prevProcDisk[pid] = procDiskCounters{
		Read:  readBytes,
		Write: writeBytes,
	}

	return diskRead, diskWrite
}

// getBootTime reads boot time from /proc/stat
func getBootTime() uint64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "btime") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				bootTime, _ := strconv.ParseUint(parts[1], 10, 64)
				return bootTime
			}
		}
	}
	return 0
}

// getTicksSinceBoot calculates current time in ticks since boot
func getTicksSinceBoot() uint64 {
	now := time.Now().Unix()
	bootTime := getBootTime()
	if bootTime > 0 {
		elapsed := uint64(now) - bootTime
		ticksPerSec := uint64(100)
		return elapsed * ticksPerSec
	}
	return 0
}

// getProcessDetail fetches expensive per-process data
func (m *Monitor) getProcessDetail(pid uint32) *ProcessDetail {
	detail := &ProcessDetail{
		PID: pid,
	}

	// Read environment
	environPath := filepath.Join("/proc", strconv.FormatUint(uint64(pid), 10), "environ")
	data, err := os.ReadFile(environPath)
	if err == nil {
		// environ is null-separated
		parts := bytes.Split(data, []byte{0})
		for _, part := range parts {
			if len(part) > 0 {
				detail.Environ = append(detail.Environ, string(part))
			}
		}
	}

	// Read root
	detail.Root = readRoot(pid)

	// Read vsize from stat
	stat := readProcStat(pid)
	if stat != nil {
		detail.VirtualMemory = stat.vsize
	}

	return detail
}

// killProcess sends SIGTERM to a process
func (m *Monitor) killProcess(pid uint32) bool {
	err := unix.Kill(int(pid), syscall.SIGTERM)
	return err == nil
}
