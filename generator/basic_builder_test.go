package generator

import (
	"strings"
	"testing"
)

func TestCompileWithAspect_SanitizesMetaPhrases(t *testing.T) {
	builder := NewBasicBuilder()

	// Test case 1: Exact reported prompt with camera shot prefix and typography instructions
	inputPrompt := `A full and complete shot of an elegant wedding invitation card styled in South Indian Traditional. Centered high-contrast legible event typography with text "Wedding of Priya & Arjun on Oct 12 2026, MCC Marriage Hall, Chennai" on a translucent background plate.`

	res := builder.CompileWithAspect(inputPrompt, "", "1:1")

	// 1. Verify display title is cleanly extracted event text
	expectedTitle := "Wedding of Priya & Arjun on Oct 12 2026, MCC Marriage Hall, Chennai"
	if res.DisplayTitle != expectedTitle {
		t.Errorf("Expected DisplayTitle to be %q, got %q", expectedTitle, res.DisplayTitle)
	}

	// 2. Verify core prompt does NOT contain "A full and complete shot of" or "invitation card" framing headers
	unwantedPhrases := []string{
		"A full and complete shot of an elegant wedding inv",
		"A full and complete shot of",
		"Centered high-contrast legible event typography with text",
		"1. Composition:",
		"2. Typography:",
	}

	for _, phrase := range unwantedPhrases {
		if strings.Contains(res.CorePrompt, phrase) {
			t.Errorf("CorePrompt contains unwanted meta-phrase %q:\n%s", phrase, res.CorePrompt)
		}
	}

	// 3. Verify core prompt contains mandatory text specification
	mandatorySpec := "MANDATORY TEXT TO RENDER ON CARD PLATE: 'Wedding of Priya & Arjun on Oct 12 2026, MCC Marriage Hall, Chennai'"
	if !strings.Contains(res.CorePrompt, mandatorySpec) {
		t.Errorf("CorePrompt missing mandatory text spec %q:\n%s", mandatorySpec, res.CorePrompt)
	}
}

func TestCompileWithAspect_FallbackPrompts(t *testing.T) {
	prompts := GenerateFallbackPrompts("Wedding of Priya & Arjun", "Paper Cut Art")
	builder := NewBasicBuilder()

	for i, p := range prompts {
		res := builder.CompileWithAspect(p, "", "9:16")
		if strings.Contains(res.CorePrompt, "Full view complete invitation card illustration for") {
			t.Errorf("Fallback prompt %d produced core prompt with unscrubbed meta prefix:\n%s", i, res.CorePrompt)
		}
	}
}

func TestCompileWithAspect_VisualPromptWithEventDetails(t *testing.T) {
	builder := NewBasicBuilder()

	// Test case: Visual theme prompt suggestion + MANDATORY EVENT DETAILS marker
	inputPrompt := `A vintage regal wedding invitation background featuring a soft dusty rose parchment texture. An elegant double-layered Mughal arch of delicate gold foil. MANDATORY EVENT DETAILS: Wedding of Rohan & Ananya on Dec 12 at Leela Palace`

	res := builder.CompileWithAspect(inputPrompt, "", "4:5")

	expectedTitle := "Wedding of Rohan & Ananya on Dec 12 at Leela Palace"
	if res.DisplayTitle != expectedTitle {
		t.Errorf("Expected DisplayTitle to be %q, got %q", expectedTitle, res.DisplayTitle)
	}

	mandatorySpec := "MANDATORY TEXT TO RENDER ON CARD PLATE: 'Wedding of Rohan & Ananya on Dec 12 at Leela Palace'"
	if !strings.Contains(res.CorePrompt, mandatorySpec) {
		t.Errorf("CorePrompt missing mandatory text spec %q:\n%s", mandatorySpec, res.CorePrompt)
	}

	if strings.Contains(res.CorePrompt, "A vintage regal wedding invitation background featuring a soft dusty r") {
		t.Errorf("CorePrompt contains truncated visual description string as title:\n%s", res.CorePrompt)
	}
}

func TestCompileStructured_3LineHierarchy(t *testing.T) {
	builder := NewBasicBuilder()

	data := EventData{
		EventType:      "Wedding",
		HostNames:      "Rohan & Ananya",
		EventDate:      "December 12, 2026",
		Venue:          "Leela Palace, Bengaluru",
		WelcomeMessage: "Together with our families, we invite you",
		VisualPrompt:   "A vintage regal wedding invitation background featuring a soft dusty rose parchment texture.",
		Aspect:         "4:5",
	}

	res := builder.CompileStructured(data)

	expectedTitle := "Wedding of Rohan & Ananya"
	if res.DisplayTitle != expectedTitle {
		t.Errorf("Expected DisplayTitle %q, got %q", expectedTitle, res.DisplayTitle)
	}

	expected3LineMandate := "MANDATORY TYPOGRAPHY TO RENDER IN CENTRAL PLATE (3-LINE HIERARCHY): Line 1 (Main Title): 'Wedding of Rohan & Ananya'. Line 2 (Secondary Welcome Subheader): 'Together with our families, we invite you'. Line 3 (Date & Location): 'December 12, 2026 | Leela Palace, Bengaluru'."
	if !strings.Contains(res.CorePrompt, expected3LineMandate) {
		t.Errorf("CorePrompt missing expected 3-line hierarchy mandate:\n%s", res.CorePrompt)
	}
}

func TestCompileStructured_BirthdayOccasion(t *testing.T) {
	builder := NewBasicBuilder()

	data := EventData{
		EventType: "Birthday",
		HostNames: "Priya",
		EventDate: "Sep 12, 2026",
		Venue:     "Marhaba Hall, Chennai",
		Aspect:    "4:5",
	}

	res := builder.CompileStructured(data)

	expectedTitle := "Birthday Celebration of Priya"
	if res.DisplayTitle != expectedTitle {
		t.Errorf("Expected DisplayTitle %q, got %q", expectedTitle, res.DisplayTitle)
	}

	expected3LineMandate := "MANDATORY TYPOGRAPHY TO RENDER IN CENTRAL PLATE (3-LINE HIERARCHY): Line 1 (Main Title): 'Birthday Celebration of Priya'. Line 2 (Secondary Welcome Subheader): 'Let us celebrate a wonderful milestone'. Line 3 (Date & Location): 'Sep 12, 2026 | Marhaba Hall, Chennai'."
	if !strings.Contains(res.CorePrompt, expected3LineMandate) {
		t.Errorf("CorePrompt missing expected Birthday 3-line hierarchy mandate:\n%s", res.CorePrompt)
	}
}
