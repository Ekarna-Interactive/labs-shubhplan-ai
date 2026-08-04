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

* **Guided 4-Step Design Wizard**:
  1. **API Key Verification**: Prompt on launch if missing, persist to local `.env`, or run in offline dry-run mode.
  2. **Event Details**: Prompt for Event Type, Host/Couple Names, Date, and Location (persisted to `event_details.md`).
  3. **Target Image Resolution / Aspect Ratio**: Choose from 4 canvas layouts: `9:16` Mobile Story/Poster, `4:5` Social Feed/Portrait, `1:1` Square Card, or `16:9` Desktop Banner.
  4. **Design Style Selection**: Choose from 7 curated aesthetic presets (South Indian Traditional, Paper Cut Art, Clay 3D Render, Pop Art, Mughal Palace, Minimalist Gold Foil, Loose Watercolor) or type a custom style.
  5. **AI Prompt Suggestions**: Generate 4 live Gemini API prompt suggestions (`/suggest`) or enter a custom prompt. Option 5 (`5` / `more` / `r`) generates 4 brand-new suggestions anytime.
  6. **Generate & Preview**: Render high-fidelity invitation PNG, save asset to `./output`, and launch live web preview in default browser (`http://localhost:3000`).
* **Persistent Event Profile (`event_details.md`)**: Automatically saves your event profile (Event Details, Target Aspect Ratio) to `event_details.md`. On subsequent runs, Shubh CLI loads your active profile and skips straight to style selection.
* **Direct Step Navigation Slash Commands**:
  * `/generate [details]` (Alias `/design`): Compiles clean prompt with mandatory card text instructions, triggers AI image models, saves rendered PNGs locally, and opens the live web preview.
  * `/event [details]` (Alias `/details`, `/profile`): View or update event details, or type `/event` without parameters to jump directly to the Event Details setup step.
  * `/aspect [ratio]` (Alias `/resolution`, `/res`, `/ratio`): Set resolution (`9:16`, `4:5`, `1:1`, `16:9`), or type `/aspect` without parameters to jump directly to the Aspect Ratio step menu.
  * `/style [name/number]` (Alias `/aesthetic`, `/preset`): Set aesthetic design style, or type `/style` without parameters to jump directly to the Design Style menu view.
  * `/suggest [theme]` (Alias `/ideas`): Generates 4 live AI prompt suggestions using Gemini LLM agents (`gemini-3.5-flash` / `gemini-2.5-flash`) enforcing 100% strict style isolation.
  * `/refine [changes]` (Alias `/edit`, `/modify`): Applies variation layers to active design prompts.
  * `/reset` (Alias `/restart`, `/new`): Restart the guided setup wizard from Step 1.
  * `/preview` (Alias `/web`, `/open`): Manually launch or open local web browser preview (`http://localhost:3000`).
  * `/config [key]` (Alias `/key`, `/apikey`): Inspect or set your Gemini API key from [Google AI Studio](https://aistudio.google.com/api-keys).
  * `/help` (Alias `/h`, `/?`): Displays interactive command shortcuts manual.
* **Scrollable Terminal Viewport**: Full mouse wheel and keyboard scrolling (`PageUp`, `PageDown`, `Up`, `Down`, `Home`, `End`) through long prompt suggestions and terminal logs.
* **Automatic Web Preview**: Embedded background web server (`net/http`) using **Pico CSS** that automatically launches and opens `http://localhost:3000` in your default browser as soon as a design asset is generated.

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
├── .env.example          # Environment configuration template
├── .gitignore            # Git exclusion rules (.env, *.exe, /output/)
├── main.go               # Application entry point & template binder
├── go.mod                # Module: github.com/Ekarna-Interactive/ShubhPlan-CLI
├── /config
│   ├── env.go            # Environment variable loader & output directory resolver
│   └── event_profile.go  # Persistent event_details.md loader & serializer
├── /generator
│   ├── interface.go      # PromptBuilder interface contract
│   ├── basic_builder.go  # Single-resolution prompt compiler with card text instructions
│   └── prompter.go        # Live Gemini LLM AI Prompter Agent with strict style mandate
├── /command
│   └── parser.go         # Unified slash-command parser (/generate, /style, /aspect, /event, etc.)
├── /server
│   └── http.go           # Background net/http server & browser launcher
├── /templates
│   └── index.html        # Pico CSS live asset preview template with flex-centering
└── /ui
    ├── model.go          # Bubble Tea state & dual-pane layout model
    ├── view.go           # Lipgloss terminal styling & dashboard cards
    └── update.go         # Step event handling, Gemini API request executor, & viewport renderer
```

---

## 📜 License

Distributed under the MIT License. Built with ❤️ by Ekarna Interactive Technology LLP.
