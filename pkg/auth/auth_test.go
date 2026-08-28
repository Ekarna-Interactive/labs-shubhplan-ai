package auth

import (
	"testing"
	"time"
)

func TestSessionEviction(t *testing.T) {
	am := NewAuthManager(t.TempDir() + "/users_test.json")

	// 1. Manually insert expired session and active session
	expiredToken := "token_expired_123"
	activeToken := "token_active_456"

	am.mu.Lock()
	am.sessions[expiredToken] = Session{
		Token:     expiredToken,
		UserID:    "usr_test1",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}
	am.sessions[activeToken] = Session{
		Token:     activeToken,
		UserID:    "usr_test2",
		ExpiresAt: time.Now().Add(1 * time.Hour), // Valid for 1 hour
	}
	am.mu.Unlock()

	// 2. GetSession on expired token should return false and evict it from memory
	_, valid := am.GetSession(expiredToken)
	if valid {
		t.Fatalf("expected expired session to be invalid")
	}

	am.mu.RLock()
	_, existsInMap := am.sessions[expiredToken]
	am.mu.RUnlock()

	if existsInMap {
		t.Fatalf("expected expired session to be purged from am.sessions map")
	}

	// 3. CleanExpiredSessions should return number of cleaned sessions
	am.mu.Lock()
	am.sessions["token_expired_999"] = Session{
		Token:     "token_expired_999",
		UserID:    "usr_test3",
		ExpiresAt: time.Now().Add(-10 * time.Minute),
	}
	am.mu.Unlock()

	cleaned := am.CleanExpiredSessions()
	if cleaned != 1 {
		t.Fatalf("expected CleanExpiredSessions to purge 1 session, got %d", cleaned)
	}
}
