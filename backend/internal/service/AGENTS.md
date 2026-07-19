# backend/internal/service/ — Core Business Logic

**5 files, 4 active + 1 legacy doc:** AgnesClient (HTTP to Agnes AI), TaskQueue (unified async worker pool), GithubStorage (GitHub Contents API), StoryboardGenerator (shot video pipeline). TaskManager (legacy, removed — fully replaced by TaskQueue).

## WHERE TO LOOK

| File | Concern | Key Types |
|------|---------|-----------|
| `agnes.go` | Raw Agnes AI API calls | `AgnesClient` — SubmitImageTask, SubmitVideoTask, CheckVideoStatus, GenerateScript, ExpandIdea, ChatCompletion |
| `task_queue.go` | Unified async task queue | `TaskQueue` — worker pool + SSE subscriber pattern (replaces TaskManager for new features) |
| `storyboard_generator.go` | Shot video pipeline | `StoryboardGenerator` — GenerateAll, GenerateOne, pollVideoStatus |
| _(removed)_ | TaskManager — legacy video polling, replaced by TaskQueue | `video_manager.go` removed from disk |
| `github_storage.go` | Remote file persistence | `GithubStorage` — Upload, Delete, DeleteByURL via GitHub Contents API |

## ARCHITECTURE

- `AgnesClient` is the **sole HTTP client** — all handlers receive it via constructor injection
- `TaskQueue` is the primary async worker: submits tasks (image/video) → goroutine pool → completion callback → history save. 10 concurrent workers, 5s poll interval, exponential backoff retry.
- `TaskManager` was the legacy video poller, now removed. `TaskQueue` fully replaces it for all features.
- `StoryboardGenerator` wraps `TaskQueue` for the shot video pipeline: submits shots → polls video status → downloads result to `outputs/` → updates shot record.
- `VideoOptions` struct carries resolution/duration/frame-rate config, validated in handler's `buildVideoOptions()`

## CONVENTIONS

- Errors returned as strings from all methods (no custom error types)
- Video payload building (`BuildVideoPayload`) returns `map[string]any` — dynamic keys for Agnes API flexibility
- Chinese log prefixes: `[Video]`, `[Task %s]`, `[Image]`, `[TaskQueue]`, `[StoryboardGenerator]`
- Image download (`DownloadVideo`) saved to `outputs/` then falls back to `../outputs/` (used only by storyboard)

## ANTI-PATTERNS

- Do NOT add new HTTP client instances — always inject `AgnesClient`
- Do NOT bypass `TaskQueue` for async video status — use Subscribe pattern
- Do NOT hardcode URLs — always use `client.BaseURL` from config
- Do NOT use `TaskManager` for new code — it's legacy; use `TaskQueue` instead
