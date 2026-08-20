# ✨ Shubh Plan AI — Open-Source Multi-Agent Event Engine

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![Google ADK v2](https://img.shields.io/badge/Google%20ADK-v2.2.0-4285F4?style=for-the-badge&logo=google&logoColor=white)](https://google.golang.org/adk)
[![Honcho v3 REST](https://img.shields.io/badge/Honcho%20Memory-v3%20REST-7C3AED?style=for-the-badge)](https://honcho.dev)
[![Wish SSH TUI](https://img.shields.io/badge/Wish%20SSH-Bubble%20Tea-FF4081?style=for-the-badge)](https://github.com/charmbracelet/wish)
[![HTMX Web UI](https://img.shields.io/badge/HTMX-Tailwind%20CSS-38BDF8?style=for-the-badge&logo=htmx&logoColor=white)](https://htmx.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=for-the-badge&logo=apache)](LICENSE)

**Shubh Plan AI is an all-in-one event planning workspace and invitation studio. It helps hosts and event planners design personalized invitation artwork, structure ceremony schedules, manage guest RSVPs and dietary requirements, and coordinate every celebration detail through an intuitive AI copilot.**

[Quickstart](#-quickstart) • [Feature Matrix](#-feature-capabilities--api-key-matrix) • [Architecture](#-architecture) • [Key Features](#-key-features) • [User Interfaces](#-user-interfaces) • [Data Persistence](#-unified-data-persistence-data) • [TUI Guide](TUI.md) • [Deployment Guide](DEPLOYMENT.md) • [Environment Setup](#%EF%B8%8F-environment-configuration)

</div>

---

## 🚀 Quickstart

### 📋 Prerequisites

Before running `shubh-plan-open`, ensure you have the following installed:

* **Go 1.22 or higher**: [Download Go](https://go.dev/dl/) *(Required for building from source)*
* **Docker & Docker Compose**: [Download Docker](https://www.docker.com/) *(Optional, recommended for quick containerized setup)*
* **SSH Client**: Standard OpenSSH client (`ssh`) *(Built-in on Linux, macOS, and Windows PowerShell)*

---

### Option A: Run with Docker Compose (Recommended)

```bash
# 1. Clone the repository
git clone https://github.com/Ekarna-Interactive/labs-shubhplan-ai.git
cd labs-shubhplan-ai

# 2. Launch with Docker Compose
docker compose up -d
```

### Option B: Build & Run from Source (Go 1.22+)

```bash
# 1. Clone and enter repository directory
git clone https://github.com/Ekarna-Interactive/labs-shubhplan-ai.git
cd labs-shubhplan-ai/apps/shubh-plan-open

# 2. Build single Go binary
go build -o shubh-plan-open .

# 3. Launch in server mode (Web UI + SSH TUI Server)
./shubh-plan-open --server
```

### 📍 Accessing Your Workspace

* **🌐 HTMX Web Dashboard**: Open [`http://localhost:3000`](http://localhost:3000) in your browser. On fresh boot, follow the on-screen **Owner Setup Wizard** to create your primary account.
* **💻 Wish SSH Terminal UI**: Connect instantly via `ssh -i ~/.ssh/id_ed25519 -p 2222 localhost`.

---

## ⚡ Feature Capabilities & API Key Matrix

`shubh-plan-open` is designed with zero-dependency offline fallbacks. You can run and evaluate the application completely keyless in **Offline Dry-Run Mode**, or supply API keys to unlock cloud AI features:

| Feature Area | Keyless / Offline Dry-Run Mode | Cloud Key Unlocked Mode | Required Key |
| :--- | :--- | :--- | :--- |
| **Authentication & Auth** | Full Native Argon2id Auth (`data/users.json`) | Full Native Auth & SSH Public Key Mapping | None *(Inbuilt)* |
| **Event Profile & Itinerary** | Full local editing (`data/event-details.json` & `itinerary.json`) | Full 2-Way Sync with Markdown (`event_details.md`) | None *(Inbuilt)* |
| **Guest RSVPs & Logistics** | Full headcount meters, dietary tracking & CSV export | Full headcount meters, dietary tracking & CSV export | None *(Inbuilt)* |
| **Smart Memory** | Zero-config Local Vector Store (`./data/honcho_memory.json`) | Cloud Session & Vector Sync (`api.honcho.dev/v3`) | `HONCHO_API_KEY` *(Optional)* |
| **Venue Search** | Inbuilt AI Venue Agent & Curated Venue Autocomplete | Live Google Maps / Places API Search & Metadata | `GOOGLE_PLACES_API_KEY` *(Optional)* |
| **AI Invitation Artwork** | Concept compilation & layout preview | Full High-Res Imagen Artwork Generation | `GEMINI_API_KEY` *(Required for AI Images)* |
| **AI Copilot Chat** | Basic command execution | Full Multi-Agent Reasoning via Google ADK v2 | `GEMINI_API_KEY` *(Required for Chat AI)* |

---

## 🏗 Architecture

| Layer | Implementation | Description |
| :--- | :--- | :--- |
| **Web Server** | Go `net/http` + Custom Middleware | Single Go binary serving HTMX partials, HTTP-Only session cookies, and SSE streams on port `3000`. |
| **Terminal UI Server** | Charm Wish + Bubble Tea | Zero-install SSH terminal application serving interactive viewports and keybindings on port `2222`. |
| **Authentication Engine** | Native Argon2id + POSIX `0600` | Gitea-style self-hosted user store (`data/users.json`) with password hashing and SSH public key auto-login. |
| **Multi-Agent Engine** | Official Google ADK v2 (`adk/v2/agent`) | 5 specialized Go agents (`AI Concierge`, `GuestConcierge`, `TimelineAgent`, `BudgetAgent`, `VendorAgent`). |
| **Smart Memory** | Honcho v3 REST + Local Fallback | Dual-mode memory driver syncing session representations with `api.honcho.dev/v3` or `./data/honcho_memory.json`. |
| **Data Persistence** | POSIX Local Files & Unified JSON | Shared JSON files under `./data/` (`event-details.json`, `itinerary.json`, `rsvps.json`) 2-way synced across interfaces. |

---

## 🌟 Key Features

* **⚡ Unified Single Go Binary**: Operates 100% locally in Go with zero `node_modules` or JavaScript runtime requirements.
* **🔐 Self-Hosted Native Go Authentication**: Zero-dependency user management engine storing Argon2id password hashes and SSH public keys in `data/users.json` with POSIX `0600` permissions. Features HTTP-Only 7-day session cookies (`shubh_session`) and First-Time Owner Setup.
* **🎉 Guided Setup Wizards**: Onboarding modals for Owner account creation, Event Profile configuration, and API Key entry.
* **🧩 Pure HTMX & Middleware Architecture**: Operates on a pure Go + HTMX engine with Go middleware (`HTMXMiddleware`, `APIKeyContextMiddleware`, `RequireAuthHTMX`), Out-of-Band (`hx-swap-oob`) multi-target swaps, and server event triggers (`HX-Trigger`).
* **🔄 2-Way Live Real-Time Controls Synchronization**: Executing slash commands (`/aspect`, `/style`, `/welcome`, `/currency`) in Copilot chat instantly synchronizes Tab 1 (AI Invitation Studio) sidebar dropdown controls, `data/event-details.json`, and `event_details.md` in 100% real time.
* **💬 Interactive Copilot Chat & Dynamic Autocomplete**: Real-time slash command autocomplete popup that filters as you type (`/sugg`, `/asp`, `/sty`), rendering rich, clickable option buttons directly in chat for `/suggest`, `/style`, `/aspect`, and `/currency`.
* **🗓️ Ordinal Date Typography & Smart Banner Prompt Compiler**: Automatic date parser formatting dates into human-readable ordinal strings (`September 20th, 2026`) on generated invitation cards, with anti-blank prompt sanitization guaranteeing text is printed directly inside central banner plaques.
* **📦 Unified JSON Storage & 1:1 Web/TUI Synchronization**: Shared data stores under `./data/`. Any update in Web UI (e.g. adding a ceremony sub-event or updating guest headcount) immediately reflects in the SSH TUI and vice versa.

---

## 🖥 User Interfaces

### 1. 🌐 HTMX Web Dashboard (`http://localhost:3000`)
Features 5 modular workspace tabs:
1. **✨ AI Invitation Studio**: Event parameters, aesthetic theme presets (South Indian Gold, Mughal Heritage, Paper Cut 3D, Minimalist), Gemini prompt compilation, and output card gallery.
2. **👥 Guest Roster & RSVPs**: Real-time headcount meters (Confirmed, Pending, Declined), dietary breakdown chips, interactive guest table, and CSV download.
3. **📅 Event Timeline**: Dynamic ceremony itinerary list (`data/itinerary.json`) with interactive form to add, view, and delete sub-events.
4. **🤖 AI Concierge Copilot**: Real-time multi-agent chat interface connected via Server-Sent Events (SSE), Quick Pills bar, and slash autocomplete popup.
5. **🧠 Smart Memory & Sub-Agents**: Live Honcho memory representations, recorded sessions, and cloud peer status.

### 2. 💻 Wish SSH Terminal UI (`ssh -p 2222 localhost`)
Interactive terminal application supporting live tab switching, keybindings, RSVP wizards (`/add-rsvp`), event profile editing (`/planner`, `/event`), and key setup prompts.

📖 **For full SSH authentication, PowerShell setup, and TUI slash commands, see the dedicated [TUI Guide](TUI.md).**
🚀 **For Fly.io cloud deployment, volume mounting, and IP configuration, see the [Deployment Guide](DEPLOYMENT.md).**

#### 🔑 SSH Client Public Key Connection Methods

Wish SSH uses standard SSH public key cryptography (`~/.ssh/id_ed25519` / `~/.ssh/id_rsa`).

* **Interactive Connection (On-Screen Key Prompt)**:
  ```bash
  ssh -i ~/.ssh/id_ed25519 -p 2222 shubh-plan-ai.fly.dev
  ```

* **Direct One-Liner Connection (`SendEnv` Auto-Authentication)**:
  ```bash
  GEMINI_API_KEY="......" HONCHO_API_KEY="....." ssh -i ~/.ssh/id_ed25519 -o SendEnv=GEMINI_API_KEY -o SendEnv=HONCHO_API_KEY -p 2222 shubh-plan-ai.fly.dev
  ```

---

## 📦 Unified Data Persistence (`./data/`)

All workspace state is persisted inside `./data/` (or Fly.io persistent volume mount at `/app/data/`):

| File Path | Description | Security / Permissions |
| :--- | :--- | :--- |
| `data/users.json` | User accounts, Argon2id password hashes, roles, and SSH keys | POSIX `0600` (Owner read/write only) |
| `data/event-details.json` | Canonical JSON representation of active event profile | POSIX `0644` |
| `event_details.md` | Human-editable Markdown profile (2-way dual-synced with JSON) | POSIX `0644` |
| `data/itinerary.json` | Ceremony sub-events, timeline slots, and run-of-show schedule | POSIX `0644` |
| `data/rsvps.json` | Guest roster, headcount meters, and dietary requirements | POSIX `0644` |
| `data/honcho_memory.json` | Local vector memory store for AI Concierge | POSIX `0644` |

---

## ⚙️ Environment Configuration

| Variable | Default | Description |
| :--- | :--- | :--- |
| `GEMINI_API_KEY` | *(Optional)* | Server-wide Google Gemini API Key. Web/SSH users can override with client keys. |
| `GEMINI_TEXT_MODEL` | `gemini-3.5-flash` | Primary text model for prompt compilation & copilot chat. |
| `GEMINI_IMAGE_MODEL` | `gemini-3.1-flash-image` | Model for invitation card artwork generation. |
| `GOOGLE_PLACES_API_KEY` | *(Optional)* | Key for live Google Places venue search. Uses AI fallback if empty. |
| `HONCHO_API_KEY` | *(Optional)* | Key for `api.honcho.dev/v3`. Defaults to zero-config local JSON store if empty. |
| `HONCHO_APP_ID` | `shubh-plan-ai` | Honcho Application ID namespace. |
| `SERVER_MODE` / `MULTI_USER` | `false` | Enable to prevent client browser keys from writing to global server `.env`. |
| `PORT` | `3000` | HTTP Web UI server listening port. |
| `SSH_PORT` | `2222` | Wish SSH Terminal server listening port. |
| `SHUBH_DATA_DIR` | `./data` | Local directory for user accounts, RSVPs, itinerary JSON, and memory store. |

---

## 🤝 Contributing

`shubh-plan-open` operates under a source-available / read-only open-source model. Please see the [CONTRIBUTING.md](CONTRIBUTING.md) guide for details.

---

## 📄 License

This open-source project is licensed under the **Apache License 2.0**. Copyright 2026 Ekarna Interactive Technology LLP. See the [LICENSE](LICENSE) file for details.
