package config

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const EventProfileFilename = "event_details.md"

// EventProfile holds structured event info loaded from or saved to event_details.md
type EventProfile struct {
	ID              string
	RawDetails      string
	EventType       string
	HostNames       string
	EventDate       string
	ISODate         string
	Venue           string
	DefaultCurrency string
	WelcomeMessage  string
	TargetAspect    string
	PlannerName     string
	PlannerRole     string
	TotalBudget     float64
	EstimatedGuests int
}

// ParseAndNormalizeMachineDate converts various date formats into machine-readable ISO (YYYY-MM-DD) and human display formats
func ParseAndNormalizeMachineDate(rawDate string) (isoDate string, displayDate string, parsedTime time.Time) {
	rawDate = strings.TrimSpace(rawDate)
	if rawDate == "" {
		now := time.Now()
		return now.Format("2006-01-02"), now.Format("January 02, 2006"), now
	}

	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"02-01-2006",
		"02/01/2006",
		"01/02/2006",
		"Jan 02, 2006",
		"January 02, 2006",
		"02 Jan 2006",
		"02 January 2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 Jan 2006",
		"2 January 2006",
		"2006-01-02T15:04:05Z07:00",
	}

	for _, fmtStr := range formats {
		if t, err := time.Parse(fmtStr, rawDate); err == nil {
			return t.Format("2006-01-02"), t.Format("January 02, 2006"), t
		}
	}

	return rawDate, rawDate, time.Time{}
}

// GenerateUniqueID creates a unique 4-char suffix slug (e.g. evt-naming-ceremony-rohan-a7f9)
func GenerateUniqueID(slug string) string {
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	hash := fmt.Sprintf("%x", b)
	if slug == "" || slug == "evt-shubh-event" {
		return fmt.Sprintf("evt-shubh-%s", hash)
	}
	return fmt.Sprintf("%s-%s", slug, hash)
}

// GetEventID derives or returns the unique workspace ID slug for Honcho Cloud Memory
func (p EventProfile) GetEventID() string {
	if p.ID != "" {
		return p.ID
	}

	parts := []string{}
	if p.EventType != "" {
		parts = append(parts, p.EventType)
	}
	if p.HostNames != "" {
		parts = append(parts, p.HostNames)
	}
	if len(parts) == 0 {
		return GenerateUniqueID("evt-shubh-event")
	}
	raw := strings.Join(parts, "-")
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := strings.ToLower(reg.ReplaceAllString(raw, "-"))
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return GenerateUniqueID("evt-shubh-event")
	}
	return GenerateUniqueID("evt-" + slug)
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
		TargetAspect:    "9:16", // Default aspect ratio if unspecified
		DefaultCurrency: "USD",  // Default currency if unspecified
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
		} else if strings.HasPrefix(trimmed, "* **Default Currency**:") {
			profile.DefaultCurrency = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Default Currency**:"))
		} else if strings.HasPrefix(trimmed, "* **Currency**:") {
			profile.DefaultCurrency = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Currency**:"))
		} else if strings.HasPrefix(trimmed, "* **Event ID**:") {
			profile.ID = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Event ID**:"))
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
		} else if strings.HasPrefix(trimmed, "* **Total Budget**:") {
			valStr := strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Total Budget**:"))
			valStr = strings.ReplaceAll(valStr, ",", "")
			valStr = strings.ReplaceAll(valStr, "₹", "")
			valStr = strings.ReplaceAll(valStr, "$", "")
			if b, err := strconv.ParseFloat(valStr, 64); err == nil {
				profile.TotalBudget = b
			}
		} else if strings.HasPrefix(trimmed, "* **Estimated Guests**:") {
			valStr := strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Estimated Guests**:"))
			if g, err := strconv.Atoi(valStr); err == nil {
				profile.EstimatedGuests = g
			}
		} else if strings.HasPrefix(trimmed, "* **Guest Count**:") {
			valStr := strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Guest Count**:"))
			if g, err := strconv.Atoi(valStr); err == nil {
				profile.EstimatedGuests = g
			}
		} else if strings.HasPrefix(trimmed, "* **Event Planner**:") {
			profile.PlannerName = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Event Planner**:"))
		} else if strings.HasPrefix(trimmed, "* **Planner Role**:") {
			profile.PlannerRole = strings.TrimSpace(strings.TrimPrefix(trimmed, "* **Planner Role**:"))
		}
	}

	if profile.ID == "" {
		profile.ID = profile.GetEventID()
	}

	if profile.DefaultCurrency == "" {
		profile.DefaultCurrency = "USD"
	}

	if profile.PlannerName == "" {
		profile.PlannerName = "Gokul"
	}
	if profile.PlannerRole == "" {
		profile.PlannerRole = "Lead Event Planner"
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
		if profile.DefaultCurrency != "" {
			parts = append(parts, fmt.Sprintf("(Currency: %s)", profile.DefaultCurrency))
		}
		if profile.TotalBudget > 0 {
			parts = append(parts, fmt.Sprintf("(Budget: %.2f)", profile.TotalBudget))
		}
		if profile.EstimatedGuests > 0 {
			parts = append(parts, fmt.Sprintf("(Guests: %d)", profile.EstimatedGuests))
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
	return SaveStructuredEventProfileWithBudget("Event", rawDetails, "", "", "USD", welcomeMessage, targetAspect, "Gokul", "Lead Event Planner", 0, 0)
}

// SaveStructuredEventProfileWithBudget persists structured event details including budget and guest count into event_details.md
func SaveStructuredEventProfileWithBudget(eventType, hostNames, eventDate, venue, currency, welcomeMsg, targetAspect string, plannerName string, plannerRole string, totalBudget float64, guestCount int) error {
	if currency == "" {
		currency = "USD"
	}
	if targetAspect == "" {
		targetAspect = "9:16"
	}
	if plannerName == "" {
		plannerName = "Gokul"
	}
	if plannerRole == "" {
		plannerRole = "Lead Event Planner"
	}

	// Read existing profile to preserve Event ID or budget if not explicitly provided
	existingProfile, hasProfile := LoadEventProfile()
	eventID := existingProfile.ID
	if !hasProfile || eventID == "" {
		p := EventProfile{EventType: eventType, HostNames: hostNames}
		eventID = p.GetEventID()
	}

	if totalBudget <= 0 && hasProfile {
		totalBudget = existingProfile.TotalBudget
	}
	if guestCount <= 0 && hasProfile {
		guestCount = existingProfile.EstimatedGuests
	}

	parts := []string{}
	if eventType != "" {
		parts = append(parts, eventType)
	}
	if hostNames != "" {
		parts = append(parts, "for "+hostNames)
	}
	if eventDate != "" {
		parts = append(parts, "on "+eventDate)
	}
	if venue != "" {
		parts = append(parts, "at "+venue)
	}
	if currency != "" {
		parts = append(parts, fmt.Sprintf("(Currency: %s)", currency))
	}
	if totalBudget > 0 {
		parts = append(parts, fmt.Sprintf("(Budget: %.2f)", totalBudget))
	}
	if guestCount > 0 {
		parts = append(parts, fmt.Sprintf("(Guests: %d)", guestCount))
	}
	rawDetails := strings.Join(parts, " ")

	filePath := filepath.Join(".", EventProfileFilename)

	mdContent := fmt.Sprintf(`# 📋 Active Event Profile

* **Target Aspect Ratio**: %s
* **Default Currency**: %s
* **Event ID**: %s
* **Event Type**: %s
* **Host / Couple Names**: %s
* **Event Date**: %s
* **Venue & Location**: %s
* **Welcome Message**: %s
* **Total Budget**: %.2f
* **Estimated Guests**: %d
* **Event Planner**: %s
* **Planner Role**: %s

## Full Raw Details
%s
`, targetAspect, currency, eventID, eventType, hostNames, eventDate, venue, welcomeMsg, totalBudget, guestCount, plannerName, plannerRole, rawDetails)

	return os.WriteFile(filePath, []byte(mdContent), 0644)
}

// SaveStructuredEventProfile persists structured event details into event_details.md
func SaveStructuredEventProfile(eventType, hostNames, eventDate, venue, currency, welcomeMsg, targetAspect string, plannerName string, plannerRole string) error {
	return SaveStructuredEventProfileWithBudget(eventType, hostNames, eventDate, venue, currency, welcomeMsg, targetAspect, plannerName, plannerRole, 0, 0)
}

// LocalRSVPRecord represents guest RSVP details saved to rsvps.json locally
type LocalRSVPRecord struct {
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Status    string `json:"status"`
	Headcount int    `json:"headcount"`
	Dietary   string `json:"dietary"`
	Cab       bool   `json:"cab"`
}

// SaveRSVPRecord appends or updates a guest RSVP record in rsvps.json
func SaveRSVPRecord(rec LocalRSVPRecord) error {
	filePath := filepath.Join(".", "rsvps.json")
	var records []LocalRSVPRecord
	if data, err := os.ReadFile(filePath); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &records)
	}
	found := false
	for i, r := range records {
		if (rec.Phone != "" && r.Phone == rec.Phone) || (r.Name != "" && strings.EqualFold(r.Name, rec.Name)) {
			records[i] = rec
			found = true
			break
		}
	}
	if !found {
		records = append(records, rec)
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// LoadRSVPRecords loads local guest RSVPs from rsvps.json
func LoadRSVPRecords() ([]LocalRSVPRecord, error) {
	filePath := filepath.Join(".", "rsvps.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var records []LocalRSVPRecord
	err = json.Unmarshal(data, &records)
	return records, err
}
