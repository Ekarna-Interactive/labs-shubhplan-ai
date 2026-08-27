# Shubh Plan Web — Deployment & Operations Guide

This guide details how to configure, test, containerize, and deploy **Shubh Plan Web** across local servers and cloud platforms like **Fly.io**.

---

## ⚙️ Operating Modes

**Shubh Plan Web** supports two execution modes controlled via the `APP_MODE` environment variable:

| Mode | Environment Variable | Target Environment | API Key Source | User Auth & Sessions |
| :--- | :--- | :--- | :--- | :--- |
| **Server Mode** | `APP_MODE=server` | Local / Private Server | System `.env` (`GEMINI_API_KEY`) | Shared or Authenticated |
| **Demo Mode** | `APP_MODE=demo` | Fly.io Public Cloud | Client Browser (`localStorage` / Header) | Isolated Sessions / Instant Guest Demo |

---

## 💻 1. Local Running & Testing

### Prerequisites
- **Go**: 1.24+ or 1.25+ installed on your system.
- **Gemini API Key**: (Optional for local testing, required for live AI generation).

### Quick Local Start
1. Navigate to the project folder:
   ```bash
   cd apps/shubh-plan-web
   ```
2. Create a `.env` file in `apps/shubh-plan-web`:
   ```env
   PORT=3000
   APP_MODE=server
   GEMINI_API_KEY=your_gemini_api_key_here
   GOOGLE_MAPS_API_KEY=your_google_maps_api_key_here
   ```
3. Run the application:
   ```bash
   go run main.go
   ```
4. Access the web interface at [`http://localhost:3000`](http://localhost:3000).

---

## 🔑 2. Demo Mode & Bring Your Own Key (BYOK)

In **Demo Mode** (`APP_MODE=demo`), the server runs as an open public demonstration:
- **Client Key Storage**: Users enter their Gemini API key into the top-bar **"🔑 API Key Settings"** modal.
- **Privacy & Security**: The key is stored **strictly in browser `localStorage`** and transmitted per request via the `X-Gemini-API-Key` HTTP header. It is never saved to the server database.
- **Free Key Acquisition Instructions**:
  1. Go to [Google AI Studio (aistudio.google.com)](https://aistudio.google.com/app/apikey).
  2. Click **"Create API Key"** (free tier available).
  3. Paste the key into the Shubh Plan Web settings modal and click **"Save to Browser"**.

---

## ☁️ 3. Deploying to Fly.io

### Step 1: Install Fly CLI & Authenticate
Install `flyctl` if you haven't already:
```bash
# macOS / Linux
curl -L https://fly.io/install.sh | sh

# Windows (PowerShell)
powershell -Command "iwr https://fly.io/install.ps1 -useb | iex"
```

Authenticate with Fly.io:
```bash
fly auth login
```

### Step 2: Initialize Fly Application
Inside `apps/shubh-plan-web`, launch or link your application name matching `fly.toml`:
```bash
fly launch --name shubh-plan-demo --region sin --no-deploy
```

Verify your `fly.toml` configuration:
```toml
app = "shubh-plan-demo"
primary_region = "sin"

[build]
  dockerfile = "Dockerfile"

[env]
  PORT = "8080"
  APP_MODE = "demo"
  SHUBH_DATA_DIR = "/app/data"

[mounts]
  source = "shubh_plan_web_data"
  destination = "/app/data"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = "off"
  auto_start_machines = true
  min_machines_running = 1
  processes = ["app"]

  [[http_service.checks]]
    grace_period = "5s"
    interval = "10s"
    method = "GET"
    path = "/api/health"
    timeout = "2s"

[[vm]]
  size = "shared-cpu-1x"
  memory = "512mb"
```

### Step 3: Create Persistent Storage Volume
To ensure user accounts, guest rosters, and invitation card artwork persist across server restarts, create a Fly volume:
```bash
fly volume create shubh_plan_web_data --region sin --size 1
```

### Step 4: Deploy Container
Deploy the multi-stage Docker container to Fly.io:
```bash
fly deploy
```

### Step 5: Monitor & Verify
Verify application status and logs:
```bash
# Check running instances
fly status

# Stream live server logs
fly logs

# Open deployed web app in browser
fly open
```

---

## 🐳 4. Docker Container Setup (Local Docker / Self-Hosted)

Build and run using Docker locally or on your own VPS:

```bash
cd apps/shubh-plan-web

# Build multi-stage Docker image
docker build -t shubh-plan-web:latest .

# Run container in Demo Mode on port 8080
docker run -d -p 8080:8080 --name shubh-web -e APP_MODE=demo shubh-plan-web:latest
```
