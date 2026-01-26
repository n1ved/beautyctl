package player

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Metadata represents the current song and player state
type Metadata struct {
	Title      string
	Artist     string
	Album      string
	ArtURL     string
	Status     string
	Position   time.Duration
	Length     time.Duration
	PlayerName string
}

// Control provides methods to interact with the player
type Control struct{}

// NewControl creates a new player controller
func NewControl() *Control {
	return &Control{}
}

// GetMetadata fetches the current player metadata using playerctl
// Uses a custom format string to parse the output
func (c *Control) GetMetadata() (*Metadata, error) {
	// Format: Title|Artist|Album|ArtUrl|Status|Position|Length|PlayerName
	format := "{{ title }}<|>{{ artist }}<|>{{ album }}<|>{{ mpris:artUrl }}<|>{{ status }}<|>{{ position }}<|>{{ mpris:length }}<|>{{ playerName }}"
	cmd := exec.Command("playerctl", "metadata", "--format", format)
	out, err := cmd.Output()
	if err != nil {
		// If no player is found, playerctl returns exit code 1
		return nil, fmt.Errorf("no player found or error: %v", err)
	}

	parts := strings.Split(string(out), "<|>")
	if len(parts) < 8 {
		return nil, fmt.Errorf("unexpected output format")
	}

	pos, _ := strconv.ParseInt(strings.TrimSpace(parts[5]), 10, 64)
	len, _ := strconv.ParseInt(strings.TrimSpace(parts[6]), 10, 64)

	return &Metadata{
		Title:      strings.TrimSpace(parts[0]),
		Artist:     strings.TrimSpace(parts[1]),
		Album:      strings.TrimSpace(parts[2]),
		ArtURL:     strings.TrimSpace(parts[3]),
		Status:     strings.TrimSpace(parts[4]),
		Position:   time.Duration(pos) * time.Microsecond,
		Length:     time.Duration(len) * time.Microsecond,
		PlayerName: strings.TrimSpace(parts[7]),
	}, nil
}

// FormatDuration converts a duration to MM:SS format
func FormatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

// PlayPause toggles playback
func (c *Control) PlayPause() error {
	return exec.Command("playerctl", "play-pause").Run()
}

// Next plays the next track
func (c *Control) Next() error {
	return exec.Command("playerctl", "next").Run()
}

// Previous plays the previous track
func (c *Control) Previous() error {
	return exec.Command("playerctl", "previous").Run()
}

// VolumeUp increases volume by 5%
func (c *Control) VolumeUp() error {
    return exec.Command("playerctl", "volume", "0.05+").Run()
}

// VolumeDown decreases volume by 5%
func (c *Control) VolumeDown() error {
    return exec.Command("playerctl", "volume", "0.05-").Run()
}

// SeekForward seeks forward 5 seconds
func (c *Control) SeekForward() error {
    return exec.Command("playerctl", "position", "5+").Run()
}

// SeekBackward seeks backward 5 seconds
func (c *Control) SeekBackward() error {
    return exec.Command("playerctl", "position", "5-").Run()
}
