package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type AgentStreamEvent struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Agent   string `json:"agent"`
}

type AgentClient struct {
	AgentServiceURL string
	SessionSecret   string
	HTTPClient      *http.Client
}

func NewAgentClient() *AgentClient {
	url := os.Getenv("AGENT_SERVICE_URL")
	if url == "" {
		url = "http://localhost:8082"
	}
	engine := GetAgentEngine()
	sess := engine.CreateHandshakeSession("go-client")
	return &AgentClient{
		AgentServiceURL: url,
		SessionSecret:   sess.SessionSecret,
		HTTPClient:      &http.Client{Timeout: 0},
	}
}

// StreamMessage wraps SendMessageStreams and pipes events into a Go channel
func (c *AgentClient) StreamMessage(sessionID string, prompt string, plannerName string, plannerRole string, eventID string, eventContext string, ch chan<- AgentStreamEvent) error {
	defer close(ch)
	return c.SendMessageStreams(sessionID, prompt, plannerName, plannerRole, eventID, eventContext, func(ev AgentStreamEvent) {
		ch <- ev
	})
}

// SendMessageStreams sends a user prompt to the Go ADK Orchestrator daemon and streams SSE chunks back.
func (c *AgentClient) SendMessageStreams(sessionID string, prompt string, plannerName string, plannerRole string, eventID string, eventContext string, eventHandler func(AgentStreamEvent)) error {
	endpoint := fmt.Sprintf("%s/api/v1/orchestrator/stream", c.AgentServiceURL)
	if plannerName == "" {
		if plannerRole != "" {
			plannerName = plannerRole
		} else {
			plannerName = "Event Planner"
		}
	}
	if eventID == "" {
		eventID = "evt-shubh-event"
	}
	payload := map[string]string{
		"sessionId":    sessionID,
		"message":      prompt,
		"plannerName":  plannerName,
		"plannerRole":  plannerRole,
		"eventId":      eventID,
		"eventContext": eventContext,
	}
	bodyData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.SessionSecret != "" {
		req.Header.Set("Authorization", "Bearer "+c.SessionSecret)
	} else if secret := os.Getenv("AGENT_SHARED_SECRET"); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	resp, err := c.HTTPClient.Do(req)

	// If initial port returned 404 or connection error, try alternate port (8082 <-> 8081)
	if err != nil || (resp != nil && resp.StatusCode == http.StatusNotFound) {
		if resp != nil {
			resp.Body.Close()
		}
		altURL := "http://localhost:8082"
		if strings.Contains(c.AgentServiceURL, "8082") {
			altURL = "http://localhost:8081"
		}
		altEndpoint := fmt.Sprintf("%s/api/v1/orchestrator/stream", altURL)
		altReq, altErr := http.NewRequest("POST", altEndpoint, bytes.NewBuffer(bodyData))
		if altErr == nil {
			altReq.Header.Set("Content-Type", "application/json")
			if c.SessionSecret != "" {
				altReq.Header.Set("Authorization", "Bearer "+c.SessionSecret)
			} else if secret := os.Getenv("AGENT_SHARED_SECRET"); secret != "" {
				altReq.Header.Set("Authorization", "Bearer "+secret)
			}
			altResp, doErr := c.HTTPClient.Do(altReq)
			if doErr == nil && altResp.StatusCode == http.StatusOK {
				resp = altResp
				err = nil
			}
		}
	}

	if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
		if resp != nil {
			resp.Body.Close()
		}
		// In-process fallback when standalone daemon on 8082/8081 is not running
		GetAgentEngine().StreamMultiAgentResponse(context.Background(), prompt, plannerName, eventContext, eventHandler)
		return nil
	}

	scanner := bufio.NewScanner(resp.Body)
	eventCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataJSON := strings.TrimPrefix(line, "data: ")
			var ev AgentStreamEvent
			if err := json.Unmarshal([]byte(dataJSON), &ev); err == nil {
				eventCount++
				eventHandler(ev)
			}
		}
	}

	if eventCount == 0 {
		eventHandler(AgentStreamEvent{
			Type:    "content",
			Agent:   "MasterOrchestrator",
			Content: fmt.Sprintf("👋 Hello %s! I am your AI Event Assistant. I've received your prompt: '%s'. How can I assist you with your active event today?", plannerName, prompt),
		})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
