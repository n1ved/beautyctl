package main

import (
    "fmt"
    "os"
    
    "beautyctl/logger"
    "beautyctl/ui"
    
    "github.com/charmbracelet/bubbletea"
)

func main() {
    if err := logger.Init("beautyctl.log"); err != nil {
        fmt.Fprintf(os.Stderr, "Could not initialize logger: %v\n", err)
        os.Exit(1)
    }
    defer logger.Close()
    
    logger.Println("Starting BeautyCTL...")

    m, err := ui.NewModel()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
        os.Exit(1)
    }
    
    p := tea.NewProgram(m, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
        os.Exit(1)
    }
}
