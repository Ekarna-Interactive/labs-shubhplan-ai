# ✨ Shubh Plan Web — AI Event Operating System & Bespoke Invitation Studio

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![Firebase Genkit](https://img.shields.io/badge/Genkit-v1.12-FFCA28?style=for-the-badge&logo=firebase&logoColor=black)](https://firebase.google.com/docs/genkit-go)
[![Argon2id Auth](https://img.shields.io/badge/Auth-Argon2id-7C3AED?style=for-the-badge)](pkg/auth)
[![HTMX Web UI](https://img.shields.io/badge/HTMX-Vanilla%20CSS-38BDF8?style=for-the-badge&logo=htmx&logoColor=white)](https://htmx.org)
[![Fly.io Ready](https://img.shields.io/badge/Fly.io-Deployment%20Ready-24185B?style=for-the-badge&logo=fly.io&logoColor=white)](fly.toml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=for-the-badge&logo=apache)](LICENSE)

**Shubh Plan Web is an all-in-one AI-native event planning workspace and generative invitation studio. It empowers event planners and hosts to design high-resolution invitation artwork, verify venue details with Google Maps, manage guest RSVPs and dietary needs, and coordinate ceremony schedules in real time.**

[Quickstart](#-quickstart) • [Architecture](#-architecture) • [Feature Capabilities](#-feature-capabilities) • [Operating Modes](#-operating-modes--byok) • [Data Persistence](#-unified-data-persistence) • [Deployment Guide](DEPLOYMENT.md) • [Environment Setup](#%EF%B8%8F-environment-configuration)

> 🌐 **Live Web Demo**: [https://shubh-plan-demo.fly.dev](https://shubh-plan-demo.fly.dev)  
> 🔑 **Pre-seeded Admin Account**: `admin@shubhplan.ai` / `shubh2026` (Argon2id Auth)  
> ⚡ **Demo Key Setup**: Bring Your Own Gemini API Key (BYOK) via top-bar settings modal or 1-click Instant Guest Demo.

</div>

---

```mermaid
graph TD
    classDef web fill:#0f172a,stroke:#38bdf8,stroke-width:2px,color:#f8fafc;
    classDef core fill:#1e1b4b,stroke:#818cf8,stroke-width:2px,color:#f8fafc;
    classDef cloud fill:#0284c7,stroke:#38bdf8,stroke-width:2px,color:#f8fafc;
    classDef store fill:#064e3b,stroke:#34d399,stroke-width:2px,color:#f8fafc;

    subgraph CLIENTS ["🌐 Client Web User Interface"]
        WEB["🌐 HTMX & Vanilla JS SPA<br/><i>(Port 3000 • Single-Page Dashboard)</i>"]:::web
        SSE["⚡ SSE Real-Time Streaming Chat<br/><i>(Server-Sent Events)</i>"]:::web
    end

    subgraph CORE ["⚡ Shubh Plan Web Core"]
        AUTH["🔐 Native Argon2id Auth Engine<br/><i>(pkg/auth • Session Cookies)</i>"]:::core
        GENKIT["🤖 Firebase Genkit AI Engine<br/><i>(pkg/genkit • 4 Registered Flows & Tools)</i>"]:::core
        AGENTS["🧠 Multi-Agent Flow Registry<br/><i>(Planner, Designer, RSVP, Logistics)</i>"]:::core
    end

    subgraph CLOUD ["☁️ Google AI & Places Cloud"]
        GEMINI["✨ Google Gemini LLM & Imagen API<br/><i>(gemini-2.5-flash / gemini-1.5-flash)</i>"]:::cloud
        PLACES["📍 Google Places Search API<br/><i>(Verified Venues & Photos)</i>"]:::cloud
    end

    subgraph DATA ["📦 Unified Data Persistence (/app/data)"]
        USERDB["📄 Users & Sessions Store<br/><i>(users.json • Argon2id Hashes)</i>"]:::store
        STOREDB["📄 Event Domain Store<br/><i>(store.json • Events, Guests, Designs)</i>"]:::store
        ASSETS["🎨 Card Assets Store<br/><i>(web/assets/*.png • PNG Artwork)</i>"]:::store
    end

    WEB <--> AUTH
    WEB <--> GENKIT
    SSE <--> GENKIT
    GENKIT <--> AGENTS
    AGENTS <--> GEMINI
    AGENTS <--> PLACES
    AUTH <--> USERDB
    GENKIT <--> STOREDB
    GENKIT --> ASSETS
```

---

## ⚡ Quickstart

Launch **Shubh Plan Web** locally in one command:

```bash
cd apps/shubh-plan-web

# Run the local web server
go run main.go
```

Open your browser and navigate to:
- 🌐 **Web Interface**: [`http://localhost:3000`](http://localhost:3000)
- 🔍 **Healthcheck Endpoint**: [`http://localhost:3000/api/health`](http://localhost:3000/api/health)

---

## ✨ Feature Capabilities

### 🔐 1. Dual Operating Modes & Bring Your Own Key (BYOK)
- **Demo Mode (`APP_MODE=demo`)**: Tailored for public cloud deployments (e.g. Fly.io). Server-side API key is unconfigured by default. Users provide their own free Gemini API key in a top-bar settings modal (stored strictly in browser `localStorage` for privacy). Includes 1-click **Instant Guest Demo** with pre-populated sample event data.
- **Server Mode (`APP_MODE=server`)**: Designed for local or private server usage. Automatically loads system environment variables (`GEMINI_API_KEY`, `GOOGLE_MAPS_API_KEY`) from `.env` for zero-configuration team access.

### 👤 2. Native Argon2id Authentication
- **Enterprise-Grade Hashing**: Uses Argon2id password hashing (`$argon2id$...` with salt & key derivation) matching `apps/shubh-plan-open`.
- **Session Security**: Supports `admin`, `planner`, and `guest` roles with secure `shubh_session` HTTP cookies and `X-Session-ID` headers.

### 🎨 3. Bespoke AI Invitation Studio
- **Dynamic Aspect Ratios**: Design in `9:16` vertical poster, `4:5` portrait, `1:1` square, or `16:9` landscape.
- **7 Signature Aesthetic Presets**: Choose from *South Indian*, *Paper Cut 3D*, *Clay 3D*, *Pop Art*, *Mughal*, *Minimalist Gold*, and *Watercolor*.
- **Custom Visual Elements & Typography**: Select custom tags (*marigold garlands, kolam art, vintage peacock*) and typography pairings (*Cinzel Decorative & Outfit*, *Great Vibes*, *Playfair Display*).
- **Gemini LLM Prompt Synthesizer**: Click **"✨ Step 1: Generate AI Prompt Suggestions"** to invoke Gemini LLM as an AI Art Director to synthesize 4 custom prompt concepts.
- **High-Res PNG Card Rendering**: Renders full-resolution `.png` card artwork with 1-click download and clipboard prompt copying.

### 🤖 4. AI Event Assistant, Slash Commands & Component Widgets
- **Conversational Setup & Real-Time SSE**: Real-time Server-Sent Events (SSE) chat for natural conversation setup (*"Plan a wedding for Maya & Vikram on Dec 20 at Hyatt Regency Mumbai"*) with real-time `⏳ Thinking...` loading states and glowing 3-dot typing animations.
- **Interactive Quick Action Chips & Slash Commands (`/`)**: Trigger quick actions (`/summarize`, `/add-guests`, `/schedule`, `/generate-invitation`) via top chips or by typing `/` to open a floating backdrop-blur autocomplete menu with full keyboard (`Up`/`Down`/`Enter`/`Tab`) navigation.
- **Modular In-Chat Component Widgets ([`web/widgets.js`](file:///c:/Users/Gokul/Documents/Programming/Antigravity/shubh-plan/apps/shubh-plan-web/web/widgets.js))**: Self-contained 1-click HTML form components embedded inside chat bubbles:
  - **`AddGuestWidget`**: Category pills (`Family`, `Friends`, `VIPs`, `Colleagues`), RSVP pills, plus-ones counter, and guest name input.
  - **`GenerateInvitationWidget`**: 2-step flow with style preset pills (`Clay 3D`, `South Indian`, `Paper Cut`, `Mughal`, etc.), aspect ratio pills (`4:5`, `9:16`, `1:1`, `16:9`), custom prompt input, 4 synthesized Gemini AI prompt option cards, and 1-click PNG artwork rendering.
  - **`ScheduleSessionWidget`**: Session title, time, and location inputs.
- **Full-Screen Artwork Preview Modal**: Click any generated card image in chat or Invitation Studio to open high-res `#preview-card-modal` with direct PNG download options.

### 📍 5. Verified Venue Showcase & Google Places API
- **Google Maps Integration**: Queries Google Places Text Search API for exact formatted addresses,Place IDs, 1-click Google Maps links, turn-by-turn directions, and real venue photos with fallback luxury banquet hall photography.

---

## 📦 Unified Data Persistence

All application data is persisted in human-readable JSON files inside the data directory (`./data` locally or `/app/data` when deployed):

| Data File | Contents | File Format |
| :--- | :--- | :--- |
| `data/users.json` | User accounts, Argon2id password hashes, roles, and registration dates | POSIX `0644` JSON |
| `data/store.json` | Active event profile, guest roster, itinerary run-of-show, and generated invitation designs | POSIX `0644` JSON |
| `web/widgets.js` | Modular in-chat UI component widgets (`AddGuestWidget`, `GenerateInvitationWidget`, `ScheduleSessionWidget`) | Vanilla JS Module |
| `web/assets/*.png` | High-resolution generated PNG invitation card artwork files | Binary PNG Images |

---

## ⚙️ Environment Configuration

| Variable | Default | Description |
| :--- | :--- | :--- |
| `GEMINI_API_KEY` | *(Optional)* | Server-wide Google Gemini API Key. Web users can override with client keys in Demo Mode. |
| `APP_MODE` | `server` | Execution mode (`server` for local/on-premise vs `demo` for Fly.io BYOK). |
| `SHUBH_DATA_DIR` | `./data` | Local directory for user accounts, RSVPs, itinerary JSON, and store persistence. |
| `PORT` | `3000` | HTTP Web UI server listening port. |
| `GOOGLE_MAPS_API_KEY` | *(Optional)* | Key for live Google Places venue search and photo fetching. |

---

## 🤝 Contributing

`shubh-plan-web` operates under a source-available / read-only open-source model. Please see the [CONTRIBUTING.md](CONTRIBUTING.md) guide for details.

---

## 📄 License

This open-source project is licensed under the **Apache License 2.0**. Copyright 2026 Ekarna Interactive Technology LLP. See the [LICENSE](LICENSE) file for details.
