package web

import (
	"encoding/json"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/client"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/config"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/generator"
)

//go:embed templates/*
var templateFS embed.FS

type HTTPServer struct {
	port    int
	builder *generator.BasicBuilder
	engine  *client.AgentEngine
	honcho  *client.HonchoMemoryStore
}

func NewHTTPServer(port int) *HTTPServer {
	return &HTTPServer{
		port:    port,
		builder: generator.NewBasicBuilder(),
		engine:  client.GetAgentEngine(),
		honcho:  client.GetHonchoManager(),
	}
}

func (s *HTTPServer) Start() error {
	startMu.Lock()
	serverPort = fmt.Sprintf("%d", s.port)
	isStarted = true
	startMu.Unlock()

	mux := http.NewServeMux()

	// Root index page handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		tmpl, err := template.ParseFS(templateFS, "templates/index.html")
		if err != nil {
			http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, nil); err != nil {
			log.Printf("[Web UI] Execute error: %v", err)
		}
	})

	// Dynamic Ephemeral Session Secret Handshake Endpoint
	mux.HandleFunc("/api/v1/session/handshake", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		clientID := r.Header.Get("X-Client-ID")
		if clientID == "" {
			clientID = "web-client"
		}

		sess := s.engine.CreateHandshakeSession(clientID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sess)
	})

	// Inbuilt SSE Multi-Agent Stream Endpoint
	mux.HandleFunc("/api/v1/orchestrator/stream", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		authHeader := r.Header.Get("Authorization")
		secret := strings.TrimPrefix(authHeader, "Bearer ")
		if !s.engine.ValidateSessionSecret(secret) {
			http.Error(w, "401 Unauthorized: Invalid sessionSecret", http.StatusUnauthorized)
			return
		}

		var reqData struct {
			Message     string `json:"message"`
			PlannerName string `json:"plannerName"`
			Context     string `json:"eventContext"`
		}
		_ = json.NewDecoder(r.Body).Decode(&reqData)

		if reqData.PlannerName == "" {
			reqData.PlannerName = "Lead Planner"
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		s.engine.StreamMultiAgentResponse(r.Context(), reqData.Message, reqData.PlannerName, reqData.Context, func(ev client.AgentStreamEvent) {
			evJSON, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", string(evJSON))
			flusher.Flush()
		})
	})

	// Guest Roster HTMX Fragment Handler
	mux.HandleFunc("/api/guests", func(w http.ResponseWriter, r *http.Request) {
		cards := s.honcho.GetPeerCards()
		statusMsg := s.honcho.GetHonchoStatusMessage()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div style="background: rgba(15, 23, 42, 0.7); border: 1px solid #334155; padding: 0.75rem 1rem; border-radius: 10px; font-size: 0.8rem; color: #cbd5e1; margin-bottom: 1rem;">%s</div>`, statusMsg)

		if len(cards) == 0 {
			fmt.Fprintf(w, `<div style="padding: 1.5rem; text-align: center; color: #94a3b8;">No guest RSVPs recorded yet. Use the TUI or Copilot chat to record guest RSVPs.</div>`)
			return
		}

		for name, card := range cards {
			cabText := "Self Transport"
			if card.Cab {
				cabText = "Cab Needed"
			}
			fmt.Fprintf(w, `
			<div style="background: rgba(15, 23, 42, 0.6); border: 1px solid #334155; padding: 1rem; border-radius: 12px; margin-bottom: 0.75rem; display: flex; justify-content: space-between; align-items: center;">
				<div>
					<strong style="color: #f1f5f9; font-size: 1rem;">%s</strong>
					<div style="font-size: 0.8rem; color: #94a3b8;">Headcount: %d • Diet: %s • %s</div>
				</div>
				<span style="background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3); padding: 0.25rem 0.75rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 600;">%s</span>
			</div>
			`, name, card.Headcount, card.Dietary, cabText, card.Status)
		}
	})

	// HTMX Generate Prompt & Design Handler
	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		eventType := r.FormValue("eventType")
		hosts := r.FormValue("hosts")
		welcome := r.FormValue("welcome")
		style := r.FormValue("style")

		prompts := s.builder.CompilePrompts(eventType, hosts, welcome, style)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		for _, p := range prompts {
			fmt.Fprintf(w, `
			<div class="bg-slate-900/80 border border-slate-800 rounded-xl p-4 mb-3 theme-card shadow-sm">
				<div class="flex justify-between items-center text-xs mb-2">
					<strong class="text-amber-400 font-bold text-sm">✨ %s</strong>
					<span class="text-slate-400 font-mono text-[11px]">%s</span>
				</div>
				<p class="text-xs text-slate-200 font-mono bg-slate-950 p-3 rounded-lg border border-slate-800 line-clamp-6 leading-relaxed theme-panel">
					%s
				</p>
			</div>
			`, p.Title, p.Style, p.Prompt)
		}
	})

	// API Keys Status Query Handler
	mux.HandleFunc("/api/keys/status", func(w http.ResponseWriter, r *http.Request) {
		clientGemini := strings.TrimSpace(r.Header.Get("X-Gemini-API-Key"))
		clientHoncho := strings.TrimSpace(r.Header.Get("X-Honcho-API-Key"))

		geminiKey := config.LoadConfig().GeminiAPIKey
		if clientGemini != "" {
			geminiKey = clientGemini
		}
		honchoKey := strings.TrimSpace(os.Getenv("HONCHO_API_KEY"))
		if clientHoncho != "" {
			honchoKey = clientHoncho
		}

		status := map[string]interface{}{
			"geminiSet":    geminiKey != "",
			"geminiStatus": "🔴 Missing (Offline Dry-Run)",
			"honchoSet":    honchoKey != "",
			"honchoStatus": "🟡 Inbuilt Local Store (./data/honcho_memory.json)",
		}
		if geminiKey != "" {
			status["geminiStatus"] = "🟢 Active (Live AI Generation)"
		}
		if honchoKey != "" {
			status["honchoStatus"] = "🟢 Honcho Cloud Sync (api.honcho.dev/v3)"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	})

	// API Keys Save Handler
	mux.HandleFunc("/api/keys", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			geminiKey := strings.TrimSpace(r.FormValue("geminiKey"))
			honchoKey := strings.TrimSpace(r.FormValue("honchoKey"))

			// If local single-user workstation mode, save to .env
			if os.Getenv("SERVER_MODE") != "true" && os.Getenv("MULTI_USER") != "true" {
				if geminiKey != "" {
					_ = config.SaveGeminiAPIKey(geminiKey)
				}
				if honchoKey != "" {
					_ = config.SaveHonchoAPIKey(honchoKey)
				}
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<div style="background: rgba(16, 185, 129, 0.15); border: 1px solid rgba(16, 185, 129, 0.4); color: #34d399; padding: 0.75rem 1rem; border-radius: 10px; font-size: 0.8rem; margin-bottom: 1rem;">✅ API Keys activated for your session! Saved to browser localStorage.</div><script>saveClientKeys('%s', '%s');</script>`, geminiKey, honchoKey)
			return
		}
	})

	// Event Profile API Handler
	mux.HandleFunc("/api/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			profile, hasProfile := config.LoadEventProfile()
			isoDate, displayDate, _ := config.ParseAndNormalizeMachineDate(profile.EventDate)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"hasProfile":     hasProfile,
				"eventType":      profile.EventType,
				"hostNames":      profile.HostNames,
				"eventDate":      displayDate,
				"isoDate":        isoDate,
				"venue":          profile.Venue,
				"welcomeMessage": profile.WelcomeMessage,
				"rawDetails":     profile.RawDetails,
			})
			return
		}

		if r.Method == http.MethodPost {
			eType := r.FormValue("eventType")
			hosts := r.FormValue("hosts")
			eDate := r.FormValue("eventDate")
			venue := r.FormValue("venue")
			welcome := r.FormValue("welcome")
			style := r.FormValue("style")

			if eDate == "" {
				eDate = "November 24, 2026"
			}
			if venue == "" {
				venue = "Palace Grounds, Bengaluru"
			}

			err := config.SaveStructuredEventProfileWithBudget(eType, hosts, eDate, venue, "USD", welcome, "9:16", "Gokul", "Lead Event Planner", 0, 0)
			_ = style

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err != nil {
				fmt.Fprintf(w, `<div style="background: rgba(239, 68, 68, 0.15); border: 1px solid rgba(239, 68, 68, 0.4); color: #fca5a5; padding: 0.75rem 1rem; border-radius: 10px; font-size: 0.8rem; margin-bottom: 1rem;">⚠️ Failed to save event profile: %v</div>`, err)
			} else {
				fmt.Fprintf(w, `<div style="background: rgba(16, 185, 129, 0.15); border: 1px solid rgba(16, 185, 129, 0.4); color: #34d399; padding: 0.75rem 1rem; border-radius: 10px; font-size: 0.8rem; margin-bottom: 1rem;">✅ Event profile saved to event_details.md!</div><script>fetchEventProfile();</script>`)
			}
			return
		}
	})

	// AI Welcome Suggestions Handler
	mux.HandleFunc("/api/suggest-welcome", func(w http.ResponseWriter, r *http.Request) {
		eventType := r.FormValue("eventType")
		clientGemini := strings.TrimSpace(r.Header.Get("X-Gemini-API-Key"))
		geminiKey := config.LoadConfig().GeminiAPIKey
		if clientGemini != "" {
			geminiKey = clientGemini
		}

		suggestions, _ := generator.GenerateAIWelcomeSuggestions(geminiKey, eventType)
		if len(suggestions) == 0 {
			suggestions = generator.GenerateFallbackWelcomeSubheaders(eventType)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div class="mt-2 space-y-1.5 p-2 bg-slate-950/80 border border-slate-800 rounded-xl theme-panel">`)
		fmt.Fprintf(w, `<div class="text-[11px] font-bold text-amber-400 mb-1 flex justify-between items-center"><span>🪄 AI Welcome Subheader Suggestions (Click to insert):</span></div>`)
		for idx, sug := range suggestions {
			escapedSug := strings.ReplaceAll(sug, "'", "\\'")
			escapedSug = strings.ReplaceAll(escapedSug, `"`, "&quot;")
			fmt.Fprintf(w, `<button type="button" onclick="selectWelcomeSuggestion('%s')" class="w-full text-left p-2 rounded-lg bg-slate-900 border border-slate-800 hover:border-amber-500/50 hover:bg-amber-500/10 text-xs text-slate-200 transition-all theme-card">✨ Option %d: %s</button>`, escapedSug, idx+1, sug)
		}
		fmt.Fprintf(w, `</div>`)
	})

	// List Generated Output Cards API
	mux.HandleFunc("/api/outputs", func(w http.ResponseWriter, r *http.Request) {
		outputDir := "./output"
		entries, err := os.ReadDir(outputDir)
		w.Header().Set("Content-Type", "application/json")

		sessionID := r.URL.Query().Get("sessionID")

		type CardItem struct {
			Name    string `json:"name"`
			Path    string `json:"path"`
			ModTime int64  `json:"modTime"`
		}

		var cards []CardItem
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".png") || strings.HasSuffix(entry.Name(), ".jpg")) {
					if sessionID != "" && !strings.Contains(entry.Name(), sessionID) {
						continue
					}
					info, err := entry.Info()
					modTime := int64(0)
					if err == nil {
						modTime = info.ModTime().Unix()
					}
					cards = append(cards, CardItem{
						Name:    entry.Name(),
						Path:    "/output/" + entry.Name(),
						ModTime: modTime,
					})
				}
			}
		}

		// Sort newest first
		for i := 0; i < len(cards); i++ {
			for j := i + 1; j < len(cards); j++ {
				if cards[i].ModTime < cards[j].ModTime {
					cards[i], cards[j] = cards[j], cards[i]
				}
			}
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"count": len(cards),
			"cards": cards,
		})
	})

	// Static asset server for output cards
	fileServer := http.FileServer(http.Dir("./output"))
	mux.Handle("/output/", http.StripPrefix("/output/", fileServer))

	serverAddr := fmt.Sprintf(":%d", s.port)
	log.Printf("🌐 [Web UI] Server running on http://localhost%s", serverAddr)
	return http.ListenAndServe(serverAddr, mux)
}
