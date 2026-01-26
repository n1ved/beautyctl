package image

import (
    "bytes"
    "fmt"
    "os/exec"
    
    "beautyctl/logger"
)

// RenderChafa uses the local chafa binary to render an image to a string
func RenderChafa(path string, width, height int) string {
    // Try local bin first, then system
    cmdPath := "./bin/chafa"
    if _, err := exec.LookPath(cmdPath); err != nil {
        cmdPath = "chafa" // System path
    }
    
    // Arguments:
    // -s WxH : Size
    // --format symbols : Use symbols (or 'sixel' if supported, but bubbletea handles text better)
    // -c full : Full color
    
    args := []string{
        "-s", fmt.Sprintf("%dx%d", width, height),
        "--format=symbols",
        "--symbols=all", // Use Braille/etc for higher resolution
        "-c", "full",    // Force truecolor
        path,
    }
    
    cmd := exec.Command(cmdPath, args...)
    var out bytes.Buffer
    var stderr bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &stderr
    
    if err := cmd.Run(); err != nil {
        logger.Printf("Chafa error: %v. Stderr: %s", err, stderr.String())
        return ""
    }
    
    return out.String()
}
