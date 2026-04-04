package visualizer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"beautyctl/logger"
)

// CavaControl manages the cava process
type CavaControl struct {
	cmd        *exec.Cmd
	Output     chan []int
	configPath string
}

// NewCavaControl starts a new cava process
func NewCavaControl(bars int) (*CavaControl, error) {
	// Create a temporary config file
	configFile, err := os.CreateTemp("", "beautyctl-cava-*.conf")
	if err != nil {
		return nil, err
	}
	// Do not defer remove here, moved to Stop()

	// ... (config content generation omitted for brevity if not changing, but replace_file_content needs context)
	// Actually I need to match the whole function to remove the defer safely or specifically target the defer line.
	// Let's rewrite the struct and NewCavaControl start and Stop.

	// Cava config for raw output
	configContent := fmt.Sprintf(`
[general]
bars = 200
framerate = 60

[input]
method = pulse
source = auto

[output]
method = raw
raw_target = /dev/stdout
data_format = ascii
ascii_max_range = 100
channels = mono
`, bars)

	if _, err := configFile.Write([]byte(configContent)); err != nil {
		return nil, err
	}
	configFile.Close()

	// Start cava
	cmd := exec.Command("cava", "-p", configFile.Name())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// Logging via global logger
	logger.Printf("Starting Cava with config: %s", configFile.Name())

	if err := cmd.Start(); err != nil {
		logger.Printf("Failed to start cava: %v", err)
		return nil, err
	}

	c := &CavaControl{
		cmd:        cmd,
		Output:     make(chan []int),
		configPath: configFile.Name(),
	}

	// Read output in a goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			text := scanner.Text()
			// Output is like "12;45;88;"
			// We need to trim the trailing semicolon if present
			text = strings.TrimSuffix(text, ";")
			parts := strings.Split(text, ";")
			var values []int
			for _, p := range parts {
				if v, err := strconv.Atoi(p); err == nil {
					values = append(values, v)
				}
			}
			c.Output <- values
		}
		close(c.Output)
	}()

	return c, nil
}

// Stop kills the cava process
func (c *CavaControl) Stop() {
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
	if c.configPath != "" {
		os.Remove(c.configPath)
	}
}
