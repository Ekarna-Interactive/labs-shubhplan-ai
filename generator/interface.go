package generator

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
}
