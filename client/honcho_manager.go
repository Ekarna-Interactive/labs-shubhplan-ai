package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
		apiKey := strings.TrimSpace(os.Getenv("HONCHO_API_KEY"))

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

func (h *HonchoMemoryStore) SetAPIKey(apiKey string) {
	h.mu.Lock()
	if apiKey != "" {
		h.apiKey = apiKey
	}
	appID := h.AppID
	h.mu.Unlock()

	if apiKey != "" && appID != "" && appID != "shubh-plan-ai" {
		h.EnsureWorkspaceCreated(appID, appID)
	}
}

func (h *HonchoMemoryStore) EnsureWorkspaceCreated(workspaceID string, eventName string) {
	h.mu.Lock()
	if workspaceID != "" {
		h.AppID = workspaceID
	}
	apiKey := h.apiKey
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("HONCHO_API_KEY"))
	}
	h.mu.Unlock()

	if apiKey == "" || workspaceID == "" {
		log.Printf("⚠️ Honcho workspace creation skipped: apiKey is missing, workspaceID='%s'", workspaceID)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		createWsURL := "https://api.honcho.dev/v3/workspaces"
		wsPayload, _ := json.Marshal(map[string]interface{}{
			"id":   workspaceID,
			"name": eventName,
		})
		cReq, err := http.NewRequestWithContext(ctx, "POST", createWsURL, bytes.NewBuffer(wsPayload))
		if err == nil {
			cReq.Header.Set("Content-Type", "application/json")
			cReq.Header.Set("Authorization", "Bearer "+apiKey)
			if cResp, cErr := http.DefaultClient.Do(cReq); cErr == nil {
				body, _ := io.ReadAll(cResp.Body)
				cResp.Body.Close()
				log.Printf("🟢 Honcho workspace creation response [%d]: %s", cResp.StatusCode, string(body))
			} else {
				log.Printf("⚠️ Honcho workspace HTTP error: %v", cErr)
			}
		}
	}()
}

func (h *HonchoMemoryStore) GetHonchoStatusMessage() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	k := h.apiKey
	if k == "" {
		k = os.Getenv("HONCHO_API_KEY")
	}

	if k != "" {
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
	h.RecordTurnToHoncho("default-session", card.Name, card.Role, fmt.Sprintf("RSVP Confirmed: %d headcount, Dietary: %s, Cab: %v", card.Headcount, card.Dietary, card.Cab))
}

// EnsureSessionCreated registers a conversation session under the workspace on Honcho Cloud v3
func (h *HonchoMemoryStore) EnsureSessionCreated(sessionID string) {
	apiKey := h.apiKey
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("HONCHO_API_KEY"))
	}
	if apiKey == "" {
		return
	}

	workspaceID := "shubh-plan-open"
	if h.AppID != "" && h.AppID != "shubh-plan-ai" {
		workspaceID = h.AppID
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sessURL := fmt.Sprintf("https://api.honcho.dev/v3/workspaces/%s/sessions", workspaceID)
		sessPayload, _ := json.Marshal(map[string]interface{}{
			"id": sessionID,
		})

		req, err := http.NewRequestWithContext(ctx, "POST", sessURL, bytes.NewBuffer(sessPayload))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)
			if resp, err := http.DefaultClient.Do(req); err == nil {
				_, _ = io.ReadAll(resp.Body)
				resp.Body.Close()
			}
		}
	}()
}

// RecordTurnToHoncho implements the non-blocking Record loop from Honcho memory skill:
// Registers Peer, Session, and posts turn Message to api.honcho.dev/v3
func (h *HonchoMemoryStore) RecordTurnToHoncho(sessionID string, peerName string, peerRole string, content string) {
	apiKey := h.apiKey
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("HONCHO_API_KEY"))
	}
	if apiKey == "" || strings.TrimSpace(content) == "" {
		return
	}

	workspaceID := "shubh-plan-open"
	if h.AppID != "" && h.AppID != "shubh-plan-ai" {
		workspaceID = h.AppID
	}

	if sessionID == "" {
		sessionID = "session-general"
	}

	peerID := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(peerName), " ", "-"))
	if peerID == "" {
		peerID = "planner-user"
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()

		// 1. Ensure Workspace exists
		wsURL := "https://api.honcho.dev/v3/workspaces"
		wsPayload, _ := json.Marshal(map[string]interface{}{"id": workspaceID, "name": workspaceID})
		wsReq, err := http.NewRequestWithContext(ctx, "POST", wsURL, bytes.NewBuffer(wsPayload))
		if err == nil {
			wsReq.Header.Set("Content-Type", "application/json")
			wsReq.Header.Set("Authorization", "Bearer "+apiKey)
			if resp, err := http.DefaultClient.Do(wsReq); err == nil {
				resp.Body.Close()
			}
		}

		// 2. Ensure Peer exists
		peerURL := fmt.Sprintf("https://api.honcho.dev/v3/workspaces/%s/peers", workspaceID)
		peerPayload, _ := json.Marshal(map[string]interface{}{
			"id":   peerID,
			"name": peerName,
			"meta": map[string]interface{}{"role": peerRole},
		})
		peerReq, err := http.NewRequestWithContext(ctx, "POST", peerURL, bytes.NewBuffer(peerPayload))
		if err == nil {
			peerReq.Header.Set("Content-Type", "application/json")
			peerReq.Header.Set("Authorization", "Bearer "+apiKey)
			if resp, err := http.DefaultClient.Do(peerReq); err == nil {
				resp.Body.Close()
			}
		}

		// 3. Ensure Session exists
		sessURL := fmt.Sprintf("https://api.honcho.dev/v3/workspaces/%s/sessions", workspaceID)
		sessPayload, _ := json.Marshal(map[string]interface{}{"id": sessionID})
		sessReq, err := http.NewRequestWithContext(ctx, "POST", sessURL, bytes.NewBuffer(sessPayload))
		if err == nil {
			sessReq.Header.Set("Content-Type", "application/json")
			sessReq.Header.Set("Authorization", "Bearer "+apiKey)
			if resp, err := http.DefaultClient.Do(sessReq); err == nil {
				resp.Body.Close()
			}
		}

		// 4. Record Message turn to Session (Try Array format first, fallback to wrapper)
		msgURL := fmt.Sprintf("https://api.honcho.dev/v3/workspaces/%s/sessions/%s/messages", workspaceID, sessionID)
		
		// Payload Format 1: Array of message objects
		msgArrayPayload, _ := json.Marshal([]map[string]interface{}{
			{
				"peer_id": peerID,
				"content": content,
			},
		})
		msgReq, err := http.NewRequestWithContext(ctx, "POST", msgURL, bytes.NewBuffer(msgArrayPayload))
		if err == nil {
			msgReq.Header.Set("Content-Type", "application/json")
			msgReq.Header.Set("Authorization", "Bearer "+apiKey)
			if resp, err := http.DefaultClient.Do(msgReq); err == nil {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				log.Printf("🟢 Honcho message recorded [%d] for peer '%s' in session '%s': %s", resp.StatusCode, peerID, sessionID, string(body))
				
				// If status is 400/422, try Format 2: {"messages": [...]}
				if resp.StatusCode >= 400 {
					msgWrapperPayload, _ := json.Marshal(map[string]interface{}{
						"messages": []map[string]interface{}{
							{
								"peer_id": peerID,
								"content": content,
							},
						},
					})
					wReq, wErr := http.NewRequestWithContext(ctx, "POST", msgURL, bytes.NewBuffer(msgWrapperPayload))
					if wErr == nil {
						wReq.Header.Set("Content-Type", "application/json")
						wReq.Header.Set("Authorization", "Bearer "+apiKey)
						if wResp, wDoErr := http.DefaultClient.Do(wReq); wDoErr == nil {
							wBody, _ := io.ReadAll(wResp.Body)
							wResp.Body.Close()
							log.Printf("🟢 Honcho message wrapper retry [%d]: %s", wResp.StatusCode, string(wBody))
						}
					}
				}
			}
		}
	}()
}
