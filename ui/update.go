package ui

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Ekarna-Interactive/ShubhPlan-CLI/command"
	"github.com/Ekarna-Interactive/ShubhPlan-CLI/config"
	"github.com/Ekarna-Interactive/ShubhPlan-CLI/generator"
	"github.com/Ekarna-Interactive/ShubhPlan-CLI/server"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
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
				if keyVal != "" {
					_ = config.SaveGeminiAPIKey(keyVal)
					m.Config.GeminiAPIKey = keyVal
					m.Logs = append(m.Logs, LogEntry{
						Sender: "SYSTEM",
						Text:   "✨ Saved GEMINI_API_KEY to local .env file! Live AI image generation enabled.",
					})
				} else {
					m.Logs = append(m.Logs, LogEntry{
						Sender: "SYSTEM",
						Text:   "ℹ Skipped API Key setup. Running in offline dry-run mode. You can set your key anytime using '/config <your-key>'.",
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
						Sender: "STEP 1/5",
						Text:   renderEventTypeMenu(m.OptionIndex),
					})
				}
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.SetupInput, cmd = m.SetupInput.Update(msg)
		return m, cmd
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		vpWidth := m.Width - 36
		if vpWidth < 30 {
			vpWidth = 30
		}
		vpHeight := m.Height - 10
		if vpHeight < 8 {
			vpHeight = 8
		}
		if !m.Ready {
			m.Viewport = viewport.New(vpWidth, vpHeight)
			m.Ready = true
		} else {
			m.Viewport.Width = vpWidth
			m.Viewport.Height = vpHeight
		}
		m.Viewport.SetContent(formatLogsForViewport(m.Logs, vpWidth))
		m.Viewport.GotoBottom()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "up", "k":
			maxOpts := getOptionCountForStep(m.Step)
			if maxOpts > 0 {
				m.OptionIndex = (m.OptionIndex - 1 + maxOpts) % maxOpts
				m.updateActiveStepMenuText()
				vpWidth := m.Width - 36
				if vpWidth < 30 {
					vpWidth = 30
				}
				m.Viewport.SetContent(formatLogsForViewport(m.Logs, vpWidth))
				m.Viewport.GotoBottom()
				return m, nil
			}

		case "down", "j":
			maxOpts := getOptionCountForStep(m.Step)
			if maxOpts > 0 {
				m.OptionIndex = (m.OptionIndex + 1) % maxOpts
				m.updateActiveStepMenuText()
				vpWidth := m.Width - 36
				if vpWidth < 30 {
					vpWidth = 30
				}
				m.Viewport.SetContent(formatLogsForViewport(m.Logs, vpWidth))
				m.Viewport.GotoBottom()
				return m, nil
			}

		case "enter":
			if m.Loading {
				return m, nil
			}

			input := m.TextInput.Value()
			if strings.TrimSpace(input) == "" {
				if m.Step != StepEventType && m.Step != StepAspectSelection && m.Step != StepStyleSelection && m.Step != StepWelcomeMessage && m.Step != StepEventDate && m.Step != StepVenue && m.Step != StepAwaitingSuggestionChoice && m.Step != StepAwaitingWelcomeChoice {
					return m, nil
				}
			}

			m.TextInput.SetValue("")

			// Parse command
			parsed := command.Parse(input)
			updatedModel, resCmd := m.handleParsedCommand(parsed)
			vpWidth := updatedModel.Width - 36
			if vpWidth < 30 {
				vpWidth = 30
			}
			updatedModel.Viewport.SetContent(formatLogsForViewport(updatedModel.Logs, vpWidth))
			updatedModel.Viewport.GotoBottom()
			return updatedModel, resCmd
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	case GenerationCompleteMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Logs = append(m.Logs, LogEntry{
				Sender:  "WARNING",
				Text:    fmt.Sprintf("%v", msg.Err),
				IsError: true,
			})
			m.StatusMsg = "API Key Warning"
		} else {
			m.StatusMsg = "Completed"
		}

		if msg.ImagePath != "" {
			m.LastImage = msg.ImagePath
			m.LastTitle = msg.Payload.DisplayTitle
			m.LastAspect = msg.Payload.Aspect

			if msg.Err == nil {
				m.Logs = append(m.Logs, LogEntry{
					Sender: "AI",
					Text:   fmt.Sprintf("Successfully generated & saved invitation asset to: %s", msg.ImagePath),
				})
			}

			// Update Web Preview Payload & Automatically Start Web Preview Browser
			webData := server.PreviewData{
				DisplayTitle:   msg.Payload.DisplayTitle,
				WelcomeMessage: msg.Payload.WelcomeMessage,
				CorePrompt:     msg.Payload.CorePrompt,
				Aspect:         msg.Payload.Aspect,
				ImagePath:      msg.ImagePath,
			}
			server.UpdatePayload(webData)
			url := server.StartServer(m.Config.Port)
			server.OpenBrowser(url)

			m.Logs = append(m.Logs, LogEntry{
				Sender: "SYSTEM",
				Text:   fmt.Sprintf("✨ Web Preview server live & opened in browser at %s\nType /generate to create another card, /style to change aesthetics, or /reset to restart setup.", url),
			})
		}
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
			m.Logs = append(m.Logs, LogEntry{Sender: "ERROR", Text: msg.Err.Error(), IsError: true})
		} else {
			m.Suggestions = msg.OptionList
			m.Step = StepAwaitingSuggestionChoice
			m.OptionIndex = 0
			m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type prompt"
			m.Logs = append(m.Logs, LogEntry{Sender: "AI", Text: renderSuggestionMenu(m.Suggestions, m.OptionIndex)})
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
			m.Logs = append(m.Logs, LogEntry{Sender: "ERROR", Text: msg.Err.Error(), IsError: true})
		} else {
			m.WelcomeSuggestions = msg.OptionList
			m.Step = StepAwaitingWelcomeChoice
			m.OptionIndex = 0
			m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type subheader"
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

func (m Model) handleParsedCommand(parsed command.ParsedInput) (Model, tea.Cmd) {
	// 1. Handle Slash Commands first if input begins with /
	if strings.HasPrefix(strings.TrimSpace(parsed.RawInput), "/") {
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: parsed.RawInput})
		switch parsed.Type {
		case command.CmdReset:
			m.Step = StepEventType
			m.OptionIndex = 0
			m.EventType = ""
			m.HostNames = ""
			m.EventDate = ""
			m.Venue = ""
			m.WelcomeMessage = ""
			m.EventDetails = ""
			m.SelectedStyle = ""
			m.Suggestions = nil
			m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type event type"
			m.Logs = append(m.Logs, LogEntry{
				Sender: "SYSTEM",
				Text:   fmt.Sprintf("🔄 Resetting guided wizard.\n\n%s", renderEventTypeMenu(m.OptionIndex)),
			})
			return m, nil

		case command.CmdStyle:
			styleChoice := parsed.EventDetails
			if styleChoice != "" {
				m.SelectedStyle = resolveStyleChoice(styleChoice, m.OptionIndex)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("🎨 Selected Style: %s", m.SelectedStyle),
				})
				if m.EventDetails == "" {
					if profile, ok := config.LoadEventProfile(); ok {
						m.EventDetails = profile.RawDetails
					}
				}
				if m.EventType == "" && m.HostNames == "" {
					m.Step = StepEventType
					m.OptionIndex = 0
					m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type event type"
					m.Logs = append(m.Logs, LogEntry{
						Sender: "STEP 1/5",
						Text:   renderEventTypeMenu(m.OptionIndex),
					})
				} else {
					m.Step = StepPromptChoice
					m.OptionIndex = 0
					m.TextInput.Placeholder = "Type '1' for AI Suggestions or '2' for Custom Prompt"
					m.Logs = append(m.Logs, LogEntry{
						Sender: "STEP 4/4",
						Text:   "✨ Choose Prompt Creation Mode:\n  • Type '1' (or /suggest) to generate AI Prompt Suggestions\n  • Type '2' (or enter prompt text) for Custom Prompt",
					})
				}
			} else {
				m.Step = StepStyleSelection
				m.OptionIndex = 0
				m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type style"
				m.Logs = append(m.Logs, LogEntry{
					Sender: "STEP 3/4",
					Text:   renderStyleMenu(m.OptionIndex),
				})
			}
			return m, nil

		case command.CmdGenerate:
			eventDetailsToUse := strings.TrimSpace(parsed.EventDetails)
			if eventDetailsToUse == "" {
				eventDetailsToUse = m.EventDetails
			}
			if eventDetailsToUse == "" {
				if profile, ok := config.LoadEventProfile(); ok {
					eventDetailsToUse = profile.RawDetails
					m.EventDetails = profile.RawDetails
				}
			}

			if eventDetailsToUse == "" {
				m.Logs = append(m.Logs, LogEntry{
					Sender:  "WARNING",
					Text:    "No active event profile set! Type '/event <your event details>' or run guided setup to save your event details.",
					IsError: true,
				})
				return m, nil
			}

			m.EventDetails = eventDetailsToUse
			m.Loading = true
			m.StatusMsg = "Compiling prompt & generating invitation design..."

			promptText := eventDetailsToUse
			if m.SelectedStyle != "" {
				promptText = fmt.Sprintf("%s style. %s", m.SelectedStyle, eventDetailsToUse)
			}

			compiled := m.Builder.CompileStructured(generator.EventData{
				EventType:      m.EventType,
				HostNames:      m.HostNames,
				EventDate:      m.EventDate,
				Venue:          m.Venue,
				WelcomeMessage: parsed.WelcomeMessage,
				VisualPrompt:   promptText,
				Aspect:         m.SelectedAspect,
			})
			m.Logs = append(m.Logs, LogEntry{
				Sender: "BUILDER",
				Text:   fmt.Sprintf("Compiled Prompt (%s):\n\"%s\"", compiled.Aspect, compiled.CorePrompt),
			})
			return m, tea.Batch(
				m.Spinner.Tick,
				m.runGenerationCmd(compiled),
			)

		case command.CmdSuggest:
			m.Loading = true
			m.StatusMsg = "Generating 4 AI theme suggestions..."
			inputArg := strings.TrimSpace(parsed.EventDetails)
			styleToUse := m.SelectedStyle

			if inputArg != "" {
				styleToUse = resolveStyleChoice(inputArg, m.OptionIndex)
				m.SelectedStyle = styleToUse
			}
			eventTypeToUse := m.EventType
			if eventTypeToUse == "" {
				eventTypeToUse = "Auspicious Event Celebration"
			}

			return m, tea.Batch(
				m.Spinner.Tick,
				m.runSuggestionCmd(eventTypeToUse, styleToUse),
			)

		case command.CmdRefine:
			if m.LastTitle == "" {
				m.Logs = append(m.Logs, LogEntry{
					Sender:  "WARNING",
					Text:    "No active design found to refine. Execute '/generate [details]' first.",
					IsError: true,
				})
				return m, nil
			}

			m.Loading = true
			m.StatusMsg = "Refining invitation design with requested changes..."
			refinedDetails := fmt.Sprintf("%s with modifications: %s", m.LastTitle, parsed.EventDetails)
			compiled := m.Builder.CompileWithAspect(refinedDetails, "", m.SelectedAspect)

			m.Logs = append(m.Logs, LogEntry{
				Sender: "BUILDER",
				Text:   fmt.Sprintf("Compiled Refined Prompt (%s):\n\"%s\"", compiled.Aspect, compiled.CorePrompt),
			})

			return m, tea.Batch(
				m.Spinner.Tick,
				m.runGenerationCmd(compiled),
			)

		case command.CmdPreview:
			url := server.StartServer(m.Config.Port)
			server.OpenBrowser(url)
			m.Logs = append(m.Logs, LogEntry{
				Sender: "SYSTEM",
				Text:   fmt.Sprintf("Launched web preview at %s", url),
			})
			return m, nil

		case command.CmdConfig:
			keyInput := strings.TrimSpace(parsed.EventDetails)
			if strings.HasPrefix(strings.ToLower(keyInput), "key=") {
				keyInput = strings.TrimSpace(strings.TrimPrefix(keyInput, "key="))
			}
			if keyInput != "" {
				_ = config.SaveGeminiAPIKey(keyInput)
				m.Config.GeminiAPIKey = keyInput
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   "✨ Updated GEMINI_API_KEY and persisted to local .env file!",
				})
			} else {
				if m.Config.GeminiAPIKey != "" {
					masked := m.Config.GeminiAPIKey
					if len(masked) > 8 {
						masked = masked[:4] + "..." + masked[len(masked)-4:]
					}
					m.Logs = append(m.Logs, LogEntry{
						Sender: "SYSTEM",
						Text:   fmt.Sprintf("🔑 Active GEMINI_API_KEY: %s", masked),
					})
				} else {
					m.Logs = append(m.Logs, LogEntry{
						Sender:  "WARNING",
						Text:    "GEMINI_API_KEY is not set. Get your key at https://aistudio.google.com/api-keys and set it using '/config <your-key>'.",
						IsError: true,
					})
				}
			}
			return m, nil

		case command.CmdEvent:
			details := strings.TrimSpace(parsed.EventDetails)
			if details != "" {
				m.EventDetails = details
				_ = config.SaveStructuredEventProfile("Event", details, "", "", m.WelcomeMessage, m.SelectedAspect)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("✨ Updated Event Profile and persisted to event_details.md:\n\"%s\"", details),
				})
			} else {
				m.Step = StepEventType
				m.OptionIndex = 0
				m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type event type"
				m.Logs = append(m.Logs, LogEntry{
					Sender: "STEP 1/5",
					Text:   renderEventTypeMenu(m.OptionIndex),
				})
			}
			return m, nil

		case command.CmdAspect:
			aspectArg := strings.TrimSpace(parsed.EventDetails)
			if aspectArg != "" {
				m.SelectedAspect = resolveAspectChoice(aspectArg, m.OptionIndex)
				_ = config.SaveEventProfile(m.EventDetails, m.WelcomeMessage, m.SelectedAspect)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("✨ Updated Target Resolution to %s and persisted to event_details.md!", m.SelectedAspect),
				})
			} else {
				m.Step = StepAspectSelection
				m.OptionIndex = 0
				m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type aspect ratio"
				m.Logs = append(m.Logs, LogEntry{
					Sender: "STEP 2/4",
					Text:   fmt.Sprintf("📐 Active Target Resolution: %s\n\n%s", m.SelectedAspect, renderAspectMenu(m.OptionIndex)),
				})
			}
			return m, nil

		case command.CmdWelcome:
			subheadText := strings.TrimSpace(parsed.EventDetails)
			if subheadText != "" && !strings.EqualFold(subheadText, "ai") && !strings.EqualFold(subheadText, "suggest") {
				m.WelcomeMessage = subheadText
				_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.WelcomeMessage, m.SelectedAspect)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("✨ Updated Welcome Subheader to: \"%s\" and persisted to event_details.md!", m.WelcomeMessage),
				})
				return m, nil
			}

			m.Loading = true
			m.StatusMsg = "Generating 4 AI welcome subheader suggestions..."
			eType := m.EventType
			if eType == "" {
				eType = "Auspicious Event Celebration"
			}
			return m, tea.Batch(
				m.Spinner.Tick,
				m.runWelcomeSuggestionCmd(eType),
			)

		case command.CmdHelp:
			helpText := `Available Slash Commands:
  • /generate [event details] (Compiles & generates design)
  • /event [details] (View or update active event details in event_details.md)
  • /welcome [text] (View, set, or generate AI welcome message subheaders)
  • /aspect [ratio] (Set resolution: 9:16 Mobile, 4:5 Social, 1:1 Square, 16:9 Desktop)
  • /style [preset/name] (Select aesthetic design style e.g. /style paper cut)
  • /suggest [theme] (Generate AI prompt suggestions)
  • /refine [changes] (Modify active design)
  • /preview (Launch local web preview browser)
  • /config [key] (View or set your Gemini API key from https://aistudio.google.com/api-keys)
  • /reset (Restart guided setup wizard)
  • /help (Show this reference guide)`
			m.Logs = append(m.Logs, LogEntry{Sender: "SYSTEM", Text: helpText})
			return m, nil

		default:
			fields := strings.Fields(parsed.RawInput)
			cmdToken := parsed.RawInput
			if len(fields) > 0 {
				cmdToken = fields[0]
			}
			m.Logs = append(m.Logs, LogEntry{
				Sender:  "WARNING",
				Text:    fmt.Sprintf("Unknown command '%s'. Type /help to view available commands, or /reset to restart setup.", cmdToken),
				IsError: true,
			})
			return m, nil
		}
	}

	// 2. Handle Guided Wizard Step Inputs (when non-slash text is entered)
	rawText := strings.TrimSpace(parsed.RawInput)

	switch m.Step {
	case StepEventType:
		resolved := resolveEventTypeChoice(rawText, m.OptionIndex)
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: resolved})
		m.EventType = resolved
		m.Step = StepHostNames
		m.TextInput.Placeholder = "e.g. Rohan & Ananya or Aarav or The Nair Family"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 2/5",
			Text:   fmt.Sprintf("Event Type: \"%s\"\n\n👥 Enter Host / Couple / Celebrant Names (e.g. 'Rohan & Ananya' or 'Aarav' or 'The Nair Family'):", m.EventType),
		})
		return m, nil

	case StepHostNames:
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: rawText})
		m.HostNames = rawText
		m.Step = StepEventDate
		m.TextInput.Placeholder = "e.g. December 12, 2026 or Dec 12"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 3/5",
			Text:   fmt.Sprintf("Host/Couple Names: \"%s\"\n\n📅 Enter Event Date (e.g. 'December 12, 2026' or 'Dec 12'):", m.HostNames),
		})
		return m, nil

	case StepEventDate:
		logTxt := rawText
		if logTxt == "" {
			logTxt = "(skipped)"
		}
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: logTxt})
		m.EventDate = rawText
		m.Step = StepVenue
		m.TextInput.Placeholder = "e.g. Leela Palace, Bengaluru"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 4/5",
			Text:   fmt.Sprintf("Event Date: \"%s\"\n\n📍 Enter Venue & Location (e.g. 'Leela Palace, Bengaluru'):", m.EventDate),
		})
		return m, nil

	case StepVenue:
		logTxt := rawText
		if logTxt == "" {
			logTxt = "(skipped)"
		}
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: logTxt})
		m.Venue = rawText
		m.Step = StepWelcomeMessage
		m.TextInput.Placeholder = "Type '1' (or 'ai') for AI Subheaders, or enter custom subheader (or Enter to skip)"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 5/5",
			Text:   fmt.Sprintf("Venue & Location: \"%s\"\n\n💌 Enter Optional Secondary Welcome Subheader:\n  • Type '1' (or 'ai') to generate 4 AI Welcome Subheader Suggestions\n  • Or enter custom subheader text below (or press Enter to skip):", m.Venue),
		})
		return m, nil

	case StepWelcomeMessage:
		logTxt := rawText
		if logTxt == "" {
			logTxt = "(skipped)"
		}
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: logTxt})

		if rawText == "1" || strings.EqualFold(rawText, "ai") || strings.EqualFold(rawText, "suggest") {
			m.Loading = true
			m.StatusMsg = "Generating 4 AI welcome subheader suggestions..."
			eType := m.EventType
			if eType == "" {
				eType = "Auspicious Event Celebration"
			}
			return m, tea.Batch(
				m.Spinner.Tick,
				m.runWelcomeSuggestionCmd(eType),
			)
		}

		m.WelcomeMessage = rawText
		parts := []string{}
		if m.EventType != "" {
			parts = append(parts, m.EventType)
		}
		if m.HostNames != "" {
			parts = append(parts, "for "+m.HostNames)
		}
		if m.EventDate != "" {
			parts = append(parts, "on "+m.EventDate)
		}
		if m.Venue != "" {
			parts = append(parts, "at "+m.Venue)
		}
		m.EventDetails = strings.Join(parts, " ")
		_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.WelcomeMessage, m.SelectedAspect)
		m.Step = StepAspectSelection
		m.OptionIndex = 0
		m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type resolution"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 2/4",
			Text:   fmt.Sprintf("✨ Saved Event Profile to event_details.md:\n  • Event Type: %s\n  • Host/Couple: %s\n  • Date: %s\n  • Venue: %s\n  • Welcome: %s\n\n%s", m.EventType, m.HostNames, m.EventDate, m.Venue, m.WelcomeMessage, renderAspectMenu(m.OptionIndex)),
		})
		return m, nil

	case StepAwaitingWelcomeChoice:
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: rawText})
		cleanChoice := strings.ToLower(strings.TrimSpace(rawText))
		if cleanChoice == "5" || cleanChoice == "more" || cleanChoice == "r" || cleanChoice == "refresh" || cleanChoice == "again" || (rawText == "" && m.OptionIndex == 4) {
			m.Loading = true
			m.StatusMsg = "Generating 4 brand new AI subheader suggestions..."
			eType := m.EventType
			if eType == "" {
				eType = "Auspicious Event Celebration"
			}
			return m, tea.Batch(
				m.Spinner.Tick,
				m.runWelcomeSuggestionCmd(eType),
			)
		}

		var chosenMsg string
		if rawText == "" && m.OptionIndex >= 0 && m.OptionIndex < len(m.WelcomeSuggestions) {
			chosenMsg = m.WelcomeSuggestions[m.OptionIndex]
		} else {
			switch cleanChoice {
			case "1":
				if len(m.WelcomeSuggestions) >= 1 {
					chosenMsg = m.WelcomeSuggestions[0]
				}
			case "2":
				if len(m.WelcomeSuggestions) >= 2 {
					chosenMsg = m.WelcomeSuggestions[1]
				}
			case "3":
				if len(m.WelcomeSuggestions) >= 3 {
					chosenMsg = m.WelcomeSuggestions[2]
				}
			case "4":
				if len(m.WelcomeSuggestions) >= 4 {
					chosenMsg = m.WelcomeSuggestions[3]
				}
			default:
				chosenMsg = rawText
			}
		}

		m.WelcomeMessage = chosenMsg
		parts := []string{}
		if m.EventType != "" {
			parts = append(parts, m.EventType)
		}
		if m.HostNames != "" {
			parts = append(parts, "for "+m.HostNames)
		}
		if m.EventDate != "" {
			parts = append(parts, "on "+m.EventDate)
		}
		if m.Venue != "" {
			parts = append(parts, "at "+m.Venue)
		}
		m.EventDetails = strings.Join(parts, " ")
		_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.WelcomeMessage, m.SelectedAspect)
		m.Step = StepAspectSelection
		m.OptionIndex = 0
		m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type resolution"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 2/4",
			Text:   fmt.Sprintf("✨ Saved Event Profile to event_details.md:\n  • Event Type: %s\n  • Host/Couple: %s\n  • Date: %s\n  • Venue: %s\n  • Welcome: %s\n\n%s", m.EventType, m.HostNames, m.EventDate, m.Venue, m.WelcomeMessage, renderAspectMenu(m.OptionIndex)),
		})
		return m, nil

	case StepAspectSelection:
		resolved := resolveAspectChoice(rawText, m.OptionIndex)
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: resolved})
		m.SelectedAspect = resolved
		_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.WelcomeMessage, m.SelectedAspect)
		m.Step = StepStyleSelection
		m.OptionIndex = 0
		m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type style"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 3/4",
			Text:   fmt.Sprintf("📐 Selected Resolution: %s\n\n%s", m.SelectedAspect, renderStyleMenu(m.OptionIndex)),
		})
		return m, nil

	case StepStyleSelection:
		resolved := resolveStyleChoice(rawText, m.OptionIndex)
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: resolved})
		m.SelectedStyle = resolved
		m.Step = StepPromptChoice
		m.OptionIndex = 0
		m.TextInput.Placeholder = "Type '1' for AI Suggestions or '2' for Custom Prompt"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 4/4",
			Text:   fmt.Sprintf("🎨 Selected Style: %s\n\n✨ Choose Prompt Creation Mode:\n  • Type '1' (or /suggest) to generate AI Prompt Suggestions\n  • Type '2' (or enter prompt text) for Custom Prompt", m.SelectedStyle),
		})
		return m, nil

	case StepPromptChoice:
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: rawText})
		if rawText == "1" || strings.EqualFold(rawText, "suggest") {
			m.Loading = true
			m.StatusMsg = "Generating 4 AI prompt suggestions..."
			eventTypeToUse := m.EventType
			if eventTypeToUse == "" {
				eventTypeToUse = "Auspicious Event Celebration"
			}
			return m, tea.Batch(
				m.Spinner.Tick,
				m.runSuggestionCmd(eventTypeToUse, m.SelectedStyle),
			)
		} else if rawText == "2" {
			m.TextInput.Placeholder = "Enter your custom prompt details"
			m.Logs = append(m.Logs, LogEntry{
				Sender: "STEP 4/4",
				Text:   "Enter your custom prompt details below:",
			})
			return m, nil
		} else {
			m.Loading = true
			m.StatusMsg = "Compiling prompt & generating invitation design..."
			compiled := m.Builder.CompileStructured(generator.EventData{
				EventType:      m.EventType,
				HostNames:      m.HostNames,
				EventDate:      m.EventDate,
				Venue:          m.Venue,
				WelcomeMessage: m.WelcomeMessage,
				VisualPrompt:   fmt.Sprintf("%s style. %s", m.SelectedStyle, rawText),
				Aspect:         m.SelectedAspect,
			})
			m.Logs = append(m.Logs, LogEntry{
				Sender: "BUILDER",
				Text:   fmt.Sprintf("Compiled Prompt (%s):\n\"%s\"", compiled.Aspect, compiled.CorePrompt),
			})
			m.Step = StepComplete
			m.TextInput.Placeholder = "Type a command e.g. /generate, /style, /aspect, /reset, or /help"
			return m, tea.Batch(
				m.Spinner.Tick,
				m.runGenerationCmd(compiled),
			)
		}

	case StepAwaitingSuggestionChoice:
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: rawText})
		cleanChoice := strings.ToLower(strings.TrimSpace(rawText))
		if cleanChoice == "5" || cleanChoice == "more" || cleanChoice == "r" || cleanChoice == "refresh" || cleanChoice == "again" || (rawText == "" && m.OptionIndex == 4) {
			m.Loading = true
			m.StatusMsg = "Generating 4 brand new AI prompt suggestions..."
			m.Logs = append(m.Logs, LogEntry{
				Sender: "SYSTEM",
				Text:   "🔄 Requesting 4 fresh prompt ideas from AI Prompter Agent...",
			})
			eventTypeToUse := m.EventType
			if eventTypeToUse == "" {
				eventTypeToUse = "Auspicious Event Celebration"
			}
			return m, tea.Batch(
				m.Spinner.Tick,
				m.runSuggestionCmd(eventTypeToUse, m.SelectedStyle),
			)
		}

		var chosenPrompt string
		if rawText == "" && m.OptionIndex >= 0 && m.OptionIndex < len(m.Suggestions) {
			chosenPrompt = m.Suggestions[m.OptionIndex]
		} else {
			switch cleanChoice {
			case "1":
				if len(m.Suggestions) >= 1 {
					chosenPrompt = m.Suggestions[0]
				}
			case "2":
				if len(m.Suggestions) >= 2 {
					chosenPrompt = m.Suggestions[1]
				}
			case "3":
				if len(m.Suggestions) >= 3 {
					chosenPrompt = m.Suggestions[2]
				}
			case "4":
				if len(m.Suggestions) >= 4 {
					chosenPrompt = m.Suggestions[3]
				}
			default:
				chosenPrompt = rawText
			}
		}

		if chosenPrompt == "" {
			chosenPrompt = fmt.Sprintf("%s in %s style", m.HostNames, m.SelectedStyle)
		}

		m.Loading = true
		m.StatusMsg = "Compiling prompt & generating invitation design..."
		compiled := m.Builder.CompileStructured(generator.EventData{
			EventType:      m.EventType,
			HostNames:      m.HostNames,
			EventDate:      m.EventDate,
			Venue:          m.Venue,
			WelcomeMessage: m.WelcomeMessage,
			VisualPrompt:   chosenPrompt,
			Aspect:         m.SelectedAspect,
		})
		m.Logs = append(m.Logs, LogEntry{
			Sender: "BUILDER",
			Text:   fmt.Sprintf("Compiled Prompt (%s):\n\"%s\"", compiled.Aspect, compiled.CorePrompt),
		})
		m.Step = StepComplete
		m.TextInput.Placeholder = "Type a command e.g. /generate, /style, /aspect, /reset, or /help"
		return m, tea.Batch(
			m.Spinner.Tick,
			m.runGenerationCmd(compiled),
		)

	default:
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: rawText})
		promptText := rawText
		if m.SelectedStyle != "" {
			promptText = fmt.Sprintf("%s style. %s", m.SelectedStyle, rawText)
		}
		m.Loading = true
		m.StatusMsg = "Compiling prompt & generating invitation design..."
		compiled := m.Builder.CompileWithAspect(promptText, m.WelcomeMessage, m.SelectedAspect)
		m.Logs = append(m.Logs, LogEntry{
			Sender: "BUILDER",
			Text:   fmt.Sprintf("Compiled Prompt (%s):\n\"%s\"", compiled.Aspect, compiled.CorePrompt),
		})
		return m, tea.Batch(
			m.Spinner.Tick,
			m.runGenerationCmd(compiled),
		)
	}
}

var predefinedEventTypes = []string{
	"💍 Wedding",
	"👶 Naming Ceremony",
	"🍼 Baby Shower",
	"🏡 Housewarming",
	"🎂 Birthday",
	"🏢 Corporate Gala",
	"✏️ Custom Event Type...",
}

var predefinedEventTypesRaw = []string{
	"Wedding",
	"Naming Ceremony",
	"Baby Shower",
	"Housewarming",
	"Birthday",
	"Corporate Gala",
}

func renderEventTypeMenu(activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("📋 STEP 1 of 5: Select Event Type:\n")
	for i, opt := range predefinedEventTypes {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type custom event type below:")
	return sb.String()
}

func resolveEventTypeChoice(input string, activeIdx int) string {
	trimmed := strings.TrimSpace(input)
	switch trimmed {
	case "1":
		return "Wedding"
	case "2":
		return "Naming Ceremony"
	case "3":
		return "Baby Shower"
	case "4":
		return "Housewarming"
	case "5":
		return "Birthday"
	case "6":
		return "Corporate Gala"
	case "":
		if activeIdx >= 0 && activeIdx < len(predefinedEventTypesRaw) {
			return predefinedEventTypesRaw[activeIdx]
		}
		return "Wedding"
	default:
		return trimmed
	}
}

var predefinedAspects = []string{
	"📱 Mobile Story / Poster (9:16 Vertical)",
	"📸 Social Feed / Portrait (4:5 Vertical)",
	"⏹️ Square Card / Standard (1:1 Square)",
	"🖥️ Desktop / Blog Banner (16:9 Landscape)",
}

var predefinedAspectsRaw = []string{
	"9:16",
	"4:5",
	"1:1",
	"16:9",
}

func renderAspectMenu(activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("📐 STEP 2 of 4: Select Target Image Resolution / Aspect Ratio:\n")
	for i, opt := range predefinedAspects {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type 1-4 / aspect ratio:")
	return sb.String()
}

func resolveAspectChoice(input string, activeIdx int) string {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	switch trimmed {
	case "1", "mobile", "poster", "story", "9:16":
		return "9:16"
	case "2", "social", "portrait", "4:5":
		return "4:5"
	case "3", "square", "card", "1:1":
		return "1:1"
	case "4", "desktop", "banner", "landscape", "16:9":
		return "16:9"
	case "":
		if activeIdx >= 0 && activeIdx < len(predefinedAspectsRaw) {
			return predefinedAspectsRaw[activeIdx]
		}
		return "9:16"
	default:
		if strings.Contains(trimmed, "4:5") {
			return "4:5"
		} else if strings.Contains(trimmed, "1:1") {
			return "1:1"
		} else if strings.Contains(trimmed, "16:9") {
			return "16:9"
		}
		return "9:16"
	}
}

var predefinedStyles = []string{
	"South Indian Traditional (Imperial Gold & Royal Crimson)",
	"Paper Cut Art (Multi-layered craft paper & soft shadows)",
	"Clay 3D Render (Soft glossy pastel clay figurines)",
	"Pop Art (Vibrant retro halftone dots & bold outlines)",
	"Mughal Palace (Intricate arches & floral gold motifs)",
	"Minimalist Gold Foil (Clean pastel canvas & gold typography)",
	"Loose Watercolor (Soft pastel floral paint & fluid washes)",
}

func renderStyleMenu(activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("🎨 STEP 3 of 4: Select an aesthetic design style:\n")
	for i, opt := range predefinedStyles {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type custom style name:")
	return sb.String()
}

func resolveStyleChoice(input string, activeIdx int) string {
	trimmed := strings.TrimSpace(input)
	switch trimmed {
	case "1":
		return "South Indian Traditional (Imperial Gold & Royal Crimson)"
	case "2":
		return "Paper Cut Art (Multi-layered craft paper)"
	case "3":
		return "Clay 3D Render (Soft glossy clay)"
	case "4":
		return "Pop Art (Vibrant retro halftone dots)"
	case "5":
		return "Mughal Palace (Intricate floral gold arches)"
	case "6":
		return "Minimalist Gold Foil (Clean pastel canvas)"
	case "7":
		return "Loose Watercolor (Soft pastel floral paint)"
	case "":
		if activeIdx >= 0 && activeIdx < len(predefinedStyles) {
			return predefinedStyles[activeIdx]
		}
		return "South Indian Traditional (Imperial Gold & Royal Crimson)"
	default:
		return trimmed
	}
}

func renderWelcomeSuggestionMenu(options []string, activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("🤖 AI Welcome Subheader Suggestions:\n")
	for i, opt := range options {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	if activeIdx == 4 {
		sb.WriteString("  ❯ [ 5. 🔄 Generate 4 More AI Subheaders ]\n")
	} else {
		sb.WriteString("    5. 🔄 Generate 4 More AI Subheaders\n")
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type custom subheader below:")
	return sb.String()
}

func renderSuggestionMenu(suggestions []string, activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("🤖 AI Prompt Suggestions:\n")
	for i, opt := range suggestions {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	if activeIdx == 4 {
		sb.WriteString("  ❯ [ 5. 🔄 Generate 4 More AI Prompt Suggestions ]\n")
	} else {
		sb.WriteString("    5. 🔄 Generate 4 More AI Prompt Suggestions\n")
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type 1-4 / custom prompt below:")
	return sb.String()
}

func getOptionCountForStep(step WizardStep) int {
	switch step {
	case StepEventType:
		return len(predefinedEventTypes)
	case StepAspectSelection:
		return len(predefinedAspects)
	case StepStyleSelection:
		return len(predefinedStyles)
	case StepAwaitingWelcomeChoice:
		return 5
	case StepAwaitingSuggestionChoice:
		return 5
	default:
		return 0
	}
}

func (m *Model) updateActiveStepMenuText() {
	if len(m.Logs) == 0 {
		return
	}
	lastIdx := len(m.Logs) - 1
	switch m.Step {
	case StepEventType:
		m.Logs[lastIdx].Text = renderEventTypeMenu(m.OptionIndex)
	case StepAspectSelection:
		m.Logs[lastIdx].Text = renderAspectMenu(m.OptionIndex)
	case StepStyleSelection:
		m.Logs[lastIdx].Text = renderStyleMenu(m.OptionIndex)
	case StepAwaitingWelcomeChoice:
		m.Logs[lastIdx].Text = renderWelcomeSuggestionMenu(m.WelcomeSuggestions, m.OptionIndex)
	case StepAwaitingSuggestionChoice:
		m.Logs[lastIdx].Text = renderSuggestionMenu(m.Suggestions, m.OptionIndex)
	}
}

func (m Model) runWelcomeSuggestionCmd(eventType string) tea.Cmd {
	return func() tea.Msg {
		options, err := generator.GenerateAIWelcomeSuggestions(m.Config.GeminiAPIKey, eventType)
		header := fmt.Sprintf("🤖 Live Gemini AI Welcome Subheader Suggestions for '%s':", eventType)
		if err != nil {
			header = fmt.Sprintf("⚠️ Fallback Welcome Subheaders (%v) for '%s':", err, eventType)
		}

		formattedText := fmt.Sprintf("%s\n\n"+
			"1. %s\n\n"+
			"2. %s\n\n"+
			"3. %s\n\n"+
			"4. %s\n\n"+
			"5. 🔄 Generate 4 More AI Subheader Suggestions\n\n"+
			"Press ↑/↓ to navigate, Enter to select, or type 1-4:",
			header, options[0], options[1], options[2], options[3])

		return WelcomeSuggestionCompleteMsg{
			Suggestions: formattedText,
			OptionList:  options,
		}
	}
}

func (m Model) runGenerationCmd(payload generator.ResponsePayload) tea.Cmd {
	return func() tea.Msg {
		filename := fmt.Sprintf("shubh_design_%d.png", time.Now().Unix())
		outPath := filepath.Join(m.Config.OutputDir, filename)

		// Check if Gemini API Key is missing
		if m.Config.GeminiAPIKey == "" {
			err := createPlaceholderImage(outPath, payload.DisplayTitle, payload.WelcomeMessage)
			if err != nil {
				return GenerationCompleteMsg{Err: fmt.Errorf("failed to write placeholder asset: %w", err)}
			}
			return GenerationCompleteMsg{
				Payload:   payload,
				ImagePath: outPath,
				Err:       fmt.Errorf("GEMINI_API_KEY is not configured! Generated offline placeholder. Set your key using '/config <your-key>' (Get free key at https://aistudio.google.com/api-keys)"),
			}
		}

		// Try generating image via Gemini API
		err := generateGeminiImage(m.Config.GeminiAPIKey, m.Config.ImageModel, payload.CorePrompt, payload.Aspect, outPath)
		if err != nil {
			_ = createPlaceholderImage(outPath, payload.DisplayTitle, payload.WelcomeMessage)
			return GenerationCompleteMsg{
				Payload:   payload,
				ImagePath: outPath,
				Err:       fmt.Errorf("Gemini API Error: %v. Update your key using '/config <your-key>' (Get key at https://aistudio.google.com/api-keys)", err),
			}
		}

		return GenerationCompleteMsg{
			Payload:   payload,
			ImagePath: outPath,
		}
	}
}

func (m Model) runSuggestionCmd(eventType string, style string) tea.Cmd {
	return func() tea.Msg {
		suggestions, err := generator.GenerateAIPromptSuggestions(m.Config.GeminiAPIKey, eventType, style)

		title := eventType
		if title == "" {
			title = "Event Celebration"
		}
		if style == "" {
			style = "South Indian Traditional"
		}

		header := fmt.Sprintf("🤖 Live Gemini AI Prompter Agent Suggestions for '%s' (%s):", title, style)
		if err != nil {
			header = fmt.Sprintf("⚠️ Fallback Suggestions (%v) for '%s' (%s):", err, title, style)
		}

		formattedText := fmt.Sprintf("%s\n\n"+
			"1. %s\n\n"+
			"2. %s\n\n"+
			"3. %s\n\n"+
			"4. %s\n\n"+
			"5. 🔄 Generate 4 More AI Prompt Suggestions\n\n"+
			"Enter 1-4 to select a prompt, or 5 (or 'more') to generate 4 brand-new suggestions:",
			header, suggestions[0], suggestions[1], suggestions[2], suggestions[3])

		return SuggestionCompleteMsg{
			Suggestions: formattedText,
			OptionList:  suggestions,
		}
	}
}

func createPlaceholderImage(outPath string, title string, welcome string) error {
	_ = title
	_ = welcome
	img := image.NewRGBA(image.Rect(0, 0, 800, 800))
	bgColor := color.RGBA{13, 15, 23, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Inner border
	goldColor := color.RGBA{212, 175, 55, 255}
	for x := 40; x < 760; x++ {
		img.Set(x, 40, goldColor)
		img.Set(x, 41, goldColor)
		img.Set(x, 758, goldColor)
		img.Set(x, 759, goldColor)
	}
	for y := 40; y < 760; y++ {
		img.Set(40, y, goldColor)
		img.Set(41, y, goldColor)
		img.Set(758, y, goldColor)
		img.Set(759, y, goldColor)
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return png.Encode(outFile, img)
}

func generateGeminiImage(apiKey string, modelName string, prompt string, aspect string, outPath string) error {
	if modelName == "" {
		return fmt.Errorf("model '%s' is not supported", modelName)
	}

	return executeGeminiModelRequest(apiKey, modelName, prompt, aspect, outPath)
}

func executeGeminiModelRequest(apiKey string, modelName string, prompt string, aspect string, outPath string) error {
	var url string
	var bodyBytes []byte
	var err error

	if aspect == "" {
		aspect = "9:16"
	}

	if strings.HasPrefix(modelName, "gemini-") {
		url = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
		payloadMap := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{"text": "Generate an invitation card image illustration for: " + prompt},
					},
				},
			},
			"generationConfig": map[string]interface{}{
				"responseModalities": []string{"IMAGE"},
			},
		}
		bodyBytes, err = json.Marshal(payloadMap)
	} else {
		url = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:predict?key=%s", modelName, apiKey)
		payloadMap := map[string]interface{}{
			"instances": []map[string]interface{}{
				{"prompt": prompt},
			},
			"parameters": map[string]interface{}{
				"sampleCount": 1,
				"aspectRatio": aspect,
				"outputOptions": map[string]string{
					"mimeType": "image/png",
				},
			},
		}
		bodyBytes, err = json.Marshal(payloadMap)
	}

	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("model '%s' is not supported", modelName)
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Try decoding Imagen predictions format first
	var predictRes struct {
		Predictions []struct {
			BytesBase64Encoded string `json:"bytesBase64Encoded"`
		} `json:"predictions"`
	}
	if err := json.Unmarshal(rawBody, &predictRes); err == nil && len(predictRes.Predictions) > 0 && predictRes.Predictions[0].BytesBase64Encoded != "" {
		imgData, err := base64.StdEncoding.DecodeString(predictRes.Predictions[0].BytesBase64Encoded)
		if err == nil {
			return os.WriteFile(outPath, imgData, 0644)
		}
	}

	// Try decoding Gemini generateContent inlineData format (camelCase & snake_case)
	var contentRes struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData"`
					InlineDataSnake struct {
						MimeType string `json:"mime_type"`
						Data     string `json:"data"`
					} `json:"inline_data"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(rawBody, &contentRes); err == nil && len(contentRes.Candidates) > 0 {
		for _, cand := range contentRes.Candidates {
			for _, part := range cand.Content.Parts {
				b64Str := part.InlineData.Data
				if b64Str == "" {
					b64Str = part.InlineDataSnake.Data
				}
				if b64Str != "" {
					imgData, err := base64.StdEncoding.DecodeString(b64Str)
					if err == nil {
						return os.WriteFile(outPath, imgData, 0644)
					}
				}
			}
		}
	}

	return fmt.Errorf("model '%s' is not supported", modelName)
}
