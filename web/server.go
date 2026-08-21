package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Ekarna-Interactive/labs-shubhplan-ai/client"
	"github.com/Ekarna-Interactive/labs-shubhplan-ai/command"
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
	cfg := config.LoadConfig()
	hm := client.GetHonchoManager()
	if cfg.HonchoAPIKey != "" {
		hm.SetAPIKey(cfg.HonchoAPIKey)
	}

	profile, hasProfile := config.LoadEventProfile()
	if hasProfile && profile.ID != "" && profile.EventType != "" && profile.HostNames != "" {
		hm.EnsureWorkspaceCreated(profile.ID, fmt.Sprintf("%s for %s", profile.EventType, profile.HostNames))
	}

	return &HTTPServer{
		port:    port,
		builder: generator.NewBasicBuilder(),
		engine:  client.GetAgentEngine(),
		honcho:  hm,
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
		tmpl, err := template.ParseFS(templateFS, "templates/index.html", "templates/components/*.html")
		if err != nil {
			http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, nil); err != nil {
			log.Printf("[Web UI] Execute error: %v", err)
		}
	})

	// Auth Status Check Endpoint
	mux.HandleFunc("/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		setupCompleted := config.IsSetupCompleted()

		cookie, err := r.Cookie("shubh_session")
		var activeUser map[string]interface{}
		if err == nil && cookie.Value != "" {
			if sess, ok := config.ValidateSessionToken(cookie.Value); ok {
				activeUser = map[string]interface{}{
					"userId": sess.UserID,
					"email":  sess.UserEmail,
					"role":   sess.UserRole,
				}
			}
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"setupCompleted": setupCompleted,
			"hasUsers":       setupCompleted,
			"authenticated":  activeUser != nil,
			"user":           activeUser,
		})
	})

	// Owner Setup Endpoint (First-Time Registration)
	mux.HandleFunc("/api/auth/setup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if config.IsSetupCompleted() {
			http.Error(w, "Setup already completed. Please log in.", http.StatusBadRequest)
			return
		}
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			FullName string `json:"fullName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		user, err := config.CreateUser(req.Email, req.Password, req.FullName, config.RoleAdmin)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sess, _ := config.AuthenticateUser(req.Email, req.Password)
		http.SetCookie(w, &http.Cookie{
			Name:     "shubh_session",
			Value:    sess.Token,
			Path:     "/",
			Expires:  sess.ExpiresAt,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"user":    user,
			"token":   sess.Token,
		})
	})

	// Login Endpoint
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		sess, err := config.AuthenticateUser(req.Email, req.Password)
		if err != nil {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "shubh_session",
			Value:    sess.Token,
			Path:     "/",
			Expires:  sess.ExpiresAt,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"user": map[string]interface{}{
				"email": sess.UserEmail,
				"role":  sess.UserRole,
			},
			"token": sess.Token,
		})
	})

	// Logout Endpoint
	mux.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("shubh_session")
		if err == nil && cookie.Value != "" {
			config.InvalidateSession(cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "shubh_session",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
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

		if reqData.PlannerName == "" || reqData.PlannerName == "Lead Planner" || reqData.PlannerName == "Event Planner" {
			profile, hasProfile := config.LoadEventProfile()
			if hasProfile && strings.TrimSpace(profile.PlannerName) != "" {
				reqData.PlannerName = profile.PlannerName
			} else if hasProfile && strings.TrimSpace(profile.PlannerRole) != "" {
				reqData.PlannerName = profile.PlannerRole
			} else {
				reqData.PlannerName = "Event Planner"
			}
		}

		if strings.TrimSpace(reqData.Context) == "" {
			profile, hasProfile := config.LoadEventProfile()
			if hasProfile && (profile.EventType != "" || profile.HostNames != "" || profile.Venue != "") {
				_, displayDate, _ := config.ParseAndNormalizeMachineDate(profile.EventDate)
				reqData.Context = fmt.Sprintf("%s for %s on %s at %s", profile.EventType, profile.HostNames, displayDate, profile.Venue)
				if profile.WelcomeMessage != "" {
					reqData.Context += fmt.Sprintf(". Welcome Subheader: '%s'", profile.WelcomeMessage)
				}
				if profile.VenueAddress != "" {
					reqData.Context += fmt.Sprintf(". Full Address: '%s'", profile.VenueAddress)
				}
			} else {
				reqData.Context = "No active event details recorded yet."
			}
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		trimmedMsg := strings.TrimSpace(reqData.Message)

		// Handle Slash Commands in Web UI
		if strings.HasPrefix(trimmedMsg, "/") {
			parsed := command.Parse(trimmedMsg)
			if handleWebSlashCommand(w, flusher, parsed, reqData.PlannerName) {
				return
			}
		}

		s.engine.StreamMultiAgentResponse(r.Context(), reqData.Message, reqData.PlannerName, reqData.Context, func(ev client.AgentStreamEvent) {
			evJSON, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", string(evJSON))
			flusher.Flush()
		})
	})

	// Guest Roster HTMX Fragment Handler
	mux.HandleFunc("/api/guests", func(w http.ResponseWriter, r *http.Request) {
		hKey := r.Header.Get("X-Honcho-API-Key")
		if hKey == "" {
			hKey = os.Getenv("HONCHO_API_KEY")
		}
		if hKey != "" {
			s.honcho.SetAPIKey(hKey)
		}

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

	// Honcho Memory Tab HTMX Fragment Handler
	mux.HandleFunc("/api/honcho/details", func(w http.ResponseWriter, r *http.Request) {
		hKey := r.Header.Get("X-Honcho-API-Key")
		if hKey == "" {
			hKey = os.Getenv("HONCHO_API_KEY")
		}
		if hKey != "" {
			s.honcho.SetAPIKey(hKey)
		}

		profile, hasProfile := config.LoadEventProfile()
		wsID := s.honcho.AppID
		if hasProfile && profile.ID != "" {
			wsID = profile.ID
		}

		cards := s.honcho.GetPeerCards()
		statusMsg := s.honcho.GetHonchoStatusMessage()

		type CloudPeer struct {
			ID   string                 `json:"id"`
			Name string                 `json:"name"`
			Meta map[string]interface{} `json:"meta"`
		}
		var cloudPeers []CloudPeer

		if hKey != "" {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			cURL := fmt.Sprintf("https://api.honcho.dev/v3/workspaces/%s/peers", wsID)
			cReq, _ := http.NewRequestWithContext(ctx, "GET", cURL, nil)
			if cReq != nil {
				cReq.Header.Set("Authorization", "Bearer "+hKey)
				if cResp, err := http.DefaultClient.Do(cReq); err == nil {
					if cResp.StatusCode == http.StatusOK {
						var res struct {
							Peers []CloudPeer `json:"peers"`
						}
						body, _ := io.ReadAll(cResp.Body)
						if json.Unmarshal(body, &res) == nil {
							cloudPeers = res.Peers
						}
					}
					cResp.Body.Close()
				}
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div class="space-y-4">`)
		fmt.Fprintf(w, `<div class="bg-slate-950/80 border border-slate-800 p-4 rounded-xl flex flex-col md:flex-row items-start md:items-center justify-between gap-3">
			<div>
				<div class="text-xs font-bold text-amber-400">🧠 Honcho Cloud Memory Connection:</div>
				<div class="text-xs text-slate-300 font-mono mt-0.5">%s</div>
				<div class="text-[11px] text-slate-400 mt-1">Active Workspace: <strong class="text-amber-300 font-mono">%s</strong> • Active Session: <strong class="text-emerald-400 font-mono">session-chat</strong></div>
			</div>
			<a href="https://app.honcho.dev" target="_blank" class="px-3 py-1.5 bg-amber-500/20 border border-amber-500/40 hover:bg-amber-500/30 text-amber-300 font-semibold rounded-lg transition-all text-xs flex items-center space-x-1">
				<span>🔗 Open app.honcho.dev</span>
			</a>
		</div>`, statusMsg, wsID)

		if len(cloudPeers) > 0 {
			fmt.Fprintf(w, `<div class="space-y-2">`)
			fmt.Fprintf(w, `<h3 class="text-xs font-bold text-slate-300">☁️ Live Honcho Cloud Registered Peers (%d):</h3>`, len(cloudPeers))
			for _, cp := range cloudPeers {
				role := "Registered Participant"
				if rVal, ok := cp.Meta["role"].(string); ok && rVal != "" {
					role = rVal
				}
				fmt.Fprintf(w, `<div class="bg-slate-950/60 border border-slate-800 p-3 rounded-xl flex justify-between items-center text-xs">
					<div>
						<strong class="text-slate-100">%s</strong> <span class="text-slate-400 text-[11px]">(Peer ID: %s)</span>
						<div class="text-slate-400 text-[11px] mt-0.5">Role: %s</div>
					</div>
					<span class="px-2.5 py-1 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-[11px] font-semibold">Active Cloud Peer</span>
				</div>`, cp.Name, cp.ID, role)
			}
			fmt.Fprintf(w, `</div>`)
		} else if len(cards) > 0 {
			fmt.Fprintf(w, `<div class="space-y-2">`)
			fmt.Fprintf(w, `<h3 class="text-xs font-bold text-slate-300">👥 Recorded Guest Memory Cards (%d):</h3>`, len(cards))
			for name, card := range cards {
				fmt.Fprintf(w, `<div class="bg-slate-950/60 border border-slate-800 p-3 rounded-xl flex justify-between items-center text-xs">
					<div>
						<strong class="text-slate-100">%s</strong> <span class="text-slate-400 text-[11px]">(Peer ID: %s)</span>
						<div class="text-slate-400 text-[11px] mt-0.5">Headcount: %d • Diet: %s • %s</div>
					</div>
					<span class="px-2.5 py-1 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-[11px] font-semibold">%s</span>
				</div>`, name, strings.ToLower(strings.ReplaceAll(name, " ", "-")), card.Headcount, card.Dietary, card.Status, card.Status)
			}
			fmt.Fprintf(w, `</div>`)
		} else {
			fmt.Fprintf(w, `<div class="p-4 bg-slate-950/40 border border-slate-800 rounded-xl text-center text-xs text-slate-400">No guest peers recorded yet. Use the TUI or Copilot chat to record guest RSVPs.</div>`)
		}
		fmt.Fprintf(w, `</div>`)
	})

	// Dynamic Event Timeline HTMX Handler
	mux.HandleFunc("/api/timeline", func(w http.ResponseWriter, r *http.Request) {
		profile, hasProfile := config.LoadEventProfile()
		_, displayDate, _ := config.ParseAndNormalizeMachineDate(profile.EventDate)
		if displayDate == "" {
			displayDate = profile.EventDate
		}
		if displayDate == "" {
			displayDate = "Active Event Date"
		}

		eType := profile.EventType
		if eType == "" {
			eType = "Special Event"
		}
		venue := profile.Venue
		if venue == "" {
			venue = "Main Venue"
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if !hasProfile {
			fmt.Fprintf(w, `<div class="p-6 bg-slate-950/40 border border-slate-800 rounded-xl text-center space-y-2">
				<div class="text-amber-400 font-bold text-sm">📅 No Active Event Profile Loaded</div>
				<p class="text-xs text-slate-400">Save an event profile or use the AI Copilot to generate an itinerary schedule.</p>
			</div>`)
			return
		}

		fmt.Fprintf(w, `<div class="space-y-4">`)
		fmt.Fprintf(w, `<div class="bg-slate-950/80 border border-slate-800 p-4 rounded-xl flex justify-between items-center text-xs">
			<div>
				<strong class="text-amber-400 font-bold text-sm">✨ Active Itinerary: %s for %s</strong>
				<div class="text-slate-400 mt-0.5">Date: <span class="text-slate-200 font-mono">%s</span> • Venue: <span class="text-slate-200 font-mono">%s</span></div>
			</div>
			<span class="px-2.5 py-1 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-xs font-semibold">Active Run-of-Show</span>
		</div>`, eType, profile.HostNames, displayDate, venue)

		if len(profile.Itinerary) == 0 {
			fmt.Fprintf(w, `<div class="bg-slate-950/60 border border-slate-800 p-6 rounded-xl space-y-4">
				<div class="flex items-center space-x-2 text-amber-400 font-bold text-sm">
					<span>📅 TIMELINE & RUN-OF-SHOW BUILDER</span>
				</div>
				<p class="text-xs text-slate-300">No sub-events have been added to this event itinerary schedule yet.</p>

				<div class="bg-slate-900/80 border border-slate-800 p-4 rounded-lg space-y-2 text-xs">
					<strong class="text-amber-400 font-bold flex items-center space-x-1">
						<span>💡 How to Add Itinerary Sub-Events:</span>
					</strong>
					<ul class="space-y-2 text-slate-300 list-disc list-inside text-[11px] leading-relaxed">
						<li><strong>Ask the AI Orchestrator / Copilot</strong>: Type any natural language prompt in the <strong>AI Concierge Copilot</strong> chat (or TUI terminal), such as:
							<div class="mt-1 font-mono text-amber-300 bg-slate-950 p-2 rounded border border-slate-800">"Build a 3-day Haldi, Sangeet, and Reception timeline for Rohan & Ananya"</div>
							<div class="mt-1 font-mono text-amber-300 bg-slate-950 p-2 rounded border border-slate-800">"Add Namakarana ceremony on Oct 12 from 10:30 AM to 12:00 PM at Central Mandap"</div>
						</li>
						<li><strong>Query TimelineAgent</strong>: Ask for a minute-by-minute run-of-show or venue schedule.</li>
					</ul>
				</div>
			</div></div>`)
			return
		}

		fmt.Fprintf(w, `<div class="grid grid-cols-1 md:grid-cols-2 gap-4">`)
		for _, slot := range profile.Itinerary {
			locStr := venue
			if slot.Location != "" {
				locStr = slot.Location
			}
			dressStr := "Festive / Traditional"
			if slot.DressCode != "" {
				dressStr = slot.DressCode
			}
			fmt.Fprintf(w, `
			<div class="bg-slate-950/60 border border-slate-800 p-4 rounded-xl space-y-2">
				<div class="flex justify-between items-center text-xs">
					<strong class="text-amber-400 font-bold">%s</strong>
					<span class="text-slate-400 font-mono text-[11px]">%s</span>
				</div>
				<div class="text-xs text-slate-300 font-medium">%s • %s</div>
				<div class="text-xs text-slate-400">Dress Code: %s</div>
			</div>
			`, slot.Title, displayDate, slot.Time, locStr, dressStr)
		}
		fmt.Fprintf(w, `</div></div>`)
		fmt.Fprintf(w, `</div></div>`)
	})

	// HTMX Generate Prompt & Design Handler
	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		eventType := r.FormValue("eventType")
		style := r.FormValue("style")
		if eventType == "" {
			eventType = "Wedding Celebration"
		}
		if style == "" {
			style = "South Indian Royal Gold"
		}

		clientGemini := strings.TrimSpace(r.Header.Get("X-Gemini-API-Key"))
		geminiKey := config.LoadConfig().GeminiAPIKey
		if clientGemini != "" {
			geminiKey = clientGemini
		}

		ideas, _ := generator.GenerateAIPromptIdeas(geminiKey, eventType, style)
		if len(ideas) == 0 {
			ideas = generator.GenerateFallbackPromptIdeas(eventType, style)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div class="space-y-3.5">`)
		fmt.Fprintf(w, `<div class="p-3 bg-amber-500/10 border border-amber-500/30 rounded-xl text-xs text-amber-300 flex items-center justify-between">`)
		fmt.Fprintf(w, `<span class="font-semibold">✨ AI Generated 4 Creative Design Themes for '%s' (%s):</span>`, eventType, style)
		fmt.Fprintf(w, `<span class="text-[10px] opacity-80 font-mono">Select a theme to render artwork</span></div>`)

		for idx, idea := range ideas {
			escapedPrompt := strings.ReplaceAll(idea.PromptText, "'", "\\'")
			escapedPrompt = strings.ReplaceAll(escapedPrompt, `"`, "&quot;")
			escapedPrompt = strings.ReplaceAll(escapedPrompt, "\n", " ")

			fmt.Fprintf(w, `<div class="bg-slate-900 border border-slate-800 hover:border-amber-500/40 rounded-xl p-4 transition-all space-y-2.5 shadow-md">`)
			fmt.Fprintf(w, `<div class="flex items-center justify-between">`)
			fmt.Fprintf(w, `<div class="flex items-center space-x-2">`)
			fmt.Fprintf(w, `<span class="px-2.5 py-0.5 bg-amber-500/20 text-amber-300 border border-amber-500/30 rounded-full text-[10px] font-bold tracking-wide uppercase">Option %d</span>`, idx+1)
			fmt.Fprintf(w, `<span class="text-xs font-bold text-slate-100">✨ %s</span></div>`, idea.ThemeTitle)
			fmt.Fprintf(w, `<span class="text-[10px] text-slate-400 font-mono">%s</span></div>`, style)
			fmt.Fprintf(w, `<p class="text-xs text-slate-200 leading-relaxed font-mono bg-slate-950/60 p-2.5 rounded-lg border border-slate-800/80 select-all">%s</p>`, idea.PromptText)
			fmt.Fprintf(w, `<div class="flex justify-end pt-1">`)
			fmt.Fprintf(w, `<button type="button" onclick="renderSelectedPrompt('%s', this)" class="px-4 py-2 bg-gradient-to-r from-amber-500 to-amber-600 hover:from-amber-600 hover:to-amber-700 text-slate-950 font-bold rounded-lg text-xs flex items-center space-x-1.5 shadow-lg transition-all hover:scale-[1.02]">`, escapedPrompt)
			fmt.Fprintf(w, `<span>🎨 Generate Invitation Image</span><span>➔</span></button></div></div>`)
		}
		fmt.Fprintf(w, `</div>`)
	})

	// API Keys Status Query Handler
	mux.HandleFunc("/api/keys/status", func(w http.ResponseWriter, r *http.Request) {
		clientGemini := strings.TrimSpace(r.Header.Get("X-Gemini-API-Key"))
		clientHoncho := strings.TrimSpace(r.Header.Get("X-Honcho-API-Key"))

		geminiKey := config.LoadConfig().GeminiAPIKey
		if strings.Contains(geminiKey, "your_gemini_api_key_here") || strings.HasPrefix(geminiKey, "your_") {
			geminiKey = ""
		}
		if clientGemini != "" {
			geminiKey = clientGemini
		}
		honchoKey := strings.TrimSpace(os.Getenv("HONCHO_API_KEY"))
		if strings.Contains(honchoKey, "your_honcho_api_key_here") || strings.HasPrefix(honchoKey, "your_") {
			honchoKey = ""
		}
		if clientHoncho != "" {
			honchoKey = clientHoncho
			s.honcho.SetAPIKey(clientHoncho)
			profile, hasProfile := config.LoadEventProfile()
			if hasProfile && profile.ID != "" && profile.EventType != "" && profile.HostNames != "" {
				s.honcho.EnsureWorkspaceCreated(profile.ID, fmt.Sprintf("%s for %s", profile.EventType, profile.HostNames))
			}
		}

		status := map[string]interface{}{
			"geminiConfigured": geminiKey != "",
			"geminiSet":        geminiKey != "",
			"geminiStatus":     "🔴 Missing (Offline Dry-Run)",
			"honchoConfigured": honchoKey != "",
			"honchoSet":        honchoKey != "",
			"honchoStatus":     "🟡 Inbuilt Local Store (./data/honcho_memory.json)",
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
			placesKey := strings.TrimSpace(r.FormValue("placesKey"))

			if geminiKey != "" {
				_ = config.SaveGeminiAPIKey(geminiKey)
			}
			if honchoKey != "" {
				_ = config.SaveHonchoAPIKey(honchoKey)
			}
			if placesKey != "" {
				_ = config.SavePlacesAPIKey(placesKey)
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<div style="background: rgba(16, 185, 129, 0.15); border: 1px solid rgba(16, 185, 129, 0.4); color: #34d399; padding: 0.75rem 1rem; border-radius: 10px; font-size: 0.8rem; margin-bottom: 1rem;">✅ API Keys activated for your session! Saved to browser localStorage.</div><script>saveClientKeys('%s', '%s', '%s');</script>`, geminiKey, honchoKey, placesKey)
			return
		}
	})

	// Places Autocomplete API Handler (3-Tier Resolution: Google Places API -> Gemini AI Agent -> Curated List)
	mux.HandleFunc("/api/places/autocomplete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var reqData struct {
			Input     string `json:"input"`
			EventType string `json:"eventType"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
			reqData.Input = r.FormValue("input")
			reqData.EventType = r.FormValue("eventType")
		}

		q := strings.TrimSpace(reqData.Input)
		w.Header().Set("Content-Type", "application/json")

		if q == "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"suggestions": []interface{}{}})
			return
		}

		// Tier 1: Check for Google Places API Key
		clientPlacesKey := strings.TrimSpace(r.Header.Get("X-Places-API-Key"))
		placesKey := config.GetPlacesAPIKey()
		if clientPlacesKey != "" {
			placesKey = clientPlacesKey
		}

		if placesKey != "" {
			url := "https://places.googleapis.com/v1/places:autocomplete"
			payload := map[string]interface{}{"input": q}
			pBytes, _ := json.Marshal(payload)

			req, err := http.NewRequestWithContext(r.Context(), "POST", url, bytes.NewBuffer(pBytes))
			if err == nil {
				req.Header.Set("X-Goog-Api-Key", placesKey)
				req.Header.Set("Content-Type", "application/json")

				hc := &http.Client{Timeout: 5 * time.Second}
				if resp, err := hc.Do(req); err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					var resMap map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&resMap); err == nil {
						_ = json.NewEncoder(w).Encode(resMap)
						return
					}
				}
			}
		}

		// Tier 2: Check for Gemini AI Key for AI Venue Agent
		clientGeminiKey := strings.TrimSpace(r.Header.Get("X-Gemini-API-Key"))
		geminiKey := config.LoadConfig().GeminiAPIKey
		if clientGeminiKey != "" {
			geminiKey = clientGeminiKey
		}

		if geminiKey != "" {
			aiSuggestions, err := generator.GenerateAIVenueSuggestions(geminiKey, config.LoadConfig().GeminiTextModel, reqData.EventType, q)
			if err == nil && len(aiSuggestions) > 0 {
				formatted := []map[string]interface{}{}
				for _, item := range aiSuggestions {
					formatted = append(formatted, map[string]interface{}{
						"placePrediction": map[string]interface{}{
							"placeId": item.PlaceID,
							"text":    map[string]interface{}{"text": item.Text},
						},
					})
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"suggestions": formatted})
				return
			}
		}

		// Tier 3: Smart Curated Offline Fallback List
		curated := []string{
			"Palace Grounds, Jayamahal Road, Bengaluru, Karnataka",
			"The Leela Palace, HAL Old Airport Road, Kodihalli, Bengaluru",
			"Taj Mahal Palace, Apollo Bunder, Colaba, Mumbai",
			"ITC Grand Chola, Mount Road, Guindy, Chennai",
			"Umaid Bhawan Palace, Circuit House Road, Jodhpur, Rajasthan",
			"Rambagh Palace, Bhawani Singh Road, Jaipur, Rajasthan",
			"Hyderabad International Convention Centre (HICC), Novotel, Hyderabad",
			"JW Marriott Hotel, Aerocity, New Delhi",
		}

		formatted := []map[string]interface{}{}
		lowerQ := strings.ToLower(q)
		count := 0
		for idx, venue := range curated {
			if strings.Contains(strings.ToLower(venue), lowerQ) || lowerQ == "hotel" || lowerQ == "palace" || lowerQ == "hall" || lowerQ == "bengaluru" || lowerQ == "mumbai" || lowerQ == "chennai" || lowerQ == "jaipur" || lowerQ == "delhi" || lowerQ == "hyderabad" {
				formatted = append(formatted, map[string]interface{}{
					"placePrediction": map[string]interface{}{
						"placeId": fmt.Sprintf("curated-venue-%d", idx+1),
						"text":    map[string]interface{}{"text": venue},
					},
				})
				count++
				if count >= 5 {
					break
				}
			}
		}

		if len(formatted) == 0 {
			// Return top 4 curated if no match
			for idx := 0; idx < 4 && idx < len(curated); idx++ {
				formatted = append(formatted, map[string]interface{}{
					"placePrediction": map[string]interface{}{
						"placeId": fmt.Sprintf("curated-venue-%d", idx+1),
						"text":    map[string]interface{}{"text": curated[idx]},
					},
				})
			}
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{"suggestions": formatted})
	})

	// Places Details API Handler
	mux.HandleFunc("/api/places/details", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var reqData struct {
			PlaceID string `json:"placeId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
			reqData.PlaceID = r.FormValue("placeId")
		}

		w.Header().Set("Content-Type", "application/json")

		clientPlacesKey := strings.TrimSpace(r.Header.Get("X-Places-API-Key"))
		placesKey := config.GetPlacesAPIKey()
		if clientPlacesKey != "" {
			placesKey = clientPlacesKey
		}

		if placesKey != "" && strings.HasPrefix(reqData.PlaceID, "ChI") {
			reqUrl := fmt.Sprintf("https://places.googleapis.com/v1/places/%s", reqData.PlaceID)
			req, err := http.NewRequestWithContext(r.Context(), "GET", reqUrl, nil)
			if err == nil {
				req.Header.Set("X-Goog-Api-Key", placesKey)
				req.Header.Set("X-Goog-FieldMask", "displayName,formattedAddress,googleMapsUri,adrFormatAddress,photos")

				hc := &http.Client{Timeout: 5 * time.Second}
				if resp, err := hc.Do(req); err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					var placeRes struct {
						DisplayName struct {
							Text string `json:"text"`
						} `json:"displayName"`
						FormattedAddress string `json:"formattedAddress"`
						GoogleMapsURI    string `json:"googleMapsUri"`
						AdrFormatAddress string `json:"adrFormatAddress"`
						Photos           []struct {
							Name string `json:"name"`
						} `json:"photos"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&placeRes); err == nil {
						primary := placeRes.DisplayName.Text
						if primary == "" {
							primary = placeRes.FormattedAddress
						}

						photoURL := "https://images.unsplash.com/photo-1519167758481-83f550bb49b3?auto=format&fit=crop&w=1200&q=80"
						if len(placeRes.Photos) > 0 && placeRes.Photos[0].Name != "" {
							photoURL = fmt.Sprintf("https://places.googleapis.com/v1/%s/media?key=%s&maxHeightPx=600&maxWidthPx=800", placeRes.Photos[0].Name, placesKey)
						}

						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"primary_venue":             primary,
							"venue_formatted_address":  placeRes.FormattedAddress,
							"venue_adr_format_address":  placeRes.AdrFormatAddress,
							"address":                   placeRes.FormattedAddress,
							"google_map_url":            placeRes.GoogleMapsURI,
							"google_map_directions_url": fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%s&destination_place_id=%s", url.QueryEscape(primary+", "+placeRes.FormattedAddress), reqData.PlaceID),
							"venue_photo_url":           photoURL,
							"place_id":                  reqData.PlaceID,
						})
						return
					}
				}
			}
		}

		// Fallback for AI/Curated Place IDs
		placeID := reqData.PlaceID
		if placeID == "" {
			placeID = "place-default"
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"primary_venue":             "Event Venue",
			"venue_formatted_address":  "Venue Address",
			"address":                   "Venue Address",
			"google_map_url":            fmt.Sprintf("https://maps.google.com/?q=%s", url.QueryEscape(placeID)),
			"google_map_directions_url": fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%s", url.QueryEscape(placeID)),
			"venue_photo_url":           "https://images.unsplash.com/photo-1519167758481-83f550bb49b3?auto=format&fit=crop&w=1200&q=80",
			"place_id":                  placeID,
		})
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
				"hosts":          profile.HostNames,
				"eventDate":      displayDate,
				"isoDate":        isoDate,
				"venue":          profile.Venue,
				"venueAddress":   profile.VenueAddress,
				"venueDetails":   profile.VenueDetails,
				"currency":       profile.DefaultCurrency,
				"welcomeMessage": profile.WelcomeMessage,
				"rawDetails":     profile.RawDetails,
			})
			return
		}

		if r.Method == http.MethodPost {
			clientHoncho := strings.TrimSpace(r.Header.Get("X-Honcho-API-Key"))
			if clientHoncho != "" {
				s.honcho.SetAPIKey(clientHoncho)
			}

			var eType, hosts, eDate, venue, venueAddress, currency, welcome string
			var vDetails config.VenueDetails

			if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
				var data struct {
					EventType      string              `json:"eventType"`
					Hosts          string              `json:"hosts"`
					EventDate      string              `json:"eventDate"`
					Venue          string              `json:"venue"`
					VenueAddress   string              `json:"venueAddress"`
					VenueDetails   config.VenueDetails `json:"venueDetails"`
					Currency       string              `json:"currency"`
					WelcomeMessage string              `json:"welcomeMessage"`
				}
				_ = json.NewDecoder(r.Body).Decode(&data)
				eType = data.EventType
				hosts = data.Hosts
				eDate = data.EventDate
				venue = data.Venue
				venueAddress = data.VenueAddress
				vDetails = data.VenueDetails
				currency = data.Currency
				welcome = data.WelcomeMessage
			} else {
				eType = r.FormValue("eventType")
				hosts = r.FormValue("hosts")
				eDate = r.FormValue("eventDate")
				venue = r.FormValue("venue")
				venueAddress = r.FormValue("venueAddress")
				currency = r.FormValue("currency")
				welcome = r.FormValue("welcome")
			}

			if currency == "" {
				currency = "INR"
			}
			if eDate == "" {
				eDate = "November 24, 2026"
			}
			if venue == "" {
				venue = "Palace Grounds, Bengaluru"
			}

			if vDetails.PrimaryVenue == "" {
				vDetails.PrimaryVenue = venue
			}
			if vDetails.VenueFormattedAddress == "" && venueAddress != "" {
				vDetails.VenueFormattedAddress = venueAddress
			}

			// Determine planner name from active session or registered Owner
			plannerName := "Event Planner"
			plannerRole := "Lead Event Planner"
			cookie, cErr := r.Cookie("shubh_session")
			if cErr == nil && cookie.Value != "" {
				if sess, ok := config.ValidateSessionToken(cookie.Value); ok {
					users, _ := config.LoadUsers()
					for _, u := range users {
						if u.ID == sess.UserID {
							if u.FullName != "" {
								plannerName = u.FullName
							} else if u.Email != "" {
								plannerName = u.Email
							}
							if u.Role == config.RoleAdmin {
								plannerRole = "Owner / Lead Planner"
							}
							break
						}
					}
				}
			}
			if plannerName == "Event Planner" {
				users, _ := config.LoadUsers()
				for _, u := range users {
					if u.Role == config.RoleAdmin {
						if u.FullName != "" {
							plannerName = u.FullName
						} else if u.Email != "" {
							plannerName = u.Email
						}
						plannerRole = "Owner / Lead Planner"
						break
					}
				}
			}

			err := config.SaveStructuredEventProfileWithBudget(eType, hosts, eDate, venue, currency, welcome, "9:16", plannerName, plannerRole, 0, 0)
			if err == nil {
				profile, hasProfile := config.LoadEventProfile()
				if hasProfile {
					profile.VenueAddress = venueAddress
					profile.VenueDetails = vDetails
					_ = config.SaveFullEventProfile(profile)
					if profile.ID != "" && profile.EventType != "" && profile.HostNames != "" {
						s.honcho.EnsureWorkspaceCreated(profile.ID, fmt.Sprintf("%s for %s", profile.EventType, profile.HostNames))
					}
				}
			}

			if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
				w.Header().Set("Content-Type", "application/json")
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
				} else {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Event profile saved to event_details.md!"})
				}
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err != nil {
				fmt.Fprintf(w, `<div style="background: rgba(239, 68, 68, 0.15); border: 1px solid rgba(239, 68, 68, 0.4); color: #fca5a5; padding: 0.75rem 1rem; border-radius: 10px; font-size: 0.8rem; margin-bottom: 1rem;">⚠️ Failed to save event profile: %v</div>`, err)
			} else {
				fmt.Fprintf(w, `<div style="background: rgba(16, 185, 129, 0.15); border: 1px solid rgba(16, 185, 129, 0.4); color: #34d399; padding: 0.75rem 1rem; border-radius: 10px; font-size: 0.8rem; margin-bottom: 1rem;">✅ Event profile saved to event_details.md!</div><script>fetchEventProfile();</script>`)
			}
		}
	})

	// Itinerary & Timeline REST API Handler
	mux.HandleFunc("/api/itinerary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			items, _ := config.LoadItinerary()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": items,
				"count": len(items),
			})
			return
		}

		if r.Method == http.MethodPost {
			var item config.ItineraryItem
			if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
				_ = json.NewDecoder(r.Body).Decode(&item)
			} else {
				item = config.ItineraryItem{
					ID:          r.FormValue("id"),
					Title:       r.FormValue("title"),
					Date:        r.FormValue("date"),
					Time:        r.FormValue("time"),
					Location:    r.FormValue("location"),
					DressCode:   r.FormValue("dressCode"),
					Description: r.FormValue("description"),
				}
			}

			items, err := config.AddItineraryItem(item)
			w.Header().Set("Content-Type", "application/json")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "items": items})
			}
			return
		}

		if r.Method == http.MethodDelete {
			id := r.URL.Query().Get("id")
			items, err := config.DeleteItineraryItem(id)
			w.Header().Set("Content-Type", "application/json")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "items": items})
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

	// AI Selected Prompt Image Generator API (/api/cards/generate)
	mux.HandleFunc("/api/cards/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var reqData struct {
			Prompt  string `json:"prompt"`
			Aspect  string `json:"aspect"`
			Welcome string `json:"welcome"`
		}
		_ = json.NewDecoder(r.Body).Decode(&reqData)

		if strings.TrimSpace(reqData.Prompt) == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Prompt string is required"})
			return
		}

		profile, _ := config.LoadEventProfile()

		aspect := profile.TargetAspect
		if aspect == "" {
			aspect = strings.TrimSpace(reqData.Aspect)
			if aspect == "" {
				aspect = "4:5"
			}
		}

		welcomeMsg := strings.TrimSpace(reqData.Welcome)
		if welcomeMsg == "" {
			welcomeMsg = profile.WelcomeMessage
		}

		// Use BasicBuilder CompileStructured matching TUI design pipeline 1:1
		builder := generator.NewBasicBuilder()
		compiled := builder.CompileStructured(generator.EventData{
			EventType:      profile.EventType,
			HostNames:      profile.HostNames,
			EventDate:      profile.EventDate,
			Venue:          profile.Venue,
			WelcomeMessage: welcomeMsg,
			VisualPrompt:   reqData.Prompt,
			Aspect:         aspect,
		})

		clientGemini := strings.TrimSpace(r.Header.Get("X-Gemini-API-Key"))
		geminiKey := strings.TrimSpace(config.LoadConfig().GeminiAPIKey)
		if clientGemini != "" {
			geminiKey = clientGemini
		}
		if geminiKey == "" {
			geminiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
		}

		imageBytes, err := generator.GenerateAIImage(geminiKey, compiled.CorePrompt, aspect)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		outputDir := config.LoadConfig().OutputDir
		if outputDir == "" {
			outputDir = "./output"
		}
		_ = os.MkdirAll(outputDir, 0755)

		sessionID := fmt.Sprintf("session-%s", profile.GetEventID())
		filename := fmt.Sprintf("shubh_design_%s_%d.png", sessionID, time.Now().Unix())
		filePath := filepath.Join(outputDir, filename)
		_ = os.WriteFile(filePath, imageBytes, 0644)

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"filename": filename,
			"path":     "/output/" + filename,
		})
	})

	// List Generated Output Cards API (/api/outputs & /api/cards)
	outputsHandler := func(w http.ResponseWriter, r *http.Request) {
		outputDir := config.LoadConfig().OutputDir
		if outputDir == "" {
			outputDir = "./output"
		}
		_ = os.MkdirAll(outputDir, 0755)

		entries, err := os.ReadDir(outputDir)
		w.Header().Set("Content-Type", "application/json")

		sessionID := r.URL.Query().Get("sessionID")
		if sessionID == "" {
			sessionID = r.Header.Get("X-Session-ID")
		}
		if sessionID == "" {
			profile, _ := config.LoadEventProfile()
			if profile.GetEventID() != "" {
				sessionID = "session-" + profile.GetEventID()
			}
		}

		type CardItem struct {
			Name    string `json:"name"`
			Path    string `json:"path"`
			ModTime int64  `json:"modTime"`
		}

		var cards []CardItem
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".png") || strings.HasSuffix(entry.Name(), ".jpg")) {
					if sessionID != "" && strings.Contains(entry.Name(), "session-") && !strings.Contains(entry.Name(), sessionID) {
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
	}
	mux.HandleFunc("/api/outputs", outputsHandler)
	mux.HandleFunc("/api/cards", outputsHandler)

	// Static asset server for output cards
	outDirStatic := config.LoadConfig().OutputDir
	if outDirStatic == "" {
		outDirStatic = "./output"
	}
	_ = os.MkdirAll(outDirStatic, 0755)
	fileServer := http.FileServer(http.Dir(outDirStatic))
	mux.Handle("/output/", http.StripPrefix("/output/", fileServer))

	serverAddr := fmt.Sprintf(":%d", s.port)
	log.Printf("🌐 [Web UI] Server running on http://localhost%s", serverAddr)

	var handler http.Handler = mux
	handler = APIKeyContextMiddleware(handler)
	handler = HTMXMiddleware(handler)

	return http.ListenAndServe(serverAddr, handler)
}

func handleWebSlashCommand(w http.ResponseWriter, flusher http.Flusher, parsed command.ParsedInput, _ string) bool {
	emitEvent := func(agentName, text string) {
		evJSON, _ := json.Marshal(client.AgentStreamEvent{
			Type:    "content",
			Agent:   agentName,
			Content: text,
		})
		fmt.Fprintf(w, "data: %s\n\n", string(evJSON))
		flusher.Flush()
	}

	switch parsed.Type {
	case command.CmdHelp:
		helpText := `📌 Main Slash Commands:
  • /generate [details] - Compile context & generate invitation design
  • /event [details]    - View/update profile details in event_details.md
  • /budget [amount]   - View or set estimated budget (e.g. /budget 300000)
  • /rsvp              - View guest roster & dietary facts
  • /add-rsvp          - Add/update guest RSVP & transport details
  • /timeline          - View chronological ceremony schedule
  • /currency [code]   - Set default currency (e.g. /currency INR, USD)
  • /welcome [text]    - View or set AI welcome message subheaders
  • /aspect [ratio]    - Set aspect ratio (9:16, 4:5, 1:1, 16:9)
  • /style [preset]    - Select aesthetic design style (e.g. /style paper cut)
  • /suggest [theme]   - Generate AI prompt suggestions
  • /refine [changes]  - Apply tweaks to active design
  • /preview           - Open local web preview browser
  • /honcho            - Inspect Honcho Cloud AI memory cards
  • /planner [name]    - Update event planner name & role
  • /export            - Export design or event details
  • /wizard            - Launch interactive step-by-step setup wizard
  • /config [key]      - View or set Gemini API key
  • /clear             - Clear terminal log screen
  • /reset             - Restart guided setup wizard
  • /help              - Display this reference guide

🔗 Command Aliases & Shortcuts:
  • /design, /create, /gen      --> /generate
  • /edit, /modify              --> /refine
  • /ideas, /theme, /sug        --> /suggest
  • /preset, /aesthetic, /sty   --> /style
  • /profile, /details          --> /event
  • /ratio, /res, /resolution   --> /aspect
  • /curr, /currency-code       --> /currency
  • /subheader, /msg            --> /welcome
  • /finance, /spend            --> /budget
  • /rsvps, /guests             --> /rsvp
  • /addrsvp, /new-rsvp         --> /add-rsvp
  • /schedule, /itinerary       --> /timeline
  • /memory, /cards             --> /honcho
  • /wiz                        --> /wizard
  • /web, /open                 --> /preview
  • /key, /apikey               --> /config
  • /cls                        --> /clear
  • /h, /?                      --> /help`
		emitEvent("SYSTEM", helpText)
		return true

	case command.CmdSuggest:
		profile, _ := config.LoadEventProfile()
		eType := profile.EventType
		if eType == "" {
			eType = "Naming Ceremony"
		}
		style := profile.Style
		if style == "" {
			style = "South Indian Traditional"
		}
		cfg := config.LoadConfig()
		ideas, _ := generator.GenerateAIPromptIdeas(cfg.GeminiAPIKey, eType, style)
		if len(ideas) == 0 {
			ideas = generator.GenerateFallbackPromptIdeas(eType, style)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("✨ <strong>AI Prompt Suggestions for '%s'</strong> (Style: <code class='text-amber-400'>%s</code>):<br><div class='space-y-2.5 mt-2'>", eType, style))
		for idx, idea := range ideas {
			escapedPrompt := strings.ReplaceAll(idea.PromptText, "'", "\\'")
			escapedPrompt = strings.ReplaceAll(escapedPrompt, `"`, "&quot;")
			escapedPrompt = strings.ReplaceAll(escapedPrompt, "\n", " ")
			sb.WriteString(fmt.Sprintf(`<div class="bg-slate-950 p-3 rounded-xl border border-slate-800 space-y-1.5">
				<div class="flex items-center justify-between text-xs">
					<strong class="text-amber-400 font-bold">Option %d: %s</strong>
					<button type="button" onclick="switchTab('design'); renderSelectedPrompt('%s')" class="px-2.5 py-1 bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold rounded-lg text-[11px] transition-all shadow-md">🎨 Render Image ➔</button>
				</div>
				<p class="text-[11px] text-slate-300 font-mono leading-relaxed select-all bg-slate-900/60 p-2 rounded-lg border border-slate-800/60">%s</p>
			</div>`, idx+1, idea.ThemeTitle, escapedPrompt, idea.PromptText))
		}
		sb.WriteString("</div>")
		emitEvent("SYSTEM", sb.String())
		return true

	case command.CmdAspect:
		ratio := strings.TrimSpace(parsed.EventDetails)
		profile, _ := config.LoadEventProfile()
		if ratio != "" {
			profile.TargetAspect = ratio
			_ = config.SaveStructuredEventProfileWithBudget(profile.EventType, profile.HostNames, profile.EventDate, profile.Venue, profile.DefaultCurrency, profile.WelcomeMessage, profile.TargetAspect, profile.PlannerName, profile.PlannerRole, profile.TotalBudget, profile.EstimatedGuests)
			emitEvent("SYSTEM", fmt.Sprintf("📐 Aspect ratio updated to '%s'! Saved to event profile.<script>if(typeof fetchEventProfile==='function')fetchEventProfile();</script>", ratio))
		} else {
			aspectHTML := fmt.Sprintf(`📐 <strong>Aspect Ratio Options</strong> (Active: <code class="text-amber-400">%s</code>):<br><div class="flex flex-wrap gap-2 mt-2">
				<button type="button" onclick="runPillCommand('/aspect 9:16')" class="px-3 py-1.5 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-xl font-semibold text-xs transition-all">📱 9:16 Vertical Poster</button>
				<button type="button" onclick="runPillCommand('/aspect 4:5')" class="px-3 py-1.5 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-xl font-semibold text-xs transition-all">📸 4:5 Instagram Portrait</button>
				<button type="button" onclick="runPillCommand('/aspect 1:1')" class="px-3 py-1.5 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-xl font-semibold text-xs transition-all">🟦 1:1 Square</button>
				<button type="button" onclick="runPillCommand('/aspect 16:9')" class="px-3 py-1.5 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-xl font-semibold text-xs transition-all">📺 16:9 Landscape Banner</button>
			</div>`, profile.TargetAspect)
			emitEvent("SYSTEM", aspectHTML)
		}
		return true

	case command.CmdStyle:
		preset := strings.TrimSpace(parsed.EventDetails)
		profile, _ := config.LoadEventProfile()
		if preset != "" {
			profile.Style = preset
			_ = config.SaveStructuredEventProfileWithBudget(profile.EventType, profile.HostNames, profile.EventDate, profile.Venue, profile.DefaultCurrency, profile.WelcomeMessage, profile.TargetAspect, profile.PlannerName, profile.PlannerRole, profile.TotalBudget, profile.EstimatedGuests)
			if jBytes, err := json.MarshalIndent(profile, "", "  "); err == nil {
				_ = os.WriteFile(filepath.Join(".", "data", "event-details.json"), jBytes, 0644)
			}
			emitEvent("SYSTEM", fmt.Sprintf("🎨 Aesthetic style preset updated to '%s'! Saved to event profile.<script>if(typeof fetchEventProfile==='function')fetchEventProfile();</script>", preset))
		} else {
			styleHTML := fmt.Sprintf(`🎨 <strong>Select Aesthetic Design Style Preset</strong> (Active Style: <code class="text-amber-400">%s</code>):<br><div class="grid grid-cols-1 sm:grid-cols-2 gap-2 mt-2">
				<button type="button" onclick="runPillCommand('/style South Indian Traditional')" class="p-2 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 rounded-xl text-left font-semibold text-xs transition-all">✨ 1. South Indian Traditional</button>
				<button type="button" onclick="runPillCommand('/style Paper Cut Art')" class="p-2 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 rounded-xl text-left font-semibold text-xs transition-all">✂️ 2. Paper Cut Art</button>
				<button type="button" onclick="runPillCommand('/style Clay 3D Render')" class="p-2 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 rounded-xl text-left font-semibold text-xs transition-all">🎨 3. Clay 3D Render</button>
				<button type="button" onclick="runPillCommand('/style Pop Art')" class="p-2 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 rounded-xl text-left font-semibold text-xs transition-all">💥 4. Pop Art</button>
				<button type="button" onclick="runPillCommand('/style Mughal Palace')" class="p-2 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 rounded-xl text-left font-semibold text-xs transition-all">🕌 5. Mughal Palace</button>
				<button type="button" onclick="runPillCommand('/style Minimalist Gold Foil')" class="p-2 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 rounded-xl text-left font-semibold text-xs transition-all">👑 6. Minimalist Gold Foil</button>
				<button type="button" onclick="runPillCommand('/style Loose Watercolor')" class="p-2 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 rounded-xl text-left font-semibold text-xs transition-all col-span-full">🌸 7. Loose Watercolor</button>
			</div>`, profile.Style)
			emitEvent("SYSTEM", styleHTML)
		}
		return true

	case command.CmdWelcome:
		msg := strings.TrimSpace(parsed.EventDetails)
		profile, _ := config.LoadEventProfile()
		if msg != "" {
			profile.WelcomeMessage = msg
			_ = config.SaveStructuredEventProfileWithBudget(profile.EventType, profile.HostNames, profile.EventDate, profile.Venue, profile.DefaultCurrency, profile.WelcomeMessage, profile.TargetAspect, profile.PlannerName, profile.PlannerRole, profile.TotalBudget, profile.EstimatedGuests)
			emitEvent("SYSTEM", fmt.Sprintf("💬 Welcome subheader message set to: \"%s\". Saved to event profile.", msg))
		} else {
			emitEvent("SYSTEM", fmt.Sprintf("💬 Current Welcome Subheader: \"%s\"\n\nUsage: '/welcome Celebrating our joy and new beginnings'", profile.WelcomeMessage))
		}
		return true

	case command.CmdCurrency:
		curr := strings.TrimSpace(strings.ToUpper(parsed.EventDetails))
		profile, _ := config.LoadEventProfile()
		if curr != "" {
			profile.DefaultCurrency = curr
			_ = config.SaveStructuredEventProfileWithBudget(profile.EventType, profile.HostNames, profile.EventDate, profile.Venue, profile.DefaultCurrency, profile.WelcomeMessage, profile.TargetAspect, profile.PlannerName, profile.PlannerRole, profile.TotalBudget, profile.EstimatedGuests)
			emitEvent("SYSTEM", fmt.Sprintf("🔤 Default currency code set to '%s'. Saved to event profile.", curr))
		} else {
			currHTML := fmt.Sprintf(`🔤 <strong>Select Default Currency Code</strong> (Active: <code class="text-amber-400">%s</code>):<br><div class="flex flex-wrap gap-2 mt-2">
				<button type="button" onclick="runPillCommand('/currency INR')" class="px-3 py-1.5 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-xl font-semibold text-xs transition-all">₹ INR (Indian Rupee)</button>
				<button type="button" onclick="runPillCommand('/currency USD')" class="px-3 py-1.5 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-xl font-semibold text-xs transition-all">$ USD (US Dollar)</button>
				<button type="button" onclick="runPillCommand('/currency EUR')" class="px-3 py-1.5 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-xl font-semibold text-xs transition-all">€ EUR (Euro)</button>
				<button type="button" onclick="runPillCommand('/currency GBP')" class="px-3 py-1.5 bg-slate-950 hover:bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-xl font-semibold text-xs transition-all">£ GBP (British Pound)</button>
			</div>`, profile.DefaultCurrency)
			emitEvent("SYSTEM", currHTML)
		}
		return true

	case command.CmdBudget:
		amtStr := strings.TrimSpace(parsed.EventDetails)
		profile, _ := config.LoadEventProfile()
		if amtStr != "" {
			if b, err := strconv.ParseFloat(amtStr, 64); err == nil {
				profile.TotalBudget = b
				_ = config.SaveStructuredEventProfileWithBudget(profile.EventType, profile.HostNames, profile.EventDate, profile.Venue, profile.DefaultCurrency, profile.WelcomeMessage, profile.TargetAspect, profile.PlannerName, profile.PlannerRole, profile.TotalBudget, profile.EstimatedGuests)
				emitEvent("SYSTEM", fmt.Sprintf("💰 Estimated event budget updated to '%s %.2f'. Saved to profile.", profile.DefaultCurrency, b))
			} else {
				emitEvent("SYSTEM", fmt.Sprintf("⚠️ Invalid budget amount '%s'. Usage: '/budget 500000'", amtStr))
			}
		} else {
			emitEvent("SYSTEM", fmt.Sprintf("💰 Budget Summary for %s:\n• Total Budget: %s %.2f\n• Registered Sub-Events: %d\n\nUsage: '/budget 500000'", profile.EventType, profile.DefaultCurrency, profile.TotalBudget, len(profile.Itinerary)))
		}
		return true

	case command.CmdRSVP, command.CmdAddRSVP:
		profile, _ := config.LoadEventProfile()
		rsvps, _ := config.LoadGuestRSVPsFromMarkdown()
		if len(rsvps) == 0 {
			emitEvent("SYSTEM", fmt.Sprintf("👥 Guest Roster for %s:\nNo guest RSVPs recorded yet. Add guests in Tab 2 or type '/add-rsvp Sharma Family, 4, Vegetarian'.", profile.EventType))
		} else {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("👥 Guest Roster Summary (%d groups recorded):\n", len(rsvps)))
			for idx, g := range rsvps {
				sb.WriteString(fmt.Sprintf(" %d. %s (%d guests) • %s\n", idx+1, g.Name, g.Headcount, g.Dietary))
			}
			emitEvent("SYSTEM", sb.String())
		}
		return true

	case command.CmdTimeline:
		profile, _ := config.LoadEventProfile()
		if len(profile.Itinerary) == 0 {
			emitEvent("SYSTEM", fmt.Sprintf("📅 Ceremony Timeline for %s:\nNo schedule slots added yet. Add slots in Tab 3 or tell AI Copilot: 'Add Haldi on Sep 19 at 10 AM'.", profile.EventType))
		} else {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("📅 Chronological Run-of-Show (%d slots):\n", len(profile.Itinerary)))
			for idx, s := range profile.Itinerary {
				sb.WriteString(fmt.Sprintf(" %d. %s | %s | %s (%s)\n", idx+1, s.Title, s.Time, s.Location, s.DressCode))
			}
			emitEvent("SYSTEM", sb.String())
		}
		return true

	case command.CmdEvent:
		profile, _ := config.LoadEventProfile()
		info := fmt.Sprintf("👑 Event Profile Summary:\n• Event Type: %s\n• Hosts/Couple: %s\n• Event Date: %s\n• Primary Venue: %s\n• Subheader: \"%s\"\n• Aspect Ratio: %s\n• Currency: %s",
			profile.EventType, profile.HostNames, profile.EventDate, profile.Venue, profile.WelcomeMessage, profile.TargetAspect, profile.DefaultCurrency)
		emitEvent("SYSTEM", info)
		return true

	case command.CmdPreview:
		emitEvent("SYSTEM", "🌐 Web Preview interface is active at http://localhost:3000.")
		return true

	case command.CmdClear:
		emitEvent("SYSTEM", "🧹 Log screen cleared.")
		return true
	}

	return false
}
