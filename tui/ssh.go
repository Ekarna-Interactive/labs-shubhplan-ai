package tui

import (
	"fmt"
	"log"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/generator"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
)

type SSHServer struct {
	port int
}

func NewSSHServer(port int) *SSHServer {
	return &SSHServer{port: port}
}

func (s *SSHServer) Start() error {
	server, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("0.0.0.0:%d", s.port)),
		wish.WithHostKeyPath("data/term_info_ed25519"),
		wish.WithMiddleware(
			bm.Middleware(teaHandler),
			lm.Middleware(),
		),
	)
	if err != nil {
		return err
	}

	log.Printf("⚡ Wish SSH TUI Server listening on 0.0.0.0:%d", s.port)
	return server.ListenAndServe()
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()
	cfg := config.LoadConfig()

	// Inspect client-passed environment variables (e.g. ssh -o SendEnv=GEMINI_API_KEY -o SendEnv=HONCHO_API_KEY ...)
	for _, env := range s.Environ() {
		if strings.HasPrefix(env, "GEMINI_API_KEY=") {
			cfg.GeminiAPIKey = strings.TrimPrefix(env, "GEMINI_API_KEY=")
		}
		if strings.HasPrefix(env, "HONCHO_API_KEY=") {
			cfg.HonchoAPIKey = strings.TrimPrefix(env, "HONCHO_API_KEY=")
		}
	}

	builder := generator.NewBasicBuilder()

	model := InitialModel(cfg, builder)
	model.Width = pty.Window.Width
	model.Height = pty.Window.Height

	return model, []tea.ProgramOption{tea.WithAltScreen()}
}
