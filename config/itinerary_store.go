package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ItineraryItem represents a structured ceremony event or run-of-show slot
type ItineraryItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Date        string `json:"date"`
	Time        string `json:"time"`
	Location    string `json:"location"`
	DressCode   string `json:"dressCode"`
	Description string `json:"description"`
}

// ItineraryStore represents the root structure for data/itinerary.json
type ItineraryStore struct {
	UpdatedAt time.Time       `json:"updatedAt"`
	Itinerary []ItineraryItem `json:"itinerary"`
}

var itineraryMutex sync.Mutex

func getItineraryFilePath() string {
	dataDir := filepath.Join(".", "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		_ = os.MkdirAll(dataDir, 0755)
	}
	return filepath.Join(dataDir, "itinerary.json")
}

// LoadItinerary reads and returns structured itinerary items from data/itinerary.json
func LoadItinerary() ([]ItineraryItem, error) {
	itineraryMutex.Lock()
	defer itineraryMutex.Unlock()

	filePath := getItineraryFilePath()
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []ItineraryItem{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return []ItineraryItem{}, err
	}

	var store ItineraryStore
	if err := json.Unmarshal(data, &store); err != nil {
		return []ItineraryItem{}, err
	}

	return store.Itinerary, nil
}

// SaveItinerary persists structured itinerary items into data/itinerary.json
func SaveItinerary(items []ItineraryItem) error {
	itineraryMutex.Lock()
	defer itineraryMutex.Unlock()

	filePath := getItineraryFilePath()
	store := ItineraryStore{
		UpdatedAt: time.Now(),
		Itinerary: items,
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal itinerary store: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// AddItineraryItem appends or updates a ceremony item in data/itinerary.json
func AddItineraryItem(item ItineraryItem) ([]ItineraryItem, error) {
	items, _ := LoadItinerary()
	if item.ID == "" {
		item.ID = fmt.Sprintf("item-%d", time.Now().UnixNano())
	}

	updated := false
	for i, existing := range items {
		if existing.ID == item.ID {
			items[i] = item
			updated = true
			break
		}
	}

	if !updated {
		items = append(items, item)
	}

	if err := SaveItinerary(items); err != nil {
		return items, err
	}

	return items, nil
}

// DeleteItineraryItem removes a ceremony item by ID from data/itinerary.json
func DeleteItineraryItem(id string) ([]ItineraryItem, error) {
	items, _ := LoadItinerary()
	filtered := make([]ItineraryItem, 0, len(items))

	for _, item := range items {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}

	if err := SaveItinerary(filtered); err != nil {
		return items, err
	}

	return filtered, nil
}
