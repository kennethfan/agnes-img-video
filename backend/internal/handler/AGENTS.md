# backend/internal/handler/ — HTTP Handlers

**15 files, 12 handler structs** — all API endpoints under `/api/v1`.

## HANDLER INDEX

| Handler | File | Key Methods |
|---------|------|-------------|
| `ImageHandler` | `image.go` | TextToImage, ImageToImage (dual input), Batch |
| `VideoHandler` | `video.go` | TextToVideo, ImageToVideo (dual input), MultiImageVideo, GenerateScript, Status, SSE stream |
| `HistoryHandler` | `history.go` | GetHistory, ClearHistory, DeleteHistory, DeleteRecord |
| `AssetHandler` | `asset.go` | ListAssets, ToggleFavorite, BatchDownload, DeleteAssets, SaveAsset, TransferAsset |
| `StoryboardHandler` | `storyboard.go` | ListProjects, CreateProject, GetProject, UpdateProject, DeleteProject, DuplicateProject, CreateShot, UpdateShot, DeleteShot, ReorderShots, GenerateShots, BatchCreateShots |
| `ProjectHandler` | `project.go` | CreateProject, ListProjects, GetProject, UpdateProject, DeleteProject, DuplicateProject, AIRecommend, AddStep, UpdateStep, DeleteStep, IdeateBrief, GetProjectFiles, GetProjectStats, UpdateStepProgress |
| `CollectionHandler` | `collection.go` | ListCollections, CreateCollection, UpdateCollection, DeleteCollection, AddAssetsToCollection, RemoveAssetsFromCollection |
| `TemplateHandler` | `template.go` | ListTemplates, CreateTemplate, UpdateTemplate, DeleteTemplate, ExportTemplates, ImportTemplates, SaveFromHistory |
| `IdeasHandler` | `ideas.go` | ExpandIdea (chat-based AI) |
| `ComicHandler` | `comic.go` | GeneratePrompts, GenerateStoryline (chat-based AI) |
| `ConfigHandler` | `config_handler.go` | GET/PUT config |
| `SettingsHandler` | `settings.go` | GetSettings, UpdateSettings |
| `DBHandler` | `db_handler.go` | ExportDB (JSON), RestoreDB (upload JSON) |
| `TaskHandler` | `task_handler.go` | ListTasks, GetTask, StreamSSE, CancelTask, RetryTask |
| `AccessLogHandler` | `access_log.go` | ListLogs, DeleteLog, ClearLogs |
| _(global funcs)_ | `github_handler.go` | Upload/fetch from GitHub |

## ARCHITECTURE

### Constructor Injection
All handlers receive dependencies via `New*Handler(repo/svc)`:
```
NewImageHandler(svc *service.AgnesClient)
NewVideoHandler(svc *service.AgnesClient, tq *service.TaskQueue)
NewHistoryHandler(repo *repository.HistoryRepo)
NewStoryboardHandler(repo *repository.StoryboardRepo)
NewProjectHandler(repo *gormrepo.ProjectRepository, svc *service.AgnesClient)
NewCollectionHandler(repo *gormrepo.CollectionRepository)
NewTemplateHandler(repo *gormrepo.TemplateRepository)
...
```

### Global State (history.go)
Package-level vars shared across handlers:
```
var historyRepo *repository.HistoryRepo
var githubStorage *service.GithubStorage
var settingsRepo *repository.SettingsRepo
var outputsDir = "outputs"
```
Set by `main.go` via `SetHistoryRepo()`, `SetGithubStorage()`, `SetOutputsDir()`. Used in `saveHistoryRecord()`, `updateHistoryImages()`, `deleteRecordFiles()` — called from image/video handlers for persistence.

### Dual Input Pattern (image.go, video.go)
Image-to-Image and Image-to-Video handlers support two input modes:
- **Multipart FormData**: file upload (`image` field) + text fields
- **JSON body**: `image_url` / `image_urls` field with public URL

Uploaded files are saved to `backend/tmp/` and converted to base64 data URIs. Temp files cleaned up via `defer os.Remove()`.

### SSE Streaming (video.go)
- `GET /videos/stream/:taskId` pushes `text/event-stream`
- Events: `progress` (status + %), `complete` (URL + seconds), `error`
- Backend uses `gin.Context.Stream()` + `TaskQueue` subscriber pattern

### Error Handling
- All errors return Chinese messages: `gin.H{"error": "操作失败: " + err.Error()}`
- Log prefixes: `[History]`, `[Asset]`, `[Storyboard]`, `[DB]`
- No custom error types — plain string errors

## CONVENTIONS

- **No auth middleware** — local dev tool, API key sent to Agnes AI only
- **Null safety**: return empty slices `[]model.X{}` instead of nil
- **File downloads**: `deleteRecordFiles()` handles both local paths and GitHub URLs (raw.githubusercontent.com)
- **Batch download**: `AssetHandler.BatchDownload` streams ZIP archive in-memory
- **DB export**: JSON 格式 — `exportTableOrder` 控制表顺序，原生 INSERT 避免嵌套事务

## ANTI-PATTERNS

- Do NOT add new global state — prefer constructor injection
- Do NOT add auth middleware — local dev only
- Do NOT refactor `deleteRecordFiles()` — it handles both local file deletion and GitHub remote cleanup
- Do NOT bypass `TaskQueue` for async video status — use Subscribe pattern
