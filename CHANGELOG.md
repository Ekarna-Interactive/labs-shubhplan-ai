# 🚀 Changelog — Shubh Plan Web

All notable changes to the **Shubh Plan Web** project will be documented in this file.

---

## 📦 `v0.1.0` — Release (2026-08-28)

> **Shubh Plan Web v0.1.0** is the official release of the AI-native event planning operating system and bespoke invitation studio built on **Go 1.25+**, **Firebase Genkit Go SDK**, and **Google Gemini Cloud AI**.

### 🌟 Feature & Quality Highlights

#### 🤖 1. AI Event Assistant & SSE Real-Time Streaming
- **Central Model**: Standardised on **`googleai/gemini-flash-latest`** for lightning-fast, high-reasoning text generation and event orchestration.
- **Real-Time SSE Streaming**: Full Server-Sent Events (`/api/stream/assistant`) streaming pipeline with glowing 3-dot typing indicators, instant token delivery, and cancellation monitoring (`r.Context().Done()`).
- **Slash Commands (`/`) & Quick Chips**: Immediate interception for quick action commands (`/summarize`, `/add-guests`, `/schedule`, `/generate-invitation`) with floating backdrop-blur autocomplete menu and full keyboard navigation.
- **Smart Event Parser**: Automatically extracts and persists event metadata (Title, Event Type, Host Names, Date, Venue, Target Guest Count) to the unified DataStore live.

#### 🎨 2. Bespoke AI Invitation Studio
- **High-Res PNG Image Model**: Powered strictly by **`googleai/gemini-3.1-flash-image`** for standalone, print-ready invitation card artwork.
- **4 Dynamic Aspect Ratios**: Full support for `9:16` (Story/Mobile), `4:5` (Portrait/Feed), `1:1` (Square/Post), and `16:9` (Landscape/Banner).
- **7 Signature Aesthetic Presets**: *South Indian*, *Paper Cut 3D*, *Clay 3D*, *Pop Art*, *Mughal*, *Minimalist Gold*, and *Watercolor*.
- **Prompt Synthesizer**: 2-step AI Art Director flow synthesizing custom prompt concepts.
- **1-Column Mobile Layout**: Responsive concept card grid (`.design-grid`) rendering full-width artwork on screens <768px.

#### 📱 3. Responsive UI & Mobile Fluidity
- **Sub-576px Viewport Fluidity**: Enforced `min-width: 0 !important` and `box-sizing: border-box !important` across all layout containers, ensuring 100% fluid rendering without right-edge clipping on mobile viewports (320px–576px).
- **AI Assistant Mobile Wrapping**: Word-breaking (`word-break: break-word; overflow-wrap: anywhere;`) on chat bubbles (`.msg-bubble`) and profile fields.
- **Compact Mobile Controls**: Collapsed chat Send button to an icon button (`➔`) on mobile devices.
- **Guest Roster Touch Cards**: Transformed the 6-column guest roster table into responsive stacked mobile cards with `data-label` headers, eliminating horizontal scrolling.
- **Event Profile Select Dropdowns**: Converted Event Type, Aesthetic Theme, and Target Guest Count text/number fields into custom styled `<select>` controls with celebration presets and `setSelectOrAddOption()` custom fallbacks.

#### 🧹 4. Memory Management & Package Test Suite
- **Auth Session Eviction**: `CleanExpiredSessions()` background ticker purging expired Argon2id sessions every 15 minutes.
- **Genkit Session File Pruning**: `PruneOldSessionFiles()` cleaning stale session files older than 7 days on server startup.
- **100% Test Coverage**: Full package unit test suite covering `pkg/auth`, `pkg/genkit`, `pkg/middleware`, `pkg/server`, `pkg/store`, and `main` (`go test -v ./...`).

#### 🔐 5. Security, Dual-Key BYOK & Data Persistence
- **Argon2id Authentication**: Native Argon2id password hashing matching enterprise specifications.
- **Dual-Key BYOK Modal**: Support for Gemini API Key and Google Maps Places API Key stored in browser `localStorage`.
- **Fly.io Production Deployment**: Containerised Docker build deployed to Fly.io (`shubh-plan-demo.fly.dev`).

---

### 📊 Release Summary

| Component | Technology / Specification |
| :--- | :--- |
| **Go Runtime** | Go `1.25+` |
| **AI Framework** | Firebase Genkit Go SDK (`github.com/firebase/genkit/go`) |
| **Text Generation Model** | `googleai/gemini-flash-latest` |
| **Image Generation Model** | `googleai/gemini-3.1-flash-image` |
| **Auth Engine** | Argon2id (`$argon2id$...` with salt & key derivation) |
| **Frontend UI** | Vanilla CSS, HTMX, Single-Page Architecture |
| **Deployment Target** | Fly.io (`https://shubh-plan-demo.fly.dev/`) |

---

*Shubh Plan Web v0.1.0 — Copyright © 2026 Ekarna Interactive Technology LLP.*
