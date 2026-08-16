package tui

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/client"
)

func parseBudgetAmount(raw string, currency string) float64 {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, ",", "")
	raw = strings.ReplaceAll(raw, "$", "")
	raw = strings.ReplaceAll(raw, "₹", "")
	raw = strings.ReplaceAll(raw, "€", "")
	raw = strings.ReplaceAll(raw, "£", "")
	raw = strings.ReplaceAll(raw, "a$", "")
	raw = strings.ReplaceAll(raw, "s$", "")

	if raw == "" {
		return 0
	}

	curr := strings.ToUpper(strings.TrimSpace(currency))
	if curr == "" {
		curr = "USD"
	}

	multiplier := 1.0

	if curr == "INR" {
		// INR Notation: Lakhs (L / lakh), Crores (CR / crore), Thousands (K / k)
		if strings.Contains(raw, "crore") || strings.Contains(raw, "cr") {
			multiplier = 10000000.0
			parts := strings.FieldsFunc(raw, func(r rune) bool {
				return (r < '0' || r > '9') && r != '.'
			})
			if len(parts) > 0 {
				raw = parts[0]
			}
		} else if strings.Contains(raw, "lakh") || strings.Contains(raw, "lac") || strings.HasSuffix(raw, "l") || strings.Contains(raw, "l ") {
			multiplier = 100000.0
			parts := strings.FieldsFunc(raw, func(r rune) bool {
				return (r < '0' || r > '9') && r != '.'
			})
			if len(parts) > 0 {
				raw = parts[0]
			}
		} else if strings.HasSuffix(raw, "k") || strings.Contains(raw, "thousand") {
			multiplier = 1000.0
			raw = strings.TrimSuffix(raw, "k")
			raw = strings.ReplaceAll(raw, "thousand", "")
		} else if strings.HasSuffix(raw, "m") || strings.Contains(raw, "million") {
			multiplier = 1000000.0
			raw = strings.TrimSuffix(raw, "m")
			raw = strings.ReplaceAll(raw, "million", "")
		}
	} else {
		// Global/Western Currencies (USD, EUR, GBP, AUD, SGD): Thousands (K), Millions (M), Billions (B)
		if strings.HasSuffix(raw, "b") || strings.Contains(raw, "billion") {
			multiplier = 1000000000.0
			raw = strings.TrimSuffix(raw, "b")
			raw = strings.ReplaceAll(raw, "billion", "")
		} else if strings.HasSuffix(raw, "m") || strings.Contains(raw, "million") {
			multiplier = 1000000.0
			raw = strings.TrimSuffix(raw, "m")
			raw = strings.ReplaceAll(raw, "million", "")
		} else if strings.HasSuffix(raw, "k") || strings.Contains(raw, "thousand") {
			multiplier = 1000.0
			raw = strings.TrimSuffix(raw, "k")
			raw = strings.ReplaceAll(raw, "thousand", "")
		} else if strings.Contains(raw, "lakh") || strings.Contains(raw, "lac") {
			multiplier = 100000.0
			parts := strings.FieldsFunc(raw, func(r rune) bool {
				return (r < '0' || r > '9') && r != '.'
			})
			if len(parts) > 0 {
				raw = parts[0]
			}
		} else if strings.Contains(raw, "crore") || strings.Contains(raw, "cr") {
			multiplier = 10000000.0
			parts := strings.FieldsFunc(raw, func(r rune) bool {
				return (r < '0' || r > '9') && r != '.'
			})
			if len(parts) > 0 {
				raw = parts[0]
			}
		}
	}

	raw = strings.TrimSpace(raw)
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return val * multiplier
}

func parseGuestCount(raw string) int {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "guests", "")
	raw = strings.ReplaceAll(raw, "guest", "")
	raw = strings.TrimSpace(raw)
	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return val
}

func parseBudgetMutationFromPrompt(text string, currency string) float64 {
	lower := strings.ToLower(strings.TrimSpace(text))
	// If it's a question or lookup query, do NOT parse as a budget mutation!
	if strings.HasPrefix(lower, "what") || strings.HasPrefix(lower, "how") || strings.HasPrefix(lower, "check") || strings.HasPrefix(lower, "show") || strings.HasPrefix(lower, "get") || strings.HasPrefix(lower, "display") || strings.Contains(lower, "?") {
		return 0
	}
	re := regexp.MustCompile(`(?i)(?:set\s+budget|update\s+budget|budget\s*(?:is|=|:)|lock\s+budget|budget\s+to)\s*(?::|\||is|of|to)?\s*(?:[₹$€£]|INR|USD|EUR|GBP|AUD|SGD)?\s*([0-9,]+(?:\.[0-9]+)?|[0-9\.]+\s*(?:lakh|lakhs|l|cr|crore|k|m|million)?)`)
	m := re.FindStringSubmatch(text)
	if len(m) > 1 {
		return parseBudgetAmount(m[1], currency)
	}
	return 0
}

func parseGuestCountFromText(text string) int {
	// Pattern 1: guest count/list/headcount is/of/set at 500
	re1 := regexp.MustCompile(`(?i)(?:guest\s+(?:count|list)|headcount|expected\s+guest\s+count|count\s+of)\s*(?::|\||is|of|at)?\s*(?:set\s+at\s*)?([0-9]+)`)
	if m := re1.FindStringSubmatch(text); len(m) > 1 {
		if g, err := strconv.Atoi(m[1]); err == nil && g > 0 {
			return g
		}
	}
	// Pattern 2: 500 guests / 500 headcount / 500 people / 500 attendees / 500 pax
	re2 := regexp.MustCompile(`(?i)\b([0-9]+)\s*(?:guests|headcount|people|attendees|pax)\b`)
	if m := re2.FindStringSubmatch(text); len(m) > 1 {
		if g, err := strconv.Atoi(m[1]); err == nil && g > 0 {
			return g
		}
	}
	return 0
}

func parseBudgetFromAgentResponse(content string, currency string) (client.BudgetSummary, int, bool) {
	var summary client.BudgetSummary
	totalGuests := parseGuestCountFromText(content)
	hasData := totalGuests > 0

	reTotal := regexp.MustCompile(`(?i)(?:total\s+(?:event\s+)?budget|budget\s+(?:is|of))\s*(?::|\||is|of)?\s*(?:[₹$€£]|INR|USD|EUR|GBP|AUD|SGD)?\s*([0-9,]+(?:\.[0-9]+)?|[0-9\.]+\s*(?:lakh|lakhs|l|cr|crore|k|m|million)?)`)
	matchesTotal := reTotal.FindStringSubmatch(content)
	if len(matchesTotal) > 1 {
		amt := parseBudgetAmount(matchesTotal[1], currency)
		if amt > 0 {
			summary.TotalEstimated = amt
			hasData = true
		}
	}

	lines := strings.Split(content, "\n")
	var categories []client.BudgetCategory
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			if isTableSeparator(trimmed) {
				continue
			}
			cells := splitTableCells(trimmed)
			if len(cells) >= 3 {
				catName := strings.TrimSpace(stripMarkdownSyntax(cells[0]))
				if strings.EqualFold(catName, "category") || strings.EqualFold(catName, "metric") || strings.EqualFold(catName, "total") || strings.EqualFold(catName, "total budget") || strings.EqualFold(catName, "total event budget") {
					continue
				}
				lastCell := cells[len(cells)-1]
				amt := parseBudgetAmount(lastCell, currency)
				if amt > 0 {
					categories = append(categories, client.BudgetCategory{
						Name:      catName,
						Estimated: amt,
						Actual:    0,
					})
				}
			}
		}
	}

	if len(categories) > 0 {
		summary.Categories = categories
		hasData = true
		if summary.TotalEstimated == 0 {
			sumEst := 0.0
			for _, c := range categories {
				sumEst += c.Estimated
			}
			summary.TotalEstimated = sumEst
		}
	}

	return summary, totalGuests, hasData
}
