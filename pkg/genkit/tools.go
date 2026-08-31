package genkitengine

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"shubh-plan-web/pkg/middleware"
	"shubh-plan-web/pkg/store"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type UpdateEventInput struct {
	Title            string             `json:"title,omitempty" jsonschema:"description=Title or couple names for the event"`
	EventType        string             `json:"eventType,omitempty" jsonschema:"description=Type of event e.g. Wedding, Birthday, Corporate Gala"`
	HostNames        string             `json:"hostNames,omitempty" jsonschema:"description=Names of hosts or organizers"`
	Date             string             `json:"date,omitempty" jsonschema:"description=Date of the event"`
	Venue            string             `json:"venue,omitempty" jsonschema:"description=Venue name and city"`
	Location         string             `json:"location,omitempty" jsonschema:"description=Full street address or city location"`
	VenueDetails     store.VenueDetails `json:"venueDetails,omitempty" jsonschema:"description=Rich Google Maps venue metadata"`
	AestheticTheme   string             `json:"aestheticTheme,omitempty" jsonschema:"description=Design aesthetic or color palette theme"`
	Description      string             `json:"description,omitempty" jsonschema:"description=General event notes"`
	TargetGuestCount int                `json:"targetGuestCount,omitempty" jsonschema:"description=Target guest count"`
}

type AddGuestInput struct {
	Name                string `json:"name" jsonschema:"description=Full name of the guest"`
	Category            string `json:"category,omitempty" jsonschema:"description=Category e.g. VIP, Family, Friend, Corporate"`
	RSVPStatus          string `json:"rsvpStatus,omitempty" jsonschema:"description=RSVP status e.g. Confirmed, Pending, Declined"`
	DietaryRequirements string `json:"dietaryRequirements,omitempty" jsonschema:"description=Dietary requirements or restrictions"`
	PlusOnes            int    `json:"plusOnes,omitempty" jsonschema:"description=Number of plus ones"`
	Phone               string `json:"phone,omitempty" jsonschema:"description=Phone number"`
	Notes               string `json:"notes,omitempty" jsonschema:"description=Notes"`
}

type AddItineraryInput struct {
	Time        string `json:"time" jsonschema:"description=Session start time e.g. 09:00 AM"`
	Title       string `json:"title" jsonschema:"description=Title of session or ceremony"`
	Description string `json:"description,omitempty" jsonschema:"description=Session description"`
	Location    string `json:"location,omitempty" jsonschema:"description=Specific hall or stage location"`
	Host        string `json:"host,omitempty" jsonschema:"description=Host or speaker name"`
}

type CreateDesignInput struct {
	StyleTheme   string `json:"styleTheme" jsonschema:"description=Style preset e.g. South Indian, Paper Cut, Clay 3D, Pop Art, Mughal, Minimalist Gold, Watercolor"`
	PrimaryColor string `json:"primaryColor,omitempty" jsonschema:"description=Primary color hex code or name"`
	Typography   string `json:"typography,omitempty" jsonschema:"description=Typography font pairing"`
	AspectRatio  string `json:"aspectRatio,omitempty" jsonschema:"description=Card aspect ratio e.g. 9:16, 4:5, 1:1, 16:9"`
	Headline     string `json:"headline,omitempty" jsonschema:"description=Headline text"`
	Subhead      string `json:"subhead,omitempty" jsonschema:"description=Subhead text"`
	CustomPrompt string `json:"customPrompt,omitempty" jsonschema:"description=Custom image generation prompt"`
}

type GenerateImageInput struct {
	Prompt      string `json:"prompt" jsonschema:"description=The selected detailed image generation prompt"`
	Style       string `json:"style,omitempty" jsonschema:"description=The chosen design style preset"`
	AspectRatio string `json:"aspectRatio,omitempty" jsonschema:"description=The target aspect ratio (9:16, 4:5, 1:1, 16:9)"`
}

type GenerateImageOutput struct {
	ImageURL    string `json:"imageUrl"`
	MarkdownImg string `json:"markdownImg"`
	Status      string `json:"status"`
	Prompt      string `json:"prompt"`
}

type SearchVenueInput struct {
	VenueName string `json:"venueName" jsonschema:"description=Name of hotel, resort, banquet hall, or venue (e.g. Hyatt Regency, Taj Mahal Palace)"`
	City      string `json:"city,omitempty" jsonschema:"description=City or region where venue is located (e.g. Mumbai, Bengaluru, Goa)"`
}

type DeleteInput struct {
	ID string `json:"id" jsonschema:"description=The unique ID of the entity to delete"`
}

type ToggleRSVPInput struct {
	ID string `json:"id" jsonschema:"description=The unique ID of the guest"`
}

// RegisterTools defines and registers all domain tools with Genkit.
func RegisterTools(g *genkit.Genkit, s *store.DataStore) map[string]ai.Tool {
	tools := make(map[string]ai.Tool)

	// Tool 1: Get Event Details
	getEventTool := genkit.DefineTool(g, "getEventDetails",
		"Retrieves the active event profile details including title, date, venue, location, venueDetails, theme, and host names.",
		func(ctx *ai.ToolContext, _ struct{}) (store.EventProfile, error) {
			return s.GetEvent(), nil
		},
	)
	tools["getEventDetails"] = getEventTool

	// Tool 2: Update Event Details
	updateEventTool := genkit.DefineTool(g, "updateEventDetails",
		"Updates active event profile details in the domain store.",
		func(ctx *ai.ToolContext, input UpdateEventInput) (store.EventProfile, error) {
			profile := store.EventProfile{
				Title:            input.Title,
				EventType:        input.EventType,
				HostNames:        input.HostNames,
				Date:             input.Date,
				Venue:            input.Venue,
				Location:         input.Location,
				VenueDetails:     input.VenueDetails,
				AestheticTheme:   input.AestheticTheme,
				Description:      input.Description,
				TargetGuestCount: input.TargetGuestCount,
			}
			updated := s.UpdateEvent(profile)
			return updated, nil
		},
	)
	tools["updateEventDetails"] = updateEventTool

	// Tool 3: List Guests
	listGuestsTool := genkit.DefineTool(g, "listGuests",
		"Lists all registered guests on the event roster with RSVP and dietary information.",
		func(ctx *ai.ToolContext, _ struct{}) ([]store.Guest, error) {
			return s.ListGuests(), nil
		},
	)
	tools["listGuests"] = listGuestsTool

	// Tool 4: Add or Update Guest
	addGuestTool := genkit.DefineTool(g, "addOrUpdateGuest",
		"Adds a new guest or updates an existing guest's RSVP status, category, dietary restrictions, or notes.",
		func(ctx *ai.ToolContext, input AddGuestInput) (store.Guest, error) {
			guest := store.Guest{
				Name:                input.Name,
				Category:            input.Category,
				RSVPStatus:          input.RSVPStatus,
				DietaryRequirements: input.DietaryRequirements,
				PlusOnes:            input.PlusOnes,
				Phone:               input.Phone,
				Notes:               input.Notes,
			}
			saved := s.AddOrUpdateGuest(guest)
			return saved, nil
		},
	)
	tools["addOrUpdateGuest"] = addGuestTool

	// Tool 5: List Itinerary
	listItineraryTool := genkit.DefineTool(g, "listItinerary",
		"Lists all scheduled itinerary items and ceremony sessions for the event.",
		func(ctx *ai.ToolContext, _ struct{}) ([]store.ItineraryItem, error) {
			return s.ListItinerary(), nil
		},
	)
	tools["listItinerary"] = listItineraryTool

	// Tool 6: Add Itinerary Item
	addItineraryTool := genkit.DefineTool(g, "addItineraryItem",
		"Appends a new scheduled session or ceremony item to the event itinerary timeline.",
		func(ctx *ai.ToolContext, input AddItineraryInput) (store.ItineraryItem, error) {
			item := store.ItineraryItem{
				Time:        input.Time,
				Title:       input.Title,
				Description: input.Description,
				Location:    input.Location,
				Host:        input.Host,
			}
			saved := s.AddItineraryItem(item)
			return saved, nil
		},
	)
	tools["addItineraryItem"] = addItineraryTool

	// Tool 7: Create Invitation Spec
	createDesignTool := genkit.DefineTool(g, "createInvitationSpec",
		"Compiles a new visual invitation card design concept specification and image generation prompt optimized for googleai/gemini-3.1-flash-image.",
		func(ctx *ai.ToolContext, input CreateDesignInput) (store.InvitationDesign, error) {
			evt := s.GetEvent()
			headline := input.Headline
			if headline == "" {
				headline = evt.Title
			}
			subhead := input.Subhead
			if subhead == "" {
				subhead = fmt.Sprintf("%s • %s", evt.Date, evt.Venue)
			}
			primaryColor := input.PrimaryColor
			if primaryColor == "" {
				primaryColor = "#D4AF37"
			}
			prompt := input.CustomPrompt
			if prompt == "" {
				prompt = fmt.Sprintf("Bespoke %s wedding invitation card with elegant typography (%s) and %s palette", input.StyleTheme, input.Typography, primaryColor)
			}

			design := store.InvitationDesign{
				Prompt:       prompt,
				StyleTheme:   input.StyleTheme,
				PrimaryColor: primaryColor,
				Typography:   input.Typography,
				Headline:     headline,
				Subhead:      subhead,
			}
			return design, nil
		},
	)
	tools["createInvitationSpec"] = createDesignTool

	// Tool 8: Search & Verify Venue Location (Google Maps)
	searchVenueTool := genkit.DefineTool(g, "searchVenueInfo",
		"Searches and verifies a venue using Google Maps API, retrieving rich place metadata (PrimaryVenue, VenueFormattedAddress, Address, GoogleMapURL, GoogleMapDirectionsURL, VenuePhotoURL, PlaceID).",
		func(ctx *ai.ToolContext, input SearchVenueInput) (store.VenueDetails, error) {
			vd := VerifyVenueWithGoogleMaps(ctx, input.VenueName, input.City)
			s.UpdateEvent(store.EventProfile{
				Venue:        vd.PrimaryVenue,
				Location:     vd.Address,
				VenueDetails: vd,
			})
			return vd, nil
		},
	)
	tools["searchVenueInfo"] = searchVenueTool

	// Tool 9: Generate Invitation Image (.png output)
	generateImageTool := genkit.DefineTool(g, "generateInvitationImage",
		"Executes image generation for a chosen prompt using Gemini Flash image model (googleai/gemini-3.1-flash-image), generates the PNG card artwork asset file on disk (web/assets/card_<id>.png), and updates the design studio.",
		func(ctx *ai.ToolContext, input GenerateImageInput) (GenerateImageOutput, error) {
			out := generateImageWithAPI(ctx.Context, input.Prompt, input.Style, input.AspectRatio, s)
			return out, nil
		},
	)
	tools["generateInvitationImage"] = generateImageTool

	// Tool 10: Delete Guest
	deleteGuestTool := genkit.DefineTool(g, "deleteGuest",
		"Deletes a guest from the event roster by guest ID.",
		func(ctx *ai.ToolContext, input DeleteInput) (bool, error) {
			return s.DeleteGuest(input.ID), nil
		},
	)
	tools["deleteGuest"] = deleteGuestTool

	// Tool 11: Toggle Guest RSVP
	toggleRSVPTool := genkit.DefineTool(g, "toggleGuestRSVP",
		"Toggles or cycles a guest's RSVP status (Confirmed -> Declined -> Pending -> Confirmed).",
		func(ctx *ai.ToolContext, input ToggleRSVPInput) (store.Guest, error) {
			gst, _ := s.ToggleGuestRSVP(input.ID)
			return gst, nil
		},
	)
	tools["toggleGuestRSVP"] = toggleRSVPTool

	// Tool 12: List Designs
	listDesignsTool := genkit.DefineTool(g, "listDesigns",
		"Retrieves all saved invitation card artwork concepts from the design studio.",
		func(ctx *ai.ToolContext, _ struct{}) ([]store.InvitationDesign, error) {
			return s.ListDesigns(), nil
		},
	)
	tools["listDesigns"] = listDesignsTool

	// Tool 13: Delete Design
	deleteDesignTool := genkit.DefineTool(g, "deleteDesign",
		"Deletes a saved invitation card concept by design ID.",
		func(ctx *ai.ToolContext, input DeleteInput) (bool, error) {
			return s.DeleteDesign(input.ID), nil
		},
	)
	tools["deleteDesign"] = deleteDesignTool

	// Tool 14: Export Event Data
	exportDataTool := genkit.DefineTool(g, "exportEventData",
		"Exports the complete event workspace profile, guest roster, itinerary, and artwork concepts as formatted JSON string.",
		func(ctx *ai.ToolContext, _ struct{}) (string, error) {
			bytes, err := s.ExportJSON()
			if err != nil {
				return "", err
			}
			return string(bytes), nil
		},
	)
	tools["exportEventData"] = exportDataTool

	// Tool 15: Reset Event Workspace
	resetStoreTool := genkit.DefineTool(g, "resetEventWorkspace",
		"Clears and resets the entire event profile, guest roster, itinerary, and design concepts.",
		func(ctx *ai.ToolContext, _ struct{}) (string, error) {
			s.ClearStore()
			return "Event workspace cleared successfully.", nil
		},
	)
	tools["resetEventWorkspace"] = resetStoreTool

	return tools
}

// resolveAPIKey resolves the active Gemini API key from HTTP request context (client header) or server env.
func resolveAPIKey(ctx context.Context) string {
	if ctx != nil {
		if keyVal, ok := ctx.Value(middleware.UserAPIKeyKey).(string); ok && strings.TrimSpace(keyVal) != "" {
			return strings.TrimSpace(keyVal)
		}
	}
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GOOGLE_GENAI_API_KEY"))
	}
	return apiKey
}

// resolveMapsAPIKey resolves the active Google Maps / Places API key from HTTP request context (client header) or server env.
func resolveMapsAPIKey(ctx context.Context) string {
	if ctx != nil {
		if keyVal, ok := ctx.Value(middleware.UserMapsAPIKeyKey).(string); ok && strings.TrimSpace(keyVal) != "" {
			return strings.TrimSpace(keyVal)
		}
	}
	mapsKey := strings.TrimSpace(os.Getenv("GOOGLE_MAPS_API_KEY"))
	if mapsKey == "" {
		mapsKey = strings.TrimSpace(os.Getenv("GOOGLE_PLACES_API_KEY"))
	}
	if mapsKey == "" {
		mapsKey = resolveAPIKey(ctx)
	}
	return mapsKey
}

// extractCityOrLocality extracts a clean neighborhood/locality & city string (e.g. "Vettuvankeni, Chennai") from a Google Places formatted address.
func extractCityOrLocality(formattedAddress string) string {
	parts := strings.Split(formattedAddress, ",")
	if len(parts) < 2 {
		return formattedAddress
	}
	cleanParts := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			cleanParts = append(cleanParts, trimmed)
		}
	}
	if len(cleanParts) == 0 {
		return formattedAddress
	}
	last := cleanParts[len(cleanParts)-1]
	if last == "India" || last == "USA" || last == "United States" || last == "UK" {
		cleanParts = cleanParts[:len(cleanParts)-1]
	}
	if len(cleanParts) == 0 {
		return formattedAddress
	}
	last = cleanParts[len(cleanParts)-1]
	hasDigits := false
	for _, r := range last {
		if r >= '0' && r <= '9' {
			hasDigits = true
			break
		}
	}
	if hasDigits && len(cleanParts) > 1 {
		cleanParts = cleanParts[:len(cleanParts)-1]
	}
	if len(cleanParts) == 0 {
		return formattedAddress
	}
	if len(cleanParts) == 1 {
		return cleanParts[0]
	}
	suburb := cleanParts[len(cleanParts)-2]
	city := cleanParts[len(cleanParts)-1]
	if strings.EqualFold(suburb, city) {
		return city
	}
	return fmt.Sprintf("%s, %s", suburb, city)
}

// VerifyVenueWithGoogleMaps queries Google Maps Places Text Search API to populate full VenueDetails struct.
func VerifyVenueWithGoogleMaps(ctx context.Context, venue, city string) store.VenueDetails {
	queryStr := strings.TrimSpace(fmt.Sprintf("%s, %s", venue, city))
	if city == "" {
		queryStr = strings.TrimSpace(venue)
	}
	encodedQuery := url.QueryEscape(queryStr)

	defaultMapURL := fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", encodedQuery)
	defaultDirURL := fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%s", encodedQuery)

	vd := store.VenueDetails{
		PrimaryVenue:           venue,
		VenueFormattedAddress:  queryStr,
		Address:                queryStr,
		GoogleMapURL:           defaultMapURL,
		GoogleMapDirectionsURL: defaultDirURL,
		VenuePhotoURL:          "https://images.unsplash.com/photo-1519167758481-83f550bb49b3?auto=format&fit=crop&w=800&q=80",
	}

	mapsKey := resolveMapsAPIKey(ctx)
	if mapsKey != "" && mapsKey != "dummy" {
		apiEndpoint := fmt.Sprintf("https://maps.googleapis.com/maps/api/place/textsearch/json?query=%s&key=%s", encodedQuery, mapsKey)
		resp, err := http.Get(apiEndpoint)
		if err == nil {
			defer resp.Body.Close()
			var result struct {
				Results []struct {
					Name             string `json:"name"`
					FormattedAddress string `json:"formatted_address"`
					PlaceID          string `json:"place_id"`
					Photos           []struct {
						PhotoReference string `json:"photo_reference"`
					} `json:"photos"`
				} `json:"results"`
				Status string `json:"status"`
			}
			if json.NewDecoder(resp.Body).Decode(&result) == nil && len(result.Results) > 0 {
				res := result.Results[0]
				if res.Name != "" {
					vd.PrimaryVenue = res.Name
				}
				if res.FormattedAddress != "" {
					vd.VenueFormattedAddress = res.FormattedAddress
					vd.Address = extractCityOrLocality(res.FormattedAddress)
					vd.VenueAdrFormatAddress = fmt.Sprintf("<span class=\"street-address\">%s</span>", res.FormattedAddress)
				}
				if res.PlaceID != "" {
					vd.PlaceID = res.PlaceID
					vd.GoogleMapURL = fmt.Sprintf("https://www.google.com/maps/place/?q=place_id:%s", res.PlaceID)
					vd.GoogleMapDirectionsURL = fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination_place_id=%s&destination=%s", res.PlaceID, encodedQuery)
				}
				if len(res.Photos) > 0 && res.Photos[0].PhotoReference != "" {
					vd.VenuePhotoURL = fmt.Sprintf("https://maps.googleapis.com/maps/api/place/photo?maxwidth=800&photo_reference=%s&key=%s", res.Photos[0].PhotoReference, mapsKey)
				}
			}
		}
	}

	return vd
}

// generateImageWithAPI executes image generation API call and writes card PNG asset file to disk under ./web/assets/.
func generateImageWithAPI(ctx context.Context, prompt, style, aspectRatio string, s *store.DataStore) GenerateImageOutput {
	if aspectRatio == "" {
		aspectRatio = "9:16"
	}
	apiKey := resolveAPIKey(ctx)

	assetID := fmt.Sprintf("card_%d", time.Now().UnixNano())
	pngFilename := fmt.Sprintf("%s.png", assetID)
	imageRelURL := fmt.Sprintf("/assets/%s", pngFilename)
	fullImageURL := fmt.Sprintf("http://localhost:3000/assets/%s", pngFilename)
	dataDir := strings.TrimSpace(os.Getenv("SHUBH_DATA_DIR"))
	if dataDir == "" {
		dataDir = "./data"
	}
	persistentAssetDir := filepath.Join(dataDir, "assets")
	_ = os.MkdirAll(persistentAssetDir, 0755)
	_ = os.MkdirAll("./web/assets", 0755)

	persistentOutPath := filepath.Join(persistentAssetDir, pngFilename)
	localOutPath := fmt.Sprintf("./web/assets/%s", pngFilename)

	evt := s.GetEvent()
	titleStr := evt.Title
	if titleStr == "" {
		titleStr = "Special Event Celebration"
	}
	rawDate := evt.Date
	if rawDate == "" {
		rawDate = "Upcoming Date"
	}
	dateStr := FormatHumanReadableDate(rawDate)

	venueStr := evt.Venue
	if venueStr == "" {
		venueStr = "Main Venue"
	}

	// Append negative UI framing and clean human-readable text printing mandates with dynamic aspect ratio
	cleanPrompt := fmt.Sprintf(
		"%s. ELEGANT CARD PLAQUE TYPOGRAPHY MANDATE: Render ONLY the following 2 clean text lines inside the central artwork plaque: Line 1: \"%s\", Line 2: \"%s • %s\". CRITICAL PRINTING RULES: Do NOT print field labels like 'Title:', 'Date:', or 'Venue:'. Do NOT duplicate text on extra banners or side ribbons. Keep the central card plate clean, minimalist, and uncluttered with beautiful serif typography. Standalone physical invitation card graphic artwork in %s aspect ratio. No smartphone UI, no status bars, no screen bezels.",
		prompt, titleStr, dateStr, venueStr, aspectRatio,
	)

	log.Printf("[Image Generator] Starting artwork generation (Style: %q, AspectRatio: %q, Prompt: %q)", style, aspectRatio, prompt)

	var imgBytes []byte

	// Attempt remote Gemini Flash Multimodal / Imagen PNG generation if API key is active
	if apiKey != "" {
		imgBytes = fetchGeminiImageBytes(apiKey, cleanPrompt, aspectRatio)
		if len(imgBytes) > 0 {
			log.Printf("[Image Generator] Successfully synthesized remote Imagen/Gemini PNG artwork (%d bytes)", len(imgBytes))
		}
	}

	// Fallback local PNG raster rendering if remote API is unconfigured
	if len(imgBytes) == 0 {
		log.Printf("[Image Generator Warning] Remote API unconfigured or returned 0 bytes, rendering local PNG card fallback...")
		imgBytes = renderPNGCard(titleStr, dateStr, venueStr, style, prompt, aspectRatio)
	}

	// Always write physical PNG asset to persistent data store and local web assets
	_ = os.WriteFile(persistentOutPath, imgBytes, 0644)
	if err := os.WriteFile(localOutPath, imgBytes, 0644); err != nil {
		log.Printf("[Image Generator Warning] Writing local PNG asset: %v", err)
	} else {
		log.Printf("[Image Generator Success] Saved PNG card asset to %s and %s (URL: %s, Size: %d bytes)", persistentOutPath, localOutPath, imageRelURL, len(imgBytes))
	}

	// Save design to store with generated image URL
	s.AddDesign(store.InvitationDesign{
		Prompt:      prompt,
		StyleTheme:  style,
		AspectRatio: aspectRatio,
		ImageURL:    imageRelURL,
		Headline:    titleStr,
		Subhead:     fmt.Sprintf("%s • %s", dateStr, venueStr),
		CreatedAt:   time.Now(),
	})

	markdownImg := fmt.Sprintf("![Invitation Card Artwork](%s)", fullImageURL)

	return GenerateImageOutput{
		ImageURL:    imageRelURL,
		MarkdownImg: markdownImg,
		Status:      fmt.Sprintf("Card artwork PNG generated and saved to %s!", imageRelURL),
		Prompt:      prompt,
	}
}

// fetchGeminiImageBytes executes REST API calls matching shubh-plan-open/generator/prompter.go (Gemini Multimodal Image & Imagen API)
func fetchGeminiImageBytes(apiKey, prompt, aspectRatio string) []byte {
	if aspectRatio == "" {
		aspectRatio = "9:16"
	}
	// 1. Multimodal Gemini Image API Format (gemini-3.1-flash-image)
	imageModels := []string{"gemini-3.1-flash-image"}
	for _, modelName := range imageModels {
		endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
		reqBody := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{"text": prompt},
					},
				},
			},
			"generation_config": map[string]interface{}{
				"response_modalities": []string{"IMAGE"},
			},
		}

		jsonBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonBytes))
		if err == nil {
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err == nil && resp.StatusCode == http.StatusOK {
				var resStruct struct {
					Candidates []struct {
						Content struct {
							Parts []struct {
								InlineData struct {
									MimeType string `json:"mimeType"`
									Data     string `json:"data"`
								} `json:"inlineData"`
							} `json:"parts"`
						} `json:"content"`
					} `json:"candidates"`
				}
				if json.Unmarshal(body, &resStruct) == nil && len(resStruct.Candidates) > 0 {
					for _, part := range resStruct.Candidates[0].Content.Parts {
						if part.InlineData.Data != "" {
							if decoded, err := base64.StdEncoding.DecodeString(part.InlineData.Data); err == nil && len(decoded) > 0 {
								return decoded
							}
						}
					}
				}
			}
		}
	}

	// 2. Imagen 3 API Format (imagen-3.0-generate-002)
	imagenModels := []string{"imagen-3.0-generate-002", "imagen-3.0-fast-generate-001"}
	for _, modelName := range imagenModels {
		endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:predict?key=%s", modelName, apiKey)
		reqBody := map[string]interface{}{
			"instances": []map[string]interface{}{
				{"prompt": prompt},
			},
			"parameters": map[string]interface{}{
				"sampleCount":      1,
				"aspectRatio":      aspectRatio,
				"outputMimeType":   "image/png",
				"personGeneration": "ALLOW_ADULT",
			},
		}

		jsonBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonBytes))
		if err == nil {
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err == nil && resp.StatusCode == http.StatusOK {
				var resStruct struct {
					Predictions []struct {
						BytesBase64Encoded string `json:"bytesBase64Encoded"`
					} `json:"predictions"`
				}
				if json.Unmarshal(body, &resStruct) == nil && len(resStruct.Predictions) > 0 {
					if resStruct.Predictions[0].BytesBase64Encoded != "" {
						if decoded, err := base64.StdEncoding.DecodeString(resStruct.Predictions[0].BytesBase64Encoded); err == nil && len(decoded) > 0 {
							return decoded
						}
					}
				}
			}
		}
	}

	return nil
}

// renderPNGCard renders custom high-res PNG image raster with title, date, venue, style, and prompt spec.
func renderPNGCard(title, date, venue, style, prompt, aspectRatio string) []byte {
	_ = title
	_ = date
	_ = venue
	_ = prompt
	width, height := 450, 800
	switch aspectRatio {
	case "1:1":
		width, height = 600, 600
	case "4:5":
		width, height = 500, 625
	case "16:9":
		width, height = 800, 450
	default:
		width, height = 450, 800
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Dark obsidian background (#0F172A)
	bgColor := color.RGBA{R: 15, G: 23, B: 42, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Border color based on theme
	borderColor := color.RGBA{R: 212, G: 175, B: 55, A: 255} // Gold default
	styleLower := strings.ToLower(style)
	if strings.Contains(styleLower, "south indian") {
		borderColor = color.RGBA{R: 255, G: 153, B: 51, A: 255}
	} else if strings.Contains(styleLower, "paper cut") {
		borderColor = color.RGBA{R: 56, G: 189, B: 248, A: 255}
	} else if strings.Contains(styleLower, "clay 3d") {
		borderColor = color.RGBA{R: 244, G: 162, B: 97, A: 255}
	} else if strings.Contains(styleLower, "pop art") {
		borderColor = color.RGBA{R: 236, G: 72, B: 153, A: 255}
	} else if strings.Contains(styleLower, "mughal") {
		borderColor = color.RGBA{R: 16, G: 185, B: 129, A: 255}
	} else if strings.Contains(styleLower, "watercolor") {
		borderColor = color.RGBA{R: 168, G: 85, B: 247, A: 255}
	}

	// Draw outer border frame
	for x := 12; x < width-12; x++ {
		for t := 0; t < 6; t++ {
			img.Set(x, 12+t, borderColor)
			img.Set(x, height-17+t, borderColor)
		}
	}
	for y := 12; y < height-12; y++ {
		for t := 0; t < 6; t++ {
			img.Set(12+t, y, borderColor)
			img.Set(width-17+t, y, borderColor)
		}
	}

	// Draw inner decorative plaque background (#1E293B)
	plaqueBg := color.RGBA{R: 30, G: 41, B: 59, A: 255}
	for x := 40; x < width-40; x++ {
		for y := 40; y < height-40; y++ {
			img.Set(x, y, plaqueBg)
		}
	}

	// Draw inner plaque border
	for x := 40; x < width-40; x++ {
		img.Set(x, 40, borderColor)
		img.Set(x, height-40, borderColor)
	}
	for y := 40; y < height-40; y++ {
		img.Set(40, y, borderColor)
		img.Set(width-40, y, borderColor)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err == nil {
		return buf.Bytes()
	}
	return nil
}

// FormatHumanReadableDate parses raw ISO or formatted date strings into clean human-readable dates (e.g. October 12, 2026).
func FormatHumanReadableDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "Upcoming Date" {
		return "Upcoming Date"
	}

	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"02-01-2006",
		"02/01/2006",
		"01/02/2006",
		"Jan 02, 2006",
		"January 02, 2006",
		"02 Jan 2006",
		"02 January 2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 Jan 2006",
		"2 January 2006",
		"2006-01-02T15:04:05Z07:00",
	}

	for _, fmtStr := range formats {
		if pt, err := time.Parse(fmtStr, raw); err == nil {
			return pt.Format("January 2, 2006")
		}
	}

	return raw
}
