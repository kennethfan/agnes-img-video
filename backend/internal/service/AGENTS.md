# backend/internal/service/ — Core Business Logic

**3 files, 3 distinct concerns:** AgnesClient (HTTP to Agnes AI), TaskManager (goroutine video polling), GithubStorage (GitHub Contents API).

## WHERE TO LOOK

| File | Concern | Key Types |
|------|---------|-----------|
| `agnes.go` | Raw Agnes AI API calls | `AgnesClient` — SubmitImageTask, SubmitVideoTask, CheckVideoStatus, GenerateScript, ExpandIdea, ChatCompletion |
| `video_manager.go` | Async video task lifecycle | `TaskManager`, `VideoTask` — CreateTask, Subscribe/Unsubscribe (SSE), polling goroutine, exponential backoff |
| `github_storage.go` | Remote file persistence | `GithubStorage` — Upload, Download, Delete via GitHub Contents API |

## ARCHITECTURE

- `AgnesClient` is the **sole HTTP client** — all handlers receive it via constructor injection
- `TaskManager` wraps `AgnesClient.CheckVideoStatus` with a polling goroutine (5s interval, 30min timeout, max 10 concurrent, exponential backoff 1s→30s, max 10 retries)
- `VideoTask.notifySubscribers` fan-out to SSE subscribers (buffered channels, drops overflow silently)
- `VideoOptions` struct carries resolution/duration/frame-rate config, validated in handler's `buildVideoOptions()`

## CONVENTIONS

- Errors returned as strings from all methods (no custom error types)
- Video payload building (`BuildVideoPayload`, `BuildImagePayload`) returns `map[string]any` — dynamic keys for Agnes API flexibility
- Chinese log prefixes: `[Video]`, `[Task %s]`, `[Image]`
- Image download (`DownloadVideo`) saves to `outputs/` then falls back to `../outputs/`

## ANTI-PATTERNS

- Do NOT add new HTTP client instances — always inject `AgnesClient`
- Do NOT bypass `TaskManager` for video status — use Subscribe pattern
- Do NOT hardcode URLs — always use `client.BaseURL` from config
