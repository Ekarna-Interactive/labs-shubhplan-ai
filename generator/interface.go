package generator

// EventData holds structured details for event typography layout (matching agents-adk).
type EventData struct {
	EventType      string `json:"event_type,omitempty"`
	HostNames      string `json:"host_names,omitempty"`
	EventDate      string `json:"event_date,omitempty"`
	Venue          string `json:"venue,omitempty"`
	WelcomeMessage string `json:"welcome_message,omitempty"`
	VisualPrompt   string `json:"visual_prompt,omitempty"`
	Aspect         string `json:"aspect,omitempty"`
}

// ResponsePayload holds the compiled prompt result for generation runs.
type ResponsePayload struct {
	CorePrompt     string `json:"corePrompt"`
	DisplayTitle   string `json:"displayTitle"`
	WelcomeMessage string `json:"welcomeMessage,omitempty"`
	Aspect         string `json:"aspect"`
}

// PromptBuilder defines the contract for compiling user input into clean AI prompts.
// Community Open-Source Impl: BasicBuilder (clean 1:1 standard pass-through prompt)
// Private Enterprise Impl: MultiResBuilder (3-viewport aspect ratios & skill templates)
type PromptBuilder interface {
	Compile(eventDetails string, welcomeMessage string) ResponsePayload
	CompileWithAspect(eventDetails string, welcomeMessage string, aspect string) ResponsePayload
	CompileStructured(data EventData) ResponsePayload
}
