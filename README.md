# ✨ Shubh Plan AI — Open-Source Multi-Agent Event Engine

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![Google ADK v2](https://img.shields.io/badge/Google%20ADK-v2.2.0-4285F4?style=for-the-badge&logo=google&logoColor=white)](https://google.golang.org/adk)
[![Honcho v3 REST](https://img.shields.io/badge/Honcho%20Memory-v3%20REST-7C3AED?style=for-the-badge)](https://honcho.dev)
[![Wish SSH TUI](https://img.shields.io/badge/Wish%20SSH-Bubble%20Tea-FF4081?style=for-the-badge)](https://github.com/charmbracelet/wish)
[![HTMX Web UI](https://img.shields.io/badge/HTMX-Tailwind%20CSS-38BDF8?style=for-the-badge&logo=htmx&logoColor=white)](https://htmx.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=for-the-badge&logo=apache)](LICENSE)

**An AI-Native, Single-Binary Multi-Agent Event Engine & Invitation Design Workspace built with Google ADK v2, Honcho Memory, Wish SSH TUI, and HTMX.**

[Quickstart](#-quickstart) • [Architecture](#-architecture) • [Features](#-key-features) • [User Interfaces](#-user-interfaces) • [Multi-User Security](#-per-user-privacy--key-isolation) • [Environment Setup](#%EF%B8%8F-environment-configuration)

</div>

---

## 🌟 Key Features

* **⚡ Unified Single Go Binary**: Operates 100% locally in Go with zero `node_modules` or JavaScript runtime requirements.
* **🔑 Mandatory Key Onboarding & Multi-User Key Privacy**: First-time setup gate for both Web UI and SSH TUI. Web keys are stored safely inside browser `localStorage` and passed via request headers (`X-Gemini-API-Key`), while SSH keys live strictly in connection RAM. Global server `.env` files are never overwritten in multi-user mode.
* **📋 Automatic Event Profile Setup & 1:1 Live Web/TUI Sync**: Verification check for `event_details.md` right after key setup. Real-time pre-filling of form inputs across Web UI and SSH TUI.
* **🗓️ Interactive Calendar & Machine-Readable ISO Date Normalization**: Click-to-select graphical calendar date picker in Web UI, and robust multi-format date parser (`ParseAndNormalizeMachineDate`) in TUI converting user inputs into ISO 8601 `YYYY-MM-DD` strings.
* **🪄 AI Welcome Subheader Generator**: Gemini AI-powered 4-style invitation welcome message copywriter (`/api/suggest-welcome`) with one-click insertion.
* **🌙 Light Mode & Dark Mode Switcher**: High-contrast theme toggle with persistent `localStorage` theme preference.
* **🤖 Inbuilt Lite Google ADK Engine**: Built natively with official [`google.golang.org/adk/v2/agent`](https://google.golang.org/adk) package across 5 specialized subagents:
  * `MasterOrchestrator`: Multi-agent supervisor and task router.
  * `GuestConcierge`: Guest RSVP management, dietary requirements & transport logistics.
  * `TimelineAgent`: Chronological ceremony scheduling & dress codes.
  * `BudgetAgent`: Spend metrics & industry category allocations.
  * `VendorAgent`: Catering, photography, and decor vendor coordination.
* **🧠 Honcho v3 Cloud & Local Memory**: Native HTTP REST driver for [`api.honcho.dev/v3`](https://honcho.dev) with automatic fallback to zero-dependency local JSON store (`./data/honcho_memory.json`).
* **🔐 Dynamic Session Secret Handshake**: HMAC ephemeral token authentication (`POST /api/v1/session/handshake`) issuing 24-hour bearer tokens for all SSE streams.
* **💻 Wish SSH Terminal UI (`:2222`)**: Zero-client install terminal experience built with `github.com/charmbracelet/wish` and `bubbletea`.
* **🌐 HTMX Web UI (`:3000`)**: Single-page responsive web dashboard powered by HTMX + Tailwind CSS matching the full MagicPath design system.

---

## 🏗 Architecture

```plaintext
┌──────────────────────────────────────────────────────────────────────────────────┐
│                    UNIFIED OPEN-SOURCE BINARY (shubh-plan-open)                   │
├────────────────────────────────────────┬─────────────────────────────────────────┤
│    ⚡ WISH SSH TUI SERVER (Port 2222)   │      🌐 HTMX WEB UI SERVER (Port 3000) │
│    • Bubble Tea Model & Keybindings    │      • Tailwind CSS + HTMX Fragments    │
│    • Real-time Viewports & Wizards     │      • 4 Tabs matching MagicPath UI    │
│    • Connection RAM Key Isolation      │      • Browser LocalStorage Keys       │
├────────────────────────────────────────┴─────────────────────────────────────────┤
│              INBUILT LITE GO ADK AGENT ENGINE & HONCHO v3 REST DRIVER            │
│   • Inbuilt Multi-Agent Reasoning (Orchestrator, Timeline, Vendor, Budget, RSVP) │
│   • Honcho v3 HTTP REST Driver (Direct `https://api.honcho.dev/v3/` API calls)   │
│   • Inbuilt Local Memory Store Fallback (`data/honcho_memory.json` & `guests.md`)│
│   • Real-Time SSE Token Streamer (`/api/v1/orchestrator/stream`)                 │
│   • Ephemeral Session Secret Handshake (`/api/v1/session/handshake`)             │
│   • Direct Gemini LLM & Imagen 3 API Drivers (`generator/prompter.go`)           │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## 🛡 Per-User Privacy & Key Isolation

When deployed to a shared server, `shubh-plan-open` guarantees strict per-user key privacy:

1. **Web UI Isolation**: Keys are saved strictly inside each user's browser `localStorage`. When User A closes their tab and reopens the site, their saved key auto-unlocks their workspace. User B accessing the server from their own browser will see an empty onboarding modal prompting for User B's key. User A's key is never written to disk or exposed to User B.
2. **SSH TUI Isolation**: Each SSH connection runs in an isolated in-memory Go process. Keys entered during an SSH session live in RAM for that connection only. Supporting clients can also pass keys via SSH `SendEnv GEMINI_API_KEY`.

---

## 🖥 User Interfaces

### 1. 🌐 HTMX Web Dashboard (`http://localhost:3000`)
Features 4 interactive workspace tabs:
1. **✨ AI Invitation Studio**: Event parameters, aesthetic theme presets (South Indian Gold, Mughal Heritage, Paper Cut 3D, Minimalist), Gemini prompt compilation, and direct `event_details.md` save button.
2. **👥 Guest Roster & RSVPs**: Real-time headcount meters (Confirmed, Pending, Declined), dietary breakdown chips, interactive guest table, and CSV download.
3. **📅 Event Timeline**: Chronological event session cards (Haldi, Sangeet, Muhurtham, Reception).
4. **🤖 AI Concierge Copilot**: Real-time multi-agent chat interface connected via Server-Sent Events (SSE).

### 2. 💻 Wish SSH Terminal UI (`ssh -p 2222 localhost`)
Interactive terminal application supporting live tab switching, keybindings, RSVP wizards (`/add-rsvp`), event profile editing (`/planner`, `/event`), and key setup prompts.

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
  GEMINI_API_KEY="AIzaSy..." HONCHO_API_KEY="hnc_..." ssh -i ~/.ssh/id_ed25519 -o SendEnv=GEMINI_API_KEY -o SendEnv=HONCHO_API_KEY -p 2222 shubh-plan-ai.fly.dev
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

#### 🖼️ SSH Live Web Design Preview & Port Forwarding

SSH terminal users can preview their generated high-resolution invitation cards live on the Web UI:

* **Method 1: Direct Web Preview URL (Production / Fly.io)**:
  Open `http://shubh-plan-ai.fly.dev:3000` (or your deployment domain) in your browser. Any design generated with `/generate` in the TUI automatically populates in real-time in the **🖼️ Generated Invitation Cards** gallery.

* **Method 2: SSH Local Port Forwarding (Recommended for Remote/Private Hosts)**:
  Tunnel the remote Web Preview port (`3000`) to your local machine using the SSH `-L` flag:
  ```bash
  ssh -L 3000:localhost:3000 -i ~/.ssh/id_ed25519 -p 2222 shubh-plan-ai.fly.dev
  ```
  Then open [`http://localhost:3000`](http://localhost:3000) in your local browser. Whenever you execute `/generate` in your SSH terminal, your local browser auto-refreshes and displays the artwork with interactive **👁️ Full Design Lightbox** preview and **📥 PNG Download**!

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
| `SHUBH_DATA_DIR` | `./data` | Local directory for SQLite database, `guests.md`, and memory JSON. |

---

## 🤝 Contributing

`shubh-plan-open` operates under a source-available / read-only open-source model. Please see the [CONTRIBUTING.md](CONTRIBUTING.md) guide for details.

---

## 📄 License

This open-source project is licensed under the **Apache License 2.0**. Copyright 2026 Ekarna Interactive Technology LLP. See the [LICENSE](LICENSE) file for details.
