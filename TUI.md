# 💻 Wish SSH Terminal UI (TUI) Guide

`shubh-plan-open` includes an in-memory, zero-client-install Terminal User Interface (TUI) powered by [`github.com/charmbracelet/wish`](https://github.com/charmbracelet/wish) and [`bubbletea`](https://github.com/charmbracelet/bubbletea).

---

## 🔑 Authentication & Connection Methods

Wish SSH relies on standard SSH key pairs (`~/.ssh/id_ed25519` or `~/.ssh/id_rsa`).

### 1. Direct SSH Connection (Interactive Key Prompt)

Connect directly to your deployment. If API keys are not supplied via environment variables, the application will prompt you on-screen to enter your Google Gemini and Honcho API keys securely into connection RAM:

```bash
ssh -i ~/.ssh/id_ed25519 -p 2222 shubh-plan-open.fly.dev
```

---

### 2. Auto-Authentication with `SendEnv`

Pass your API keys directly from your terminal session without on-screen prompts:

#### On Linux / macOS (Bash / Zsh):
```bash
GEMINI_API_KEY="......" HONCHO_API_KEY="....." ssh -i ~/.ssh/id_ed25519 -o SendEnv=GEMINI_API_KEY -o SendEnv=HONCHO_API_KEY -p 2222 shubh-plan-open.fly.dev
```

#### On Windows (PowerShell):
```powershell
$env:GEMINI_API_KEY="......"; $env:HONCHO_API_KEY="....."; ssh -i ~/.ssh/id_ed25519 -o SendEnv=GEMINI_API_KEY -o SendEnv=HONCHO_API_KEY -p 2222 shubh-plan-open.fly.dev
```

---

### 3. SSH Config Setup for One-Click Login

Add a custom host block to your `~/.ssh/config` file (`C:\Users\Gokul\.ssh\config` on Windows or `~/.ssh/config` on Linux/macOS):

```sshconfig
Host shubh-open
    HostName shubh-plan-open.fly.dev
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
    SendEnv GEMINI_API_KEY HONCHO_API_KEY
```

#### Set up `~/.ssh/config` via PowerShell:
```powershell
@'
Host shubh-open
    HostName shubh-plan-open.fly.dev
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
    SendEnv GEMINI_API_KEY HONCHO_API_KEY
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
- `Ctrl+C` / `Esc`: Exit TUI session or close active overlay modal
- `Enter`: Submit form field / execute command

### Built-in Slash Commands
- `/planner`: Configure host names, event role, and planner details.
- `/event`: Edit event date, venue, welcome subheader, and theme style.
- `/add-rsvp`: Open interactive guest RSVP registration modal.
- `/generate`: Compile multi-style AI prompts and trigger design generation.
- `/keys`: Manage or update session API keys in RAM.

---

## 🖼️ Live Web Preview over SSH Tunnel

To preview generated invitation artwork in your local browser while connected via SSH:

```bash
ssh -L 3000:localhost:3000 -p 2222 shubh-open
```

Then open [`http://localhost:3000`](http://localhost:3000) in your browser. Any artwork generated in the TUI session will render live in the browser gallery!
