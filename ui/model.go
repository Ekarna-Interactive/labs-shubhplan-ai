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
	StepEventType
	StepHostNames
	StepEventDate
	StepVenue
	StepWelcomeMessage
	StepAwaitingWelcomeChoice
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
	TextInput          textinput.Model
	SetupInput         textinput.Model
	Viewport           viewport.Model
	IsSetupMode        bool
	Step               WizardStep
	EventType          string
	HostNames          string
	EventDate          string
	Venue              string
	WelcomeMessage     string
	WelcomeSuggestions []string
	EventDetails       string
	SelectedAspect string
	SelectedStyle  string
	OptionIndex    int
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
	ti.Placeholder = "e.g. Wedding, Naming Ceremony, Housewarming"
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
	initialStep := StepEventType
	if isSetup {
		initialStep = StepAPIKey
	}

	initialLogs := []LogEntry{
		{Sender: "SYSTEM", Text: "✨ Welcome to Shubh CLI — AI Event Design Terminal!"},
	}

	profile, hasProfile := config.LoadEventProfile()
	if profile.TargetAspect == "" {
		profile.TargetAspect = "9:16"
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
		ti.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type style"
		initialLogs = append(initialLogs, LogEntry{
			Sender: "PROFILE",
			Text:   fmt.Sprintf("📋 Loaded active event profile from event_details.md:\n  • Event Type: %s\n  • Host/Couple Names: %s\n  • Date: %s\n  • Venue: %s\n  • Welcome Subheader: %s\n  • Resolution: %s", profile.EventType, profile.HostNames, profile.EventDate, profile.Venue, profile.WelcomeMessage, profile.TargetAspect),
		})
		initialLogs = append(initialLogs, LogEntry{
			Sender: "STEP 3/4",
			Text:   renderStyleMenu(0),
		})
	} else {
		ti.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type event type"
		initialLogs = append(initialLogs, LogEntry{
			Sender: "STEP 1/5",
			Text:   renderEventTypeMenu(0),
		})
	}

	return Model{
		TextInput:      ti,
		SetupInput:     si,
		Viewport:       vp,
		IsSetupMode:    isSetup,
		Step:           initialStep,
		EventType:      profile.EventType,
		HostNames:      profile.HostNames,
		EventDate:      profile.EventDate,
		Venue:          profile.Venue,
		WelcomeMessage: profile.WelcomeMessage,
		EventDetails:   profile.RawDetails,
		SelectedAspect: profile.TargetAspect,
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
