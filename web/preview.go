package web

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"sync"
)

var (
	isStarted  bool
	startMu    sync.Mutex
	serverPort string
)

// StartServer returns the active web preview URL
func StartServer(port string) string {
	startMu.Lock()
	defer startMu.Unlock()

	if port != "" {
		serverPort = port
	} else {
		serverPort = "3000"
	}
	isStarted = true
	return fmt.Sprintf("http://localhost:%s", serverPort)
}

// OpenBrowser launches default OS browser pointing to url
func OpenBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Printf("[Web Preview] Failed to launch browser: %v", err)
	}
}
