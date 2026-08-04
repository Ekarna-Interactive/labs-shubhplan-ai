package generator

import (
	"fmt"
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
		cleanDetails = "Auspicious Celebration Invitation Card"
	}

	welcomeText := strings.TrimSpace(welcomeMessage)
	title := cleanDetails

	if aspect == "" {
		aspect = "9:16"
	}

	aspectInstruction := "aspect ratio 9:16 vertical poster story layout"
	switch aspect {
	case "4:5":
		aspectInstruction = "aspect ratio 4:5 vertical portrait social feed layout"
	case "1:1":
		aspectInstruction = "aspect ratio 1:1 square standard card layout"
	case "16:9":
		aspectInstruction = "aspect ratio 16:9 landscape banner desktop layout"
	case "9:16":
		aspectInstruction = "aspect ratio 9:16 vertical mobile story layout"
	}

	var corePrompt string
	if welcomeText != "" {
		corePrompt = fmt.Sprintf(
			"Full view complete invitation card illustration, entire card visible from top header to bottom footer with generous margin padding around all edges, no cropped top or bottom text, uncropped complete framing, %s. MANDATORY TEXT TO RENDER ON CARD PLATE: '%s'. Secondary welcome text: '%s'. High contrast legible typography on a translucent background plate, premium ornate floral gold borders, vibrant colors, clean studio lighting.",
			aspectInstruction,
			cleanDetails,
			welcomeText,
		)
	} else {
		corePrompt = fmt.Sprintf(
			"Full view complete invitation card illustration, entire card visible from top header to bottom footer with generous margin padding around all edges, no cropped top or bottom text, uncropped complete framing, %s. MANDATORY TEXT TO RENDER ON CARD PLATE: '%s'. High contrast legible central typography, ornate floral gold borders, vibrant colors, clean studio lighting.",
			aspectInstruction,
			cleanDetails,
		)
	}

	return ResponsePayload{
		CorePrompt:     corePrompt,
		DisplayTitle:   title,
		WelcomeMessage: welcomeText,
		Aspect:         aspect,
	}
}
