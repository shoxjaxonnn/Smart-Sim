# Smart Edu — AI Simulation Platform (MVP)

Monorepo. Go backend + Vue 3 frontend. Replaces static tests with situational AI simulations.

```
smart-edu/
├── backend/    Go API (Gemini + get_fact anti-hallucination + rubric grading)
└── frontend/   Vue 3 + Vite chat UI (desktop-first, responsive)
```

## Quick start

### 1. Backend
```bash
cd backend
# optional — without a key it runs in offline Mock mode:
#   set GEMINI_API_KEY=...   (PowerShell: $env:GEMINI_API_KEY="...")
go run .
```
Serves on `http://localhost:8080`. Loads every `*.md` in `backend/scenarios/`.

### 2. Frontend
```bash
cd frontend
npm install
npm run dev
```
Open `http://localhost:5173`. Vite proxies `/api` → backend `:8080`.

## How it works (spec sections)

- **3.1 Anti-hallucination** — the model gets one tool, `get_fact(key)`. Numbers come
  from the per-scenario Facts Store, never from the model. Missing key → code returns
  "unavailable". See `backend/internal/llm/gemini.go`.
- **3.2 Rubric grading** — `/api/grade` returns structured JSON scored per criterion,
  forced valid via Gemini `responseSchema`. Keywords credited only when used correctly.
- **3.3 LLM abstraction** — `llm.Provider` interface. Swap Gemini ↔ Mock ↔ OpenAI by
  changing one line in `main.go`.
- **Section 4 scenario file** — one Markdown file = facts + rubric + model answer + prose.
  New subject = new file, no code. See `backend/scenarios/sql-injection-001.md`.

## API

| Method | Endpoint | Purpose |
|---|---|---|
| GET  | `/api/scenarios`     | List scenarios |
| GET  | `/api/scenarios/:id` | Scenario detail (situation prose) |
| POST | `/api/session`       | Start session `{scenario_id}` |
| POST | `/api/chat`          | `{session_id, message}` → reply (RAG-less: prose in prompt) |
| POST | `/api/grade`         | `{session_id, answer}` → GradeResult JSON |

## MVP scope notes

Cut from the full spec for hackathon speed: pgvector/RAG and the Python embedding
sidecar are deferred — scenario prose is short, so it's injected directly into the
system prompt. Sessions are in-memory (no Postgres needed to demo). Both are drop-in
later: the `Provider` interface and scenario schema already match the full design.
# smart-edu
