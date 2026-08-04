# ✨ Shubh CLI — AI Event Design Terminal

> Public Repository: [`github.com/Ekarna-Interactive/ShubhPlan-CLI`](https://github.com/Ekarna-Interactive/ShubhPlan-CLI)  
> Module: `github.com/Ekarna-Interactive/ShubhPlan-CLI`

An open-source, standalone terminal application and conversational prompt interface for boutique event planners and individual hosts, derived from the **Shubh Plan** platform.

---

## 🏗 About Shubh Plan (Work-In-Progress)

**Shubh Plan** is an upcoming AI-native B2B event management platform built for event professionals, agencies, and individual hosts:

* **🎨 Generative AI Invitation Design (Flagship Feature)**: Advanced AI design engine that creates print-ready, bespoke invitation artwork with native text rendering, cultural color swatches (Weddings, Naming Ceremonies, Baby Showers, Housewarmings, Corporate Galas), and elegant multi-line typography layout plates, eliminating the need for manual graphic design editors.
* **Shubh Plan (Planner Workspace)**: An all-in-one workspace where event planners manage multi-session event itineraries, collaborate on client design approvals, track real-time guest RSVPs, and organize event logistics.
* **Shubh Card (Guest Experience)**: A mobile-first digital invitation platform featuring an interactive AI concierge that assists guests with event schedules, venue directions, and RSVP check-ins.

*Note: **Shubh CLI** is the standalone, open-source terminal design tool derived from Shubh Plan's core generative prompt engine.*

---

## 🌟 Key Features

* **Guided 3-Step Design Wizard**:
  1. **API Key Verification**: Prompt on launch if missing, persist to local `.env`, or run in offline dry-run mode.
  2. **Event Details**: Prompt for Event Type, Host/Couple Names, Date, and Location.
  3. **Style Selection**: Choose from 7 curated design aesthetics (South Indian Traditional, Paper Cut Art, Clay 3D, Pop Art, Mughal Palace, Minimalist Gold Foil, Loose Watercolor, or custom).
  4. **Prompt Options**: Generate 4 AI prompt suggestions (`/suggest`) or enter a custom prompt.
  5. **Generate & Preview**: Render high-fidelity invitation PNG, save to `./output`, and launch the web preview in browser (`http://localhost:3000`).
* **Persistent Event Profile (`event_details.md`)**: Automatically saves your event details (Event Type, Host Names, Date, Location) to `event_details.md` on first input. On subsequent launches, Shubh CLI automatically loads your active event profile and skips straight to style selection! Manage or update anytime with `/event <new_details>`.
* **Unified Slash Commands**:
  * `/generate [details]` (Alias `/design`): Compiles clean prompts, triggers AI image models, saves rendered PNGs locally, and opens the live web preview.
  * `/event [details]` (Alias `/details`): View or update your active event details saved in `event_details.md`.
  * `/style [name/number]` (Alias `/aesthetic`): Select an aesthetic design style (e.g. `/style paper cut`).
  * `/suggest [theme]` (Alias `/ideas`): Generates 4 tailored AI prompt suggestions.
  * `/refine [changes]` (Alias `/edit`): Applies variation layers to active design prompts.
  * `/reset` (Alias `/restart`): Restart the guided setup wizard from Step 1.
  * `/preview`: Manually launches or opens the local web browser preview.
  * `/config [key]`: Inspect or set your Gemini API key from [Google AI Studio](https://aistudio.google.com/api-keys).
  * `/help`: Displays interactive command shortcuts.
* **Automatic Web Preview**: Embedded background web server (`net/http`) using **Pico CSS** that automatically launches and opens `http://localhost:3000` in your default browser as soon as a design asset is generated.
* **Gemini Imagen API Integration**: Direct integration with Gemini Imagen model APIs (`imagen-3.0-generate-002`) with zero-config local placeholder fallbacks for offline or API-key-free testing.

---

## 🚀 Architecture & Platform Roadmap

Shubh CLI is the open-source terminal application for AI event design. The main Shubh web application and platform are currently under active development, featuring enhanced agentic AI workflows, multi-viewport rendering engines, and collaborative design suites.

---

## 🚀 Quickstart Guide

### Prerequisites
* **Go 1.22+** installed on your system.
* (Optional) A **Gemini API Key** set in your environment or `.env` file for live image rendering.

### 1. Installation & Build

```bash
# Navigate to the CLI directory
cd apps/shubh-cli

# Download dependencies
go mod tidy

# Build executable binary for Windows
go build -o shubh-cli.exe main.go
```

### 2. Initial Setup & Execution

```bash
# Run the application
.\shubh-cli.exe
```

* **Interactive Setup**: On your first launch, if `GEMINI_API_KEY` is not detected, Shubh CLI prompts you interactively to enter your key (Generate your free key at [Google AI Studio](https://aistudio.google.com/api-keys)):
  * **Paste Key**: Pasting your Gemini API Key automatically persists it to a local `.env` file (`GEMINI_API_KEY="your_gemini_api_key_here"`) for all future runs.
  * **Skip (Press Enter)**: Pressing Enter skips setup and runs in offline dry-run mode.
* **Manage API Key Anytime**: You can inspect or update your API key inside the TUI at any time using `/config <your-key>` or `/key <your-key>`.

---

## 🤖 Gemini Image Model Selection & Versions

Shubh CLI supports model selection for Google Gemini image models, allowing you to choose between high-fidelity production rendering, fast low-latency previewing, and experimental multimodal media models.

### Supported Model Options

| Model Name | Description & Use Case | Official Documentation Reference |
| :--- | :--- | :--- |
| `gemini-3.1-flash-image` / *Nano Banana*  **(Default)** | Primary high-fidelity production model. Best typography rendering, visual prompt adherence, and rich detail. | [Google AI Imagen Guide](https://ai.google.dev/gemini-api/docs/image-generation#model-selection) |
| `imagen-4.0-generate-001` | High-speed, lower-latency model optimized for rapid iterative draft previews. | [Imagen Model Versions](https://ai.google.dev/gemini-api/docs/imagen#model-versions) |
| `imagen-4.0-fast-generate-001` | Multimodal generative media model supporting joint text and visual generation. | [Generative Media Models](https://ai.google.dev/gemini-api/docs/models#generative_media_models) |

### Configuring the Image Model

You can configure the active image generation model in two ways:

1. **Environment Configuration (`.env`)**:
   Add `GEMINI_IMAGE_MODEL` to your local `.env` file:
   ```env
   GEMINI_API_KEY="your_gemini_api_key_here"
   GEMINI_IMAGE_MODEL="imagen-3.0-generate-002"
   ```

2. **Official Google AI Documentation References**:
   * [Google AI Image Generation & Model Selection](https://ai.google.dev/gemini-api/docs/image-generation#model-selection)
   * [Imagen Model Versions & Specifications](https://ai.google.dev/gemini-api/docs/imagen#model-versions)
   * [Gemini Generative Media Models Guide](https://ai.google.dev/gemini-api/docs/models#generative_media_models)

---

## 🛠 Directory Structure

```plaintext
/apps/shubh-cli
├── main.go               # Application entry point & template binder
├── go.mod                # Module: github.com/Ekarna-Interactive/ShubhPlan-CLI
├── /config
│   └── env.go            # Environment variable loader & output directory resolver
├── /generator
│   ├── interface.go      # PromptBuilder interface contract
│   └── basic_builder.go  # Community 1:1 single-resolution prompt compiler
├── /command
│   └── parser.go         # Unified slash-command parser (/generate, /refine, /suggest, /preview)
├── /server
│   └── http.go           # Background net/http server & browser launcher
├── /templates
│   └── index.html        # Pico CSS live asset preview template
└── /ui
    ├── model.go          # Bubble Tea state initialization
    ├── view.go           # Lipgloss terminal styling
    └── update.go         # Gemini API handler & web preview trigger loop
```

---

## 📜 License

Distributed under the MIT License. Built with ❤️ by Ekarna Interactive Technology LLP.
