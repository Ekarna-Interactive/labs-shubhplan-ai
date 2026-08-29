package genkitengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"shubh-plan-web/pkg/store"

	"github.com/firebase/genkit/go/ai"
	aix "github.com/firebase/genkit/go/ai/exp"
	"github.com/firebase/genkit/go/genkit"
)

type AssistantInput struct {
	UserMessage string `json:"userMessage"`
	SessionID   string `json:"sessionId"`
}

type AssistantOutput struct {
	Response    string   `json:"response"`
	SessionID   string   `json:"sessionId,omitempty"`
	ToolsCalled []string `json:"toolsCalled"`
}

type PromptSuggestionInput struct {
	AspectRatio         string   `json:"aspectRatio"`
	StylePreset         string   `json:"stylePreset"`
	CustomElements      []string `json:"customElements"`
	Typography          string   `json:"typography"`
	PrimaryColor        string   `json:"primaryColor"`
	SpecialInstructions string   `json:"specialInstructions"`
}

type PromptSuggestion struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	PromptText  string   `json:"promptText"`
	Elements    []string `json:"elements"`
	Style       string   `json:"style"`
	AspectRatio string   `json:"aspectRatio"`
}

type PromptSuggestionsOutput struct {
	Suggestions []PromptSuggestion `json:"suggestions"`
}

type InvitationGenInput struct {
	AestheticTheme      string   `json:"aestheticTheme"`
	StyleTheme          string   `json:"styleTheme,omitempty"`
	PrimaryColor        string   `json:"primaryColor"`
	Typography          string   `json:"typography"`
	AspectRatio         string   `json:"aspectRatio"`
	CustomElements      []string `json:"customElements"`
	SpecialInstructions string   `json:"specialInstructions"`
	PromptText          string   `json:"promptText,omitempty"`
	Prompt              string   `json:"prompt,omitempty"`
}

type InvitationGenOutput struct {
	MainConcept  store.InvitationDesign   `json:"mainConcept"`
	Alternatives []store.InvitationDesign `json:"alternatives"`
}

// FlowRegistry holds HTTP handlers and typed execution closures for all registered flows.
type FlowRegistry struct {
	AssistantHandler         http.HandlerFunc
	InvitationHandler        http.HandlerFunc
	PromptSuggestionsHandler http.HandlerFunc
	RSVPHandler              http.HandlerFunc
	ItineraryHandler         http.HandlerFunc

	RunAssistant         func(ctx context.Context, input AssistantInput) (AssistantOutput, error)
	RunInvitation        func(ctx context.Context, input InvitationGenInput) (InvitationGenOutput, error)
	RunPromptSuggestions func(ctx context.Context, input PromptSuggestionInput) (PromptSuggestionsOutput, error)
	RunRSVP              func(ctx context.Context, input AddGuestInput) (store.Guest, error)
	RunItinerary         func(ctx context.Context, input AddItineraryInput) (store.ItineraryItem, error)
}

// RegisterFlows registers all core Genkit flows and exports their HTTP handlers.
func RegisterFlows(engine *Engine, s *store.DataStore, toolMap map[string]ai.Tool, agents *AgentRegistry) *FlowRegistry {
	g := engine.Genkit
	reg := &FlowRegistry{}

	// 1. Intelligent Event Assistant Flow
	asstFlow := genkit.DefineFlow(g, "eventAssistantFlow",
		func(ctx context.Context, input AssistantInput) (AssistantOutput, error) {
			emptyTools := make([]string, 0)
			userMsgTrim := strings.TrimSpace(input.UserMessage)
			if userMsgTrim == "" {
				return AssistantOutput{Response: "How can I assist you with your event planning today?", ToolsCalled: emptyTools}, nil
			}

			msgLower := strings.ToLower(userMsgTrim)
			// Intercept Slash Commands and Quick Action keywords immediately for 100% reliable interactive widget rendering
			if strings.HasPrefix(msgLower, "/") || msgLower == "generate invitation" || msgLower == "add guest" || msgLower == "add guests" || msgLower == "schedule session" || msgLower == "summarize event" {
				return runFallbackAssistant(input.UserMessage, s)
			}

			// Configure agent run options for multi-turn session state persistence
			opts := []aix.InvocationOption[any]{}
			if input.SessionID != "" {
				opts = append(opts, aix.WithSessionID[any](input.SessionID))
			}

			activeKey := resolveAPIKey(ctx)
			if activeKey != "" && activeKey != "dummy" && activeKey != "byok_placeholder_key" {
				out, err := callGeminiAssistant(ctx, activeKey, input.UserMessage, s, input.SessionID)
				if err == nil && strings.TrimSpace(out.Response) != "" {
					// Ensure interactive component widgets are attached for natural language intents
					if (strings.Contains(msgLower, "invitation") || strings.Contains(msgLower, "invite") || strings.Contains(msgLower, "card")) && (strings.Contains(msgLower, "design") || strings.Contains(msgLower, "generate") || strings.Contains(msgLower, "create") || strings.Contains(msgLower, "make") || strings.Contains(msgLower, "want") || strings.Contains(msgLower, "like")) {
						if !strings.Contains(out.Response, "[WIDGET:") {
							out.Response += "\n\n[WIDGET:GENERATE_INVITATION]"
						}
					} else if strings.Contains(msgLower, "add") && (strings.Contains(msgLower, "guest") || strings.Contains(msgLower, "roster")) {
						if !strings.Contains(out.Response, "[WIDGET:") {
							out.Response += "\n\n[WIDGET:ADD_GUEST]"
						}
					} else if strings.Contains(msgLower, "schedule") || strings.Contains(msgLower, "itinerary") || strings.Contains(msgLower, "timeline") {
						if !strings.Contains(out.Response, "[WIDGET:") {
							out.Response += "\n\n[WIDGET:ADD_ITINERARY]"
						}
					}
					return out, nil
				}
				log.Printf("[Gemini Assistant Warning] callGeminiAssistant error: %v, falling back...", err)
			}

			return runFallbackAssistant(input.UserMessage, s)
		},
	)
	reg.AssistantHandler = genkit.Handler(asstFlow)
	reg.RunAssistant = func(ctx context.Context, input AssistantInput) (AssistantOutput, error) {
		return asstFlow.Run(ctx, input)
	}

	// 2. AI Prompt Suggestions Generator Flow
	promptSugFlow := genkit.DefineFlow(g, "invitationPromptSuggestionsFlow",
		func(ctx context.Context, input PromptSuggestionInput) (PromptSuggestionsOutput, error) {
			evt := s.GetEvent()

			if input.CustomElements == nil {
				input.CustomElements = []string{}
			}

			ar := input.AspectRatio
			if ar == "" {
				ar = "9:16"
			}
			style := input.StylePreset
			if style == "" {
				style = "Clay 3D"
			}
			typo := input.Typography
			if typo == "" {
				typo = "Cinzel Decorative & Outfit"
			}
			color := input.PrimaryColor
			if color == "" {
				color = "#D4AF37"
			}

			// Try remote Gemini LLM generation if API key is active
			apiKey := resolveAPIKey(ctx)
			if apiKey != "" && apiKey != "dummy" {
				llmSugs, err := fetchLLMPromptSuggestions(apiKey, input, evt)
				if err == nil && len(llmSugs) > 0 {
					log.Printf("[Genkit Engine] Successfully synthesized %d AI prompt suggestions via Gemini LLM", len(llmSugs))
					return PromptSuggestionsOutput{Suggestions: llmSugs}, nil
				}
				log.Printf("[Genkit Engine Warning] Gemini LLM prompt synthesis failed (%v), using local fallback...", err)
			}

			title := evt.Title
			if title == "" {
				title = "Special Celebration"
			}
			humanDate := FormatHumanReadableDate(evt.Date)
			if humanDate == "" || evt.Date == "" {
				humanDate = "Date To Be Announced"
			}
			venueStr := evt.Venue
			if venueStr == "" {
				venueStr = "Main Function Hall"
			}

			elementsStr := "heritage decorative motifs"
			if len(input.CustomElements) > 0 {
				elementsStr = strings.Join(input.CustomElements, ", ")
			}

			extraInst := ""
			if strings.TrimSpace(input.SpecialInstructions) != "" {
				extraInst = fmt.Sprintf(" Custom notes: %s.", input.SpecialInstructions)
			}

			suggestions := []PromptSuggestion{
				{
					ID:          "opt_1",
					Title:       fmt.Sprintf("Option 1: Traditional %s Heritage", style),
					Description: fmt.Sprintf("Classic %s arrangement for '%s' (%s) optimized for %s format, featuring prominent %s with %s typography.", style, title, humanDate, ar, elementsStr, typo),
					PromptText:  fmt.Sprintf("Bespoke %s invitation card artwork for '%s' on %s at %s in %s format with primary color accent %s. Incorporating intricate %s, elegant %s typography, and symmetrical border framing.%s", style, title, humanDate, venueStr, ar, color, elementsStr, typo, extraInst),
					Elements:    input.CustomElements,
					Style:       style,
					AspectRatio: ar,
				},
				{
					ID:          "opt_2",
					Title:       fmt.Sprintf("Option 2: Modern Minimalist %s Foil", style),
					Description: fmt.Sprintf("Sleek obsidian background with polished gold line art, subtle %s accents, and clean modern framing for %s.", elementsStr, title),
					PromptText:  fmt.Sprintf("Luxury minimalist invitation card for '%s' on %s in %s format. Deep dark obsidian background with sleek %s line art in %s color, featuring stylized %s and serif typography.%s", title, humanDate, ar, style, color, elementsStr, extraInst),
					Elements:    input.CustomElements,
					Style:       style,
					AspectRatio: ar,
				},
				{
					ID:          "opt_3",
					Title:       fmt.Sprintf("Option 3: Sculpted Filigree & %s Motifs", style),
					Description: fmt.Sprintf("High-contrast 3D depth composition for '%s' with layered filigree arches, highlighted %s, and warm ambient lighting.", title, elementsStr),
					PromptText:  fmt.Sprintf("Ornate 3D %s invitation graphic for '%s' on %s at %s in %s format. Rich sculpted depth featuring central plaque framed by %s, warm gold lighting, and %s accent details.%s", style, title, humanDate, venueStr, ar, elementsStr, color, extraInst),
					Elements:    input.CustomElements,
					Style:       style,
					AspectRatio: ar,
				},
				{
					ID:          "opt_4",
					Title:       fmt.Sprintf("Option 4: Royal Floral & %s Vignette", style),
					Description: fmt.Sprintf("Vibrant celebratory composition for '%s' surrounded by soft botanical flora, festive %s, and royal plaque borders.", title, elementsStr),
					PromptText:  fmt.Sprintf("Vibrant hand-crafted %s invitation card artwork for '%s' on %s tailored for %s. Decorative background with festive %s, rich %s color palette, and elegant central crest.%s", style, title, humanDate, ar, elementsStr, color, extraInst),
					Elements:    input.CustomElements,
					Style:       style,
					AspectRatio: ar,
				},
			}

			return PromptSuggestionsOutput{Suggestions: suggestions}, nil
		},
	)
	reg.PromptSuggestionsHandler = genkit.Handler(promptSugFlow)
	reg.RunPromptSuggestions = func(ctx context.Context, input PromptSuggestionInput) (PromptSuggestionsOutput, error) {
		return promptSugFlow.Run(ctx, input)
	}

	// 3. Multi-Option Invitation Studio Flow (Generates Real PNG Card Artwork)
	invFlow := genkit.DefineFlow(g, "invitationGeneratorFlow",
		func(ctx context.Context, input InvitationGenInput) (InvitationGenOutput, error) {
			evt := s.GetEvent()

			theme := input.AestheticTheme
			if theme == "" {
				theme = input.StyleTheme
			}
			if theme == "" {
				theme = evt.AestheticTheme
			}
			if theme == "" {
				theme = "Clay 3D"
			}
			color := input.PrimaryColor
			if color == "" {
				color = "#D4AF37"
			}
			typo := input.Typography
			if typo == "" {
				typo = "Cinzel Decorative & Outfit"
			}
			ar := input.AspectRatio
			if ar == "" {
				ar = "4:5"
			}

			title := evt.Title
			if title == "" {
				title = "Special Celebration"
			}
			humanDate := FormatHumanReadableDate(evt.Date)
			subheadStr := fmt.Sprintf("%s • %s", humanDate, evt.Venue)
			if evt.Date == "" {
				subheadStr = "Date & Venue To Be Announced"
			}

			mainPrompt := input.PromptText
			if strings.TrimSpace(mainPrompt) == "" {
				mainPrompt = input.Prompt
			}
			if strings.TrimSpace(mainPrompt) == "" {
				elemStr := ""
				if len(input.CustomElements) > 0 {
					elemStr = fmt.Sprintf(" Custom elements: %s.", strings.Join(input.CustomElements, ", "))
				}
				mainPrompt = fmt.Sprintf("Bespoke %s invitation card artwork in %s aspect ratio with %s typography, %s accent color.%s %s", theme, ar, typo, color, elemStr, input.SpecialInstructions)
			}

			imgRes := generateImageWithAPI(ctx, mainPrompt, theme, ar, s)
			imageURL := imgRes.ImageURL
			if imageURL == "" {
				imageURL = "/assets/card_fallback.png"
			}

			if input.CustomElements == nil {
				input.CustomElements = []string{}
			}

			mainSpec := store.InvitationDesign{
				Prompt:         mainPrompt,
				StyleTheme:     theme,
				PrimaryColor:   color,
				Typography:     typo,
				AspectRatio:    ar,
				CustomElements: input.CustomElements,
				ImageURL:       imageURL,
				Headline:       title,
				Subhead:        subheadStr,
				CreatedAt:      time.Now(),
			}

			return InvitationGenOutput{
				MainConcept:  mainSpec,
				Alternatives: []store.InvitationDesign{},
			}, nil
		},
	)
	reg.InvitationHandler = genkit.Handler(invFlow)
	reg.RunInvitation = func(ctx context.Context, input InvitationGenInput) (InvitationGenOutput, error) {
		return invFlow.Run(ctx, input)
	}

	// 3. RSVP Management Flow
	rsvpFlow := genkit.DefineFlow(g, "rsvpManagementFlow",
		func(ctx context.Context, input AddGuestInput) (store.Guest, error) {
			guest := store.Guest{
				Name:                input.Name,
				Category:            input.Category,
				RSVPStatus:          input.RSVPStatus,
				DietaryRequirements: input.DietaryRequirements,
				PlusOnes:            input.PlusOnes,
				Phone:               input.Phone,
				Notes:               input.Notes,
			}
			saved := s.AddOrUpdateGuest(guest)
			return saved, nil
		},
	)
	reg.RSVPHandler = genkit.Handler(rsvpFlow)
	reg.RunRSVP = func(ctx context.Context, input AddGuestInput) (store.Guest, error) {
		return rsvpFlow.Run(ctx, input)
	}

	// 4. Itinerary Planner Flow
	itinFlow := genkit.DefineFlow(g, "itineraryPlannerFlow",
		func(ctx context.Context, input AddItineraryInput) (store.ItineraryItem, error) {
			item := store.ItineraryItem{
				Time:        input.Time,
				Title:       input.Title,
				Description: input.Description,
				Location:    input.Location,
				Host:        input.Host,
			}
			saved := s.AddItineraryItem(item)
			return saved, nil
		},
	)
	reg.ItineraryHandler = genkit.Handler(itinFlow)
	reg.RunItinerary = func(ctx context.Context, input AddItineraryInput) (store.ItineraryItem, error) {
		return itinFlow.Run(ctx, input)
	}

	return reg
}

// callGeminiAssistant invokes Gemini REST API using the client's API key to execute AI assistant logic.
func callGeminiAssistant(ctx context.Context, apiKey string, userMsg string, s *store.DataStore, sessionID string) (AssistantOutput, error) {
	evt := s.GetEvent()
	guests := s.ListGuests()
	confirmedCount := 0
	pendingCount := 0
	declinedCount := 0
	guestSummaryList := []string{}
	for _, g := range guests {
		if strings.EqualFold(g.RSVPStatus, "confirmed") {
			confirmedCount++
		} else if strings.EqualFold(g.RSVPStatus, "declined") {
			declinedCount++
		} else {
			pendingCount++
		}
		guestSummaryList = append(guestSummaryList, fmt.Sprintf("%s (%s - %s, Plus Ones: +%d, Dietary: %s)", g.Name, g.Category, g.RSVPStatus, g.PlusOnes, g.DietaryRequirements))
	}
	guestStr := fmt.Sprintf("Total Registered: %d (Confirmed: %d, Pending: %d, Declined: %d)", len(guests), confirmedCount, pendingCount, declinedCount)
	if len(guestSummaryList) > 0 {
		guestStr += "\n- Guest Entries: " + strings.Join(guestSummaryList, "; ")
	} else {
		guestStr += "\n- Guest Entries: None registered yet"
	}

	itinItems := s.ListItinerary()
	itinList := []string{}
	for _, item := range itinItems {
		itinList = append(itinList, fmt.Sprintf("%s: %s at %s", item.Time, item.Title, item.Location))
	}
	itinStr := fmt.Sprintf("%d scheduled sessions", len(itinItems))
	if len(itinList) > 0 {
		itinStr += "\n- Sessions: " + strings.Join(itinList, "; ")
	}

	sysPrompt := fmt.Sprintf(`You are Shubh Plan Copilot, an expert AI Event Planner & Concierge.
Current Event Workspace Profile:
- Title: %q
- Event Type: %q
- Date: %q
- Venue: %q
- Host Names: %q
- Aesthetic Theme: %q
- Target Expected Guest Count: %d

Current Guest Roster & RSVP Statuses:
- %s

Current Event Itinerary Schedule:
- %s

User Request: %q

INSTRUCTIONS:
1. When asked about guests or RSVPs, give exact numbers for Target Expected Guests (%d), Total Registered (%d), and Confirmed RSVPs (%d), including specific guest entries if relevant.
2. If the user mentions event creation or event details (e.g. naming ceremony, wedding, birthday, host names, venue, date, guest count, aesthetic theme), extract those event details.
3. If the user asks to generate/create an invitation card or design artwork, append [WIDGET:GENERATE_INVITATION] at the end of your response text.
4. If the user asks to add guests or manage guest roster, append [WIDGET:ADD_GUEST] at the end of your response text.
5. If the user asks to schedule a session or update itinerary, append [WIDGET:ADD_ITINERARY] at the end of your response text.
6. Return a JSON object with NO markdown formatting matching this exact schema:
{
  "response": "<your warm, professional, markdown-formatted conversational response>",
  "updateEvent": {
    "title": "<updated title or empty>",
    "eventType": "<updated event type or empty>",
    "date": "<updated date or empty>",
    "venue": "<updated venue or empty>",
    "hosts": "<updated host names or empty>",
    "aestheticTheme": "<updated theme or empty>",
    "targetGuestCount": <number of target guests as int, or 0>
  }
}`, evt.Title, evt.EventType, evt.Date, evt.Venue, evt.HostNames, evt.AestheticTheme, evt.TargetGuestCount, guestStr, itinStr, userMsg, evt.TargetGuestCount, len(guests), confirmedCount)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": sysPrompt},
				},
			},
		},
		"generation_config": map[string]interface{}{
			"temperature":        0.7,
			"response_mime_type": "application/json",
		},
	}

	jsonBytes, _ := json.Marshal(reqBody)
	models := []string{"gemini-flash-latest"}
	for _, modelName := range models {
		endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBytes))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err == nil && resp.StatusCode == http.StatusOK {
				var resStruct struct {
					Candidates []struct {
						Content struct {
							Parts []struct {
								Text string `json:"text"`
							} `json:"parts"`
						} `json:"content"`
					} `json:"candidates"`
				}
				if json.Unmarshal(body, &resStruct) == nil && len(resStruct.Candidates) > 0 {
					rawText := resStruct.Candidates[0].Content.Parts[0].Text
					rawText = strings.TrimPrefix(rawText, "```json")
					rawText = strings.TrimPrefix(rawText, "```")
					rawText = strings.TrimSuffix(rawText, "```")
					rawText = strings.TrimSpace(rawText)

					var parsedRes struct {
						Response    string `json:"response"`
						UpdateEvent struct {
							Title            string `json:"title"`
							EventType        string `json:"eventType"`
							Date             string `json:"date"`
							Venue            string `json:"venue"`
							Hosts            string `json:"hosts"`
							AestheticTheme   string `json:"aestheticTheme"`
							TargetGuestCount int    `json:"targetGuestCount"`
						} `json:"updateEvent"`
					}
					if json.Unmarshal([]byte(rawText), &parsedRes) == nil && strings.TrimSpace(parsedRes.Response) != "" {
						ue := parsedRes.UpdateEvent
						if ue.Title != "" || ue.EventType != "" || ue.Venue != "" || ue.Hosts != "" || ue.Date != "" || ue.AestheticTheme != "" || ue.TargetGuestCount > 0 {
							curEvt := s.GetEvent()
							if ue.Title != "" {
								curEvt.Title = ue.Title
							}
							if ue.EventType != "" {
								curEvt.EventType = ue.EventType
							}
							if ue.Date != "" {
								curEvt.Date = ue.Date
							}
							if ue.Venue != "" {
								curEvt.Venue = ue.Venue
								curEvt.VenueDetails = verifyVenueWithGoogleMaps(ctx, ue.Venue, "")
							}
							if ue.Hosts != "" {
								curEvt.HostNames = ue.Hosts
							}
							if ue.AestheticTheme != "" {
								curEvt.AestheticTheme = ue.AestheticTheme
							}
							if ue.TargetGuestCount > 0 {
								curEvt.TargetGuestCount = ue.TargetGuestCount
							}
							s.UpdateEvent(curEvt)
						}
						return AssistantOutput{
							Response:    parsedRes.Response,
							SessionID:   sessionID,
							ToolsCalled: []string{"updateEventDetails"},
						}, nil
					}
				}
			}
		}
	}
	return AssistantOutput{}, fmt.Errorf("Gemini REST API generation failed")
}

// runFallbackAssistant performs local keyword parsing and interactive slash command handling.
func runFallbackAssistant(msg string, s *store.DataStore) (AssistantOutput, error) {
	msgLower := strings.TrimSpace(strings.ToLower(msg))
	evt := s.GetEvent()
	toolsCalled := make([]string, 0)

	// Smart event detail detection
	if strings.Contains(msgLower, "naming ceremony") || strings.Contains(msgLower, "wedding") || strings.Contains(msgLower, "birthday") || strings.Contains(msgLower, "anniversary") || strings.Contains(msgLower, "event for") {
		evtType := "Special Celebration"
		if strings.Contains(msgLower, "naming ceremony") {
			evtType = "Naming Ceremony"
		} else if strings.Contains(msgLower, "wedding") {
			evtType = "Wedding"
		} else if strings.Contains(msgLower, "birthday") {
			evtType = "Birthday"
		} else if strings.Contains(msgLower, "anniversary") {
			evtType = "Anniversary"
		}

		title := fmt.Sprintf("%s Celebration", evtType)
		if idx := strings.Index(msgLower, "for "); idx != -1 {
			namePart := strings.TrimSpace(msg[idx+4:])
			if namePart != "" {
				title = fmt.Sprintf("%s's %s", strings.Title(namePart), evtType)
				evt.HostNames = fmt.Sprintf("%s's Family", strings.Title(namePart))
			}
		}

		evt.Title = title
		evt.EventType = evtType
		s.UpdateEvent(evt)
		return AssistantOutput{
			Response:    fmt.Sprintf("I have configured your event workspace for **%s** (%s)! 🌟\n\nPlease share the event date, venue location, or host names so I can complete your event profile!", title, evtType),
			ToolsCalled: []string{"updateEventDetails"},
		}, nil
	}

	if evt.Title == "" {
		return AssistantOutput{
			Response:    "Hello and welcome to **Shubh Plan Web**! 🌟\n\nNo active event is configured in your workspace yet. Please share a few details about your upcoming event (e.g. event type, couple/host names, date, and venue) and I will configure your workspace!",
			ToolsCalled: []string{"getEventDetails"},
		}, nil
	}

	// 1. /summarize or Summarize Event
	if msgLower == "/summarize" || strings.Contains(msgLower, "summarize") {
		toolsCalled = append(toolsCalled, "getEventDetails", "listGuests", "listItinerary", "listDesigns")
		guests := s.ListGuests()
		conf := 0
		for _, g := range guests {
			if strings.EqualFold(g.RSVPStatus, "confirmed") {
				conf++
			}
		}
		items := s.ListItinerary()
		designs := s.ListDesigns()

		resp := fmt.Sprintf("📊 **Executive Summary for %s**\n\n"+
			"• **Event Profile**: %s on %s at %s\n"+
			"• **Guest Roster**: %d total guests (%d Confirmed, %d Pending/Declined)\n"+
			"• **Itinerary Schedule**: %d sessions configured\n"+
			"• **Design Studio**: %d invitation card concepts\n\n"+
			"✨ *What details would you like to update or refine next? (e.g., Guest Roster, Venue Details, Schedule, or Card Artwork)*",
			evt.Title, evt.Title, evt.Date, evt.Venue, len(guests), conf, len(guests)-conf, len(items), len(designs))
		return AssistantOutput{Response: resp, ToolsCalled: toolsCalled}, nil
	}

	// 2. /add-guests or Add Guests
	if msgLower == "/add-guests" || (strings.Contains(msgLower, "add") && strings.Contains(msgLower, "guest")) {
		toolsCalled = append(toolsCalled, "listGuests")
		resp := "👥 Here is your Quick Add Guest component widget:\n\n[WIDGET:ADD_GUEST]"
		return AssistantOutput{Response: resp, ToolsCalled: toolsCalled}, nil
	}

	// 3. /schedule or Schedule Session
	if msgLower == "/schedule" || strings.Contains(msgLower, "schedule") {
		toolsCalled = append(toolsCalled, "listItinerary")
		resp := "📅 Here is your Quick Schedule Session component widget:\n\n[WIDGET:ADD_ITINERARY]"
		return AssistantOutput{Response: resp, ToolsCalled: toolsCalled}, nil
	}

	// 4. /generate-invitation or Generate Invitation
	if msgLower == "/generate-invitation" || strings.Contains(msgLower, "generate invitation") || strings.Contains(msgLower, "create invitation") || strings.Contains(msgLower, "design invitation") || strings.Contains(msgLower, "design a invitation") || strings.Contains(msgLower, "design an invitation") || ((strings.Contains(msgLower, "invitation") || strings.Contains(msgLower, "invite") || strings.Contains(msgLower, "card")) && (strings.Contains(msgLower, "design") || strings.Contains(msgLower, "create") || strings.Contains(msgLower, "make") || strings.Contains(msgLower, "want") || strings.Contains(msgLower, "like"))) {
		toolsCalled = append(toolsCalled, "createInvitationSpec")
		resp := "🎨 Here is your Quick Generate Invitation component widget:\n\n[WIDGET:GENERATE_INVITATION]"
		return AssistantOutput{Response: resp, ToolsCalled: toolsCalled}, nil
	}

	if strings.Contains(msgLower, "guest") || strings.Contains(msgLower, "rsvp") || strings.Contains(msgLower, "roster") {
		toolsCalled = append(toolsCalled, "listGuests")
		guests := s.ListGuests()
		conf := 0
		for _, g := range guests {
			if strings.EqualFold(g.RSVPStatus, "confirmed") {
				conf++
			}
		}
		resp := fmt.Sprintf("You currently have **%d total guests** on your roster (%d Confirmed, %d Pending). Would you like me to add a new guest or export the RSVP list?", len(guests), conf, len(guests)-conf)
		return AssistantOutput{Response: resp, ToolsCalled: toolsCalled}, nil
	}

	if strings.Contains(msgLower, "itinerary") || strings.Contains(msgLower, "time") {
		toolsCalled = append(toolsCalled, "listItinerary")
		items := s.ListItinerary()
		if len(items) == 0 {
			return AssistantOutput{Response: "Your itinerary schedule is currently empty. Would you like me to add a session?", ToolsCalled: toolsCalled}, nil
		}
		resp := fmt.Sprintf("Your event schedule currently has **%d main sessions**:\n", len(items))
		for _, it := range items {
			resp += fmt.Sprintf("• **%s**: %s (%s)\n", it.Time, it.Title, it.Location)
		}
		return AssistantOutput{Response: resp, ToolsCalled: toolsCalled}, nil
	}

	// Default helpful overview
	resp := fmt.Sprintf("Hello! You are currently working on **%s** on %s at %s. How can I assist you with your event planning or invitation design today?", evt.Title, evt.Date, evt.Venue)
	return AssistantOutput{Response: resp, ToolsCalled: toolsCalled}, nil
}

// fetchLLMPromptSuggestions calls Gemini LLM API to dynamically synthesize 4 distinct invitation prompt suggestions.
func fetchLLMPromptSuggestions(apiKey string, input PromptSuggestionInput, evt store.EventProfile) ([]PromptSuggestion, error) {
	title := evt.Title
	if title == "" {
		title = "Special Celebration"
	}
	humanDate := FormatHumanReadableDate(evt.Date)
	if humanDate == "" || evt.Date == "" {
		humanDate = "Upcoming Date"
	}
	venueStr := evt.Venue
	if venueStr == "" {
		venueStr = "Main Function Hall"
	}

	elementsStr := "heritage decorative motifs"
	if len(input.CustomElements) > 0 {
		elementsStr = strings.Join(input.CustomElements, ", ")
	}

	sysPrompt := fmt.Sprintf(`You are an expert AI Invitation Card Art Director.
Generate 4 distinct, highly creative visual invitation prompt suggestions based on these parameters:
- Event Title: %s
- Date: %s
- Venue: %s
- Target Aspect Ratio: %s
- Style Preset: %s
- Custom Elements to include: %s
- Typography Pair: %s
- Accent Color: %s
- Additional Instructions: %s

CRITICAL REQUIREMENT: Return ONLY a valid JSON object with NO markdown formatting, NO backtick code blocks, strictly matching this exact schema:
{
  "suggestions": [
    {
      "id": "opt_1",
      "title": "Option 1: <Short Creative Title>",
      "description": "<1-2 sentence description of layout, mood, and visual composition>",
      "promptText": "<Detailed image generation prompt string describing the physical card, plaque text, background motifs, lighting, color accents, and aspect ratio>"
    },
    {
      "id": "opt_2",
      "title": "Option 2: <Short Creative Title>",
      "description": "<1-2 sentence description>",
      "promptText": "<Detailed image prompt>"
    },
    {
      "id": "opt_3",
      "title": "Option 3: <Short Creative Title>",
      "description": "<1-2 sentence description>",
      "promptText": "<Detailed image prompt>"
    },
    {
      "id": "opt_4",
      "title": "Option 4: <Short Creative Title>",
      "description": "<1-2 sentence description>",
      "promptText": "<Detailed image prompt>"
    }
  ]
}`, title, humanDate, venueStr, input.AspectRatio, input.StylePreset, elementsStr, input.Typography, input.PrimaryColor, input.SpecialInstructions)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": sysPrompt},
				},
			},
		},
		"generation_config": map[string]interface{}{
			"temperature":        0.7,
			"response_mime_type": "application/json",
		},
	}

	jsonBytes, _ := json.Marshal(reqBody)
	models := []string{"gemini-flash-latest"}
	for _, modelName := range models {
		endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
		resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonBytes))
		if err == nil {
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err == nil && resp.StatusCode == http.StatusOK {
				var resStruct struct {
					Candidates []struct {
						Content struct {
							Parts []struct {
								Text string `json:"text"`
							} `json:"parts"`
						} `json:"content"`
					} `json:"candidates"`
				}
				if json.Unmarshal(body, &resStruct) == nil && len(resStruct.Candidates) > 0 {
					text := resStruct.Candidates[0].Content.Parts[0].Text
					text = strings.TrimPrefix(text, "```json")
					text = strings.TrimPrefix(text, "```")
					text = strings.TrimSuffix(text, "```")
					text = strings.TrimSpace(text)

					var out PromptSuggestionsOutput
					if json.Unmarshal([]byte(text), &out) == nil && len(out.Suggestions) > 0 {
						for i := range out.Suggestions {
							out.Suggestions[i].Style = input.StylePreset
							out.Suggestions[i].AspectRatio = input.AspectRatio
							out.Suggestions[i].Elements = input.CustomElements
						}
						return out.Suggestions, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("gemini LLM API call returned empty or invalid json")
}
