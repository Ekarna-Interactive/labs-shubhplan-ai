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
					m.TextInput.Placeholder = "Enter style number (1-7) or type style e.g. 2 or Paper Cut"
					m.Logs = append(m.Logs, LogEntry{
						Sender: "PROFILE",
						Text:   fmt.Sprintf("📋 Loaded active event profile from event_details.md:\n\"%s\"", m.EventDetails),
					})
					m.Logs = append(m.Logs, LogEntry{
						Sender: "STEP 2/3",
						Text:   renderStyleMenu(),
					})
				} else {
					m.Step = StepEventDetails
					m.TextInput.Placeholder = "Enter Event Details e.g. Wedding for Rohan & Ananya on Dec 12 at Bengaluru"
					m.Logs = append(m.Logs, LogEntry{
						Sender: "STEP 1/3",
						Text:   "📋 Enter your Event Details (Event Type, Names, Date, Location):\ne.g. 'Wedding for Rohan & Ananya on Dec 12 at Leela Palace, Bengaluru' or 'Naming Ceremony for Aarav on Nov 5'",
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

		case "enter":
			if m.Loading {
				return m, nil
			}

			input := m.TextInput.Value()
			if strings.TrimSpace(input) == "" {
				return m, nil
			}

			m.TextInput.SetValue("")

			// Parse command
			parsed := command.Parse(input)
			resModel, resCmd := m.handleParsedCommand(parsed)
			if updatedModel, ok := resModel.(Model); ok {
				vpWidth := updatedModel.Width - 36
				if vpWidth < 30 {
					vpWidth = 30
				}
				updatedModel.Viewport.SetContent(formatLogsForViewport(updatedModel.Logs, vpWidth))
				updatedModel.Viewport.GotoBottom()
				return updatedModel, resCmd
			}
			return resModel, resCmd
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
			m.TextInput.Placeholder = "Enter 1-4 to pick a prompt, or 5 (or 'more') to generate 4 fresh options"
			m.Logs = append(m.Logs, LogEntry{Sender: "AI", Text: msg.Suggestions})
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

func (m *Model) handleParsedCommand(parsed command.ParsedInput) (tea.Model, tea.Cmd) {
	// 1. Handle Slash Commands first if input begins with /
	if strings.HasPrefix(strings.TrimSpace(parsed.RawInput), "/") {
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: parsed.RawInput})
		switch parsed.Type {
		case command.CmdReset:
			m.Step = StepEventDetails
			m.EventDetails = ""
			m.SelectedStyle = ""
			m.WelcomeMessage = ""
			m.TextInput.Placeholder = "Enter Event Details e.g. Wedding for Rohan & Ananya on Dec 12 at Bengaluru"
			m.Logs = append(m.Logs, LogEntry{
				Sender: "SYSTEM",
				Text:   "🔄 Resetting guided wizard.\n\n📋 STEP 1/3: Enter your Event Details (Event Type, Names, Date, Location):",
			})
			return m, nil

		case command.CmdStyle:
			styleChoice := parsed.EventDetails
			if styleChoice != "" {
				m.SelectedStyle = resolveStyleChoice(styleChoice)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("🎨 Selected Style: %s", m.SelectedStyle),
				})
				if m.EventDetails == "" {
					if profile, ok := config.LoadEventProfile(); ok {
						m.EventDetails = profile.RawDetails
					}
				}
				if m.EventDetails == "" {
					m.Step = StepEventDetails
					m.Logs = append(m.Logs, LogEntry{
						Sender: "STEP 1/4",
						Text:   "📋 Enter your Event Details (Event Type, Names, Date, Location):",
					})
				} else {
					m.Step = StepPromptChoice
					m.TextInput.Placeholder = "Type '1' for AI Suggestions or '2' for Custom Prompt"
					m.Logs = append(m.Logs, LogEntry{
						Sender: "STEP 4/4",
						Text:   "✨ Choose Prompt Creation Mode:\n  • Type '1' (or /suggest) to generate AI Prompt Suggestions\n  • Type '2' (or enter prompt text) for Custom Prompt",
					})
				}
			} else {
				m.Logs = append(m.Logs, LogEntry{
					Sender: "STEP 3/4",
					Text:   renderStyleMenu(),
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

			compiled := m.Builder.CompileWithAspect(promptText, parsed.WelcomeMessage, m.SelectedAspect)
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
			eventDetailsToUse := m.EventDetails

			if inputArg != "" {
				styleToUse = resolveStyleChoice(inputArg)
				m.SelectedStyle = styleToUse
			}
			if eventDetailsToUse == "" {
				eventDetailsToUse = "Auspicious Celebration Event"
			}

			return m, tea.Batch(
				m.Spinner.Tick,
				m.runSuggestionCmd(eventDetailsToUse, styleToUse),
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
				_ = config.SaveEventProfile(details, m.WelcomeMessage, m.SelectedAspect)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("✨ Updated Event Details and persisted to event_details.md:\n\"%s\"", details),
				})
			} else {
				if m.EventDetails != "" {
					m.Logs = append(m.Logs, LogEntry{
						Sender: "SYSTEM",
						Text:   fmt.Sprintf("📋 Active Event Details (saved in event_details.md):\n\"%s\"", m.EventDetails),
					})
				} else {
					m.Logs = append(m.Logs, LogEntry{
						Sender:  "WARNING",
						Text:    "No active event profile set. Type '/event <details>' to save your event details.",
						IsError: true,
					})
				}
			}
			return m, nil

		case command.CmdAspect:
			aspectArg := strings.TrimSpace(parsed.EventDetails)
			if aspectArg != "" {
				m.SelectedAspect = resolveAspectChoice(aspectArg)
				_ = config.SaveEventProfile(m.EventDetails, m.WelcomeMessage, m.SelectedAspect)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("✨ Updated Target Resolution to %s and persisted to event_details.md!", m.SelectedAspect),
				})
			} else {
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("📐 Active Target Resolution: %s\nUsage: /aspect [9:16 | 4:5 | 1:1 | 16:9]", m.SelectedAspect),
				})
			}
			return m, nil

		case command.CmdHelp:
			helpText := `Available Slash Commands:
  • /generate [event details] (Compiles & generates design)
  • /event [details] (View or update active event details in event_details.md)
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
		}
	}

	// 2. Handle Guided Wizard Step Inputs (when non-slash text is entered)
	rawText := strings.TrimSpace(parsed.RawInput)

	switch m.Step {
	case StepEventDetails:
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: rawText})
		m.EventDetails = rawText
		_ = config.SaveEventProfile(rawText, m.WelcomeMessage, m.SelectedAspect)
		m.Step = StepAspectSelection
		m.TextInput.Placeholder = "Enter resolution number (1-4) or type aspect ratio e.g. 1 or 9:16"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 2/4",
			Text:   fmt.Sprintf("Saved Event Details to event_details.md: \"%s\"\n\n%s", m.EventDetails, renderAspectMenu()),
		})
		return m, nil

	case StepAspectSelection:
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: rawText})
		m.SelectedAspect = resolveAspectChoice(rawText)
		_ = config.SaveEventProfile(m.EventDetails, m.WelcomeMessage, m.SelectedAspect)
		m.Step = StepStyleSelection
		m.TextInput.Placeholder = "Enter style number (1-7) or type style e.g. 2 or Paper Cut"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 3/4",
			Text:   fmt.Sprintf("📐 Selected Resolution: %s\n\n%s", m.SelectedAspect, renderStyleMenu()),
		})
		return m, nil

	case StepStyleSelection:
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: rawText})
		m.SelectedStyle = resolveStyleChoice(rawText)
		m.Step = StepPromptChoice
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
			return m, tea.Batch(
				m.Spinner.Tick,
				m.runSuggestionCmd(m.EventDetails, m.SelectedStyle),
			)
		} else if rawText == "2" {
			m.TextInput.Placeholder = "Enter your custom prompt details"
			m.Logs = append(m.Logs, LogEntry{
				Sender: "STEP 4/4",
				Text:   "Enter your custom prompt details below:",
			})
			return m, nil
		} else {
			promptText := fmt.Sprintf("%s style. %s. Aesthetic: %s", m.SelectedStyle, m.EventDetails, rawText)
			m.Loading = true
			m.StatusMsg = "Compiling prompt & generating invitation design..."
			compiled := m.Builder.CompileWithAspect(promptText, m.WelcomeMessage, m.SelectedAspect)
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
		if cleanChoice == "5" || cleanChoice == "more" || cleanChoice == "r" || cleanChoice == "refresh" || cleanChoice == "again" {
			m.Loading = true
			m.StatusMsg = "Generating 4 brand new AI prompt suggestions..."
			m.Logs = append(m.Logs, LogEntry{
				Sender: "SYSTEM",
				Text:   "🔄 Requesting 4 fresh prompt ideas from AI Prompter Agent...",
			})
			return m, tea.Batch(
				m.Spinner.Tick,
				m.runSuggestionCmd(m.EventDetails, m.SelectedStyle),
			)
		}

		var chosenPrompt string
		switch cleanChoice {
		case "1", "":
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

		if chosenPrompt == "" {
			chosenPrompt = fmt.Sprintf("%s in %s style", m.EventDetails, m.SelectedStyle)
		}

		m.Loading = true
		m.StatusMsg = "Compiling prompt & generating invitation design..."
		compiled := m.Builder.CompileWithAspect(chosenPrompt, m.WelcomeMessage, m.SelectedAspect)
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

func renderAspectMenu() string {
	return `📐 STEP 2 of 4: Select Target Image Resolution / Aspect Ratio:
  1. 📱 Mobile Story / Poster (9:16 Vertical)
  2. 📸 Social Feed / Portrait (4:5 Vertical)
  3. ⏹️ Square Card / Standard (1:1 Square)
  4. 💻 Desktop / Blog Banner (16:9 Landscape)
Enter resolution number (1-4) or type aspect ratio e.g. 1 or 9:16:`
}

func resolveAspectChoice(input string) string {
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

func renderStyleMenu() string {
	return `🎨 STEP 3 of 4: Select an aesthetic design style:
  1. South Indian Traditional (Imperial Gold & Royal Crimson)
  2. Paper Cut Art (Multi-layered craft paper & soft shadows)
  3. Clay 3D Render (Soft glossy pastel clay figurines)
  4. Pop Art (Vibrant retro halftone dots & bold outlines)
  5. Mughal Palace (Intricate arches & floral gold motifs)
  6. Minimalist Gold Foil (Clean pastel canvas & gold typography)
  7. Loose Watercolor (Soft pastel floral paint & fluid washes)
Enter style number (1-7) or type your custom style name:`
}

func resolveStyleChoice(input string) string {
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
	default:
		if trimmed == "" {
			return "South Indian Traditional (Imperial Gold & Royal Crimson)"
		}
		return trimmed
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

func (m Model) runSuggestionCmd(details string, style string) tea.Cmd {
	return func() tea.Msg {
		suggestions, err := generator.GenerateAIPromptSuggestions(m.Config.GeminiAPIKey, details, style)

		title := details
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
		modelName = "imagen-3.0-generate-002"
	}

	err := executeGeminiModelRequest(apiKey, modelName, prompt, aspect, outPath)
	if err != nil && (strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "NOT_FOUND")) && modelName != "imagen-3.0-generate-002" {
		// Auto-fallback to official production model if requested experimental model returns 404
		return executeGeminiModelRequest(apiKey, "imagen-3.0-generate-002", prompt, aspect, outPath)
	}
	return err
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
				"responseModalities": []string{"IMAGE", "TEXT"},
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

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
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

	return fmt.Errorf("could not parse image data from model response: %s", string(rawBody))
}
