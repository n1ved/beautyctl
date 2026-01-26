package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#bd93f9") // Dracula Purple
	SecondaryColor = lipgloss.Color("#ff79c6") // Dracula Pink
	SubTitleColor  = lipgloss.Color("#6272a4") // Dracula Comment
    TextColor      = lipgloss.Color("#f8f8f2") // Dracula Foreground
	BorderColor    = lipgloss.Color("#44475a") // Dracula Selection

	// Styles
	AppStyle = lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderColor)

	TitleStyle = lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		MarginBottom(1)

	ArtistStyle = lipgloss.NewStyle().
		Foreground(SecondaryColor).
		Italic(true)

    InfoStyle = lipgloss.NewStyle().
        Foreground(SubTitleColor)

	BarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8be9fd")) // Dracula Cyan

    ActiveBarStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#50fa7b")) // Dracula Green
)
