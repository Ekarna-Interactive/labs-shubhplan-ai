package tui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/client"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/generator"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
)

type SSHServer struct {
	port int
}

func NewSSHServer(port int) *SSHServer {
	return &SSHServer{port: port}
}

func (s *SSHServer) Start() error {
	options := []ssh.Option{
		wish.WithAddress(fmt.Sprintf("0.0.0.0:%d", s.port)),
		func(srv *ssh.Server) error {
			srv.LocalPortForwardingCallback = func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
				return true
			}
			srv.ChannelHandlers = map[string]ssh.ChannelHandler{
				"session":      ssh.DefaultSessionHandler,
				"direct-tcpip": ssh.DirectTCPIPHandler,
			}
			return nil
		},
		wish.WithMiddleware(
			bm.Middleware(teaHandler),
			lm.Middleware(),
		),
	}

	hostKeyPEM := strings.TrimSpace(os.Getenv("SSH_HOST_KEY"))
	if hostKeyPEM != "" {
		options = append(options, wish.WithHostKeyPEM([]byte(hostKeyPEM)))
	} else {
		_ = os.MkdirAll("data", 0755)
		hostKeyPath := filepath.Join("data", "term_info_ed25519")
		options = append(options, wish.WithHostKeyPath(hostKeyPath))
	}

	server, err := wish.NewServer(options...)
	if err != nil {
		log.Printf("⚠️ Wish server creation error: %v", err)
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

	if cfg.GeminiAPIKey != "" {
		_ = os.Setenv("GEMINI_API_KEY", cfg.GeminiAPIKey)
	}
	if cfg.HonchoAPIKey != "" {
		_ = os.Setenv("HONCHO_API_KEY", cfg.HonchoAPIKey)
		client.GetHonchoManager().SetAPIKey(cfg.HonchoAPIKey)
	}

	// Force 24-bit TrueColor rendering for Lipgloss styles over Wish SSH
	lipgloss.SetColorProfile(termenv.TrueColor)

	builder := generator.NewBasicBuilder()

	model := InitialModel(cfg, builder)
	model.Width = pty.Window.Width
	model.Height = pty.Window.Height

	return model, []tea.ProgramOption{tea.WithAltScreen()}
}
