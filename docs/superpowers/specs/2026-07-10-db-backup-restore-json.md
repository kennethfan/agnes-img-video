# DB 备份恢复 — JSON 格式支持（驱动无关）

**Date:** 2026-07-10
**Status:** Design Draft
**Driver:** `sqlite` | `mysql` | `postgres`

## Problem

当前备份恢复（`db_handler.go`）全部耦合 SQLite 特有 API：

- `dumpSQL()` 查询 `sqlite_master` + `PRAGMA table_info` — 仅 SQLite 可用
- `ExportDB(.db)` 发送 `history.db` 文件 — MySQL/PG 无此文件
- `RestoreDB(.db)` 走 `replaceFunc` 文件重命名 — 仅 SQLite 可用
- `execSQL()` 执行裸 SQL — 依赖 SQLite 格式的 DDL/DML

切换到 MySQL/PostgreSQL 后，这些功能全部失效。

## Goal

新增 **JSON 格式** 备份恢复，通过 GORM Model 操作实现驱动无关：

1. 导出：`GET /api/v1/db/export?format=json` — 通过 GORM `Model.Find()` 读取所有表
2. 恢复：`POST /api/v1/db/restore` + `.json` 文件 — 通过 GORM `Transaction + Create` 写入
3. 不修改已有 `.db` / `.sql` 格式（保持向下兼容）

## Design

### JSON 导出格式

```json
{
  "version": 1,
  "driver": "sqlite",
  "tables": {
    "histories": [
      {
        "id": 1,
        "mode": "text2image",
        "prompt": "a cat",
        "images": "[\"outputs/img1.png\"]",
        "time": "2026-07-09 12:00:00",
        "extra": "{\"size\": \"1024x1024\"}",
        "task_id": 0,
        "favorite": false,
        "asset_type": "image",
        "thumb": ""
      }
    ],
    "favorites": [
      { "history_id": 1 }
    ],
    "storyboard_projects": [
      {
        "id": 1,
        "title": "My Project",
        "script": "Scene 1...",
        "created_at": "...",
        "updated_at": "..."
      }
    ],
    "storyboard_shots": [
      {
        "id": 1,
        "project_id": 1,
        "sequence": 0,
        "prompt": "...",
        "type": "text2video",
        "reference_image": "",
        "status": "pending",
        "result_video": "",
        "task_id": "",
        "created_at": "..."
      }
    ],
    "settings": [
      { "key": "storage_target", "value": "local" }
    ],
    "access_logs": [
      {
        "id": 1,
        "method": "POST",
        "path": "/api/v1/images/text-to-image",
        "status": 200,
        "duration": 1234,
        "created_at": "..."
      }
    ],
    "task_queue": [
      {
        "id": 1,
        "type": "video",
        "status": "completed",
        "params": "{\"prompt\": \"...\"}",
        "progress": 100,
        "result": "https://...",
        "error": null,
        "retry_count": 0,
        "created_at": "...",
        "updated_at": "...",
        "completed_at": "..."
      }
    ]
  }
}
```

**字段说明：**
- `version` — 格式版本号（当前 = 1），用于未来迁移兼容
- `driver` — 导出时的数据库驱动名称，记录源数据库类型
- `tables` — 键为表名，值为该表全部行的数组（JSON 编码）
- 所有时间字段为字符串格式 `"2006-01-02 15:04:05"`
- `null` 字段保留为 JSON `null`（如 `error: null`）

### 后端改动

#### `db_handler.go`

**修改 `DBHandler` 结构体：**

```go
type DBHandler struct {
    dbPath      string
    replaceFunc ReplaceFunc
    getDB       func() *sql.DB     // 仅用于 .sql 导出（保持兼容）
    gormDB      *gorm.DB           // 新增：JSON 格式使用
}

// SetGormDB 更新 gormDB 引用（用于 dbReplaceFunc 重建时同步）
func (h *DBHandler) SetGormDB(db *gorm.DB) { h.gormDB = db }
```

**修改 `NewDBHandler` 签名：**

```go
func NewDBHandler(dbPath string, replaceFunc ReplaceFunc, getDB func() *sql.DB, gormDB *gorm.DB) *DBHandler
```

**.sql 和 .db 格式调用链保持不变，新增 JSON 分支：**

**`ExportDB` 新增 `format=json` 分支：**

```
GET /api/v1/db/export?format=json
```

1. 定义导出顺序列表（按外键依赖排序，先导出无依赖表）
2. 遍历表列表，对每个 GORM model 调用 `gormDB.Model(&T{}).Find(&records)`
3. 将 `[]T` 编码为 `[]map[string]any`
4. 组装为 `ExportPayload{}` 结构体
5. 序列化 JSON → `c.Data(http.StatusOK, "application/json", jsonBytes)`
6. 文件名：`history.json`

**`RestoreDB` 新增 `.json` 文件分支：**

```
POST /api/v1/db/restore  (上传 .json 文件)
```

1. 检测 `file.Filename` 后缀为 `.json`
2. 读取并反序列化为 `ExportPayload`
3. 校验 `version == 1`
4. 在 `gormDB.Transaction` 内：
   - 按**反序** Delete 各表（先删依赖表再删主表，避免 FK 冲突）
   - 按正序遍历表，对每个记录调用 `gormDB.Create(&record)`
5. 成功返回 `{"ok": true, "message": "数据库恢复成功"}`

**表顺序（外键依赖顺序）：**

| 导出/恢复顺序 | 表 | 依赖 |
|---|---|---|
| 1 | `settings` | 无 |
| 2 | `histories` | 无 |
| 3 | `favorites` | FK → histories.id |
| 4 | `storyboard_projects` | 无 |
| 5 | `storyboard_shots` | FK → storyboard_projects.id |
| 6 | `access_logs` | 无 |
| 7 | `task_queue` | 无 |

清除时的反序：7→6→5→4→3→2→1

#### `main.go`

```go
// 当前
dbHandler := handler.NewDBHandler(dbPath, dbReplaceFunc, func() *sql.DB { return sqlDB })

// 改为
dbHandler := handler.NewDBHandler(dbPath, dbReplaceFunc, func() *sql.DB { return sqlDB }, gormDB)
```

`dbReplaceFunc` 内部重建时需要同步更新 DBHandler 的 gormDB 引用：

```go
// 在 dbReplaceFunc 中，创建新 repo 之后
dbHandler.SetGormDB(newGormDB)
```

> ⚠️ **原因**：Go 中 `*gorm.DB` 是指针值传递，`DBHandler.gormDB` 在构造时捕获的是当时的指针值。`dbReplaceFunc` 更新局部变量 `gormDB = newGormDB` 不会影响到 `DBHandler` 内部持有的指针。必须显式调用 setter 方法。

这与现有的 `handler.SetHistoryRepo`、`middleware.SetAccessLogRepo` 等模式完全一致。

#### `NewDBHandler` 迁移

**现有调用方**（仅 `main.go`）需要传第 4 个参数。无其他调用方。

### GORM Model 映射

导出/恢复涉及以下 7 个 GORM model（定义在 `internal/repository/gorm/models.go`）：

| 表名 | Go 类型 | GORM 字段 |
|---|---|---|
| `histories` | `History` | ID, Mode, Prompt, Images, Time, Extra, TaskID, Favorite, AssetType, Thumb |
| `favorites` | `Favorite` | HistoryID |
| `storyboard_projects` | `StoryboardProject` | ID, Title, Script, CreatedAt, UpdatedAt |
| `storyboard_shots` | `StoryboardShot` | ID, ProjectID, Sequence, Prompt, Type, ReferenceImage, Status, ResultVideo, TaskID, CreatedAt |
| `settings` | `Setting` | Key, Value |
| `access_logs` | `AccessLog` | ID, Method, Path, Status, Duration, CreatedAt |
| `task_queue` | `TaskRecord` | ID, Type, Status, Params, Progress, Result, Error, RetryCount, CreatedAt, UpdatedAt, CompletedAt |

### 前端改动

#### `frontend/src/api/db.ts`

```typescript
export async function exportDB(format: 'db' | 'sql' | 'json' = 'db') {
//                                              ^^^^^ 新增
```

- 类型扩展：`format` 参数从 `'db' | 'sql'` 改为 `'db' | 'sql' | 'json'`
- 下载文件名改为 `history.${format}`（已支持）

#### `frontend/src/views/DBManage.vue`

- 导出区新增 **"导出 .json"** 按钮（`<el-button>导出 JSON</el-button>`）
- 上传恢复的 `accept` 加入 `.json`：
  ```html
  <el-upload accept=".db,.sqlite,.sql,.json">
  ```
- `handleRestore` 中加入 `.json` 文件处理分支，弹出确认对话框：
  ```
  "确定用「xxx.json」恢复数据？当前所有数据将被覆盖，此操作不可撤销。"
  ```

## Migration / Compatibility

### 已有数据迁移

用户从 SQLite 切换到 MySQL/PG 的路径：

1. 在 SQLite 下导出 `.json`（`GET /api/v1/db/export?format=json`）
2. 修改 `.config.json` 切换 `db_driver` + `db_dsn`
3. 重启服务器（AutoMigrate 自动建表）
4. 在界面上传 `.json` 恢复

### 兼容性

| 格式 | 导出 | 恢复 | SQLite | MySQL | PG |
|------|------|------|--------|-------|----|
| `.db` | `c.FileAttachment` | `replaceFunc` 文件替换 | ✅ | ❌ | ❌ |
| `.sql` | `dumpSQL()` | `execSQL()` | ✅ | ❌ | ❌ |
| `.json` | GORM `Find` | GORM `Create` | ✅ | ✅ | ✅ |

### 限制

- JSON 恢复是**全量替换**（truncate + insert），不支持增量恢复
- 大表数据（>1万行）可能较慢，因逐行 Create 在单个事务内
- JSON 文件中的 driver 字段仅用于信息记录，恢复时不会做 driver 校验

## 实现建议

1. 增加 `Encoder`/`Decoder` 概念进行行编解码，避免大量的 struct → map 重复代码
2. `ExportPayload` 的 `Tables` 字段可以用 `map[string][]map[string]any`，用泛型辅助函数减少转换
3. 恢复时的 Create 先用 `Session(&gorm.Session{FullSaveAssociations: false})` 避免 GORM 自动关联写入
