package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/client"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/generator"
	"golang.org/x/term"

	"github.com/charmbracelet/bubbles/key"
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
	StepCurrency
	StepWelcomeMessage
	StepAwaitingWelcomeChoice
	StepAspectSelection
	StepStyleSelection
	StepPromptChoice
	StepAwaitingSuggestionChoice
	StepComplete
)

// ActiveTab defines the 5 main top-level views
type ActiveTab int

const (
	TabAgentChat ActiveTab = iota
	TabItinerary
	TabBudget
	TabRSVP
	TabDesignStudio
)

// LogEntry represents an item in the terminal log output stream
type LogEntry struct {
	Sender  string
	Text    string
	IsError bool
}

// Model represents the Bubble Tea application state
type Model struct {
	SessionID          string
	TextInput          textinput.Model
	SetupInput         textinput.Model
	Viewport           viewport.Model
	ActiveTab          ActiveTab
	IsSetupMode        bool
	SetupStep          int
	Step               WizardStep
	EventID            string
	EventType          string
	HostNames          string
	EventDate          string
	Venue              string
	Currency           string
	WelcomeMessage     string
	PlannerName        string
	PlannerRole        string
	WelcomeSuggestions []string
	EventDetails       string
	SelectedAspect     string
	SelectedStyle      string
	OptionIndex        int
	Suggestions        []string
	Spinner            spinner.Model
	Loading            bool
	StatusMsg          string
	Config             config.Config
	Builder            *generator.BasicBuilder
	Logs               []LogEntry
	ItineraryItems     []client.SubEventItem
	BudgetSummary      client.BudgetSummary
	RSVPOverview       client.RSVPOverview
	PeerCards          map[string]string
	LastImage          string
	LastTitle          string
	LastAspect         string
	RSVPStep           RSVPWizardStep
	RSVPData           RSVPWizardData
	ShowVerbose        bool
	Width              int
	Height             int
	Ready              bool
}

type RSVPWizardStep int

const (
	RSVPStepInactive RSVPWizardStep = iota
	RSVPStepGuestName
	RSVPStepPhone
	RSVPStepStatus
	RSVPStepHeadcount
	RSVPStepDietary
	RSVPStepCab
)

type RSVPWizardData struct {
	Name      string
	Phone     string
	Status    string
	Headcount int
	PlusOnes  int
	Dietary   string
	Cab       bool
}

// InitialModel initializes the Bubble Tea state
func InitialModel(cfg config.Config, builder *generator.BasicBuilder) Model {
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
	vp.KeyMap.PageUp = key.NewBinding(key.WithKeys("pgup", "ctrl+u"))
	vp.KeyMap.PageDown = key.NewBinding(key.WithKeys("pgdown", "ctrl+d"))
	vp.KeyMap.HalfPageUp = key.NewBinding(key.WithKeys("ctrl+b"))
	vp.KeyMap.HalfPageDown = key.NewBinding(key.WithKeys("ctrl+f"))
	vp.KeyMap.Up = key.NewBinding(key.WithKeys("up"))
	vp.KeyMap.Down = key.NewBinding(key.WithKeys("down"))

	geminiKey := cfg.GeminiAPIKey
	honchoKey := cfg.HonchoAPIKey
	if honchoKey == "" {
		honchoKey = strings.TrimSpace(os.Getenv("HONCHO_API_KEY"))
	}

	isSetup := geminiKey == ""
	initialStep := StepEventType
	if isSetup {
		initialStep = StepAPIKey
	}

	honchoStatusText := "🟡 Inbuilt Local Store (./data/honcho_memory.json)"
	if honchoKey != "" {
		honchoStatusText = "🟢 Honcho Cloud Memory (api.honcho.dev/v3)"
	}

	initialLogs := []LogEntry{
		{Sender: "SYSTEM", Text: "✨ Welcome to Shubh Plan AI — Event Planning Concierge!"},
		{Sender: "AI Concierge", Text: fmt.Sprintf("🤖 AI Assistants ready: Timeline Assistant, Vendor Assistant, Budget Assistant, Guest Assistant.\nMemory: %s", honchoStatusText)},
	}

	profile, hasProfile := config.LoadEventProfile()
	if profile.TargetAspect == "" {
		profile.TargetAspect = "9:16"
	}
	if profile.DefaultCurrency == "" {
		profile.DefaultCurrency = "USD"
	}

	if isSetup {
		si.Focus()
		initialLogs = append(initialLogs, LogEntry{
			Sender:  "SETUP",
			Text:    "🔑 SETUP CHECK: GEMINI_API_KEY is missing.\nGet your free key at https://aistudio.google.com/api-keys\nEnter your key below to enable live AI generation, or press Enter to skip for offline dry-run mode.",
			IsError: true,
		})
	} else {
		initialLogs = append(initialLogs, LogEntry{
			Sender: "SETUP",
			Text:   "🟢 GEMINI_API_KEY is configured and active.",
		})
		if hasProfile {
			initialStep = StepComplete
			client.GetHonchoManager().EnsureWorkspaceCreated(profile.GetEventID(), fmt.Sprintf("%s for %s", profile.EventType, profile.HostNames))
			ti.Placeholder = "Type an event prompt (e.g. 'Build Mehendi timeline', 'Check floral budget')..."
			initialLogs = append(initialLogs, LogEntry{
				Sender: "PROFILE",
				Text:   fmt.Sprintf("📋 Active event profile from event_details.md loaded: %s for %s (%s at %s, Currency: %s)", profile.EventType, profile.HostNames, profile.EventDate, profile.Venue, profile.DefaultCurrency),
			})
			initialLogs = append(initialLogs, LogEntry{
				Sender: "ORCHESTRATOR",
				Text:   "💬 Type any prompt below to query agents (or '/wizard' to run invitation design generator, '/clear' to wipe terminal).",
			})
		} else {
			initialStep = StepEventType
			ti.Focus()
			ti.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type event type"
			initialLogs = append(initialLogs, LogEntry{
				Sender: "PROFILE CHECK",
				Text:   "📋 No active event_details.md found. Let's set up your event profile!\nStep 1/3: Select or type Event Type below.",
			})
			initialLogs = append(initialLogs, LogEntry{
				Sender: "STEP 1/3",
				Text:   renderEventTypeMenu(0),
			})
		}
	}

	var initialBudget client.BudgetSummary
	if profile.TotalBudget > 0 {
		initialBudget = client.BudgetSummary{
			TotalEstimated: profile.TotalBudget,
			TotalActual:    0,
			Categories: []client.BudgetCategory{
				{Name: "Venue Rental (20%)", Estimated: profile.TotalBudget * 0.20, Actual: 0},
				{Name: "Food & Beverage (35%)", Estimated: profile.TotalBudget * 0.35, Actual: 0},
				{Name: "Decor & Styling (20%)", Estimated: profile.TotalBudget * 0.20, Actual: 0},
				{Name: "Sound & Entertainment (10%)", Estimated: profile.TotalBudget * 0.10, Actual: 0},
				{Name: "Logistics & Misc (5%)", Estimated: profile.TotalBudget * 0.05, Actual: 0},
				{Name: "Contingency Buffer (10%)", Estimated: profile.TotalBudget * 0.10, Actual: 0},
			},
		}
	}

	var initialRSVP client.RSVPOverview
	if profile.EstimatedGuests > 0 {
		initialRSVP.TotalGuests = profile.EstimatedGuests
	}
	if localRSVPs, err := config.LoadGuestRSVPsFromMarkdown(); err == nil && len(localRSVPs) > 0 {
		initialRSVP.DietaryReqs = make(map[string]int)
		for _, r := range localRSVPs {
			if strings.EqualFold(r.Status, "declined") {
				initialRSVP.Declined += r.Headcount
			} else {
				initialRSVP.Attending += r.Headcount
			}
			if r.Dietary != "" && r.Dietary != "None" {
				initialRSVP.DietaryReqs[r.Dietary] += r.Headcount
			}
		}
		if initialRSVP.TotalGuests > 0 {
			initialRSVP.Pending = initialRSVP.TotalGuests - (initialRSVP.Attending + initialRSVP.Declined)
			if initialRSVP.Pending < 0 {
				initialRSVP.Pending = 0
			}
		}
	}

	termW, termH, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || termW <= 0 || termH <= 0 {
		termW = 120
		termH = 35
	}

	vpW := termW - 36
	if vpW < 30 {
		vpW = 30
	}
	vpH := termH - 10
	if vpH < 10 {
		vpH = 10
	}

	m := Model{
		TextInput:      ti,
		SetupInput:     si,
		Viewport:       vp,
		IsSetupMode:    isSetup,
		Step:           initialStep,
		EventID:        profile.GetEventID(),
		EventType:      profile.EventType,
		HostNames:      profile.HostNames,
		EventDate:      profile.EventDate,
		Venue:          profile.Venue,
		Currency:       profile.DefaultCurrency,
		WelcomeMessage: profile.WelcomeMessage,
		PlannerName:    profile.PlannerName,
		PlannerRole:    profile.PlannerRole,
		EventDetails:   profile.RawDetails,
		SelectedAspect: profile.TargetAspect,
		BudgetSummary:  initialBudget,
		RSVPOverview:   initialRSVP,
		Spinner:        s,
		Loading:        false,
		StatusMsg:      "Ready",
		Config:         cfg,
		Builder:        builder,
		Logs:           initialLogs,
		SessionID:      fmt.Sprintf("session-%s", profile.GetEventID()),
		Width:          termW,
		Height:         termH,
		Ready:          true,
	}
	m.Viewport.Width = vpW
	m.Viewport.Height = vpH
	m.updateViewportContent()
	return m
}

// Init initializes Bubble Tea commands
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}
