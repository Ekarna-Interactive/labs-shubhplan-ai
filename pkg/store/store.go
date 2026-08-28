package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Helper function to parse time strings into minutes from midnight for sorting.
func parseTimeToMinutesGo(timeStr string) int {
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		return 0
	}
	t, err := time.Parse("03:04 PM", strings.ToUpper(timeStr))
	if err != nil {
		t2, err2 := time.Parse("3:04 PM", strings.ToUpper(timeStr))
		if err2 != nil {
			t3, err3 := time.Parse("15:04", timeStr)
			if err3 != nil {
				return 0
			}
			return t3.Hour()*60 + t3.Minute()
		}
		return t2.Hour()*60 + t2.Minute()
	}
	return t.Hour()*60 + t.Minute()
}

func getStoreFilePath() string {
	dir := strings.TrimSpace(os.Getenv("SHUBH_DATA_DIR"))
	if dir == "" {
		dir = "./data"
	}
	return filepath.Join(dir, "store.json")
}

// VenueDetails holds rich Google Maps place metadata.
type VenueDetails struct {
	PrimaryVenue           string `json:"primary_venue,omitempty"`
	VenueFormattedAddress  string `json:"venue_formatted_address,omitempty"`
	VenueAdrFormatAddress  string `json:"venue_adr_format_address,omitempty"`
	Address                string `json:"address,omitempty"`
	GoogleMapURL           string `json:"google_map_url,omitempty"`
	GoogleMapDirectionsURL string `json:"google_map_directions_url,omitempty"`
	VenuePhotoURL          string `json:"venue_photo_url,omitempty"`
	PlaceID                string `json:"place_id,omitempty"`
}

// EventProfile represents the core metadata for an event.
type EventProfile struct {
	ID               string       `json:"id"`
	Title            string       `json:"title"`
	EventType        string       `json:"eventType"`
	HostNames        string       `json:"hostNames"`
	Date             string       `json:"date"`
	Venue            string       `json:"venue"`
	Location         string       `json:"location"`
	VenueDetails     VenueDetails `json:"venueDetails"`
	AestheticTheme   string       `json:"aestheticTheme"`
	Description      string       `json:"description"`
	TargetGuestCount int          `json:"targetGuestCount"`
	UpdatedAt        time.Time    `json:"updatedAt"`
}

// Guest represents an attendee profile and RSVP record.
type Guest struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Category            string    `json:"category"`   // VIP, Family, Friend, Corporate
	RSVPStatus          string    `json:"rsvpStatus"` // Confirmed, Declined, Pending
	DietaryRequirements string    `json:"dietaryRequirements"`
	PlusOnes            int       `json:"plusOnes"`
	Phone               string    `json:"phone"`
	Notes               string    `json:"notes"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// ItineraryItem represents a scheduled event agenda item.
type ItineraryItem struct {
	ID          string `json:"id"`
	Time        string `json:"time"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Host        string `json:"host"`
}

// InvitationDesign represents a generated visual invitation card concept.
type InvitationDesign struct {
	ID             string    `json:"id"`
	Prompt         string    `json:"prompt"`
	StyleTheme     string    `json:"styleTheme"`
	PrimaryColor   string    `json:"primaryColor"`
	Typography     string    `json:"typography"`
	AspectRatio    string    `json:"aspectRatio"`
	CustomElements []string  `json:"customElements"`
	ImageURL       string    `json:"imageUrl"`
	Headline       string    `json:"headline"`
	Subhead        string    `json:"subhead"`
	CreatedAt      time.Time `json:"createdAt"`
}

// DataStore provides synchronized access to event management domain state with file-backed persistence.
type DataStore struct {
	mu        sync.RWMutex
	event     EventProfile
	guests    map[string]Guest
	itinerary []ItineraryItem
	designs   []InvitationDesign
}

var globalStore *DataStore
var once sync.Once

// GetStore returns the singleton DataStore instance auto-persisted to disk at ./data/store.json.
func GetStore() *DataStore {
	once.Do(func() {
		globalStore = &DataStore{
			event:     EventProfile{},
			guests:    make(map[string]Guest),
			itinerary: make([]ItineraryItem, 0),
			designs:   make([]InvitationDesign, 0),
		}
		globalStore.loadDisk()
	})
	return globalStore
}

// loadDisk reads persisted store state from store.json if present.
func (s *DataStore) loadDisk() {
	s.mu.Lock()
	defer s.mu.Unlock()

	targetPath := getStoreFilePath()
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return
	}

	var state struct {
		Event     EventProfile       `json:"event"`
		Guests    []Guest            `json:"guests"`
		Itinerary []ItineraryItem    `json:"itinerary"`
		Designs   []InvitationDesign `json:"designs"`
	}

	if err := json.Unmarshal(data, &state); err == nil {
		s.event = state.Event
		for _, g := range state.Guests {
			s.guests[g.ID] = g
		}
		s.itinerary = state.Itinerary
		s.designs = state.Designs
	}

	if s.event.Title == "" {
		s.event = EventProfile{
			ID:               "evt_default",
			Title:            "Aarav's Naming Ceremony",
			EventType:        "Naming Ceremony",
			HostNames:        "Surya & Ananya",
			Date:             "2026-10-12",
			Venue:            "Marhaba Mini Function Hall",
			Location:         "Vadapalani, Chennai",
			AestheticTheme:   "South Indian Traditional",
			Description:      "Traditional Naming Ceremony (Namkaran) celebration for baby Aarav.",
			TargetGuestCount: 150,
			VenueDetails: VenueDetails{
				PrimaryVenue:           "Marhaba Mini Function Hall",
				VenueFormattedAddress:  "60/2, 2nd St, Sarvamangala Colony, Aruna Colony, Vadapalani, Chennai, Tamil Nadu 600026",
				Address:                "Vadapalani, Chennai",
				GoogleMapURL:           "https://maps.google.com/?q=Marhaba+Mini+Function+Hall+Vadapalani+Chennai",
				GoogleMapDirectionsURL: "https://maps.google.com/maps?daddr=Marhaba+Mini+Function+Hall+Vadapalani+Chennai",
				VenuePhotoURL:          "https://images.unsplash.com/photo-1519167758481-83f550bb49b3?auto=format&fit=crop&w=800&q=80",
			},
		}
	}
}

// saveDisk flushes current store state to store.json on disk.
func (s *DataStore) saveDisk() {
	targetPath := getStoreFilePath()
	_ = os.MkdirAll(filepath.Dir(targetPath), 0755)

	payload := map[string]interface{}{
		"event":     s.event,
		"guests":    s.ListGuests(),
		"itinerary": s.itinerary,
		"designs":   s.designs,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err == nil {
		_ = os.WriteFile(targetPath, data, 0644)
	}
}

// ClearStore resets all event data, guests, itinerary, and designs on disk and in memory.
func (s *DataStore) ClearStore() {
	s.mu.Lock()
	s.event = EventProfile{}
	s.guests = make(map[string]Guest)
	s.itinerary = make([]ItineraryItem, 0)
	s.designs = make([]InvitationDesign, 0)
	s.mu.Unlock()

	s.saveDisk()
}

// IsConfigured returns true if the event profile has a non-empty title and date.
func (s *DataStore) IsConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.event.Title != "" && s.event.Date != ""
}

// GetEvent returns a copy of the current EventProfile.
func (s *DataStore) GetEvent() EventProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.event
}

// UpdateEvent updates the EventProfile fields, generates an ID if unconfigured, and auto-flushes to disk.
func (s *DataStore) UpdateEvent(profile EventProfile) EventProfile {
	s.mu.Lock()

	if s.event.ID == "" {
		s.event.ID = fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	if profile.Title != "" {
		s.event.Title = profile.Title
	}
	if profile.EventType != "" {
		s.event.EventType = profile.EventType
	}
	if profile.HostNames != "" {
		s.event.HostNames = profile.HostNames
	}
	if profile.Date != "" {
		s.event.Date = profile.Date
	}
	if profile.Venue != "" {
		s.event.Venue = profile.Venue
	}
	if profile.Location != "" {
		s.event.Location = profile.Location
	}
	if profile.VenueDetails.PrimaryVenue != "" || profile.VenueDetails.PlaceID != "" {
		s.event.VenueDetails = profile.VenueDetails
	}
	if profile.AestheticTheme != "" {
		s.event.AestheticTheme = profile.AestheticTheme
	}
	if profile.Description != "" {
		s.event.Description = profile.Description
	}
	if profile.TargetGuestCount > 0 {
		s.event.TargetGuestCount = profile.TargetGuestCount
	}
	s.event.UpdatedAt = time.Now()
	res := s.event
	s.mu.Unlock()

	s.saveDisk()
	return res
}

// ListGuests returns all guests in the store.
func (s *DataStore) ListGuests() []Guest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Guest, 0, len(s.guests))
	for _, g := range s.guests {
		list = append(list, g)
	}
	return list
}

// AddOrUpdateGuest adds or mutates a guest entry and auto-flushes to disk.
func (s *DataStore) AddOrUpdateGuest(g Guest) Guest {
	s.mu.Lock()

	if g.ID == "" {
		g.ID = fmt.Sprintf("gst_%d", time.Now().UnixNano())
	}
	if g.RSVPStatus == "" {
		g.RSVPStatus = "Pending"
	}
	g.UpdatedAt = time.Now()
	s.guests[g.ID] = g
	res := g
	s.mu.Unlock()

	s.saveDisk()
	return res
}

func (s *DataStore) DeleteGuest(id string) bool {
	s.mu.Lock()
	_, exists := s.guests[id]
	if exists {
		delete(s.guests, id)
	}
	s.mu.Unlock()
	if exists {
		s.saveDisk()
	}
	return exists
}

func (s *DataStore) ToggleGuestRSVP(id string) (Guest, bool) {
	s.mu.Lock()
	g, exists := s.guests[id]
	if exists {
		if strings.EqualFold(g.RSVPStatus, "Confirmed") {
			g.RSVPStatus = "Pending"
		} else if strings.EqualFold(g.RSVPStatus, "Pending") {
			g.RSVPStatus = "Declined"
		} else {
			g.RSVPStatus = "Confirmed"
		}
		g.UpdatedAt = time.Now()
		s.guests[id] = g
	}
	s.mu.Unlock()

	if exists {
		s.saveDisk()
	}
	return g, exists
}

// ListItinerary returns the scheduled itinerary items sorted chronologically by time.
func (s *DataStore) ListItinerary() []ItineraryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := append([]ItineraryItem(nil), s.itinerary...)
	sort.Slice(items, func(i, j int) bool {
		return parseTimeToMinutesGo(items[i].Time) < parseTimeToMinutesGo(items[j].Time)
	})
	return items
}

// AddItineraryItem appends an item to the schedule and auto-flushes to disk.
func (s *DataStore) AddItineraryItem(item ItineraryItem) ItineraryItem {
	s.mu.Lock()

	if item.ID == "" {
		item.ID = fmt.Sprintf("itin_%d", time.Now().UnixNano())
	}
	s.itinerary = append(s.itinerary, item)
	res := item
	s.mu.Unlock()

	s.saveDisk()
	return res
}

// ListDesigns returns generated design options.
func (s *DataStore) ListDesigns() []InvitationDesign {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]InvitationDesign(nil), s.designs...)
}

// AddDesign saves a new design concept and auto-flushes to disk.
func (s *DataStore) AddDesign(design InvitationDesign) InvitationDesign {
	s.mu.Lock()

	if design.ID == "" {
		design.ID = fmt.Sprintf("des_%d", time.Now().UnixNano())
	}
	design.CreatedAt = time.Now()
	s.designs = append([]InvitationDesign{design}, s.designs...)
	if len(s.designs) > 50 {
		s.designs = s.designs[:50]
	}
	res := design
	s.mu.Unlock()

	s.saveDisk()
	return res
}

// DeleteDesign removes a design concept by ID and auto-flushes to disk.
func (s *DataStore) DeleteDesign(id string) bool {
	s.mu.Lock()
	found := false
	var updated []InvitationDesign
	for _, d := range s.designs {
		if d.ID == id {
			found = true
			continue
		}
		updated = append(updated, d)
	}
	if found {
		s.designs = updated
	}
	s.mu.Unlock()

	if found {
		s.saveDisk()
	}
	return found
}

// ExportJSON returns the entire store state as formatted JSON bytes.
func (s *DataStore) ExportJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	payload := map[string]interface{}{
		"event":     s.event,
		"guests":    s.ListGuests(),
		"itinerary": s.itinerary,
		"designs":   s.designs,
	}
	return json.MarshalIndent(payload, "", "  ")
}
