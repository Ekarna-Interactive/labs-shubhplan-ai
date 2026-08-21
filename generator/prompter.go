package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
)

// GetStrictEnvModels returns models defined strictly in environment variables (no hardcoded fallback models in code)
func GetStrictEnvModels() []string {
	config.LoadConfig()
	var models []string
	primary := strings.TrimSpace(os.Getenv("GEMINI_TEXT_MODEL"))
	fallback := strings.TrimSpace(os.Getenv("GEMINI_FALLBACK_TEXT_MODEL"))

	if primary != "" {
		models = append(models, primary)
	}
	if fallback != "" && fallback != primary {
		models = append(models, fallback)
	}
	return models
}

// GetStrictEnvImageModels returns image models defined strictly in environment variables
func GetStrictEnvImageModels() []string {
	config.LoadConfig()
	var models []string
	primary := strings.TrimSpace(os.Getenv("GEMINI_IMAGE_MODEL"))
	fallback := strings.TrimSpace(os.Getenv("GEMINI_FALLBACK_IMAGE_MODEL"))

	if primary != "" {
		models = append(models, primary)
	}
	if fallback != "" && fallback != primary {
		models = append(models, fallback)
	}

	return models
}

type PromptIdea struct {
	ThemeTitle string `json:"themeTitle"`
	PromptText string `json:"promptText"`
	Style      string `json:"style"`
}

// GenerateAIPromptIdeas calls Gemini LLM to generate 4 distinct prompt concepts with theme titles & visual prompts
func GenerateAIPromptIdeas(apiKey string, eventType string, style string) ([]PromptIdea, error) {
	eType := strings.TrimSpace(eventType)
	if eType == "" {
		eType = "Auspicious Event Celebration"
	}
	if style == "" {
		style = "South Indian Royal Gold"
	}

	if apiKey == "" {
		return GenerateFallbackPromptIdeas(eType, style), fmt.Errorf("GEMINI_API_KEY is not set. Using offline fallback prompt ideas.")
	}

	promptMsg := fmt.Sprintf(`You are an expert AI invitation designer agent.
EVENT TYPE: %s
STYLE: %s

Generate EXACTLY 4 distinct, highly creative invitation design concept ideas in the requested '%s' style for a '%s'.
For each concept option, provide a catchy 2-4 word "themeTitle" (e.g. "Pastel Celestial Cradle", "Golden Marigold Archway", "Royal Mandap Cutwork") and a detailed visual "promptText" describing the background scene.

Return JSON array format:
[
  {
    "themeTitle": "Creative Theme Name",
    "promptText": "Detailed visual background scene prompt text..."
  }
]`, eType, style, style, eType)

	modelsToTry := GetStrictEnvModels()
	for _, modelName := range modelsToTry {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)

		payloadMap := map[string]interface{}{
			"contents": []map[string]interface{}{
				{"parts": []map[string]interface{}{{"text": promptMsg}}},
			},
			"generationConfig": map[string]interface{}{
				"temperature": 0.9,
			},
		}

		bodyBytes, _ := json.Marshal(payloadMap)
		req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 12 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			rawBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var resStruct struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}
			if err := json.Unmarshal(rawBytes, &resStruct); err == nil && len(resStruct.Candidates) > 0 {
				txt := resStruct.Candidates[0].Content.Parts[0].Text
				txt = strings.TrimPrefix(strings.TrimSpace(txt), "```json")
				txt = strings.TrimPrefix(txt, "```")
				txt = strings.TrimSuffix(txt, "```")
				txt = strings.TrimSpace(txt)

				var ideas []PromptIdea
				if err := json.Unmarshal([]byte(txt), &ideas); err == nil && len(ideas) >= 4 {
					for i := range ideas {
						ideas[i].Style = style
					}
					return ideas[:4], nil
				}
			}
		} else {
			resp.Body.Close()
		}
	}

	return GenerateFallbackPromptIdeas(eType, style), nil
}

func getThemeTitlesForStyle(style string, eventType string) []string {
	s := strings.ToLower(style)
	e := strings.ToLower(eventType)

	if strings.Contains(s, "paper_cut") || strings.Contains(s, "paper cut") {
		if strings.Contains(e, "naming") {
			return []string{"Pastel Celestial Cradle", "Marigold Garland Layer", "Golden Constellation Arch", "Gentle Starry Cloudscape"}
		} else if strings.Contains(e, "wedding") {
			return []string{"Royal Mandap Cutwork", "Peacock & Lotus Lattice", "Terracotta & Gold Layers", "Floral Jharokha Silhouette"}
		} else if strings.Contains(e, "birthday") {
			return []string{"Festive Streamer Canopy", "Confetti Balloon Layer", "Playful Party Bunting", "Starry Celebration Frame"}
		}
		return []string{"Layered Paper Craftwork", "Gentle Shadow Silhouette", "Ornate Paper Border", "Pastel Drop Shadow Cut"}
	}

	if strings.Contains(s, "south_indian") || strings.Contains(s, "south indian") {
		return []string{"Temple Filigree & Lotus Mandap", "Royal Jharokha Entrance", "Golden Diya & Jasmine Toran", "Silken Kanjivaram Border"}
	}

	if strings.Contains(s, "mughal") {
		return []string{"Floral Lattice & Peacock Arch", "Emerald & Gold Minaret", "Royal Garden Evening Glow", "Ornate Marble Inlay"}
	}

	if strings.Contains(s, "clay") {
		return []string{"Soft Clay Pastel Canopy", "Tactile Terracotta Haven", "Matte Sculpted Flora", "Charming Clay Balloon Fest"}
	}

	return []string{"Subtle Foil Leaf Crest", "Monochrome Botanical Trace", "Elegant Linen & Gold Edge", "Geometric Gold Halo"}
}

func GenerateFallbackPromptIdeas(eventType string, style string) []PromptIdea {
	prompts := GenerateFallbackPrompts(eventType, style)
	titles := getThemeTitlesForStyle(style, eventType)

	ideas := make([]PromptIdea, 4)
	for i := 0; i < 4; i++ {
		tTitle := fmt.Sprintf("Design Theme %d", i+1)
		if i < len(titles) {
			tTitle = titles[i]
		}
		pText := fmt.Sprintf("Bespoke %s invitation artwork for %s.", style, eventType)
		if i < len(prompts) {
			pText = prompts[i]
		}
		ideas[i] = PromptIdea{
			ThemeTitle: tTitle,
			PromptText: pText,
			Style:      style,
		}
	}
	return ideas
}

// GenerateAIPromptSuggestions uses Gemini LLM to dynamically generate 4 creative prompts tailored to event details and style.
func GenerateAIPromptSuggestions(apiKey string, eventType string, style string) ([]string, error) {
	return GenerateAIPromptSuggestionsWithModel(apiKey, os.Getenv("GEMINI_TEXT_MODEL"), eventType, style)
}

func GenerateAIPromptSuggestionsWithModel(apiKey string, textModel string, eventType string, style string) ([]string, error) {
	eType := strings.TrimSpace(eventType)
	if eType == "" {
		eType = "Auspicious Event Celebration"
	}
	if style == "" {
		style = "South Indian Traditional"
	}

	if apiKey == "" {
		return GenerateFallbackPrompts(eType, style), fmt.Errorf("GEMINI_API_KEY is not set. Using offline fallback prompts.")
	}

	var motifInstruction string
	switch strings.ToLower(eType) {
	case "birthday":
		motifInstruction = "CRITICAL EVENT OCCASION DIRECTIVE: The visual artwork MUST explicitly convey a festive Birthday Celebration by including iconic birthday visual motifs (such as celebratory balloon garlands, paper bunting streamers, festive cake/candle accents, party ribbon layers, or star sparkles) integrated seamlessly into the requested style."
	case "naming_ceremony", "naming ceremony", "naming":
		motifInstruction = "CRITICAL EVENT OCCASION DIRECTIVE: The visual artwork MUST explicitly convey a Naming Ceremony by including tender baby celebration motifs (such as a soft wooden cradle silhouette, star constellations, gentle marigold garlands, or baby footprints) integrated seamlessly into the requested style."
	case "baby_shower", "baby shower", "seemantham":
		motifInstruction = "CRITICAL EVENT OCCASION DIRECTIVE: The visual artwork MUST explicitly convey a Baby Shower by including motherly & baby celebration motifs (such as a floral Jhula swing, lotus blooms, pastel balloon accents, or delicate foliage arches) integrated seamlessly into the requested style."
	case "housewarming", "griha pravesh":
		motifInstruction = "CRITICAL EVENT OCCASION DIRECTIVE: The visual artwork MUST explicitly convey a Housewarming / Griha Pravesh by including auspicious new home motifs (such as a traditional carved entrance door arch, brass Kalash with mango leaves, or vibrant marigold toran garland) integrated seamlessly into the requested style."
	case "wedding":
		motifInstruction = "CRITICAL EVENT OCCASION DIRECTIVE: The visual artwork MUST explicitly convey a Wedding Celebration by including wedding motifs (such as a royal Jharokha archway, lotus & peacock filigree, or golden mandap accents) integrated seamlessly into the requested style."
	default:
		motifInstruction = fmt.Sprintf("CRITICAL EVENT OCCASION DIRECTIVE: The visual artwork MUST explicitly convey a %s by incorporating appropriate celebratory motifs for the occasion integrated seamlessly into the requested style.", eType)
	}

	promptMsg := fmt.Sprintf(`You are an expert AI invitation prompter agent for boutique events (Weddings, Naming Ceremonies, Baby Showers, Housewarmings, Birthdays, Anniversaries, Corporate Galas).

EVENT TYPE: %s
TARGET DESIGN AESTHETIC STYLE: %s

%s

CRITICAL STYLE MANDATE:
All 4 generated prompts MUST strictly use and explore the requested TARGET DESIGN AESTHETIC STYLE ('%s').
Do NOT include, mention, or mix in any other unrelated design styles (e.g. if 'Paper Cut Art' is requested, do NOT mention pop art, clay 3D, or watercolor in any of the prompts).
Every prompt must be a unique, highly creative variation entirely inside the requested '%s' style for a '%s'.

CRITICAL FORMATTING MANDATE:
Each prompt must describe ONLY visual background scene aesthetics (background textures, color swatches, ornate borders, motifs, lighting, translucent central label plate) for a '%s' in '%s' style.
ABSOLUTELY DO NOT include host names, dates, venues, locations, or literal text instructions inside the prompt string.

Output format requirement: Return EXACTLY 4 numbered lines without introduction or markdown formatting:
1. <Prompt 1>
2. <Prompt 2>
3. <Prompt 3>
4. <Prompt 4>`, eType, style, motifInstruction, style, style, eType, eType, style)

	// Models strictly loaded from environment variables (.env)
	modelsToTry := GetStrictEnvModels()
	if len(modelsToTry) == 0 {
		return GenerateFallbackPrompts(eType, style), fmt.Errorf("No Gemini text models configured in environment (GEMINI_TEXT_MODEL / GEMINI_FALLBACK_TEXT_MODEL)")
	}
	var lastErr error

	for _, modelName := range modelsToTry {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)

		payloadMap := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{"text": promptMsg},
					},
				},
			},
			"generationConfig": map[string]interface{}{
				"temperature": 0.9,
			},
		}

		bodyBytes, err := json.Marshal(payloadMap)
		if err != nil {
			lastErr = err
			continue
		}

		req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 12 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("%s status %d: %s", modelName, resp.StatusCode, string(body))
			continue
		}

		rawBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		var resStruct struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal(rawBytes, &resStruct); err != nil || len(resStruct.Candidates) == 0 {
			lastErr = fmt.Errorf("failed to parse %s response JSON", modelName)
			continue
		}

		fullText := ""
		for _, cand := range resStruct.Candidates {
			for _, part := range cand.Content.Parts {
				fullText += part.Text + "\n"
			}
		}

		suggestions := parseNumberedPrompts(fullText)
		if len(suggestions) >= 4 {
			return suggestions[:4], nil
		}
	}

	return GenerateFallbackPrompts(eType, style), lastErr
}

func parseNumberedPrompts(text string) []string {
	lines := strings.Split(text, "\n")
	res := []string{}
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		// Strip leading "1. ", "2. ", "1) ", etc.
		if len(trimmed) > 3 && (trimmed[0] >= '1' && trimmed[0] <= '9') && (trimmed[1] == '.' || trimmed[1] == ')') {
			p := strings.TrimSpace(trimmed[2:])
			if p != "" {
				res = append(res, p)
			}
		} else if len(trimmed) > 15 {
			res = append(res, trimmed)
		}
	}
	return res
}

// GenerateFallbackPrompts generates 4 variations strictly adhering to the requested style
func GenerateFallbackPrompts(eventType string, style string) []string {
	eType := strings.TrimSpace(eventType)
	if eType == "" {
		eType = "Auspicious Celebration Event"
	}
	if style == "" {
		style = "Paper Cut Art"
	}

	var motifDesc string
	switch strings.ToLower(eType) {
	case "birthday":
		motifDesc = "festive birthday balloon garlands, paper bunting streamers, party confetti accents"
	case "naming_ceremony", "naming ceremony", "naming":
		motifDesc = "soft wooden cradle silhouette, star constellations, gentle marigold garlands"
	case "baby_shower", "baby shower":
		motifDesc = "floral Jhula swing, lotus blooms, pastel ribbon accents"
	case "housewarming", "griha pravesh":
		motifDesc = "traditional carved entrance door arch, brass Kalash with mango leaves, marigold toran"
	case "wedding":
		motifDesc = "royal Jharokha archway, lotus & peacock filigree, golden mandap accents"
	default:
		motifDesc = "ornate decorative borders, festive motif accents"
	}

	return []string{
		fmt.Sprintf("Bespoke %s invitation artwork for %s featuring %s, central glowing translucent card plate, multi-layered intricate borders, soft drop shadows, clean studio lighting.", style, eType, motifDesc),
		fmt.Sprintf("Elegant %s visual design for %s featuring %s, multi-tiered layered paper silhouettes, floral corner filigree, rich pastel color palette, centered translucent label plate.", style, eType, motifDesc),
		fmt.Sprintf("Ornate %s visual theme for %s with %s, textured craft elements with depth separation, royal gold accent borders, softly illuminated central label plate.", style, eType, motifDesc),
		fmt.Sprintf("Modern premium %s invitation background for %s incorporating %s, clean geometric decorative layers, glowing ambient backdrop, crisp studio border framing.", style, eType, motifDesc),
	}
}

// GenerateAIWelcomeSuggestions calls Gemini LLM to generate 4 contrasting welcome message subheader suggestions for the event type.
func GenerateAIWelcomeSuggestions(apiKey string, eventType string) ([]string, error) {
	eType := strings.TrimSpace(eventType)
	if eType == "" {
		eType = "Auspicious Event Celebration"
	}

	if apiKey == "" {
		return GenerateFallbackWelcomeSubheaders(eType), fmt.Errorf("GEMINI_API_KEY is not set. Using offline fallback welcome subheaders.")
	}

	promptMsg := fmt.Sprintf(`You are an expert event invitation copywriter. Generate 4 contrasting, highly elegant secondary welcome message subheader phrasing options for a '%s' across 4 distinct styles:
1. Family-focused (Warm, emotional, family-centered invitation subheader)
2. Poetic & Celebratory (Graceful, elegant phrasing)
3. Modern & Direct (Clean, punchy contemporary greeting)
4. Auspicious & Traditional (Customary blessing request & traditional wording)

CRITICAL MANDATE: Return ONLY short, elegant single-sentence subheader phrases (max 12 words per line) suitable for an invitation card subheader.
DO NOT include quotation marks, intro headers, or bullet prefixes. Return EXACTLY 4 numbered lines:
1. <Subheader 1>
2. <Subheader 2>
3. <Subheader 3>
4. <Subheader 4>`, eType)

	modelsToTry := GetStrictEnvModels()
	if len(modelsToTry) == 0 {
		return GenerateFallbackWelcomeSubheaders(eType), fmt.Errorf("No Gemini text models configured in environment (GEMINI_TEXT_MODEL / GEMINI_FALLBACK_TEXT_MODEL)")
	}
	var lastErr error

	for _, modelName := range modelsToTry {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)

		payloadMap := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{"text": promptMsg},
					},
				},
			},
			"generationConfig": map[string]interface{}{
				"temperature": 0.8,
			},
		}

		bodyBytes, err := json.Marshal(payloadMap)
		if err != nil {
			lastErr = err
			continue
		}

		req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 12 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("%s status %d: %s", modelName, resp.StatusCode, string(body))
			continue
		}

		rawBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		var resStruct struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal(rawBytes, &resStruct); err != nil || len(resStruct.Candidates) == 0 {
			lastErr = fmt.Errorf("failed to parse %s response JSON", modelName)
			continue
		}

		fullText := ""
		for _, cand := range resStruct.Candidates {
			for _, part := range cand.Content.Parts {
				fullText += part.Text + "\n"
			}
		}

		suggestions := parseNumberedPrompts(fullText)
		if len(suggestions) >= 4 {
			return suggestions[:4], nil
		}
	}

	return GenerateFallbackWelcomeSubheaders(eType), lastErr
}

// GenerateFallbackWelcomeSubheaders provides 4 pre-curated subheader options matching PROMPT_TEMPLATES.json
func GenerateFallbackWelcomeSubheaders(eventType string) []string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "naming_ceremony", "naming ceremony", "naming":
		return []string{
			"Together with our families, join us as we name our little angel",
			"With happy hearts, join us to bless our baby boy",
			"Blessed with love, arriving with joy — Come celebrate with us",
			"Seeking your presence and warm blessings for our child's Namakaran",
		}
	case "baby_shower", "baby shower", "seemantham":
		return []string{
			"Blessed with love, arriving with joy",
			"Surrounding our glowing mom-to-be with love and blessings",
			"A baby is on the way, let us shower them with love",
			"Join us to celebrate new beginnings and motherly joy",
		}
	case "housewarming", "griha pravesh":
		return []string{
			"New home, new beginnings, endless blessings",
			"Warm hearts and open doors — Welcome to our new abode",
			"New home, new memories, endless love and blessings",
			"With divine blessings, join us to celebrate our Griha Pravesh",
		}
	case "birthday":
		return []string{
			"Let us celebrate a wonderful milestone together",
			"Laughter, joy, and cheers — Join us to celebrate!",
			"Another year of wonderful memories and bright dreams",
			"Please join us for an evening of festive birthday joy",
		}
	default:
		return []string{
			"Together with our families, we request the pleasure of your company",
			"Together with happy families, we request your warm presence",
			"Two hearts, two souls, one eternal celebration",
			"Seeking your auspicious presence and blessings on our special day",
		}
	}
}

// GenerateAIChatResponse handles conversational agent chat responses using Gemini 3.5 Flash LLM.
func GenerateAIChatResponse(apiKey string, plannerName string, userMsg string, eventContext string) (string, error) {
	if apiKey == "" {
		if strings.Contains(eventContext, "No active event details") || eventContext == "" {
			return fmt.Sprintf("Hello %s! Welcome to Shubh Plan AI. Since we don't have an active event set up yet, we can get started by configuring your event details using the /event command.", plannerName), nil
		}
		return fmt.Sprintf("Hello %s! Welcome to Shubh Plan AI. I am your Master Orchestrator assistant. Active event context loaded: %s. How can I help you plan your celebration today?", plannerName, eventContext), nil
	}

	systemPrompt := fmt.Sprintf(`You are the Master Orchestrator AI Copilot for Shubh Plan AI, an expert event planning assistant.
The active user is '%s'.
The active event context is: '%s'.

Instructions:
- Provide concise, helpful, friendly, and professional advice on event planning, guest RSVPs, budgets, timelines, or invitation designs.
- You are fully aware of all Shubh Plan AI slash commands and their aliases:
  * Budget: /budget <amount> (aliases: /finance, /spend) - Set/view estimated budget.
  * RSVPs: /rsvp (aliases: /guests, /rsvps) - View guest list. /add-rsvp (aliases: /addrsvp, /new-rsvp) - Add guest RSVP.
  * Timeline: /timeline (aliases: /schedule, /itinerary) - View run-of-show itinerary.
  * Profile: /event (aliases: /profile, /details) - View/edit event_details.md. /currency <code> - Set currency. /welcome <msg> - Set subheader. /planner <name> - Set planner name.
  * Design: /generate (aliases: /design, /create), /style (aliases: /preset, /aesthetic), /aspect (aliases: /ratio, /res), /suggest (aliases: /ideas, /theme), /refine (aliases: /edit, /modify), /preview (aliases: /web).
  * System/Memory: /honcho (aliases: /memory, /cards), /config <key> (aliases: /key, /apikey), /wizard (aliases: /wiz), /clear (aliases: /cls), /help (aliases: /h, /?).
- When answering questions about setting budget, currency, design generation, or API keys:
  * Explain the status clearly and advise the user on which slash command to run.
  * If the user needs to set their Gemini API key, inform them to get a free key at https://aistudio.google.com/api-keys and run '/config <key>' or save it in .env.
  * If asked about budget status, explain if the benchmark budget is currently set and inform the user how to update it: "To update your budget: In Terminal, type /budget <amount> (e.g., /budget 300000 or /budget 2L) to set your custom budget, or update Total Budget in event_details.md."
- Keep responses under 5 sentences unless asked for detailed breakdowns.
- Do NOT generate raw image generation prompts unless explicitly asked for invitation design concepts.`, plannerName, eventContext)

	promptMsg := fmt.Sprintf("%s\n\nUser Message: %s", systemPrompt, userMsg)

	modelsToTry := GetStrictEnvModels()
	if len(modelsToTry) == 0 {
		return fmt.Sprintf("Hello %s! Active event context loaded: %s. (No text models configured in GEMINI_TEXT_MODEL / GEMINI_FALLBACK_TEXT_MODEL)", plannerName, eventContext), nil
	}

	for _, modelName := range modelsToTry {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)

		payloadMap := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{"text": promptMsg},
					},
				},
			},
			"generationConfig": map[string]interface{}{
				"temperature":     0.7,
				"maxOutputTokens": 1000,
			},
		}

		bodyBytes, err := json.Marshal(payloadMap)
		if err != nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			cancel()
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 12 * time.Second}
		resp, err := client.Do(req)
		cancel()
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}

		rawBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var resStruct struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal(rawBytes, &resStruct); err == nil && len(resStruct.Candidates) > 0 {
			for _, part := range resStruct.Candidates[0].Content.Parts {
				if strings.TrimSpace(part.Text) != "" {
					return strings.TrimSpace(part.Text), nil
				}
			}
		}
	}

	return fmt.Sprintf("Hello %s! As your Shubh Plan AI Copilot, I can help you structure budgets, record guest RSVPs, generate ceremony timelines, and compile invitation card concepts for '%s'!", plannerName, eventContext), nil
}

// VenueSuggestion represents a single venue autocomplete prediction item
type VenueSuggestion struct {
	PlaceID string `json:"placeId"`
	Text    string `json:"text"`
}

// GenerateAIVenueSuggestions uses Gemini LLM to dynamically generate 5 realistic venue recommendations tailored to query & event type
func GenerateAIVenueSuggestions(apiKey string, textModel string, eventType string, query string) ([]VenueSuggestion, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		q = "Banquet Hall"
	}
	if eventType == "" {
		eventType = "Wedding Celebration"
	}

	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}
	if textModel == "" {
		envModels := GetStrictEnvModels()
		if len(envModels) > 0 {
			textModel = envModels[0]
		}
	}

	promptMsg := fmt.Sprintf(`You are an expert AI Venue Concierge for event planning.
User Query: "%s"
Event Type: "%s"

Generate EXACTLY 5 real or highly realistic luxury event venue recommendations matching this query/location.
Output format requirement: Return JSON array containing objects with "placeId" and "text" fields ONLY. No markdown formatting, no code block backticks.
Example:
[
  {"placeId": "ai-venue-1", "text": "Palace Grounds, Jayamahal Road, Bengaluru, Karnataka"},
  {"placeId": "ai-venue-2", "text": "The Leela Palace, HAL Old Airport Rd, Kodihalli, Bengaluru"}
]`, q, eventType)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", textModel, apiKey)
	payloadMap := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": promptMsg},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.7,
		},
	}

	bodyBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini places API returned status: %d", resp.StatusCode)
	}

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var resStruct struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(rawBytes, &resStruct); err == nil && len(resStruct.Candidates) > 0 {
		for _, part := range resStruct.Candidates[0].Content.Parts {
			txt := strings.TrimSpace(part.Text)
			txt = strings.TrimPrefix(txt, "```json")
			txt = strings.TrimPrefix(txt, "```")
			txt = strings.TrimSuffix(txt, "```")
			txt = strings.TrimSpace(txt)

			var suggestions []VenueSuggestion
			if err := json.Unmarshal([]byte(txt), &suggestions); err == nil && len(suggestions) > 0 {
				return suggestions, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to parse AI venue suggestions")
}

// ResolveVenueQuery queries Google Places API (or Gemini AI Venue Agent / Fallback) to resolve venue details
func ResolveVenueQuery(query string, placesKey string, geminiKey string) (primary string, address string, mapsURL string, directionsURL string) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "Main Event Venue", "Venue Address pending configuration", "", ""
	}

	// 1. Google Places API (New) Search
	if placesKey != "" {
		reqURL := "https://places.googleapis.com/v1/places:searchText"
		payload := map[string]interface{}{"textQuery": q}
		pBytes, _ := json.Marshal(payload)

		req, err := http.NewRequestWithContext(context.Background(), "POST", reqURL, bytes.NewBuffer(pBytes))
		if err == nil {
			req.Header.Set("X-Goog-Api-Key", placesKey)
			req.Header.Set("X-Goog-FieldMask", "places.displayName,places.formattedAddress,places.googleMapsUri,places.id")
			req.Header.Set("Content-Type", "application/json")

			hc := &http.Client{Timeout: 5 * time.Second}
			if resp, err := hc.Do(req); err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				var searchRes struct {
					Places []struct {
						DisplayName struct {
							Text string `json:"text"`
						} `json:"displayName"`
						FormattedAddress string `json:"formattedAddress"`
						GoogleMapsURI    string `json:"googleMapsUri"`
						ID               string `json:"id"`
					} `json:"places"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&searchRes); err == nil && len(searchRes.Places) > 0 {
					p := searchRes.Places[0]
					primary = p.DisplayName.Text
					if primary == "" {
						primary = q
					}
					address = p.FormattedAddress
					mapsURL = p.GoogleMapsURI
					if mapsURL == "" {
						mapsURL = fmt.Sprintf("https://maps.google.com/?q=%s", url.QueryEscape(primary+", "+address))
					}
					directionsURL = fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%s", url.QueryEscape(primary+", "+address))
					return primary, address, mapsURL, directionsURL
				}
			}
		}
	}

	// 2. Gemini AI Venue Agent Fallback
	if geminiKey != "" {
		sug, err := GenerateAIVenueSuggestions(geminiKey, config.LoadConfig().GeminiTextModel, "Event", q)
		if err == nil && len(sug) > 0 {
			primary = q
			address = sug[0].Text
			mapsURL = fmt.Sprintf("https://maps.google.com/?q=%s", url.QueryEscape(q+", "+address))
			directionsURL = fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%s", url.QueryEscape(q+", "+address))
			return primary, address, mapsURL, directionsURL
		}
	}

	// 3. Fallback Format
	primary = q
	address = q
	mapsURL = fmt.Sprintf("https://maps.google.com/?q=%s", url.QueryEscape(q))
	directionsURL = fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%s", url.QueryEscape(q))
	return primary, address, mapsURL, directionsURL
}

// FetchVenueSuggestions returns 4-5 venue predictions using Google Places API, Gemini AI, or Curated directory
func FetchVenueSuggestions(query string, placesKey string, geminiKey string, eventType string) []VenueSuggestion {
	q := strings.TrimSpace(query)
	if q == "" {
		q = "Banquet Hall"
	}

	// 1. Google Places API (New) Autocomplete / SearchText
	if placesKey != "" {
		reqURL := "https://places.googleapis.com/v1/places:autocomplete"
		payload := map[string]interface{}{"input": q}
		pBytes, _ := json.Marshal(payload)

		req, err := http.NewRequestWithContext(context.Background(), "POST", reqURL, bytes.NewBuffer(pBytes))
		if err == nil {
			req.Header.Set("X-Goog-Api-Key", placesKey)
			req.Header.Set("Content-Type", "application/json")

			hc := &http.Client{Timeout: 5 * time.Second}
			if resp, err := hc.Do(req); err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				var resMap struct {
					Suggestions []struct {
						PlacePrediction struct {
							PlaceID string `json:"placeId"`
							Text    struct {
								Text string `json:"text"`
							} `json:"text"`
						} `json:"placePrediction"`
					} `json:"suggestions"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&resMap); err == nil && len(resMap.Suggestions) > 0 {
					var results []VenueSuggestion
					for _, item := range resMap.Suggestions {
						if item.PlacePrediction.Text.Text != "" {
							results = append(results, VenueSuggestion{
								PlaceID: item.PlacePrediction.PlaceID,
								Text:    item.PlacePrediction.Text.Text,
							})
						}
					}
					if len(results) > 0 {
						return results
					}
				}
			}
		}
	}

	// 2. Gemini AI Venue Agent
	if geminiKey != "" {
		sug, err := GenerateAIVenueSuggestions(geminiKey, config.LoadConfig().GeminiTextModel, eventType, q)
		if err == nil && len(sug) > 0 {
			return sug
		}
	}

	// 3. Fallback Curated Suggestions
	return []VenueSuggestion{
		{PlaceID: "place-1", Text: q + ", Main Road, Bengaluru, Karnataka"},
		{PlaceID: "place-2", Text: "The Leela Palace, HAL Old Airport Road, Kodihalli, Bengaluru"},
		{PlaceID: "place-3", Text: "Palace Grounds, Jayamahal Road, Bengaluru, Karnataka"},
		{PlaceID: "place-4", Text: "Taj West End, Race Course Road, Bengaluru, Karnataka"},
	}
}

// FetchPlaceDetails fetches full 7 venue details for a placeId matching web version /api/places/details
func FetchPlaceDetails(placeID string, placesKey string) config.VenueDetails {
	cleanID := strings.TrimSpace(placeID)
	defaultPhoto := "https://images.unsplash.com/photo-1519167758481-83f550bb49b3?auto=format&fit=crop&w=1200&q=80"

	if placesKey != "" && strings.HasPrefix(cleanID, "ChI") {
		reqURL := fmt.Sprintf("https://places.googleapis.com/v1/places/%s", cleanID)
		req, err := http.NewRequestWithContext(context.Background(), "GET", reqURL, nil)
		if err == nil {
			req.Header.Set("X-Goog-Api-Key", placesKey)
			req.Header.Set("X-Goog-FieldMask", "displayName,formattedAddress,googleMapsUri,adrFormatAddress,photos")

			hc := &http.Client{Timeout: 5 * time.Second}
			if resp, err := hc.Do(req); err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				var placeRes struct {
					DisplayName struct {
						Text string `json:"text"`
					} `json:"displayName"`
					FormattedAddress string `json:"formattedAddress"`
					GoogleMapsURI    string `json:"googleMapsUri"`
					AdrFormatAddress string `json:"adrFormatAddress"`
					Photos           []struct {
						Name string `json:"name"`
					} `json:"photos"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&placeRes); err == nil {
					primary := placeRes.DisplayName.Text
					if primary == "" {
						primary = placeRes.FormattedAddress
					}
					photoURL := defaultPhoto
					if len(placeRes.Photos) > 0 && placeRes.Photos[0].Name != "" {
						photoURL = fmt.Sprintf("https://places.googleapis.com/v1/%s/media?key=%s&maxHeightPx=600&maxWidthPx=800", placeRes.Photos[0].Name, placesKey)
					}

					return config.VenueDetails{
						PrimaryVenue:           primary,
						VenueFormattedAddress:  placeRes.FormattedAddress,
						VenueAdrFormatAddress:  placeRes.AdrFormatAddress,
						Address:                placeRes.FormattedAddress,
						GoogleMapURL:            placeRes.GoogleMapsURI,
						GoogleMapDirectionsURL: fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%s&destination_place_id=%s", url.QueryEscape(primary+", "+placeRes.FormattedAddress), cleanID),
						VenuePhotoURL:           photoURL,
						PlaceID:                  cleanID,
					}
				}
			}
		}
	}

	// Fallback for non-ChI Place IDs / Gemini Suggestions
	primary := cleanID
	if strings.Contains(cleanID, ",") {
		parts := strings.SplitN(cleanID, ",", 2)
		primary = strings.TrimSpace(parts[0])
	}
	return config.VenueDetails{
		PrimaryVenue:           primary,
		VenueFormattedAddress:  cleanID,
		VenueAdrFormatAddress:  cleanID,
		Address:                cleanID,
		GoogleMapURL:            fmt.Sprintf("https://maps.google.com/?q=%s", url.QueryEscape(cleanID)),
		GoogleMapDirectionsURL: fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%s", url.QueryEscape(cleanID)),
		VenuePhotoURL:           defaultPhoto,
		PlaceID:                  cleanID,
	}
}

// GenerateAIImage generates image bytes using Gemini/Imagen models configured strictly in .env
func GenerateAIImage(apiKey string, prompt string, aspect string) ([]byte, error) {
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	}
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not configured in settings or .env file")
	}

	modelsToTry := GetStrictEnvImageModels()
	if len(modelsToTry) == 0 {
		return nil, fmt.Errorf("No Gemini image models configured in environment (GEMINI_IMAGE_MODEL / GEMINI_FALLBACK_IMAGE_MODEL)")
	}

	ar := "9:16"
	if aspect != "" {
		ar = aspect
	}

	var lastErr error
	for _, modelName := range modelsToTry {
		imgBytes, err := executeSingleImageRequest(apiKey, modelName, prompt, ar)
		if err == nil {
			return imgBytes, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all image model attempts failed. Last error: %v", lastErr)
}

func executeSingleImageRequest(apiKey string, modelName string, prompt string, ar string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if strings.HasPrefix(modelName, "imagen-") {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:predict?key=%s", modelName, apiKey)
		reqBody := map[string]interface{}{
			"instances": []map[string]interface{}{
				{"prompt": prompt},
			},
			"parameters": map[string]interface{}{
				"sampleCount":      1,
				"aspectRatio":      ar,
				"outputMimeType":   "image/png",
				"personGeneration": "ALLOW_ADULT",
			},
		}

		jsonBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			bodyStr := string(body)
			if len(bodyStr) > 250 {
				bodyStr = bodyStr[:250] + "..."
			}
			return nil, fmt.Errorf("%s status %d: %s", modelName, resp.StatusCode, bodyStr)
		}

		var resStruct struct {
			Predictions []struct {
				BytesBase64Encoded string `json:"bytesBase64Encoded"`
			} `json:"predictions"`
		}
		if err := json.Unmarshal(body, &resStruct); err != nil {
			return nil, fmt.Errorf("%s JSON decode error: %w", modelName, err)
		}
		if len(resStruct.Predictions) == 0 || resStruct.Predictions[0].BytesBase64Encoded == "" {
			return nil, fmt.Errorf("%s returned empty prediction", modelName)
		}
		return base64.StdEncoding.DecodeString(resStruct.Predictions[0].BytesBase64Encoded)
	}

	// Multimodal Gemini Image API Format (gemini-3.1-flash-image / gemini-2.5-flash-image)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": fmt.Sprintf("Generate a high-resolution invitation card graphic artwork PNG (aspect ratio %s) for: %s", ar, prompt)},
				},
			},
		},
		"generation_config": map[string]interface{}{
			"response_modalities": []string{"IMAGE"},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		if len(bodyStr) > 250 {
			bodyStr = bodyStr[:250] + "..."
		}
		return nil, fmt.Errorf("%s status %d: %s", modelName, resp.StatusCode, bodyStr)
	}

	var resStruct struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &resStruct); err != nil {
		return nil, fmt.Errorf("%s JSON decode error: %w", modelName, err)
	}

	if len(resStruct.Candidates) > 0 {
		for _, part := range resStruct.Candidates[0].Content.Parts {
			if part.InlineData.Data != "" {
				return base64.StdEncoding.DecodeString(part.InlineData.Data)
			}
		}
	}

	return nil, fmt.Errorf("%s returned no valid image binary in response", modelName)
}
