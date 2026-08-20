package tui

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

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/client"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/generator"

	tea "github.com/charmbracelet/bubbletea"
)

type AgentStreamMsg struct {
	Events []client.AgentStreamEvent
	Err    error
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
			Err:         err,
		}
	}
}

func (m Model) runGenerationCmd(payload generator.ResponsePayload) tea.Cmd {
	return func() tea.Msg {
		filename := fmt.Sprintf("shubh_design_%s_%d.png", m.SessionID, time.Now().Unix())
		outPath := filepath.Join(m.Config.OutputDir, filename)

		var err error
		if m.Config.GeminiAPIKey != "" {
			err = generateGeminiImage(m.Config.GeminiAPIKey, m.Config.GeminiImageModel, payload.CorePrompt, payload.Aspect, outPath)
		} else {
			err = createPlaceholderImage(outPath, payload.DisplayTitle, payload.WelcomeMessage)
		}

		return GenerationCompleteMsg{
			Payload:   payload,
			ImagePath: outPath,
			Err:       err,
		}
	}
}

func (m Model) runSuggestionCmd(eventType string, style string) tea.Cmd {
	return func() tea.Msg {
		suggestions, err := generator.GenerateAIPromptSuggestions(m.Config.GeminiAPIKey, eventType, style)
		header := fmt.Sprintf("🤖 Live Gemini AI Prompt Suggestions for '%s' (%s):", eventType, style)
		if err != nil {
			header = fmt.Sprintf("⚠️ Fallback Prompt Suggestions (%v) for '%s':", err, eventType)
		}

		formattedText := fmt.Sprintf("%s\n\n"+
			"1. %s\n\n"+
			"2. %s\n\n"+
			"3. %s\n\n"+
			"4. %s\n\n"+
			"5. 🔄 Generate 4 More AI Prompt Suggestions\n\n"+
			"Press ↑/↓ to navigate, Enter to select, or type 1-4 / custom prompt below:",
			header, suggestions[0], suggestions[1], suggestions[2], suggestions[3])

		return SuggestionCompleteMsg{
			Suggestions: formattedText,
			OptionList:  suggestions,
			Err:         err,
		}
	}
}

func createPlaceholderImage(outPath string, title string, welcome string) error {
	_ = title
	_ = welcome
	img := image.NewRGBA(image.Rect(0, 0, 1024, 1024))
	bg := color.RGBA{R: 25, G: 20, B: 15, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	gold := color.RGBA{R: 212, G: 175, B: 55, A: 255}
	for x := 50; x < 974; x++ {
		img.Set(x, 50, gold)
		img.Set(x, 974, gold)
	}
	for y := 50; y < 974; y++ {
		img.Set(50, y, gold)
		img.Set(974, y, gold)
	}

	_ = os.MkdirAll(filepath.Dir(outPath), 0755)
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func generateGeminiImage(apiKey string, modelName string, prompt string, aspect string, outPath string) error {
	modelsToTry := generator.GetStrictEnvImageModels()
	if len(modelsToTry) == 0 && modelName != "" {
		modelsToTry = []string{modelName}
	}
	if len(modelsToTry) == 0 {
		return fmt.Errorf("No Gemini image models configured in environment (GEMINI_IMAGE_MODEL / GEMINI_FALLBACK_IMAGE_MODEL)")
	}

	var lastErr error
	for _, m := range modelsToTry {
		err := executeGeminiModelRequest(apiKey, m, prompt, aspect, outPath)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("all image model attempts failed. Last error: %v", lastErr)
}

func executeGeminiModelRequest(apiKey string, modelName string, prompt string, aspect string, outPath string) error {
	ar := "1:1"
	switch aspect {
	case "9:16":
		ar = "9:16"
	case "4:5":
		ar = "4:5"
	case "16:9":
		ar = "16:9"
	}

	// 1. Imagen Predict API Format (imagen-*)
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
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
		if err != nil {
			return err
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Imagen API HTTP %d: %s", resp.StatusCode, string(body))
		}

		var resStruct struct {
			Predictions []struct {
				BytesBase64Encoded string `json:"bytesBase64Encoded"`
			} `json:"predictions"`
		}

		if err := json.Unmarshal(body, &resStruct); err != nil {
			return fmt.Errorf("failed to parse Imagen API JSON response: %w", err)
		}

		if len(resStruct.Predictions) == 0 || resStruct.Predictions[0].BytesBase64Encoded == "" {
			return fmt.Errorf("Imagen API returned empty image prediction")
		}

		imgBytes, err := base64.StdEncoding.DecodeString(resStruct.Predictions[0].BytesBase64Encoded)
		if err != nil {
			return fmt.Errorf("failed to decode base64 image data: %w", err)
		}

		_ = os.MkdirAll(filepath.Dir(outPath), 0755)
		return os.WriteFile(outPath, imgBytes, 0644)
	}

	// 2. Gemini Multimodal Image API Format (gemini-3.1-flash-image / gemini-2.5-flash-image)
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
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Gemini Image API HTTP %d: %s", resp.StatusCode, string(body))
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

	if err := json.Unmarshal(body, &resStruct); err == nil && len(resStruct.Candidates) > 0 {
		for _, part := range resStruct.Candidates[0].Content.Parts {
			if part.InlineData.Data != "" {
				imgBytes, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err == nil {
					_ = os.MkdirAll(filepath.Dir(outPath), 0755)
					return os.WriteFile(outPath, imgBytes, 0644)
				}
			}
		}
	}

	return fmt.Errorf("Gemini Image API returned no valid image binary in response")
}

func (m Model) runAgentCmd(prompt string) tea.Cmd {
	return func() tea.Msg {
		c := client.NewAgentClient()
		var events []client.AgentStreamEvent
		sessionID := m.SessionID
		if sessionID == "" {
			sessionID = fmt.Sprintf("session-%s", m.EventID)
		}

		eventID := m.EventID
		if eventID == "" {
			eventID = "evt-shubh-event"
		}

		eventContext := m.EventDetails
		sym := GetCurrencySymbol(m.Currency)
		sugAmt, sugGuests, _ := GetSuggestedBudgetForEvent(m.EventType, m.Currency)
		sugInfo := fmt.Sprintf("Active Event Type: %s, Active Currency: %s. Standard Industry Benchmark Budget Suggestion for %s (%s): %s%.2f for %d guests (~%s%.2f/guest).",
			m.EventType, m.Currency, m.EventType, m.Currency, sym, sugAmt, sugGuests, sym, sugAmt/float64(sugGuests))

		if eventContext == "" {
			if profile, ok := config.LoadEventProfile(); ok {
				eventContext = profile.RawDetails
			}
		}
		if eventContext == "" {
			eventContext = fmt.Sprintf("Event Type: %s, Hosts: %s, Date: %s, Venue: %s, Welcome Message: %s. %s", m.EventType, m.HostNames, m.EventDate, m.Venue, m.WelcomeMessage, sugInfo)
		} else {
			eventContext = fmt.Sprintf("%s | %s", eventContext, sugInfo)
		}

		err := c.SendMessageStreams(sessionID, prompt, m.PlannerName, m.PlannerRole, eventID, eventContext, func(ev client.AgentStreamEvent) {
			events = append(events, ev)
		})

		return AgentStreamMsg{
			Events: events,
			Err:    err,
		}
	}
}
