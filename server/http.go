package server

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

// PreviewData represents state passed to the HTML template
type PreviewData struct {
	DisplayTitle   string `json:"displayTitle"`
	WelcomeMessage string `json:"welcomeMessage"`
	CorePrompt     string `json:"corePrompt"`
	Aspect         string `json:"aspect"`
	ImagePath      string `json:"imagePath"`
}

var (
	templateFS embed.FS

	currentData PreviewData
	dataMu      sync.RWMutex
	isStarted   bool
	startMu     sync.Mutex
	serverPort  string
)

// SetTemplateFS allows setting embedded template FS from main
func SetTemplateFS(fs embed.FS) {
	templateFS = fs
}

// UpdatePayload updates the active web preview data
func UpdatePayload(data PreviewData) {
	dataMu.Lock()
	currentData = data
	dataMu.Unlock()
}

// StartServer launches the HTTP web server if it hasn't been started yet
func StartServer(port string) string {
	startMu.Lock()
	defer startMu.Unlock()

	serverPort = port
	if isStarted {
		return fmt.Sprintf("http://localhost:%s", serverPort)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/asset", handleAsset)

	isStarted = true
	go func() {
		addr := ":" + port
		log.Printf("[Web Preview] Server running on http://localhost%s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("[Web Preview] Server stopped: %v", err)
		}
	}()

	return fmt.Sprintf("http://localhost:%s", serverPort)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	dataMu.RLock()
	data := currentData
	dataMu.RUnlock()

	tmplContent, err := templateFS.ReadFile("templates/index.html")
	if err != nil {
		http.Error(w, "Failed to load index template", http.StatusInternalServerError)
		return
	}

	t, err := template.New("index").Parse(string(tmplContent))
	if err != nil {
		http.Error(w, "Failed to parse template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.Execute(w, data)
}

func handleAsset(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "Path parameter missing", http.StatusBadRequest)
		return
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, absPath)
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
