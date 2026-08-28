package genkitengine

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneOldSessionFiles(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Create a fresh session file (modified now)
	freshFile := filepath.Join(tempDir, "session_fresh.json")
	if err := os.WriteFile(freshFile, []byte(`{"session":"active"}`), 0644); err != nil {
		t.Fatalf("failed to write fresh session file: %v", err)
	}

	// 2. Create an old session file (modified 10 days ago)
	oldFile := filepath.Join(tempDir, "session_old.json")
	if err := os.WriteFile(oldFile, []byte(`{"session":"stale"}`), 0644); err != nil {
		t.Fatalf("failed to write old session file: %v", err)
	}
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set modtime on old file: %v", err)
	}

	// 3. Run PruneOldSessionFiles with 7 day maxAge
	pruned := PruneOldSessionFiles(tempDir, 7*24*time.Hour)
	if pruned != 1 {
		t.Fatalf("expected 1 file pruned, got %d", pruned)
	}

	// 4. Verify fresh file exists and old file is removed
	if _, err := os.Stat(freshFile); os.IsNotExist(err) {
		t.Fatalf("expected fresh session file to still exist")
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatalf("expected old session file to be pruned")
	}
}
