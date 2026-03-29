package types

import (
	"testing"
)

func TestSortFieldString(t *testing.T) {
	t.Run("SortByPID returns PID", func(t *testing.T) {
		if SortByPID.String() != "PID" {
			t.Errorf("expected 'PID', got '%s'", SortByPID.String())
		}
	})

	t.Run("SortByName returns Name", func(t *testing.T) {
		if SortByName.String() != "Name" {
			t.Errorf("expected 'Name', got '%s'", SortByName.String())
		}
	})

	t.Run("SortByCPU returns CPU%", func(t *testing.T) {
		if SortByCPU.String() != "CPU%" {
			t.Errorf("expected 'CPU%%', got '%s'", SortByCPU.String())
		}
	})

	t.Run("SortByMemory returns Memory", func(t *testing.T) {
		if SortByMemory.String() != "Memory" {
			t.Errorf("expected 'Memory', got '%s'", SortByMemory.String())
		}
	})

	t.Run("SortByStatus returns Status", func(t *testing.T) {
		if SortByStatus.String() != "Status" {
			t.Errorf("expected 'Status', got '%s'", SortByStatus.String())
		}
	})

	t.Run("SortByUser returns User", func(t *testing.T) {
		if SortByUser.String() != "User" {
			t.Errorf("expected 'User', got '%s'", SortByUser.String())
		}
	})

	t.Run("SortByCommand returns Command", func(t *testing.T) {
		if SortByCommand.String() != "Command" {
			t.Errorf("expected 'Command', got '%s'", SortByCommand.String())
		}
	})

	t.Run("SortByRunTime returns Runtime", func(t *testing.T) {
		if SortByRunTime.String() != "Runtime" {
			t.Errorf("expected 'Runtime', got '%s'", SortByRunTime.String())
		}
	})

	t.Run("SortByDisk returns Disk I/O", func(t *testing.T) {
		if SortByDisk.String() != "Disk I/O" {
			t.Errorf("expected 'Disk I/O', got '%s'", SortByDisk.String())
		}
	})

	t.Run("SortByThreads returns Threads", func(t *testing.T) {
		if SortByThreads.String() != "Threads" {
			t.Errorf("expected 'Threads', got '%s'", SortByThreads.String())
		}
	})

	t.Run("unknown SortField returns ?", func(t *testing.T) {
		unknown := SortField(999)
		if unknown.String() != "?" {
			t.Errorf("expected '?', got '%s'", unknown.String())
		}
	})

	t.Run("all defined constants have non-? strings", func(t *testing.T) {
		fields := []SortField{
			SortByPID,
			SortByName,
			SortByCPU,
			SortByMemory,
			SortByStatus,
			SortByUser,
			SortByCommand,
			SortByRunTime,
			SortByDisk,
			SortByThreads,
		}

		for _, field := range fields {
			str := field.String()
			if str == "?" {
				t.Errorf("field %d should have a string representation, got ?", field)
			}
		}
	})
}

func TestSortDirectionConstants(t *testing.T) {
	t.Run("SortAsc is 0", func(t *testing.T) {
		if SortAsc != 0 {
			t.Errorf("expected SortAsc=0, got %d", SortAsc)
		}
	})

	t.Run("SortDesc is 1", func(t *testing.T) {
		if SortDesc != 1 {
			t.Errorf("expected SortDesc=1, got %d", SortDesc)
		}
	})

	t.Run("SortAsc and SortDesc are different", func(t *testing.T) {
		if SortAsc == SortDesc {
			t.Error("expected SortAsc and SortDesc to be different")
		}
	})
}

func TestOverlayTypeConstants(t *testing.T) {
	t.Run("OverlayNone is 0", func(t *testing.T) {
		if OverlayNone != 0 {
			t.Errorf("expected OverlayNone=0, got %d", OverlayNone)
		}
	})

	t.Run("OverlayHelp is 1", func(t *testing.T) {
		if OverlayHelp != 1 {
			t.Errorf("expected OverlayHelp=1, got %d", OverlayHelp)
		}
	})

	t.Run("OverlayProcessDetails is 2", func(t *testing.T) {
		if OverlayProcessDetails != 2 {
			t.Errorf("expected OverlayProcessDetails=2, got %d", OverlayProcessDetails)
		}
	})

	t.Run("OverlayKillConfirm is 3", func(t *testing.T) {
		if OverlayKillConfirm != 3 {
			t.Errorf("expected OverlayKillConfirm=3, got %d", OverlayKillConfirm)
		}
	})

	t.Run("OverlayFilters is 4", func(t *testing.T) {
		if OverlayFilters != 4 {
			t.Errorf("expected OverlayFilters=4, got %d", OverlayFilters)
		}
	})

	t.Run("OverlayColumns is 5", func(t *testing.T) {
		if OverlayColumns != 5 {
			t.Errorf("expected OverlayColumns=5, got %d", OverlayColumns)
		}
	})

	t.Run("OverlayThemes is 6", func(t *testing.T) {
		if OverlayThemes != 6 {
			t.Errorf("expected OverlayThemes=6, got %d", OverlayThemes)
		}
	})

	t.Run("all overlay types are distinct", func(t *testing.T) {
		overlays := map[OverlayType]string{
			OverlayNone:            "None",
			OverlayHelp:            "Help",
			OverlayProcessDetails:  "ProcessDetails",
			OverlayKillConfirm:     "KillConfirm",
			OverlayFilters:         "Filters",
			OverlayColumns:         "Columns",
			OverlayThemes:          "Themes",
		}

		seen := make(map[OverlayType]bool)
		for overlay := range overlays {
			if seen[overlay] {
				t.Errorf("overlay type %d appears more than once", overlay)
			}
			seen[overlay] = true
		}
	})
}

func TestProcessType(t *testing.T) {
	t.Run("Process struct has required fields", func(t *testing.T) {
		p := Process{
			PID:        123,
			PPID:       1,
			Name:       "test",
			CPUUsage:   5.5,
			MemoryUsage: 1024,
		}

		if p.PID != 123 {
			t.Errorf("expected PID 123, got %d", p.PID)
		}
		if p.PPID != 1 {
			t.Errorf("expected PPID 1, got %d", p.PPID)
		}
		if p.Name != "test" {
			t.Errorf("expected name 'test', got '%s'", p.Name)
		}
		if p.CPUUsage != 5.5 {
			t.Errorf("expected CPU 5.5, got %f", p.CPUUsage)
		}
	})

	t.Run("Process tree fields", func(t *testing.T) {
		p := Process{
			PID:        100,
			TreePrefix: "├─ ",
			TreeDepth:  1,
		}

		if p.TreePrefix != "├─ " {
			t.Errorf("expected TreePrefix '├─ ', got '%s'", p.TreePrefix)
		}
		if p.TreeDepth != 1 {
			t.Errorf("expected TreeDepth 1, got %d", p.TreeDepth)
		}
	})

	t.Run("Process thread field is pointer", func(t *testing.T) {
		threads := uint32(5)
		p := Process{
			PID:     100,
			Threads: &threads,
		}

		if p.Threads == nil {
			t.Fatal("expected non-nil Threads pointer")
		}
		if *p.Threads != 5 {
			t.Errorf("expected 5 threads, got %d", *p.Threads)
		}
	})

	t.Run("Process disk fields", func(t *testing.T) {
		p := Process{
			PID:       100,
			DiskRead:  1000,
			DiskWrite: 2000,
		}

		if p.DiskRead != 1000 {
			t.Errorf("expected DiskRead 1000, got %d", p.DiskRead)
		}
		if p.DiskWrite != 2000 {
			t.Errorf("expected DiskWrite 2000, got %d", p.DiskWrite)
		}
	})
}

func TestSystemStatsType(t *testing.T) {
	t.Run("SystemStats struct has required fields", func(t *testing.T) {
		stats := SystemStats{
			CPUBrand:      "Intel",
			MemoryTotal:   16 * 1024 * 1024 * 1024,
			MemoryUsed:    8 * 1024 * 1024 * 1024,
			DiskTotalBytes: 500 * 1024 * 1024 * 1024,
			Uptime:        3600,
		}

		if stats.CPUBrand != "Intel" {
			t.Errorf("expected 'Intel', got '%s'", stats.CPUBrand)
		}
		if stats.MemoryTotal == 0 {
			t.Error("expected non-zero MemoryTotal")
		}
		if stats.Uptime != 3600 {
			t.Errorf("expected uptime 3600, got %d", stats.Uptime)
		}
	})

	t.Run("SystemStats LoadAvg is array", func(t *testing.T) {
		stats := SystemStats{
			LoadAvg: [3]float64{1.0, 1.5, 2.0},
		}

		if stats.LoadAvg[0] != 1.0 {
			t.Errorf("expected LoadAvg[0]=1.0, got %f", stats.LoadAvg[0])
		}
		if stats.LoadAvg[2] != 2.0 {
			t.Errorf("expected LoadAvg[2]=2.0, got %f", stats.LoadAvg[2])
		}
	})
}

func TestSortConfig(t *testing.T) {
	t.Run("SortConfig can be created", func(t *testing.T) {
		cfg := SortConfig{
			Field:     SortByCPU,
			Direction: SortDesc,
		}

		if cfg.Field != SortByCPU {
			t.Errorf("expected SortByCPU, got %d", cfg.Field)
		}
		if cfg.Direction != SortDesc {
			t.Errorf("expected SortDesc, got %d", cfg.Direction)
		}
	})

	t.Run("SortConfig field can be queried", func(t *testing.T) {
		cfg := SortConfig{
			Field:     SortByMemory,
			Direction: SortAsc,
		}

		fieldStr := cfg.Field.String()
		if fieldStr != "Memory" {
			t.Errorf("expected 'Memory', got '%s'", fieldStr)
		}
	})
}
