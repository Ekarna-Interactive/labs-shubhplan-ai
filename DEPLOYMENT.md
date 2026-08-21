# 🚀 Deployment Guide for Shubh Plan AI (`shubh-plan-open`)

This document provides a comprehensive guide for deploying **Shubh Plan AI Open Source** (`shubh-plan-open`) to [Fly.io](https://fly.io) or any cloud infrastructure, configuring network ports, managing IP addresses, and enabling terminal (SSH) & web access.

---

## 📋 Prerequisites

- **Fly CLI (`flyctl` / `fly`)**: Installed and logged in (`fly auth login`).
- **Go 1.24+ / Docker**: For local building and testing before deployment.
- **SSH Key Pair**: Local `~/.ssh/id_ed25519` key pair for TUI authentication.

---

## 🎯 Deployment Modes Comparison

| Feature / Behavior | 🏠 Self-Hosted Production (`DEMO_MODE=false`) | 🌐 Public Demo Instance (`DEMO_MODE=true`) |
| :--- | :--- | :--- |
| **Primary Use Case** | Private hosting for your own events & planning team. | Public portfolio / showcase demo for reviewers. |
| **First-Time Boot** | Presents **👑 Owner Setup Wizard** (`#owner-setup-modal`) to create your custom admin account. | Auto-seeds demo credentials (`admin@shubhplan.ai` / `shubh2026`). |
| **Client API Key Storage** | Stored strictly in user browser `localStorage` (Web) / RAM (SSH). | Stored strictly in user browser `localStorage` (Web) / RAM (SSH). |
| **`fly.toml` Setting** | `DEMO_MODE = "false"` | `DEMO_MODE = "true"` |

---

## 🛠️ Step-by-Step Fly.io Deployment

### 1. Initialize the Fly App

From the project root directory, create the Fly application:

```bash
fly apps create shubh-plan-open
```

### 2. Configure `fly.toml`

Choose your deployment mode in `fly.toml`:

#### Option A: Self-Hosted Production Deployment (`DEMO_MODE="false"` — Recommended)
```toml
app = "shubh-plan-open"
primary_region = "sin"

[build]
  dockerfile = "Dockerfile"

[env]
  PORT = "3000"
  SSH_PORT = "2222"
  SERVER_MODE = "true"
  MULTI_USER = "true"
  DEMO_MODE = "false"
  SHUBH_DATA_DIR = "/app/data"
  SHUBH_OUTPUT_DIR = "/app/output"

[mounts]
  source = "shubh_data"
  destination = "/app/data"

[http_service]
  internal_port = 3000
  force_https = true
  auto_stop_machines = "off"
  auto_start_machines = true
  min_machines_running = 1
  processes = ["app"]

  [[http_service.checks]]
    grace_period = "5s"
    interval = "10s"
    method = "GET"
    path = "/"
    timeout = "2s"

[[vm]]
  size = "shared-cpu-1x"
  memory = "512mb"

[[services]]
  internal_port = 2222
  protocol = "tcp"
  auto_stop_machines = "off"
  auto_start_machines = true

  [[services.ports]]
    port = 2222
```

#### Option B: Public Showcase Demo Deployment (`DEMO_MODE="true"`)
Set `DEMO_MODE = "true"` in the `[env]` section of `fly.toml` to automatically pre-configure `admin@shubhplan.ai` / `shubh2026` on boot:

```toml
[env]
  PORT = "3000"
  SSH_PORT = "2222"
  SERVER_MODE = "true"
  MULTI_USER = "true"
  DEMO_MODE = "true"
  SHUBH_DATA_DIR = "/app/data"
  SHUBH_OUTPUT_DIR = "/app/output"
```

---

### 3. Create Persistent Volume for Data Storage (`/app/data`)

To ensure user accounts (`data/users.json`), event profiles, ceremony itineraries, and guest RSVPs persist across machine restarts and redeployments, create a 1GB Fly volume:

```bash
fly volumes create shubh_data --size 1 -a shubh-plan-open
```

Ensure your `fly.toml` references this volume:

```toml
[mounts]
  source = "shubh_data"
  destination = "/app/data"
```

---

### 4. 🔐 Set Fly Secrets & SSH Host Key

#### A. Generate & Set Persistent SSH Host Key (`SSH_HOST_KEY`)
To prevent client terminal warnings (`WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!`) across redeployments, set a persistent SSH server host key secret:

```bash
# 1. Generate an Ed25519 host key locally
ssh-keygen -t ed25519 -N "" -f /tmp/ssh_host_ed25519_key

# 2. Set SSH_HOST_KEY secret on Fly.io
fly secrets set SSH_HOST_KEY="$(cat /tmp/ssh_host_ed25519_key)" -a shubh-plan-open
```

#### B. Set Optional Server-Wide Default API Keys
If you wish to provide server-wide default API keys for demo users:

```bash
fly secrets set GEMINI_API_KEY="your_gemini_key" GOOGLE_PLACES_API_KEY="your_places_key" HONCHO_API_KEY="your_honcho_key" -a shubh-plan-open
```

---

### 5. Networking & IP Allocation (Important for SSH)

Fly.io assigns a **Shared IPv4 address** by default. Shared IPv4 only proxies HTTP/HTTPS traffic (ports 80 and 443). **Custom TCP ports like SSH (`2222`) require a Dedicated IP.**

#### Option A: Allocate a Dedicated IPv4 ($2/mo — Recommended for direct SSH)
```bash
fly ips allocate-v4 -a shubh-plan-open
```

Then release the old shared IPv4 address to avoid DNS conflicts:
```bash
fly ips release <SHARED_IPV4_ADDRESS> -a shubh-plan-open
```

#### Option B: Use `fly proxy` (Free Alternative)
If you prefer not to allocate a dedicated IPv4 address, you can tunnel port 2222 locally:
```bash
fly proxy 2222:2222 -a shubh-plan-open
```
And connect via `localhost:2222`.

---

### 6. Deploy the Container

Deploy the application to Fly.io:

```bash
fly deploy
```

---

### 7. Scale Machine Count

By default, Fly launches 2 machines for high availability. To scale down to 1 machine:

```bash
fly scale count 1
```

---

## 🔑 Key Privacy & Storage Architecture on Fly.io

When deployed in `MULTI_USER="true"` mode, `shubh-plan-open` enforces strict client key privacy:

1. **🌐 Web UI (`:3000`)**:
   - **Key Storage**: User keys are stored **strictly in the user's browser `localStorage`** on their local device.
   - **Request Transmission**: Sent to Fly.io over encrypted HTTPS via custom request headers (`X-Gemini-API-Key`, `X-Places-API-Key`, `X-Honcho-API-Key`).
   - **Server Memory**: The Go backend reads them in HTTP request memory to fulfill the AI call and **never writes them to server disk or `.env` files**.

2. **💻 Wish SSH TUI (`:2222`)**:
   - **Key Storage**: User keys live in temporary Go connection goroutine memory (RAM) for the active SSH session.
   - **Instant Purge**: The moment the user disconnects or closes their terminal, the keys are **instantly purged from server RAM**.

3. **Key Generation Links**:
   - **Google Gemini API Key**: [aistudio.google.com/api-keys](https://aistudio.google.com/api-keys) *(Free 1-click generation with Google account)*.
   - **Google Places API Key**: [console.cloud.google.com](https://console.cloud.google.com/google/maps-apis/credentials).
   - **Honcho API Key**: Optional *(defaults to zero-config local vector store `./data/honcho_memory.json`)*.

---

## 🔍 Troubleshooting & Logs

- **View Live Logs**:
  ```bash
  fly logs -a shubh-plan-open
  ```

- **Check App & Machine Status**:
  ```bash
  fly status -a shubh-plan-open
  ```

- **Check Assigned IP Addresses**:
  ```bash
  fly ips list -a shubh-plan-open
  ```

- **Connection Closed on Port 2222**:
  If connecting via SSH returns `Connection closed by remote host`, verify that:
  1. A **Dedicated IPv4** (`fly ips allocate-v4`) or `fly proxy` is used.
  2. The shared IPv4 address (`66.241.125.x`) was released so DNS points to the dedicated IPv4.
  3. `auto_stop_machines` is set to `"off"` in `fly.toml`.
