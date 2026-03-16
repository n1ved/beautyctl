package lyrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ErrNotFound is returned when the lyrics API responds with 404.
var ErrNotFound = errors.New("lyrics not found")

type lyricsResponse struct {
	Lyrics string `json:"lyrics"`
	Error  string `json:"error"`
}

// Fetch retrieves lyrics for the given artist and title from lyrics.ovh.
// Returns ErrNotFound when the track has no lyrics (HTTP 404).
func Fetch(artist, title string) (string, error) {
	if artist == "" || title == "" {
		return "", ErrNotFound
	}

	apiURL := fmt.Sprintf(
		"https://api.lyrics.ovh/v1/%s/%s",
		url.PathEscape(artist),
		url.PathEscape(title),
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("lyrics fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lyrics API returned status %d", resp.StatusCode)
	}

	var lr lyricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return "", fmt.Errorf("lyrics decode error: %w", err)
	}

	text := strings.TrimSpace(lr.Lyrics)
	if text == "" {
		return "", ErrNotFound
	}

	return text, nil
}
