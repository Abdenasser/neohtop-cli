package filter

import (
	"testing"

	"github.com/abdenasser/neohtop-cli/types"
)

func TestFilterProcesses(t *testing.T) {
	t.Run("no filters returns input unchanged", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "init"},
			{PID: 2, Name: "systemd"},
		}
		result := FilterProcesses(procs, "", NewConfig())
		if len(result) != len(procs) {
			t.Errorf("expected %d processes, got %d", len(procs), len(result))
		}
		if result[0].PID != 1 {
			t.Errorf("expected PID 1, got %d", result[0].PID)
		}
	})

	t.Run("search by name case insensitive", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "init"},
			{PID: 2, Name: "Systemd"},
			{PID: 3, Name: "bash"},
		}
		result := FilterProcesses(procs, "systemd", NewConfig())
		if len(result) != 1 {
			t.Errorf("expected 1 process, got %d", len(result))
		}
		if result[0].PID != 2 {
			t.Errorf("expected PID 2, got %d", result[0].PID)
		}
	})

	t.Run("search by PID string", func(t *testing.T) {
		procs := []types.Process{
			{PID: 100, Name: "process1"},
			{PID: 200, Name: "process2"},
			{PID: 234, Name: "process3"},
		}
		result := FilterProcesses(procs, "23", NewConfig())
		if len(result) != 1 {
			t.Errorf("expected 1 process, got %d", len(result))
		}
		if result[0].PID != 234 {
			t.Errorf("expected PID 234, got %d", result[0].PID)
		}
	})

	t.Run("search by regex pattern", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "chrome"},
			{PID: 2, Name: "chromium"},
			{PID: 3, Name: "bash"},
		}
		result := FilterProcesses(procs, "^chr", NewConfig())
		if len(result) != 2 {
			t.Errorf("expected 2 processes, got %d", len(result))
		}
		if result[0].PID != 1 {
			t.Errorf("expected PID 1, got %d", result[0].PID)
		}
		if result[1].PID != 2 {
			t.Errorf("expected PID 2, got %d", result[1].PID)
		}
	})

	t.Run("comma-separated multi-term search", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "init"},
			{PID: 2, Name: "systemd"},
			{PID: 3, Name: "bash"},
			{PID: 4, Name: "grep"},
		}
		result := FilterProcesses(procs, "bash, grep", NewConfig())
		if len(result) != 2 {
			t.Errorf("expected 2 processes, got %d", len(result))
		}
		// Should match bash or grep
		found := false
		for _, p := range result {
			if p.PID == 3 || p.PID == 4 {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find bash or grep")
		}
	})

	t.Run("CPU filter with operator >", func(t *testing.T) {
		cfg := NewConfig()
		cfg.CPU.Enabled = true
		cfg.CPU.Operator = ">"
		cfg.CPU.Value = 50.0

		procs := []types.Process{
			{PID: 1, Name: "low", CPUUsage: 20.0},
			{PID: 2, Name: "high", CPUUsage: 75.5},
		}
		result := FilterProcesses(procs, "", cfg)
		if len(result) != 1 {
			t.Errorf("expected 1 process, got %d", len(result))
		}
		if result[0].PID != 2 {
			t.Errorf("expected PID 2, got %d", result[0].PID)
		}
	})

	t.Run("CPU filter with operator >=", func(t *testing.T) {
		cfg := NewConfig()
		cfg.CPU.Enabled = true
		cfg.CPU.Operator = ">="
		cfg.CPU.Value = 50.0

		procs := []types.Process{
			{PID: 1, Name: "low", CPUUsage: 49.9},
			{PID: 2, Name: "exact", CPUUsage: 50.0},
			{PID: 3, Name: "high", CPUUsage: 50.1},
		}
		result := FilterProcesses(procs, "", cfg)
		if len(result) != 2 {
			t.Errorf("expected 2 processes, got %d", len(result))
		}
	})

	t.Run("CPU filter with operator <", func(t *testing.T) {
		cfg := NewConfig()
		cfg.CPU.Enabled = true
		cfg.CPU.Operator = "<"
		cfg.CPU.Value = 50.0

		procs := []types.Process{
			{PID: 1, Name: "low", CPUUsage: 30.0},
			{PID: 2, Name: "high", CPUUsage: 75.5},
		}
		result := FilterProcesses(procs, "", cfg)
		if len(result) != 1 {
			t.Errorf("expected 1 process, got %d", len(result))
		}
		if result[0].PID != 1 {
			t.Errorf("expected PID 1, got %d", result[0].PID)
		}
	})

	t.Run("CPU filter with operator <=", func(t *testing.T) {
		cfg := NewConfig()
		cfg.CPU.Enabled = true
		cfg.CPU.Operator = "<="
		cfg.CPU.Value = 50.0

		procs := []types.Process{
			{PID: 1, Name: "low", CPUUsage: 49.9},
			{PID: 2, Name: "exact", CPUUsage: 50.0},
			{PID: 3, Name: "high", CPUUsage: 50.1},
		}
		result := FilterProcesses(procs, "", cfg)
		if len(result) != 2 {
			t.Errorf("expected 2 processes, got %d", len(result))
		}
	})

	t.Run("CPU filter with operator =", func(t *testing.T) {
		cfg := NewConfig()
		cfg.CPU.Enabled = true
		cfg.CPU.Operator = "="
		cfg.CPU.Value = 50.0

		procs := []types.Process{
			{PID: 1, Name: "low", CPUUsage: 49.9},
			{PID: 2, Name: "exact", CPUUsage: 50.0},
			{PID: 3, Name: "high", CPUUsage: 50.1},
		}
		result := FilterProcesses(procs, "", cfg)
		if len(result) != 1 {
			t.Errorf("expected 1 process, got %d", len(result))
		}
		if result[0].PID != 2 {
			t.Errorf("expected PID 2, got %d", result[0].PID)
		}
	})

	t.Run("RAM filter converts bytes to MB", func(t *testing.T) {
		cfg := NewConfig()
		cfg.RAM.Enabled = true
		cfg.RAM.Operator = ">"
		cfg.RAM.Value = 100.0 // 100 MB

		procs := []types.Process{
			{PID: 1, Name: "low", MemoryUsage: 50 * 1024 * 1024},       // 50 MB
			{PID: 2, Name: "high", MemoryUsage: 200 * 1024 * 1024},     // 200 MB
		}
		result := FilterProcesses(procs, "", cfg)
		if len(result) != 1 {
			t.Errorf("expected 1 process, got %d", len(result))
		}
		if result[0].PID != 2 {
			t.Errorf("expected PID 2, got %d", result[0].PID)
		}
	})

	t.Run("Runtime filter converts seconds to minutes", func(t *testing.T) {
		cfg := NewConfig()
		cfg.Runtime.Enabled = true
		cfg.Runtime.Operator = ">"
		cfg.Runtime.Value = 5.0 // 5 minutes

		procs := []types.Process{
			{PID: 1, Name: "short", RunTime: 180},  // 3 minutes
			{PID: 2, Name: "long", RunTime: 600},   // 10 minutes
		}
		result := FilterProcesses(procs, "", cfg)
		if len(result) != 1 {
			t.Errorf("expected 1 process, got %d", len(result))
		}
		if result[0].PID != 2 {
			t.Errorf("expected PID 2, got %d", result[0].PID)
		}
	})

	t.Run("Status filter case insensitive", func(t *testing.T) {
		cfg := NewConfig()
		cfg.Status.Enabled = true
		cfg.Status.Values = []string{"running"}

		procs := []types.Process{
			{PID: 1, Name: "p1", Status: "Running"},
			{PID: 2, Name: "p2", Status: "RUNNING"},
			{PID: 3, Name: "p3", Status: "Sleeping"},
		}
		result := FilterProcesses(procs, "", cfg)
		if len(result) != 2 {
			t.Errorf("expected 2 processes, got %d", len(result))
		}
	})

	t.Run("combined filters search + CPU + status", func(t *testing.T) {
		cfg := NewConfig()
		cfg.CPU.Enabled = true
		cfg.CPU.Operator = ">"
		cfg.CPU.Value = 30.0
		cfg.Status.Enabled = true
		cfg.Status.Values = []string{"running"}

		procs := []types.Process{
			{PID: 1, Name: "bash", CPUUsage: 50.0, Status: "Running"},
			{PID: 2, Name: "bash", CPUUsage: 20.0, Status: "Running"},
			{PID: 3, Name: "init", CPUUsage: 60.0, Status: "Sleeping"},
		}
		result := FilterProcesses(procs, "bash", cfg)
		if len(result) != 1 {
			t.Errorf("expected 1 process, got %d", len(result))
		}
		if result[0].PID != 1 {
			t.Errorf("expected PID 1, got %d", result[0].PID)
		}
	})

	t.Run("empty process list", func(t *testing.T) {
		procs := []types.Process{}
		result := FilterProcesses(procs, "anything", NewConfig())
		if len(result) != 0 {
			t.Errorf("expected 0 processes, got %d", len(result))
		}
	})

	t.Run("search by command name", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, Name: "bash", Command: "/bin/bash"},
			{PID: 2, Name: "vim", Command: "/usr/bin/vim"},
		}
		result := FilterProcesses(procs, "bash", NewConfig())
		if len(result) != 1 {
			t.Errorf("expected 1 process, got %d", len(result))
		}
		if result[0].PID != 1 {
			t.Errorf("expected PID 1, got %d", result[0].PID)
		}
	})
}

func TestCompareValue(t *testing.T) {
	t.Run("compareValue >", func(t *testing.T) {
		if !compareValue(10.0, ">", 5.0) {
			t.Error("expected 10 > 5 to be true")
		}
		if compareValue(5.0, ">", 5.0) {
			t.Error("expected 5 > 5 to be false")
		}
	})

	t.Run("compareValue <", func(t *testing.T) {
		if !compareValue(5.0, "<", 10.0) {
			t.Error("expected 5 < 10 to be true")
		}
		if compareValue(10.0, "<", 5.0) {
			t.Error("expected 10 < 5 to be false")
		}
	})

	t.Run("compareValue =", func(t *testing.T) {
		if !compareValue(5.0, "=", 5.0) {
			t.Error("expected 5 = 5 to be true")
		}
		if compareValue(5.0, "=", 5.1) {
			t.Error("expected 5 = 5.1 to be false")
		}
	})

	t.Run("compareValue >=", func(t *testing.T) {
		if !compareValue(10.0, ">=", 5.0) {
			t.Error("expected 10 >= 5 to be true")
		}
		if !compareValue(5.0, ">=", 5.0) {
			t.Error("expected 5 >= 5 to be true")
		}
		if compareValue(4.0, ">=", 5.0) {
			t.Error("expected 4 >= 5 to be false")
		}
	})

	t.Run("compareValue <=", func(t *testing.T) {
		if !compareValue(5.0, "<=", 10.0) {
			t.Error("expected 5 <= 10 to be true")
		}
		if !compareValue(5.0, "<=", 5.0) {
			t.Error("expected 5 <= 5 to be true")
		}
		if compareValue(6.0, "<=", 5.0) {
			t.Error("expected 6 <= 5 to be false")
		}
	})

	t.Run("compareValue unknown operator defaults to true", func(t *testing.T) {
		if !compareValue(5.0, "unknown", 10.0) {
			t.Error("expected unknown operator to return true")
		}
	})
}
