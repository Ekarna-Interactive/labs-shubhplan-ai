package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"shubh-plan-web/pkg/auth"
	"shubh-plan-web/pkg/middleware"
	"shubh-plan-web/pkg/store"
)

// Dummy embed.FS for testing
var dummyWebFS embed.FS
var dummyPromptFS embed.FS

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	os.Setenv("SHUBH_DATA_DIR", t.TempDir())
	return NewServer(0, dummyWebFS, dummyPromptFS)
}

func TestHealthCheckEndpoint(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	srv.handleHealthCheck(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse health response: %v", err)
	}

	if res["status"] != "ok" {
		t.Fatalf("expected status 'ok', got %v", res["status"])
	}
}

func TestGetModeEndpoint(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/mode", nil)
	rr := httptest.NewRecorder()

	srv.handleGetMode(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse mode response: %v", err)
	}

	if res["mode"] == nil {
		t.Fatalf("expected mode field in response")
	}
}

func TestDomainRESTEndpoints(t *testing.T) {
	srv := setupTestServer(t)
	sess := auth.GetAuthManager().CreateGuestDemoSession()

	// Build handler mux wrapped in middleware
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/event", srv.handleGetEvent)
	mux.HandleFunc("GET /api/guests", srv.handleListGuests)
	mux.HandleFunc("GET /api/itinerary", srv.handleListItinerary)
	mux.HandleFunc("GET /api/designs", srv.handleListDesigns)
	mux.HandleFunc("POST /api/reset", srv.handleResetStore)

	handler := middleware.Chain(mux, middleware.CORSMiddleware)

	// Unauthenticated test should fail with 401
	reqUnauth := httptest.NewRequest(http.MethodGet, "/api/event", nil)
	rrUnauth := httptest.NewRecorder()
	handler.ServeHTTP(rrUnauth, reqUnauth)
	if rrUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated GET /api/event, got %d", rrUnauth.Code)
	}

	// 1. GET /api/event authenticated
	reqEvt := httptest.NewRequest(http.MethodGet, "/api/event", nil)
	reqEvt.Header.Set("X-Session-ID", sess.Token)
	rrEvt := httptest.NewRecorder()
	handler.ServeHTTP(rrEvt, reqEvt)
	if rrEvt.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /api/event, got %d", rrEvt.Code)
	}

	// 2. GET /api/guests authenticated
	reqGst := httptest.NewRequest(http.MethodGet, "/api/guests", nil)
	reqGst.Header.Set("X-Session-ID", sess.Token)
	rrGst := httptest.NewRecorder()
	handler.ServeHTTP(rrGst, reqGst)
	if rrGst.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /api/guests, got %d", rrGst.Code)
	}

	// 3. POST /api/reset
	reqReset := httptest.NewRequest(http.MethodPost, "/api/reset", nil)
	rrReset := httptest.NewRecorder()
	handler.ServeHTTP(rrReset, reqReset)
	if rrReset.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on POST /api/reset, got %d", rrReset.Code)
	}
}

func TestDeleteDesignEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	sess := auth.GetAuthManager().CreateGuestDemoSession()

	d1 := srv.store.AddDesign(store.InvitationDesign{Headline: "Test Design"})

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/designs", srv.handleDeleteDesign)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/designs?id=%s", d1.ID), nil)
	req.Header.Set("X-Session-ID", sess.Token)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on DELETE /api/designs, got %d", rr.Code)
	}

	if len(srv.store.ListDesigns()) != 0 {
		t.Fatalf("expected design to be deleted from store")
	}
}
