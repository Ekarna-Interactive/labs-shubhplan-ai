package genkitengine

import (
	"fmt"
	"log"

	"github.com/firebase/genkit/go/ai"
	aix "github.com/firebase/genkit/go/ai/exp"
	"github.com/firebase/genkit/go/ai/exp/localstore"
	genkitx "github.com/firebase/genkit/go/genkit/exp"
	"shubh-plan-web/pkg/store"
)

// AgentRegistry holds references to defined Genkit experimental agents.
type AgentRegistry struct {
	EventPlannerAgent     *aix.Agent[any]
	GuestConciergeAgent   *aix.Agent[any]
	InvitationStudioAgent *aix.Agent[any]
	ItineraryAgent        *aix.Agent[any]
}

// RegisterAgents defines and registers experimental Genkit Agents so they appear under Agents (4) in the Dev UI.
func RegisterAgents(engine *Engine, s *store.DataStore, toolMap map[string]ai.Tool) *AgentRegistry {
	g := engine.Genkit
	reg := &AgentRegistry{}

	toolRefList := make([]ai.ToolRef, 0, len(toolMap))
	for _, t := range toolMap {
		toolRefList = append(toolRefList, t)
	}

	var sessionStore aix.SessionStore[any]
	fileStore, err := localstore.NewFileSessionStore[any]("./data/sessions")
	if err != nil {
		log.Printf("[Genkit Engine Warning] Could not initialize file session store, falling back to in-memory: %v", err)
		sessionStore = localstore.NewInMemorySessionStore[any]()
	} else {
		sessionStore = fileStore
	}

	// Security Boundary & Privacy Mandate
	securityPrivacyMandate := "SECURITY BOUNDARY & PRIVACY MANDATE:\n" +
		"• NO ACCESS TO SYSTEM SECRETS OR .ENV: You do NOT have access to environment configuration files (.env), server file paths, API keys, or system credentials.\n" +
		"• STRICT CONFIDENTIALITY: Never attempt to inspect, read, print, or disclose API keys, environment variables, or system secrets under any circumstances."

	// Web Client Interactive Widgets Awareness
	clientWidgetsMandate := "INTERACTIVE WEB UI WIDGETS AWARENESS:\n" +
		"• GUEST MANAGEMENT WIDGET: When discussing guest registration, RSVPs, or adding attendees, inform the user they can use the Quick Add Guest Widget embedded directly in the chat bubble or guest roster tab to select options (Category, Status, Plus-Ones, Dietary) for 1-click submission.\n" +
		"• PROMPT ACTION BUTTONS: When presenting card artwork concepts, format options as '### Option 1: ...', '### Option 2: ...', '### Option 3: ...', '### Option 4: ...' so the web UI automatically renders interactive '[🎨 Use Option X]' buttons for the user.\n" +
		"• VERIFIED VENUE SHOWCASE CARD: When searching for venues using `searchVenueInfo`, explain that the verified location photo, address, and Google Maps & Directions shortcuts are displayed in the right sidebar's Verified Venue Showcase card."

	// 7 Presets for Invitation Card Design
	designStylesPreset := "7 SIGNATURE DESIGN STYLE PRESETS:\n" +
		"1. South Indian (Kolam art, banana leaf motifs, marigold garlands, traditional temple gold)\n" +
		"2. Paper Cut (Intricate 3D layered papercraft, delicate die-cut filigree & drop shadows)\n" +
		"3. Clay 3D (Charming claymation 3D characters, vibrant sculpted textures & playful depth)\n" +
		"4. Pop Art (Bold graphic comic lines, vibrant retro pop colors, high contrast halftone)\n" +
		"5. Mughal (Royal jali lattice arches, intricate Persian floral vines, rich emerald & gold)\n" +
		"6. Minimalist Gold (Obsidian dark / ivory background, sleek gold foil line art & serif typography)\n" +
		"7. Watercolor (Soft hand-painted pastel washes, delicate botanical flora & fluid artistic gradients)"

	// Negative framing & clean generation mandates (matching apps/shubh-plan-open/generator/basic_builder.go)
	cleanGenerationMandate := "CLEAN IMAGE GENERATION MANDATES:\n" +
		"• STANDALONE PHYSICAL CARD: Formulate visual prompts for standalone physical invitation card graphic artwork.\n" +
		"• NEGATIVE UI FRAMING: Always include negative instructions: 'no smartphone UI, no mobile status bar, no clock or battery status bar, no screen bezels'.\n" +
		"• NO EXTRANEOUS TEXT: Render ONLY the exact event title, date, and venue inside the central plaque. Do NOT print prompt meta-descriptions, camera instructions, or labels on the card.\n" +
		"• DYNAMIC ASPECT RATIO: Structure prompts matching the user's requested aspect ratio (9:16 vertical story, 4:5 portrait feed, 1:1 square post, or 16:9 landscape banner) with generous margin padding around all borders."

	// Guided multi-step conversation rule for Invitation Studio & Event Planner with Inline Image Rendering
	invitationStudioRule := "INVITATION STUDIO GUIDED USER FLOW:\n" +
		"1. Always call `getEventDetails` first at the start of a turn.\n" +
		"2. IF EVENT IS UNCONFIGURED: Ask the user for event title/names and date, then call `updateEventDetails` to save.\n" +
		"3. VENUE SEARCH & INTERACTIVE USER CONFIRMATION:\n" +
		"   a. When the user provides venue or city details, IMMEDIATELY call `searchVenueInfo` tool.\n" +
		"   b. Present `PrimaryVenue`, full `Address`, and `GoogleMapURL` to the user and ask for confirmation.\n" +
		"   c. Save confirmed `venueDetails` using `updateEventDetails`.\n" +
		"4. ASK FOR DESIGN STYLE PRESET & ASPECT RATIO PREFERENCE: Ask the user to choose a design style from the 7 presets:\n" + designStylesPreset + "\n" +
		"   AND ask for their preferred aspect ratio (9:16 Story/Mobile, 4:5 Portrait/Feed, 1:1 Square, or 16:9 Landscape Banner) before generating prompt suggestions.\n" +
		"5. WHEN DESIGN STYLE & ASPECT RATIO ARE SELECTED:\n" +
		"   a. Call `createInvitationSpec` to persist the design specification to the store.\n" +
		"   b. Output EXACTLY 4 distinct Image Prompt Suggestions formatted for " + DefaultImageModelName + " image generation (Options 1, 2, 3, 4) following the " + cleanGenerationMandate + "\n" +
		"6. PNG IMAGE GENERATION TRIGGER & INLINE DISPLAY (CRITICAL MANDATE):\n" +
		"   a. WHEN THE USER SELECTS A PROMPT OPTION (e.g. 'Option 2', 'Option 1', 'Use Option 1', or a custom prompt): YOU MUST IMMEDIATELY CALL THE `generateInvitationImage` TOOL passing the prompt string, chosen style, and aspect ratio.\n" +
		"   b. AFTER `generateInvitationImage` EXECUTES: YOU MUST RENDER THE GENERATED IMAGE INLINE using Markdown syntax: `![Invitation Card Artwork](http://localhost:3000/assets/card_<id>.png)` (using the exact `markdownImg` value returned by the tool)."

	generalOnboardingRule := "CONVERSATIONAL RULES:\n" +
		"1. Always call `getEventDetails` first.\n" +
		"2. IF AN EVENT IS CONFIGURED: Warmly greet the user, summarize current event details (Title, Date, Venue, Theme), and ask how to assist.\n" +
		"3. IF NO EVENT IS CONFIGURED: Greet the user, state that no active event is set up yet, and ask for basic event info in a brief 1-2 sentence prompt.\n" +
		"4. VENUE SEARCH & CONFIRMATION: Call `searchVenueInfo` tool when venue names are mentioned, present the formatted address and Google Maps link, and ask the user to confirm if it is the correct location before finalizing."

	slashCommandsMandate := "SLASH COMMANDS & IN-CHAT COMPONENT WIDGETS MANDATE:\n" +
		"CRITICAL WIDGET PLACEMENT MANDATE:\n" +
		"• Keep all text responses extremely brief (1-2 sentences max) when rendering widgets. DO NOT list design presets, aspect ratio bullet points, or instructions in text as the widget handles all options visually.\n" +
		"• Always place the widget tag ([WIDGET:ADD_GUEST], [WIDGET:GENERATE_INVITATION], [WIDGET:ADD_ITINERARY]) at the VERY END of your response.\n" +
		"• /summarize: Present a brief executive summary of current event profile, guest counts, itinerary sessions, and design concepts.\n" +
		"• /add-guests: Output a brief 1-line greeting and append [WIDGET:ADD_GUEST] at the very end.\n" +
		"• /schedule: Output a brief 1-line greeting and append [WIDGET:ADD_ITINERARY] at the very end.\n" +
		"• /generate-invitation: Output a brief 1-line greeting and append [WIDGET:GENERATE_INVITATION] at the very end."

	// 1. Master Event Planner Agent
	reg.EventPlannerAgent = genkitx.DefineAgent(g, "eventPlannerAgent",
		aix.InlinePrompt{
			ai.WithModelName(DefaultModelName),
			ai.WithSystem(
				fmt.Sprintf(
					"You are the Master Event Planner Agent for Shubh Plan Web. You leverage Gemini text intelligence (%s) to orchestrate event creation, venue logistics, guest rosters, and itineraries. "+
						"%s\n%s\n%s\n%s\n%s",
					DefaultModelName, securityPrivacyMandate, clientWidgetsMandate, slashCommandsMandate, generalOnboardingRule, invitationStudioRule,
				),
			),
			ai.WithTools(toolRefList...),
		},
		aix.WithSessionStore(sessionStore),
	)

	// 2. Guest Concierge Agent
	reg.GuestConciergeAgent = genkitx.DefineAgent(g, "guestConciergeAgent",
		aix.InlinePrompt{
			ai.WithModelName(DefaultModelName),
			ai.WithSystem(
				fmt.Sprintf(
					"You are the Guest Concierge Agent for Shubh Plan Web. You manage guest rosters, dietary restrictions, RSVP tracking, and VIP escalation. "+
						"%s\n%s\n%s",
					securityPrivacyMandate, clientWidgetsMandate, generalOnboardingRule,
				),
			),
			ai.WithTools(toolRefList...),
		},
		aix.WithSessionStore(sessionStore),
	)

	// 3. Invitation Studio Agent
	reg.InvitationStudioAgent = genkitx.DefineAgent(g, "invitationStudioAgent",
		aix.InlinePrompt{
			ai.WithModelName(DefaultModelName),
			ai.WithSystem(
				fmt.Sprintf(
					"You are the Invitation Studio Agent for Shubh Plan Web. You specialize in designing luxury visual invitation cards. "+
						"%s\n%s\n%s",
					securityPrivacyMandate, clientWidgetsMandate, invitationStudioRule,
				),
			),
			ai.WithTools(toolRefList...),
		},
		aix.WithSessionStore(sessionStore),
	)

	// 4. Itinerary & Logistics Agent
	reg.ItineraryAgent = genkitx.DefineAgent(g, "itineraryLogisticsAgent",
		aix.InlinePrompt{
			ai.WithModelName(DefaultModelName),
			ai.WithSystem(
				fmt.Sprintf(
					"You are the Itinerary & Logistics Agent for Shubh Plan Web. You structure ceremony agendas, venue stage timelines, and dress codes. "+
						"%s\n%s\n%s",
					securityPrivacyMandate, clientWidgetsMandate, generalOnboardingRule,
				),
			),
			ai.WithTools(toolRefList...),
		},
		aix.WithSessionStore(sessionStore),
	)

	log.Printf("[Genkit Engine] Successfully registered 4 Genkit Agents with interactive client UI widget awareness targeting %s & %s", DefaultModelName, DefaultImageModelName)
	return reg
}
