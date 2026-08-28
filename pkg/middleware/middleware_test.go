package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// OPTIONS request check
	reqOpt := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	rrOpt := httptest.NewRecorder()
	handler.ServeHTTP(rrOpt, reqOpt)

	if rrOpt.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 No Content for OPTIONS request, got %d", rrOpt.Code)
	}

	if rrOpt.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected CORS origin header '*'")
	}

	// GET request check
	reqGet := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rrGet := httptest.NewRecorder()
	handler.ServeHTTP(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK for GET request, got %d", rrGet.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	panickingHandler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went critically wrong!")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/panic", nil)
	rr := httptest.NewRecorder()

	panickingHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 Internal Server Error when panic occurs, got %d", rr.Code)
	}
}

func TestAuthMiddlewareAPIKeyExtraction(t *testing.T) {
	var extractedKey string

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if val, ok := r.Context().Value(UserAPIKeyKey).(string); ok {
			extractedKey = val
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Gemini-API-Key", "test_gemini_key_123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if extractedKey != "test_gemini_key_123" {
		t.Fatalf("expected extracted API key 'test_gemini_key_123', got '%s'", extractedKey)
	}
}

func TestMiddlewareChain(t *testing.T) {
	callOrder := []string{}

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "m1_start")
			next.ServeHTTP(w, r)
			callOrder = append(callOrder, "m1_end")
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "m2_start")
			next.ServeHTTP(w, r)
			callOrder = append(callOrder, "m2_end")
		})
	}

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chained := Chain(finalHandler, m1, m2)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	expectedOrder := []string{"m1_start", "m2_start", "handler", "m2_end", "m1_end"}
	if len(callOrder) != len(expectedOrder) {
		t.Fatalf("expected order len %d, got %d", len(expectedOrder), len(callOrder))
	}
	for i, v := range expectedOrder {
		if callOrder[i] != v {
			t.Fatalf("expected callOrder[%d] to be %s, got %s", i, v, callOrder[i])
		}
	}
}
