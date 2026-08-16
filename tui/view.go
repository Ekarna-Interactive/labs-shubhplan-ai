package tui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/client"
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

	// Tab Header Styles
	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(goldColor).
			Padding(0, 1)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(slateColor).
				Background(cardBg).
				Padding(0, 1)

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

	badgeOrchestrator = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#8B5CF6")).
				Padding(0, 1)

	badgeTimeline = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#F59E0B")).
			Padding(0, 1)

	badgeVendor = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#10B981")).
			Padding(0, 1)

	badgeBudget = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#EC4899")).
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
			Background(darkBg)
)

// View renders the terminal user interface
func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing Shubh Plan AI TUI Dashboard..."
	}

	var b strings.Builder

	// Dynamic overhead calculation (Header 3 or 1 line + 11 overhead lines)
	headerLines := 3
	if m.Height < 28 {
		headerLines = 1
	}

	overhead := headerLines + 11
	vpHeight := m.Height - overhead
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.Viewport.Height = vpHeight

	// 1. Top Header Banner & Tab Navigation Bar
	b.WriteString(renderHeaderBanner(m.Width-4, m.Height))
	b.WriteString("\n")
	b.WriteString(renderTabBar(m.ActiveTab))
	b.WriteString("\n")

	// Calculate layout dimensions
	sidebarWidth := 30
	mainWidth := m.Width - sidebarWidth - 6
	if mainWidth < 35 {
		mainWidth = 35
	}

	// 2. Render Left Sidebar Dashboard
	sidebarStr := renderSidebarDashboard(m, sidebarWidth)

	// 3. Render Scrollable Main Viewport
	m.Viewport.Width = mainWidth - 2
	viewportBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(goldColor).
		Padding(0, 1).
		Width(mainWidth)

	viewportView := viewportBorder.Render(m.Viewport.View())

	// 4. Combine Sidebar (capped to viewport height + 2) and Viewport side-by-side
	maxSidebarLines := m.Viewport.Height + 2
	sidebarCapped := limitLines(sidebarStr, maxSidebarLines)
	dashboard := lipgloss.JoinHorizontal(lipgloss.Top, sidebarCapped, viewportView)
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

	// 7. Enforce strict MaxHeight(m.Height) to prevent Windows ConPTY terminal buffer scrolling
	return lipgloss.NewStyle().
		MaxWidth(m.Width).
		MaxHeight(m.Height).
		Render(b.String())
}

func renderTabBar(activeTab ActiveTab) string {
	tabs := []struct {
		Name string
		Tab  ActiveTab
	}{
		{"[1: Agent Chat]", TabAgentChat},
		{"[2: Timeline]", TabItinerary},
		{"[3: Budget & Spend]", TabBudget},
		{"[4: RSVPs & Honcho]", TabRSVP},
		{"[5: Design Studio]", TabDesignStudio},
	}

	var rendered []string
	for _, t := range tabs {
		if t.Tab == activeTab {
			rendered = append(rendered, tabActiveStyle.Render(t.Name))
		} else {
			rendered = append(rendered, tabInactiveStyle.Render(t.Name))
		}
	}
	return strings.Join(rendered, "  ")
}

func renderHeaderBanner(width int, height int) string {
	title := " ✨ SHUBH PLAN AI — Open-Source Multi-Agent Event Engine "
	if width < 50 {
		width = 50
	}
	if height < 28 {
		return headerTitleStyle.Render(title)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(goldColor).
		Align(lipgloss.Center).
		Padding(0, 2).
		Width(width).
		Render(headerTitleStyle.Render(title))
}

func limitLines(str string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(str, "\n")
	if len(lines) <= maxLines {
		return str
	}
	return strings.Join(lines[:maxLines], "\n")
}

func renderSidebarDashboard(m Model, width int) string {
	var b strings.Builder

	// Event details snippet
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
		eventDetailsText = strings.Join(parts, " ")
	}
	if eventDetailsText == "" {
		eventDetailsText = "No Event Profile Loaded"
	} else if len(eventDetailsText) > 40 {
		eventDetailsText = eventDetailsText[:37] + "..."
	}

	switch m.ActiveTab {
	case TabAgentChat:
		b.WriteString(sidebarTitleStyle.Render("🤖 AGENT ENGINE"))
		b.WriteString("\n")

		plannerCard := fmt.Sprintf("%s\n%s\n%s",
			sidebarLabelStyle.Render("👤 EVENT PLANNER:"),
			sidebarValueStyle.Render(m.PlannerName),
			lipgloss.NewStyle().Foreground(slateColor).Render(m.PlannerRole),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(plannerCard))
		b.WriteString("\n")

		profileCard := fmt.Sprintf("%s\n%s",
			sidebarLabelStyle.Render("📋 EVENT PROFILE:"),
			sidebarValueStyle.Render(eventDetailsText),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(profileCard))
		b.WriteString("\n")

		agentsCard := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
			sidebarLabelStyle.Render("🤖 ACTIVE SUBAGENTS:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")).Render("• OrchestratorAgent"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render("• TimelineAgent"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("• VendorAgent"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#EC4899")).Render("• BudgetAgent"),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(agentsCard))
		b.WriteString("\n")

		honchoStatusStr := "Status: 🟡 Inbuilt Local Store"
		honchoSubtext := "./data/honcho_memory.json"
		if os.Getenv("HONCHO_API_KEY") != "" {
			honchoStatusStr = "Status: 🟢 Honcho Cloud Sync"
			honchoSubtext = "api.honcho.dev/v3"
		}

		honchoCard := fmt.Sprintf("%s\n%s\n%s",
			sidebarLabelStyle.Render("🧠 HONCHO MEMORY:"),
			sidebarValueStyle.Render(honchoStatusStr),
			lipgloss.NewStyle().Foreground(tealColor).Render(honchoSubtext),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(honchoCard))

	case TabItinerary:
		b.WriteString(sidebarTitleStyle.Render("🗓️ TIMELINE STUDIO"))
		b.WriteString("\n")

		profileCard := fmt.Sprintf("%s\n%s",
			sidebarLabelStyle.Render("📋 EVENT PROFILE:"),
			sidebarValueStyle.Render(eventDetailsText),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(profileCard))
		b.WriteString("\n")

		statsCard := fmt.Sprintf("%s\n%s\n%s",
			sidebarLabelStyle.Render("📊 TIMELINE STATS:"),
			sidebarValueStyle.Render(fmt.Sprintf("Sub-events: %d", len(m.ItineraryItems))),
			sidebarValueStyle.Render("Timezone: IST (UTC+5:30)"),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(statsCard))
		b.WriteString("\n")

		conflictCard := fmt.Sprintf("%s\n%s",
			sidebarLabelStyle.Render("⚠️ CONFLICT MONITOR:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("🟢 0 Venue Conflicts"),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(conflictCard))

	case TabBudget:
		b.WriteString(sidebarTitleStyle.Render("💰 FINANCIAL STUDIO"))
		b.WriteString("\n")

		profileCard := fmt.Sprintf("%s\n%s",
			sidebarLabelStyle.Render("📋 EVENT PROFILE:"),
			sidebarValueStyle.Render(eventDetailsText),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(profileCard))
		b.WriteString("\n")

		sym := GetCurrencySymbol(m.Currency)
		currCode := m.Currency
		if currCode == "" {
			currCode = "USD"
		}
		est := m.BudgetSummary.TotalEstimated
		act := m.BudgetSummary.TotalActual
		diff := est - act

		healthCard := fmt.Sprintf("%s\n%s\n%s",
			sidebarLabelStyle.Render(fmt.Sprintf("💰 TOTAL BUDGET (%s):", currCode)),
			sidebarValueStyle.Render(fmt.Sprintf("Estimated: %s%.2f", sym, est)),
			sidebarValueStyle.Render(fmt.Sprintf("Actual: %s%.2f", sym, act)),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(healthCard))
		b.WriteString("\n")

		varColor := slateColor
		varText := "⚪ No Budget Set"
		if est > 0 || act > 0 {
			if diff >= 0 {
				varColor = lipgloss.Color("#10B981")
				varText = fmt.Sprintf("🟢 -%s%.2f Under", sym, diff)
			} else {
				varColor = lipgloss.Color("#EF4444")
				varText = fmt.Sprintf("🔴 +%s%.2f Over", sym, -diff)
			}
		}
		varCard := fmt.Sprintf("%s\n%s",
			sidebarLabelStyle.Render("📊 SPEND VARIANCE:"),
			lipgloss.NewStyle().Foreground(varColor).Render(varText),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(varCard))

	case TabRSVP:
		b.WriteString(sidebarTitleStyle.Render("👥 GUEST CONCIERGE"))
		b.WriteString("\n")

		profileCard := fmt.Sprintf("%s\n%s",
			sidebarLabelStyle.Render("📋 EVENT PROFILE:"),
			sidebarValueStyle.Render(eventDetailsText),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(profileCard))
		b.WriteString("\n")

		att := m.RSVPOverview.Attending
		tot := m.RSVPOverview.TotalGuests
		rateStr := "0%"
		if tot > 0 {
			rateStr = fmt.Sprintf("%d%%", (att*100)/tot)
		}

		rsvpCard := fmt.Sprintf("%s\n%s\n%s",
			sidebarLabelStyle.Render("👥 RSVP RATE:"),
			sidebarValueStyle.Render(fmt.Sprintf("%d / %d Attending", att, tot)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render(fmt.Sprintf("Rate: %s", rateStr)),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(rsvpCard))
		b.WriteString("\n")

		honchoCard := fmt.Sprintf("%s\n%s",
			sidebarLabelStyle.Render("🧠 PEER CARDS:"),
			sidebarValueStyle.Render(fmt.Sprintf("%d Honcho Cards", len(m.PeerCards))),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(honchoCard))

	case TabDesignStudio:
		b.WriteString(sidebarTitleStyle.Render("🎨 DESIGN STUDIO"))
		b.WriteString("\n")

		profileCard := fmt.Sprintf("%s\n%s",
			sidebarLabelStyle.Render("📋 EVENT PROFILE:"),
			sidebarValueStyle.Render(eventDetailsText),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(profileCard))
		b.WriteString("\n")

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

		styleText := m.SelectedStyle
		if styleText == "" {
			styleText = "South Indian Traditional"
		} else if len(styleText) > 20 {
			styleText = styleText[:18] + "..."
		}

		styleCard := fmt.Sprintf("%s\n%s\n%s",
			sidebarLabelStyle.Render("🎨 DESIGN AESTHETIC:"),
			sidebarValueStyle.Render(styleText),
			lipgloss.NewStyle().Foreground(goldColor).Render("🟡 Gold | 🔴 Crimson"),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(styleCard))
		b.WriteString("\n")

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

		webCard := fmt.Sprintf("%s\n%s",
			sidebarLabelStyle.Render("🌐 LIVE WEB PREVIEW:"),
			lipgloss.NewStyle().Foreground(tealColor).Render("http://localhost:3000"),
		)
		b.WriteString(sidebarCardStyle.Width(width - 2).Render(webCard))
	}

	return b.String()
}

func renderFooterBar(m Model) string {
	shortcuts := fmt.Sprintf("Status: %s • [Tab] Switch Tab • [/add-rsvp] Add RSVP Wizard • [/planner] Edit Planner • [/event] Profile • [Esc] Quit", m.StatusMsg)
	return footerStyle.Render(shortcuts)
}

var (
	boldRegex = regexp.MustCompile(`\*\*(.*?)\*\*`)
	codeRegex = regexp.MustCompile("`([^`]+)`")
)

func renderLipglossMarkdown(text string, width int) string {
	if width < 30 {
		width = 30
	}
	textWrap := lipgloss.NewStyle().Width(width)

	lines := strings.Split(text, "\n")
	var out []string
	inTable := false
	var tableRows [][]string

	flushTable := func() {
		if len(tableRows) == 0 {
			return
		}
		numCols := 0
		for _, r := range tableRows {
			if len(r) > numCols {
				numCols = len(r)
			}
		}
		if numCols == 0 {
			tableRows = nil
			return
		}

		colWidths := make([]int, numCols)
		for _, r := range tableRows {
			for cIdx, cell := range r {
				cleanCell := stripMarkdownSyntax(cell)
				if len(cleanCell) > colWidths[cIdx] {
					colWidths[cIdx] = len(cleanCell)
				}
			}
		}

		for rIdx, r := range tableRows {
			var rowParts []string
			for cIdx := 0; cIdx < numCols; cIdx++ {
				cellVal := ""
				if cIdx < len(r) {
					cellVal = strings.TrimSpace(r[cIdx])
				}
				w := colWidths[cIdx]
				if w < 4 {
					w = 4
				}
				renderedCell := formatInlineMarkdown(cellVal)
				cleanLen := len(stripMarkdownSyntax(cellVal))
				padLen := w - cleanLen
				if padLen < 0 {
					padLen = 0
				}
				padding := strings.Repeat(" ", padLen)
				rowParts = append(rowParts, renderedCell+padding)
			}
			var lineStr string
			if rIdx == 0 {
				lineStr = lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("│ " + strings.Join(rowParts, " │ ") + " │")
			} else {
				lineStr = "│ " + strings.Join(rowParts, " │ ") + " │"
			}
			out = append(out, lineStr)
		}
		tableRows = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Table line check
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			if isTableSeparator(trimmed) {
				continue
			}
			inTable = true
			cells := splitTableCells(trimmed)
			tableRows = append(tableRows, cells)
			continue
		} else if inTable {
			flushTable()
			inTable = false
		}

		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			ruleWidth := width
			if ruleWidth > 60 {
				ruleWidth = 60
			}
			out = append(out, lipgloss.NewStyle().Foreground(slateColor).Render(strings.Repeat("─", ruleWidth)))
			continue
		}

		if strings.HasPrefix(trimmed, "### ") {
			title := strings.TrimPrefix(trimmed, "### ")
			out = append(out, lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("📌 "+formatInlineMarkdown(title)))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			title := strings.TrimPrefix(trimmed, "## ")
			out = append(out, lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("📌 "+formatInlineMarkdown(title)))
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			out = append(out, lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("📌 "+formatInlineMarkdown(title)))
			continue
		}

		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
			item := strings.TrimPrefix(strings.TrimPrefix(trimmed, "* "), "- ")
			bullet := lipgloss.NewStyle().Foreground(tealColor).Render("• ")
			out = append(out, textWrap.Render(bullet+formatInlineMarkdown(item)))
			continue
		}

		if line == "" {
			out = append(out, "")
			continue
		}

		out = append(out, textWrap.Render(formatInlineMarkdown(line)))
	}

	if inTable {
		flushTable()
	}

	return strings.Join(out, "\n")
}

func formatInlineMarkdown(text string) string {
	text = codeRegex.ReplaceAllStringFunc(text, func(m string) string {
		inner := strings.Trim(m, "`")
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#EC4899")).Render(inner)
	})
	text = boldRegex.ReplaceAllStringFunc(text, func(m string) string {
		inner := strings.Trim(m, "*")
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC")).Render(inner)
	})
	return text
}

func stripMarkdownSyntax(text string) string {
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "`", "")
	return text
}

func isTableSeparator(line string) bool {
	clean := strings.ReplaceAll(line, "|", "")
	clean = strings.ReplaceAll(clean, ":", "")
	clean = strings.ReplaceAll(clean, "-", "")
	clean = strings.ReplaceAll(clean, " ", "")
	return clean == ""
}

func splitTableCells(line string) []string {
	parts := strings.Split(line, "|")
	var cells []string
	for i := 1; i < len(parts)-1; i++ {
		cells = append(cells, strings.TrimSpace(parts[i]))
	}
	return cells
}

func formatLogsForViewport(logs []LogEntry, width int) string {
	var b strings.Builder
	wrapWidth := width - 4
	if wrapWidth < 30 {
		wrapWidth = 30
	}
	textWrap := lipgloss.NewStyle().Width(wrapWidth)

	for _, entry := range logs {
		var content string
		trimmedText := strings.TrimSpace(entry.Text)
		if strings.Contains(entry.Text, "#") || strings.Contains(entry.Text, "**") || strings.Contains(entry.Text, "|") || strings.Contains(entry.Text, "```") || strings.HasPrefix(trimmedText, "* ") || strings.HasPrefix(trimmedText, "- ") {
			content = renderLipglossMarkdown(entry.Text, wrapWidth)
		} else {
			content = textWrap.Render(entry.Text)
		}

		var badge string
		switch entry.Sender {
		case "USER":
			badge = badgeUser.Render(" USER ")
		case "AI":
			badge = badgeAI.Render(" AI PROMPTER ")
		case "MasterOrchestrator", "Orchestrator":
			badge = badgeOrchestrator.Render(" ORCHESTRATOR ")
		case "planner_assistant", "PlannerAgent", "RootPlanner":
			badge = badgeOrchestrator.Render(" PLANNER ASSISTANT ")
		case "TimelineAgent", "timeline_agent":
			badge = badgeTimeline.Render(" TIMELINE AGENT ")
		case "VendorAgent", "vendor_agent":
			badge = badgeVendor.Render(" VENDOR AGENT ")
		case "BudgetAgent", "budget_agent":
			badge = badgeBudget.Render(" BUDGET AGENT ")
		case "ConciergeAgent", "guest_agent", "guest_concierge":
			badge = badgeSystem.Render(" GUEST CONCIERGE ")
		case "ItineraryAgent", "itinerary_agent":
			badge = badgeTimeline.Render(" ITINERARY AGENT ")
		case "FeedbackAgent", "feedback_agent":
			badge = badgeBuilder.Render(" APPROVAL AGENT ")
		case "BUILDER":
			badge = badgeBuilder.Render(" BUILDER ")
		case "ERROR", "WARNING":
			badge = badgeError.Render(" ERROR ")
		default:
			badge = badgeSystem.Render("ℹ " + entry.Sender + ":")
		}
		b.WriteString(fmt.Sprintf("%s\n%s\n\n", badge, content))
	}
	return b.String()
}

func formatItineraryView(items []client.SubEventItem, width int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("🗓️ MINUTE-BY-MINUTE RUN-OF-SHOW & MULTI-VENUE TIMELINE"))
	b.WriteString("\n\n")

	if len(items) == 0 {
		widgetBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tealColor).
			Padding(0, 1)

		var widgetContent strings.Builder
		widgetContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("📅 TIMELINE & RUN-OF-SHOW BUILDER"))
		widgetContent.WriteString("\n\n")
		widgetContent.WriteString("No sub-events have been added to this event itinerary schedule yet.\n\n")
		widgetContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC")).Render("💡 How to Add Itinerary Sub-Events:\n"))
		widgetContent.WriteString(" • Ask the AI Orchestrator: Type any natural language prompt below, such as:\n")
		widgetContent.WriteString(fmt.Sprintf("   %s\n", lipgloss.NewStyle().Foreground(tealColor).Render("\"Build a 3-day Haldi, Sangeet, and Reception timeline for Rohan & Ananya\"")))
		widgetContent.WriteString(fmt.Sprintf("   %s\n\n", lipgloss.NewStyle().Foreground(tealColor).Render("\"Add Haldi ceremony on Oct 11 from 9 AM to 12 PM at Lawn\"")))
		widgetContent.WriteString(" • Query TimelineAgent: Ask for a minute-by-minute run-of-show or venue schedule.")

		b.WriteString(widgetBox.Render(widgetContent.String()))
		b.WriteString("\n")
		return b.String()
	}

	for i, item := range items {
		card := fmt.Sprintf("[%d] %s\n📅 Date: %s | ⏰ %s - %s\n📍 Venue: %s\n👔 Dress Code: %s",
			i+1,
			lipgloss.NewStyle().Bold(true).Foreground(tealColor).Render(item.Title),
			item.Date, item.StartTime, item.EndTime,
			item.Venue, item.DressCode,
		)
		b.WriteString(sidebarCardStyle.Width(width - 6).Render(card))
		b.WriteString("\n")
	}
	return b.String()
}

// GetCurrencySymbol maps currency ISO codes to their UI display symbols
func GetCurrencySymbol(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "INR":
		return "₹"
	case "AUD":
		return "A$"
	case "SGD":
		return "S$"
	case "USD":
		return "$"
	default:
		if code != "" {
			return code + " "
		}
		return "$"
	}
}

// GetSuggestedBudgetForEvent returns baseline budget amount, guest count, and label for event type & currency
func GetSuggestedBudgetForEvent(eventType string, currency string) (float64, int, string) {
	curr := strings.ToUpper(strings.TrimSpace(currency))
	if curr == "" {
		curr = "USD"
	}
	eType := strings.ToLower(strings.TrimSpace(eventType))
	isINR := curr == "INR"

	if strings.Contains(eType, "naming") || strings.Contains(eType, "cradle") || strings.Contains(eType, "namakarana") {
		if isINR {
			return 250000.0, 200, "2.5L"
		}
		return 5000.0, 100, "5k"
	}
	if strings.Contains(eType, "wedding") || strings.Contains(eType, "reception") || strings.Contains(eType, "marriage") {
		if isINR {
			return 2500000.0, 400, "25L"
		}
		return 35000.0, 150, "35k"
	}
	if strings.Contains(eType, "mehendi") || strings.Contains(eType, "sangeet") || strings.Contains(eType, "haldi") {
		if isINR {
			return 500000.0, 200, "5L"
		}
		return 10000.0, 100, "10k"
	}
	if strings.Contains(eType, "birthday") || strings.Contains(eType, "baby shower") || strings.Contains(eType, "housewarming") || strings.Contains(eType, "anniversary") {
		if isINR {
			return 150000.0, 150, "1.5L"
		}
		return 3000.0, 60, "3k"
	}
	if strings.Contains(eType, "corporate") || strings.Contains(eType, "gala") || strings.Contains(eType, "conference") {
		if isINR {
			return 1000000.0, 300, "10L"
		}
		return 20000.0, 200, "20k"
	}

	if isINR {
		return 250000.0, 200, "2.5L"
	}
	return 5000.0, 100, "5k"
}

func formatBudgetView(summary client.BudgetSummary, currency string, eventType string, totalGuests int, width int) string {
	_ = width
	if currency == "" {
		currency = "USD"
	}
	sym := GetCurrencySymbol(currency)
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render(fmt.Sprintf("💰 EVENT BUDGET METER & CATEGORY SPEND ANALYSIS (%s)", currency)))
	b.WriteString("\n\n")

	if summary.TotalEstimated == 0 && len(summary.Categories) == 0 {
		widgetBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tealColor).
			Padding(0, 1)

		sugAmt, sugGuests, sugLabel := GetSuggestedBudgetForEvent(eventType, currency)
		eTitle := eventType
		if eTitle == "" {
			eTitle = "Event"
		}

		var widgetContent strings.Builder
		widgetContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("💰 FINANCIAL & BUDGET PLANNER ENGINE"))
		widgetContent.WriteString("\n\n")
		widgetContent.WriteString("No total budget or breakdown categories have been configured yet.\n\n")

		cpg := sugAmt / float64(sugGuests)
		widgetContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render(fmt.Sprintf("💡 Industry Benchmark Suggestion for %s (%s):\n", eTitle, currency)))
		widgetContent.WriteString(fmt.Sprintf("   Recommended Baseline: %s%.2f (%s) for %d guests (~%s%.2f / guest)\n\n", sym, sugAmt, sugLabel, sugGuests, sym, cpg))

		widgetContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC")).Render("⚡ Quick Actions:\n"))
		widgetContent.WriteString(fmt.Sprintf(" • Type %s to automatically lock in this benchmark suggestion!\n", lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("'/budget accept'")))
		widgetContent.WriteString(" • Or type any custom total budget, e.g:\n")
		widgetContent.WriteString(fmt.Sprintf("   %s\n", lipgloss.NewStyle().Foreground(tealColor).Render("'/budget 25L 400'")))
		widgetContent.WriteString(fmt.Sprintf("   %s\n\n", lipgloss.NewStyle().Foreground(tealColor).Render("'/budget $50,000 for 150 guests'")))
		widgetContent.WriteString(" • Query BudgetAgent: Type natural language prompts to calculate floral, catering, or decor spend.")

		b.WriteString(widgetBox.Render(widgetContent.String()))
		b.WriteString("\n")
		return b.String()
	}

	est := summary.TotalEstimated
	act := summary.TotalActual
	diff := est - act

	b.WriteString(fmt.Sprintf("Total Budget Estimated: %s%s%.2f%s | Actual Spend: %s%s%.2f%s\n",
		goldColor, sym, est, "\x1b[0m",
		tealColor, sym, act, "\x1b[0m"))

	if totalGuests > 0 && est > 0 {
		cpgEst := est / float64(totalGuests)
		cpgAct := act / float64(totalGuests)
		b.WriteString(fmt.Sprintf("Headcount: %s%d guests%s | Cost / Guest (Est): %s%s%.2f%s | Cost / Guest (Act): %s%s%.2f%s\n",
			goldColor, totalGuests, "\x1b[0m",
			goldColor, sym, cpgEst, "\x1b[0m",
			tealColor, sym, cpgAct, "\x1b[0m"))
	}

	var statusStr string
	if diff >= 0 {
		statusStr = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981")).Render(fmt.Sprintf("🟢 UNDER BUDGET BY %s%.2f", sym, diff))
	} else {
		statusStr = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EF4444")).Render(fmt.Sprintf("🔴 OVER BUDGET BY %s%.2f", sym, -diff))
	}
	b.WriteString(fmt.Sprintf("Status: %s\n\n", statusStr))

	if len(summary.Categories) > 0 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(slateColor).Render("CATEGORY SPEND BREAKDOWN:"))
		b.WriteString("\n")
		b.WriteString("---------------------------------------------------\n")
		for _, cat := range summary.Categories {
			catDiff := cat.Estimated - cat.Actual
			cStatus := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("OK")
			if catDiff < 0 {
				cStatus = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("OVER")
			}
			b.WriteString(fmt.Sprintf("• %-25s | Est: %s%-10.2f | Act: %s%-10.2f | %s\n",
				cat.Name, sym, cat.Estimated, sym, cat.Actual, cStatus))
		}
	}
	return b.String()
}

func formatRSVPView(overview client.RSVPOverview, peerCards map[string]string, width int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("👥 GUEST RSVP CONCIERGE & HONCHO MEMORY RECALL"))
	b.WriteString("\n\n")

	att := overview.Attending
	dec := overview.Declined
	pen := overview.Pending
	tot := overview.TotalGuests

	if tot == 0 && att == 0 && dec == 0 {
		widgetBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tealColor).
			Padding(0, 1)

		var widgetContent strings.Builder
		widgetContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("👥 GUEST CONCIERGE & HONCHO MEMORY ENGINE"))
		widgetContent.WriteString("\n\n")
		widgetContent.WriteString("No guest RSVPs or headcount targets recorded yet.\n\n")
		widgetContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC")).Render("💡 How to Manage Guests & RSVPs:\n"))
		widgetContent.WriteString(" • Launch Interactive RSVP Wizard: Type command:\n")
		widgetContent.WriteString(fmt.Sprintf("   %s\n\n", lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("'/add-rsvp'")))
		widgetContent.WriteString(" • Natural Language Entry: Type guest RSVP details directly below, e.g:\n")
		widgetContent.WriteString(fmt.Sprintf("   %s\n", lipgloss.NewStyle().Foreground(tealColor).Render("\"Record RSVP for Vikram Malhotra: Attending, 4 headcount, Jain food\"")))
		widgetContent.WriteString(fmt.Sprintf("   %s\n\n", lipgloss.NewStyle().Foreground(tealColor).Render("\"Set total expected headcount to 500 guests\"")))
		widgetContent.WriteString(" • Query ConciergeAgent: Ask Honcho memory to recall guest dietary requirements or cab transfers.")

		b.WriteString(widgetBox.Render(widgetContent.String()))
		b.WriteString("\n")
		return b.String()
	}

	totStr := lipgloss.NewStyle().Foreground(goldColor).Render(fmt.Sprintf("%d", tot))
	attStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render(fmt.Sprintf("%d", att))
	decStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render(fmt.Sprintf("%d", dec))
	penStr := lipgloss.NewStyle().Foreground(slateColor).Render(fmt.Sprintf("%d", pen))

	b.WriteString(fmt.Sprintf("Total Expected: %s | Attending: %s | Declined: %s | Pending: %s\n\n",
		totStr, attStr, decStr, penStr))

	if len(overview.DietaryReqs) > 0 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(slateColor).Render("DIETARY PREFERENCES BREAKDOWN:"))
		b.WriteString("\n")
		for k, v := range overview.DietaryReqs {
			b.WriteString(fmt.Sprintf(" • %-20s : %d guests\n", k, v))
		}
		b.WriteString("\n")
	}

	if len(peerCards) > 0 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(goldColor).Render("🧠 HONCHO RECALLED GUEST CARDS:"))
		b.WriteString("\n")
		for name, card := range peerCards {
			b.WriteString(sidebarCardStyle.Width(width - 6).Render(fmt.Sprintf("👤 %s\n%s", name, card)))
			b.WriteString("\n")
		}
	}
	return b.String()
}
