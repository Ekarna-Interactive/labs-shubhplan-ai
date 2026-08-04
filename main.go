package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/Ekarna-Interactive/ShubhPlan-CLI/config"
	"github.com/Ekarna-Interactive/ShubhPlan-CLI/generator"
	"github.com/Ekarna-Interactive/ShubhPlan-CLI/server"
	"github.com/Ekarna-Interactive/ShubhPlan-CLI/ui"

	tea "github.com/charmbracelet/bubbletea"
)

//go:embed templates/index.html
var templateFS embed.FS

func main() {
	// 1. Pass embedded HTML template filesystem to server
	server.SetTemplateFS(templateFS)

	// 2. Load runtime environment configuration
	cfg := config.LoadConfig()

	// 3. Instantiate open-source community PromptBuilder implementation
	builder := generator.NewBasicBuilder()

	// 4. Initialize Bubble Tea UI model
	model := ui.InitialModel(cfg, builder)

	// 5. Start Bubble Tea TUI program
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting Shubh CLI terminal app: %v\n", err)
		os.Exit(1)
	}
}
