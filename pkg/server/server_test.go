package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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

	// Build handler mux wrapped in middleware
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/event", srv.handleGetEvent)
	mux.HandleFunc("GET /api/guests", srv.handleListGuests)
	mux.HandleFunc("GET /api/itinerary", srv.handleListItinerary)
	mux.HandleFunc("GET /api/designs", srv.handleListDesigns)
	mux.HandleFunc("POST /api/reset", srv.handleResetStore)

	handler := middleware.Chain(mux, middleware.CORSMiddleware)

	// 1. GET /api/event
	reqEvt := httptest.NewRequest(http.MethodGet, "/api/event", nil)
	rrEvt := httptest.NewRecorder()
	handler.ServeHTTP(rrEvt, reqEvt)
	if rrEvt.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /api/event, got %d", rrEvt.Code)
	}

	// 2. GET /api/guests
	reqGst := httptest.NewRequest(http.MethodGet, "/api/guests", nil)
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

	d1 := srv.store.AddDesign(store.InvitationDesign{Headline: "Test Design"})

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/designs", srv.handleDeleteDesign)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/designs?id=%s", d1.ID), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on DELETE /api/designs, got %d", rr.Code)
	}

	if len(srv.store.ListDesigns()) != 0 {
		t.Fatalf("expected design to be deleted from store")
	}
}
