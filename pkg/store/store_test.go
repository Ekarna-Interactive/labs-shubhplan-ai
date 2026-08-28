package store

import (
	"fmt"
	"os"
	"testing"
)

func setupTestStore(t *testing.T) *DataStore {
	tmpDir := t.TempDir()
	os.Setenv("SHUBH_DATA_DIR", tmpDir)
	s := &DataStore{
		guests:    make(map[string]Guest),
		itinerary: make([]ItineraryItem, 0),
		designs:   make([]InvitationDesign, 0),
	}
	s.loadDisk()
	return s
}

func TestEventProfile(t *testing.T) {
	s := setupTestStore(t)

	if s.IsConfigured() {
		t.Fatalf("expected store to be unconfigured initially")
	}

	updated := s.UpdateEvent(EventProfile{
		Title: "Aarav & Diya Wedding",
		Date:  "2026-11-15",
		Venue: "Royal Palace",
	})

	if !s.IsConfigured() {
		t.Fatalf("expected store to be configured after setting title and date")
	}

	evt := s.GetEvent()
	if evt.Title != "Aarav & Diya Wedding" || evt.Venue != "Royal Palace" {
		t.Fatalf("unexpected event profile data: %+v", evt)
	}

	if updated.ID == "" {
		t.Fatalf("expected auto-generated event ID")
	}
}

func TestGuestRoster(t *testing.T) {
	s := setupTestStore(t)

	g1 := s.AddOrUpdateGuest(Guest{Name: "Rohan Sharma", Category: "Family"})
	if g1.ID == "" || g1.RSVPStatus != "Pending" {
		t.Fatalf("unexpected guest defaults: %+v", g1)
	}

	guests := s.ListGuests()
	if len(guests) != 1 {
		t.Fatalf("expected 1 guest, got %d", len(guests))
	}

	toggled, ok := s.ToggleGuestRSVP(g1.ID)
	if !ok || toggled.RSVPStatus != "Declined" {
		t.Fatalf("expected RSVP to toggle from Pending to Declined, got %s", toggled.RSVPStatus)
	}

	toggled2, ok := s.ToggleGuestRSVP(g1.ID)
	if !ok || toggled2.RSVPStatus != "Confirmed" {
		t.Fatalf("expected RSVP to toggle from Declined to Confirmed, got %s", toggled2.RSVPStatus)
	}

	deleted := s.DeleteGuest(g1.ID)
	if !deleted || len(s.ListGuests()) != 0 {
		t.Fatalf("expected guest deletion to succeed")
	}
}

func TestItinerarySorting(t *testing.T) {
	s := setupTestStore(t)

	s.AddItineraryItem(ItineraryItem{Time: "07:00 PM", Title: "Reception"})
	s.AddItineraryItem(ItineraryItem{Time: "10:00 AM", Title: "Haldi Ceremony"})
	s.AddItineraryItem(ItineraryItem{Time: "04:00 PM", Title: "Sangeet Night"})

	items := s.ListItinerary()
	if len(items) != 3 {
		t.Fatalf("expected 3 itinerary items, got %d", len(items))
	}

	// Chronological check: 10:00 AM -> 04:00 PM -> 07:00 PM
	if items[0].Title != "Haldi Ceremony" || items[1].Title != "Sangeet Night" || items[2].Title != "Reception" {
		t.Fatalf("expected chronological sorting, got: %+v", items)
	}
}

func TestDesignConceptCap(t *testing.T) {
	s := setupTestStore(t)

	for i := 1; i <= 60; i++ {
		s.AddDesign(InvitationDesign{
			Headline: fmt.Sprintf("Concept %d", i),
		})
	}

	designs := s.ListDesigns()
	if len(designs) != 50 {
		t.Fatalf("expected designs slice capped at 50, got %d", len(designs))
	}

	// The newest design (Concept 60) should be at the front
	if designs[0].Headline != "Concept 60" {
		t.Fatalf("expected newest design at index 0, got %s", designs[0].Headline)
	}
}

func TestClearStore(t *testing.T) {
	s := setupTestStore(t)

	s.UpdateEvent(EventProfile{Title: "Party", Date: "2026-12-01"})
	s.AddOrUpdateGuest(Guest{Name: "Guest 1"})
	s.ClearStore()

	if s.IsConfigured() || len(s.ListGuests()) != 0 {
		t.Fatalf("expected store to be completely reset")
	}
}

func TestDeleteDesign(t *testing.T) {
	s := setupTestStore(t)

	d1 := s.AddDesign(InvitationDesign{Headline: "Royal Gold Invitation"})
	if d1.ID == "" {
		t.Fatalf("expected auto-generated design ID")
	}

	designs := s.ListDesigns()
	if len(designs) != 1 {
		t.Fatalf("expected 1 design concept, got %d", len(designs))
	}

	ok := s.DeleteDesign(d1.ID)
	if !ok {
		t.Fatalf("expected DeleteDesign to return true")
	}

	if len(s.ListDesigns()) != 0 {
		t.Fatalf("expected 0 design concepts after deletion")
	}

	ok2 := s.DeleteDesign("non_existent_id")
	if ok2 {
		t.Fatalf("expected DeleteDesign for invalid ID to return false")
	}
}
