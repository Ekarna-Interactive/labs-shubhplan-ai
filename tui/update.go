package tui

import (
	"fmt"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/client"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/command"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/generator"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/web"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Msg structs for async operations
type GenerationCompleteMsg struct {
	Payload   generator.ResponsePayload
	ImagePath string
	Err       error
}

type SuggestionCompleteMsg struct {
	Suggestions string
	OptionList  []string
	Err         error
}

type WelcomeSuggestionCompleteMsg struct {
	Suggestions string
	OptionList  []string
	Err         error
}

// Update handles Bubble Tea event messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.IsSetupMode {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit

			case "enter":
				keyVal := strings.TrimSpace(m.SetupInput.Value())
				switch m.SetupStep {
				case 0:
					if keyVal != "" {
						_ = config.SaveGeminiAPIKey(keyVal)
						m.Config.GeminiAPIKey = keyVal
						m.Logs = append(m.Logs, LogEntry{
							Sender: "SYSTEM",
							Text:   "✨ Saved GEMINI_API_KEY to local .env file! Live AI generation enabled.",
						})
					} else {
						m.Logs = append(m.Logs, LogEntry{
							Sender: "SYSTEM",
							Text:   "ℹ Skipped Gemini API Key setup. Running in offline dry-run mode.",
						})
					}

					// Step 2: Check Honcho API Key
					m.SetupStep = 1
					m.SetupInput.Reset()
					m.SetupInput.Placeholder = "Paste Honcho API Key here (or press Enter to use Inbuilt Local Memory Store)"
					m.Logs = append(m.Logs, LogEntry{
						Sender: "SETUP",
						Text:   "🧠 OPTIONAL SETUP: HONCHO_API_KEY check.\nGet key at https://honcho.dev\nEnter key below for Honcho Cloud Memory sync, or press Enter to use the Inbuilt Local Memory Store (./data/honcho_memory.json).",
					})
					return m, nil

				case 1:
					if keyVal != "" {
						_ = config.SaveHonchoAPIKey(keyVal)
						client.GetHonchoManager().SetAPIKey(keyVal)
						m.Logs = append(m.Logs, LogEntry{
							Sender: "SYSTEM",
							Text:   "✨ Saved HONCHO_API_KEY to local .env file! Honcho Cloud Memory sync active.",
						})
					} else {
						m.Logs = append(m.Logs, LogEntry{
							Sender: "SYSTEM",
							Text:   "🟡 HONCHO_API_KEY omitted. Operating with Inbuilt Local Memory Store (./data/honcho_memory.json).",
						})
					}
					m.IsSetupMode = false
					m.TextInput.Focus()
					if profile, ok := config.LoadEventProfile(); ok {
						m.EventDetails = profile.RawDetails
						m.WelcomeMessage = profile.WelcomeMessage
						m.Step = StepStyleSelection
						m.OptionIndex = 0
						m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type style"
						m.Logs = append(m.Logs, LogEntry{
							Sender: "PROFILE",
							Text:   fmt.Sprintf("📋 Loaded active event profile from event_details.md:\n\"%s\"", m.EventDetails),
						})
						m.Logs = append(m.Logs, LogEntry{
							Sender: "STEP 2/3",
							Text:   renderStyleMenu(m.OptionIndex),
						})
					} else {
						m.Step = StepEventType
						m.OptionIndex = 0
						m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type event type"
						m.Logs = append(m.Logs, LogEntry{
							Sender: "STEP 1/3",
							Text:   renderEventTypeMenu(m.OptionIndex),
						})
					}
					return m, nil
				}
			}
		}

		var cmd tea.Cmd
		m.SetupInput, cmd = m.SetupInput.Update(msg)
		return m, cmd
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "tab":
			m.ActiveTab = (m.ActiveTab + 1) % 5
			m.updateViewportContent()
			return m, nil

		case "shift+tab":
			m.ActiveTab = (m.ActiveTab - 1 + 5) % 5
			m.updateViewportContent()
			return m, nil

		case "alt+1", "f1":
			m.ActiveTab = TabAgentChat
			m.updateViewportContent()
			return m, nil

		case "alt+2", "f2":
			m.ActiveTab = TabItinerary
			m.updateViewportContent()
			return m, nil

		case "alt+3", "f3":
			m.ActiveTab = TabBudget
			m.updateViewportContent()
			return m, nil

		case "alt+4", "f4":
			m.ActiveTab = TabRSVP
			m.updateViewportContent()
			return m, nil

		case "alt+5", "f5":
			m.ActiveTab = TabDesignStudio
			m.updateViewportContent()
			return m, nil

		case "up":
			maxOpts := getOptionCountForStep(m.Step)
			if maxOpts > 0 {
				m.OptionIndex = (m.OptionIndex - 1 + maxOpts) % maxOpts
				m.updateActiveStepMenuText()
				m.updateViewportContent()
				m.Viewport.GotoBottom()
				return m, nil
			}
			m.Viewport.LineUp(1)
			return m, nil

		case "down":
			maxOpts := getOptionCountForStep(m.Step)
			if maxOpts > 0 {
				m.OptionIndex = (m.OptionIndex + 1) % maxOpts
				m.updateActiveStepMenuText()
				m.updateViewportContent()
				m.Viewport.GotoBottom()
				return m, nil
			}
			m.Viewport.LineDown(1)
			return m, nil

		case "shift+up", "ctrl+u":
			m.Viewport.LineUp(2)
			return m, nil

		case "shift+down", "ctrl+d":
			m.Viewport.LineDown(2)
			return m, nil

		case "pgup":
			m.Viewport.ViewUp()
			return m, nil

		case "pgdown":
			m.Viewport.ViewDown()
			return m, nil

		case "enter":
			input := strings.TrimSpace(m.TextInput.Value())
			if m.RSVPStep != RSVPStepInactive {
				m.TextInput.SetValue("")
				resModel, cmd := m.handleRSVPWizardInput(input)
				resModel.updateViewportContent()
				return resModel, cmd
			}
			if input == "" {
				if m.ActiveTab == TabAgentChat && m.Step != StepComplete {
					input = ""
				} else {
					return m, nil
				}
			}

			m.TextInput.SetValue("")
			parsed := command.Parse(input)
			resModel, cmd := m.handleParsedCommand(parsed)
			resModel.updateViewportContent()
			return resModel, cmd
		}

	case tea.WindowSizeMsg:
		if msg.Width <= 0 || msg.Height <= 0 {
			return m, nil
		}
		m.Width = msg.Width
		m.Height = msg.Height
		vpWidth := msg.Width - 36
		if vpWidth < 30 {
			vpWidth = 30
		}
		headerLines := 3
		if msg.Height < 28 {
			headerLines = 1
		}
		vpHeight := msg.Height - (headerLines + 11)
		if vpHeight < 1 {
			vpHeight = 1
		}
		m.Viewport.Width = vpWidth
		m.Viewport.Height = vpHeight
		m.updateViewportContent()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	case AgentStreamMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Logs = append(m.Logs, LogEntry{
				Sender:  "AGENT_ERR",
				Text:    fmt.Sprintf("❌ Agent stream error: %v", msg.Err),
				IsError: true,
			})
		} else {
			var fullResponse strings.Builder
			for _, ev := range msg.Events {
				if m.ShowVerbose {
					if ev.Agent != "" || ev.Type != "" {
						m.Logs = append(m.Logs, LogEntry{
							Sender: "ROUTER",
							Text:   fmt.Sprintf("[%s -> %s] %s", ev.Agent, ev.Type, ev.Content),
						})
					}
				}
				if ev.Type == "content" || ev.Type == "done" || ev.Content != "" {
					fullResponse.WriteString(ev.Content)
				}
			}

			ansText := strings.TrimSpace(fullResponse.String())
			if ansText != "" {
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SHUBH_AI",
					Text:   ansText,
				})

				if bs, guests, ok := parseBudgetFromAgentResponse(ansText, m.Currency); ok {
					if bs.TotalEstimated > 0 {
						m.BudgetSummary = bs
					}
					if guests > 0 {
						m.RSVPOverview.TotalGuests = guests
					}
					_ = config.SaveStructuredEventProfileWithBudget(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole, m.BudgetSummary.TotalEstimated, m.RSVPOverview.TotalGuests)
				}
			}
		}
		m.StatusMsg = "Ready"
		vpWidth := m.Width - 36
		if vpWidth < 30 {
			vpWidth = 30
		}
		m.Viewport.SetContent(formatLogsForViewport(m.Logs, vpWidth))
		m.Viewport.GotoBottom()
		return m, nil

	case GenerationCompleteMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Logs = append(m.Logs, LogEntry{
				Sender:  "ERROR",
				Text:    fmt.Sprintf("❌ Design generation failed: %v", msg.Err),
				IsError: true,
			})
		} else {
			m.LastTitle = msg.Payload.DisplayTitle
			m.LastImage = msg.ImagePath
			m.Logs = append(m.Logs, LogEntry{
				Sender: "GENERATOR",
				Text: fmt.Sprintf("🎨 Design Created Successfully!\n  • Theme Title: %s\n  • Resolution Aspect: %s\n  • Output Image Path: %s\n  • Welcome Subheader: \"%s\"",
					msg.Payload.DisplayTitle, msg.Payload.Aspect, msg.ImagePath, msg.Payload.WelcomeMessage),
			})

			url := web.StartServer(m.Config.Port)
			web.OpenBrowser(url)
			m.Logs = append(m.Logs, LogEntry{
				Sender: "SYSTEM",
				Text:   fmt.Sprintf("✨ Real-time preview server running at %s?sessionID=%s", url, m.SessionID),
			})
		}
		m.StatusMsg = "Ready"
		vpWidth := m.Width - 36
		if vpWidth < 30 {
			vpWidth = 30
		}
		m.Viewport.SetContent(formatLogsForViewport(m.Logs, vpWidth))
		m.Viewport.GotoBottom()
		return m, nil

	case SuggestionCompleteMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Logs = append(m.Logs, LogEntry{
				Sender:  "ERROR",
				Text:    fmt.Sprintf("❌ Suggestion generation failed: %v", msg.Err),
				IsError: true,
			})
		} else {
			m.Suggestions = msg.OptionList
			m.Step = StepAwaitingSuggestionChoice
			m.OptionIndex = 0
			m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select option 1-5, or type prompt below"
			m.Logs = append(m.Logs, LogEntry{Sender: "PROMPTER", Text: msg.Suggestions})
		}
		m.StatusMsg = "Ready"
		vpWidth := m.Width - 36
		if vpWidth < 30 {
			vpWidth = 30
		}
		m.Viewport.SetContent(formatLogsForViewport(m.Logs, vpWidth))
		m.Viewport.GotoBottom()
		return m, nil

	case WelcomeSuggestionCompleteMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Logs = append(m.Logs, LogEntry{
				Sender:  "ERROR",
				Text:    fmt.Sprintf("❌ Welcome suggestion generation failed: %v", msg.Err),
				IsError: true,
			})
		} else {
			m.WelcomeSuggestions = msg.OptionList
			m.Step = StepAwaitingWelcomeChoice
			m.OptionIndex = 0
			m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select option 1-5, or type subheader below"
			m.Logs = append(m.Logs, LogEntry{Sender: "AI", Text: renderWelcomeSuggestionMenu(m.WelcomeSuggestions, m.OptionIndex)})
		}
		m.StatusMsg = "Ready"
		vpWidth := m.Width - 36
		if vpWidth < 30 {
			vpWidth = 30
		}
		m.Viewport.SetContent(formatLogsForViewport(m.Logs, vpWidth))
		m.Viewport.GotoBottom()
		return m, nil
	}

	var vpCmd tea.Cmd
	m.Viewport, vpCmd = m.Viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	var tiCmd tea.Cmd
	m.TextInput, tiCmd = m.TextInput.Update(msg)
	cmds = append(cmds, tiCmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) updateViewportContent() {
	vpWidth := m.Width - 36
	if vpWidth < 30 {
		vpWidth = 30
	}
	switch m.ActiveTab {
	case TabAgentChat:
		m.Viewport.SetContent(formatLogsForViewport(m.Logs, vpWidth))
		m.Viewport.GotoBottom()
	case TabItinerary:
		if len(m.ItineraryItems) == 0 && m.EventID != "" {
			ec := client.NewEventClient()
			if items, err := ec.FetchItinerary(m.EventID); err == nil {
				m.ItineraryItems = items
			}
		}
		m.Viewport.SetContent(formatItineraryView(m.ItineraryItems, vpWidth))
	case TabBudget:
		if len(m.BudgetSummary.Categories) == 0 && m.BudgetSummary.TotalEstimated == 0 && m.EventID != "" {
			ec := client.NewEventClient()
			if bs, err := ec.FetchBudgetSummary(m.EventID); err == nil {
				m.BudgetSummary = bs
			}
		}
		m.Viewport.SetContent(formatBudgetView(m.BudgetSummary, m.Currency, m.EventType, m.RSVPOverview.TotalGuests, vpWidth))
	case TabRSVP:
		if m.EventID != "" {
			ec := client.NewEventClient()
			if rsvp, err := ec.FetchRSVPOverview(m.EventID); err == nil && rsvp.TotalGuests > 0 {
				if m.RSVPOverview.TotalGuests > rsvp.TotalGuests {
					rsvp.TotalGuests = m.RSVPOverview.TotalGuests
				}
				if rsvp.Attending >= m.RSVPOverview.Attending {
					m.RSVPOverview.Attending = rsvp.Attending
				}
				if rsvp.Declined >= m.RSVPOverview.Declined {
					m.RSVPOverview.Declined = rsvp.Declined
				}
				m.RSVPOverview.TotalGuests = rsvp.TotalGuests
				m.RSVPOverview.Pending = m.RSVPOverview.TotalGuests - (m.RSVPOverview.Attending + m.RSVPOverview.Declined)
				if m.RSVPOverview.Pending < 0 {
					m.RSVPOverview.Pending = 0
				}
				if len(rsvp.DietaryReqs) > 0 {
					m.RSVPOverview.DietaryReqs = rsvp.DietaryReqs
				}
			} else if m.RSVPOverview.TotalGuests > 0 {
				m.RSVPOverview.Pending = m.RSVPOverview.TotalGuests - (m.RSVPOverview.Attending + m.RSVPOverview.Declined)
				if m.RSVPOverview.Pending < 0 {
					m.RSVPOverview.Pending = 0
				}
			}
			if cards, err := ec.FetchHonchoCards(m.EventID); err == nil && len(cards) > 0 {
				m.PeerCards = cards
			}
		}
		m.Viewport.SetContent(formatRSVPView(m.RSVPOverview, m.PeerCards, vpWidth))
	case TabDesignStudio:
		m.Viewport.SetContent(formatLogsForViewport(m.Logs, vpWidth))
		m.Viewport.GotoBottom()
	}
}
