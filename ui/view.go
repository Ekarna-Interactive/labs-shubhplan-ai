package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Palette Colors
	goldColor    = lipgloss.Color("#D4AF37")
	crimsonColor = lipgloss.Color("#E11D48")
	tealColor    = lipgloss.Color("#06B6D4")
	slateColor   = lipgloss.Color("#94A3B8")
	darkBg       = lipgloss.Color("#0F172A")
	cardBg       = lipgloss.Color("#161926")

	// Header Banner Style
	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(goldColor).
				Background(darkBg).
				Padding(0, 1)

	// Step Progress Bar Styles
	stepActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(goldColor).
			Padding(0, 1)

	stepInactiveStyle = lipgloss.NewStyle().
				Foreground(slateColor).
				Background(cardBg).
				Padding(0, 1)

	stepArrowStyle = lipgloss.NewStyle().
			Foreground(tealColor)

	// Sidebar Styles
	sidebarTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(goldColor).
				MarginBottom(1)

	sidebarCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#334155")).
				Background(cardBg).
				Padding(0, 1).
				MarginBottom(1)

	sidebarLabelStyle = lipgloss.NewStyle().
				Foreground(slateColor).
				Bold(true)

	sidebarValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8FAFC"))

	// Log Entry Sender Badges
	badgeUser = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(goldColor).
			Padding(0, 1)

	badgeAI = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(tealColor).
		Padding(0, 1)

	badgeBuilder = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(crimsonColor).
			Padding(0, 1)

	badgeSystem = lipgloss.NewStyle().
			Foreground(slateColor).
			Bold(true)

	badgeError = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#DC2626")).
			Padding(0, 1)

	// Footer Help Style
	footerStyle = lipgloss.NewStyle().
			Foreground(slateColor).
			Background(darkBg).
			MarginTop(1)
)

// View renders the terminal user interface
func (m Model) View() string {
	if m.Width == 0 {
		return "Initializing Shubh CLI TUI Dashboard..."
	}

	var b strings.Builder

	// 1. Top Header Banner & Step Breadcrumb Progress Bar
	b.WriteString(renderHeaderBanner())
	b.WriteString("\n")
	b.WriteString(renderStepProgressBar(m.Step, m.IsSetupMode))
	b.WriteString("\n\n")

	// Calculate layout dimensions
	sidebarWidth := 30
	mainWidth := m.Width - sidebarWidth - 4
	if mainWidth < 40 {
		mainWidth = 40
	}

	// 2. Render Left Sidebar Dashboard
	sidebarStr := renderSidebarDashboard(m, sidebarWidth)

	// 3. Render Scrollable Main Viewport
	viewportBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(goldColor).
		Padding(0, 1).
		Width(mainWidth)

	viewportView := viewportBorder.Render(m.Viewport.View())

	// 4. Combine Sidebar and Viewport side-by-side using Lipgloss JoinHorizontal
	dashboard := lipgloss.JoinHorizontal(lipgloss.Top, sidebarStr, viewportView)
	b.WriteString(dashboard)
	b.WriteString("\n")

	// 5. Input Field / Loading Spinner
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tealColor).
		Padding(0, 1).
		Width(m.Width - 4)

	if m.IsSetupMode {
		b.WriteString(inputStyle.Render("🔑 Setup Gemini Key: " + m.SetupInput.View()))
	} else if m.Loading {
		b.WriteString(inputStyle.Render(fmt.Sprintf("%s %s...", m.Spinner.View(), m.StatusMsg)))
	} else {
		b.WriteString(inputStyle.Render("❯ " + m.TextInput.View()))
	}
	b.WriteString("\n")

	// 6. Keyboard Shortcuts Footer Bar
	b.WriteString(renderFooterBar(m))

	return b.String()
}

func renderHeaderBanner() string {
	title := " ✨ SHUBH CLI — Open-Source AI Event Design Terminal "
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(goldColor).
		Align(lipgloss.Center).
		Padding(0, 2).
		Render(headerTitleStyle.Render(title))
}

func renderStepProgressBar(currentStep WizardStep, isSetup bool) string {
	steps := []struct {
		Name string
		Step WizardStep
	}{
		{"🔑 API Key", StepAPIKey},
		{"📋 Event Details", StepEventType},
		{"📐 Aspect Ratio", StepAspectSelection},
		{"🎨 Design Style", StepStyleSelection},
		{"🤖 AI Prompts", StepPromptChoice},
		{"✨ Render & Preview", StepComplete},
	}

	renderedSteps := []string{}
	for _, s := range steps {
		if (isSetup && s.Step == StepAPIKey) || (!isSetup && s.Step <= currentStep) {
			renderedSteps = append(renderedSteps, stepActiveStyle.Render(s.Name))
		} else {
			renderedSteps = append(renderedSteps, stepInactiveStyle.Render(s.Name))
		}
	}

	sep := stepArrowStyle.Render(" ➔ ")
	return strings.Join(renderedSteps, sep)
}

func renderSidebarDashboard(m Model, width int) string {
	var b strings.Builder

	// Title
	b.WriteString(sidebarTitleStyle.Render("📊 DASHBOARD"))
	b.WriteString("\n")

	// 1. Active Event Profile Card
	eventDetailsText := m.EventDetails
	if m.EventType != "" || m.HostNames != "" {
		parts := []string{}
		if m.EventType != "" {
			parts = append(parts, m.EventType)
		}
		if m.HostNames != "" {
			parts = append(parts, "for "+m.HostNames)
		}
		if m.EventDate != "" {
			parts = append(parts, "on "+m.EventDate)
		}
		if m.Venue != "" {
			parts = append(parts, "at "+m.Venue)
		}
		eventDetailsText = strings.Join(parts, " ")
	}
	if eventDetailsText == "" {
		eventDetailsText = "Not configured yet"
	} else if len(eventDetailsText) > 45 {
		eventDetailsText = eventDetailsText[:42] + "..."
	}

	profileCard := fmt.Sprintf("%s\n%s",
		sidebarLabelStyle.Render("📋 EVENT PROFILE:"),
		sidebarValueStyle.Render(eventDetailsText),
	)
	b.WriteString(sidebarCardStyle.Width(width - 2).Render(profileCard))
	b.WriteString("\n")

	// 2. Selected Aspect Ratio Card
	aspectLabel := "9:16 Mobile Story"
	switch m.SelectedAspect {
	case "4:5":
		aspectLabel = "4:5 Social Feed"
	case "1:1":
		aspectLabel = "1:1 Square Card"
	case "16:9":
		aspectLabel = "16:9 Desktop Banner"
	case "9:16":
		aspectLabel = "9:16 Mobile Poster"
	}

	aspectCard := fmt.Sprintf("%s\n%s",
		sidebarLabelStyle.Render("📐 TARGET RESOLUTION:"),
		sidebarValueStyle.Render(aspectLabel),
	)
	b.WriteString(sidebarCardStyle.Width(width - 2).Render(aspectCard))
	b.WriteString("\n")

	// 3. Selected Style & Palette Card
	styleText := m.SelectedStyle
	if styleText == "" {
		styleText = "South Indian Traditional"
	} else if len(styleText) > 24 {
		styleText = styleText[:22] + "..."
	}

	styleCard := fmt.Sprintf("%s\n%s\n%s",
		sidebarLabelStyle.Render("🎨 DESIGN AESTHETIC:"),
		sidebarValueStyle.Render(styleText),
		lipgloss.NewStyle().Foreground(goldColor).Render("🟡 Gold | 🔴 Crimson"),
	)
	b.WriteString(sidebarCardStyle.Width(width - 2).Render(styleCard))
	b.WriteString("\n")

	// 4. Gemini Model Status Card
	keyStatus := "🟢 Configured"
	if m.Config.GeminiAPIKey == "" {
		keyStatus = "🟡 Offline Dry-Run"
	}

	displayModel := m.Config.ImageModel
	if displayModel == "" {
		displayModel = "Not Configured"
	}

	modelCard := fmt.Sprintf("%s\n%s\n%s",
		sidebarLabelStyle.Render("🤖 AI MODEL STATUS:"),
		sidebarValueStyle.Render(displayModel),
		sidebarValueStyle.Render(keyStatus),
	)
	b.WriteString(sidebarCardStyle.Width(width - 2).Render(modelCard))
	b.WriteString("\n")

	// 5. Live Web Preview Link Card
	webCard := fmt.Sprintf("%s\n%s",
		sidebarLabelStyle.Render("🌐 LIVE WEB PREVIEW:"),
		lipgloss.NewStyle().Foreground(tealColor).Render("http://localhost:3000"),
	)
	b.WriteString(sidebarCardStyle.Width(width - 2).Render(webCard))

	return b.String()
}

func renderFooterBar(m Model) string {
	scrollHint := "Scroll: Mouse Wheel / PgUp / PgDn"
	shortcuts := fmt.Sprintf("Status: %s • [Enter] Submit • %s • [/aspect] Ratio • [/event] Edit Details • [/reset] Reset • [Esc] Quit", m.StatusMsg, scrollHint)
	return footerStyle.Render(shortcuts)
}

func formatLogsForViewport(logs []LogEntry, width int) string {
	var b strings.Builder
	wrapWidth := width - 12
	if wrapWidth < 30 {
		wrapWidth = 30
	}
	textWrap := lipgloss.NewStyle().Width(wrapWidth)

	for _, entry := range logs {
		wrapped := textWrap.Render(entry.Text)
		switch entry.Sender {
		case "USER":
			b.WriteString(fmt.Sprintf("%s %s\n\n", badgeUser.Render(" USER "), wrapped))
		case "AI":
			b.WriteString(fmt.Sprintf("%s %s\n\n", badgeAI.Render(" AI PROMPTER "), wrapped))
		case "BUILDER":
			b.WriteString(fmt.Sprintf("%s %s\n\n", badgeBuilder.Render(" BUILDER "), wrapped))
		case "ERROR", "WARNING":
			b.WriteString(fmt.Sprintf("%s %s\n\n", badgeError.Render(" ERROR "), wrapped))
		default:
			b.WriteString(fmt.Sprintf("%s %s\n\n", badgeSystem.Render("ℹ "+entry.Sender+":"), wrapped))
		}
	}
	return b.String()
}
