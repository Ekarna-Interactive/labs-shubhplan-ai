package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
)

type contextKey string

const (
	HTMXContextKey contextKey = "htmxContext"
	AuthUserKey    contextKey = "authUser"
	APIKeysKey     contextKey = "apiKeys"
)

// HTMXContext holds metadata about incoming HTMX requests
type HTMXContext struct {
	IsHTMX     bool
	Target     string
	Trigger    string
	CurrentURL string
}

// UserContext holds authenticated user details
type UserContext struct {
	Email string
	Role  string
}

// ResolvedKeys holds Gemini & Honcho API keys for the current request
type ResolvedKeys struct {
	GeminiKey string
	HonchoKey string
}

// HTMXMiddleware inspects incoming headers to detect HTMX requests & targets
func HTMXMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hCtx := HTMXContext{
			IsHTMX:     r.Header.Get("HX-Request") == "true",
			Target:     r.Header.Get("HX-Target"),
			Trigger:    r.Header.Get("HX-Trigger"),
			CurrentURL: r.Header.Get("HX-Current-Url"),
		}
		ctx := context.WithValue(r.Context(), HTMXContextKey, hCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// APIKeyContextMiddleware extracts client-side headers or environment fallbacks
func APIKeyContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := config.LoadConfig()

		clientGemini := strings.TrimSpace(r.Header.Get("X-Gemini-API-Key"))
		geminiKey := cfg.GeminiAPIKey
		if clientGemini != "" {
			geminiKey = clientGemini
		}
		if strings.Contains(geminiKey, "your_gemini_api_key_here") || strings.HasPrefix(geminiKey, "your_") {
			geminiKey = ""
		}

		clientHoncho := strings.TrimSpace(r.Header.Get("X-Honcho-API-Key"))
		honchoKey := cfg.HonchoAPIKey
		if clientHoncho != "" {
			honchoKey = clientHoncho
		}

		keys := ResolvedKeys{
			GeminiKey: geminiKey,
			HonchoKey: honchoKey,
		}
		ctx := context.WithValue(r.Context(), APIKeysKey, keys)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuthHTMX ensures user is authenticated, redirecting via HTMX header if needed
func RequireAuthHTMX(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check cookie or auth state
		cookie, err := r.Cookie("shubh_session")
		if err != nil || cookie.Value == "" {
			hCtx, ok := r.Context().Value(HTMXContextKey).(HTMXContext)
			if ok && hCtx.IsHTMX {
				// HTMX-friendly redirect header (avoids loading full login inside inner div)
				w.Header().Set("HX-Redirect", "/")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Helper to check if request is an HTMX partial request
func IsHTMXRequest(r *http.Request) bool {
	hCtx, ok := r.Context().Value(HTMXContextKey).(HTMXContext)
	return ok && hCtx.IsHTMX
}

// Helper to retrieve resolved API keys from context
func GetResolvedKeys(r *http.Request) ResolvedKeys {
	keys, ok := r.Context().Value(APIKeysKey).(ResolvedKeys)
	if !ok {
		cfg := config.LoadConfig()
		return ResolvedKeys{GeminiKey: cfg.GeminiAPIKey, HonchoKey: cfg.HonchoAPIKey}
	}
	return keys
}
