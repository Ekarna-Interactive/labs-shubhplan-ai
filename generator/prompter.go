package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GenerateAIPromptSuggestions uses Gemini LLM to dynamically generate 4 creative prompts tailored to event details and style.
func GenerateAIPromptSuggestions(apiKey string, eventType string, style string) ([]string, error) {
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

	// Models to try in sequence
	modelsToTry := []string{"gemini-2.0-flash", "gemini-1.5-flash", "gemini-2.5-flash", "gemini-3.5-flash"}
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

	modelsToTry := []string{"gemini-2.0-flash", "gemini-1.5-flash", "gemini-2.5-flash", "gemini-3.5-flash"}
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
