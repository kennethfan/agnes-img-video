# Asset Gallery & Management — Design Spec

## Overview

Add a unified gallery and management interface for all generated assets (images & videos). Currently, outputs are only accessible via individual generation views or the raw `outputs/` directory — there is no way to browse, search, favorite, or batch-manage past creations.

## Goals

1. Provide a browsable grid/list view of all generated assets across all modes
2. Enable search, filtering by type/time, and sorting
3. Add favorites/bookmarking for quick access to selected works
4. Support batch operations: multi-select, batch delete, batch download (zip)
5. Integrate with existing history system — no separate storage needed

## Non-Goals

- Image/video editing or transformation
- Sharing to social platforms
- Cloud sync or multi-device support
- Advanced tagging/categorization system (favorites only for v1)

## Architecture

### Data Source

The gallery pulls from two existing sources:

| Source | Purpose |
|--------|---------|
| `history.db` (SQLite) | Metadata: prompt, mode, timestamps, file paths, extras |
| `outputs/` directory | Actual asset files (images/videos) |

### New Backend API

```
GET  /api/v1/assets?page=1&per_page=20&type=image|video|all&sort=newest|oldest&search=&favorite=true|false
  → { items: AssetItem[], total: number, page: number }
  (favorite=true → only favorited; favorite=false → all; omitted = all)

POST /api/v1/assets/favorite    { "history_id": 1, "favorite": true }
  → { ok }

POST /api/v1/assets/batch-download  { "ids": [1,2,3] }
  → streams a zip file (Content-Type: application/zip)

DELETE /api/v1/assets  { "ids": [1,2,3], "delete_files": true }
  → { ok }
```

### AssetItem Type

```go
type AssetItem struct {
    ID        int64  `json:"id"`          // history record ID
    Mode      string `json:"mode"`        // text2image, image2video, etc.
    Prompt    string `json:"prompt"`      // truncated
    Files     []string `json:"files"`     // relative paths like /outputs/img_xxx.png
    Thumbnail string `json:"thumbnail"`   // first image or video poster
    Type      string `json:"type"`        // "image" or "video"
    Time      string `json:"time"`
    Favorite  bool   `json:"favorite"`
}
```

### Favorites Storage

Separate `favorites` table in SQLite (cleaner isolation, avoids schema migration issues):

```sql
CREATE TABLE IF NOT EXISTS favorites (
    history_id INTEGER PRIMARY KEY,
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (history_id) REFERENCES history(id) ON DELETE CASCADE
);
```

## Frontend: Assets Gallery Page

### New View: `src/views/Assets.vue`

Add as the 10th tab in `App.vue` between 点子库 and 历史记录.

**Layout:**

```
┌──────────────────────────────────────────────────┐
│ 🔍 [search input]  [type filter ▼] [sort ▼]     │
│ [grid view / list view toggle]                   │
├──────────────────────────────────────────────────┤
│ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐            │
│ │ img  │ │ vid  │ │ img  │ │ img  │            │
│ │ ★    │ │ ★    │ │      │ │ ★    │            │
│ │prompt│ │prompt│ │prompt│ │prompt│            │
│ └──────┘ └──────┘ └──────┘ └──────┘            │
│ ┌──────┐ ┌──────┐                               │
│ │ ...  │ │ ...  │                               │
│ └──────┘ └──────┘                               │
├──────────────────────────────────────────────────┤
│ [pagination]                                      │
└──────────────────────────────────────────────────┘
```

**Card Component (`AssetCard.vue`):**
- Thumbnail: `<el-image>` with lazy loading, or `<video>` poster frame
- Overlay on hover: prompt preview, mode badge (colored)
- Favorite star toggle (top-right corner)
- Checkbox for batch selection mode

**Detail Drawer/Dialog:**
- Click card → opens side drawer with:
  - Full-size image / video player
  - Full prompt text
  - Mode + timestamp
  - "Open in original view" / "Redo" buttons
  - Download single file

**Batch Mode:**
- Toolbar button "选择" toggles selection mode
- Bottom bar appears when items selected: "下载 (N)" + "删除 (N)"
- Delete → confirmation dialog with "同时删除文件" option

## Implementation Plan

### Phase 1: Backend (assets API)
1. Add `favorite` column to history table (migration)
2. Implement `GET /api/v1/assets` — paginated query from history, join with favorites
3. Implement `POST /api/v1/assets/favorites` — toggle favorites
4. Implement `POST /api/v1/assets/batch-download` — zip streaming
5. Implement `DELETE /api/v1/assets` — batch delete (records + files)

### Phase 2: Frontend (gallery view)
1. Create `src/api/assets.ts` — API client for new endpoints
2. Create `src/components/AssetCard.vue` — card component
3. Create `src/views/Assets.vue` — gallery page with search/filter/grid
4. Add tab to `App.vue`
5. Wire batch operations + confirmation dialogs

### Phase 3: Polish
1. Add lazy loading / virtual scroll for large galleries
2. Keyboard navigation
3. Empty states and error states

## Open Questions

- Should favorites survive history clear? → Yes, favorites reference history IDs; clearing history should prompt if user wants to keep favorites.
- Video thumbnails: extract first frame or use a placeholder? → Use a generic video icon + mode badge for v1; extract frame in a future iteration.
- Batch download file naming: → `agnes-assets-{date}.zip`, internal files keep original names.
