package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type EventOverview struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	EventType string `json:"eventType"`
	Date      string `json:"date"`
	Venue     string `json:"venue"`
	Status    string `json:"status"`
}

type SubEventItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Date      string `json:"date"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Venue     string `json:"venue"`
	DressCode string `json:"dressCode"`
}

type BudgetSummary struct {
	TotalEstimated float64            `json:"totalEstimated"`
	TotalActual    float64            `json:"totalActual"`
	Categories     []BudgetCategory   `json:"categories"`
}

type BudgetCategory struct {
	Name      string  `json:"name"`
	Estimated float64 `json:"estimated"`
	Actual    float64 `json:"actual"`
}

type RSVPOverview struct {
	TotalGuests int            `json:"totalGuests"`
	Attending   int            `json:"attending"`
	Declined    int            `json:"declined"`
	Pending     int            `json:"pending"`
	DietaryReqs map[string]int `json:"dietaryReqs"`
}

type EventClient struct {
	BackendURL string
	HTTPClient *http.Client
}

func NewEventClient() *EventClient {
	url := os.Getenv("BACKEND_URL")
	if url == "" {
		url = "http://localhost:8080"
	}
	return &EventClient{
		BackendURL: url,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *EventClient) FetchEventOverview(eventID string) (EventOverview, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/events/%s", c.BackendURL, eventID))
	if err != nil {
		return EventOverview{}, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return EventOverview{}, fmt.Errorf("HTTP status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var res EventOverview
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return EventOverview{}, err
	}
	return res, nil
}

func (c *EventClient) FetchItinerary(eventID string) ([]SubEventItem, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/events/%s/itinerary", c.BackendURL, eventID))
	if err != nil {
		return []SubEventItem{}, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return []SubEventItem{}, fmt.Errorf("HTTP status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var items []SubEventItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return []SubEventItem{}, err
	}
	return items, nil
}

func (c *EventClient) FetchBudgetSummary(eventID string) (BudgetSummary, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/events/%s/budget", c.BackendURL, eventID))
	if err != nil {
		return BudgetSummary{}, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return BudgetSummary{}, fmt.Errorf("HTTP status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var summary BudgetSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return BudgetSummary{}, err
	}
	return summary, nil
}

func (c *EventClient) FetchRSVPOverview(eventID string) (RSVPOverview, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/events/%s/rsvps", c.BackendURL, eventID))
	if err != nil {
		return RSVPOverview{}, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return RSVPOverview{}, fmt.Errorf("HTTP status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var rsvp RSVPOverview
	if err := json.NewDecoder(resp.Body).Decode(&rsvp); err != nil {
		return RSVPOverview{}, err
	}
	return rsvp, nil
}

func (c *EventClient) FetchHonchoCards(workspaceID string) (map[string]string, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/honcho/cards?workspaceId=%s", c.BackendURL, workspaceID))
	if err != nil {
		return map[string]string{}, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return map[string]string{}, fmt.Errorf("HTTP status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var res struct {
		Cards map[string]string `json:"cards"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return map[string]string{}, err
	}
	return res.Cards, nil
}
