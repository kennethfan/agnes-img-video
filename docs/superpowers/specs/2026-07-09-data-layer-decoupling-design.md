# Data Layer Decoupling — Repository Interface + GORM

**Date:** 2026-07-09
**Status:** Design Draft

## Problem

当前 5 个 Repository 全部直接操作 `*sql.DB`（SQLite driver），handler 依赖具体类型，
无法切换数据库后端，单元测试需要真实 SQLite 实例。

## Goal

1. Repository Interface 抽象，handler 依赖接口
2. GORM 替换裸 `database/sql`，支持多数据库（SQLite / PostgreSQL / MySQL）
3. 分三期落地，每期可独立上线

## Architecture

```
Before: handler → concrete HistoryRepo → *sql.DB (sqlite3)

After:  handler → repository.HistoryRepository → gorm.historyRepository → *gorm.DB
```

## Package Structure

```
internal/repository/
  interfaces.go              ← 5 个 Repository interface 定义
  gorm/
    gorm.go                  ← OpenDB() 工厂 + AutoMigrate
    models.go                ← GORM model structs
    history.go               ← HistoryRepository 的 GORM 实现
    storyboard.go            ← StoryboardRepository 的 GORM 实现
    settings.go              ← SettingsRepository 的 GORM 实现
    access_log.go            ← AccessLogRepository 的 GORM 实现
    task.go                  ← TaskRepository 的 GORM 实现
```

## Repository Interfaces

定义在 `repository/interfaces.go`：

- **HistoryRepository** — Insert/List/GetByIDs/Delete/DeleteMany/Clear/UpdateImages/FindByTaskId/FindPendingVideos/Trim/ToggleFavorite/GetFavoriteIDs
- **StoryboardRepository** — ListProjects/CreateProject/GetProject/UpdateProject/DeleteProject/DuplicateProject/CreateShot/UpdateShot/DeleteShot/ReorderShots/GetShotsByProject
- **SettingsRepository** — Get/Update
- **AccessLogRepository** — Insert/List/Delete/Clear/Cleanup
- **TaskRepository** — Insert/Get/List/Update/Delete/FindPending

Handler 构造函数改为接收 interface 而非具体类型，内部代码不变。

## GORM Models

7 张表映射为 GORM struct：History, Favorite, StoryboardProject, StoryboardShot, Setting, AccessLog, TaskRecord。

字段类型与现有 SQLite schema 兼容，`AutoMigrate` 自动建表/加字段。

## DB Factory

```go
func OpenDB(cfg DBConfig) (*gorm.DB, error)
```

通过 `cfg.Driver` 切换 dialector（sqlite / postgres / mysql），启动时执行 `AutoMigrate`。

配置项：`DB_DRIVER` + `DB_DSN`（来自 `.env` 或 `.config.json`）。

## Migration Strategy

所有改代码、不动数据。现有 `history.db` 可通过 GORM SQLite driver 直接打开，
`AutoMigrate` 不会破坏已有数据。

## Implementation Phases

### Phase 1 — Interface Extraction（不改数据库）

- 定义 `repository/interfaces.go`（5 个 interface）
- 现有裸 SQL 实现改成 interface 适配器（Adapter 模式）
- handler 签名 `*concrete` → `interface`
- main.go 注入适配器

### Phase 2 — GORM Implementation

- 创建 `repository/gorm/` 包
- 实现所有 Repository interface 的 GORM 版本
- main.go 切换为 GORM driver
- 验证数据兼容性，保留旧实现备份

### Phase 3 — Multi-Backend + Cleanup

- 配置层添加 `DB_DRIVER`/`DB_DSN`
- 删除旧 SQLite 实现
- 文档补充

## Open Questions

- 是否需要保留旧 `repository/*.go` 作为 fallback？→ Phase 3 删除，Phase 2 期间备份为 `.bak`
- GORM SQLite driver 版本选择？→ `gorm.io/driver/sqlite` latest
