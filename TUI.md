# 💻 Wish SSH Terminal UI (TUI) Guide

`shubh-plan-open` includes an in-memory, zero-client-install Terminal User Interface (TUI) powered by [`github.com/charmbracelet/wish`](https://github.com/charmbracelet/wish) and [`bubbletea`](https://github.com/charmbracelet/bubbletea).

---

## 🔑 Authentication & Connection Methods

Wish SSH relies on standard SSH key pairs (`~/.ssh/id_ed25519` or `~/.ssh/id_rsa`).

### 1. Direct SSH Connection (Interactive 3-Step Onboarding Wizard)

Connect directly to your deployment. If API keys are missing from environment variables, the Wish SSH server runs a secure 3-step setup wizard in session RAM:

1. **Step 1/3**: `GEMINI_API_KEY` (Required for live AI generation & copilot chat)
2. **Step 2/3**: `GOOGLE_PLACES_API_KEY` (Optional for live Google Places venue search. Press Enter to use Gemini AI Venue Agent)
3. **Step 3/3**: `HONCHO_API_KEY` (Optional for Honcho Cloud Memory sync. Press Enter to use Inbuilt Local Store)

```bash
ssh -i ~/.ssh/id_ed25519 -p 2222 shubh-plan-open.fly.dev
```

---

### 2. Auto-Authentication with `SendEnv`

Pass your API keys directly from your terminal session without on-screen prompts:

#### On Linux / macOS (Bash / Zsh):
```bash
GEMINI_API_KEY="......" GOOGLE_PLACES_API_KEY="......" HONCHO_API_KEY="....." ssh -i ~/.ssh/id_ed25519 -o SendEnv=GEMINI_API_KEY -o SendEnv=GOOGLE_PLACES_API_KEY -o SendEnv=HONCHO_API_KEY -p 2222 shubh-plan-open.fly.dev
```

#### On Windows (PowerShell):
```powershell
$env:GEMINI_API_KEY="......"; $env:GOOGLE_PLACES_API_KEY="......"; $env:HONCHO_API_KEY="....."; ssh -i ~/.ssh/id_ed25519 -o SendEnv=GEMINI_API_KEY -o SendEnv=GOOGLE_PLACES_API_KEY -o SendEnv=HONCHO_API_KEY -p 2222 shubh-plan-open.fly.dev
```

---

### 3. SSH Config Setup for One-Click Login

Add a custom host block to your `~/.ssh/config` file (`C:\Users\Gokul\.ssh\config` on Windows or `~/.ssh/config` on Linux/macOS):

```sshconfig
Host shubh-open
    HostName shubh-plan-open.fly.dev
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
    SendEnv GEMINI_API_KEY GOOGLE_PLACES_API_KEY HONCHO_API_KEY
```

#### Set up `~/.ssh/config` via PowerShell:
```powershell
@'
Host shubh-open
    HostName shubh-plan-open.fly.dev
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
    SendEnv GEMINI_API_KEY GOOGLE_PLACES_API_KEY HONCHO_API_KEY
'@ | Set-Content -Path "$HOME\.ssh\config"
```

Once configured, connect anytime with:
```bash
ssh shubh-open
```

---

## 🎮 TUI Navigation & Slash Commands

Inside the Wish SSH TUI session, navigate using keyboard shortcuts and slash commands:

### Keyboard Shortcuts
- `Tab` / `Shift+Tab`: Switch workspace tabs (Studio, Guests, Timeline, Copilot)
- `↑` / `↓`: Navigate interactive option lists (Styles, Resolutions, Google Places predictions)
- `Enter`: Submit selection / confirm input
- `Ctrl+C` / `Esc`: Exit TUI session or close active overlay modal

### Built-in Slash Commands
- `/event`: Display current active event profile, address, maps link, directions, budget, and command instructions.
- `/event new` (or `/wizard`): Launch interactive step-by-step profile creation wizard.
- `/event update` (or `/edit`): Modify existing event profile details.
- `/venue <query>` (or `/vendor <query>`): Search Google Places & AI Autocomplete predictions in real-time, navigate with arrow keys (`↑`/`↓`), and select a venue to update your event profile and save all 7 place details!
- `/add-rsvp`: Open interactive guest RSVP registration wizard.
- `/budget <amount>`: Update total estimated event budget.
- `/currency <code>`: Set default currency (e.g. `INR`, `USD`, `EUR`, `GBP`).
- `/generate`: Compile multi-style AI prompts and trigger design generation.
- `/preview`: Open public web preview URL.
- `/config`: Manage or update session API keys in RAM.

---

## 🖼️ Live Web Preview over SSH

When connected to Fly.io or a self-hosted server, generating invitation artwork in the SSH TUI outputs a direct web preview URL (e.g. `https://shubh-plan-open.fly.dev?sessionID=...`). 

Opening that URL in any browser instantly renders the artwork and opens the live Invitation Studio gallery!
