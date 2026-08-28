package main

import (
	"testing"
)

func TestWebFS(t *testing.T) {
	// Verify embedded web filesystem variable is accessible
	entries, err := webFS.ReadDir("web")
	if err != nil {
		t.Fatalf("expected web embed.FS to read 'web' dir, got error: %v", err)
	}

	if len(entries) == 0 {
		t.Fatalf("expected embedded 'web' directory to contain static files")
	}
}
