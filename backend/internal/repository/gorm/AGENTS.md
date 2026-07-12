# backend/internal/repository/gorm/ — GORM 数据持久层

**17 个文件 (12 Go + 5 test), 8 个仓库** — SQLite/Postgres/MySQL 数据访问 + AutoMigrate。

## 文件索引

| 文件 | 用途 | 核心类型 |
|------|------|----------|
| `gorm.go` | DB 初始化 + AutoMigrate | `OpenDB()`, `DBConfig` |
| `models.go` | 7 个 GORM 模型 | `History`, `Favorite`, `StoryboardProject`, `StoryboardShot`, `Setting`, `AccessLog`, `TaskRecord` |
| `history.go` | 历史记录 + 收藏 CRUD | `HistoryRepository` — 15 个方法 |
| `storyboard.go` | 分镜项目 + 镜头 CRUD | `StoryboardRepository` — 12 个方法 |
| `settings.go` | 键值对配置持久化 | `SettingsRepository` — Get/Update |
| `access_log.go` | 访问日志写/查/删 + 自动清理 | `AccessLogRepository` — 5 个方法, `StartDailyCleanup()` |
| `task.go` | 统一任务队列持久化 | `TaskRepository` — 12 个方法, 乐观锁取消 |
| `project.go` | 创作项目+步骤 CRUD | `ProjectRepository` — Create/List/Get/Update/Delete + Duplicate + Step CRUD |
| `collection.go` | 作品集 CRUD + 资产关联 | `CollectionRepository` — CRUD + AddAssets/RemoveAssets |
| `template.go` | 提示词模板 CRUD + 导入导出 | `TemplateRepository` — CRUD + Export/Import |
| `asset.go` | 作品库资产 CRUD | `AssetRepository` — Create/List/Get/ToggleFavorite/Transfer/Delete |
| `*_test.go` (5 个) | 单元测试 | history, settings, storyboard, task, asset, access_log |

## 架构

### OpenDB 初始化

```go
gormDB, err := gormrepo.OpenDB(gormrepo.DBConfig{Driver: "sqlite", DSN: "history.db"})
// → WAL 模式 + 5s busy timeout
// → AutoMigrate 所有 7 个模型
// → 支持 sqlite / postgres / mysql 三种驱动
```

### 仓库模式

所有仓库通过构造函数注入 `*gorm.DB`:
```go
histRepo := gormrepo.NewHistoryRepository(gormDB)
taskRepo := gormrepo.NewTaskRepository(gormDB)
projectRepo := gormrepo.NewProjectRepository(gormDB)
collectionRepo := gormrepo.NewCollectionRepository(gormDB)
templateRepo := gormrepo.NewTemplateRepository(gormDB)
assetRepo := gormrepo.NewAssetRepository(gormDB)
```

接口定义在 `backend/internal/repository/interfaces.go`，便于替换实现。

### 7+ GORM 模型 (models.go + project/collection/template)

| 模型 | 表名 | 关键字段 |
|------|------|----------|
| `History` | `histories` | ID, Time, Mode, Prompt, Images(JSON), Extra(JSON) |
| `Favorite` | `favorites` | ID, HistoryID (FK) |
| `StoryboardProject` | `storyboard_projects` | ID, Title, Script, CreatedAt, UpdatedAt |
| `StoryboardShot` | `storyboard_shots` | ID, ProjectID (FK), Seq, Prompt, Type, RefImage |
| `Setting` | `settings` | Key (PK), Value |
| `AccessLog` | `access_logs` | ID, Timestamp, Method, Path, Status, DurationMs, ClientIP, ... |
| `TaskRecord` | `task_records` | ID, Type, Status, Params(JSON), Result(JSON), Progress, RetryCount, ... |
| `Project` | `projects` | ID, Title, Brief, AIResult, Status, CoverURL, FinalURL, AssetIDs, Notes, HasAsset (has many Steps) |
| `ProjectStep` | `project_steps` | ID, ProjectID (FK), StepType, Position, Input, Output |
| `Asset` | `assets` | ID, Type, Prompt, OriginalURL, SavedURL, Size, StorageType, IsFavorite |
| `Collection` | `collections` | ID, Name, Assets (many2many via collection_assets) |
| `PromptTemplate` | `prompt_templates` | ID, Name, Type, Category, Prompt, Size, NegativePrompt, Model, Strength |

### 键值对存储

`Setting` 表使用 key-value 模式存储配置，`SettingsRepository` 通过 switch case 映射到 `model.Settings` 结构体。

## 约定

- 字符串时间戳格式: `"2006-01-02 15:04:05"`
- JSON 字段: `Images`, `Extra`, `Params`, `Result` 存储为 JSON 字符串，Go 层序列化/反序列化
- `AllowGlobalUpdate: true` session 用于全表删除
- 错误日志前缀: `[AccessLog]`

## 反模式

- 不要手动创建表 — 使用 `OpenDB().AutoMigrate()`
- 不要修改 `models.go` 字段类型或大小 — 迁移需谨慎
- 不要在仓库层使用 `gin.H{"error": ...}` — 仓库只返回 `error`
