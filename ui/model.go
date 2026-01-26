package ui

import (
	"fmt"
    "io"
    "net/http"
    "os"
	"strings"
	"time"

	"beautyctl/player"
	"beautyctl/ui/image"
	"beautyctl/visualizer"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
    "github.com/common-nighthawk/go-figure"
)

type tickMsg time.Time
type visualizerMsg []int
type imageMsg string // contains the render string

type Model struct {
	playerControl *player.Control
	cavaControl   *visualizer.CavaControl
	
	metadata      *player.Metadata
	visualizerData []int
	
	err           error
    
    width         int
    height        int
    
    // Progress bar related
    progress      float64
    
    // Config
    renderer      string
    
    // Cache
    titleRender   string
    
    // Image related
    lastArtURL    string
    artRender     string
}

func NewModel(renderer string) (*Model, error) {
	pc := player.NewControl()
    // 200 bars for visualizer to cover wide screens
	cc, err := visualizer.NewCavaControl(200)
	if err != nil {
		return nil, err
	}

	return &Model{
		playerControl:     pc,
		cavaControl:       cc,
		visualizerData:    make([]int, 200),
        renderer:          renderer,
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
        case "up":
            m.playerControl.VolumeUp()
        case "down":
            m.playerControl.VolumeDown()
        case "right":
            m.playerControl.SeekForward()
        case "left":
            m.playerControl.SeekBackward()
		}
		// Force an immediate update after action
		return m, tickCmd()

	case tickMsg:
        meta, err := m.playerControl.GetMetadata()
		m.metadata = meta
		m.err = err
        
        // Check if art changed
        if meta != nil {
            // Check title change for ASCII art cache
            if m.metadata == nil || meta.Title != m.metadata.Title {
                // Generate ASCII art
                // Note: We need width for responsive font choice, but here we might not have updated width yet if it's a resize msg?
                // It's safe to use m.width.
                
                tArt := figure.NewFigure(meta.Title, "rectangles", true).String()
                headingWidth := lipgloss.Width(tArt)
                // Heuristic: If wider than screen minus safety padding (e.g. 10)
                if headingWidth > m.width - 10 {
                    tArt = figure.NewFigure(meta.Title, "small", true).String()
                    if lipgloss.Width(tArt) > m.width - 10 {
                        tArt = TitleStyle.Render(meta.Title)
                    }
                }
                m.titleRender = TitleStyle.Render(tArt)
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
                     
                     // 80x20 approx size? Or 20x10?
                     // Let's try 34x15 for higher res text art.
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
	}

	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nEnsure a music player is running.", m.err)
	}
	if m.metadata == nil {
		return "Waiting for player..."
	}

    width := m.width
    height := m.height
    
    // --- Layout Calculations ---
    // If width is too small, just show simplistic view
    if width < 20 {
        return "Terminal too small"
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
    if infoMaxWidth < 20 { infoMaxWidth = 20 }
    
    // Title is already cached in m.titleRender
    titleArt := m.titleRender

    
    // Progress Bar
    barWidth := int(float64(infoMaxWidth) * 0.8)
    if barWidth > 60 { barWidth = 60 }
    if barWidth < 10 { barWidth = 10 }

    progress := 0.0
    if m.metadata.Length > 0 {
        progress = float64(m.metadata.Position) / float64(m.metadata.Length)
    }
    if progress > 1.0 { progress = 1.0 }
    if progress < 0.0 { progress = 0.0 }
    
    filledChars := int(progress * float64(barWidth))
    emptyChars := barWidth - filledChars
    if emptyChars < 0 { emptyChars = 0 }
    
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
    help := helpStyle.Render(" [Space] Play/Pause  [N] Next  [P] Prev  [←/→] Seek  [↑/↓] Vol  [Q] Quit ")
    
    // Combine Top + Help
    topContent := lipgloss.JoinVertical(lipgloss.Center, topSection, "", help)
    topHeight := lipgloss.Height(topContent)

    
    // --- Visualizer Section ---
    // Make visualizer full width minus padding
    visWidth := width - 4 
    if visWidth < 0 { visWidth = 0 }
    
    // Dynamic height calculation
    // Available height = Window Height - Top Content Height - Padding (approx 2 lines)
    visHeight := height - topHeight - 2
    
    // Clamp height
    if visHeight < 3 { visHeight = 3 }
    if visHeight > 30 { visHeight = 30 } // allow taller visuals effectively filling screen
    
    // We have m.visualizerData (200 items potentially). 
    // We need to slice it to visWidth. 
    numBars := visWidth
    if numBars > len(m.visualizerData) {
        numBars = len(m.visualizerData)
    }
    
    dataSubset := m.visualizerData[:numBars]
    
    // Grid rendering
    var rows []string
    
    // We want 'smooth' bars using 1/8th blocks.
    // Total vertical resolution = visHeight * 8
    
    for y := visHeight - 1; y >= 0; y-- {
        var rowBuilder strings.Builder
        for _, val := range dataSubset {
            // val is 0-100
            // Scaled to total resolution
            totalLevels := visHeight * 8
            scaledVal := (val * totalLevels) / 100
            
            // Current row represents levels from (y*8) to ((y+1)*8 - 1)
            rowBase := y * 8
            
            var char string
            if scaledVal >= rowBase+8 {
                // Fully filled above this row
                char = "█"
            } else if scaledVal < rowBase {
                char = " "
            } else {
                // Fractional part in this row
                remainder := scaledVal - rowBase
                chars := []string{" ", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
                if remainder >= 0 && remainder < len(chars) {
                    char = chars[remainder]
                } else {
                    char = "█" // Should not happen given check above
                }
            }
            rowBuilder.WriteString(char)
        }
        rows = append(rows, BarStyle.Render(rowBuilder.String()))
    }
    
    vis := lipgloss.JoinVertical(lipgloss.Left, rows...)
    visBox := lipgloss.NewStyle().
        Width(width - 2). // Border accounts for 2
        Align(lipgloss.Center).
        Border(lipgloss.RoundedBorder()).
        BorderForeground(BorderColor).
        Render(vis)
        
    // Final Layout
    // We want the visualizer at the bottom, and info centered in the remaining top space.
    // Actually, simply stacking them is fine if we sized the visualizer to take the remaining space.
    // But lipgloss.Place helps center the top content if there is extra gap.
    
    // Calculate gap
    contentHeight := topHeight + lipgloss.Height(visBox)
    gap := height - contentHeight
    if gap < 0 { gap = 0 }
    
    // Simplest approach: Join vertical. The visualizer size calc should have ensured it fits.
    // If we want Info to be visually centered in the top area:
    // It's effectively free flowing since visualizer takes the rest.
    
    return lipgloss.JoinVertical(lipgloss.Center,
        lipgloss.PlaceHorizontal(width, lipgloss.Center, topContent),
        lipgloss.NewStyle().Height(gap/2).Render(""), // Gap spacer? Or just let it be.
        visBox,
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
