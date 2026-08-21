package tui

import (
	"fmt"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/generator"
)

var predefinedEventTypes = []string{
	"💍 Wedding",
	"👶 Naming Ceremony",
	"🍼 Baby Shower",
	"🏡 Housewarming",
	"🎂 Birthday",
	"🏢 Corporate Gala",
	"✏️ Custom Event Type...",
}

var predefinedEventTypesRaw = []string{
	"Wedding",
	"Naming Ceremony",
	"Baby Shower",
	"Housewarming",
	"Birthday",
	"Corporate Gala",
}

func renderEventTypeMenu(activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("📋 STEP 1 of 5: Select Event Type:\n")
	for i, opt := range predefinedEventTypes {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type custom event type below:")
	return sb.String()
}

func resolveEventTypeChoice(input string, activeIdx int) string {
	trimmed := strings.TrimSpace(input)
	switch trimmed {
	case "1":
		return "Wedding"
	case "2":
		return "Naming Ceremony"
	case "3":
		return "Baby Shower"
	case "4":
		return "Housewarming"
	case "5":
		return "Birthday"
	case "6":
		return "Corporate Gala"
	case "":
		if activeIdx >= 0 && activeIdx < len(predefinedEventTypesRaw) {
			return predefinedEventTypesRaw[activeIdx]
		}
		return "Wedding"
	default:
		return trimmed
	}
}

var predefinedAspects = []string{
	"📱 Mobile Story / Poster (9:16 Vertical)",
	"📸 Social Feed / Portrait (4:5 Vertical)",
	"⏹️ Square Card / Standard (1:1 Square)",
	"🖥️ Desktop / Blog Banner (16:9 Landscape)",
}

var predefinedAspectsRaw = []string{
	"9:16",
	"4:5",
	"1:1",
	"16:9",
}

func renderAspectMenu(activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("📐 STEP 2 of 4: Select Target Image Resolution / Aspect Ratio:\n")
	for i, opt := range predefinedAspects {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type 1-4 / aspect ratio:")
	return sb.String()
}

func resolveAspectChoice(input string, activeIdx int) string {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	switch trimmed {
	case "1", "mobile", "poster", "story", "9:16":
		return "9:16"
	case "2", "social", "portrait", "4:5":
		return "4:5"
	case "3", "square", "card", "1:1":
		return "1:1"
	case "4", "desktop", "banner", "landscape", "16:9":
		return "16:9"
	case "":
		if activeIdx >= 0 && activeIdx < len(predefinedAspectsRaw) {
			return predefinedAspectsRaw[activeIdx]
		}
		return "9:16"
	default:
		if strings.Contains(trimmed, "4:5") {
			return "4:5"
		} else if strings.Contains(trimmed, "1:1") {
			return "1:1"
		} else if strings.Contains(trimmed, "16:9") {
			return "16:9"
		}
		return "9:16"
	}
}

var predefinedStyles = []string{
	"South Indian Traditional (Imperial Gold & Royal Crimson)",
	"Paper Cut Art (Multi-layered craft paper & soft shadows)",
	"Clay 3D Render (Soft glossy pastel clay figurines)",
	"Pop Art (Vibrant retro halftone dots & bold outlines)",
	"Mughal Palace (Intricate arches & floral gold motifs)",
	"Minimalist Gold Foil (Clean pastel canvas & gold typography)",
	"Loose Watercolor (Soft pastel floral paint & fluid washes)",
}

func renderStyleMenu(activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("🎨 STEP 3 of 4: Select an aesthetic design style:\n")
	for i, opt := range predefinedStyles {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type custom style name:")
	return sb.String()
}

func resolveStyleChoice(input string, activeIdx int) string {
	trimmed := strings.TrimSpace(input)
	switch trimmed {
	case "1":
		return "South Indian Traditional (Imperial Gold & Royal Crimson)"
	case "2":
		return "Paper Cut Art (Multi-layered craft paper)"
	case "3":
		return "Clay 3D Render (Soft glossy clay)"
	case "4":
		return "Pop Art (Vibrant retro halftone dots)"
	case "5":
		return "Mughal Palace (Intricate floral gold arches)"
	case "6":
		return "Minimalist Gold Foil (Clean pastel canvas)"
	case "7":
		return "Loose Watercolor (Soft pastel floral paint)"
	case "":
		if activeIdx >= 0 && activeIdx < len(predefinedStyles) {
			return predefinedStyles[activeIdx]
		}
		return "South Indian Traditional (Imperial Gold & Royal Crimson)"
	default:
		return trimmed
	}
}

func renderWelcomeSuggestionMenu(options []string, activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("🤖 AI Welcome Subheader Suggestions:\n")
	for i, opt := range options {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	if activeIdx == 4 {
		sb.WriteString("  ❯ [ 5. 🔄 Generate 4 More AI Subheaders ]\n")
	} else {
		sb.WriteString("    5. 🔄 Generate 4 More AI Subheaders\n")
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type custom subheader below:")
	return sb.String()
}

func renderSuggestionMenu(suggestions []string, activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("🤖 AI Prompt Suggestions:\n")
	for i, opt := range suggestions {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	if activeIdx == 4 {
		sb.WriteString("  ❯ [ 5. 🔄 Generate 4 More AI Prompt Suggestions ]\n")
	} else {
		sb.WriteString("    5. 🔄 Generate 4 More AI Prompt Suggestions\n")
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type 1-4 / custom prompt below:")
	return sb.String()
}

var predefinedCurrencies = []string{
	"USD ($ - United States Dollar)",
	"EUR (€ - Euro)",
	"GBP (£ - British Pound)",
	"INR (₹ - Indian Rupee)",
	"AUD (A$ - Australian Dollar)",
	"SGD (S$ - Singapore Dollar)",
}

var predefinedCurrenciesRaw = []string{
	"USD",
	"EUR",
	"GBP",
	"INR",
	"AUD",
	"SGD",
}

func renderCurrencyMenu(activeIdx int) string {
	var sb strings.Builder
	sb.WriteString("💵 Select Default Budget & Event Currency:\n")
	for i, opt := range predefinedCurrencies {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, opt))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, opt))
		}
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select, or type currency code (e.g. USD, EUR, GBP, INR, AUD, SGD):")
	return sb.String()
}

func resolveCurrencyChoice(input string, activeIdx int) string {
	trimmed := strings.TrimSpace(strings.ToUpper(input))
	switch trimmed {
	case "1", "USD", "$":
		return "USD"
	case "2", "EUR", "€":
		return "EUR"
	case "3", "GBP", "£":
		return "GBP"
	case "4", "INR", "₹", "RS", "RUPEE", "RUPEES":
		return "INR"
	case "5", "AUD", "A$":
		return "AUD"
	case "6", "SGD", "S$":
		return "SGD"
	case "":
		if activeIdx >= 0 && activeIdx < len(predefinedCurrenciesRaw) {
			return predefinedCurrenciesRaw[activeIdx]
		}
		return "USD"
	default:
		if len(trimmed) == 3 {
			return trimmed
		}
		return "USD"
	}
}

func renderVenueMenu(items []generator.VenueSuggestion, activeIdx int, query string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📍 Select Google Places & AI Autocomplete match for '%s':\n", query))
	for i, item := range items {
		if i == activeIdx {
			sb.WriteString(fmt.Sprintf("  ❯ [ %d. %s ]\n", i+1, item.Text))
		} else {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, item.Text))
		}
	}
	sb.WriteString("\nPress ↑/↓ to navigate, Enter to select option, or type custom venue name:")
	return sb.String()
}

func resolveVenueChoice(input string, items []generator.VenueSuggestion, activeIdx int, fallback string) generator.VenueSuggestion {
	trimmed := strings.TrimSpace(input)
	if trimmed != "" {
		for i, item := range items {
			if fmt.Sprintf("%d", i+1) == trimmed {
				return item
			}
		}
		return generator.VenueSuggestion{
			PlaceID: "custom-" + trimmed,
			Text:    trimmed,
		}
	}
	if activeIdx >= 0 && activeIdx < len(items) {
		return items[activeIdx]
	}
	if len(items) > 0 {
		return items[0]
	}
	return generator.VenueSuggestion{
		PlaceID: "custom-" + fallback,
		Text:    fallback,
	}
}

func getOptionCountForStep(step WizardStep) int {
	switch step {
	case StepEventType:
		return len(predefinedEventTypes)
	case StepCurrency:
		return len(predefinedCurrencies)
	case StepAspectSelection:
		return len(predefinedAspects)
	case StepStyleSelection:
		return len(predefinedStyles)
	case StepAwaitingWelcomeChoice:
		return 5
	case StepAwaitingSuggestionChoice:
		return 5
	default:
		return 0
	}
}

func (m *Model) getOptionCount() int {
	if m.Step == StepVenueSelection {
		return len(m.VenueSuggestions)
	}
	return getOptionCountForStep(m.Step)
}

func (m *Model) updateActiveStepMenuText() {
	if len(m.Logs) == 0 {
		return
	}
	lastIdx := len(m.Logs) - 1
	switch m.Step {
	case StepEventType:
		m.Logs[lastIdx].Text = renderEventTypeMenu(m.OptionIndex)
	case StepCurrency:
		m.Logs[lastIdx].Text = renderCurrencyMenu(m.OptionIndex)
	case StepAspectSelection:
		m.Logs[lastIdx].Text = renderAspectMenu(m.OptionIndex)
	case StepStyleSelection:
		m.Logs[lastIdx].Text = renderStyleMenu(m.OptionIndex)
	case StepVenueSelection:
		m.Logs[lastIdx].Text = renderVenueMenu(m.VenueSuggestions, m.OptionIndex, m.VenueSearchQuery)
	case StepAwaitingWelcomeChoice:
		m.Logs[lastIdx].Text = renderWelcomeSuggestionMenu(m.WelcomeSuggestions, m.OptionIndex)
	case StepAwaitingSuggestionChoice:
		m.Logs[lastIdx].Text = renderSuggestionMenu(m.Suggestions, m.OptionIndex)
	}
}
