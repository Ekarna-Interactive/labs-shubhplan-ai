package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const EventProfileFilename = "event_details.md"

// EventProfile holds structured event info loaded from or saved to event_details.md
type EventProfile struct {
	RawDetails     string
	EventType      string
	HostNames      string
	EventDate      string
	Venue          string
	WelcomeMessage string
	TargetAspect   string
}

// LoadEventProfile reads event_details.md from the CLI working directory
func LoadEventProfile() (EventProfile, bool) {
	filePath := filepath.Join(".", EventProfileFilename)
	data, err := os.ReadFile(filePath)
	if err != nil || len(data) == 0 {
		return EventProfile{}, false
	}

	content := string(data)
	profile := EventProfile{
		TargetAspect: "9:16", // Default aspect ratio if unspecified
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	inRawSection := false
	rawLines := []string{}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## Full Raw Details") {
			inRawSection = true
			continue
		}

		if inRawSection {
			if trimmed != "" {
				rawLines = append(rawLines, trimmed)
			}
			continue
		}

		if strings.HasPrefix(trimmed, "* **Target Aspect Ratio**:") {
			profile.TargetAspect = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Target Aspect Ratio**:"))
		} else if strings.HasPrefix(trimmed, "* **Event Type**:") {
			profile.EventType = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Event Type**:"))
		} else if strings.HasPrefix(trimmed, "* **Host / Couple Names**:") {
			profile.HostNames = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Host / Couple Names**:"))
		} else if strings.HasPrefix(trimmed, "* **Event Date**:") {
			profile.EventDate = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Event Date**:"))
		} else if strings.HasPrefix(trimmed, "* **Venue & Location**:") {
			profile.Venue = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Venue & Location**:"))
		} else if strings.HasPrefix(trimmed, "* **Welcome Message**:") {
			profile.WelcomeMessage = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Welcome Message**:"))
		}
	}

	if len(rawLines) > 0 {
		profile.RawDetails = strings.Join(rawLines, " ")
	} else {
		// Fallback raw details string reconstructed from fields
		parts := []string{}
		if profile.EventType != "" {
			parts = append(parts, profile.EventType)
		}
		if profile.HostNames != "" {
			parts = append(parts, "for "+profile.HostNames)
		}
		if profile.EventDate != "" {
			parts = append(parts, "on "+profile.EventDate)
		}
		if profile.Venue != "" {
			parts = append(parts, "at "+profile.Venue)
		}
		profile.RawDetails = strings.Join(parts, " ")
	}

	if profile.RawDetails == "" {
		return EventProfile{}, false
	}

	return profile, true
}

// SaveEventProfile persists the user's event details into event_details.md
func SaveEventProfile(rawDetails string, welcomeMessage string, targetAspect string) error {
	cleanDetails := strings.TrimSpace(rawDetails)
	if cleanDetails == "" {
		return nil
	}
	if targetAspect == "" {
		targetAspect = "9:16"
	}

	filePath := filepath.Join(".", EventProfileFilename)

	mdContent := fmt.Sprintf(`# 📋 Active Event Profile

* **Target Aspect Ratio**: %s
* **Welcome Message**: %s

## Full Raw Details
%s
`, targetAspect, welcomeMessage, cleanDetails)

	return os.WriteFile(filePath, []byte(mdContent), 0644)
}
