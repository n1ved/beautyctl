package ui

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"beautyctl/lyrics"
	"beautyctl/player"
	"beautyctl/ui/image"
	"beautyctl/visualizer"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

type tickMsg time.Time
type visualizerMsg []int
type imageMsg string  // contains the render string
type lyricsMsg string // contains lyrics text, or "" if not found

type Model struct {
	playerControl *player.Control
	cavaControl   *visualizer.CavaControl

	metadata       *player.Metadata
	visualizerData []int

	err error

	width  int
	height int

	// Progress bar related
	progress float64

	// Config
	renderer string

	// Cache
	titleRender string

	// Image related
	lastArtURL string
	artRender  string

	// Lyrics
	lyrics      string // current lyrics text; empty = not available
	lyricsKey   string // "artist|title" to detect track change
	lyricsScroll int   // scroll offset (lines)
}

func NewModel(renderer string) (*Model, error) {
	pc := player.NewControl()
	// 200 bars for visualizer to cover wide screens
	cc, err := visualizer.NewCavaControl(200)
	if err != nil {
		return nil, err
	}

	return &Model{
		playerControl:  pc,
		cavaControl:    cc,
		visualizerData: make([]int, 200),
		renderer:       renderer,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		waitForVisualizer(m.cavaControl.Output),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateTitle() // Recalculate title size on resize
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.cavaControl.Stop()
			return m, tea.Quit
		case " ":
			m.playerControl.PlayPause()
		case "n":
			m.playerControl.Next()
		case "p":
			m.playerControl.Previous()
		case "right":
			m.playerControl.SeekForward()
		case "left":
			m.playerControl.SeekBackward()
		case "u", "up":
			if m.lyricsScroll > 0 {
				m.lyricsScroll--
			}
			return m, nil
		case "d", "down":
			m.lyricsScroll++
			return m, nil
		}
		// Force an immediate update after action
		return m, tickCmd()

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.lyricsScroll > 0 {
				m.lyricsScroll--
			}
			return m, nil
		case tea.MouseButtonWheelDown:
			m.lyricsScroll++
			return m, nil
		}

	case tickMsg:
		meta, err := m.playerControl.GetMetadata()

		// Check if art changed
		if meta != nil {
			oldTitle := ""
			if m.metadata != nil {
				oldTitle = m.metadata.Title
			}

			m.metadata = meta
			m.err = err

			if meta.Title != oldTitle {
				m.updateTitle()
			}

			// --- Lyrics: fetch when track changes ---
			lyricsKey := meta.Artist + "|" + meta.Title
			if lyricsKey != m.lyricsKey {
				m.lyricsKey = lyricsKey
				m.lyrics = ""       // clear while loading
				m.lyricsScroll = 0  // reset scroll
				artist := meta.Artist
				title := meta.Title
				return m, tea.Batch(tickCmd(), func() tea.Msg {
					text, err := lyrics.Fetch(artist, title)
					if err != nil {
						return lyricsMsg("") // not found or error → hide
					}
					return lyricsMsg(text)
				})
			}

			if meta.ArtURL != m.lastArtURL {
				m.lastArtURL = meta.ArtURL

				if m.renderer == "none" {
					return m, nil
				}

				renderer := m.renderer
				return m, tea.Batch(tickCmd(), func() tea.Msg {
					// Async load image
					// strip file:// prefix if present
					url := strings.TrimPrefix(meta.ArtURL, "file://")

					targetPath := url
					if strings.HasPrefix(url, "http") {
						// Download to temp file
						resp, err := http.Get(url)
						if err != nil {
							return imageMsg("")
						}
						defer resp.Body.Close()

						tmpFile, err := os.CreateTemp("", "beautyctl-art-*.jpg")
						if err != nil {
							return imageMsg("")
						}

						_, err = io.Copy(tmpFile, resp.Body)
						tmpFile.Close()
						if err != nil {
							return imageMsg("")
						}
						targetPath = tmpFile.Name()
					}

					// Render art
					var art string
					if renderer == "jp2a" {
						art = image.RenderJP2A(targetPath, 34, 15)
					} else {
						// chafa default
						art = image.RenderChafa(targetPath, 34, 15)
					}
					return imageMsg(art)
				})
			}
		}

		return m, tickCmd()

	case visualizerMsg:
		m.visualizerData = msg
		return m, waitForVisualizer(m.cavaControl.Output)

	case imageMsg:
		m.artRender = string(msg)
		return m, nil

	case lyricsMsg:
		m.lyrics = string(msg)
		m.lyricsScroll = 0
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	width := m.width
	height := m.height

	// Check if window is too small first to avoid panic in Layout
	if width < 20 || height < 10 {
		return "Terminal too small"
	}

	if m.err != nil || m.metadata == nil {
		// --- Idle / Error Screen ---

		// Big Title
		titleArt := figure.NewFigure("BeautyCTL", "rectangles", true).String()
		if lipgloss.Width(titleArt) > width-4 {
			titleArt = figure.NewFigure("BeautyCTL", "small", true).String()
		}
		titleArt = TitleStyle.Render(titleArt)

		statusMsg := "Waiting for music player..."
		if m.err != nil {
			statusMsg = fmt.Sprintf("Error: %v", m.err)
		}

		subTitle := lipgloss.NewStyle().
			Foreground(SubTitleColor).
			Render(statusMsg)

		hint := lipgloss.NewStyle().
			Foreground(SubTitleColor).
			Faint(true).
			Render("Start Spotify, VLC, mpv, or any MPRIS player.")

		content := lipgloss.JoinVertical(lipgloss.Center,
			titleArt,
			"",
			subTitle,
			"",
			hint,
		)

		return lipgloss.Place(width, height,
			lipgloss.Center, lipgloss.Center,
			content,
		)
	}

	// --- Cover Art ---
	var coverView string
	showCover := m.renderer != "none"
	var coverWidth int

	if showCover {
		if m.artRender != "" {
			coverView = m.artRender
		} else {
			// Placeholder
			coverView = lipgloss.NewStyle().
				Width(34).
				Height(15).
				Border(lipgloss.DoubleBorder()).
				Align(lipgloss.Center, lipgloss.Center).
				Render("🎵")
		}
	}

	var coverBox string
	if showCover {
		coverBox = lipgloss.NewStyle().
			MarginRight(2).
			Render(coverView)
		coverWidth = lipgloss.Width(coverBox)
	}

	// --- Info Section ---
	statusIcon := "▶"
	if strings.ToLower(m.metadata.Status) != "playing" {
		statusIcon = "⏸"
	}

	// Adjust info max width to account for cover art
	infoMaxWidth := width - coverWidth - 4
	if infoMaxWidth < 20 {
		infoMaxWidth = 20
	}

	// Title is already cached in m.titleRender
	titleArt := m.titleRender

	// Progress Bar
	barWidth := int(float64(infoMaxWidth) * 0.8)
	if barWidth > 60 {
		barWidth = 60
	}
	if barWidth < 10 {
		barWidth = 10
	}

	progress := 0.0
	if m.metadata.Length > 0 {
		progress = float64(m.metadata.Position) / float64(m.metadata.Length)
	}
	if progress > 1.0 {
		progress = 1.0
	}
	if progress < 0.0 {
		progress = 0.0
	}

	filledChars := int(progress * float64(barWidth))
	emptyChars := barWidth - filledChars
	if emptyChars < 0 {
		emptyChars = 0
	}

	progressBar := "[" +
		ActiveBarStyle.Render(strings.Repeat("━", filledChars)) +
		InfoStyle.Render(strings.Repeat("─", emptyChars)) +
		"]"

	info := lipgloss.JoinVertical(lipgloss.Center,
		titleArt,
		ArtistStyle.Render(m.metadata.Artist),
		InfoStyle.Render(m.metadata.Album),
		"",
		fmt.Sprintf("%s  %s / %s", statusIcon, player.FormatDuration(m.metadata.Position), player.FormatDuration(m.metadata.Length)),
		progressBar,
		fmt.Sprintf("Player: %s", m.metadata.PlayerName),
	)

	// Combine Cover + Info
	var topSection string
	if showCover {
		topSection = lipgloss.JoinHorizontal(lipgloss.Center, coverBox, info)
	} else {
		topSection = info
	}

	// --- Help Section ---
	helpStyle := lipgloss.NewStyle().Foreground(SubTitleColor).Faint(true)
	helpLine := " [Space] Play/Pause  [N] Next  [P] Prev  [←/→] Seek  [Q] Quit "
	if m.lyrics != "" {
		helpLine += " [U/D] Scroll Lyrics "
	}
	help := helpStyle.Render(helpLine)

	// Combine Top + Help
	topContent := lipgloss.JoinVertical(lipgloss.Center, topSection, "", help)
	topHeight := lipgloss.Height(topContent)

	// --- Visualizer Section ---
	// Dynamic height calculation
	// Available height = Window Height - Top Content Height - Padding (approx 2 lines)
	visHeight := height - topHeight - 2

	// Clamp height – no upper bound so cava fills all remaining vertical space
	if visHeight < 3 {
		visHeight = 3
	}

	// --- Width split: visualizer left, lyrics right ---
	showLyrics := m.lyrics != ""

	// Lyrics box takes ~38% of total width; visualizer gets the rest.
	lyricsBoxOuter := 0
	if showLyrics {
		lyricsBoxOuter = int(float64(width) * 0.38)
		if lyricsBoxOuter < 20 {
			showLyrics = false
		}
	}

	visOuterWidth := width - 2 // default: full width (border accounts for 2)
	if showLyrics {
		visOuterWidth = width - lyricsBoxOuter - 2
	}

	// Build visualizer bars
	visWidth := visOuterWidth - 2 // inner content width inside the border
	if visWidth < 0 {
		visWidth = 0
	}
	numBars := visWidth
	if numBars > len(m.visualizerData) {
		numBars = len(m.visualizerData)
	}
	dataSubset := m.visualizerData[:numBars]

	var rows []string
	for y := visHeight - 1; y >= 0; y-- {
		var rowBuilder strings.Builder
		for _, val := range dataSubset {
			totalLevels := visHeight * 8
			scaledVal := (val * totalLevels) / 100
			rowBase := y * 8

			var char string
			if scaledVal >= rowBase+8 {
				char = "█"
			} else if scaledVal < rowBase {
				char = " "
			} else {
				remainder := scaledVal - rowBase
				chars := []string{" ", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
				if remainder >= 0 && remainder < len(chars) {
					char = chars[remainder]
				} else {
					char = "█"
				}
			}
			rowBuilder.WriteString(char)
		}
		rows = append(rows, BarStyle.Render(rowBuilder.String()))
	}

	vis := lipgloss.JoinVertical(lipgloss.Left, rows...)
	// Width/Height = content dims; border adds 1 char each side → total = (w+2)×(h+2)
	visBox := lipgloss.NewStyle().
		Width(visOuterWidth - 2).
		Height(visHeight).
		Align(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderColor).
		Render(vis)

	// --- Build bottom row ---
	var bottomRow string
	if showLyrics {
		// Inner dimensions of the lyrics box
		lyricsInnerWidth := lyricsBoxOuter - 4  // 2 border + 2 padding
		lyricsInnerHeight := visHeight           // match visualizer height

		// Wrap and scroll lyrics
		lyricLines := wrapLines(m.lyrics, lyricsInnerWidth)
		maxScroll := len(lyricLines) - lyricsInnerHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		scroll := m.lyricsScroll
		if scroll > maxScroll {
			scroll = maxScroll
		}
		end := scroll + lyricsInnerHeight
		if end > len(lyricLines) {
			end = len(lyricLines)
		}
		visibleLines := lyricLines[scroll:end]

		// Pad to fill the box height
		for len(visibleLines) < lyricsInnerHeight {
			visibleLines = append(visibleLines, "")
		}

		lyricsContent := strings.Join(visibleLines, "\n")
		// Same Height(visHeight) as visBox so both boxes have identical total height
		lyricsBox := LyricsStyle.
			Width(lyricsBoxOuter - 4).
			Height(visHeight).
			Render(lyricsContent)

		bottomRow = lipgloss.JoinHorizontal(lipgloss.Top, visBox, lyricsBox)
	} else {
		bottomRow = visBox
	}

	// Final Layout – stack top content then bottom row, no artificial gap
	return lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.PlaceHorizontal(width, lipgloss.Center, topContent),
		bottomRow,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func waitForVisualizer(ch chan []int) tea.Cmd {
	return func() tea.Msg {
		v, ok := <-ch
		if !ok {
			return nil
		}
		return visualizerMsg(v)
	}
}

func (m *Model) updateTitle() {
	if m.metadata == nil {
		return
	}

	// Check if title is purely ASCII
	isAscii := true
	for i := 0; i < len(m.metadata.Title); i++ {
		if m.metadata.Title[i] > 127 {
			isAscii = false
			break
		}
	}

	// Fallback to normal text if foreign language character is present
	if !isAscii || m.metadata.Title == "" {
		m.titleRender = TitleStyle.Render(m.metadata.Title)
		return
	}

	tArt := figure.NewFigure(m.metadata.Title, "rectangles", true).String()
	headingWidth := lipgloss.Width(tArt)
	// Heuristic: If wider than screen minus safety padding (e.g. 10)
	if headingWidth > m.width-10 {
		tArt = figure.NewFigure(m.metadata.Title, "small", true).String()
		if lipgloss.Width(tArt) > m.width-10 {
			tArt = TitleStyle.Render(m.metadata.Title)
		}
	}
	m.titleRender = TitleStyle.Render(tArt)
}

// wrapLines wraps a block of text to fit within maxWidth columns.
func wrapLines(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		maxWidth = 1
	}
	var result []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, " \r")
		if len(line) == 0 {
			result = append(result, "")
			continue
		}
		for len(line) > maxWidth {
			// Try to break at a space
			breakAt := strings.LastIndex(line[:maxWidth], " ")
			if breakAt <= 0 {
				breakAt = maxWidth
			}
			result = append(result, line[:breakAt])
			line = strings.TrimLeft(line[breakAt:], " ")
		}
		result = append(result, line)
	}
	return result
}
