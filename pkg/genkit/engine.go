package genkitengine

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// DefaultModelName is the central unversioned alias for Gemini Flash Text model.
const DefaultModelName = "googleai/gemini-flash-latest"

// DefaultImageModelName is the exact model ID for Gemini Flash Image generation.
const DefaultImageModelName = "googleai/gemini-3.1-flash-image"

// Engine holds the initialized Genkit instance and capability flags.
type Engine struct {
	Genkit    *genkit.Genkit
	HasAPIKey bool
	PromptFS  embed.FS
}

// InitEngine initializes the Genkit Go SDK with Google AI plugin and experimental agents enabled.
func InitEngine(ctx context.Context, promptFS embed.FS) *Engine {
	if ctx == nil {
		ctx = context.Background()
	}

	// Auto-load .env file if environment variables are not already present
	loadDotEnv()

	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GOOGLE_GENAI_API_KEY"))
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
	}

	hasKey := apiKey != "" && apiKey != "dummy"

	pluginKey := apiKey
	if pluginKey == "" {
		pluginKey = "byok_placeholder_key"
	}

	g := genkit.Init(ctx,
		genkit.WithExperimental(),
		genkit.WithPlugins(&googlegenai.GoogleAI{APIKey: pluginKey}),
	)
	log.Printf("[Genkit Engine] Initialized with Google GenAI plugin (APIKey active: %t)", hasKey)

	loadPrompts(g, promptFS)

	return &Engine{
		Genkit:    g,
		HasAPIKey: hasKey,
		PromptFS:  promptFS,
	}
}

// loadPrompts parses and registers embedded or local .prompt template files into Genkit.
func loadPrompts(g *genkit.Genkit, promptFS embed.FS) {
	loadedCount := 0
	entries, err := promptFS.ReadDir("prompts")
	if err == nil && len(entries) > 0 {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".prompt") {
				name := strings.TrimSuffix(entry.Name(), ".prompt")
				if genkit.LookupPrompt(g, name) == nil {
					data, readErr := promptFS.ReadFile("prompts/" + entry.Name())
					if readErr == nil {
						genkit.DefinePrompt(g, name, ai.WithPrompt(string(data)))
						loadedCount++
						log.Printf("[Genkit Engine] Loaded embedded Dotprompt template: %s", name)
					}
				} else {
					loadedCount++
					log.Printf("[Genkit Engine] Dotprompt template already registered: %s", name)
				}
			}
		}
	}

	// Fallback to local disk ./prompts if embed yielded 0 files
	if loadedCount == 0 {
		diskEntries, diskErr := os.ReadDir("./prompts")
		if diskErr == nil {
			for _, entry := range diskEntries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".prompt") {
					name := strings.TrimSuffix(entry.Name(), ".prompt")
					if genkit.LookupPrompt(g, name) == nil {
						data, readErr := os.ReadFile(filepath.Join("./prompts", entry.Name()))
						if readErr == nil {
							genkit.DefinePrompt(g, name, ai.WithPrompt(string(data)))
							loadedCount++
							log.Printf("[Genkit Engine] Loaded local disk Dotprompt template: %s", name)
						}
					} else {
						loadedCount++
						log.Printf("[Genkit Engine] Dotprompt template already registered: %s", name)
					}
				}
			}
		}
	}
	log.Printf("[Genkit Engine] Successfully registered %d Dotprompt templates", loadedCount)
}

// loadDotEnv searches for .env in current root repository directory and loads KEY=VAL into os.Getenv.
func loadDotEnv() {
	paths := []string{".env", "../.env", "../../.env"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			absPath, _ := filepath.Abs(p)
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					k := strings.TrimSpace(parts[0])
					v := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
					if v != "" {
						os.Setenv(k, v)
					}
				}
			}
			log.Printf("[Genkit Engine] Successfully loaded environment configuration from %s", absPath)
			break
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
