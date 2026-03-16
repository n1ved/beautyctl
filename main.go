package main

import (
	"flag"
    "fmt"
    "os"
    "strings"
    
    "beautyctl/logger"
    "beautyctl/ui"
    
    "github.com/charmbracelet/bubbletea"
)

func main() {
    coverMode := flag.String("cover", "chafa", "Cover art render mode: chafa | jp2a | none")
    flag.Parse()
    
    mode := strings.ToLower(*coverMode)
    if mode != "chafa" && mode != "jp2a" && mode != "none" {
        fmt.Fprintf(os.Stderr, "Invalid cover mode: %s. Use: chafa, jp2a, or none.\n", mode)
        os.Exit(1)
    }

    if err := logger.Init("beautyctl.log"); err != nil {
        fmt.Fprintf(os.Stderr, "Could not initialize logger: %v\n", err)
        os.Exit(1)
    }
    defer logger.Close()
    
    logger.Println("Starting BeautyCTL...")

    m, err := ui.NewModel(mode)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
        os.Exit(1)
    }
    
    p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
        os.Exit(1)
    }
}
