# CLAUDE.md

## Project Overview

NeoHtopCLI is a terminal-based process monitor — the CLI companion to [NeoHtop](https://github.com/Abdenasser/NeoHtop). It's built with Go (Bubble Tea v2 + Lip Gloss v2) and an optional Rust FFI backend.

## Quick Reference

```bash
# Build & run (requires CGo on macOS)
make build && ./neohtop-cli

# Dev build with race detector
make dev

# Run tests
make test

# Resolve dependencies
make deps
```

## Architecture

The app follows the **Elm architecture** (Model → Update → View) via Bubble Tea v2.

```
cli/
├── main.go              # Entry point, --version flag
├── model/               # App state + update logic (Bubble Tea Model)
│   ├── app.go           # Central model: state, keybindings, tick loop
│   ├── process.go       # Process list management
│   └── system.go        # System stats polling
├── view/                # All rendering (pure functions, no state mutation)
│   ├── stats_bar.go     # CPU sparklines, memory, disk, network panels
│   ├── toolbar.go       # Button-style keybinding hints (3-tier responsive)
│   ├── process_table.go # Main process table with sort indicators
│   ├── footer.go        # Status bar (hostname, OS, selected PID)
│   ├── help.go          # Help overlay
│   ├── process_details.go
│   ├── kill_confirm.go
│   ├── filter_panel.go
│   ├── column_panel.go
│   ├── theme_panel.go
│   ├── sparkline.go     # Braille dot-matrix charts
│   ├── bar.go           # Block-character progress bars
│   ├── format.go        # Formatting helpers (truncate, bytes, duration)
│   ├── icons.go         # Emoji icons
│   ├── process_icons.go # 140+ Nerd Font process icons
│   └── layout.go        # Layout math
├── monitor/             # Platform-specific data collection
│   ├── monitor.go       # Common interface
│   ├── types.go         # ProcessInfo, SystemStats structs
│   ├── *_darwin.go      # macOS: libproc + mach APIs via CGo
│   ├── *_linux.go       # Linux: /proc filesystem (pure Go)
│   └── *_windows.go     # Windows: syscalls (pure Go)
├── theme/               # 15 built-in color themes
│   ├── theme.go         # Theme interface + registry
│   └── catppuccin.go    # All theme definitions
├── filter/              # Search (regex) and sort logic
├── config/              # Persistent config (~/.config/neohtop-cli/config.json)
├── bridge/              # Rust FFI bridge (optional, unused in pure-Go mode)
└── types/               # Shared type definitions
```

## Key Conventions

- **Charm ecosystem v2** — imports are `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2`, NOT the old `github.com/charmbracelet/` paths
- **lipgloss.Width()** for string measurement — never use `len()` on styled/emoji strings
- **Unicode rendering** — braille dots (U+2800–U+28FF) for sparklines, block chars (▏▎▍▌▋▊▉█) for bars, `…` for truncation
- **Theme colors only** — always use `theme.Current()` colors, never hardcode ANSI codes
- **No state in view/** — view functions are pure renderers that take data and return strings
- **CGo required on macOS** — `CGO_ENABLED=1` for libproc/mach; Linux/Windows can be `CGO_ENABLED=0`

## Build Targets

| Target | Platform | CGo | Notes |
|---|---|---|---|
| `make build` | Native | Yes (macOS) | Default build |
| `make build-linux-amd64` | Linux x86_64 | No | Pure Go |
| `make build-linux-arm64` | Linux ARM64 | No | Pure Go |
| `make build-macos-arm64` | macOS ARM | Yes | Apple Silicon |
| `make build-macos-amd64` | macOS Intel | Yes | Cross-compile on macOS |

## Adding Things

**New theme:** Add to `cli/theme/catppuccin.go`, register in `ThemeNames` slice in `theme.go`.

**New view component:** Create file in `cli/view/`, accept theme + data as params, return string. Wire into `model/app.go` View().

**New keybinding:** Handle in `model/app.go` Update() under `KeyPressMsg`. Add hint to `view/toolbar.go` and `view/help.go`.

**New monitor metric:** Add field to `monitor/types.go`, implement per-platform in `*_darwin.go`, `*_linux.go`, `*_windows.go`.

## Release

Push a git tag to trigger the GitHub Actions release workflow:

```bash
git tag -a v0.1.0 -m "Initial release"
git push --tags
```

This builds binaries for macOS (arm64 + amd64), Linux (amd64 + arm64), and Windows (amd64), then creates a GitHub Release with checksums.
