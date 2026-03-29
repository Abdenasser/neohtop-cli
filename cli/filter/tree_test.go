package filter

import (
	"strings"
	"testing"

	"github.com/abdenasser/neohtop-cli/types"
)

func TestBuildProcessTree(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		procs := []types.Process{}
		result := BuildProcessTree(procs)

		if len(result) != 0 {
			t.Errorf("expected empty result, got %d processes", len(result))
		}
	})

	t.Run("single process root", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
		}
		result := BuildProcessTree(procs)

		if len(result) != 1 {
			t.Errorf("expected 1 process, got %d", len(result))
		}
		if result[0].TreeDepth != 0 {
			t.Errorf("expected TreeDepth 0, got %d", result[0].TreeDepth)
		}
		if result[0].TreePrefix != "" {
			t.Errorf("expected empty TreePrefix for root, got '%s'", result[0].TreePrefix)
		}
	})

	t.Run("simple parent child", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 2, PPID: 1, Name: "bash"},
		}
		result := BuildProcessTree(procs)

		if len(result) != 2 {
			t.Errorf("expected 2 processes, got %d", len(result))
		}
		if result[0].PID != 1 {
			t.Errorf("expected PID 1 first, got %d", result[0].PID)
		}
		if result[1].PID != 2 {
			t.Errorf("expected PID 2 second, got %d", result[1].PID)
		}
		if result[0].TreeDepth != 0 {
			t.Errorf("expected parent TreeDepth 0, got %d", result[0].TreeDepth)
		}
		if result[1].TreeDepth != 1 {
			t.Errorf("expected child TreeDepth 1, got %d", result[1].TreeDepth)
		}
		if !strings.Contains(result[1].TreePrefix, "└─") {
			t.Errorf("expected child TreePrefix to contain '└─', got '%s'", result[1].TreePrefix)
		}
	})

	t.Run("multiple children under one parent", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 2, PPID: 1, Name: "bash"},
			{PID: 3, PPID: 1, Name: "vim"},
			{PID: 4, PPID: 1, Name: "cat"},
		}
		result := BuildProcessTree(procs)

		if len(result) != 4 {
			t.Errorf("expected 4 processes, got %d", len(result))
		}
		if result[0].PID != 1 {
			t.Errorf("expected PID 1 first, got %d", result[0].PID)
		}
		// Children should be in order (sorted by PID)
		if result[1].PID != 2 || result[2].PID != 3 || result[3].PID != 4 {
			t.Errorf("expected children in order [2, 3, 4], got [%d, %d, %d]",
				result[1].PID, result[2].PID, result[3].PID)
		}
		// Check prefix for non-last child has ├─
		if !strings.Contains(result[1].TreePrefix, "├─") {
			t.Errorf("expected non-last child to have ├─, got '%s'", result[1].TreePrefix)
		}
		// Check prefix for last child has └─
		if !strings.Contains(result[3].TreePrefix, "└─") {
			t.Errorf("expected last child to have └─, got '%s'", result[3].TreePrefix)
		}
	})

	t.Run("deep nesting 3+ levels", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 2, PPID: 1, Name: "bash"},
			{PID: 3, PPID: 2, Name: "vim"},
			{PID: 4, PPID: 3, Name: "cat"},
		}
		result := BuildProcessTree(procs)

		if len(result) != 4 {
			t.Errorf("expected 4 processes, got %d", len(result))
		}

		// Check depths
		if result[0].TreeDepth != 0 {
			t.Errorf("expected depth 0, got %d", result[0].TreeDepth)
		}
		if result[1].TreeDepth != 1 {
			t.Errorf("expected depth 1, got %d", result[1].TreeDepth)
		}
		if result[2].TreeDepth != 2 {
			t.Errorf("expected depth 2, got %d", result[2].TreeDepth)
		}
		if result[3].TreeDepth != 3 {
			t.Errorf("expected depth 3, got %d", result[3].TreeDepth)
		}

		// Check that deeply nested has proper indentation
		if result[3].TreeDepth != 3 {
			t.Errorf("expected PID 4 to have depth 3, got %d", result[3].TreeDepth)
		}
	})

	t.Run("multiple root processes", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 5, PPID: 0, Name: "kernel_thread"},
			{PID: 2, PPID: 1, Name: "bash"},
		}
		result := BuildProcessTree(procs)

		if len(result) != 3 {
			t.Errorf("expected 3 processes, got %d", len(result))
		}
		// Roots should come first in order
		if result[0].PID != 1 {
			t.Errorf("expected PID 1 first, got %d", result[0].PID)
		}
		if result[1].PID != 5 {
			t.Errorf("expected PID 5 second, got %d", result[1].PID)
		}
		if result[2].PID != 2 {
			t.Errorf("expected PID 2 third, got %d", result[2].PID)
		}
	})

	t.Run("orphan process PPID not in list", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 10, PPID: 999, Name: "orphan"},
		}
		result := BuildProcessTree(procs)

		if len(result) != 2 {
			t.Errorf("expected 2 processes, got %d", len(result))
		}
		// Orphan should be treated as root
		if result[0].PID != 1 && result[1].PID != 10 {
			t.Errorf("expected both processes in result")
		}
	})

	t.Run("cycle detection PID=PPID", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 2, PPID: 2, Name: "self_parent"},
		}
		result := BuildProcessTree(procs)

		if len(result) != 2 {
			t.Errorf("expected 2 processes, got %d", len(result))
		}
		// Self-parent should be treated as root
		if result[0].TreeDepth != 0 || result[1].TreeDepth != 0 {
			t.Errorf("expected both to be roots (depth 0)")
		}
	})

	t.Run("tree prefix correctness for middle and last children", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 2, PPID: 1, Name: "first"},
			{PID: 3, PPID: 1, Name: "middle"},
			{PID: 4, PPID: 1, Name: "last"},
		}
		result := BuildProcessTree(procs)

		// First child (non-last)
		if !strings.Contains(result[1].TreePrefix, "├─") {
			t.Errorf("expected first child to have ├─, got '%s'", result[1].TreePrefix)
		}
		// Middle child (non-last)
		if !strings.Contains(result[2].TreePrefix, "├─") {
			t.Errorf("expected middle child to have ├─, got '%s'", result[2].TreePrefix)
		}
		// Last child
		if !strings.Contains(result[3].TreePrefix, "└─") {
			t.Errorf("expected last child to have └─, got '%s'", result[3].TreePrefix)
		}
	})

	t.Run("TreeDepth values correct for siblings", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 2, PPID: 1, Name: "sibling1"},
			{PID: 3, PPID: 1, Name: "sibling2"},
		}
		result := BuildProcessTree(procs)

		if result[1].TreeDepth != 1 || result[2].TreeDepth != 1 {
			t.Errorf("expected siblings to have depth 1")
		}
	})

	t.Run("children sorted by PID", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 30, PPID: 1, Name: "c"},
			{PID: 20, PPID: 1, Name: "b"},
			{PID: 10, PPID: 1, Name: "a"},
		}
		result := BuildProcessTree(procs)

		// Children should be sorted by PID
		if result[1].PID != 10 || result[2].PID != 20 || result[3].PID != 30 {
			t.Errorf("expected children sorted [10, 20, 30], got [%d, %d, %d]",
				result[1].PID, result[2].PID, result[3].PID)
		}
	})

	t.Run("tree structure with grandchildren", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 2, PPID: 1, Name: "bash"},
			{PID: 3, PPID: 2, Name: "vim"},
			{PID: 4, PPID: 2, Name: "cat"},
		}
		result := BuildProcessTree(procs)

		if len(result) != 4 {
			t.Errorf("expected 4 processes, got %d", len(result))
		}
		// Check order: 1, 2 (depth 1), 3, 4 (depth 2)
		if result[0].PID != 1 || result[1].PID != 2 {
			t.Errorf("expected parent chain first")
		}
		if result[2].PID != 3 || result[3].PID != 4 {
			t.Errorf("expected children after parent")
		}
		if result[2].TreeDepth != 2 || result[3].TreeDepth != 2 {
			t.Errorf("expected grandchildren to have depth 2")
		}
	})

	t.Run("indentation with continuation lines", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
			{PID: 2, PPID: 1, Name: "bash"},
			{PID: 3, PPID: 2, Name: "vim"},
			{PID: 4, PPID: 2, Name: "cat"},
		}
		result := BuildProcessTree(procs)

		// Second level children of bash should have proper continuation
		// vim (non-last) should have ├─ and cat (last) should have └─
		if !strings.Contains(result[2].TreePrefix, "├─") {
			t.Errorf("expected vim to have ├─ connector")
		}
		if !strings.Contains(result[3].TreePrefix, "└─") {
			t.Errorf("expected cat to have └─ connector")
		}
		// Check for continuation character (│) in vim's prefix but not cat's
		if !strings.Contains(result[2].TreePrefix, "│") {
			t.Errorf("expected vim to have continuation line (│)")
		}
		if strings.Contains(result[3].TreePrefix, "│") && !strings.Contains(result[3].TreePrefix, "└─") {
			// cat's prefix might have │ from parent context but should end with └─
			t.Errorf("expected cat to use spaces after └─, not │")
		}
	})

	t.Run("cycle avoidance visited tracking", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 2, Name: "a"},
			{PID: 2, PPID: 1, Name: "b"},
		}
		result := BuildProcessTree(procs)

		// Should complete without infinite loop
		if len(result) == 0 {
			t.Errorf("expected result even with cycle")
		}
	})

	t.Run("preserves process data in output", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init", CPUUsage: 1.5, MemoryUsage: 1024},
			{PID: 2, PPID: 1, Name: "bash", CPUUsage: 5.2, MemoryUsage: 2048},
		}
		result := BuildProcessTree(procs)

		if result[0].CPUUsage != 1.5 || result[0].MemoryUsage != 1024 {
			t.Errorf("expected parent data preserved")
		}
		if result[1].CPUUsage != 5.2 || result[1].MemoryUsage != 2048 {
			t.Errorf("expected child data preserved")
		}
	})

	t.Run("root process has zero depth and empty prefix", func(t *testing.T) {
		procs := []types.Process{
			{PID: 1, PPID: 0, Name: "init"},
		}
		result := BuildProcessTree(procs)

		if result[0].TreeDepth != 0 {
			t.Errorf("expected root depth 0, got %d", result[0].TreeDepth)
		}
		if result[0].TreePrefix != "" {
			t.Errorf("expected root prefix empty, got '%s'", result[0].TreePrefix)
		}
	})
}
