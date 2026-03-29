package main

import (
	"fmt"
	"os"

	"github.com/abdenasser/neohtop-cli/model"
	"github.com/abdenasser/neohtop-cli/monitor"

	tea "charm.land/bubbletea/v2"
)

// version is set at build time via -ldflags
var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("neohtop-cli %s\n", version)
		os.Exit(0)
	}

	// Initialize the native Go monitor (reads OS interfaces directly — no FFI)
	mon := monitor.New()
	defer mon.Destroy()

	// Create and run the Bubble Tea program
	app := model.NewApp(mon)
	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
