package tui

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/client"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/command"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/generator"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/web"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleParsedCommand(parsed command.ParsedInput) (Model, tea.Cmd) {
	if parsed.Type == command.CmdClear {
		m.Logs = nil
		m.Viewport.SetContent("")
		m.Logs = append(m.Logs, LogEntry{
			Sender: "SYSTEM",
			Text:   "🧹 Cleared terminal activity log.",
		})
		return m, nil
	}

	// 1. Handle Slash Commands first if input begins with /
	if strings.HasPrefix(strings.TrimSpace(parsed.RawInput), "/") {
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: parsed.RawInput})
		switch parsed.Type {
		case command.CmdClear:
			m.Logs = nil
			m.Viewport.SetContent("")
			m.Logs = append(m.Logs, LogEntry{
				Sender: "SYSTEM",
				Text:   "🧹 Cleared terminal activity log.",
			})
			return m, nil
		case command.CmdReset, command.CmdWizard:
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
				Text:   fmt.Sprintf("🎨 Starting guided card design wizard.\n\n%s", renderEventTypeMenu(m.OptionIndex)),
			})
			return m, nil

		case command.CmdTimeline:
			m.ActiveTab = TabItinerary
			m.updateViewportContent()
			return m, nil

		case command.CmdRSVP:
			m.ActiveTab = TabRSVP
			m.updateViewportContent()
			return m, nil

		case command.CmdAddRSVP:
			m.ActiveTab = TabAgentChat
			m.RSVPStep = RSVPStepGuestName
			m.RSVPData = RSVPWizardData{}
			m.TextInput.SetValue("")
			m.TextInput.Placeholder = "Type guest full name and press Enter..."
			m.Logs = append(m.Logs, LogEntry{
				Sender: "WIZARD",
				Text:   "📋 [RSVP WIZARD] Add New Guest RSVP\nStep 1 of 6 [Mandatory]: Enter Guest Full Name (e.g. 'Rohan Kumar')",
			})
			m.updateViewportContent()
			return m, nil

		case command.CmdHoncho:
			m.ActiveTab = TabRSVP
			if !config.HasHonchoAPIKey() {
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   "🔑 Operating in Local Memory Mode (./data/honcho_memory.json). Configure HONCHO_API_KEY via '/config' or .env to enable live Cloud Sync at api.honcho.dev/v3.",
				})
			}
			m.updateViewportContent()
			return m, nil

		case command.CmdExport:
			outPath := filepath.Join(m.Config.OutputDir, "event_run_of_show.md")
			_ = os.MkdirAll(m.Config.OutputDir, 0755)
			_ = os.WriteFile(outPath, []byte("# Event Run-of-Show & Budget Summary\n\nExported from Shubh Plan AI."), 0644)
			m.Logs = append(m.Logs, LogEntry{Sender: "SYSTEM", Text: fmt.Sprintf("📄 Exported event run-of-show to %s", outPath)})
			m.updateViewportContent()
			return m, nil

		case command.CmdPlanner:
			arg := strings.TrimSpace(parsed.EventDetails)
			if arg != "" {
				parts := strings.SplitN(arg, "|", 2)
				name := strings.TrimSpace(parts[0])
				role := "Lead Event Planner"
				if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
					role = strings.TrimSpace(parts[1])
				}
				m.PlannerName = name
				m.PlannerRole = role
				_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("👤 Updated Event Planner Profile to: %s (%s) and saved to event_details.md!", m.PlannerName, m.PlannerRole),
				})
			} else {
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("👤 Active Event Planner: %s (%s)\nTo update, type: '/planner <Name> | <Role>' (e.g. '/planner Gokul | Senior Coordinator').", m.PlannerName, m.PlannerRole),
				})
			}
			m.updateViewportContent()
			return m, nil

		case command.CmdVerbose:
			m.ShowVerbose = !m.ShowVerbose
			statusStr := "OFF (Clean Mode: internal orchestrator routing steps hidden)"
			if m.ShowVerbose {
				statusStr = "ON (Debug Mode: showing internal orchestrator routing steps)"
			}
			m.Logs = append(m.Logs, LogEntry{
				Sender: "SYSTEM",
				Text:   fmt.Sprintf("🔍 Toggled Orchestrator Verbose Logging to: %s\nType '/verbose' again to toggle back.", statusStr),
			})
			m.updateViewportContent()
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
			if !config.HasGeminiAPIKey() {
				m.Logs = append(m.Logs, LogEntry{
					Sender:  "WARNING",
					Text:    "⚠️ GEMINI_API_KEY is not configured! Get your free key at https://aistudio.google.com/api-keys and run '/config <your_key>' to enable live AI generation.",
					IsError: true,
				})
				return m, nil
			}

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
			if !config.HasGeminiAPIKey() {
				m.Logs = append(m.Logs, LogEntry{
					Sender:  "WARNING",
					Text:    "⚠️ GEMINI_API_KEY is not configured! Get your free key at https://aistudio.google.com/api-keys and run '/config <your_key>' to enable live AI generation.",
					IsError: true,
				})
				return m, nil
			}

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
			if !config.HasGeminiAPIKey() {
				m.Logs = append(m.Logs, LogEntry{
					Sender:  "WARNING",
					Text:    "⚠️ GEMINI_API_KEY is not configured! Get your free key at https://aistudio.google.com/api-keys and run '/config <your_key>' to enable live AI generation.",
					IsError: true,
				})
				return m, nil
			}

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
			url := web.StartServer(m.Config.Port)
			web.OpenBrowser(url)
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
			subCmd := strings.ToLower(strings.TrimSpace(parsed.EventDetails))
			if subCmd == "new" || subCmd == "create" || subCmd == "setup" || subCmd == "wizard" {
				m.Step = StepEventType
				m.OptionIndex = 0
				m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type event type"
				m.Logs = append(m.Logs, LogEntry{
					Sender: "STEP 1/5",
					Text:   renderEventTypeMenu(m.OptionIndex),
				})
				return m, nil
			}

			if subCmd == "update" || subCmd == "edit" {
				m.Step = StepEventType
				m.OptionIndex = 0
				m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type event type"
				m.Logs = append(m.Logs, LogEntry{
					Sender: "EDIT PROFILE",
					Text:   renderEventTypeMenu(m.OptionIndex),
				})
				return m, nil
			}

			if subCmd != "" && subCmd != "show" && subCmd != "info" {
				m.EventDetails = parsed.EventDetails
				_ = config.SaveStructuredEventProfile("Event", parsed.EventDetails, "", "", m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("✨ Updated Event Profile and persisted to event_details.md:\n\"%s\"", parsed.EventDetails),
				})
				return m, nil
			}

			// /event without subcommands -> display current active event details
			profile, _ := config.LoadEventProfile()
			m.Logs = append(m.Logs, LogEntry{
				Sender: "PROFILE",
				Text:   renderEventProfileTUI(profile),
			})
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

		case command.CmdCurrency:
			currArg := strings.TrimSpace(parsed.EventDetails)
			if currArg != "" {
				m.Currency = resolveCurrencyChoice(currArg, 0)
				_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   fmt.Sprintf("✨ Updated Default Event Currency to %s (%s) and persisted to event_details.md!", m.Currency, GetCurrencySymbol(m.Currency)),
				})
			} else {
				m.Step = StepCurrency
				m.OptionIndex = 0
				m.TextInput.Placeholder = "Use ↑/↓ arrow keys or type currency code (e.g. USD, EUR, GBP, INR, AUD, SGD)"
				m.Logs = append(m.Logs, LogEntry{
					Sender: "CURRENCY",
					Text:   renderCurrencyMenu(m.OptionIndex),
				})
			}
			return m, nil

		case command.CmdWelcome:
			subheadText := strings.TrimSpace(parsed.EventDetails)
			if subheadText != "" && !strings.EqualFold(subheadText, "ai") && !strings.EqualFold(subheadText, "suggest") {
				m.WelcomeMessage = subheadText
				_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole)
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

		case command.CmdBudget:
			args := parsed.Args
			details := strings.TrimSpace(parsed.EventDetails)

			if strings.EqualFold(details, "accept") || strings.EqualFold(details, "auto") || strings.EqualFold(details, "suggest") || strings.EqualFold(details, "yes") {
				sugAmt, sugGuests, _ := GetSuggestedBudgetForEvent(m.EventType, m.Currency)
				m.BudgetSummary = client.BudgetSummary{
					TotalEstimated: sugAmt,
					TotalActual:    0,
					Categories: []client.BudgetCategory{
						{Name: "Venue Rental (20%)", Estimated: sugAmt * 0.20, Actual: 0},
						{Name: "Food & Beverage (35%)", Estimated: sugAmt * 0.35, Actual: 0},
						{Name: "Decor & Styling (20%)", Estimated: sugAmt * 0.20, Actual: 0},
						{Name: "Sound & Entertainment (10%)", Estimated: sugAmt * 0.10, Actual: 0},
						{Name: "Logistics & Misc (5%)", Estimated: sugAmt * 0.05, Actual: 0},
						{Name: "Contingency Buffer (10%)", Estimated: sugAmt * 0.10, Actual: 0},
					},
				}
				m.RSVPOverview.TotalGuests = sugGuests
				m.ActiveTab = TabBudget
				m.updateViewportContent()

				_ = config.SaveStructuredEventProfileWithBudget(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole, m.BudgetSummary.TotalEstimated, m.RSVPOverview.TotalGuests)

				sym := GetCurrencySymbol(m.Currency)
				cpg := sugAmt / float64(sugGuests)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "BudgetAgent",
					Text:   fmt.Sprintf("✨ Applied suggested baseline budget of %s%.2f for %s (%d guests, ~%s%.2f/guest)!", sym, sugAmt, m.EventType, sugGuests, sym, cpg),
				})
				return m, nil
			}

			if len(args) == 0 && details == "" {
				m.ActiveTab = TabBudget
				m.updateViewportContent()
				m.Logs = append(m.Logs, LogEntry{
					Sender: "SYSTEM",
					Text:   "💰 Switched to Tab 3: Budget & Spend Analysis Dashboard.",
				})
				return m, nil
			}

			amountStr := ""
			guestsStr := ""
			if len(args) >= 1 {
				amountStr = args[0]
			}
			if len(args) >= 2 {
				guestsStr = args[1]
			}
			if amountStr == "" && details != "" {
				fields := strings.Fields(details)
				if len(fields) >= 1 {
					amountStr = fields[0]
				}
				if len(fields) >= 2 {
					guestsStr = fields[1]
				}
			}

			targetBudget := parseBudgetAmount(amountStr, m.Currency)
			guestCount := parseGuestCount(guestsStr)

			if targetBudget > 0 {
				m.BudgetSummary = client.BudgetSummary{
					TotalEstimated: targetBudget,
					TotalActual:    0,
					Categories: []client.BudgetCategory{
						{Name: "Venue Rental (20%)", Estimated: targetBudget * 0.20, Actual: 0},
						{Name: "Food & Beverage (35%)", Estimated: targetBudget * 0.35, Actual: 0},
						{Name: "Decor & Styling (20%)", Estimated: targetBudget * 0.20, Actual: 0},
						{Name: "Sound & Entertainment (10%)", Estimated: targetBudget * 0.10, Actual: 0},
						{Name: "Logistics & Misc (5%)", Estimated: targetBudget * 0.05, Actual: 0},
						{Name: "Contingency Buffer (10%)", Estimated: targetBudget * 0.10, Actual: 0},
					},
				}
			}

			if guestCount > 0 {
				m.RSVPOverview.TotalGuests = guestCount
			}

			if targetBudget > 0 || guestCount > 0 {
				_ = config.SaveStructuredEventProfileWithBudget(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole, m.BudgetSummary.TotalEstimated, m.RSVPOverview.TotalGuests)
			}

			m.ActiveTab = TabBudget
			m.updateViewportContent()

			sym := GetCurrencySymbol(m.Currency)
			var logMsg string
			if targetBudget > 0 && guestCount > 0 {
				costPerGuest := targetBudget / float64(guestCount)
				logMsg = fmt.Sprintf("✨ Configured Total Budget: %s%.2f across standard categories for %d guests (Cost per guest: %s%.2f)!", sym, targetBudget, guestCount, sym, costPerGuest)
			} else if targetBudget > 0 {
				logMsg = fmt.Sprintf("✨ Configured Total Budget: %s%.2f across standard categories!", sym, targetBudget)
			} else if guestCount > 0 {
				logMsg = fmt.Sprintf("✨ Updated Event Headcount: %d guests!", guestCount)
			} else {
				logMsg = "💰 Switched to Budget & Spend Analysis Dashboard."
			}

			m.Logs = append(m.Logs, LogEntry{
				Sender: "BudgetAgent",
				Text:   logMsg,
			})
			return m, nil

		case command.CmdVendor:
			promptQuery := strings.TrimSpace(parsed.EventDetails)
			profile, _ := config.LoadEventProfile()

			if promptQuery != "" {
				placesKey := config.GetPlacesAPIKey()
				geminiKey := m.Config.GeminiAPIKey
				if geminiKey == "" {
					geminiKey = config.LoadConfig().GeminiAPIKey
				}

				suggestions := generator.FetchVenueSuggestions(promptQuery, placesKey, geminiKey, m.EventType)
				m.VenueSuggestions = suggestions
				m.VenueSearchQuery = promptQuery
				m.OptionIndex = 0
				m.Step = StepVenueSelection

				logText := renderVenueMenu(suggestions, 0, promptQuery)
				m.Logs = append(m.Logs, LogEntry{
					Sender: "VendorAgent",
					Text:   logText,
				})
				return m, nil
			}

			// Display current active venue details
			venueName := profile.Venue
			if venueName == "" {
				venueName = "Main Event Venue"
			}
			addr := profile.VenueAddress
			if addr == "" && profile.VenueDetails.VenueFormattedAddress != "" {
				addr = profile.VenueDetails.VenueFormattedAddress
			}
			if addr == "" {
				addr = "Address pending configuration"
			}
			mapsURL := profile.VenueDetails.GoogleMapURL
			if mapsURL == "" {
				mapsURL = "https://maps.google.com/?q=" + url.QueryEscape(venueName)
			}

			logText := fmt.Sprintf("📍 Active Venue Profile:\n  • Venue: %s\n  • Address: %s\n  • Google Maps: %s\n\n💡 Tip: Type '/vendor <query>' or '/venue <name>' (e.g. /venue MCC Hall Chennai) to search live Google Places & update venue!", venueName, addr, mapsURL)
			m.Logs = append(m.Logs, LogEntry{
				Sender: "VendorAgent",
				Text:   logText,
			})
			return m, nil

		case command.CmdHelp:
			helpText := `📌 Main Slash Commands:
  • /generate [details] - Compile context & generate invitation design
  • /event [details]    - View/update profile details in event_details.md
  • /vendor [query]     - Search venues, pricing & vendor recommendations
  • /budget [amount]   - View or set estimated budget (e.g. /budget 300000)
  • /rsvp              - Open Tab 4 (Guest Roster & Dietary Facts)
  • /add-rsvp          - Add/update guest RSVP & transport details
  • /timeline          - Open Tab 2 (Chronological Ceremony Schedule)
  • /currency [code]   - Set default currency (e.g. /currency INR, USD)
  • /welcome [text]    - View or set AI welcome message subheaders
  • /aspect [ratio]    - Set aspect ratio (9:16, 4:5, 1:1, 16:9)
  • /style [preset]    - Select aesthetic design style (e.g. /style paper cut)
  • /suggest [theme]   - Generate AI prompt suggestions
  • /refine [changes]  - Apply tweaks to active design
  • /preview           - Open local web preview browser
  • /honcho            - Inspect Honcho Cloud AI memory cards
  • /planner [name]    - Update event planner name & role
  • /export            - Export design or event details
  • /wizard            - Launch interactive step-by-step setup wizard
  • /config [key]      - View or set Gemini API key
  • /clear             - Clear terminal log screen
  • /reset             - Restart guided setup wizard
  • /help              - Display this reference guide

🔗 Command Aliases & Shortcuts:
  • /design, /create, /gen      --> /generate
  • /edit, /modify              --> /refine
  • /ideas, /theme, /sug        --> /suggest
  • /preset, /aesthetic, /sty   --> /style
  • /profile, /details          --> /event
  • /venue, /location           --> /vendor
  • /ratio, /res, /resolution   --> /aspect
  • /curr, /currency-code       --> /currency
  • /subheader, /msg            --> /welcome
  • /finance, /spend            --> /budget
  • /rsvps, /guests             --> /rsvp
  • /addrsvp, /new-rsvp         --> /add-rsvp
  • /schedule, /itinerary       --> /timeline
  • /memory, /cards             --> /honcho
  • /wiz                        --> /wizard
  • /web, /open                 --> /preview
  • /key, /apikey               --> /config
  • /cls                        --> /clear
  • /h, /?                      --> /help`
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
		
		// Parse & normalize raw date into machine-readable ISO 8601 & display strings
		isoDate, displayDate, _ := config.ParseAndNormalizeMachineDate(rawText)
		m.EventDate = displayDate

		m.Step = StepVenue
		m.TextInput.Placeholder = "e.g. Leela Palace, Bengaluru"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 4/5",
			Text:   fmt.Sprintf("Event Date: \"%s\" (Machine-Readable ISO: %s)\n\n📍 Enter Venue & Location (e.g. 'Leela Palace, Bengaluru'):", m.EventDate, isoDate),
		})
		return m, nil

	case StepVenue:
		logTxt := rawText
		if logTxt == "" {
			logTxt = "(skipped)"
		}
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: logTxt})
		if rawText != "" && rawText != "(skipped)" {
			placesKey := config.GetPlacesAPIKey()
			geminiKey := m.Config.GeminiAPIKey
			if geminiKey == "" {
				geminiKey = config.LoadConfig().GeminiAPIKey
			}

			suggestions := generator.FetchVenueSuggestions(rawText, placesKey, geminiKey, m.EventType)
			m.VenueSuggestions = suggestions
			m.VenueSearchQuery = rawText
			m.OptionIndex = 0
			m.Step = StepVenueSelection

			m.Logs = append(m.Logs, LogEntry{
				Sender: "STEP 4/5 - VENUE AUTOCOMPLETE",
				Text:   renderVenueMenu(suggestions, 0, rawText),
			})
			return m, nil
		}

		m.Venue = rawText
		m.Step = StepCurrency
		m.OptionIndex = 0
		m.TextInput.Placeholder = "Use ↑/↓ arrow keys or type currency code (e.g. USD, EUR, GBP, INR, AUD, SGD)"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 5/5",
			Text:   fmt.Sprintf("Venue & Location: \"%s\"\n\n%s", m.Venue, renderCurrencyMenu(m.OptionIndex)),
		})
		return m, nil

	case StepVenueSelection:
		selected := resolveVenueChoice(rawText, m.VenueSuggestions, m.OptionIndex, m.VenueSearchQuery)
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: selected.Text})

		placesKey := config.GetPlacesAPIKey()
		venueDetails := generator.FetchPlaceDetails(selected.PlaceID, placesKey)
		if venueDetails.PrimaryVenue == "" || venueDetails.PrimaryVenue == selected.PlaceID {
			venueDetails.PrimaryVenue = selected.Text
		}

		m.Venue = venueDetails.PrimaryVenue

		_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole)
		_ = config.SaveVenueDetails(venueDetails)

		m.Step = StepCurrency
		m.OptionIndex = 0
		m.TextInput.Placeholder = "Use ↑/↓ arrow keys or type currency code (e.g. USD, EUR, GBP, INR, AUD, SGD)"

		logTxt := fmt.Sprintf("✨ Selected & Saved Venue Details to event_details.md:\n  • Primary Venue: %s\n  • Full Address: %s\n  • Google Maps: %s\n  • Directions: %s\n  • Place ID: %s\n\n%s",
			venueDetails.PrimaryVenue, venueDetails.VenueFormattedAddress, venueDetails.GoogleMapURL, venueDetails.GoogleMapDirectionsURL, venueDetails.PlaceID, renderCurrencyMenu(m.OptionIndex))

		m.Logs = append(m.Logs, LogEntry{
			Sender: "VendorAgent",
			Text:   logTxt,
		})
		return m, nil

	case StepCurrency:
		resolved := resolveCurrencyChoice(rawText, m.OptionIndex)
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: resolved})
		m.Currency = resolved
		m.Step = StepWelcomeMessage
		m.TextInput.Placeholder = "Type '1' (or 'ai') for AI Subheaders, or enter custom subheader (or Enter to skip)"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 5/5 COMPLETE",
			Text:   fmt.Sprintf("Default Currency: \"%s\"\n\n💌 Enter Optional Secondary Welcome Subheader:\n  • Type '1' (or 'ai') to generate 4 AI Welcome Subheader Suggestions\n  • Or enter custom subheader text below (or press Enter to skip):", m.Currency),
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
		if m.Currency != "" {
			parts = append(parts, fmt.Sprintf("(Currency: %s)", m.Currency))
		}
		m.EventDetails = strings.Join(parts, " ")
		_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole)
		if m.ActiveTab != TabDesignStudio {
			m.Step = StepComplete
			m.TextInput.Placeholder = "Type an event prompt (e.g. 'Build Mehendi timeline', 'Check floral budget')..."
			m.Logs = append(m.Logs, LogEntry{
				Sender: "PROFILE",
				Text:   fmt.Sprintf("✨ Saved Event Profile to event_details.md:\n  • Event Type: %s\n  • Host/Couple: %s\n  • Date: %s\n  • Venue: %s\n  • Currency: %s\n  • Welcome: %s", m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage),
			})
			return m, nil
		}
		m.Step = StepAspectSelection
		m.OptionIndex = 0
		m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type resolution"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 2/4",
			Text:   fmt.Sprintf("✨ Saved Event Profile to event_details.md:\n  • Event Type: %s\n  • Host/Couple: %s\n  • Date: %s\n  • Venue: %s\n  • Currency: %s\n  • Welcome: %s\n\n%s", m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, renderAspectMenu(m.OptionIndex)),
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
		if m.Currency != "" {
			parts = append(parts, fmt.Sprintf("(Currency: %s)", m.Currency))
		}
		m.EventDetails = strings.Join(parts, " ")
		_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole)
		if m.ActiveTab != TabDesignStudio {
			m.Step = StepComplete
			m.TextInput.Placeholder = "Type an event prompt (e.g. 'Build Mehendi timeline', 'Check floral budget')..."
			m.Logs = append(m.Logs, LogEntry{
				Sender: "PROFILE",
				Text:   fmt.Sprintf("✨ Saved Event Profile to event_details.md:\n  • Event Type: %s\n  • Host/Couple: %s\n  • Date: %s\n  • Venue: %s\n  • Currency: %s\n  • Welcome: %s", m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage),
			})
			return m, nil
		}
		m.Step = StepAspectSelection
		m.OptionIndex = 0
		m.TextInput.Placeholder = "Use ↑/↓ arrow keys to select, press Enter to confirm, or type resolution"
		m.Logs = append(m.Logs, LogEntry{
			Sender: "STEP 2/4",
			Text:   fmt.Sprintf("✨ Saved Event Profile to event_details.md:\n  • Event Type: %s\n  • Host/Couple: %s\n  • Date: %s\n  • Venue: %s\n  • Currency: %s\n  • Welcome: %s\n\n%s", m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, renderAspectMenu(m.OptionIndex)),
		})
		return m, nil

	case StepAspectSelection:
		resolved := resolveAspectChoice(rawText, m.OptionIndex)
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: resolved})
		m.SelectedAspect = resolved
		_ = config.SaveStructuredEventProfile(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole)
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

	case StepComplete:
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: rawText})
		if g := parseGuestCountFromText(rawText); g > 0 {
			m.RSVPOverview.TotalGuests = g
			_ = config.SaveStructuredEventProfileWithBudget(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole, m.BudgetSummary.TotalEstimated, m.RSVPOverview.TotalGuests)
		}
		if amt := parseBudgetMutationFromPrompt(rawText, m.Currency); amt > 0 {
			m.BudgetSummary.TotalEstimated = amt
			_ = config.SaveStructuredEventProfileWithBudget(m.EventType, m.HostNames, m.EventDate, m.Venue, m.Currency, m.WelcomeMessage, m.SelectedAspect, m.PlannerName, m.PlannerRole, m.BudgetSummary.TotalEstimated, m.RSVPOverview.TotalGuests)
		}
		m.Loading = true
		m.StatusMsg = "Routing prompt to AI Assistant & Smart Memory..."
		return m, tea.Batch(
			m.Spinner.Tick,
			m.runAgentCmd(rawText),
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

func renderEventProfileTUI(p config.EventProfile) string {
	eType := p.EventType
	if eType == "" {
		eType = "Not Configured"
	}
	hosts := p.HostNames
	if hosts == "" {
		hosts = "Not Configured"
	}
	eDate := p.EventDate
	if eDate == "" {
		eDate = "Not Configured"
	}
	vd := p.VenueDetails
	venueName := p.Venue
	if venueName == "" {
		venueName = vd.PrimaryVenue
	}
	if venueName == "" {
		venueName = "Main Event Venue"
	}
	addr := p.VenueAddress
	if addr == "" {
		addr = vd.VenueFormattedAddress
	}
	if addr == "" {
		addr = vd.Address
	}
	if addr == "" {
		addr = "Venue address pending selection."
	}
	placeID := vd.PlaceID
	if placeID == "" {
		placeID = "TBD"
	}

	mapURL := vd.GoogleMapURL
	if mapURL == "" && venueName != "" {
		mapURL = "https://maps.google.com/?q=" + url.QueryEscape(venueName+" "+addr)
	}
	dirURL := vd.GoogleMapDirectionsURL
	if dirURL == "" && venueName != "" {
		dirURL = "https://www.google.com/maps/dir/?api=1&destination=" + url.QueryEscape(venueName+" "+addr)
	}

	sym := GetCurrencySymbol(p.DefaultCurrency)
	budgetStr := fmt.Sprintf("%s%.2f", sym, p.TotalBudget)
	if p.TotalBudget <= 0 {
		budgetStr = "Not Configured"
	}

	var sb strings.Builder
	sb.WriteString("📋 ACTIVE EVENT PROFILE & DETAILS (event_details.md / data/event-details.json)\n")
	sb.WriteString("──────────────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("🎉  Event Type:   %s\n", eType))
	sb.WriteString(fmt.Sprintf("👥  Hosts/Couple: %s\n", hosts))
	sb.WriteString(fmt.Sprintf("📅  Event Date:   %s\n", eDate))
	sb.WriteString(fmt.Sprintf("🏛️  Venue Name:   %s\n", venueName))
	sb.WriteString(fmt.Sprintf("📍  Full Address: %s\n", addr))
	sb.WriteString(fmt.Sprintf("🗺️  Google Maps:  %s\n", mapURL))
	sb.WriteString(fmt.Sprintf("🚗  Directions:   %s\n", dirURL))
	sb.WriteString(fmt.Sprintf("🔑  Place ID:     %s\n", placeID))
	sb.WriteString(fmt.Sprintf("💰  Total Budget: %s\n", budgetStr))
	sb.WriteString(fmt.Sprintf("💱  Currency:     %s (%s)\n", p.DefaultCurrency, sym))
	sb.WriteString(fmt.Sprintf("👤  Planner:      %s (%s)\n", p.PlannerName, p.PlannerRole))

	if p.WelcomeMessage != "" {
		sb.WriteString(fmt.Sprintf("💌  Welcome Msg:  \"%s\"\n", p.WelcomeMessage))
	}

	sb.WriteString("──────────────────────────────────────────────────────────────────\n")
	sb.WriteString("💡 Event Management Commands:\n")
	sb.WriteString("  • /event new (or /wizard)   - Launch interactive step-by-step setup wizard\n")
	sb.WriteString("  • /event update (or /edit)  - Modify existing event profile details\n")
	sb.WriteString("  • /venue <query>            - Search Google Places & update venue\n")
	sb.WriteString("  • /budget <amount>          - Update total estimated budget\n")
	sb.WriteString("  • /currency <code>          - Update default currency")

	return sb.String()
}
