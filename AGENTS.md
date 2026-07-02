# AGENTS.md — Agnes Creator Studio

## Quick Start

```bash
# Terminal 1: Backend (Go)
cd backend
cp .env.example ../.env   # edit AGNES_API_KEY
go run ./cmd/server        # → http://localhost:8080

# Terminal 2: Frontend (Vue)
cd frontend
pnpm install
pnpm dev                   # → http://localhost:5173
```

## Project Structure

**Two codebases + legacy Python files at repo root:**

| Directory | Role |
|---|---|
| `backend/` | Go + Gin API server |
| `frontend/` | Vue 3 + Vite + Element Plus SPA |
| `docs/` | Design doc + implementation plan |
| root `*.py` | **Legacy** Gradio app (not used with new B/S architecture) |

## Backend (`backend/`)

| Path | Role |
|---|---|
| `cmd/server/main.go` | Entry: loads `.env`, config, wires Gin + routes + static /outputs |
| `internal/config/config.go` | `.config.json` file I/O, env var fallback |
| `internal/model/types.go` | All shared types (request/response/SSE events) |
| `internal/service/agnes.go` | `AgnesClient` — raw HTTP to Agnes AI API (image/video) |
| `internal/service/video_manager.go` | `TaskManager` — goroutine polling + subscriber pattern for SSE |
| `internal/handler/image.go` | 3 handlers: text-to-image, image-to-image (multipart), batch |
| `internal/handler/video.go` | 5 handlers: text-to-video, image-to-video, multi-image, status, SSE stream |
| `internal/handler/history.go` | `history.json` read/write (max 100 records) |
| `internal/handler/config_handler.go` | GET/PUT config |
| `internal/middleware/cors.go` | Allow localhost:5173 / :4173 |

### Routes (all under `/api/v1`)

```
POST /images/text-to-image    POST /images/image-to-image    POST /images/batch
POST /videos/text-to-video    POST /videos/image-to-video    POST /videos/multi-image
GET  /videos/:taskId          GET  /videos/stream/:taskId
GET  /config                  PUT  /config
GET  /history                 DELETE /history
GET  /outputs/*filepath       (static files)
```

### Build

```bash
cd backend
go build -o bin/server ./cmd/server
go vet ./...
# Cross-compile: GOOS=linux GOARCH=amd64 go build -o bin/server-linux ./cmd/server
```

### Config

- `.config.json` at project root (gitignored). Fields: `api_key`, `base_url`, `model`.
- `AGNES_API_KEY` env var overrides `.config.json` value.
- `.env.example` in `backend/` — copy to `../.env` for local dev.
- Default base URL: `https://apihub.agnes-ai.com/v1`

## Frontend (`frontend/`)

| Path | Role |
|---|---|
| `src/views/` | 7 views: TextToImage, ImageToImage, BatchGen, TextToVideo, ImageToVideo, MultiImageVideo, History |
| `src/components/ApiConfig.vue` | API key/URL/model form |
| `src/components/ImageResult.vue` | Image gallery with preview + download |
| `src/api/` | Axios wrappers for all API endpoints |
| `src/utils/sse.ts` | `connectSSE()` — EventSource helper for video progress |
| `src/stores/config.ts` | Pinia store for API config |

### Stack

Vue 3 (Composition API + `<script setup>`) · TypeScript · Vite 8 · Element Plus · Pinia · Axios · pnpm

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

- **Image-to-Image**: sends image as `data:image/png;base64,...` via `extra_body.image` (handled in Go backend from uploaded file).
- **Image-to-Video**: accepts either a multipart file upload or image URLs. Uploaded files are converted to base64 data URIs.
- **Multi-Image Video & Keyframes**: requires **publicly accessible image URLs**. Uses `extra_body.image` (array of URLs). Keyframes mode sets `extra_body.mode = "keyframes"`.
- **Video frame count must satisfy `8n + 1`** — enforced in `BuildVideoPayload()`. See `maxFramesForResolution()`: 1080p=169, 720p=409, 480p=961.
- **Video polling**: 5s interval, 30min timeout, max 10 concurrent tasks, exponential backoff on errors (max 10 retries).
- **Model names** are hardcoded in `backend/internal/service/agnes.go`: `agnes-image-2.1-flash` / `agnes-video-v2.0`.
- **Image download**: saved to `outputs/` with timestamped filenames. Video downloads stream in chunks.

## SSE (Server-Sent Events)

- Video progress pushed via `GET /api/v1/videos/stream/:taskId`.
- Events: `progress` (status + percentage), `complete` (download URL + seconds), `error`.
- Backend: `gin.Context.Stream()` with `text/event-stream` content type. Subscriber pattern via `TaskManager`.
- Frontend: `EventSource` in `src/utils/sse.ts`. Auto-closes on `complete` or `error`.

## Testing

No test framework. No linter, formatter, or typechecker configured for Go. Frontend typechecks via `vue-tsc -b` during build.

## Known Quirks

- **pnpm + lightningcss**: `lightningcss-darwin-x64` optional native binary sometimes not downloaded by pnpm. If `vite build` fails with `Cannot find module '../lightningcss.darwin-x64.node'`, extract the binary manually:
  ```bash
  curl -sL https://registry.npmjs.org/lightningcss-darwin-x64/-/lightningcss-darwin-x64-1.32.0.tgz | tar xz -C node_modules/.pnpm/lightningcss-darwin-x64@1.32.0/node_modules/lightningcss-darwin-x64/ --strip-components=1
  ```
- **`history.json`** and **`.config.json`** live at project root (gitignored). Backend contains logic to auto-detect project root from CWD or `backend/` parent.
- **`outputs/`** is gitignored. Backend serves it as static files via `/outputs/*filepath`.
- **Temp files**: image-to-image and image-to-video handlers save uploads to `backend/tmp/` which is gitignored and cleaned up after each request.
- **No auth middleware** — API is designed for local/dev use only. API key is sent to Agnes AI, not validated by the backend itself.
