# DB 备份恢复 JSON 格式 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add driver-agnostic JSON format to backup/restore, keeping existing `.db`/`.sql` formats intact.

**Architecture:** Backend `db_handler.go` gets a `gormDB *gorm.DB` field + `SetGormDB()` setter. JSON export uses GORM `Table(name).Find(&[]map[string]any{})` — no model types needed. JSON restore uses `Transaction(DELETE → Create)` — driver-agnostic. `main.go` passes gormDB + calls `SetGormDB()` in dbReplaceFunc. Frontend adds JSON button and `.json` accept.

**Tech Stack:** Go 1.25, Gin, GORM v2, Vue 3, Element Plus, Axios

## Global Constraints

- Keep `.db` / `.sql` export/restore logic unchanged
- Use `gormDB.Table("name").Find(&[]map[string]any{})` — avoids importing model types
- Go struct fields capture value at construction — use `SetGormDB()` setter, not pointer capture
- All time fields: `"2006-01-02 15:04:05"` string format
- Restore is full replacement (DELETE + Create), not incremental
- Export table order: settings → histories → favorites → storyboard_projects → storyboard_shots → access_logs → task_queue
- Delete (restore prepare) order: reverse of export order
- `version == 1` only supported (future-proof)
- Frontend: `<script setup>` + Composition API only
- Frontend: `erasableSyntaxOnly` — no enums, use union types

---

### Task 1: Backend — Add gormDB + SetGormDB to DBHandler

**Files:**
- Modify: `backend/internal/handler/db_handler.go:18-27`

**Interfaces:**
- Consumes: nothing (first task)
- Produces: `DBHandler` with `gormDB *gorm.DB` field + `SetGormDB(*gorm.DB)` method

- [ ] **Step 1: Add import for gorm.io/gorm**

```go
import (
	"gorm.io/gorm"
)
```

- [ ] **Step 2: Add gormDB field to DBHandler struct and setter**

```go
type DBHandler struct {
	dbPath      string
	replaceFunc ReplaceFunc
	getDB       func() *sql.DB    // 仅用于 .sql 导出（保持兼容）
	gormDB      *gorm.DB          // 新增：JSON 格式使用
}

func (h *DBHandler) SetGormDB(db *gorm.DB) { h.gormDB = db }
```

- [ ] **Step 3: Update NewDBHandler signature**

```go
func NewDBHandler(dbPath string, replaceFunc ReplaceFunc, getDB func() *sql.DB, gormDB *gorm.DB) *DBHandler {
	return &DBHandler{dbPath: dbPath, replaceFunc: replaceFunc, getDB: getDB, gormDB: gormDB}
}
```

- [ ] **Step 4: Run go vet to verify**

```bash
cd backend && go vet ./internal/handler/
```
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/db_handler.go
git commit -m "feat: add gormDB field and SetGormDB to DBHandler"
```

---

### Task 2: Backend — JSON export logic

**Files:**
- Modify: `backend/internal/handler/db_handler.go` (add ExportPayload types, exportJSON method, update ExportDB)

**Interfaces:**
- Consumes: `DBHandler.gormDB`
- Produces: `GET /api/v1/db/export?format=json` returns `application/json` with `ExportPayload{...}`

- [ ] **Step 1: Add encoding/json and fmt imports**

Add `"encoding/json"` to the import block (alongside existing `"fmt"`, `gorm.io/gorm` from Task 1, etc.):

```go
import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)
```

- [ ] **Step 2: Add types and helpers after DBHandler definition**

```go
// ExportPayload JSON 导出/恢复负载
type ExportPayload struct {
	Version int                       `json:"version"`
	Driver  string                    `json:"driver"`
	Tables  map[string][]map[string]any `json:"tables"`
}

// exportTableOrder 导出顺序（按外键依赖排序）
var exportTableOrder = []string{
	"settings",
	"histories",
	"favorites",
	"storyboard_projects",
	"storyboard_shots",
	"access_logs",
	"task_queue",
}
```

- [ ] **Step 3: Add exportJSON method**

```go
func (h *DBHandler) exportJSON() ([]byte, error) {
	payload := ExportPayload{
		Version: 1,
		Driver:  "gorm",
		Tables:  make(map[string][]map[string]any),
	}

	for _, table := range exportTableOrder {
		var rows []map[string]any
		if err := h.gormDB.Table(table).Find(&rows).Error; err != nil {
			return nil, fmt.Errorf("导出表 %s 失败: %w", table, err)
		}
		payload.Tables[table] = rows
	}

	return json.MarshalIndent(payload, "", "  ")
}
```

- [ ] **Step 4: Update ExportDB to handle format=json**

```go
// ExportDB 导出数据库
// GET /api/v1/db/export?format=db|sql|json
func (h *DBHandler) ExportDB(c *gin.Context) {
	format := c.DefaultQuery("format", "db")

	switch format {
	case "sql":
		dump, err := h.dumpSQL()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "导出SQL失败: " + err.Error()})
			return
		}
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", `attachment; filename="history.sql"`)
		c.Data(http.StatusOK, "application/octet-stream", []byte(dump))
	case "json":
		data, err := h.exportJSON()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "导出JSON失败: " + err.Error()})
			return
		}
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", `attachment; filename="history.json"`)
		c.Data(http.StatusOK, "application/json", data)
	default:
		c.FileAttachment(h.dbPath, "history.db")
	}
}
```

- [ ] **Step 5: Run go vet to verify**

```bash
cd backend && go vet ./internal/handler/
```
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/db_handler.go
git commit -m "feat: add JSON export format via GORM"
```

---

### Task 3: Backend — JSON restore logic

**Files:**
- Modify: `backend/internal/handler/db_handler.go` (add restoreJSON method, update RestoreDB)

**Interfaces:**
- Consumes: `DBHandler.gormDB`, `ExportPayload` types (Task 2)
- Produces: `POST /api/v1/db/restore` with `.json` file → full restore

- [ ] **Step 1: Add restoreJSON method**

```go
func (h *DBHandler) restoreJSON(data []byte) error {
	var payload ExportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("JSON 解析失败: %w", err)
	}
	if payload.Version != 1 {
		return fmt.Errorf("不支持的版本号: %d", payload.Version)
	}

	return h.gormDB.Transaction(func(tx *gorm.DB) error {
		// 反序清空（先删依赖表，再删主表）
		for i := len(exportTableOrder) - 1; i >= 0; i-- {
			table := exportTableOrder[i]
			if err := tx.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
				return fmt.Errorf("清空表 %s 失败: %w", table, err)
			}
		}
		// 正序插入
		for _, table := range exportTableOrder {
			rows, ok := payload.Tables[table]
			if !ok || len(rows) == 0 {
				continue
			}
			for _, row := range rows {
				if err := tx.Table(table).Create(&row).Error; err != nil {
					return fmt.Errorf("恢复表 %s 失败: %w", table, err)
				}
			}
		}
		return nil
	})
}
```

- [ ] **Step 2: Update RestoreDB to handle .json files**

In the RestoreDB method, add a `.json` branch before the `.sql` and `.db` branches:

```go
if strings.HasSuffix(file.Filename, ".json") {
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取JSON文件失败: " + err.Error()})
		return
	}
	if err := h.restoreJSON(content); err != nil {
		log.Printf("[DB] JSON 恢复失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JSON恢复失败: " + err.Error()})
		return
	}
	log.Printf("[DB] JSON 恢复成功 (from: %s)", file.Filename)
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "JSON 恢复成功"})
	return
}
```

The updated RestoreDB method should check extensions in order: `.json` first, then `.sql`, then `.db`/`.sqlite` (default).

- [ ] **Step 3: Run go vet to verify**

```bash
cd backend && go vet ./internal/handler/
```
Expected: no errors

- [ ] **Step 4: Run go build to verify**

```bash
cd backend && go build ./...
```
Expected: no errors (this will fail because main.go doesn't pass 4th arg yet — fixed in Task 4)

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/db_handler.go
git commit -m "feat: add JSON restore via GORM transaction"
```

---

### Task 4: Backend — Update main.go

**Files:**
- Modify: `backend/cmd/server/main.go` (NewDBHandler call, dbReplaceFunc)

**Interfaces:**
- Consumes: `NewDBHandler(..., gormDB)` 4-arg signature (Task 1), `SetGormDB()` (Task 1)

- [ ] **Step 1: Update NewDBHandler call to pass gormDB**

```go
// 之前
dbHandler := handler.NewDBHandler(dbPath, dbReplaceFunc, func() *sql.DB { return sqlDB })

// 改为
dbHandler := handler.NewDBHandler(dbPath, dbReplaceFunc, func() *sql.DB { return sqlDB }, gormDB)
```

- [ ] **Step 2: Update dbReplaceFunc to call SetGormDB**

Inside `dbReplaceFunc`, after creating all new repos, add:

```go
// 更新所有引用
gormDB = newGormDB
sqlDB = newSQLDB
histRepo = newHistRepo
storyboardRepo = newStoryRepo
accessLogRepo = newAccessLogRepo
settingsRepo = newSettingsRepo
handler.SetHistoryRepo(newHistRepo)
historyHandler.SetRepo(newHistRepo)
assetHandler.SetRepo(newHistRepo)
settingsHandler = handler.NewSettingsHandler(newSettingsRepo)
middleware.SetAccessLogRepo(newAccessLogRepo)
storyboardHandler.SetRepo(newStoryRepo)
dbHandler.SetGormDB(newGormDB)  // <-- 新增：同步 gormDB 引用
```

- [ ] **Step 3: Run go build to verify**

```bash
cd backend && go build ./...
```
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: pass gormDB to DBHandler, sync in dbReplaceFunc"
```

---

### Task 5: Frontend — Add JSON export/restore support

**Files:**
- Modify: `frontend/src/api/db.ts`
- Modify: `frontend/src/views/DBManage.vue`

**Interfaces:**
- Consumes: backend `GET /api/v1/db/export?format=json` and `POST /api/v1/db/restore`

- [ ] **Step 1: Update db.ts — add 'json' to format union type**

```typescript
import { ElMessage } from 'element-plus'
import client from './client'

/** 导出数据库文件：支持 .db / .sql / .json 格式 */
export async function exportDB(format: 'db' | 'sql' | 'json' = 'db') {
  const res = await fetch(`/api/v1/db/export?format=${format}`)
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: '导出失败' }))
    ElMessage.error(err.error || '导出失败')
    return
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `history.${format}`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/** 恢复数据库：上传 .db/.sqlite/.sql/.json 文件 */
export async function restoreDB(file: File): Promise<void> {
  const form = new FormData()
  form.append('file', file)
  await client.post('/api/v1/db/restore', form)
}
```

- [ ] **Step 2: Update DBManage.vue — add JSON button and .json accept**

In the `<script setup>` section, add a handler:
```typescript
async function handleExportJSON() {
  await exportDB('json')
}
```

In the `<template>` section, add a JSON export button:
```html
<el-button @click="handleExportJSON">导出 JSON</el-button>
```

In the restore upload, update the accept attribute and the file type validation:

```html
<el-upload
  :show-file-list="false"
  :before-upload="handleRestore"
  accept=".db,.sqlite,.sql,.json"
  :disabled="uploading"
>
```

In `handleRestore`, add `.json` branch:
```typescript
async function handleRestore(rawFile: File) {
  const file = rawFile
  if (!file) return false

  if (file.name.endsWith('.json')) {
    try {
      await ElMessageBox.confirm(
        `确定用「${file.name}」恢复数据？\n\n当前所有数据将被覆盖，此操作不可撤销。`,
        '确认恢复',
        { confirmButtonText: '确定恢复', cancelButtonText: '取消', type: 'warning' },
      )
    } catch {
      return false
    }
  } else if (file.name.endsWith('.sql')) {
    // ... existing SQL logic unchanged ...
  } else if (file.name.endsWith('.db') || file.name.endsWith('.sqlite')) {
    // ... existing .db logic unchanged ...
  } else {
    ElMessage.warning('请选择 .db、.sqlite、.sql 或 .json 文件')
    return false
  }

  uploading.value = true
  try {
    const msg = file.name.endsWith('.json') ? 'JSON 恢复成功'
      : file.name.endsWith('.sql') ? 'SQL 执行成功'
      : '数据库恢复成功'
    await restoreDB(file)
    ElMessage.success(msg)
  } catch (e: any) {
    ElMessage.error(e?.message || '恢复失败')
  } finally {
    uploading.value = false
  }
  return false
}
```

Update the description text:
```html
<p class="desc">
  支持 .db / .sql / .json 三种格式。JSON 格式兼容 SQLite/MySQL/PostgreSQL 所有数据库驱动。
</p>
```

- [ ] **Step 3: Run frontend typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```
Expected: no type errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/db.ts frontend/src/views/DBManage.vue
git commit -m "feat: add JSON export/restore to frontend"
```

---

### Task 6: Verify full pipeline

- [ ] **Step 1: Run backend tests**

```bash
cd backend && go test ./... -count=1
```
Expected: all existing tests pass (42 GORM tests, plus any others)

- [ ] **Step 2: Run go vet + build**

```bash
cd backend && go vet ./... && go build ./...
```
Expected: no errors

- [ ] **Step 3: Run frontend build**

```bash
cd frontend && pnpm build
```
Expected: build succeeds

- [ ] **Step 4: Manual integration check**

Start backend and test JSON export:
```bash
curl -o /tmp/test.json 'http://localhost:8080/api/v1/db/export?format=json'
```
Expected: valid JSON with `version`, `driver`, `tables` fields and all 7 tables present.

Test JSON restore:
```bash
curl -X POST -F 'file=@/tmp/test.json' 'http://localhost:8080/api/v1/db/restore'
```
Expected: `{"ok": true, "message": "JSON 恢复成功"}`

- [ ] **Step 5: Final commit (if fixes needed)**

```bash
git add -A && git commit -m "fix: post-integration fixes"
```
