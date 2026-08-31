package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	genkitengine "shubh-plan-web/pkg/genkit"
	"shubh-plan-web/pkg/auth"
	"shubh-plan-web/pkg/middleware"
	"shubh-plan-web/pkg/store"

	"github.com/firebase/genkit/go/ai"
)

type Server struct {
	port   int
	store  *store.DataStore
	engine *genkitengine.Engine
	flows  *genkitengine.FlowRegistry
	agents *genkitengine.AgentRegistry
	tools  map[string]ai.Tool
	webFS  embed.FS
}

func NewServer(port int, webFS embed.FS, promptFS embed.FS) *Server {
	dataStore := store.GetStore()
	engine := genkitengine.InitEngine(context.Background(), promptFS)
	tools := genkitengine.RegisterTools(engine.Genkit, dataStore)
	agents := genkitengine.RegisterAgents(engine, dataStore, tools)
	flows := genkitengine.RegisterFlows(engine, dataStore, tools, agents)

	log.Printf("[Genkit Boot] Registered %d tools (getEventDetails, updateEventDetails, searchVenueInfo, etc.)", len(tools))
	log.Printf("[Genkit Boot] Initialized Agents: EventPlannerAgent, GuestConciergeAgent, InvitationStudioAgent, ItineraryAgent")
	log.Printf("[Genkit Boot] Text Model: %s | Image Model: %s", genkitengine.DefaultModelName, genkitengine.DefaultImageModelName)
	log.Printf("[Genkit Boot] Registered 5 Core Genkit Flows & HTTP Handlers")

	return &Server{
		port:   port,
		store:  dataStore,
		engine: engine,
		flows:  flows,
		agents: agents,
		tools:  tools,
		webFS:  webFS,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// 1. Static Filesystem Handler (Local Disk + Embedded Web Assets with Cache Prevention)
	webSub, err := fs.Sub(s.webFS, "web")
	if err != nil {
		log.Printf("[Server Warning] Embedded web assets sub-filesystem error: %v", err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		cleanPath := filepath.Clean(r.URL.Path)
		if cleanPath == "\\" || cleanPath == "/" {
			cleanPath = "/index.html"
		}

		// 1. Try loading asset directly from persistent SHUBH_DATA_DIR volume mount first
		dataDir := strings.TrimSpace(os.Getenv("SHUBH_DATA_DIR"))
		if dataDir == "" {
			dataDir = "./data"
		}
		persistentAssetPath := filepath.Join(dataDir, filepath.FromSlash(cleanPath))
		if stat, statErr := os.Stat(persistentAssetPath); statErr == nil && !stat.IsDir() {
			http.ServeFile(w, r, persistentAssetPath)
			return
		}

		// 2. Try loading static asset directly from local ./web directory
		localDiskPath := filepath.Join(".", "web", filepath.FromSlash(cleanPath))
		if stat, statErr := os.Stat(localDiskPath); statErr == nil && !stat.IsDir() {
			http.ServeFile(w, r, localDiskPath)
			return
		}

		// Fallback to embedded filesystem
		if webSub != nil {
			http.FileServer(http.FS(webSub)).ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})

	// Dynamic asset generator for SVG card placeholders
	mux.HandleFunc("GET /assets/", s.handleAssetPlaceholder)

	// Healthcheck & Mode Endpoints
	mux.HandleFunc("GET /api/health", s.handleHealthCheck)
	mux.HandleFunc("GET /api/mode", s.handleGetMode)

	// Auth Endpoints (Aligned with apps/shubh-plan-open)
	mux.HandleFunc("POST /api/auth/signup", s.handleSignup)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.handleGetMe)
	mux.HandleFunc("POST /api/auth/guest-demo", s.handleGuestDemo)
	mux.HandleFunc("DELETE /api/auth/account", s.handleDeleteAccount)

	// 2. Genkit Flow HTTP Handlers
	mux.HandleFunc("POST /api/flows/eventAssistantFlow", s.flows.AssistantHandler)
	mux.HandleFunc("POST /api/flows/invitationGeneratorFlow", s.handleCreateDesign)
	mux.HandleFunc("POST /api/flows/invitationPromptSuggestionsFlow", s.handleCreatePromptSuggestions)
	mux.HandleFunc("POST /api/flows/rsvpManagementFlow", s.flows.RSVPHandler)
	mux.HandleFunc("POST /api/flows/itineraryPlannerFlow", s.flows.ItineraryHandler)

	// 3. Real-Time SSE Streaming Assistant Chat
	mux.HandleFunc("GET /api/stream/assistant", s.handleStreamingAssistant)

	// 4. REST Domain Endpoints
	mux.HandleFunc("GET /api/event", s.handleGetEvent)
	mux.HandleFunc("POST /api/event", s.handleUpdateEvent)
	mux.HandleFunc("GET /api/venue/search", s.handleSearchVenue)
	mux.HandleFunc("GET /api/guests", s.handleListGuests)
	mux.HandleFunc("POST /api/guests", s.handleAddOrUpdateGuest)
	mux.HandleFunc("POST /api/guests/{id}/rsvp", s.handleToggleRSVP)
	mux.HandleFunc("DELETE /api/guests/{id}", s.handleDeleteGuest)
	mux.HandleFunc("GET /api/itinerary", s.handleListItinerary)
	mux.HandleFunc("POST /api/itinerary", s.handleAddItineraryItem)
	mux.HandleFunc("GET /api/designs", s.handleListDesigns)
	mux.HandleFunc("POST /api/designs/suggestions", s.handleCreatePromptSuggestions)
	mux.HandleFunc("POST /api/designs", s.handleCreateDesign)
	mux.HandleFunc("DELETE /api/designs", s.handleDeleteDesign)
	mux.HandleFunc("POST /api/reset", s.handleResetStore)

	// Wrap entire mux in middleware chain (CORS, Auth, Logging, Panic Recovery)
	handler := middleware.Chain(mux,
		middleware.RecoveryMiddleware,
		middleware.LoggingMiddleware,
		middleware.CORSMiddleware,
		middleware.AuthMiddleware,
	)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("=================================================================")
	log.Printf("🚀 Shubh Plan Web Server listening on http://localhost:%d", s.port)
	log.Printf("   • AI Model: googleai/gemini-flash-latest")
	log.Printf("   • API Key Active: %t", s.engine.HasAPIKey)
	log.Printf("   • Genkit Agents Active: 4 registered experimental agents")
	log.Printf("   • Genkit Flows Active: 4 registered endpoints")
	log.Printf("   • Web UI: Embedded SPA (Vanilla CSS + SSE Streaming)")
	log.Printf("   • Graceful Shutdown: Active (Ctrl+C terminates cleanly)")
	log.Printf("=================================================================")

	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Println("\n[Server Shutdown] Signal received (Ctrl+C). Terminating gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}

// REST & HTMX Handlers

func (s *Server) isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true" || strings.Contains(r.Header.Get("Accept"), "text/html")
}

func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	if userID == "" {
		token := ""
		if cookie, err := r.Cookie("shubh_session"); err == nil {
			token = cookie.Value
		}
		if token == "" {
			token = r.Header.Get("X-Session-ID")
		}
		if token != "" {
			if _, ok := auth.GetAuthManager().GetSession(token); ok {
				return true
			}
		}
		http.Error(w, `{"error":"Unauthorized","message":"Authentication required"}`, http.StatusUnauthorized)
		return false
	}
	return true
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.store.GetEvent())
}

func (s *Server) handleUpdateEvent(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var profile store.EventProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}
	updated := s.store.UpdateEvent(profile)
	json.NewEncoder(w).Encode(updated)
}

func (s *Server) handleSearchVenue(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		http.Error(w, `{"error":"Query parameter required"}`, http.StatusBadRequest)
		return
	}

	vd := genkitengine.VerifyVenueWithGoogleMaps(r.Context(), query, "")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"query":   query,
		"results": []store.VenueDetails{vd},
	})
}

func (s *Server) handleListGuests(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.store.ListGuests())
}

func (s *Server) handleAddOrUpdateGuest(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var guest store.Guest
	if err := json.NewDecoder(r.Body).Decode(&guest); err != nil {
		http.Error(w, `{"error":"Invalid guest payload"}`, http.StatusBadRequest)
		return
	}
	saved := s.store.AddOrUpdateGuest(guest)
	json.NewEncoder(w).Encode(saved)
}

func (s *Server) handleToggleRSVP(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	id := r.PathValue("id")
	guest, exists := s.store.ToggleGuestRSVP(id)
	if !exists {
		http.Error(w, `{"error":"Guest not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(guest)
}

func (s *Server) handleDeleteGuest(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	id := r.PathValue("id")
	s.store.DeleteGuest(id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"deleted"}`))
}

func (s *Server) handleListItinerary(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.store.ListItinerary())
}

func (s *Server) handleAddItineraryItem(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var item store.ItineraryItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, `{"error":"Invalid itinerary payload"}`, http.StatusBadRequest)
		return
	}
	saved := s.store.AddItineraryItem(item)
	json.NewEncoder(w).Encode(saved)
}

func (s *Server) handleListDesigns(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.store.ListDesigns())
}

func (s *Server) checkAPIKeyPresent(r *http.Request) bool {
	if s.engine.HasAPIKey {
		return true
	}
	apiKeyHeader := strings.TrimSpace(r.Header.Get("X-Gemini-API-Key"))
	if apiKeyHeader == "" {
		apiKeyHeader = strings.TrimSpace(r.Header.Get("X-Google-API-Key"))
	}
	return apiKeyHeader != ""
}

func (s *Server) handleCreatePromptSuggestions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !s.checkAPIKeyPresent(r) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "API_KEY_REQUIRED",
			"message": "Gemini API Key is required for prompt suggestion synthesis. Please configure your API key.",
		})
		return
	}

	var req genkitengine.PromptSuggestionInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Prompt Suggestions Endpoint Error] Invalid payload: %v", err)
		http.Error(w, `{"error":"Invalid prompt suggestion payload"}`, http.StatusBadRequest)
		return
	}
	if req.CustomElements == nil {
		req.CustomElements = []string{}
	}

	log.Printf("[Prompt Suggestions Endpoint] Synthesizing prompt suggestions for Style: %q, AspectRatio: %q", req.StylePreset, req.AspectRatio)

	res, err := s.flows.RunPromptSuggestions(r.Context(), req)
	if err != nil {
		log.Printf("[Prompt Suggestions Endpoint Error] Synthesis failed: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	log.Printf("[Prompt Suggestions Endpoint Success] Synthesized %d prompt suggestions", len(res.Suggestions))
	json.NewEncoder(w).Encode(res)
}

func (s *Server) handleCreateDesign(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !s.checkAPIKeyPresent(r) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "API_KEY_REQUIRED",
			"message": "Gemini API Key is required for invitation artwork generation. Please configure your API key.",
		})
		return
	}

	var req genkitengine.InvitationGenInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Image Generator Endpoint Error] Invalid payload: %v", err)
		http.Error(w, `{"error":"Invalid design payload"}`, http.StatusBadRequest)
		return
	}
	if req.CustomElements == nil {
		req.CustomElements = []string{}
	}

	style := req.StyleTheme
	if style == "" {
		style = req.AestheticTheme
	}
	prompt := req.PromptText
	if prompt == "" {
		prompt = req.Prompt
	}

	log.Printf("[Image Generator Endpoint] Received request to generate invitation artwork (Style: %q, AspectRatio: %q, Prompt: %q)", style, req.AspectRatio, prompt)

	res, err := s.flows.RunInvitation(r.Context(), req)
	if err != nil {
		log.Printf("[Image Generator Endpoint Error] Flow execution failed: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	log.Printf("[Image Generator Endpoint Success] Generated design concept ID %s (ImageURL: %s)", res.MainConcept.ID, res.MainConcept.ImageURL)
	json.NewEncoder(w).Encode(res)
}

func (s *Server) handleDeleteDesign(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing design ID"})
		return
	}

	ok := s.store.DeleteDesign(id)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Design not found"})
		return
	}

	log.Printf("[Design Endpoint Success] Deleted design concept ID %s", id)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Design concept deleted successfully",
		"id":      id,
	})
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	appMode := strings.ToLower(strings.TrimSpace(os.Getenv("APP_MODE")))
	if appMode == "" {
		appMode = "demo"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "ok",
		"mode":            appMode,
		"hasServerAPIKey": s.engine.HasAPIKey,
		"time":            time.Now(),
	})
}

func (s *Server) handleGetMode(w http.ResponseWriter, r *http.Request) {
	appMode := strings.ToLower(strings.TrimSpace(os.Getenv("APP_MODE")))
	if appMode == "" {
		appMode = "demo"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mode":            appMode,
		"hasServerAPIKey": s.engine.HasAPIKey,
	})
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"fullName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid signup payload"}`, http.StatusBadRequest)
		return
	}

	am := auth.GetAuthManager()
	user, err := am.RegisterUser(req.Email, req.Password, req.FullName, auth.RolePlanner)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	session, err := am.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "shubh_session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"session": session,
		"user":    user,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid login payload"}`, http.StatusBadRequest)
		return
	}

	am := auth.GetAuthManager()
	session, err := am.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "shubh_session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"session": session,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := ""
	if cookie, err := r.Cookie("shubh_session"); err == nil {
		token = cookie.Value
	}
	if token == "" {
		token = r.Header.Get("X-Session-ID")
	}

	if token != "" {
		auth.GetAuthManager().InvalidateSession(token)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "shubh_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := ""
	if cookie, err := r.Cookie("shubh_session"); err == nil {
		token = cookie.Value
	}
	if token == "" {
		token = r.Header.Get("X-Session-ID")
	}

	session, ok := auth.GetAuthManager().GetSession(token)
	if !ok {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(session)
}

func (s *Server) handleGuestDemo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	am := auth.GetAuthManager()
	session := am.CreateGuestDemoSession()

	http.SetCookie(w, &http.Cookie{
		Name:     "shubh_session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"session": session,
	})
}

func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := ""
	if cookie, err := r.Cookie("shubh_session"); err == nil {
		token = cookie.Value
	}
	if token == "" {
		token = r.Header.Get("X-Session-ID")
	}

	am := auth.GetAuthManager()
	session, ok := am.GetSession(token)
	if !ok {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	err := am.DeleteUser(session.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "shubh_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "account_deleted",
		"message": "User account and session successfully deleted.",
	})
}

func (s *Server) handleResetStore(w http.ResponseWriter, r *http.Request) {
	s.store.ClearStore()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success","message":"Domain store reset to clean slate"}`))
}

// handleStreamingAssistant streams tokens via SSE to client web UI.
func (s *Server) handleStreamingAssistant(w http.ResponseWriter, r *http.Request) {
	promptMsg := r.URL.Query().Get("message")
	if promptMsg == "" {
		promptMsg = "Hello! Give me a quick summary of this event."
	}
	sessionID := r.URL.Query().Get("sessionId")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, _ := w.(http.Flusher)
	flushNow := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}

	// Send immediate SSE connection ping to keep browser EventSource open during agent tool execution
	fmt.Fprintf(w, ": ping\n\n")
	flushNow()

	// Run flow with request context to carry client-supplied API keys from headers/query params
	out, err := s.flows.RunAssistant(r.Context(), genkitengine.AssistantInput{
		UserMessage: promptMsg,
		SessionID:   sessionID,
	})
	if err != nil {
		errPayload, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
		errStr := strings.ReplaceAll(string(errPayload), "\n", "\\n")
		fmt.Fprintf(w, "data: %s\n\n", errStr)
		flushNow()
		return
	}

	// Stream response preserving original line breaks and list formatting
	lines := strings.Split(out.Response, "\n")
	for lIdx, line := range lines {
		select {
		case <-r.Context().Done():
			log.Println("[SSE Assistant Stream] Client disconnected, aborting token streaming loop.")
			return
		default:
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			if lIdx < len(lines)-1 {
				dataPayload, _ := json.Marshal(map[string]interface{}{
					"token":     "\n",
					"sessionId": out.SessionID,
					"done":      false,
				})
				dataStr := strings.ReplaceAll(string(dataPayload), "\n", "\\n")
				fmt.Fprintf(w, "data: %s\n\n", dataStr)
				flushNow()
			}
			continue
		}

		for wIdx, word := range words {
			select {
			case <-r.Context().Done():
				log.Println("[SSE Assistant Stream] Client disconnected, aborting token streaming loop.")
				return
			default:
			}

			chunk := word
			if wIdx < len(words)-1 {
				chunk += " "
			} else if lIdx < len(lines)-1 {
				chunk += "\n"
			}
			isDone := lIdx == len(lines)-1 && wIdx == len(words)-1
			dataPayload, _ := json.Marshal(map[string]interface{}{
				"token":     chunk,
				"sessionId": out.SessionID,
				"done":      isDone,
			})
			dataStr := strings.ReplaceAll(string(dataPayload), "\n", "\\n")
			dataStr = strings.ReplaceAll(dataStr, "\r", "\\r")

			fmt.Fprintf(w, "data: %s\n\n", dataStr)
			flushNow()
			time.Sleep(15 * time.Millisecond)
		}
	}
}

// handleAssetPlaceholder renders custom card assets or fallback PNGs.
func (s *Server) handleAssetPlaceholder(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/assets/")
	if filename != "" {
		pathsToTry := []string{
			fmt.Sprintf("./web/assets/%s", filename),
			fmt.Sprintf("./assets/%s", filename),
		}
		for _, p := range pathsToTry {
			if _, err := os.Stat(p); err == nil {
				if strings.HasSuffix(filename, ".svg") {
					w.Header().Set("Content-Type", "image/svg+xml")
				} else {
					w.Header().Set("Content-Type", "image/png")
				}
				w.Header().Set("Cache-Control", "public, max-age=86400")
				http.ServeFile(w, r, p)
				return
			}
		}

		// Fallback: If requested PNG does not exist on disk, serve any valid PNG card from web/assets
		dirsToTry := []string{"./web/assets", "./assets"}
		for _, dir := range dirsToTry {
			entries, err := os.ReadDir(dir)
			if err == nil {
				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".png") {
						w.Header().Set("Content-Type", "image/png")
						w.Header().Set("Cache-Control", "public, max-age=86400")
						http.ServeFile(w, r, fmt.Sprintf("%s/%s", dir, entry.Name()))
						return
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, "./web/assets/card_1787721714508660800.png")
}
