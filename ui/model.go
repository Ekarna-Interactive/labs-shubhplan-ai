package ui

import (
	"fmt"

	"github.com/Ekarna-Interactive/ShubhPlan-CLI/config"
	"github.com/Ekarna-Interactive/ShubhPlan-CLI/generator"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WizardStep defines the guided user flow steps
type WizardStep int

const (
	StepAPIKey WizardStep = iota
	StepEventDetails
	StepAspectSelection
	StepStyleSelection
	StepPromptChoice
	StepAwaitingSuggestionChoice
	StepComplete
)

// LogEntry represents an item in the terminal log output stream
type LogEntry struct {
	Sender  string
	Text    string
	IsError bool
}

// Model represents the Bubble Tea application state
type Model struct {
	TextInput      textinput.Model
	SetupInput     textinput.Model
	Viewport       viewport.Model
	IsSetupMode    bool
	Step           WizardStep
	EventDetails   string
	SelectedAspect string
	SelectedStyle  string
	WelcomeMessage string
	Suggestions    []string
	Spinner        spinner.Model
	Loading        bool
	StatusMsg      string
	Config         config.Config
	Builder        generator.PromptBuilder
	Logs           []LogEntry
	LastImage      string
	LastTitle      string
	LastAspect     string
	Width          int
	Height         int
	Ready          bool
}

// InitialModel initializes the Bubble Tea state
func InitialModel(cfg config.Config, builder generator.PromptBuilder) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter Event Details e.g. Wedding for Rohan & Ananya on Dec 12 at Bengaluru"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 80

	si := textinput.New()
	si.Placeholder = "Paste Gemini API Key here (or press Enter to skip for offline dry-run mode)"
	si.CharLimit = 256
	si.Width = 80

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	vp := viewport.New(80, 20)

	isSetup := cfg.GeminiAPIKey == ""
	initialStep := StepEventDetails
	if isSetup {
		initialStep = StepAPIKey
	}

	initialLogs := []LogEntry{
		{Sender: "SYSTEM", Text: "✨ Welcome to Shubh CLI — AI Event Design Terminal!"},
	}

	loadedDetails := ""
	loadedWelcome := ""
	loadedAspect := "9:16"
	profile, hasProfile := config.LoadEventProfile()
	if hasProfile {
		loadedDetails = profile.RawDetails
		loadedWelcome = profile.WelcomeMessage
		if profile.TargetAspect != "" {
			loadedAspect = profile.TargetAspect
		}
	}

	if isSetup {
		si.Focus()
		initialLogs = append(initialLogs, LogEntry{
			Sender:  "SETUP",
			Text:    "🔑 INITIAL SETUP: GEMINI_API_KEY is missing.\nGet your free key at https://aistudio.google.com/api-keys\nEnter your key below to enable live AI image generation, or press Enter to skip for offline dry-run mode.",
			IsError: true,
		})
	} else if hasProfile {
		initialStep = StepStyleSelection
		ti.Placeholder = "Enter style number (1-7) or type style e.g. 2 or Paper Cut"
		initialLogs = append(initialLogs, LogEntry{
			Sender: "PROFILE",
			Text:   fmt.Sprintf("📋 Loaded active event profile from event_details.md:\n\"%s\" (Aspect: %s)", loadedDetails, loadedAspect),
		})
		initialLogs = append(initialLogs, LogEntry{
			Sender: "STEP 3/4",
			Text:   renderStyleMenu(),
		})
	} else {
		initialLogs = append(initialLogs, LogEntry{
			Sender: "STEP 1/4",
			Text:   "📋 Enter your Event Details (Event Type, Names, Date, Location):\ne.g. 'Wedding for Rohan & Ananya on Dec 12 at Leela Palace, Bengaluru' or 'Naming Ceremony for Aarav on Nov 5'",
		})
	}

	return Model{
		TextInput:      ti,
		SetupInput:     si,
		Viewport:       vp,
		IsSetupMode:    isSetup,
		Step:           initialStep,
		EventDetails:   loadedDetails,
		SelectedAspect: loadedAspect,
		WelcomeMessage: loadedWelcome,
		Spinner:        s,
		Loading:        false,
		StatusMsg:      "Ready",
		Config:         cfg,
		Builder:        builder,
		Logs:           initialLogs,
		Ready:          false,
	}
}

// Init initializes Bubble Tea commands
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}
