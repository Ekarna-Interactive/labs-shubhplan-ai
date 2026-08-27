package main

import (
	"context"
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"shubh-plan-web/pkg/server"
)

//go:embed all:web
var webFS embed.FS

func main() {
	defaultPort := 3000
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			defaultPort = p
		}
	}
	portFlag := flag.Int("port", defaultPort, "HTTP Web Server Port")
	flag.Parse()

	// Capture Ctrl+C (SIGINT) and SIGTERM for instant graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("[Shubh Plan Web] Booting independent Genkit Go + Web UI application...")
	srv := server.NewServer(*portFlag, webFS)

	go func() {
		if err := srv.Start(ctx); err != nil && err != http.ErrServerClosed {
			log.Printf("[Shubh Plan Web Server Error] %v", err)
		}
	}()

	log.Println("🟢 Running Web Server mode. Press Ctrl+C to stop.")
	<-ctx.Done()
	log.Println("[Shubh Plan Web] Shutdown complete.")
}
