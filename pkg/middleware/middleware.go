package middleware

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"shubh-plan-web/pkg/auth"
)

// responseWriterInterceptor captures HTTP status code for logging and satisfies http.Flusher for SSE streaming.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterInterceptor) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterInterceptor) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// ContextKey type for type-safe context values.
type ContextKey string

const (
	UserIDKey        ContextKey = "userID"
	UserEmailKey     ContextKey = "userEmail"
	UserRoleKey      ContextKey = "userRole"
	UserAPIKeyKey    ContextKey = "userAPIKey"
	UserMapsAPIKeyKey ContextKey = "userMapsAPIKey"
)

// CORSMiddleware enables CORS for browser-based API clients and local development.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Session-ID, X-Gemini-API-Key, X-Google-Maps-API-Key, X-Google-Places-API-Key")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs request duration, status code, and HTTP verb.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		interceptor := &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(interceptor, r)

		duration := time.Since(start)
		log.Printf("[HTTP] %s %s %d %v", r.Method, r.URL.Path, interceptor.statusCode, duration)
	})
}

// RecoveryMiddleware catches panics in HTTP handlers to prevent server crashes.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[RECOVERY] Panic caught: %v\nStack Trace:\n%s", err, string(debug.Stack()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"Internal Server Error","message":"An unexpected error occurred."}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware inspects session tokens from shubh_session cookie or X-Session-ID header and enriches context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if cookie, err := r.Cookie("shubh_session"); err == nil {
			token = cookie.Value
		}
		if token == "" {
			token = r.Header.Get("X-Session-ID")
		}

		ctx := r.Context()
		am := auth.GetAuthManager()
		if session, ok := am.GetSession(token); ok {
			ctx = context.WithValue(ctx, UserIDKey, session.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, session.UserEmail)
			ctx = context.WithValue(ctx, UserRoleKey, string(session.UserRole))
		}

		clientAPIKey := strings.TrimSpace(r.Header.Get("X-Gemini-API-Key"))
		if clientAPIKey == "" {
			clientAPIKey = strings.TrimSpace(r.URL.Query().Get("apiKey"))
		}
		if clientAPIKey != "" {
			ctx = context.WithValue(ctx, UserAPIKeyKey, clientAPIKey)
		}

		clientMapsKey := strings.TrimSpace(r.Header.Get("X-Google-Maps-API-Key"))
		if clientMapsKey == "" {
			clientMapsKey = strings.TrimSpace(r.Header.Get("X-Google-Places-API-Key"))
		}
		if clientMapsKey == "" {
			clientMapsKey = strings.TrimSpace(r.URL.Query().Get("mapsKey"))
		}
		if clientMapsKey != "" {
			ctx = context.WithValue(ctx, UserMapsAPIKeyKey, clientMapsKey)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Chain applies multiple middleware handlers in order.
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
