# 🚀 Deployment Guide for Shubh Plan AI (`shubh-plan-open`)

This document provides a comprehensive guide for deploying **Shubh Plan AI Open Source** (`shubh-plan-open`) to [Fly.io](https://fly.io) or any cloud infrastructure, configuring network ports, managing IP addresses, and enabling terminal (SSH) & web access.

---

## 📋 Prerequisites

- **Fly CLI (`flyctl` / `fly`)**: Installed and logged in (`fly auth login`).
- **Go 1.24+ / Docker**: For local building and testing before deployment.
- **SSH Key Pair**: Local `~/.ssh/id_ed25519` key pair for TUI authentication.

---

## 🛠️ Step-by-Step Fly.io Deployment

### 1. Initialize the Fly App

From the project root directory, create the Fly application:

```bash
fly apps create shubh-plan-open
```

### 2. Configure `fly.toml`

Ensure your [fly.toml](fly.toml) contains both the HTTP Web UI (`:3000`) and the Wish SSH TUI (`:2222`) service configurations:

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
  SHUBH_DATA_DIR = "/app/data"
  SHUBH_OUTPUT_DIR = "/app/output"

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

---

### 3. Networking & IP Allocation (Important for SSH)

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

### 4. Deploy the Container

Deploy the application to Fly.io:

```bash
fly deploy
```

---

### 5. Scale Machine Count

By default, Fly launches 2 machines for high availability. To scale down to 1 machine:

```bash
fly scale count 1
```

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
