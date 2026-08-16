package generator

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type PromptConcept struct {
	ID      int      `json:"id"`
	Title   string   `json:"title"`
	Style   string   `json:"style"`
	Palette []string `json:"palette"`
	Prompt  string   `json:"prompt"`
}

// BasicBuilder is the open-source community implementation of PromptBuilder.
type BasicBuilder struct{}

// NewBasicBuilder initializes a new community prompt builder.
func NewBasicBuilder() *BasicBuilder {
	return &BasicBuilder{}
}

func (b *BasicBuilder) ReadLocalContext(dataDir string) (string, string) {
	guests, err := os.ReadFile("guests.md")
	if err != nil {
		guests, _ = os.ReadFile(dataDir + "/guests.md")
	}

	details, err := os.ReadFile("event_details.md")
	if err != nil {
		details, _ = os.ReadFile(dataDir + "/event_details.md")
	}

	return string(guests), string(details)
}

func (b *BasicBuilder) CompilePrompts(eventType, hosts, welcome, style string) []PromptConcept {
	if hosts == "" {
		hosts = "Ananya & Rohan"
	}
	if welcome == "" {
		welcome = fmt.Sprintf("We joyfully invite you to celebrate the wedding of %s.", hosts)
	}

	styleName := "South Indian Royal Gold"
	switch style {
	case "mughal":
		styleName = "Mughal Heritage & Floral"
	case "paper_cut":
		styleName = "Paper Cut Craftwork"
	case "clay_3d":
		styleName = "Modern Clay 3D Render"
	case "minimalist":
		styleName = "Minimalist Gold Leaf"
	}

	return []PromptConcept{
		{
			ID:      1,
			Title:   "Royal Mandap & Temple Pillars Concept",
			Style:   styleName,
			Palette: []string{"#D4AF37", "#800020", "#FFFDD0"},
			Prompt:  fmt.Sprintf("Standalone invitation card artwork celebrating the event of %s. Traditional mandap with carved temple pillars, marigold garlands, brass lamps, rich gold embroidery border on cream parchment background.", hosts),
		},
		{
			ID:      2,
			Title:   "Jasmine Arch & Silk Drapery Concept",
			Style:   "Elegance Edition",
			Palette: []string{"#FFFDD0", "#D4AF37", "#E63946"},
			Prompt:  fmt.Sprintf("Luxurious invitation card for %s. %s Archway woven with fresh white jasmine and red roses, royal silk drapes in deep maroon, embossed gold foil typography.", hosts, strings.TrimSuffix(welcome, ".")),
		},
		{
			ID:      3,
			Title:   "Ornate Peacock & Geometrical Motif Concept",
			Style:   "Heritage Fine Art",
			Palette: []string{"#800020", "#FFD700", "#004225"},
			Prompt:  fmt.Sprintf("Traditional ceremony invitation artwork for %s. Symmetrical peacock motifs, ornate hand-drawn kolam background, deep ruby red and antique gold color palette.", hosts),
		},
		{
			ID:      4,
			Title:   "Minimalist Lotus & Geometrical Frame Concept",
			Style:   "Modern Minimal",
			Palette: []string{"#111827", "#F59E0B", "#F3F4F6"},
			Prompt:  fmt.Sprintf("Clean modern invitation design for %s. Blooming pink lotus flower illustration at base, gold geometric border, soft blush cream background, elegant serif typography hierarchy.", hosts),
		},
	}
}

// Compile constructs a standard clean prompt for event design generation defaulting to 9:16 aspect ratio.
func (b *BasicBuilder) Compile(eventDetails string, welcomeMessage string) ResponsePayload {
	return b.CompileWithAspect(eventDetails, welcomeMessage, "9:16")
}

// CompileWithAspect constructs a prompt tailored to the requested aspect ratio layout.
func (b *BasicBuilder) CompileWithAspect(eventDetails string, welcomeMessage string, aspect string) ResponsePayload {
	return b.CompileStructured(EventData{
		VisualPrompt:   eventDetails,
		WelcomeMessage: welcomeMessage,
		Aspect:         aspect,
	})
}

// CompileStructured compiles structured event details (matching agents-adk) into target prompt payload.
func (b *BasicBuilder) CompileStructured(data EventData) ResponsePayload {
	aspect := data.Aspect
	if aspect == "" {
		aspect = "9:16"
	}

	aspectInstruction := "aspect ratio 9:16 vertical poster format"
	switch aspect {
	case "4:5":
		aspectInstruction = "aspect ratio 4:5 vertical portrait format"
	case "1:1":
		aspectInstruction = "aspect ratio 1:1 square format"
	case "16:9":
		aspectInstruction = "aspect ratio 16:9 landscape banner format"
	case "9:16":
		aspectInstruction = "aspect ratio 9:16 vertical poster format"
	}

	negativeNoUIInstruction := "standalone physical invitation card graphic artwork, no smartphone UI, no mobile status bar, no clock or battery status bar, no screen bezels"
	noPromptTextInstruction := "CRITICAL PRINTING MANDATE: Render ONLY the exact text specified in MANDATORY TEXT inside the central card plate. Do NOT print any meta-descriptions, prompt text, camera instructions, or extraneous labels anywhere on the image."

	// Format Title / Hosts with occasion context
	names := strings.TrimSpace(data.HostNames)
	eventType := strings.TrimSpace(data.EventType)

	if names != "" && eventType != "" && !strings.Contains(strings.ToLower(names), strings.ToLower(eventType)) {
		switch strings.ToLower(eventType) {
		case "birthday":
			names = fmt.Sprintf("Birthday Celebration of %s", names)
		case "naming_ceremony", "naming ceremony", "naming":
			names = fmt.Sprintf("Naming Ceremony of %s", names)
		case "baby_shower", "baby shower", "seemantham":
			names = fmt.Sprintf("Baby Shower for %s", names)
		case "housewarming", "griha pravesh":
			names = fmt.Sprintf("Housewarming Celebration of %s", names)
		case "wedding":
			names = fmt.Sprintf("Wedding of %s", names)
		default:
			names = fmt.Sprintf("%s Celebration of %s", eventType, names)
		}
	} else if names == "" {
		if data.VisualPrompt != "" {
			names = extractCardTextFromPrompt(data.VisualPrompt)
		}
		if names == "" || names == "Auspicious Event Celebration" {
			if eventType != "" {
				names = fmt.Sprintf("%s Celebration", eventType)
			} else {
				names = "Auspicious Event Celebration"
			}
		}
	}

	displayTitle := names

	// Format Date & Venue
	dateStr := strings.TrimSpace(data.EventDate)
	venueStr := strings.TrimSpace(data.Venue)
	welcomeMsg := strings.TrimSpace(data.WelcomeMessage)

	if welcomeMsg == "" && eventType != "" {
		switch strings.ToLower(eventType) {
		case "birthday":
			welcomeMsg = "Let us celebrate a wonderful milestone"
		case "naming_ceremony", "naming ceremony", "naming":
			welcomeMsg = "Join us as we name our little angel"
		case "baby_shower", "baby shower":
			welcomeMsg = "Blessed with love, arriving with joy"
		case "housewarming", "griha pravesh":
			welcomeMsg = "New home, new beginnings, endless blessings"
		case "wedding":
			welcomeMsg = "Together with our families, we invite you"
		}
	}

	dateVenueParts := []string{}
	if dateStr != "" {
		dateVenueParts = append(dateVenueParts, dateStr)
	}
	if venueStr != "" {
		dateVenueParts = append(dateVenueParts, venueStr)
	}
	dateVenueStr := strings.Join(dateVenueParts, " | ")

	// Build 3-line mandatory typography mandate matching agents-adk
	var mandatoryTextSpec string
	if welcomeMsg != "" && dateVenueStr != "" {
		mandatoryTextSpec = fmt.Sprintf("MANDATORY TYPOGRAPHY TO RENDER IN CENTRAL PLATE (3-LINE HIERARCHY): Line 1 (Main Title): '%s'. Line 2 (Secondary Welcome Subheader): '%s'. Line 3 (Date & Location): '%s'.", names, welcomeMsg, dateVenueStr)
	} else if welcomeMsg != "" {
		mandatoryTextSpec = fmt.Sprintf("MANDATORY TYPOGRAPHY TO RENDER IN CENTRAL PLATE (2-LINE HIERARCHY): Line 1 (Main Title): '%s'. Line 2 (Secondary Welcome Subheader): '%s'.", names, welcomeMsg)
	} else if dateVenueStr != "" {
		mandatoryTextSpec = fmt.Sprintf("MANDATORY TYPOGRAPHY TO RENDER IN CENTRAL PLATE (2-LINE HIERARCHY): Line 1 (Main Title): '%s'. Line 2 (Date & Location): '%s'.", names, dateVenueStr)
	} else {
		mandatoryTextSpec = fmt.Sprintf("MANDATORY TYPOGRAPHY TO RENDER IN CENTRAL PLATE: '%s'.", names)
	}

	sanitizedVisualPrompt := sanitizeVisualPromptBody(data.VisualPrompt)
	if sanitizedVisualPrompt == "" || sanitizedVisualPrompt == "Bespoke event invitation artwork with elegant central card plate, ornate decorative borders, and soft studio lighting" {
		if eventType != "" {
			sanitizedVisualPrompt = fmt.Sprintf("Bespoke %s invitation graphic artwork with ornate decorative borders, translucent central label plate, and soft studio lighting", eventType)
		}
	}

	corePrompt := fmt.Sprintf(
		"Visual theme: %s, %s, %s. %s %s",
		sanitizedVisualPrompt,
		negativeNoUIInstruction,
		aspectInstruction,
		mandatoryTextSpec,
		noPromptTextInstruction,
	)

	return ResponsePayload{
		CorePrompt:     corePrompt,
		DisplayTitle:   displayTitle,
		WelcomeMessage: welcomeMsg,
		Aspect:         aspect,
	}
}

func sanitizeVisualPromptBody(prompt string) string {
	cleaned := prompt

	// Strip MANDATORY EVENT DETAILS marker if present
	if idx := strings.Index(strings.ToLower(cleaned), "mandatory event details:"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	// 1. Remove structural section headers like "1. Composition:", "2. Typography:", "Aesthetics:"
	reHeader := regexp.MustCompile(`(?i)^\s*\d+[\.\)]?\s*(Composition|Typography|Aesthetics):\s*`)
	cleaned = reHeader.ReplaceAllString(cleaned, "")

	// 2. Strip camera shot & framing meta-phrases
	metaRegexes := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^A\s+full\s+and\s+complete\s+shot\s+of\s+(an?\s+)?(elegant\s+)?(wedding\s+)?(invitation\s+card\s+)?(styled\s+in\s+)?`),
		regexp.MustCompile(`(?i)\bA\s+full\s+and\s+complete\s+shot\s+of\s+(an?\s+)?(elegant\s+)?(wedding\s+)?inv(itation)?\b`),
		regexp.MustCompile(`(?i)\bFull\s+view\s+of\s+a\s+complete\s+(rectangular\s+)?(wedding\s+)?invitation\s+card(\s+with\s+generous\s+margin\s+padding\s+around\s+all\s+four\s+borders)?,?\s*(styled\s+in)?`),
		regexp.MustCompile(`(?i)\bFull\s+view\s+complete\s+invitation\s+card\s+illustration\s+for\b`),
		regexp.MustCompile(`(?i)\bFull\s+view\s+of\s+a\s+complete\b`),
		regexp.MustCompile(`(?i)\bA\s+complete\s+shot\s+of\b`),
		regexp.MustCompile(`(?i)\b(Composition|Typography|Aesthetics):\s*`),
	}
	for _, re := range metaRegexes {
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	// 3. Strip typography instruction sentences like "Centered high-contrast legible event typography with text '...'"
	reTypo := regexp.MustCompile(`(?i)\bCentered\s+high-contrast\s+legible\s+(event\s+)?typography\s+(with\s+text\s*["'“][^"'”]+["'”])?\s*(on\s+a\s+[a-z\s]+plate)?\.?\b`)
	cleaned = reTypo.ReplaceAllString(cleaned, "")

	// 4. Strip leftover quoted text specifications from visual body to prevent model reading text twice
	reQuotes := regexp.MustCompile(`(?i)(?:with\s+the\s+text|with\s+text|text)\s*["'“][^"'”]+["'”]`)
	cleaned = reQuotes.ReplaceAllString(cleaned, "")

	// 5. Clean up extra punctuation/spaces
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimPrefix(cleaned, ",")
	cleaned = strings.TrimPrefix(cleaned, ".")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		cleaned = "Bespoke event invitation artwork with elegant central card plate, ornate decorative borders, and soft studio lighting"
	}
	return cleaned
}

func extractCardTextFromPrompt(prompt string) string {
	markerRegex := regexp.MustCompile(`(?i)(?:MANDATORY\s+EVENT\s+DETAILS|Event\s+details):\s*([^\.]+?)(?:\.|$|\n)`)
	matchesMarker := markerRegex.FindStringSubmatch(prompt)
	if len(matchesMarker) > 1 && strings.TrimSpace(matchesMarker[1]) != "" {
		res := stripMetaPhrases(strings.TrimSpace(matchesMarker[1]))
		if res != "" {
			return res
		}
	}

	quotedRegex := regexp.MustCompile(`(?:text|with the text|titled)\s*["'“]([^"'”]+)["'”]`)
	matches := quotedRegex.FindStringSubmatch(prompt)
	if len(matches) > 1 && strings.TrimSpace(matches[1]) != "" {
		res := stripMetaPhrases(strings.TrimSpace(matches[1]))
		if res != "" {
			return res
		}
	}

	forRegex := regexp.MustCompile(`(?:for|of)\s+([A-Za-z0-9\s&',.-]+?)(?:\s+(?:rendered|styled|featuring|with|in\s+[A-Z]|on\s+a|central)|$)`)
	matchesFor := forRegex.FindStringSubmatch(prompt)
	if len(matchesFor) > 1 && strings.TrimSpace(matchesFor[1]) != "" {
		result := strings.TrimSpace(matchesFor[1])
		if len(result) > 3 && len(result) < 80 {
			res := stripMetaPhrases(result)
			if res != "" && !isVisualDescriptionText(res) {
				return res
			}
		}
	}

	cleaned := stripMetaPhrases(prompt)
	if !isVisualDescriptionText(cleaned) && len(cleaned) > 3 && len(cleaned) < 80 {
		return cleaned
	}

	return "Auspicious Event Celebration"
}

func isVisualDescriptionText(text string) bool {
	lower := strings.ToLower(text)
	keywords := []string{
		"vintage", "regal", "background", "parchment", "texture", "arch", "gold foil",
		"sage green", "leaves", "lotus", "floral", "mughal", "plate", "glowing",
		"haze", "aesthetic", "silhouettes", "filigree", "watercolor", "clay", "pop art",
		"paper cut", "craft paper", "studio lighting", "artwork", "composition",
		"banner", "minimalist", "south indian", "crimson", "pastel", "halftone",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func stripMetaPhrases(text string) string {
	phrases := []string{
		"a full and complete shot of an elegant",
		"a full and complete shot of",
		"a complete shot of",
		"a complete wedding invitation card",
		"complete wedding invitation card",
		"wedding invitation card",
		"invitation card",
		"full view of a complete",
		"full view complete",
		"full view of",
		"full view",
		"graphic illustration",
		"illustration for",
		"artwork for",
		"composition for",
		"centered high-contrast legible event typography",
		"centered high-contrast legible typography",
	}
	cleaned := text
	for _, p := range phrases {
		re := regexp.MustCompile("(?i)\\b" + regexp.QuoteMeta(p) + "\\b")
		cleaned = re.ReplaceAllString(cleaned, "")
	}
	return strings.TrimSpace(cleaned)
}
