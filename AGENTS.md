# AGENTS.md — Agnes Creator Studio

## Quick Start

```bash
# Terminal 1: Backend (Go)
cd backend
cp .env.example .env      # edit AGNES_API_KEY
go run ./cmd/server        # → http://localhost:8080

# Terminal 2: Frontend (Vue)
cd frontend
pnpm install
pnpm dev                   # → http://localhost:5173
```

## Project Structure

| Directory | Role |
|---|---|
| `backend/` | Go + Gin API server (Go 1.25) |
| `frontend/` | Vue 3 + Vite + Element Plus SPA |
| `docs/` | Design doc + implementation plan |

No legacy Python/Gradio code — everything goes through the B/S architecture.

## Backend (`backend/`)

| Path | Role |
|---|---|
| `cmd/server/main.go` | Entry: loads `.env`, config, wires Gin + routes + static /outputs |
| `internal/config/config.go` | `.config.json` file I/O, env var fallback, default models |
| `internal/model/types.go` | All shared types (request/response/SSE events) |
| `internal/service/agnes.go` | `AgnesClient` — raw HTTP to Agnes AI API (image/video/chat/idea expansion) |
| `internal/service/video_manager.go` | `TaskManager` — goroutine polling + subscriber pattern for SSE |
| `internal/service/github_storage.go` | `GithubStorage` — upload/download files via GitHub Contents API |
| `internal/handler/image.go` | 3 handlers: text-to-image, image-to-image (multipart), batch |
| `internal/handler/video.go` | 6 handlers: text-to-video, image-to-video, multi-image, script-gen, status, SSE stream |
| `internal/handler/ideas.go` | `ExpandIdea` — AI enhancement for creative ideas via chat completions |
| `internal/handler/history.go` | History API (SQLite via repository) + file deletion |
| `internal/handler/config_handler.go` | GET/PUT config |
| `internal/middleware/cors.go` | Allow localhost:5173 / :4173 |
| `internal/repository/history.go` | SQLite CRUD for history, migration from legacy JSON |

### Routes (all under `/api/v1`)

```
POST /images/text-to-image     POST /images/image-to-image     POST /images/batch
POST /videos/text-to-video     POST /videos/image-to-video     POST /videos/multi-image
POST /videos/generate-script   POST /ideas/expand
GET  /videos/:taskId           GET  /videos/stream/:taskId
GET  /config                   PUT  /config
GET  /history                  DELETE /history
DELETE /history/:id            POST /history/delete
GET  /outputs/*filepath        (static files)
```

### Build

```bash
cd backend
go build -o bin/server ./cmd/server
go vet ./...
make build    # uses go build with -s -w ldflags
make test     # go test ./...
make run      # build + run
make clean    # rm -rf bin/
```

### Config

- `.config.json` at `backend/` (gitignored). Fields: `api_key`, `base_url`, `model`, `github_token`, `github_repo`, `github_branch`, `image_model`, `video_model`, `chat_model`.
- `AGNES_API_KEY`, `GITHUB_TOKEN`, `GITHUB_REPO`, `GITHUB_BRANCH` env vars override `.config.json`.
- `IMAGE_MODEL`, `VIDEO_MODEL`, `CHAT_MODEL` env vars override model defaults (see `.env.example`).
- `.env.example` in `backend/` — copy to `backend/.env` for local dev.
- Default base URL: `https://apihub.agnes-ai.com/v1`
- Default models: `agnes-image-2.1-flash` (image), `agnes-video-v2.0` (video), `agnes-2.0-flash` (chat/script).

### GitHub File Storage (Optional)

When `GITHUB_TOKEN` and `GITHUB_REPO` are set, generated images/videos are uploaded to the configured GitHub repo via Contents API. The returned URL becomes the public download URL. File deletion (via history clear) also cleans up remote GitHub files.

## Frontend (`frontend/`)

| Path | Role |
|---|---|
| `src/views/` | 9 views: TextToImage, ImageToImage, BatchGen, ScriptGen, TextToVideo, ImageToVideo, MultiImageVideo, Ideas, History |
| `src/components/ImageResult.vue` | Image gallery with preview + download |
| `src/api/client.ts` | Axios instance (baseURL: '', 120s timeout) |
| `src/api/image.ts` | textToImage, imageToImage, batchGenerate |
| `src/api/video.ts` | text-to-video, image-to-video, multi-image, script-gen, task status |
| `src/api/history.ts` | getHistory, clearHistory, deleteHistory, deleteRecord |
| `src/api/ideas.ts` | expandIdea — AI idea enhancement |
| `src/stores/redo.ts` | Pinia store: cross-view "redo" data passing via custom event `redo-trigger` |
| `src/utils/sse.ts` | `connectSSE()` — EventSource helper for video progress |
| `src/types/index.ts` | TypeScript interfaces for all API request/response types |

### Stack

Vue 3 (Composition API + `<script setup>`) · TypeScript 6 · Vite 8 · Element Plus · Pinia · Axios · Vue Router (registered but not used — nav is via `el-tabs`)

### Build

```bash
cd frontend
pnpm install
pnpm build      # vue-tsc -b && vite build, output to dist/
pnpm dev        # dev server on :5173 with proxy to :8080
```

### Vite Proxy

- `/api` → `http://localhost:8080`
- `/outputs` → `http://localhost:8080`

## API Peculiarities

- **Image-to-Image**: dual input — upload file (multipart) or provide `image_url` (JSON). Multipart form: `image` + `prompt` + `size` + `strength`. Uploaded files are converted to base64 data URI and sent via `extra_body.image`. JSON body: `{"image_url": "https://...", "prompt": "...", "size": "...", "strength": 0.75}`.
- **Image-to-Video**: dual input — upload file (multipart) or provide `image_url`/`image_urls` (JSON). Multipart form: `image` + `prompt`. JSON body: `{"image_url": "https://...", ...}` or `{"image_urls": ["https://..."], ...}`. Uploaded files are converted to base64 data URIs.
- **Multi-Image Video & Keyframes**: accepts JSON with **publicly accessible image URLs** in `image_urls[]`. Uses `extra_body.image` (array of URLs). Keyframes mode sets `extra_body.mode = "keyframes"`.
- **Script Generation**: calls `POST /videos/generate-script` → goes to chat API (`/chat/completions`) with a system prompt for video script writing. Supports `zh`/`en`.
- **Ideas (点子库) expansion**: calls `POST /ideas/expand` → goes to chat API with a creative writing system prompt. Supports template-based ideas (video story, idiom story, poem adaptation, fable, art concept) with structured Markdown output.
- **Video frame count must satisfy `8n + 1`** — enforced in `BuildVideoPayload()`. See `maxFramesForResolution()`: 1080p=169, 720p=409, 480p=961.
- **Video polling**: 5s interval, 30min timeout, max 10 concurrent tasks (semaphore channel), exponential backoff on errors (max 10 retries, 1s→30s).
- **Video status API quirk**: status query URL strips `/v1` from baseURL, queries `{baseDomain}/agnesapi?video_id={id}`. Video download URL sometimes appears in `remixed_from_video_id` field.
- **Image download**: saved to `outputs/` with timestamped filenames (`{prefix}_{timestamp}.png`). Video downloads stream in chunks to mp4 files.
- **Download path fallback**: tries `outputs/` (relative to backend/), then falls back to `../outputs/` (project root).

## SSE (Server-Sent Events)

- Video progress pushed via `GET /api/v1/videos/stream/:taskId`.
- Events: `progress` (status + percentage), `complete` (download URL + seconds), `error`.
- Backend: `gin.Context.Stream()` with `text/event-stream` content type. Subscriber pattern via `TaskManager` channel.
- Frontend: `EventSource` with `addEventListener` in `src/utils/sse.ts`. Auto-closes on `complete` or `error`.
- Max 10 buffered events per subscriber channel; drops overflow silently.

## Cross-View Redo Pattern

Views can push their input data back to any other view via the Pinia redo store + a custom DOM event:

1. Source view stores data in `useRedoStore().setRedoData({ mode, prompt, ... })`.
2. The store sets `targetTab` which `App.vue` watches via a `redo-trigger` custom event listener.
3. `App.vue` switches tabs; the target view consumes data via `useRedoStore().consumeRedoData()`.

The `modeToTab` mapping in `src/stores/redo.ts` translates `text2image|image2image|batch|script_gen|text2video|image2video|multi_image_video` to tab names.

## Testing

- **Go**: basic test exists for history repository (`internal/repository/history_test.go`). Run via `go test ./...` or `make test`.
- **Frontend**: typecheck via `vue-tsc -b` during `pnpm build`.
- No CI, no linter/formatter configured for Go.

## Known Quirks

- **pnpm + lightningcss**: `lightningcss-darwin-x64` optional native binary sometimes not downloaded by pnpm. If `vite build` fails with `Cannot find module '../lightningcss.darwin-x64.node'`, extract the binary manually:
  ```bash
  curl -sL https://registry.npmjs.org/lightningcss-darwin-x64/-/lightningcss-darwin-x64-1.32.0.tgz | tar xz -C node_modules/.pnpm/lightningcss-darwin-x64@1.32.0/node_modules/lightningcss-darwin-x64/ --strip-components=1
  ```
- **`history.db`** and **`.config.json`** live in `backend/` (gitignored).
- **`outputs/`** lives in `backend/` (gitignored). Backend serves it as static files via `/outputs/*filepath`.
- **Temp files**: image-to-image and image-to-video handlers save uploads to `backend/tmp/` which is gitignored and cleaned up after each request (`defer os.Remove`).
- **Config & runtime data** (`.config.json`, `history.db`): all in `backend/`. Copy `backend/.env.example` to `backend/.env` for local dev.
- **No auth middleware** — API is designed for local/dev use only. API key is sent to Agnes AI, not validated by the backend itself.
- **Startup recovery**: on boot, scans SQLite for pending video tasks (images empty + extra has taskId) and checks their status, updating history records for completed ones.
- **Frontend navigation**: uses `el-tabs` (Element Plus) in `App.vue` with 9 tabs — not Vue Router, even though `vue-router` is a dependency.
- **Frontend TypeScript 6**: `erasableSyntaxOnly` enabled in tsconfig — no enums or namespaces; use `as const` or union types.
- **Axios error interceptor**: errors are caught and converted to a plain `Error` with the server message. Always use `.catch()` or try/catch; raw Axios errors are never exposed to views.

## Image Input Dual Mode

Image-to-image and image-to-video views support both file upload and URL input via `inputMode` ref:
- `upload` mode: `file` ref holds the File object, sent as multipart FormData
- `url` mode: `imageUrl` ref holds the URL string, sent as JSON with `image_url` field

Validation (early in setup) checks the active source and warns if empty.
