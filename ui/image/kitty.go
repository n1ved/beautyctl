package image

import (
    "bytes"
    "encoding/base64"
    "fmt"
    "io"
    "net/http"
    "os"
    
    "beautyctl/logger"
)

// RenderKitty downloads an image and renders it using Kitty ICAT protocol
// Returns the escape sequence string
func RenderKitty(url string, width, height int) string {
    logger.Printf("RenderKitty called with URL: %s", url)
    if url == "" {
        logger.Println("RenderKitty: Empty URL")
        return ""
    }
    
    var data string
    isLocal := false
    
    if len(url) > 4 && url[:4] == "http" {
        logger.Printf("Downloading image from: %s", url)
        resp, err := http.Get(url)
        if err != nil {
            logger.Printf("HTTP Get error: %v", err)
            return ""
        }
        defer resp.Body.Close()
        body, err := io.ReadAll(resp.Body)
        if err != nil {
            logger.Printf("ReadAll error: %v", err)
            return ""
        }
        data = base64.StdEncoding.EncodeToString(body)
    } else {
        // Assume local file
        logger.Printf("Using local file path: %s", url)
        // For t=f, we send the base64 encoded PATH, not content.
        isLocal = true
        data = base64.StdEncoding.EncodeToString([]byte(url))
    }
    
    // Construct escape sequence
    var buf bytes.Buffer
    
    if isLocal {
        // t=f: File
        // payload is encoded path
        // We still split if path is super long (unlikely to exceed 4096, but safe to do? No, spec says payload is data)
        // Usually path fits in one chunk.
        
        // a=T: Transmit and Display
        // t=f: File
        // c,r: Dimensions
        // z=1: Z-index
        header := fmt.Sprintf("a=T,t=f,c=%d,r=%d,z=1,", width, height)
        
        // Write single chunk (assume path < 4096 bytes)
        fmt.Fprintf(&buf, "\x1b_G%sm=0;%s\x1b\\", header, data)
        
    } else {
        // t=d: Direct data (HTTP download)
        chunks := splitSubN(data, 4096)
        for i, chunk := range chunks {
            isLast := 0
            if i == len(chunks)-1 {
                isLast = 0
            } else {
                isLast = 1
            }
            
            header := ""
            if i == 0 {
                header = fmt.Sprintf("a=T,t=d,c=%d,r=%d,z=1,", width, height)
            }
            fmt.Fprintf(&buf, "\x1b_G%sm=%d;%s\x1b\\", header, isLast, chunk)
        }
    }
    
    str := buf.String()
    _ = os.WriteFile("debug_escape.txt", buf.Bytes(), 0644)
    return str
}


func splitSubN(s string, n int) []string {
    sub := ""
    subs := []string{}
    runes := []rune(s)
    l := len(runes)
    for i, r := range runes {
        sub = sub + string(r)
        if (i+1)%n == 0 {
            subs = append(subs, sub)
            sub = ""
        } else if (i+1) == l {
            subs = append(subs, sub)
        }
    }
    return subs
}
