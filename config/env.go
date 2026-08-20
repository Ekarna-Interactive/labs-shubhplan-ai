package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Config holds local runtime preferences for Shubh CLI
type Config struct {
	GeminiAPIKey     string
	HonchoAPIKey     string
	GeminiImageModel string
	GeminiTextModel  string
	ImageModel       string
	OutputDir        string
	Port             string
}

// LoadConfig loads environment configuration from .env files or system environment variables
func LoadConfig() Config {
	loadDotEnv(".env")

	key := getEnvClean("GEMINI_API_KEY")
	honchoKey := getEnvClean("HONCHO_API_KEY")
	imageModel := getEnvClean("GEMINI_IMAGE_MODEL")
	if imageModel == "" {
		imageModel = getEnvClean("GEMINI_FALLBACK_IMAGE_MODEL")
	}
	textModel := getEnvClean("GEMINI_TEXT_MODEL")
	if textModel == "" {
		textModel = getEnvClean("GEMINI_FALLBACK_TEXT_MODEL")
	}
	outDir := getEnvClean("SHUBH_OUTPUT_DIR")
	if outDir == "" {
		outDir = "./output"
	}
	port := getEnvClean("PORT")
	if port == "" {
		port = "3000"
	}

	// Ensure output directory exists
	_ = os.MkdirAll(outDir, 0755)

	return Config{
		GeminiAPIKey:     key,
		HonchoAPIKey:     honchoKey,
		GeminiImageModel: imageModel,
		GeminiTextModel:  textModel,
		ImageModel:       imageModel,
		OutputDir:        outDir,
		Port:             port,
	}
}

// HasGeminiAPIKey returns true if a Gemini API Key is configured
func HasGeminiAPIKey() bool {
	cfg := LoadConfig()
	return cfg.GeminiAPIKey != "" || getEnvClean("GOOGLE_API_KEY") != ""
}

// HasHonchoAPIKey returns true if a Honcho Cloud API Key is configured
func HasHonchoAPIKey() bool {
	cfg := LoadConfig()
	return cfg.HonchoAPIKey != ""
}

func loadDotEnv(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
	_ = scanner.Err()
}

func getEnvClean(key string) string {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	val = strings.Trim(val, `"'`)
	return val
}

// GetAbsOutputDir returns the absolute path of the output directory
func (c Config) GetAbsOutputDir() string {
	abs, err := filepath.Abs(c.OutputDir)
	if err != nil {
		return c.OutputDir
	}
	return abs
}

// SaveGeminiAPIKey persists the Gemini API key to local .env file and environment
func SaveGeminiAPIKey(key string) error {
	cleanKey := strings.TrimSpace(key)
	cleanKey = strings.Trim(cleanKey, `"'`)
	_ = os.Setenv("GEMINI_API_KEY", cleanKey)

	envPath := ".env"
	lines := []string{}

	if file, err := os.Open(envPath); err == nil {
		scanner := bufio.NewScanner(file)
		found := false
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(strings.TrimSpace(line), "GEMINI_API_KEY=") {
				lines = append(lines, `GEMINI_API_KEY="`+cleanKey+`"`)
				found = true
			} else {
				lines = append(lines, line)
			}
		}
		_ = scanner.Err()
		_ = file.Close()
		if !found {
			lines = append(lines, `GEMINI_API_KEY="`+cleanKey+`"`)
		}
	} else {
		lines = append(lines, `GEMINI_API_KEY="`+cleanKey+`"`)
	}

	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// SaveHonchoAPIKey persists the Honcho API key to local .env file and environment
func SaveHonchoAPIKey(key string) error {
	cleanKey := strings.TrimSpace(key)
	cleanKey = strings.Trim(cleanKey, `"'`)
	_ = os.Setenv("HONCHO_API_KEY", cleanKey)

	envPath := ".env"
	lines := []string{}

	if file, err := os.Open(envPath); err == nil {
		scanner := bufio.NewScanner(file)
		found := false
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(strings.TrimSpace(line), "HONCHO_API_KEY=") {
				lines = append(lines, `HONCHO_API_KEY="`+cleanKey+`"`)
				found = true
			} else {
				lines = append(lines, line)
			}
		}
		_ = scanner.Err()
		_ = file.Close()
		if !found {
			lines = append(lines, `HONCHO_API_KEY="`+cleanKey+`"`)
		}
	} else {
		lines = append(lines, `HONCHO_API_KEY="`+cleanKey+`"`)
	}

	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// GetPlacesAPIKey returns the Google Places API Key if configured
func GetPlacesAPIKey() string {
	key := getEnvClean("GOOGLE_PLACES_API_KEY")
	if key == "" {
		key = getEnvClean("GOOGLE_MAPS_API_KEY")
	}
	if key == "" {
		key = getEnvClean("PLACES_API_KEY")
	}
	return key
}

// SavePlacesAPIKey persists the Places API key to local .env file and environment
func SavePlacesAPIKey(key string) error {
	cleanKey := strings.TrimSpace(key)
	cleanKey = strings.Trim(cleanKey, `"'`)
	_ = os.Setenv("GOOGLE_PLACES_API_KEY", cleanKey)

	envPath := ".env"
	lines := []string{}

	if file, err := os.Open(envPath); err == nil {
		scanner := bufio.NewScanner(file)
		found := false
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(strings.TrimSpace(line), "GOOGLE_PLACES_API_KEY=") {
				lines = append(lines, `GOOGLE_PLACES_API_KEY="`+cleanKey+`"`)
				found = true
			} else {
				lines = append(lines, line)
			}
		}
		_ = scanner.Err()
		_ = file.Close()
		if !found {
			lines = append(lines, `GOOGLE_PLACES_API_KEY="`+cleanKey+`"`)
		}
	} else {
		lines = append(lines, `GOOGLE_PLACES_API_KEY="`+cleanKey+`"`)
	}

	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
