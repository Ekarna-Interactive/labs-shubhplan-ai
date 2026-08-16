package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const GuestProfileFilename = "guests.md"

// GuestRSVPRecord represents a guest RSVP entry stored in guests.md
type GuestRSVPRecord struct {
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Status    string `json:"status"`
	Headcount int    `json:"headcount"`
	Dietary   string `json:"dietary"`
	Cab       bool   `json:"cab"`
}

// SaveGuestRSVPToMarkdown appends or updates a guest record in guests.md
func SaveGuestRSVPToMarkdown(rec GuestRSVPRecord) error {
	records, _ := LoadGuestRSVPsFromMarkdown()

	found := false
	for i, r := range records {
		if (rec.Phone != "" && r.Phone == rec.Phone) || (rec.Name != "" && strings.EqualFold(r.Name, rec.Name)) {
			records[i] = rec
			found = true
			break
		}
	}
	if !found {
		records = append(records, rec)
	}

	totalAttending := 0
	totalDeclined := 0
	for _, r := range records {
		if strings.EqualFold(r.Status, "declined") {
			totalDeclined += r.Headcount
		} else {
			totalAttending += r.Headcount
		}
	}

	profile, _ := LoadEventProfile()
	totalGuests := 500
	if profile.EstimatedGuests > 0 {
		totalGuests = profile.EstimatedGuests
	}
	pending := totalGuests - (totalAttending + totalDeclined)
	if pending < 0 {
		pending = 0
	}

	var sb strings.Builder
	sb.WriteString("# 👥 Event Guest RSVPs\n\n")
	sb.WriteString(fmt.Sprintf("* **Total Recorded Guests**: %d\n", len(records)))
	sb.WriteString(fmt.Sprintf("* **Attending**: %d\n", totalAttending))
	sb.WriteString(fmt.Sprintf("* **Declined**: %d\n", totalDeclined))
	sb.WriteString(fmt.Sprintf("* **Pending**: %d\n\n", pending))
	sb.WriteString("## Guest Directory\n\n")
	sb.WriteString("| Guest Full Name | Phone Number | Status | Headcount | Dietary Preferences | Transport Request |\n")
	sb.WriteString("| --------------- | ------------ | ------ | --------- | ------------------- | ----------------- |\n")

	for _, r := range records {
		statusDisplay := "Confirmed"
		if strings.EqualFold(r.Status, "declined") {
			statusDisplay = "Declined"
		}
		dietDisplay := r.Dietary
		if dietDisplay == "" {
			dietDisplay = "None"
		}
		cabDisplay := "No (Self-Transport)"
		if r.Cab {
			cabDisplay = "Yes (Cab Needed)"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s |\n",
			r.Name, r.Phone, statusDisplay, r.Headcount, dietDisplay, cabDisplay))
	}

	filePath := filepath.Join(".", GuestProfileFilename)
	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}

// LoadGuestRSVPsFromMarkdown reads and parses guests.md into structured GuestRSVPRecords
func LoadGuestRSVPsFromMarkdown() ([]GuestRSVPRecord, error) {
	filePath := filepath.Join(".", GuestProfileFilename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var records []GuestRSVPRecord
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inTable := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "| Guest Full Name |") {
			inTable = true
			continue
		}
		if strings.HasPrefix(line, "| --------------- |") {
			continue
		}

		if inTable && strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 7 {
				name := strings.TrimSpace(parts[1])
				phone := strings.TrimSpace(parts[2])
				status := strings.TrimSpace(parts[3])
				headcountStr := strings.TrimSpace(parts[4])
				dietary := strings.TrimSpace(parts[5])
				cabStr := strings.TrimSpace(parts[6])

				headcount, _ := strconv.Atoi(headcountStr)
				if headcount <= 0 {
					headcount = 1
				}

				cab := strings.Contains(strings.ToLower(cabStr), "yes") || strings.Contains(strings.ToLower(cabStr), "cab")

				records = append(records, GuestRSVPRecord{
					Name:      name,
					Phone:     phone,
					Status:    status,
					Headcount: headcount,
					Dietary:   dietary,
					Cab:       cab,
				})
			}
		}
	}

	return records, nil
}
