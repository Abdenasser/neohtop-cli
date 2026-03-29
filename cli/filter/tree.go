package filter

import (
	"sort"

	"github.com/abdenasser/neohtop-cli/types"
)

// BuildProcessTree arranges a flat process list into a depth-first tree order
// based on PPID→PID relationships. Each process gets a TreePrefix string
// for rendering (e.g. "├─ ", "│  └─ ") and a TreeDepth level.
//
// Processes whose parent is not in the list are treated as roots.
// Children are sorted by PID within each parent.
func BuildProcessTree(procs []types.Process) []types.Process {
	if len(procs) == 0 {
		return procs
	}

	// Build lookup maps
	pidSet := make(map[uint32]bool, len(procs))
	children := make(map[uint32][]int, len(procs)) // PPID → indices into procs

	for i, p := range procs {
		pidSet[p.PID] = true
		_ = i
	}

	for i, p := range procs {
		children[p.PPID] = append(children[p.PPID], i)
	}

	// Sort children by PID for stable ordering
	for ppid := range children {
		sort.Slice(children[ppid], func(a, b int) bool {
			return procs[children[ppid][a]].PID < procs[children[ppid][b]].PID
		})
	}

	// Find roots: processes whose parent is not in the list (or PID 0/1)
	var roots []int
	for i, p := range procs {
		if !pidSet[p.PPID] || p.PPID == 0 || p.PID == p.PPID {
			roots = append(roots, i)
		}
	}
	sort.Slice(roots, func(a, b int) bool {
		return procs[roots[a]].PID < procs[roots[b]].PID
	})

	// DFS to flatten tree
	result := make([]types.Process, 0, len(procs))
	visited := make(map[uint32]bool, len(procs))

	var walk func(idx, depth int, prefix string, isLast bool)
	walk = func(idx, depth int, prefix string, isLast bool) {
		p := procs[idx]
		if visited[p.PID] {
			return // avoid cycles
		}
		visited[p.PID] = true

		// Build the tree connector for this node
		proc := p // copy
		if depth == 0 {
			proc.TreePrefix = ""
		} else {
			if isLast {
				proc.TreePrefix = prefix + "└─ "
			} else {
				proc.TreePrefix = prefix + "├─ "
			}
		}
		proc.TreeDepth = depth
		result = append(result, proc)

		// Recurse into children
		kids := children[p.PID]
		childPrefix := prefix
		if depth > 0 {
			if isLast {
				childPrefix = prefix + "   "
			} else {
				childPrefix = prefix + "│  "
			}
		}

		for i, kidIdx := range kids {
			isLastChild := i == len(kids)-1
			walk(kidIdx, depth+1, childPrefix, isLastChild)
		}
	}

	for i, rootIdx := range roots {
		_ = i
		walk(rootIdx, 0, "", true)
	}

	// Any processes not visited (orphans from cycles) go at the end
	for i, p := range procs {
		if !visited[p.PID] {
			proc := p
			proc.TreePrefix = "? "
			proc.TreeDepth = 0
			result = append(result, proc)
			_ = i
		}
	}

	return result
}
