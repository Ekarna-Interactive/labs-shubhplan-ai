# ✨ Shubh Plan AI — Open-Source Multi-Agent Event Engine

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![Google ADK v2](https://img.shields.io/badge/Google%20ADK-v2.2.0-4285F4?style=for-the-badge&logo=google&logoColor=white)](https://google.golang.org/adk)
[![Honcho v3 REST](https://img.shields.io/badge/Honcho%20Memory-v3%20REST-7C3AED?style=for-the-badge)](https://honcho.dev)
[![Wish SSH TUI](https://img.shields.io/badge/Wish%20SSH-Bubble%20Tea-FF4081?style=for-the-badge)](https://github.com/charmbracelet/wish)
[![HTMX Web UI](https://img.shields.io/badge/HTMX-Tailwind%20CSS-38BDF8?style=for-the-badge&logo=htmx&logoColor=white)](https://htmx.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=for-the-badge&logo=apache)](LICENSE)

**Shubh Plan AI is an all-in-one event planning workspace and invitation studio. It helps hosts and event planners design personalized invitation artwork, structure ceremony schedules, manage guest RSVPs and dietary requirements, and coordinate every celebration detail through an intuitive AI copilot.**

[Quickstart](#-quickstart) • [Architecture](#-architecture) • [Features](#-key-features) • [User Authentication](#-gitea-style-native-go-authentication) • [Data Persistence](#-unified-data-persistence-data) • [User Interfaces](#-user-interfaces) • [TUI Guide](TUI.md) • [Deployment Guide](DEPLOYMENT.md) • [Environment Setup](#%EF%B8%8F-environment-configuration)

</div>

---

## 🌟 Key Features

* **⚡ Unified Single Go Binary**: Operates 100% locally in Go with zero `node_modules` or JavaScript runtime requirements.
* **🔐 Gitea-Style Native Go Authentication**: Zero-dependency user management engine storing Argon2id password hashes and SSH public keys in `data/users.json` with POSIX `0600` permissions. Features HTTP-Only 7-day session cookies (`shubh_session`) and First-Time Owner Registration.
* **🎉 Guided First-Time Setup Wizards**:
  * **👑 Owner Setup Wizard**: Prompts on fresh boot to create the primary Admin/Owner account.
  * **📋 Event Setup Wizard**: Prompts to configure event type, hosts, date, venue, currency, and welcome message whenever `event_details.md` or `data/event-details.json` is missing.
  * **🔑 API Key Onboarding Wizard**: Guided popup for entering Google Gemini and optional Honcho keys.
* **📦 Unified JSON Storage & 1:1 Web/TUI Synchronization**: Shared data stores under `./data/` (`event-details.json`, `itinerary.json`, `rsvps.json`, `users.json`). Any update in Web UI (e.g. adding a ceremony sub-event or updating guest headcount) immediately reflects in the SSH TUI and vice versa.
* **🧩 Hypermedia-Driven HTMX & Middleware Architecture**: Operates on a pure Go + HTMX engine with Go middleware (`HTMXMiddleware`, `APIKeyContextMiddleware`, `RequireAuthHTMX`), Out-of-Band (`hx-swap-oob`) multi-target swaps, and server event triggers (`HX-Trigger`).
* **🔄 2-Way Live Real-Time Controls Synchronization**: Executing slash commands (`/aspect`, `/style`, `/welcome`, `/currency`) in Copilot chat instantly synchronizes Tab 1 (AI Invitation Studio) sidebar dropdown controls, `data/event-details.json`, and `event_details.md` in 100% real time.
* **💬 Interactive Copilot Chat & Dynamic Autocomplete**: Real-time slash command autocomplete popup that filters as you type (`/sugg`, `/asp`, `/sty`), rendering rich, clickable option buttons directly in chat for `/suggest`, `/style`, `/aspect`, and `/currency`.
* **🗓️ Ordinal Date Typography & Smart Banner Prompt Compiler**: Automatic date parser formatting dates into human-readable ordinal strings (`September 20th, 2026`) on generated invitation cards, with anti-blank prompt sanitization guaranteeing text is printed directly inside central banner plaques.
* **🗓️ Interactive Calendar & Date Normalization**: Click-to-select graphical calendar date picker in Web UI, and robust multi-format date parser (`ParseAndNormalizeMachineDate`) in TUI converting user inputs into machine-readable ISO 8601 `YYYY-MM-DD` strings.
* **🌙 Light Mode & Dark Mode Switcher**: Ultra-crisp high-contrast theme toggle with persistent `localStorage` theme preference.
* **🤖 Inbuilt Lite Google ADK Engine**: Built natively with official [`google.golang.org/adk/v2/agent`](https://google.golang.org/adk) package across 5 specialized subagents:
  * `AI Concierge`: Multi-agent supervisor and task router.
  * `GuestConcierge`: Guest RSVP management, dietary requirements & transport logistics.
  * `TimelineAgent`: Chronological ceremony scheduling & dress codes.
  * `BudgetAgent`: Spend metrics & industry category allocations.
  * `VendorAgent`: Catering, photography, and decor vendor coordination.
* **🧠 Smart Memory (Honcho v3 Cloud & Local Memory)**: Native HTTP REST driver for [`api.honcho.dev/v3`](https://honcho.dev) with automatic fallback to zero-dependency local JSON store (`./data/honcho_memory.json`).
* **💻 Wish SSH Terminal UI (`:2222`)**: Zero-client install terminal experience built with `github.com/charmbracelet/wish` and `bubbletea` with native SSH public key mapping.
* **🌐 HTMX Web UI (`:3000`)**: Single-page responsive web dashboard powered by HTMX + Tailwind CSS.

---

## 🏗 Architecture

```plaintext
┌──────────────────────────────────────────────────────────────────────────────────┐
│                    UNIFIED OPEN-SOURCE BINARY (shubh-plan-open)                   │
├────────────────────────────────────────┬─────────────────────────────────────────┤
│    ⚡ WISH SSH TUI SERVER (Port 2222)   │      🌐 HTMX WEB UI SERVER (Port 3000) │
│    • Bubble Tea Model & Keybindings    │      • Tailwind CSS + HTMX Components   │
│    • SSH Public Key User Auto-Mapping  │      • Modular Component Partials        │
│    • Real-time Viewports & Wizards     │      • Browser LocalStorage Keys       │
├────────────────────────────────────────┴─────────────────────────────────────────┤
│         UNIFIED POSIX DATA PERSISTENCE & NATIVE ARGON2id AUTH ENGINE             │
│   • Auth Store (`data/users.json` with POSIX 0600 file permissions)               │
│   • Event Store (`data/event-details.json` & 2-Way Synced `event_details.md`)    │
│   • Timeline Store (`data/itinerary.json` for Ceremony Sub-Events)              │
│   • Guest Store (`data/rsvps.json` for Headcounts & Dietary Requirements)        │
│   • Smart Memory (`data/honcho_memory.json` Local Store & Honcho v3 Cloud)       │
├──────────────────────────────────────────────────────────────────────────────────┤
│              INBUILT LITE GO ADK AGENT ENGINE & HONCHO v3 REST DRIVER            │
│   • AI Concierge & Sub-Agents (Timeline, Vendor, Budget, Guest Concierge)        │
│   • Real-Time SSE Token Streamer (`/api/v1/orchestrator/stream`)                 │
│   • Ephemeral Session Secret Handshake (`/api/v1/session/handshake`)             │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## 🔐 Gitea-Style Native Go Authentication

`shubh-plan-open` includes a native, self-hosted Go authentication system:

1. **First-Time Owner Registration**: On fresh installation, the web UI automatically presents the Owner Setup Wizard (`#owner-setup-modal`) to register the primary Admin/Owner account.
2. **Argon2id Password Hashing**: Passwords are saved with secure salt and key parameters (`time=1, memory=64MB, threads=4`) in `/app/data/users.json`.
3. **HTTP-Only Session Cookies**: Authenticated sessions issue `shubh_session` cookies expiring in 7 days.
4. **Wish SSH Public Key Mapping**: Users can register SSH public keys (`ssh-ed25519 ...`) to auto-authenticate over SSH port 2222 without entering passwords.

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

#### 🔑 SSH Client Public Key Authentication & Connection Methods

Wish SSH uses standard SSH public key cryptography (`~/.ssh/id_ed25519` / `~/.ssh/id_rsa`).

* **Step 1: Ensure you have an SSH Key Pair**:
  ```bash
  # Generate an Ed25519 SSH key pair if you haven't already
  ssh-keygen -t ed25519 -C "your_email@example.com"
  ```

* **Step 2A: Interactive Connection (On-Screen Key Prompt)**:
  ```bash
  ssh -i ~/.ssh/id_ed25519 -p 2222 shubh-plan-ai.fly.dev
  ```
  *Prompts on-screen for Gemini and optional Honcho keys. Keys live in RAM for that connection only.*

* **Step 2B: Direct One-Liner Connection (`SendEnv` Auto-Authentication)**:
  ```bash
  GEMINI_API_KEY="......" HONCHO_API_KEY="....." ssh -i ~/.ssh/id_ed25519 -o SendEnv=GEMINI_API_KEY -o SendEnv=HONCHO_API_KEY -p 2222 shubh-plan-ai.fly.dev
  ```

* **Step 3 (Optional): Add to `~/.ssh/config` for Instant Login**:
  ```sshconfig
  Host shubh
      HostName shubh-plan-ai.fly.dev
      Port 2222
      IdentityFile ~/.ssh/id_ed25519
      SendEnv GEMINI_API_KEY HONCHO_API_KEY
  ```
  Then connect anytime with a simple command: `shubh`

---

## 🚀 Quickstart

### Option A: Run with Docker Compose (Recommended)

```bash
# 1. Clone the public repository
git clone https://github.com/Ekarna-Interactive/labs-shubhplan-ai.git
cd labs-shubhplan-ai

# 2. Launch with Docker Compose
docker compose up -d
```

### Option B: Build from Source (Go 1.22+)

1. **Clone & enter repository**:
   ```bash
   git clone https://github.com/Ekarna-Interactive/labs-shubhplan-ai.git
   cd labs-shubhplan-ai
   ```

2. **Configure environment variables (Optional for server mode)**:
   ```bash
   cp .env.example .env
   ```

3. **Build and launch binary**:
   ```bash
   # Build single Go binary
   go build -o shubh-plan-open .

   # Launch in server mode (Web UI + SSH TUI Server)
   ./shubh-plan-open --server
   ```

4. **Access the application**:
   * **Web Dashboard**: Open [`http://localhost:3000`](http://localhost:3000)
   * **Wish SSH Terminal**: Run `ssh -i ~/.ssh/id_ed25519 -p 2222 localhost`

---

## ⚙️ Environment Configuration

| Variable | Default | Description |
|---|---|---|
| `GEMINI_API_KEY` | *(Optional Server Default)* | Server-wide Google Gemini API Key. Web/SSH users can override with their own keys. |
| `GEMINI_TEXT_MODEL` | `gemini-3.5-flash` | Primary text model for prompt compilation & copilot chat. |
| `GEMINI_IMAGE_MODEL` | `gemini-3.1-flash-image` | Model for invitation card artwork generation. |
| `HONCHO_API_KEY` | *(Optional)* | Honcho Cloud Memory key for sync with `api.honcho.dev/v3`. Defaults to local JSON store if empty. |
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
