# 🚀 Changelog — Shubh Plan Web

All notable changes to the **Shubh Plan Web** project will be documented in this file.

---

## 📦 `v0.1.0` — Initial Release (2026-08-27)

> **Shubh Plan Web v0.1.0** is the first official release of the AI-native event planning operating system and bespoke invitation studio built on **Go 1.25+**, **Firebase Genkit Go SDK**, and **Google Gemini Cloud AI**.

### 🌟 Key Feature Highlights

#### 🤖 1. AI Event Assistant & SSE Real-Time Streaming
- **Central Model**: Standardised on **`googleai/gemini-flash-latest`** for lightning-fast, high-reasoning text generation and event orchestration.
- **Real-Time SSE Streaming**: Full Server-Sent Events (`/api/stream/assistant`) streaming pipeline with glowing 3-dot typing indicators and instant token delivery.
- **Slash Commands (`/`) & Quick Chips**: Immediate interception for quick action commands (`/summarize`, `/add-guests`, `/schedule`, `/generate-invitation`) with floating backdrop-blur autocomplete menu and full keyboard navigation.
- **Smart Event Parser**: Automatically extracts and persists event metadata (Title, Event Type, Host Names, Date, Venue, Target Guest Count) to the unified DataStore live.

#### 🎨 2. Bespoke AI Invitation Studio
- **High-Res PNG Image Model**: Powered strictly by **`googleai/gemini-3.1-flash-image`** for standalone, print-ready invitation card artwork.
- **4 Dynamic Aspect Ratios**: Full support for `9:16` (Story/Mobile), `4:5` (Portrait/Feed), `1:1` (Square/Post), and `16:9` (Landscape/Banner).
- **7 Signature Aesthetic Presets**:
  1. *South Indian* (Kolam art, banana leaf motifs, marigold garlands, temple gold)
  2. *Paper Cut 3D* (Intricate die-cut filigree, crisp drop shadows)
  3. *Clay 3D* (Charming claymation 3D characters, tactile sculpted depth)
  4. *Pop Art* (Bold retro comic lines, vibrant halftone contrast)
  5. *Mughal* (Royal jali lattice arches, emerald & gold floral vines)
  6. *Minimalist Gold* (Obsidian / ivory backdrop, sleek gold foil line art)
  7. *Watercolor* (Soft hand-painted pastel washes, fluid botanical flora)
- **Prompt Synthesizer**: 2-step AI Art Director flow synthesizing 4 custom prompt concepts.
- **Auto-Clear Studio UX**: Automatically clears prompt suggestion cards once PNG artwork rendering completes.

#### 📍 3. Verified Venue Showcase & Google Places API
- **Context-Aware Venue Verification**: Queries Google Places Text Search API dynamically per request.
- **Rich Metadata Extraction**: Populates verified formatted addresses, Google Place IDs, 1-click Google Maps links, and turn-by-turn directions shortcuts.
- **High-Res Venue Photos**: Displays real venue photos (`VenuePhotoURL`) in the right sidebar's Verified Venue Showcase card.

#### 👥 4. Modular In-Chat Component Widgets (`web/widgets.js`)
- **`AddGuestWidget`**:
  - Category selection pills (*Family*, *Friends*, *VIPs*, *Colleagues*).
  - RSVP status pills (*Confirmed*, *Pending*).
  - Plus-ones counter (*+0*, *+1*, *+2*, *+3*).
  - **Dietary Preference Dropdown Selection**: *No Specific Preference (Standard)*, *🥗 Vegetarian (Pure Veg)*, *🪷 Jain (No Onion / No Garlic)*, *🌱 Vegan (Plant-Based)*, *🌙 Halal*, *🥚 Eggetarian*, *🌾 Gluten-Free / Special Allergies*.
- **`GenerateInvitationWidget`**: 2-step prompt synthesizer and card renderer.
- **`ScheduleSessionWidget`**: Session title, time, and location picker.

#### 🔐 5. Security, Dual-Key BYOK & Data Persistence
- **Argon2id Authentication**: Native Argon2id password hashing matching enterprise open-source specifications.
- **Dual-Key BYOK Modal**: Browser settings modal supporting both **Gemini API Key** and **Google Maps Places API Key** stored strictly in browser `localStorage`.
- **Automatic Key Detection**: Top navbar status indicator dot (`🔑 API Key Active`) and auto-populating input fields with active status banners.
- **Fly.io Production Deployment**: Containerised Docker build with rolling updates and `initial_delay = 5s` healthcheck configuration.

---

### 📊 Release Summary

| Component | Technology / Specification |
| :--- | :--- |
| **Go Runtime** | Go `1.25+` |
| **AI Framework** | Firebase Genkit Go SDK (`github.com/firebase/genkit/go`) |
| **Text Generation Model** | `googleai/gemini-flash-latest` |
| **Image Generation Model** | `googleai/gemini-3.1-flash-image` |
| **Auth Engine** | Argon2id (`$argon2id$...` with salt & key derivation) |
| **Frontend UI** | Vanilla CSS, HTMX, Single-Page Architecture |
| **Deployment Target** | Fly.io (`sin` Singapore region) |

---

*Shubh Plan Web v0.1.0 — Copyright © 2026 Ekarna Interactive Technology LLP.*
