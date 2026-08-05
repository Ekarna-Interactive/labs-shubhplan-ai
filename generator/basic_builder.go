package generator

import (
	"fmt"
	"regexp"
	"strings"
)

// BasicBuilder is the open-source community implementation of PromptBuilder.
type BasicBuilder struct{}

// NewBasicBuilder initializes a new community prompt builder.
func NewBasicBuilder() *BasicBuilder {
	return &BasicBuilder{}
}

// Compile constructs a standard clean prompt for event design generation defaulting to 9:16 aspect ratio.
func (b *BasicBuilder) Compile(eventDetails string, welcomeMessage string) ResponsePayload {
	return b.CompileWithAspect(eventDetails, welcomeMessage, "9:16")
}

// CompileWithAspect constructs a prompt tailored to the requested aspect ratio layout.
func (b *BasicBuilder) CompileWithAspect(eventDetails string, welcomeMessage string, aspect string) ResponsePayload {
	cleanDetails := strings.TrimSpace(eventDetails)
	if cleanDetails == "" {
		cleanDetails = "Auspicious Celebration Invitation"
	}

	welcomeText := strings.TrimSpace(welcomeMessage)

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
	noPromptTextInstruction := "CRITICAL TYPOGRAPHY MANDATE: Do NOT print meta-descriptions, structural phrases, or headers like 'a complete wedding invitation card', 'invitation card', 'full view', or prompt text anywhere on the image. ONLY print the actual event details (host names, date, venue) on the center plate."

	var corePrompt string
	var displayTitle string

	if isFullPromptDescription(cleanDetails) {
		cardText := extractCardTextFromPrompt(cleanDetails)
		displayTitle = cardText
		sanitizedVisualPrompt := sanitizeVisualPromptBody(cleanDetails)

		if welcomeText != "" {
			corePrompt = fmt.Sprintf(
				"%s, %s, %s. MANDATORY TEXT TO RENDER ON CARD PLATE: '%s'. Secondary welcome text: '%s'. %s",
				sanitizedVisualPrompt,
				negativeNoUIInstruction,
				aspectInstruction,
				cardText,
				welcomeText,
				noPromptTextInstruction,
			)
		} else {
			corePrompt = fmt.Sprintf(
				"%s, %s, %s. MANDATORY TEXT TO RENDER ON CARD PLATE: '%s'. %s",
				sanitizedVisualPrompt,
				negativeNoUIInstruction,
				aspectInstruction,
				cardText,
				noPromptTextInstruction,
			)
		}
	} else {
		displayTitle = stripMetaPhrases(cleanDetails)
		if welcomeText != "" {
			corePrompt = fmt.Sprintf(
				"Bespoke event invitation graphic artwork, %s, entire card visible from top header to bottom footer with generous margin padding around all edges, no cropped top or bottom text, uncropped complete framing, %s. MANDATORY TEXT TO RENDER ON CARD PLATE: '%s'. Secondary welcome text: '%s'. High contrast legible typography on a translucent background plate, premium ornate floral gold borders, vibrant colors, clean studio lighting. %s",
				negativeNoUIInstruction,
				aspectInstruction,
				displayTitle,
				welcomeText,
				noPromptTextInstruction,
			)
		} else {
			corePrompt = fmt.Sprintf(
				"Bespoke event invitation graphic artwork, %s, entire card visible from top header to bottom footer with generous margin padding around all edges, no cropped top or bottom text, uncropped complete framing, %s. MANDATORY TEXT TO RENDER ON CARD PLATE: '%s'. High contrast legible central typography, ornate floral gold borders, vibrant colors, clean studio lighting. %s",
				negativeNoUIInstruction,
				aspectInstruction,
				displayTitle,
				noPromptTextInstruction,
			)
		}
	}

	return ResponsePayload{
		CorePrompt:     corePrompt,
		DisplayTitle:   displayTitle,
		WelcomeMessage: welcomeText,
		Aspect:         aspect,
	}
}

func isFullPromptDescription(text string) bool {
	lower := strings.ToLower(text)
	return len(text) > 80 ||
		strings.Contains(lower, "full view") ||
		strings.Contains(lower, "styled in") ||
		strings.Contains(lower, "typography with") ||
		strings.Contains(lower, "artwork for") ||
		strings.Contains(lower, "illustration for") ||
		strings.Contains(lower, "composition for")
}

func sanitizeVisualPromptBody(prompt string) string {
	// Strip meta-phrases that cause models to render literal description headings on card plates
	cleaned := prompt
	metaPhrasesToClean := []string{
		"Full view of a complete rectangular wedding invitation card with generous margin padding around all four borders, styled in ",
		"Full view of a complete rectangular invitation card with generous margin padding around all four borders, styled in ",
		"Full view of a complete wedding invitation card",
		"Full view complete invitation card illustration for ",
		"Full view of a complete ",
		"a complete wedding invitation card",
		"complete wedding invitation card",
	}
	for _, phrase := range metaPhrasesToClean {
		cleaned = strings.ReplaceAll(cleaned, phrase, "Bespoke event invitation artwork styled in ")
	}
	return strings.TrimSpace(cleaned)
}

func extractCardTextFromPrompt(prompt string) string {
	// 1. Look for quoted text e.g. text "Priya & Arjun" or text 'Priya & Arjun'
	quotedRegex := regexp.MustCompile(`(?:text|with the text|titled)\s*["'“]([^"'”]+)["'”]`)
	matches := quotedRegex.FindStringSubmatch(prompt)
	if len(matches) > 1 && strings.TrimSpace(matches[1]) != "" {
		return stripMetaPhrases(strings.TrimSpace(matches[1]))
	}

	// 2. Look for "for <Event Name> rendered/featuring/styled"
	forRegex := regexp.MustCompile(`(?:for|of)\s+([A-Za-z0-9\s&'-]+?)(?:\s+(?:rendered|styled|featuring|with|in\s+[A-Z])|$)`)
	matchesFor := forRegex.FindStringSubmatch(prompt)
	if len(matchesFor) > 1 && strings.TrimSpace(matchesFor[1]) != "" {
		result := strings.TrimSpace(matchesFor[1])
		if len(result) > 3 && len(result) < 60 {
			return stripMetaPhrases(result)
		}
	}

	// 3. Fallback: Strip common visual prefix phrases
	cleaned := prompt
	prefixes := []string{
		"Full view complete invitation card illustration for ",
		"Full view of a complete rectangular wedding invitation card with generous margin padding around all four borders, styled in ",
		"Full view of a complete ",
		"Bespoke invitation artwork for ",
		"Ornate composition for ",
		"Modern premium banner artwork for ",
	}
	for _, prefix := range prefixes {
		cleaned = strings.TrimPrefix(cleaned, prefix)
	}

	if idx := strings.IndexAny(cleaned, ",."); idx > 0 && idx < 60 {
		cleaned = cleaned[:idx]
	} else if len(cleaned) > 50 {
		cleaned = cleaned[:50]
	}

	return stripMetaPhrases(cleaned)
}

func stripMetaPhrases(text string) string {
	phrases := []string{
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
	}
	cleaned := text
	for _, p := range phrases {
		re := regexp.MustCompile("(?i)\\b" + regexp.QuoteMeta(p) + "\\b")
		cleaned = re.ReplaceAllString(cleaned, "")
	}
	return strings.TrimSpace(cleaned)
}
