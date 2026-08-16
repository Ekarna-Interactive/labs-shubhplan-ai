package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type HonchoPeerCard struct {
	Name        string            `json:"name"`
	Role        string            `json:"role"`
	Status      string            `json:"status"`
	Headcount   int               `json:"headcount"`
	Dietary     string            `json:"dietary"`
	Cab         bool              `json:"cab"`
	Attributes  map[string]string `json:"attributes"`
	LastUpdated string            `json:"lastUpdated"`
}

type HonchoMemoryStore struct {
	AppID     string                    `json:"appId"`
	Peers     map[string]HonchoPeerCard `json:"peers"`
	SessionID string                    `json:"sessionId"`
	mu        sync.RWMutex
	filePath  string
	apiKey    string
}

var globalHonchoManager *HonchoMemoryStore
var onceHoncho sync.Once

func GetHonchoManager() *HonchoMemoryStore {
	onceHoncho.Do(func() {
		dataDir := os.Getenv("SHUBH_DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
		_ = os.MkdirAll(dataDir, 0755)

		storePath := filepath.Join(dataDir, "honcho_memory.json")
		apiKey := os.Getenv("HONCHO_API_KEY")

		store := &HonchoMemoryStore{
			AppID:     "shubh-plan-ai",
			Peers:     make(map[string]HonchoPeerCard),
			SessionID: fmt.Sprintf("session-%d", time.Now().Unix()),
			filePath:  storePath,
			apiKey:    apiKey,
		}

		store.loadLocal()
		globalHonchoManager = store
	})
	return globalHonchoManager
}

func (h *HonchoMemoryStore) GetHonchoStatusMessage() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.apiKey != "" {
		return "🟢 Connected to Honcho Cloud Memory (https://api.honcho.dev/v3/)"
	}
	return "🟡 Operating with Inbuilt Local Memory Store (./data/honcho_memory.json) — HONCHO_API_KEY is not set"
}

func (h *HonchoMemoryStore) loadLocal() {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := os.ReadFile(h.filePath)
	if err == nil {
		var temp HonchoMemoryStore
		if err := json.Unmarshal(data, &temp); err == nil && temp.Peers != nil {
			h.Peers = temp.Peers
			if temp.AppID != "" {
				h.AppID = temp.AppID
			}
		}
	}
}

func (h *HonchoMemoryStore) saveLocal() error {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.filePath, data, 0644)
}

func (h *HonchoMemoryStore) RecordPeerCard(name string, role string, headcount int, dietary string, cab bool) {
	h.mu.Lock()
	card := HonchoPeerCard{
		Name:        name,
		Role:        role,
		Status:      "Confirmed",
		Headcount:   headcount,
		Dietary:     dietary,
		Cab:         cab,
		Attributes:  make(map[string]string),
		LastUpdated: time.Now().Format(time.RFC3339),
	}
	h.Peers[name] = card
	_ = h.saveLocal()
	h.mu.Unlock()

	// Sync to Honcho v3 HTTP REST API if API Key is present
	if h.apiKey != "" {
		go h.syncPeerToHonchoREST(card)
	}
}

func (h *HonchoMemoryStore) GetPeerCards() map[string]HonchoPeerCard {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]HonchoPeerCard)
	for k, v := range h.Peers {
		result[k] = v
	}
	return result
}

func (h *HonchoMemoryStore) syncPeerToHonchoREST(card HonchoPeerCard) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Ensure Peer exists on Honcho v3 REST API
	peerURL := fmt.Sprintf("https://api.honcho.dev/v3/apps/%s/peers", h.AppID)
	peerPayload := map[string]interface{}{
		"id":    strings.ToLower(strings.ReplaceAll(card.Name, " ", "-")),
		"name":  card.Name,
		"meta":  map[string]interface{}{"role": card.Role, "dietary": card.Dietary, "headcount": card.Headcount},
	}
	bodyBytes, _ := json.Marshal(peerPayload)

	req, err := http.NewRequestWithContext(ctx, "POST", peerURL, bytes.NewBuffer(bodyBytes))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.apiKey)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_, _ = io.ReadAll(resp.Body)
			resp.Body.Close()
		}
	}
}
