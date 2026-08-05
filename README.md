# ✨ Shubh CLI — AI Event Invitation Design Terminal

> **Status:** 🛠️ In Active Development  
> **Public Repository:** [`github.com/Ekarna-Interactive/ShubhPlan-CLI`](https://github.com/Ekarna-Interactive/ShubhPlan-CLI)  
> **Module:** `github.com/Ekarna-Interactive/ShubhPlan-CLI`

**Shubh CLI** is an open-source, standalone terminal application and prompt generation engine specifically designed for **Event Invitation Design generation**. It generates creative visual prompt suggestions based on user-provided event details and renders bespoke invitation artwork using **Google Gemini Image Models** (such as **Nano Banana** and **Imagen**).

Shubh CLI is the standalone, open-source version of **Shubh Plan** — an Agentic AI Event Planner & Designer Application.

---

## 🏗️ About Shubh CLI & Shubh Plan

While **Shubh Plan** provides an agentic AI workspace for professional event planners and guests, **Shubh CLI** brings the core generative design engine straight to your command line:

* **🎨 Prompt Suggestions Based on Event Details**: Automatically creates rich, context-aware visual design prompts derived from specific event details (Event Type, Host/Couple Names, Date, Venue/Location, Canvas Aspect Ratio, and Aesthetic Design Style).
* **🤖 Google Gemini Image Generation**: Direct integration with Google's state-of-the-art image models — including **`gemini-3.1-flash-image` (Nano Banana)** and **Google Imagen (`imagen-4.0`)** — to generate print-ready digital invitations with clean layout plates and high visual adherence.
* **⚡ Standalone & Open Source**: Zero complex server dependencies. Built as a lightweight Go terminal user interface (TUI) with local markdown profile persistence (`event_details.md`) and an instant browser preview engine.

---

## 🌟 Key Features

* **Guided Interactive Design Wizard**:
  1. **API Key Verification**: Auto-detects `GEMINI_API_KEY`, prompts interactively on launch if missing, persists key to local `.env`, or runs in offline dry-run mode.
  2. **Event Profile Setup**: Captures Event Type, Host/Couple Names, Date, and Location (persisted to `event_details.md`).
  3. **Canvas Aspect Ratio**: Choose from 4 canvas layouts: `9:16` Mobile Story/Poster, `4:5` Social Feed/Portrait, `1:1` Square Card, or `16:9` Desktop Banner.
  4. **Curated Aesthetic Styles**: Choose from 7 curated aesthetic presets (*South Indian Traditional*, *Paper Cut Art*, *Clay 3D Render*, *Pop Art*, *Mughal Palace*, *Minimalist Gold Foil*, *Loose Watercolor*) or type a custom style.
  5. **AI Prompt Suggestions (`/suggest`)**: Dynamically generates 4 creative prompt options tailored to your exact event profile using Gemini LLM agents. Option 5 (`5` / `more`) generates 4 brand-new suggestions anytime.
  6. **Generate & Preview**: Render high-fidelity invitation PNGs to `./output` and instantly open a live browser preview (`http://localhost:3000`).
* **Persistent Event Profile (`event_details.md`)**: Automatically saves your event details and canvas settings. On subsequent runs, Shubh CLI reloads your active profile and jumps straight to style selection or generation.
* **Interactive Slash Commands**:
  * `/generate [details]` (Alias `/design`): Compiles clean prompt with card layout instructions, calls Gemini image models, saves rendered PNGs locally, and opens the live web preview.
  * `/event [details]` (Alias `/details`, `/profile`): View or update event details, or jump straight to the Event Details setup step.
  * `/aspect [ratio]` (Alias `/resolution`, `/res`, `/ratio`): Set aspect ratio (`9:16`, `4:5`, `1:1`, `16:9`) or open the ratio menu.
  * `/style [name/number]` (Alias `/aesthetic`, `/preset`): Set aesthetic design style or open the style preset menu.
  * `/suggest [theme]` (Alias `/ideas`): Generates 4 live AI prompt suggestions using Gemini LLM agents with strict style isolation.
  * `/refine [changes]` (Alias `/edit`, `/modify`): Applies variation layers to active design prompts.
  * `/reset` (Alias `/restart`, `/new`): Restart the guided setup wizard from Step 1.
  * `/preview` (Alias `/web`, `/open`): Open local web browser preview (`http://localhost:3000`).
  * `/config [key]` (Alias `/key`, `/apikey`): Inspect or update your Gemini API key from [Google AI Studio](https://aistudio.google.com/api-keys).
  * `/help` (Alias `/h`, `/?`): Displays interactive command shortcuts manual.
* **Automatic Web Preview Server**: Embedded background web server (`net/http`) built with **Pico CSS** that launches automatically at `http://localhost:3000` to inspect rendered invitation designs in your default browser.
* **Scrollable Terminal Viewport**: Full TUI keyboard and mouse wheel scrolling support (`PageUp`, `PageDown`, `Up`, `Down`, `Home`, `End`) built with Bubble Tea & Lipgloss.

---

## 🤖 Supported Google Gemini Image Models

Shubh CLI supports model selection across Google Gemini image models, allowing you to choose between high-fidelity production rendering, fast low-latency previews, and experimental multimodal media models.

| Model Name | Key Strengths & Use Cases | Official Documentation Reference |
| :--- | :--- | :--- |
| `gemini-3.1-flash-image` / **Nano Banana** *(Default)* | Primary default high-fidelity production model. Exceptional visual prompt adherence, typography text rendering, and intricate detail. | [Gemini Image Generation Guide](https://ai.google.dev/gemini-api/docs/image-generation) |
| `imagen-4.0-generate-001` | High-speed Imagen model optimized for rapid iterative draft previews. | [Imagen Model Versions](https://ai.google.dev/gemini-api/docs/imagen#model-versions) |
| `imagen-4.0-fast-generate-001` | Ultra-fast multimodal media model for low-latency visual previewing. | [Generative Media Models](https://ai.google.dev/gemini-api/docs/models#generative_media_models) |

### Configuring the Image Model

You can configure the active image generation model via environment variables in your local `.env` file:

```env
GEMINI_API_KEY="your_gemini_api_key_here"
GEMINI_IMAGE_MODEL="gemini-3.1-flash-image"
```

Official Google AI Documentation References:
* [Google AI Gemini Image Generation Guide](https://ai.google.dev/gemini-api/docs/image-generation)
* [Imagen Model Versions & Specifications](https://ai.google.dev/gemini-api/docs/imagen#model-versions)
* [Gemini Generative Media Models Guide](https://ai.google.dev/gemini-api/docs/models#generative_media_models)

---

## 🚀 Quickstart Guide

### Prerequisites
* **Go 1.22+** installed on your system.
* (Optional) A free **Gemini API Key** set in your environment or `.env` file from [Google AI Studio](https://aistudio.google.com/api-keys) for live image rendering.

### 1. Installation & Build

```bash
# Download dependencies
go mod tidy

# Build executable binary for Windows
go build -o shubh-cli.exe main.go
```

### 2. Run the Application

```bash
# Run the binary executable
.\shubh-cli.exe
```

### 3. Interactive Setup & API Key Management
* **API Key Setup**: On first launch, if `GEMINI_API_KEY` is not set, Shubh CLI interactively prompts you to enter your key (which is automatically saved to `.env`). Pressing **Enter** skips key setup and runs in offline dry-run mode.
* **Manage API Key Anytime**: You can inspect or update your key inside the TUI at any time using `/config <your-key>`.

---

## 🛠️ Directory Structure

```plaintext
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
│   └── prompter.go       # Live Gemini LLM AI Prompter Agent with strict style mandate
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

Distributed under the MIT License. Built with ❤️ by Ekarna Interactive Technology LLP as part of the open-source **Shubh Plan** ecosystem.
