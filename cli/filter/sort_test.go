package filter

import (
	"testing"

	"github.com/abdenasser/neohtop-cli/types"
)

func TestSortProcesses(t *testing.T) {
	t.Run("sort by PID ascending", func(t *testing.T) {
		procs := []types.Process{
			{PID: 300, Name: "c"},
			{PID: 100, Name: "a"},
			{PID: 200, Name: "b"},
		}
		cfg := types.SortConfig{Field: types.SortByPID, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].PID != 100 || result[1].PID != 200 || result[2].PID != 300 {
			t.Errorf("expected PIDs [100, 200, 300], got [%d, %d, %d]",
				result[0].PID, result[1].PID, result[2].PID)
		}
	})

	t.Run("sort by PID descending", func(t *testing.T) {
		procs := []types.Process{
			{PID: 100, Name: "a"},
			{PID: 300, Name: "c"},
			{PID: 200, Name: "b"},
		}
		cfg := types.SortConfig{Field: types.SortByPID, Direction: types.SortDesc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].PID != 300 || result[1].PID != 200 || result[2].PID != 100 {
			t.Errorf("expected PIDs [300, 200, 100], got [%d, %d, %d]",
				result[0].PID, result[1].PID, result[2].PID)
		}
	})

	t.Run("sort by Name case insensitive ascending", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "Zebra"},
			{PID: 2, Name: "apple"},
			{PID: 3, Name: "Banana"},
		}
		cfg := types.SortConfig{Field: types.SortByName, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].Name != "apple" || result[1].Name != "Banana" || result[2].Name != "Zebra" {
			t.Errorf("expected names [apple, Banana, Zebra], got [%s, %s, %s]",
				result[0].Name, result[1].Name, result[2].Name)
		}
	})

	t.Run("sort by Name case insensitive descending", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "apple"},
			{PID: 2, Name: "Zebra"},
			{PID: 3, Name: "Banana"},
		}
		cfg := types.SortConfig{Field: types.SortByName, Direction: types.SortDesc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].Name != "Zebra" || result[1].Name != "Banana" || result[2].Name != "apple" {
			t.Errorf("expected names [Zebra, Banana, apple], got [%s, %s, %s]",
				result[0].Name, result[1].Name, result[2].Name)
		}
	})

	t.Run("sort by CPU descending (default)", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", CPUUsage: 20.0},
			{PID: 2, Name: "b", CPUUsage: 80.0},
			{PID: 3, Name: "c", CPUUsage: 50.0},
		}
		cfg := types.SortConfig{Field: types.SortByCPU, Direction: types.SortDesc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].CPUUsage != 80.0 || result[1].CPUUsage != 50.0 || result[2].CPUUsage != 20.0 {
			t.Errorf("expected CPU [80, 50, 20], got [%.1f, %.1f, %.1f]",
				result[0].CPUUsage, result[1].CPUUsage, result[2].CPUUsage)
		}
	})

	t.Run("sort by CPU ascending", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", CPUUsage: 80.0},
			{PID: 2, Name: "b", CPUUsage: 20.0},
			{PID: 3, Name: "c", CPUUsage: 50.0},
		}
		cfg := types.SortConfig{Field: types.SortByCPU, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].CPUUsage != 20.0 || result[1].CPUUsage != 50.0 || result[2].CPUUsage != 80.0 {
			t.Errorf("expected CPU [20, 50, 80], got [%.1f, %.1f, %.1f]",
				result[0].CPUUsage, result[1].CPUUsage, result[2].CPUUsage)
		}
	})

	t.Run("sort by Memory descending", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", MemoryUsage: 100},
			{PID: 2, Name: "b", MemoryUsage: 500},
			{PID: 3, Name: "c", MemoryUsage: 300},
		}
		cfg := types.SortConfig{Field: types.SortByMemory, Direction: types.SortDesc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].MemoryUsage != 500 || result[1].MemoryUsage != 300 || result[2].MemoryUsage != 100 {
			t.Errorf("expected memory [500, 300, 100], got [%d, %d, %d]",
				result[0].MemoryUsage, result[1].MemoryUsage, result[2].MemoryUsage)
		}
	})

	t.Run("sort by Memory ascending", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", MemoryUsage: 500},
			{PID: 2, Name: "b", MemoryUsage: 100},
			{PID: 3, Name: "c", MemoryUsage: 300},
		}
		cfg := types.SortConfig{Field: types.SortByMemory, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].MemoryUsage != 100 || result[1].MemoryUsage != 300 || result[2].MemoryUsage != 500 {
			t.Errorf("expected memory [100, 300, 500], got [%d, %d, %d]",
				result[0].MemoryUsage, result[1].MemoryUsage, result[2].MemoryUsage)
		}
	})

	t.Run("pinned processes always first", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", Command: "/bin/a", CPUUsage: 10.0},
			{PID: 2, Name: "b", Command: "/bin/b", CPUUsage: 100.0},
			{PID: 3, Name: "c", Command: "/bin/c", CPUUsage: 50.0},
		}
		pinned := map[string]bool{"/bin/a": true}
		cfg := types.SortConfig{Field: types.SortByCPU, Direction: types.SortDesc}
		result := SortProcesses(procs, cfg, pinned)

		if result[0].PID != 1 {
			t.Errorf("expected pinned process (PID 1) first, got PID %d", result[0].PID)
		}
		if result[1].PID != 2 {
			t.Errorf("expected PID 2 second, got PID %d", result[1].PID)
		}
		if result[2].PID != 3 {
			t.Errorf("expected PID 3 third, got PID %d", result[2].PID)
		}
	})

	t.Run("multiple pinned processes come first in sort order", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", Command: "/bin/a", CPUUsage: 10.0},
			{PID: 2, Name: "b", Command: "/bin/b", CPUUsage: 100.0},
			{PID: 3, Name: "c", Command: "/bin/c", CPUUsage: 50.0},
		}
		pinned := map[string]bool{"/bin/a": true, "/bin/c": true}
		cfg := types.SortConfig{Field: types.SortByCPU, Direction: types.SortDesc}
		result := SortProcesses(procs, cfg, pinned)

		// Both pinned should be first, sorted by CPU descending
		if result[0].PID != 3 {
			t.Errorf("expected pinned PID 3 (CPU 50) first, got PID %d", result[0].PID)
		}
		if result[1].PID != 1 {
			t.Errorf("expected pinned PID 1 (CPU 10) second, got PID %d", result[1].PID)
		}
		if result[2].PID != 2 {
			t.Errorf("expected unpinned PID 2 last, got PID %d", result[2].PID)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		procs := []types.Process{}
		cfg := types.SortConfig{Field: types.SortByPID, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		if len(result) != 0 {
			t.Errorf("expected empty result, got %d processes", len(result))
		}
	})

	t.Run("single element", func(t *testing.T) {
		procs := []types.Process{
			{PID: 42, Name: "single"},
		}
		cfg := types.SortConfig{Field: types.SortByPID, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		if len(result) != 1 || result[0].PID != 42 {
			t.Errorf("expected single process with PID 42, got %+v", result)
		}
	})

	t.Run("sort by Threads nil-safe ascending", func(t *testing.T) {
		threads1 := uint32(10)
		threads3 := uint32(30)
		procs := []types.Process{
			{PID: 1, Name: "a", Threads: &threads1},
			{PID: 2, Name: "b", Threads: nil},
			{PID: 3, Name: "c", Threads: &threads3},
		}
		cfg := types.SortConfig{Field: types.SortByThreads, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		// nil should be treated as 0
		if result[0].PID != 2 {
			t.Errorf("expected nil threads (PID 2) first, got PID %d", result[0].PID)
		}
		if result[1].PID != 1 {
			t.Errorf("expected PID 1 (10 threads) second, got PID %d", result[1].PID)
		}
		if result[2].PID != 3 {
			t.Errorf("expected PID 3 (30 threads) third, got PID %d", result[2].PID)
		}
	})

	t.Run("sort by Threads nil-safe descending", func(t *testing.T) {
		threads1 := uint32(10)
		threads3 := uint32(30)
		procs := []types.Process{
			{PID: 1, Name: "a", Threads: &threads1},
			{PID: 2, Name: "b", Threads: nil},
			{PID: 3, Name: "c", Threads: &threads3},
		}
		cfg := types.SortConfig{Field: types.SortByThreads, Direction: types.SortDesc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].PID != 3 {
			t.Errorf("expected PID 3 (30 threads) first, got PID %d", result[0].PID)
		}
		if result[1].PID != 1 {
			t.Errorf("expected PID 1 (10 threads) second, got PID %d", result[1].PID)
		}
		if result[2].PID != 2 {
			t.Errorf("expected nil threads (PID 2) last, got PID %d", result[2].PID)
		}
	})

	t.Run("sort by Disk sum of read+write descending", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", DiskRead: 100, DiskWrite: 50},   // 150 total
			{PID: 2, Name: "b", DiskRead: 200, DiskWrite: 300},  // 500 total
			{PID: 3, Name: "c", DiskRead: 75, DiskWrite: 75},    // 150 total
		}
		cfg := types.SortConfig{Field: types.SortByDisk, Direction: types.SortDesc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].PID != 2 {
			t.Errorf("expected PID 2 (500 total) first, got PID %d", result[0].PID)
		}
		// PIDs 1 and 3 both have 150, order depends on stability
		if result[1].PID != 1 && result[1].PID != 3 {
			t.Errorf("expected PID 1 or 3 second, got PID %d", result[1].PID)
		}
	})

	t.Run("sort by Disk ascending", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", DiskRead: 100, DiskWrite: 50},   // 150 total
			{PID: 2, Name: "b", DiskRead: 200, DiskWrite: 300},  // 500 total
			{PID: 3, Name: "c", DiskRead: 0, DiskWrite: 0},      // 0 total
		}
		cfg := types.SortConfig{Field: types.SortByDisk, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].PID != 3 {
			t.Errorf("expected PID 3 (0 total) first, got PID %d", result[0].PID)
		}
		if result[1].PID != 1 {
			t.Errorf("expected PID 1 (150 total) second, got PID %d", result[1].PID)
		}
		if result[2].PID != 2 {
			t.Errorf("expected PID 2 (500 total) third, got PID %d", result[2].PID)
		}
	})

	t.Run("sort by Status case sensitive", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", Status: "Sleeping"},
			{PID: 2, Name: "b", Status: "Running"},
			{PID: 3, Name: "c", Status: "Stopped"},
		}
		cfg := types.SortConfig{Field: types.SortByStatus, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].Status != "Running" {
			t.Errorf("expected 'Running' first, got '%s'", result[0].Status)
		}
	})

	t.Run("stable sort maintains order for equal elements", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", CPUUsage: 50.0},
			{PID: 2, Name: "b", CPUUsage: 50.0},
			{PID: 3, Name: "c", CPUUsage: 50.0},
		}
		cfg := types.SortConfig{Field: types.SortByCPU, Direction: types.SortDesc}
		result := SortProcesses(procs, cfg, nil)

		// All have same CPU, should maintain original order (stable)
		if result[0].PID != 1 || result[1].PID != 2 || result[2].PID != 3 {
			t.Errorf("expected stable order [1, 2, 3], got [%d, %d, %d]",
				result[0].PID, result[1].PID, result[2].PID)
		}
	})

	t.Run("sort by RunTime descending", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", RunTime: 100},
			{PID: 2, Name: "b", RunTime: 500},
			{PID: 3, Name: "c", RunTime: 300},
		}
		cfg := types.SortConfig{Field: types.SortByRunTime, Direction: types.SortDesc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].RunTime != 500 || result[1].RunTime != 300 || result[2].RunTime != 100 {
			t.Errorf("expected RunTime [500, 300, 100], got [%d, %d, %d]",
				result[0].RunTime, result[1].RunTime, result[2].RunTime)
		}
	})

	t.Run("sort by User case insensitive", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", User: "ROOT"},
			{PID: 2, Name: "b", User: "admin"},
			{PID: 3, Name: "c", User: "guest"},
		}
		cfg := types.SortConfig{Field: types.SortByUser, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].User != "admin" || result[1].User != "guest" || result[2].User != "ROOT" {
			t.Errorf("expected user order [admin, guest, ROOT], got [%s, %s, %s]",
				result[0].User, result[1].User, result[2].User)
		}
	})

	t.Run("sort by Command case insensitive", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "a", Command: "/BIN/bash"},
			{PID: 2, Name: "b", Command: "/bin/vim"},
			{PID: 3, Name: "c", Command: "/bin/cat"},
		}
		cfg := types.SortConfig{Field: types.SortByCommand, Direction: types.SortAsc}
		result := SortProcesses(procs, cfg, nil)

		if result[0].Command != "/bin/cat" {
			t.Errorf("expected '/bin/cat' first, got '%s'", result[0].Command)
		}
	})
}
