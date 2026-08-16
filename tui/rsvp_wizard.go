package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
	tea "github.com/charmbracelet/bubbletea"
)

func parseHeadcountInput(input string) int {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return 1
	}

	// 1. Handle addition expressions like "2+1" or "2 + 1"
	if strings.Contains(raw, "+") {
		parts := strings.Split(raw, "+")
		sum := 0
		re := regexp.MustCompile(`[0-9]+`)
		for _, p := range parts {
			if numStr := re.FindString(strings.TrimSpace(p)); numStr != "" {
				if val, err := strconv.Atoi(numStr); err == nil {
					sum += val
				}
			}
		}
		if sum > 0 {
			return sum
		}
	}

	// 2. Direct Atoi check
	if val, err := strconv.Atoi(raw); err == nil && val > 0 {
		return val
	}

	// 3. Regex match for numbers in text like "2 guests" or "3 headcount"
	re := regexp.MustCompile(`[0-9]+`)
	if numStr := re.FindString(raw); numStr != "" {
		if val, err := strconv.Atoi(numStr); err == nil && val > 0 {
			return val
		}
	}

	return 1
}

func (m Model) handleRSVPWizardInput(input string) (Model, tea.Cmd) {
	trimmed := strings.TrimSpace(input)

	switch m.RSVPStep {
	case RSVPStepGuestName:
		if trimmed == "" {
			m.Logs = append(m.Logs, LogEntry{
				Sender:  "WARNING",
				Text:    "⚠️ Guest Full Name is mandatory. Please enter guest full name (e.g. 'Rohan Kumar').",
				IsError: true,
			})
			m.updateViewportContent()
			return m, nil
		}
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: trimmed})
		m.RSVPData.Name = trimmed
		m.RSVPStep = RSVPStepPhone
		m.TextInput.Placeholder = "Type phone number (e.g. +919876543210) and press Enter..."
		m.Logs = append(m.Logs, LogEntry{
			Sender: "WIZARD",
			Text:   fmt.Sprintf("✓ Guest Name: %s\nStep 2 of 6 [Mandatory]: Enter Phone Number (e.g. '+919876543210')", m.RSVPData.Name),
		})
		m.updateViewportContent()
		return m, nil

	case RSVPStepPhone:
		if trimmed == "" {
			m.Logs = append(m.Logs, LogEntry{
				Sender:  "WARNING",
				Text:    "⚠️ Phone number is mandatory. Please enter phone number (e.g. '+919876543210').",
				IsError: true,
			})
			m.updateViewportContent()
			return m, nil
		}
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: trimmed})
		m.RSVPData.Phone = trimmed
		m.RSVPStep = RSVPStepStatus
		m.TextInput.Placeholder = "Type 1 for Attending, 2 for Declined, then press Enter..."
		m.Logs = append(m.Logs, LogEntry{
			Sender: "WIZARD",
			Text:   fmt.Sprintf("✓ Phone: %s\nStep 3 of 6 [Mandatory]: Attendance Status\n  1. Confirmed / Attending\n  2. Declined", m.RSVPData.Phone),
		})
		m.updateViewportContent()
		return m, nil

	case RSVPStepStatus:
		status := "confirmed"
		statusLabel := "Confirmed (Attending)"
		if trimmed == "2" || strings.HasPrefix(strings.ToLower(trimmed), "d") {
			status = "declined"
			statusLabel = "Declined"
		}
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: statusLabel})
		m.RSVPData.Status = status
		m.RSVPStep = RSVPStepHeadcount
		m.TextInput.Placeholder = "Type headcount (e.g. '3', or '2+1' for 2 guests + 1 plus-one)..."
		m.Logs = append(m.Logs, LogEntry{
			Sender: "WIZARD",
			Text:   fmt.Sprintf("✓ Status: %s\nStep 4 of 6 [Mandatory]: Enter Total Headcount (e.g. '3', or '2+1')", statusLabel),
		})
		m.updateViewportContent()
		return m, nil

	case RSVPStepHeadcount:
		headcount := parseHeadcountInput(trimmed)
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: fmt.Sprintf("%d", headcount)})
		m.RSVPData.Headcount = headcount
		m.RSVPStep = RSVPStepDietary
		m.TextInput.Placeholder = "Select option 1-4 or press Enter to skip..."
		m.Logs = append(m.Logs, LogEntry{
			Sender: "WIZARD",
			Text:   fmt.Sprintf("✓ Headcount: %d\nStep 5 of 6 [Optional]: Dietary Preferences\n  1. Jain / Vegetarian\n  2. Non-Vegetarian\n  3. Vegan / Gluten-Free\n  4. None / Default (Press Enter to skip)", headcount),
		})
		m.updateViewportContent()
		return m, nil

	case RSVPStepDietary:
		diet := ""
		switch trimmed {
		case "1":
			diet = "Jain / Vegetarian"
		case "2":
			diet = "Non-Vegetarian"
		case "3":
			diet = "Vegan / Gluten-Free"
		default:
			if trimmed != "" && trimmed != "4" {
				diet = trimmed
			}
		}
		dietLog := diet
		if dietLog == "" {
			dietLog = "(skipped)"
		}
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: dietLog})
		m.RSVPData.Dietary = diet
		dietDisplay := diet
		if dietDisplay == "" {
			dietDisplay = "None"
		}
		m.RSVPStep = RSVPStepCab
		m.TextInput.Placeholder = "Select 1 for Cab Needed, 2 for Self-Transport..."
		m.Logs = append(m.Logs, LogEntry{
			Sender: "WIZARD",
			Text:   fmt.Sprintf("✓ Dietary: %s\nStep 6 of 6 [Optional]: Transport / Cab Request\n  1. Yes (Cab Needed)\n  2. No (Self-Transport)", dietDisplay),
		})
		m.updateViewportContent()
		return m, nil

	case RSVPStepCab:
		cab := (trimmed == "1" || strings.HasPrefix(strings.ToLower(trimmed), "y") || strings.Contains(strings.ToLower(trimmed), "cab"))
		m.RSVPData.Cab = cab
		m.RSVPStep = RSVPStepInactive
		m.TextInput.Placeholder = "Type an event prompt (e.g. 'Build Mehendi timeline', 'Check floral budget')..."

		cabDisplay := "No (Self-Transport)"
		if cab {
			cabDisplay = "Yes (Cab Needed)"
		}
		m.Logs = append(m.Logs, LogEntry{Sender: "USER", Text: cabDisplay})
		dietDisplay := m.RSVPData.Dietary
		if dietDisplay == "" {
			dietDisplay = "None"
		}
		statusDisplay := "Confirmed (Attending)"
		if m.RSVPData.Status == "declined" {
			statusDisplay = "Declined"
		}

		summary := fmt.Sprintf(
			"✅ [RSVP WIZARD COMPLETE] Guest RSVP Recorded!\n"+
				"  • Guest Full Name: %s\n"+
				"  • Phone Number: %s\n"+
				"  • Attendance Status: %s\n"+
				"  • Headcount: %d\n"+
				"  • Dietary Preferences: %s\n"+
				"  • Transport Request: %s",
			m.RSVPData.Name, m.RSVPData.Phone, statusDisplay, m.RSVPData.Headcount, dietDisplay, cabDisplay,
		)

		if m.RSVPData.Status == "declined" {
			m.RSVPOverview.Declined += m.RSVPData.Headcount
		} else {
			m.RSVPOverview.Attending += m.RSVPData.Headcount
		}
		if m.RSVPOverview.TotalGuests > 0 {
			m.RSVPOverview.Pending = m.RSVPOverview.TotalGuests - (m.RSVPOverview.Attending + m.RSVPOverview.Declined)
			if m.RSVPOverview.Pending < 0 {
				m.RSVPOverview.Pending = 0
			}
		}
		if dietDisplay != "" && dietDisplay != "None" {
			if m.RSVPOverview.DietaryReqs == nil {
				m.RSVPOverview.DietaryReqs = make(map[string]int)
			}
			m.RSVPOverview.DietaryReqs[dietDisplay] += m.RSVPData.Headcount
		}

		_ = config.SaveGuestRSVPToMarkdown(config.GuestRSVPRecord{
			Name:      m.RSVPData.Name,
			Phone:     m.RSVPData.Phone,
			Status:    m.RSVPData.Status,
			Headcount: m.RSVPData.Headcount,
			Dietary:   dietDisplay,
			Cab:       m.RSVPData.Cab,
		})

		m.Logs = append(m.Logs, LogEntry{
			Sender: "WIZARD",
			Text:   summary,
		})

		m.updateViewportContent()

		// Submit structured prompt to Go ADK & Honcho memory
		promptText := fmt.Sprintf(
			"Record RSVP for guest %s (Phone: %s): Status is %s, Headcount is %d, Dietary is %s, Cab requested is %v.",
			m.RSVPData.Name, m.RSVPData.Phone, m.RSVPData.Status, m.RSVPData.Headcount, dietDisplay, m.RSVPData.Cab,
		)

		m.Loading = true
		m.StatusMsg = "Saving RSVP record to database & Honcho memory..."
		return m, tea.Batch(
			m.Spinner.Tick,
			m.runAgentCmd(promptText),
		)
	}

	m.updateViewportContent()
	return m, nil
}
