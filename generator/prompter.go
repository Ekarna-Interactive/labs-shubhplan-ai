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
func GenerateAIPromptSuggestions(apiKey string, eventDetails string, style string) ([]string, error) {
	details := strings.TrimSpace(eventDetails)
	if details == "" {
		details = "Auspicious Event Celebration"
	}
	if style == "" {
		style = "South Indian Traditional"
	}

	if apiKey == "" {
		return GenerateFallbackPrompts(details, style), fmt.Errorf("GEMINI_API_KEY is not set. Using offline fallback prompts.")
	}

	promptMsg := fmt.Sprintf(`You are an expert AI invitation prompter agent for boutique events (Weddings, Naming Ceremonies, Baby Showers, Housewarmings, Birthdays, Anniversaries, Corporate Galas).

EVENT DETAILS: %s
TARGET DESIGN AESTHETIC STYLE: %s

CRITICAL STYLE MANDATE:
All 4 generated prompts MUST strictly use and explore the requested TARGET DESIGN AESTHETIC STYLE ('%s').
Do NOT include, mention, or mix in any other unrelated design styles (e.g. if 'Paper Cut Art' is requested, do NOT mention pop art, clay 3D, or watercolor in any of the prompts).
Every prompt must be a unique, highly creative variation entirely inside the requested '%s' style.

Each prompt must describe:
1. Composition: Full view complete invitation card visible from top to bottom with margin padding around all 4 edges.
2. Typography: Centered high-contrast legible event typography on a translucent background plate.
3. Aesthetics: Specific color swatches, textures, borders, and studio lighting strictly matching '%s'.

Output format requirement: Return EXACTLY 4 numbered lines without introduction or markdown formatting:
1. <Prompt 1>
2. <Prompt 2>
3. <Prompt 3>
4. <Prompt 4>`, details, style, style, style, style)

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

	return GenerateFallbackPrompts(details, style), lastErr
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
func GenerateFallbackPrompts(eventDetails string, style string) []string {
	details := strings.TrimSpace(eventDetails)
	if details == "" {
		details = "Auspicious Celebration Event"
	}
	if style == "" {
		style = "Paper Cut Art"
	}

	return []string{
		fmt.Sprintf("Full view complete invitation card illustration for %s rendered strictly in %s medium, central glowing translucent text plate, multi-layered intricate borders, soft drop shadows, studio lighting.", details, style),
		fmt.Sprintf("Bespoke %s invitation artwork for %s featuring multi-tiered layered paper silhouettes, floral corner filigree, rich gold and pastel color palette, centered elegant typography plate.", style, details),
		fmt.Sprintf("Ornate %s composition for %s, textured craft paper elements with depth separation, high contrast legible typography, full-bleed uncropped framing.", style, details),
		fmt.Sprintf("Modern premium %s banner artwork for %s, clean geometric paper cut layers, glowing ambient backdrop, crisp studio border framing.", style, details),
	}
}
