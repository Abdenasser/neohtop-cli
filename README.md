<div align="center">

# 👻 NeoHtop CLI

**A cross-platform terminal process monitor with btop-style visualizations**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/Abdenasser/neohtop-cli?color=purple)](https://github.com/Abdenasser/neohtop-cli/releases)
[![npm](https://img.shields.io/npm/v/neohtop-cli?color=red)](https://www.npmjs.com/package/neohtop-cli)

The CLI companion to [NeoHtop](https://github.com/Abdenasser/NeoHtop) — built with Go and the [Charm](https://charm.sh) ecosystem.

[Installation](#installation) · [Features](#features) · [Keybindings](#keybindings) · [Themes](#themes) · [Configuration](#configuration)

<br/>

<img src="assets/demo.gif" alt="NeoHtop CLI Demo" width="820" />

<br/>

<img src="assets/neohtop-cli.JPG" alt="NeoHtop CLI Screenshot" width="820" />

</div>

<br/>

## Installation

```bash
npm install -g neohtop-cli
```

Or download a prebuilt binary from the [Releases page](https://github.com/Abdenasser/neohtop-cli/releases):

<details>
<summary><b>macOS</b></summary>

```bash
# Apple Silicon
curl -LO https://github.com/Abdenasser/neohtop-cli/releases/latest/download/neohtop-cli-macos-arm64.tar.gz
tar xzf neohtop-cli-macos-arm64.tar.gz
sudo mv neohtop-cli-macos-arm64 /usr/local/bin/neohtop-cli

# Intel
curl -LO https://github.com/Abdenasser/neohtop-cli/releases/latest/download/neohtop-cli-macos-amd64.tar.gz
tar xzf neohtop-cli-macos-amd64.tar.gz
sudo mv neohtop-cli-macos-amd64 /usr/local/bin/neohtop-cli
```

</details>

<details>
<summary><b>Linux</b></summary>

```bash
# x86_64
curl -LO https://github.com/Abdenasser/neohtop-cli/releases/latest/download/neohtop-cli-linux-amd64.tar.gz
tar xzf neohtop-cli-linux-amd64.tar.gz
sudo mv neohtop-cli-linux-amd64 /usr/local/bin/neohtop-cli

# ARM64
curl -LO https://github.com/Abdenasser/neohtop-cli/releases/latest/download/neohtop-cli-linux-arm64.tar.gz
tar xzf neohtop-cli-linux-arm64.tar.gz
sudo mv neohtop-cli-linux-arm64 /usr/local/bin/neohtop-cli
```

</details>

<details>
<summary><b>Windows</b></summary>

Download [`neohtop-cli-windows-amd64.zip`](https://github.com/Abdenasser/neohtop-cli/releases/latest) and add the extracted folder to your PATH.

</details>

<details>
<summary><b>Build from source</b></summary>

Requires [Go 1.25+](https://go.dev/dl) and a C compiler for CGo on macOS.

```bash
git clone https://github.com/Abdenasser/neohtop-cli.git
cd neohtop-cli
make build
make install  # optional — copies to /usr/local/bin/
```

</details>

<br/>

## Features

<table>
<tr>
<td width="50%">

**Real-time monitoring**
CPU per-core sparklines, memory, disk I/O, and network stats rendered with braille-dot visualizations

</td>
<td width="50%">

**Powerful search**
Regex-powered filtering with live match highlighting — `^chrome`, `name|pid`, `\.log$`

</td>
</tr>
<tr>
<td>

**15 built-in themes**
Catppuccin, Dracula, Tokyo Night, Nord, Gruvbox, Synthwave, and more — switch with `t`

</td>
<td>

**Process management**
Inspect details, kill processes, pin favorites to the top, toggle tree view

</td>
</tr>
<tr>
<td>

**JSON output**
`neohtop-cli --json` pipes structured data to `jq` for scripting and dashboards

</td>
<td>

**Single binary, cross-platform**
macOS, Linux, and Windows — no runtime dependencies

</td>
</tr>
</table>

<br/>

## Quick Start

```bash
neohtop-cli          # launch the TUI
neohtop-cli --json   # one-shot JSON snapshot
neohtop-cli --help   # see all options
```

Press `?` inside the TUI to see every keybinding.

<br/>

## Keybindings

| | Key | Action |
|---|---|---|
| **General** | `q` · `Ctrl+C` | Quit |
| | `?` | Help overlay |
| | `s` · `/` | Search (regex) |
| | `Space` | Pause / resume |
| | `Esc` | Close overlay / clear search |
| **Navigate** | `↑` `↓` `j` `k` | Move selection |
| | `PgUp` · `PgDn` | Page scroll |
| | `Home` · `g` / `End` · `G` | Jump to top / bottom |
| **Process** | `i` · `Enter` | Inspect details |
| | `k` · `x` · `Del` | Kill (with confirmation) |
| | `p` | Pin / unpin |
| **Display** | `0`–`9` | Sort by column |
| | `f` | Filter panel |
| | `c` | Column visibility |
| | `T` | Tree view |
| | `t` | Theme selector |
| | `r` | Cycle refresh rate |
| **Mouse** | Click row | Select process |
| | Double-click | Open details |
| | Click header | Sort by column |
| | Scroll wheel | Navigate list |

<br/>

## Search

Press `s` or `/` to search. Supports full regex:

```
chrome          process name contains "chrome"
^sys            names starting with "sys"
\.log$          commands ending in ".log"
node|deno       matches either
1234            PID 1234
```

Matches are highlighted in yellow in the Name and Command columns.

<br/>

## Themes

Press `t` to open the theme selector with live color swatches.

| | | |
|---|---|---|
| **Charm** *(default)* | **Catppuccin Mocha** | **Catppuccin Latte** |
| **Dracula** | **Tokyo Night** | **Gruvbox Dark** |
| **Nord** | **One Dark** | **Rosé Pine** |
| **Synthwave** | **Solarized Dark** | **Monokai Pro** |
| **High Contrast** | **Green Terminal** | **Amber Terminal** |

<br/>

## JSON Output

`--json` outputs a single snapshot of system stats and all processes. Perfect for scripting:

```bash
# top 10 by CPU
neohtop-cli --json | jq '[.processes[] | {name, cpu: .cpu_usage}] | sort_by(.cpu) | reverse[:10]'

# memory usage
neohtop-cli --json | jq '.system | {memory_used, memory_total, pct: (.memory_used/.memory_total*100|round)}'

# watch mode
watch -n2 'neohtop-cli --json | jq ".system.cpu_usage_per_core"'
```

<br/>

## Configuration

Settings persist at `~/.config/neohtop-cli/config.json`:

```json
{
  "columns": ["pid", "name", "cpu", "memory", "status", "user", "command"],
  "refresh_rate_ms": 1000,
  "theme": "charm"
}
```

**Columns:** `pid` `name` `cpu` `memory` `status` `user` `command` `threads` `runtime` `disk`

**Refresh rates** — cycle with `r`: 1s *(default)* → 2s → 3s → 5s → 0.5s

<br/>

## Architecture

Pure Go application using native OS APIs — no FFI, no external dependencies.

```
┌──────────────────────────────────────────┐
│          Go TUI (Bubble Tea v2)          │
│                                          │
│  Stats Bar · Toolbar · Process Table     │
│  Sparklines · Braille Bars · Overlays    │
├──────────────────────────────────────────┤
│          Native Go Monitor               │
│                                          │
│  process_darwin.go   system_darwin.go    │
│  process_linux.go    system_linux.go     │
│  process_windows.go  system_windows.go   │
└──────────────────────────────────────────┘
```

Built with [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) and [Lip Gloss v2](https://github.com/charmbracelet/lipgloss).

<details>
<summary><b>Project structure</b></summary>

```
neohtop-cli/
├── cli/
│   ├── main.go               # entry point + --json + --version
│   ├── model/                 # Bubble Tea model (state + update loop)
│   ├── view/                  # UI components
│   │   ├── stats_bar.go       #   CPU, memory, network, info panels
│   │   ├── process_table.go   #   main data grid
│   │   ├── toolbar.go         #   button bar
│   │   ├── footer.go          #   status bar
│   │   ├── sparkline.go       #   time-series sparklines
│   │   ├── bar.go             #   braille progress bars
│   │   ├── help.go            #   keybinding overlay
│   │   ├── process_details.go #   process info modal
│   │   ├── kill_confirm.go    #   kill confirmation
│   │   ├── filter_panel.go    #   filter configuration
│   │   ├── column_panel.go    #   column visibility
│   │   ├── theme_panel.go     #   theme selector
│   │   └── process_icons.go   #   140+ Nerd Font icons
│   ├── monitor/               # OS-specific system monitoring
│   ├── theme/                 # 15 color themes
│   ├── filter/                # search, filter, sort, tree
│   └── config/                # persistent user settings
├── Makefile
└── CONTRIBUTING.md
```

</details>

<br/>

## NeoHtop CLI vs NeoHtop Desktop

| Feature | Desktop | CLI |
|---|:---:|:---:|
| Process monitoring | ✅ | ✅ |
| CPU per-core stats | ✅ | ✅ |
| Memory / Disk / Network | ✅ | ✅ |
| Regex search | ✅ | ✅ |
| Kill / Pin / Inspect | ✅ | ✅ |
| Mouse support | ✅ | ✅ |
| Themes | 12 | **15** |
| Process tree view | — | ✅ |
| JSON scripting | — | ✅ |
| Runs in terminal | — | ✅ |
| Single binary | — | ✅ |

<br/>

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, and how to add themes, icons, and keybindings.

## License

[MIT](LICENSE)

<br/>

<div align="center">

**[NeoHtop Desktop](https://github.com/Abdenasser/NeoHtop)** · **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** · **[btop](https://github.com/aristocratos/btop)**

Made with 👻 by [Abdenasser](https://github.com/Abdenasser)

</div>
