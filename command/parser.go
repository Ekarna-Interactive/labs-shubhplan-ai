package command

import (
	"strings"
)

// CommandType defines supported slash command actions
type CommandType string

const (
	CmdGenerate CommandType = "GENERATE"
	CmdRefine   CommandType = "REFINE"
	CmdSuggest  CommandType = "SUGGEST"
	CmdPreview  CommandType = "PREVIEW"
	CmdConfig   CommandType = "CONFIG"
	CmdStyle    CommandType = "STYLE"
	CmdEvent    CommandType = "EVENT"
	CmdAspect   CommandType = "ASPECT"
	CmdReset    CommandType = "RESET"
	CmdHelp     CommandType = "HELP"
	CmdUnknown  CommandType = "UNKNOWN"
)

// ParsedInput holds structured output from parsing raw command text
type ParsedInput struct {
	Type           CommandType
	RawInput       string
	EventDetails   string
	WelcomeMessage string
	Args           []string
}

// Parse takes a raw text input string and returns a structured ParsedInput struct
func Parse(input string) ParsedInput {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ParsedInput{Type: CmdUnknown, RawInput: input}
	}

	// Default to /generate if no slash prefix is present
	if !strings.HasPrefix(trimmed, "/") {
		return parseGenerateInput(trimmed)
	}

	parts := strings.Fields(trimmed)
	cmdToken := strings.ToLower(parts[0])
	argsString := strings.TrimSpace(strings.TrimPrefix(trimmed, parts[0]))

	switch cmdToken {
	case "/generate", "/design", "/create":
		return parseGenerateInput(argsString)

	case "/refine", "/edit", "/modify":
		return ParsedInput{
			Type:         CmdRefine,
			RawInput:     input,
			EventDetails: argsString,
			Args:         parts[1:],
		}

	case "/suggest", "/suggestion", "/theme", "/ideas":
		return ParsedInput{
			Type:         CmdSuggest,
			RawInput:     input,
			EventDetails: argsString,
			Args:         parts[1:],
		}

	case "/style", "/aesthetic", "/preset":
		return ParsedInput{
			Type:         CmdStyle,
			RawInput:     input,
			EventDetails: argsString,
			Args:         parts[1:],
		}

	case "/event", "/details", "/profile":
		return ParsedInput{
			Type:         CmdEvent,
			RawInput:     input,
			EventDetails: argsString,
			Args:         parts[1:],
		}

	case "/aspect", "/resolution", "/res", "/ratio":
		return ParsedInput{
			Type:         CmdAspect,
			RawInput:     input,
			EventDetails: argsString,
			Args:         parts[1:],
		}

	case "/reset", "/restart", "/new":
		return ParsedInput{
			Type:     CmdReset,
			RawInput: input,
		}

	case "/preview", "/web", "/open":
		return ParsedInput{
			Type:     CmdPreview,
			RawInput: input,
		}

	case "/config", "/key", "/apikey":
		return ParsedInput{
			Type:         CmdConfig,
			RawInput:     input,
			EventDetails: argsString,
			Args:         parts[1:],
		}

	case "/help", "/h", "/?":
		return ParsedInput{
			Type:     CmdHelp,
			RawInput: input,
		}

	default:
		return ParsedInput{
			Type:     CmdUnknown,
			RawInput: input,
			Args:     parts,
		}
	}
}

func parseGenerateInput(input string) ParsedInput {
	welcomeMsg := ""
	details := input

	// Check if welcome subheader is supplied via welcome="..." or --welcome "..."
	if idx := strings.Index(strings.ToLower(input), "welcome="); idx != -1 {
		details = strings.TrimSpace(input[:idx])
		val := input[idx+8:]
		val = strings.Trim(val, `"'`)
		welcomeMsg = val
	}

	return ParsedInput{
		Type:           CmdGenerate,
		RawInput:       input,
		EventDetails:   details,
		WelcomeMessage: welcomeMsg,
	}
}
