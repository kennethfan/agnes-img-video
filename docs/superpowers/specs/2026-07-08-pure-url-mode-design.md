# Pure URL Mode — Design Spec

## Overview

Currently, every generated image/video is downloaded from the Agnes API to `outputs/`, and if GitHub is configured, uploaded there too. This is wasteful: the API URLs work fine for display and remain valid for a reasonable period. This spec describes removing auto-download and auto-GitHub-upload, making them user-triggered on demand.

## Motivation

1. **Reduced latency**: Skip download I/O for every generation — return the API URL instantly
2. **Simpler failure mode**: No download errors to handle mid-generation
3. **Less storage**: `outputs/` only gets files the user explicitly chooses to save
4. **Cleaner separation**: Generation (fast, ephemeral URL) vs. persistence (user-controlled download/upload)

## Current Flow

```
Agnes API URL ──→ DownloadAndSave/DownloadVideo ──→ outputs/ (local) ──→ GitHub (optional)
                        ↓
                 History stores local path (/outputs/xxx.png)
                        ↓
                 Frontend displays via Vite proxy (/outputs/xxx.png)
```

## New Flow

```
Agnes API URL ──→ History stores API URL directly
                        ↓
                 Frontend displays API URL directly (works in <img>/<video>)
                        ↓
                 User clicks "下载到本地" or "上传到 GitHub" → triggers on-demand
```

## File-by-File Changes

### Backend

#### `backend/internal/handler/image.go` — 3 handlers

All three image handlers (`TextToImage`, `ImageToImage`, `BatchGenerate`) currently:
1. Call `h.svc.TextToImage()` / `h.svc.ImageToImage()` → get `[]string` URLs
2. Loop over URLs, call `h.svc.DownloadAndSave()` → get local paths
3. Store local paths in history
4. Return local paths in response

**New behavior**: Skip step 2 entirely. Store API URLs directly in history. Return API URLs directly to frontend.

**Change**: Remove the `DownloadAndSave` loop in all 3 handlers. Replace `localPaths` with the raw API URL list.

#### `backend/internal/handler/video.go` — `SetupVideoHistoryCallback`

Currently downloads video on completion:
```go
localPath, err := svc.DownloadVideo(resultURL, ...)
if err != nil { /* use original URL as fallback */ }
```

**New behavior**: Always use the API `resultURL` directly. Remove the `DownloadVideo` call. The callback stores the raw URL.

**Change**: Remove the download attempt block. `paths = []string{resultURL}` always.

#### `backend/internal/service/agnes.go` — keep methods

`DownloadAndSave()` and `DownloadVideo()` should be kept — they're still needed for the on-demand "下载到本地" feature. No changes.

#### `backend/internal/handler/github_handler.go` — NEW FILE

New endpoint for manual GitHub upload:

```
POST /api/v1/upload-to-github
  Body: { "url": "https://api.agnes.../image.png", "filename": "my_image.png" }
  → { "github_url": "https://github.com/.../outputs/my_image.png" }
```

Logic: fetch URL → upload to GitHub via existing `GithubStorage` → return GitHub URL.

#### `backend/cmd/server/main.go`

Register `POST /api/v1/upload-to-github` route.

#### `backend/internal/handler/history.go` / `asset.go` — skip local file ops

- `DeleteHistoryRecord`: currently also deletes files from `outputs/`. After the change, history records may contain remote URLs → skip file deletion if path starts with `http`.
- `ClearHistory`: same logic.
- Asset list/download: skip local file existence checks for URLs.

#### `backend/internal/model/types.go` — no change needed

`ImageResponse` already returns `Images []string`. Frontend already handles both local and remote URLs in `ImageResult.vue`.

### Frontend

#### `frontend/src/components/ImageResult.vue` — add "上传到 GitHub" button

Add a button next to "下载" that calls the new `/api/v1/upload-to-github` endpoint. Only show when the image URL starts with `http` (i.e., not a local `/outputs/` path).

#### `frontend/src/api/github.ts` — NEW FILE

```ts
export async function uploadToGitHub(url: string, filename: string): Promise<string>
```

#### History/Assets views — add "上传到 GitHub" button

Similar to `ImageResult.vue`, add an optional upload button for each image/video in History and Assets views.

#### Video views — no change needed

Video URLs are already displayed from `resultURL` (the SSE complete event). The change is backend-only for videos.

## Data Model Considerations

### History Records

The `images` field in `history` is `[]string`. Currently stores relative paths like `/outputs/img_xxx.png`. New records will store full URLs like `https://api.agnes-ai.com/v1/files/xxx.png`.

The frontend displays both correctly because:
- Local paths → served via Vite proxy `/outputs/` → `localhost:8080/outputs/xxx.png`
- Remote URLs → loaded directly from CDN

### Existing Records

Existing records with local paths (`/outputs/xxx.png`) continue to work — the files still exist in `outputs/`. No migration needed.

## On-Demand Features

### 下载到本地

Already implemented in `ImageResult.vue` — `downloadImage()` uses `fetch` + Blob. Works with both local and remote URLs.

### 上传到 GitHub

New feature. Button appears on images/videos in:
- Generation result area (`ImageResult.vue`)
- History view
- Assets view

Flow: click "上传到 GitHub" → frontend calls `POST /api/v1/upload-to-github` → backend downloads + uploads → returns GitHub URL → frontend shows the GitHub URL.

## Migration

No data migration needed. Old records keep their local paths. New records store API URLs. Both formats coexist.

## Rollback

If this change causes issues, revert the three handler files (`image.go` handlers, `video.go` callback) and the `github_handler.go` can stay (it's additive).

## Open Questions

1. **API URL expiry**: How long do Agnes API URLs remain valid? If they expire, old history records will show broken images. → User confirmed they're stable short-term. If expiry becomes a problem, add a batch download/re-upload feature later.

2. **Rate limiting**: On-demand GitHub upload may hit GitHub API rate limits if users upload many files at once. → Acceptable for now; optimize later if needed.

3. **CORS**: Agnes API URLs may have CORS restrictions when loaded in `<img>` or `<video>`. → Need to verify. If CORS blocked, fall back to download-and-serve approach for those specific URLs.
