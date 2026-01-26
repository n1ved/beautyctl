package image

import (
	"bytes"
    "fmt"
    "image"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
    "os/exec"

	"beautyctl/logger"
)

// RenderJP2A uses the jp2a binary to render a JPEG image to ASCII
func RenderJP2A(path string, width, height int) string {
    // jp2a only supports JPEG. Convert if necessary.
    jpegPath, err := ensureJPEG(path)
    if err != nil {
        logger.Printf("Image conversion error: %v", err)
        return ""
    }
    // If we created a temp file, we should probably clean it up.
    // But RenderJP2A returns a string, so we can clean up after running cmd.
    if jpegPath != path {
        defer os.Remove(jpegPath)
    }

    // jp2a --width=W --height=H --colors path
    args := []string{
        fmt.Sprintf("--width=%d", width),
        fmt.Sprintf("--height=%d", height),
        "--colors", 
        jpegPath,
    }
    
    cmd := exec.Command("jp2a", args...)
    var out bytes.Buffer
    var stderr bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &stderr
    
    if err := cmd.Run(); err != nil {
        logger.Printf("JP2A error: %v. Stderr: %s", err, stderr.String())
        return ""
    }
    
    return out.String()
}

func ensureJPEG(path string) (string, error) {
    // Check extension first for quick fail (though magic number is better)
    // jp2a usually complains about magic number.
    
    f, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer f.Close()
    
    // Peek magic number
    header := make([]byte, 512)
    if _, err := f.Read(header); err != nil {
        return "", err
    }
    
    // Reset seek
    f.Seek(0, 0)
    
    contentType := http.DetectContentType(header)
    if contentType == "image/jpeg" {
        return path, nil // Already JPEG
    }
    
    // Needs conversion
    img, _, err := image.Decode(f)
    if err != nil {
        return "", fmt.Errorf("failed to decode image: %w", err)
    }
    
    // Create temp jpeg
    tmp, err := os.CreateTemp("", "beautyctl-*.jpg")
    if err != nil {
        return "", err
    }
    defer tmp.Close()
    
    if err := jpeg.Encode(tmp, img, &jpeg.Options{Quality: 90}); err != nil {
        return "", fmt.Errorf("failed to encode jpeg: %w", err)
    }
    
    return tmp.Name(), nil
}
