# VHS Tape — generates demo.gif for NeoHtop CLI README
# Install: brew install vhs (or go install github.com/charmbracelet/vhs@latest)
# Run:     vhs demo.tape

Output assets/demo.gif

Require neohtop-cli

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "Catppuccin Mocha"
Set Padding 20
Set Framerate 30
Set TypingSpeed 60ms
Set WindowBar Colorful
Set WindowBarSize 40
Set CursorBlink false

# Launch the app
Type "neohtop-cli"
Enter
Sleep 3s

# Let the UI settle and show real-time monitoring
Sleep 2s

# Navigate through processes
Down 3
Sleep 500ms
Down 2
Sleep 500ms
Down 2
Sleep 1s

# Open search
Type "s"
Sleep 500ms
Type "chrome"
Sleep 1.5s
Escape
Sleep 1s

# Toggle tree view
Type "T"
Sleep 2.5s

# Toggle tree view off
Type "T"
Sleep 1s

# Open theme selector
Type "t"
Sleep 1s
Down 2
Sleep 500ms
Down 1
Sleep 500ms
Enter
Sleep 2s

# Open theme selector again to switch to another theme
Type "t"
Sleep 500ms
Down 4
Sleep 500ms
Enter
Sleep 2s

# Show process details
Type "i"
Sleep 2s
Escape
Sleep 1s

# Show help overlay
Type "?"
Sleep 2.5s
Escape
Sleep 1s

# Final pause on the main view
Sleep 2s

# Quit
Type "q"
