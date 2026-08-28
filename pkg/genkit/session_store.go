package genkitengine

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

// PruneOldSessionFiles removes file-backed session JSONs older than maxAge from target directory.
func PruneOldSessionFiles(dir string, maxAge time.Duration) int {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return 0
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	now := time.Now()
	pruned := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			fp := filepath.Join(dir, entry.Name())
			if err := os.Remove(fp); err == nil {
				pruned++
			}
		}
	}

	if pruned > 0 {
		log.Printf("[Genkit Session Store] Pruned %d stale session files from %s (older than %s)", pruned, dir, maxAge)
	}
	return pruned
}
