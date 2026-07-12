# frontend/src/api/ — API Client Layer

**16 files** — Axios wrappers for all backend endpoints under `/api/v1`.

## FILE INDEX

| File | Endpoints | Module |
|------|-----------|--------|
| `client.ts` | Axios instance | Base configuration |
| `image.ts` | `/images/text-to-image`, `/images/image-to-image`, `/images/batch` | Image generation |
| `video.ts` | `/videos/text-to-video`, `/videos/image-to-video`, `/videos/multi-image`, `/videos/generate-script`, `/videos/:taskId` | Video generation + status |
| `history.ts` | `/history`, `/history/:id`, `/history/delete` | History CRUD |
| `ideas.ts` | `/ideas/expand` | AI idea expansion |
| `comic.ts` | `/comic/generate-prompts` | AI comic prompt generation |
| `storyboard.ts` | `/storyboard/projects`, `/storyboard/shots` | Storyboard CRUD |
| `assets.ts` | `/assets`, `/assets/favorite`, `/assets/batch-download` | Asset gallery |
| `settings.ts` | `/settings` | Storage settings |
| `github.ts` | GitHub upload | File transfer |
| `db.ts` | `/db/export`, `/db/restore` | Database backup/restore |
| `access-logs.ts` | `/access-logs` | API call logs |
| `projects.ts` | `/projects` | Creative project CRUD + AI recommend + steps |
| `templates.ts` | `/templates` | Prompt template CRUD + export/import |
| `collections.ts` | `/collections` | Asset collection CRUD |
| `task.ts` | `/tasks` | Task list/stream/cancel/retry |

## CORE

### Axios Client (`client.ts`)
```
baseURL: ''     (relative — Vite proxy handles /api → localhost:8080)
timeout: 120000 (2 minutes — image/video generation can be slow)
```
Error interceptor: converts Axios errors to plain `Error` with server message. Views never see raw Axios errors — always `.catch(e => e.message)`.

### Request Patterns

**JSON POST** (most endpoints):
```ts
const res = await client.post('/api/v1/images/text-to-image', body)
return res.data
```

**Multipart FormData** (image-to-image, image-to-video):
```ts
const form = new FormData()
form.append('image', file)
form.append('prompt', prompt)
const res = await client.post('/api/v1/images/image-to-image', form)
```

**Dual-mode convention**: views detect input type (file or URL) and call the appropriate wrapper — the API layer is agnostic.

**Blob download** (batch download):
```ts
const res = await client.post('/api/v1/assets/batch-download', { ids }, { responseType: 'blob' })
```

### Types
- Simple request/response types are defined inline in each API file
- Shared types (`VideoCreateRequest`, `HistoryRecord`, `AssetItem`, etc.) live in `src/types/index.ts`

## CONVENTIONS

- All functions are `async` and return `Promise<T>` where `T` is the expected response data (not the Axios wrapper)
- No raw Axios objects exposed to views — unwrap in the API layer
- Query params built with `URLSearchParams` for complex filters (access-logs, assets)

## ANTI-PATTERNS

- Do NOT import Axios directly in views — always go through `src/api/*`
- Do NOT return Axios responses unwrapped — always `return res.data`
- Do NOT add auth headers — API is local-only (no auth middleware)
