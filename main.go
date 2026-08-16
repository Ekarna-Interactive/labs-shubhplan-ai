package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/generator"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/tui"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/web"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	httpPort := flag.Int("port", 3000, "HTTP Web UI Port")
	sshPort := flag.Int("ssh-port", 2222, "Wish SSH TUI Port")
	serverMode := flag.Bool("server", false, "Run in daemonized server mode without local terminal TUI")
	dataDir := flag.String("data-dir", "./data", "Local data storage directory")
	flag.Parse()

	if os.Getenv("SERVER_MODE") == "true" || os.Getenv("SERVER_MODE") == "1" {
		*serverMode = true
	}

	// Ensure directories exist
	os.MkdirAll(*dataDir, 0755)
	os.MkdirAll("./output", 0755)

	// Redirect logger to data/app.log to prevent terminal screen corruption in TUI mode
	logFile, err := os.OpenFile(filepath.Join(*dataDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		if *serverMode {
			log.SetOutput(io.MultiWriter(os.Stdout, logFile))
		} else {
			log.SetOutput(logFile)
		}
		defer logFile.Close()
	}

	log.Println("🚀 Initializing Shubh Plan AI Open Source Application...")
	log.Printf("📂 Data Directory: %s", *dataDir)
	log.Printf("🌐 HTMX Web UI Server starting on http://localhost:%d", *httpPort)
	log.Printf("⚡ Wish SSH TUI Server starting on ssh://localhost:%d", *sshPort)

	// Start Wish SSH TUI Server in background goroutine
	sshServer := tui.NewSSHServer(*sshPort)
	go func() {
		if err := sshServer.Start(); err != nil {
			log.Printf("Wish SSH Server notice: %v", err)
		}
	}()

	// Start HTTP Web UI Server in background goroutine
	httpServer := web.NewHTTPServer(*httpPort)
	go func() {
		if err := httpServer.Start(); err != nil {
			log.Printf("HTTP Web Server error: %v", err)
		}
	}()

	if *serverMode {
		log.Println("🔒 Running in Daemonized Server Mode (Headless TTY). Press Ctrl+C to stop.")
		select {} // Keep daemon process alive
	}

	// Launch local interactive Bubble Tea TUI session
	cfg := config.LoadConfig()
	builder := generator.NewBasicBuilder()
	p := tea.NewProgram(tui.InitialModel(cfg, builder), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Fatal error running TUI session: %v\n", err)
		os.Exit(1)
	}
}
