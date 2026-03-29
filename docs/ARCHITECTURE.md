# ADR-001: NeoHtopCLI — Architecture Plan

**Status:** Superseded — the project shipped as **pure Go** (no Rust FFI).
**Date:** 2026-03-26
**Deciders:** Abdenasser

> **Note:** This document captures the original design which proposed a hybrid Rust + Go
> architecture. During implementation, the decision was made to use pure Go with native
> OS APIs instead, which simplified the build pipeline and eliminated the CGo/FFI complexity
> on Linux and Windows. The Elm-style TUI architecture (Bubble Tea v2) and feature scope
> described here remain accurate.

---

## Context

NeoHtop is a desktop system monitor built with Tauri (Rust backend + Svelte frontend). We want to create **NeoHtopCLI** — a terminal-based equivalent that achieves full feature parity with NeoHtop, packaged as a single CLI binary. The CLI will use **Go** for the TUI frontend (Bubble Tea + Lip Gloss) and **Rust** for the system monitoring backend, integrated via **CGo/FFI**.

### Forces at Play

- NeoHtop's Rust backend already has well-tested, cross-platform system monitoring logic
- Reusing Rust avoids reimplementing process collection, system stats, and platform-specific quirks
- Go's Bubble Tea ecosystem provides excellent TUI capabilities with modern DX
- The final deliverable must be a single self-contained binary
- Full feature parity with NeoHtop is required

---

## Decision

Build NeoHtopCLI as a **hybrid Rust + Go project** where Rust compiles to a C-compatible static library (`libneohtop_core`) and Go consumes it via CGo/FFI, producing a single binary.

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    NeoHtopCLI Binary                     │
│                                                          │
│  ┌─────────────────────────────────┐                     │
│  │        Go TUI Frontend          │                     │
│  │  (Bubble Tea + Lip Gloss)       │                     │
│  │                                  │                     │
│  │  ┌───────────┐ ┌──────────────┐ │                     │
│  │  │ Process   │ │ Stats        │ │                     │
│  │  │ Table View│ │ Dashboard    │ │                     │
│  │  └───────────┘ └──────────────┘ │                     │
│  │  ┌───────────┐ ┌──────────────┐ │                     │
│  │  │ Search &  │ │ Process      │ │                     │
│  │  │ Filters   │ │ Details View │ │                     │
│  │  └───────────┘ └──────────────┘ │                     │
│  │  ┌───────────┐ ┌──────────────┐ │                     │
│  │  │ Toolbar & │ │ Theme        │ │                     │
│  │  │ Controls  │ │ Engine       │ │                     │
│  │  └───────────┘ └──────────────┘ │                     │
│  └──────────────┬──────────────────┘                     │
│                 │ CGo/FFI Boundary                        │
│  ┌──────────────▼──────────────────┐                     │
│  │     Rust Core Library            │                     │
│  │     (libneohtop_core.a)          │                     │
│  │                                  │                     │
│  │  ┌───────────┐ ┌──────────────┐ │                     │
│  │  │ Process   │ │ System       │ │                     │
│  │  │ Monitor   │ │ Monitor      │ │                     │
│  │  └───────────┘ └──────────────┘ │                     │
│  │  ┌───────────┐ ┌──────────────┐ │                     │
│  │  │ FFI       │ │ Types &      │ │                     │
│  │  │ Interface │ │ Serialization│ │                     │
│  │  └───────────┘ └──────────────┘ │                     │
│  └──────────────────────────────────┘                     │
└─────────────────────────────────────────────────────────┘
```

### Data Flow

```
1. Go main() starts Bubble Tea program
2. On tick (every 1–1.5s):
   │
   ├─► Go calls C FFI function: neohtop_get_processes()
   │   │
   │   └─► Rust collects process data + system stats
   │       └─► Returns JSON string via FFI
   │
   ├─► Go unmarshals JSON into Go structs
   │
   ├─► Go applies filters, sorting, pagination (in Go)
   │
   └─► Bubble Tea renders updated view

3. On kill request:
   │
   └─► Go calls: neohtop_kill_process(pid)
       └─► Rust kills process, returns success/failure
```

---

## Project Structure

```
NeoHtopCLI/
├── Makefile                     # Build orchestration
├── README.md
│
├── core/                        # Rust static library
│   ├── Cargo.toml
│   └── src/
│       ├── lib.rs               # FFI exports + C interface
│       ├── ffi.rs               # C-compatible function wrappers
│       ├── monitoring/
│       │   ├── mod.rs
│       │   ├── types.rs         # ProcessInfo, SystemStats (from NeoHtop)
│       │   ├── process_monitor.rs  # Process collection (from NeoHtop)
│       │   └── system_monitor.rs   # System stats (from NeoHtop)
│       └── state.rs             # AppState (simplified, no Tauri)
│
├── cli/                         # Go TUI application
│   ├── go.mod
│   ├── go.sum
│   ├── main.go                  # Entry point
│   ├── bridge/
│   │   ├── bridge.go            # CGo bindings to libneohtop_core
│   │   └── bridge.h             # C header (auto-generated or manual)
│   ├── model/
│   │   ├── app.go               # Main Bubble Tea model
│   │   ├── process.go           # Process data types
│   │   └── system.go            # System stats types
│   ├── view/
│   │   ├── layout.go            # Overall layout manager
│   │   ├── process_table.go     # Process table component
│   │   ├── stats_bar.go         # CPU/Memory/Disk/Network panels
│   │   ├── toolbar.go           # Search box + filter controls
│   │   ├── process_details.go   # Process details overlay
│   │   ├── help.go              # Help/keybinding overlay
│   │   └── kill_confirm.go      # Kill confirmation dialog
│   ├── filter/
│   │   ├── filter.go            # Process filtering logic
│   │   └── sort.go              # Process sorting logic
│   ├── theme/
│   │   ├── theme.go             # Theme definitions
│   │   └── catppuccin.go        # Catppuccin Mocha/Latte palettes
│   └── config/
│       ├── config.go            # User settings (columns, refresh rate)
│       └── persistence.go       # Config file read/write
│
└── scripts/
    ├── build.sh                 # Cross-platform build script
    └── generate-header.sh       # Generate C header from Rust
```

---

## Component Design — Rust Core (`core/`)

### What We Reuse from NeoHtop

The following modules port directly from `NeoHtop/src-tauri/src/`, removing all Tauri-specific code:

| NeoHtop Source | NeoHtopCLI Core | Changes |
|---|---|---|
| `monitoring/types.rs` | `monitoring/types.rs` | Remove `tauri::command` attrs, keep serde |
| `monitoring/process_monitor.rs` | `monitoring/process_monitor.rs` | Direct port, remove Tauri state deps |
| `monitoring/system_monitor.rs` | `monitoring/system_monitor.rs` | Direct port |
| `state.rs` | `state.rs` | Simplify — no Tauri AppState, use plain Mutex |
| `commands.rs` | `ffi.rs` | Replace Tauri commands with `#[no_mangle] extern "C"` functions |

### FFI Interface

The Rust library exposes a minimal C-compatible API:

```rust
// core/src/ffi.rs

/// Initialize the monitoring system. Call once at startup.
/// Returns an opaque handle.
#[no_mangle]
pub extern "C" fn neohtop_init() -> *mut c_void

/// Get processes and system stats as a JSON string.
/// Caller must free the returned string with neohtop_free_string().
#[no_mangle]
pub extern "C" fn neohtop_get_processes(handle: *mut c_void) -> *mut c_char

/// Kill a process by PID. Returns 1 on success, 0 on failure.
#[no_mangle]
pub extern "C" fn neohtop_kill_process(handle: *mut c_void, pid: u32) -> i32

/// Free a string returned by the library.
#[no_mangle]
pub extern "C" fn neohtop_free_string(s: *mut c_char)

/// Clean up and release the handle.
#[no_mangle]
pub extern "C" fn neohtop_destroy(handle: *mut c_void)
```

### Cargo.toml Key Settings

```toml
[lib]
name = "neohtop_core"
crate-type = ["staticlib"]   # Produces .a / .lib for CGo linking

[dependencies]
sysinfo = "0.35"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
libc = "0.2"
```

---

## Component Design — Go CLI (`cli/`)

### CGo Bridge (`bridge/`)

```go
// bridge/bridge.go
package bridge

/*
#cgo LDFLAGS: -L${SRCDIR}/../../core/target/release -lneohtop_core -ldl -lm -lpthread
#include "bridge.h"
*/
import "C"

type Handle struct{ ptr unsafe.Pointer }

func Init() *Handle { ... }
func (h *Handle) GetProcesses() ([]Process, SystemStats, error) { ... }
func (h *Handle) KillProcess(pid uint32) bool { ... }
func (h *Handle) Destroy() { ... }
```

### Main Bubble Tea Model (`model/app.go`)

The app model manages the overall state, equivalent to NeoHtop's `+page.svelte`:

```
AppModel
├── State
│   ├── processes []Process
│   ├── systemStats SystemStats
│   ├── searchTerm string
│   ├── currentPage int
│   ├── pinnedProcesses map[string]bool
│   ├── selectedProcess *Process
│   ├── sortConfig SortConfig
│   ├── filterConfig FilterConfig
│   ├── isFrozen bool
│   └── activeOverlay OverlayType
│
├── Messages (Bubble Tea Msg types)
│   ├── TickMsg          → periodic refresh
│   ├── ProcessDataMsg   → new data from Rust
│   ├── KillResultMsg    → kill confirmation
│   ├── FilterChangeMsg  → filter/search updates
│   └── KeyMsg           → keyboard input
│
└── Commands (Bubble Tea Cmd types)
    ├── fetchProcesses   → calls bridge.GetProcesses()
    ├── killProcess      → calls bridge.KillProcess()
    └── tick             → tea.Tick(refreshInterval)
```

### View Components Mapping (Svelte → Bubble Tea)

| NeoHtop Svelte Component | NeoHtopCLI Go Component | Implementation |
|---|---|---|
| `StatsBar.svelte` | `view/stats_bar.go` | Lip Gloss styled panels: CPU bars, memory bar, disk, network, system info |
| `CpuPanel.svelte` | Part of `stats_bar.go` | Per-core horizontal bars with percentage labels |
| `MemoryPanel.svelte` | Part of `stats_bar.go` | Stacked bar showing used/cached/free |
| `StoragePanel.svelte` | Part of `stats_bar.go` | Usage bar with total/used/free |
| `NetworkPanel.svelte` | Part of `stats_bar.go` | RX/TX formatted values |
| `SystemPanel.svelte` | Part of `stats_bar.go` | Uptime + load averages |
| `ProcessTable.svelte` | `view/process_table.go` | Bubble Tea table with sortable columns, highlighted rows |
| `ProcessRow.svelte` | Part of `process_table.go` | Row rendering with conditional coloring (high CPU/RAM) |
| `SearchBox.svelte` | `view/toolbar.go` | Text input component with help overlay |
| `FilterToggle.svelte` | `view/toolbar.go` | Keybinding-driven filter panel |
| `ProcessDetailsModal.svelte` | `view/process_details.go` | Full-screen overlay with process info, children, env vars |
| `KillProcessModal.svelte` | `view/kill_confirm.go` | Confirmation dialog overlay |
| `ToolBar.svelte` | `view/toolbar.go` | Status line with keybinding hints |

### Key Bindings

```
General:
  q / Ctrl+C    Quit
  ?             Help overlay
  f             Toggle freeze/pause updates
  /             Focus search box
  Esc           Close overlay / clear search

Navigation:
  ↑/↓ / j/k    Move selection in process table
  PgUp/PgDn    Page navigation
  Home/End      Jump to first/last process

Process Actions:
  Enter         Show process details
  x / Del       Kill selected process (with confirmation)
  p             Pin/unpin selected process

Sorting:
  1-9           Sort by column N
  s             Cycle sort field
  r             Reverse sort direction

Filtering:
  F             Open filter panel
  Tab           Cycle between filter fields (in filter panel)

Display:
  c             Toggle column visibility panel
  t             Cycle theme
  +/-           Adjust refresh rate
```

### Filtering & Sorting (Go port of `utils/index.ts`)

The filter and sort logic translates from TypeScript to Go:

```
filter/filter.go:
  - FilterProcesses(processes, searchTerm, filterConfig) → filtered
  - Supports: comma-separated multi-term, regex, PID search
  - Numeric filters: cpu>50, ram>100, runtime>60, status=running
  - Operators: >, <, =, >=, <=

filter/sort.go:
  - SortProcesses(processes, sortConfig, pinnedSet) → sorted
  - Pinned processes stay at top
  - Smart disk I/O sorting (read vs write dominance)
  - Stable sort for ties
```

### Theme System (`theme/`)

Port of NeoHtop's Catppuccin themes to Lip Gloss styles:

```go
type Theme struct {
    Name       string
    Base       lipgloss.Color  // Background
    Text       lipgloss.Color  // Primary text
    Subtext    lipgloss.Color  // Secondary text
    Surface    lipgloss.Color  // Panel backgrounds
    Overlay    lipgloss.Color  // Modal backgrounds
    Blue       lipgloss.Color  // Accent: headers, selections
    Green      lipgloss.Color  // Accent: low CPU/RAM
    Yellow     lipgloss.Color  // Accent: medium CPU/RAM
    Red        lipgloss.Color  // Accent: high CPU/RAM, errors
    Lavender   lipgloss.Color  // Accent: pinned processes
}
```

Two built-in themes: **Catppuccin Mocha** (dark) and **Catppuccin Latte** (light).

### Config Persistence (`config/`)

Settings saved to `~/.config/neohtop-cli/config.json`:

```json
{
  "columns": ["pid", "name", "cpu", "memory", "status", "user", "command"],
  "items_per_page": 50,
  "refresh_rate_ms": 1500,
  "theme": "mocha",
  "status_filters": []
}
```

---

## Build System

### Makefile

```makefile
.PHONY: build clean

# Step 1: Build Rust static library
core:
	cd core && cargo build --release

# Step 2: Build Go binary (links against Rust lib)
build: core
	cd cli && CGO_ENABLED=1 go build -o ../neohtop-cli .

# Cross-compilation targets
build-linux: core
	cd cli && GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o ../neohtop-cli .

build-macos: core
	cd cli && GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -o ../neohtop-cli .

build-windows: core
	cd cli && GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -o ../neohtop-cli.exe .

clean:
	cd core && cargo clean
	rm -f neohtop-cli neohtop-cli.exe
```

### Build Requirements

- **Rust** toolchain (rustup + cargo)
- **Go** 1.21+ with CGo enabled
- **C compiler** (gcc/clang) — required by CGo for linking
- Platform SDK headers (standard on macOS/Linux, MinGW on Windows)

---

## Trade-off Analysis

### Rust FFI vs Pure Go

| Dimension | Rust FFI (Chosen) | Pure Go (gopsutil) |
|---|---|---|
| Code reuse | High — direct port from NeoHtop | None — rewrite everything |
| Build complexity | Medium — requires Rust + Go + C toolchain | Low — just Go |
| Cross-compilation | Harder — need Rust target + C cross-compiler | Easy — Go cross-compiles natively |
| Performance | Excellent — Rust system calls | Good — gopsutil is well-optimized |
| Maintenance | Two codebases share core logic | Diverges from NeoHtop over time |
| Binary size | ~10-15 MB (Rust static lib + Go) | ~8-10 MB (Go only) |

**Rationale:** Rust FFI wins because it keeps a single source of truth for system monitoring logic. When NeoHtop's backend improves, the CLI benefits automatically.

### JSON over FFI vs Protobuf/Flatbuffers

**Chosen: JSON.** The data volume is small (hundreds of processes every 1-1.5s), and JSON avoids adding another dependency. If profiling shows serialization as a bottleneck, we can switch to a binary format later.

---

## Consequences

**What becomes easier:**
- Consistent behavior between NeoHtop desktop and NeoHtopCLI
- Backend bug fixes apply to both projects
- SSH-friendly system monitoring (use over remote terminals)

**What becomes harder:**
- Build setup requires Rust + Go + C toolchain
- Cross-compilation needs more care than a pure Go project
- Debugging across the FFI boundary can be tricky

**What we'll revisit:**
- JSON serialization performance (switch to binary if needed)
- Whether to extract `core/` into a shared crate used by both NeoHtop and NeoHtopCLI
- Windows build pipeline (MinGW/MSVC linking)

---

## Implementation Phases

### Phase 1: Foundation (Core + Bridge)
1. Set up Rust `core/` crate — extract and adapt NeoHtop backend
2. Implement FFI interface (`ffi.rs`) with init/get/kill/destroy
3. Build Go CGo bridge and verify basic data flow
4. Minimal `main.go` that prints process data to stdout

### Phase 2: TUI Shell
5. Set up Bubble Tea app model with tick-based refresh
6. Implement stats bar (CPU, memory, disk, network, system)
7. Implement process table with basic rendering
8. Add keyboard navigation (up/down/page)

### Phase 3: Interactivity
9. Search box with regex support
10. Filter panel (CPU, RAM, runtime, status)
11. Sort by column (click-equivalent via keybinding)
12. Process details overlay
13. Kill process with confirmation

### Phase 4: Polish
14. Pin/unpin processes
15. Theme system (Catppuccin Mocha + Latte)
16. Column visibility toggle
17. Config persistence (~/.config/neohtop-cli/)
18. Pagination controls
19. Help overlay with all keybindings

### Phase 5: Release
20. Cross-platform build scripts
21. README and usage docs
22. Goreleaser or similar for releases
23. Integration tests
