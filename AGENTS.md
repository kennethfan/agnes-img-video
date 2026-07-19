# AGENTS.md — Agnes Creator Studio

**Generated:** 2026-07-12
**Commit:** 54a11b6 (dev)
**Stack:** Go 1.25 · Gin · SQLite · Vue 3 · TypeScript 6 · Vite 8 · Element Plus · Pinia · Axios · SSE

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

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Image generation API | `backend/internal/handler/image.go` | 3 handlers (text-to-image, image-to-image, batch) |
| Video generation API | `backend/internal/handler/video.go` | 6 handlers + SSE streaming |
| Ideas/comic expansion | `backend/internal/handler/ideas.go`, `comic.go` | Chat-based AI features |
| History CRUD | `backend/internal/handler/history.go` | SQLite-backed, file deletion |
| Asset gallery | `backend/internal/handler/asset.go` | Uses AssetRepository |
| Config management | `backend/internal/handler/config_handler.go` | GET/PUT .config.json |
| Storyboard | `backend/internal/handler/storyboard.go` | Projects + shots CRUD |
| Project management | `backend/internal/handler/project.go` | Projects CRUD + AI recommend + steps |
| Asset collections | `backend/internal/handler/collection.go` | Collections CRUD + add/remove assets |
| Prompt templates | `backend/internal/handler/template.go` | Templates CRUD + export/import |
| Task queue management | `backend/internal/handler/task_handler.go` | List/stream/cancel/retry tasks |
| Core business logic | `backend/internal/service/` | AgnesClient, TaskQueue, GithubStorage |
| GORM persistence | `backend/internal/repository/gorm/` | 8 repositories (History/Storyboard/Settings/AccessLog/Task/Project/Collection/Template) |
| Shared types | `backend/internal/model/types.go` | All request/response/SSE types |
| Frontend views | `frontend/src/views/` | 20 views + 3 wizard subdirs |
| Project dashboard | `frontend/src/views/ProjectDashboard.vue` | Stats, file aggregation, step progress |
| API client layer | `frontend/src/api/` | Axios wrappers per domain |
| Pinia stores | `frontend/src/stores/` | redo.ts, wizard.ts |
| Shared components | `frontend/src/components/` | NavSidebar, ImageResult, ShotCard, AssetCard, TaskProgress, StepProgressBar, ProjectStatsCards, ProjectFileGrid, AssetPickerDialog, AIPanel |
| Workflow step components | `frontend/src/components/` | IdeateStep, GenStep, RefineStep, FinalStep — project creation 4-step flow |
| Types | `frontend/src/types/index.ts` | All TS interfaces |
| SSE utility | `frontend/src/utils/sse.ts` | EventSource helper |
| App entry | `frontend/src/App.vue` | Vue Router + NavSidebar navigation |

## CODE MAP

### Backend (Go) — key symbols

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `AgnesClient` | struct | `internal/service/agnes.go` | Raw HTTP to Agnes AI API |
| `TaskQueue` | struct | `internal/service/task_queue.go` | Goroutine worker pool + SSE subscriber pattern |
| `GithubStorage` | struct | `internal/service/github_storage.go` | GitHub Contents API upload/download |
| `VideoTask` | struct | `internal/service/video_manager.go` | Task state + subscriber channels |
| `VideoHandler` | struct | `internal/handler/video.go` | Video HTTP handlers |
| `ImageHandler` | struct | `internal/handler/image.go` | Image HTTP handlers |
| `HistoryRepository` | struct | `internal/repository/gorm/history.go` | GORM CRUD for history |
| `StoryboardRepository` | struct | `internal/repository/gorm/storyboard.go` | GORM CRUD for storyboard |
| `ProjectHandler` | struct | `internal/handler/project.go` | Creative projects CRUD + AI Recommend + Steps |
| `CollectionHandler` | struct | `internal/handler/collection.go` | Asset collections CRUD |
| `TemplateHandler` | struct | `internal/handler/template.go` | Prompt templates CRUD + export/import |
| `ProjectRepository` | struct | `internal/repository/gorm/project.go` | GORM CRUD for projects + steps |
| `CollectionRepository` | struct | `internal/repository/gorm/collection.go` | GORM CRUD for collections |
| `TemplateRepository` | struct | `internal/repository/gorm/template.go` | GORM CRUD for prompt templates |
| `AssetRepository` | struct | `internal/repository/gorm/asset.go` | GORM CRUD for workspace assets |
| `TaskRecord` | struct | `internal/model/types.go` | Unified task record type |
| `VideoStatus/Event` | struct | `internal/model/types.go` | SSE event types |

### Frontend (Vue/TS) — key symbols

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `useRedoStore` | Pinia store | `stores/redo.ts` | Cross-view data passing |
| `useWizardStore` | Pinia store | `stores/wizard.ts` | Wizard step state (comic/novel/image) |
| `connectSSE` | function | `utils/sse.ts` | EventSource helper for video progress |
| `ImageResult` | component | `components/ImageResult.vue` | Reusable image gallery |
| `ShotCard` | component | `components/ShotCard.vue` | Storyboard shot display |
| `AssetCard` | component | `components/AssetCard.vue` | Asset gallery card |
| `TaskProgress` | component | `components/TaskProgress.vue` | SSE-driven progress bar |
| `NavSidebar` | component | `components/NavSidebar.vue` | Left navigation sidebar |
| `IdeateStep` | component | `components/IdeateStep.vue` | Ideation step — textarea idea input → AI brief |
| `GenStep` | component | `components/GenStep.vue` | Generation step — batch image gen from brief |
| `RefineStep` | component | `components/RefineStep.vue` | Refinement step — image + prompt refinement |
| `FinalStep` | component | `components/FinalStep.vue` | Finalize step — gallery + cover + complete project |
| `StepProgressBar` | component | `components/StepProgressBar.vue` | Step progress indicator for projects |
| `ProjectStatsCards` | component | `components/ProjectStatsCards.vue` | Dashboard stat cards (images/videos/files) |
| `ProjectFileGrid` | component | `components/ProjectFileGrid.vue` | Dashboard file grid with filtering |
| `AssetPickerDialog` | component | `components/AssetPickerDialog.vue` | Gallery asset picker dialog for cross-step selection |

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
| `internal/service/video_manager.go` | `TaskManager` — goroutine polling + subscriber pattern for SSE (legacy, replaced by TaskQueue) |
| `internal/service/task_queue.go` | `TaskQueue` — worker pool + SSE subscriber pattern (replaces TaskManager for new features) |
| `internal/service/github_storage.go` | `GithubStorage` — upload/download files via GitHub Contents API |
| `internal/handler/image.go` | 3 handlers: text-to-image, image-to-image (multipart), batch |
| `internal/handler/video.go` | 6 handlers: text-to-video, image-to-video, multi-image, script-gen, status, SSE stream |
| `internal/handler/ideas.go` | `ExpandIdea` — AI enhancement for creative ideas via chat completions |
| `internal/handler/comic.go` | `GeneratePrompts` — AI generation of comic panel prompts via chat completions |
| `internal/handler/history.go` | History API (SQLite via repository) + file deletion |
| `internal/handler/asset.go` | Asset gallery: list/favorite/batch-download/delete (uses HistoryRepo) |
| `internal/handler/settings.go` | GET/PUT settings (storage target, paths) |
| `internal/handler/task_handler.go` | List/Get/Stream/Cancel/Retry tasks |
| `internal/handler/access_log.go` | List/Delete/Clear access logs |
| `internal/handler/db_handler.go` | Export/restore database (JSON format) |
| `internal/handler/github_handler.go` | Upload/fetch via GitHub |
| `internal/handler/storyboard.go` | Storyboard CRUD: projects + shots (separate SQLite tables) |
| `internal/handler/project.go` | `ProjectHandler` — CRUD + AIRecommend + Steps (AddStep/UpdateStep/DeleteStep) |
| `internal/handler/collection.go` | `CollectionHandler` — CRUD + AddAssets + RemoveAssets |
| `internal/handler/template.go` | `TemplateHandler` — CRUD + Export/Import + SaveFromHistory |
| `internal/middleware/cors.go` | Allow localhost:5173 / :4173 |
| `internal/repository/gorm/gorm.go` | GORM `OpenDB()` + AutoMigrate (7 models) |
| `internal/repository/gorm/history.go` | GORM CRUD for history + favorites |
| `internal/repository/gorm/storyboard.go` | GORM CRUD for storyboard projects + shots |
| `internal/repository/gorm/settings.go` | GORM key-value settings persistence |
| `internal/repository/gorm/access_log.go` | GORM access log CRUD + daily cleanup |
| `internal/repository/gorm/task.go` | GORM task record CRUD + optimistic lock cancel |
| `internal/repository/gorm/project.go` | `ProjectRepository` — Project + ProjectStep GORM CRUD |
| `internal/repository/gorm/collection.go` | `CollectionRepository` — Collection + CollectionAssets GORM CRUD |
| `internal/repository/gorm/template.go` | `TemplateRepository` — PromptTemplate GORM CRUD/export/import |
| `internal/repository/gorm/asset.go` | `AssetRepository` — Asset CRUD/list/favorite/transfer |
| `internal/repository/interfaces.go` | Repository interface definitions |
| `scripts/recover_history.go` | Rebuild history records from `outputs/` filenames after data loss |

### Routes (all under `/api/v1`)

```
POST /images/text-to-image     POST /images/image-to-image     POST /images/batch
POST /videos/text-to-video     POST /videos/image-to-video     POST /videos/multi-image
POST /videos/generate-script   POST /ideas/expand   POST /comic/generate-prompts
POST /comic/generate-storyline
GET  /videos/:taskId           GET  /videos/stream/:taskId
GET  /config                   PUT  /config
GET  /history                  DELETE /history
DELETE /history/:id            POST /history/delete
GET  /assets                   POST /assets
POST /assets/favorite          POST /assets/batch-download
POST /assets/:id/transfer      DELETE /assets
GET  /storyboard/projects      POST /storyboard/projects
GET  /storyboard/projects/:id  PUT  /storyboard/projects/:id
DELETE /storyboard/projects/:id POST /storyboard/projects/:id/duplicate
POST /storyboard/projects/:id/shots     POST /storyboard/projects/:id/shots/batch
PUT  /storyboard/projects/:id/shots/reorder
PUT  /storyboard/shots/:id     DELETE /storyboard/shots/:id
POST /storyboard/projects/:id/generate
GET  /collections              POST /collections              PUT  /collections/:id
DELETE /collections/:id        POST /collections/:id/assets   DELETE /collections/:id/assets
GET  /templates                POST /templates                PUT  /templates/:id
DELETE /templates/:id          POST /templates/export         POST /templates/import
POST /history/:id/save-template
GET  /projects                 POST /projects                 GET  /projects/:id
PUT  /projects/:id             DELETE /projects/:id           POST /projects/:id/duplicate
POST /projects/:id/ai-recommend    POST /projects/:id/steps
PUT  /steps/:stepId             DELETE /steps/:stepId
POST /projects/:id/ideate-brief GET  /projects/:id/files
GET  /projects/:id/stats        PUT  /projects/:id/step-progress
GET  /tasks                    GET  /tasks/:id                 GET  /tasks/:id/stream
POST /tasks/:id/cancel          POST /tasks/:id/retry
POST /assets/:id/transfer
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

- **所有配置通过环境变量读取**，无 `.config.json`。
- 配置来源为 `backend/.env`（从 `.env.example` 复制），以及系统环境变量。
- API Key **仅通过 `API_KEY_PATH` 环境变量设置路径**，Key 存于单独文件（如 `~/.agnes/api-key`），该文件可设 `chmod 600`。不支持直接写入 `.env`。
- 完整环境变量列表见 `backend/.env.example`。核心变量：

| 环境变量 | 必需 | 说明 |
|----------|------|------|
| `API_KEY_PATH` | 是 | API Key 文件路径 |
| `BASE_URL` | 否 | API 地址（默认 `https://apihub.agnes-ai.com/v1`） |
| `IMAGE_MODEL` | 否 | 图像模型（默认 `agnes-image-2.1-flash`） |
| `VIDEO_MODEL` | 否 | 视频模型（默认 `agnes-video-v2.0`） |
| `CHAT_MODEL` | 否 | 对话模型（默认 `agnes-2.0-flash`） |
| `GITHUB_TOKEN` | 否 | GitHub 存储 token |
| `GITHUB_REPO` | 否 | GitHub 存储仓库 |
| `GITHUB_BRANCH` | 否 | GitHub 分支（默认 master） |
| `DB_DRIVER` | 否 | 数据库驱动（默认 sqlite） |
| `DB_DSN` | 否 | 数据库 DSN（默认 history.db） |
| `PORT` | 否 | 服务器端口（默认 8080） |

### GitHub File Storage (Optional)

When `GITHUB_TOKEN` and `GITHUB_REPO` are set, generated images/videos are uploaded to the configured GitHub repo via Contents API. The returned URL becomes the public download URL. File deletion (via history clear) also cleans up remote GitHub files.

## Frontend (`frontend/`)

| Path | Role |
|---|---|
| `src/views/` | 20 views: TextToImage, ImageToImage, BatchGen, ScriptGen, TextToVideo, ImageToVideo, MultiImageVideo, Ideas, History, Storyboard, Assets, ProjectList, ProjectEditor, ProjectDashboard, TemplateManager, WorkflowWizard, AccessLogs, DBManage, TaskRecords, Settings + 3 wizard subdirs (comic/novel/image) |
| `src/components/ImageResult.vue` | Image gallery with preview + download |
| `src/api/client.ts` | Axios instance (baseURL: '', 120s timeout) |
| `src/api/image.ts` | textToImage, imageToImage, batchGenerate |
| `src/api/video.ts` | text-to-video, image-to-video, multi-image, script-gen, task status |
| `src/api/history.ts` | getHistory, clearHistory, deleteHistory, deleteRecord |
| `src/api/ideas.ts` | expandIdea — AI idea enhancement |
| `src/api/ideate.ts` | ideateBrief — AI project brief generation |
| `src/api/storyboard.ts` | Storyboard CRUD: projects + shots API client |
| `src/api/history.ts` | getHistory, clearHistory, deleteHistory, deleteRecord |
| `src/api/assets.ts` | getAssets, toggleFavorite, batchDownload, deleteAssets |
| `src/api/settings.ts` | getSettings, updateSettings |
| `src/api/github.ts` | uploadToGithub, proxyDownload |
| `src/api/db.ts` | exportDB, restoreDB |
| `src/api/access-logs.ts` | getLogs, deleteLog, clearLogs |
| `src/api/projects.ts` | Project CRUD API client |
| `src/api/templates.ts` | Template CRUD + export/import API client |
| `src/api/collections.ts` | Collection CRUD API client |
| `src/api/task.ts` | Task list/stream/cancel/retry API client |
| `src/stores/redo.ts` | Pinia store: cross-view "redo" data passing via custom event `redo-trigger` |
| `src/utils/sse.ts` | `connectSSE()` — EventSource helper for video progress |
| `src/types/index.ts` | TypeScript interfaces for all API request/response types |

### Stack

Vue 3 (Composition API + `<script setup>`) · TypeScript 6 · Vite 8 · Element Plus · Pinia · Axios · Vue Router

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
- **Ideas (灵感) expansion**: calls `POST /ideas/expand` → goes to chat API with a creative writing system prompt. Supports template-based ideas (video story, idiom story, poem adaptation, fable, art concept) with structured Markdown output.
- **Video frame count must satisfy `8n + 1`** — enforced in `BuildVideoPayload()`. See `maxFramesForResolution()`: 1080p=169, 720p=409, 480p=961.
- **Video polling**: 5s interval, 30min timeout, max 10 concurrent tasks (semaphore channel), exponential backoff on errors (max 10 retries, 1s→30s).
- **Video status API quirk**: status query URL strips `/v1` from baseURL, queries `{baseDomain}/agnesapi?video_id={id}`. Video download URL sometimes appears in `remixed_from_video_id` field.
- **Image/video URLs**: generation flow stores raw API URLs directly — no auto-download to `outputs/`. On-demand save/transfer available in UI.
- **Download path fallback**: tries `outputs/` (relative to backend/), then falls back to `../outputs/` (project root).

## SSE (Server-Sent Events)

- Video progress pushed via `GET /api/v1/videos/stream/:taskId`.
- Events: `progress` (status + percentage), `complete` (download URL + seconds), `error`.
- Backend: `gin.Context.Stream()` with `text/event-stream` content type. Subscriber pattern via `TaskQueue` channel (replaces legacy `TaskManager`).
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
- **Frontend navigation**: uses Vue Router + `NavSidebar` component in `App.vue` with 22 page states (19 views + 3 wizard subdirs). `activePage` ref drives conditional rendering via `v-if` chain.
- **Frontend TypeScript 6**: `erasableSyntaxOnly` enabled in tsconfig — no enums or namespaces; use `as const` or union types.
- **Axios error interceptor**: errors are caught and converted to a plain `Error` with the server message. Always use `.catch()` or try/catch; raw Axios errors are never exposed to views.

## CONVENTIONS

- **Go**: Project-layout standard (`cmd/`, `internal/`, `internal/handler/`, `internal/service/`, etc.)
- **Vue**: Composition API + `<script setup>` only — no Options API
- **TypeScript**: `erasableSyntaxOnly` — no enums, use `as const` or union types
- **Error handling**: Always return `gin.H{"error": ...}` with Chinese error messages; frontend converts Axios errors via interceptor
- **Chinese comments**: All code comments in Chinese (项目规范)
- **No auth middleware**: API is local-only; API key sent to Agnes AI only
- **Video frame count**: Must satisfy `8n + 1` formula

## ANTI-PATTERNS (THIS PROJECT)

- **Do NOT delete `history.db` or `outputs/`** — runtime data, irreversible loss
- **Do NOT suppress TS errors** with `as any`, `@ts-ignore`, `@ts-expect-error`
- **Do NOT use enums** (TypeScript 6 `erasableSyntaxOnly`)
- **Do NOT use Options API** — Composition API + `<script setup>` only
- **Do NOT introduce auth middleware** — this is a local dev tool
- **Do NOT refactor while fixing bugs** — minimal fixes only
- **Do NOT modify `history.db` schema** without migration path

## UNIQUE STYLES

- Cross-view "redo" via Pinia store + custom DOM event `redo-trigger`
- SSE for video progress — subscriber pattern with buffered channels (max 10)
- Dual input mode (upload multipart OR JSON URL) for image/video generation endpoints
- History start-up recovery: re-checks pending video tasks on boot
- Video status API quirks: strips `/v1` from baseURL for status queries
- Download path fallback: `outputs/` → `../outputs/`

## 💣 敏感操作规则（必须先确认）

**执行任何敏感操作前，必须口头陈述操作内容、影响范围、回滚方案，经用户明确同意后方可执行。**

敏感操作包括但不限于：
- 数据库写入/更新/删除（`INSERT`/`UPDATE`/`DELETE`）
- 文件系统写入/删除/移动
- 配置修改（`.config.json`、环境变量、启动参数）
- 网络请求（发送 HTTP 请求到外部服务）
- Git 操作（`reset`/`rebase`/`force push`）
- 数据迁移、数据修复

**违规后果**：直接修改数据库或文件导致数据丢失/损坏，责任人承担全部责任。

## 🚨 数据安全规则（禁止删除运行时数据）

**严禁删除以下运行时数据文件：**

| 文件/目录 | 用途 | 后果 |
|---|---|---|
| `backend/history.db` | 所有历史记录、作品库、收藏数据 | 删除后作品库和历史记录都变为空 |
| `backend/outputs/` | 生成的图片和视频文件 | 删除后媒体无法查看 |

**规则：**
1. **任何情况下不得删除 `history.db`** — 包括测试、调试、清空操作。该文件包含所有历史记录索引 + 原始提示词。
2. **`outputs/` 中的文件可以删除个别**（通过界面操作），但不得 `rm -rf outputs/` 或 `rm -f *.db`。
3. **测试时必须备份数据** — 如果测试需要干净数据库，先 `cp history.db history.db.bak`，测试结束后恢复。
4. **如意外丢失数据** — 从 `backend/` 目录运行 `go run ./scripts/recover_history.go` 从 `outputs/` 目录文件名重建记录（提示词无法恢复，标记为 `[已恢复]`）。

## Image Input Dual Mode

Image-to-image and image-to-video views support both file upload and URL input via `inputMode` ref:
- `upload` mode: `file` ref holds the File object, sent as multipart FormData
- `url` mode: `imageUrl` ref holds the URL string, sent as JSON with `image_url` field

Validation (early in setup) checks the active source and warns if empty.
