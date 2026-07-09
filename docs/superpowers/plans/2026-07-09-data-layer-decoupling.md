# Data Layer Decoupling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decouple data operations from SQLite by extracting Repository interfaces and implementing GORM-backed backends with multi-database support.

**Architecture:** Three-phase plan: (1) Extract interfaces + keep existing SQLite impls as adapters, (2) Add GORM implementations, (3) Add multi-backend config. Handlers depend on interfaces only.

**Tech Stack:** Go 1.25, GORM v2, gorm.io/driver/sqlite, gorm.io/driver/postgres (Phase 3)

## Global Constraints

- All handler internal code stays unchanged — only constructor signatures change
- Existing `history.db` data must remain compatible through all phases
- No `as any` / `@ts-ignore` / `@ts-expect-error` (Go: no `interface{}` where concrete type works)
- Phase 2 GORM implementation must use AutoMigrate, not manual CREATE TABLE
- `go vet ./...` must pass after every phase
- Comment language: Chinese (项目规范)

---

### Task 1: Define Repository Interfaces

**Files:**
- Create: `backend/internal/repository/interfaces.go`

**Interfaces:**
- Produces: 5 Go interfaces + `PendingVideoInfo` moved from history.go

- [ ] **Step 1: Create `interfaces.go` with all 5 repository interfaces**

```go
package repository

import (
	"github.com/agnes-image-tool/backend/internal/model"
)

// ==================== History ====================

// PendingVideoInfo 待恢复的视频任务信息
type PendingVideoInfo struct {
	ID     int64
	TaskID string
	Prompt string
	Mode   string
}

type HistoryRepository interface {
	InsertRecord(prompt string, images []string, mode string, extra any) (int64, error)
	GetRecords(limit int) ([]model.HistoryRecord, error)
	GetRecordsPaginated(page, perPage int, assetType, search string, favIDs map[int64]bool) ([]model.HistoryRecord, int, error)
	GetRecordsByIDs(ids []int64) ([]model.HistoryRecord, error)
	DeleteRecord(id int64) error
	DeleteRecords(ids []int64) error
	ClearRecords() error
	UpdateRecordImages(id int64, images []string) error
	FindByTaskId(taskId int64) (int64, error)
	FindPendingVideos() ([]PendingVideoInfo, error)
	TrimRecords(max int) error
	ToggleFavorite(historyID int64, favorite bool) error
	GetFavoriteIDs() (map[int64]bool, error)
}

// ==================== Storyboard ====================

type StoryboardRepository interface {
	ListProjects() ([]model.StoryboardProject, error)
	CreateProject(title, script string) (int64, error)
	GetProject(id int64) (*model.StoryboardProject, error)
	UpdateProject(id int64, title, script string) error
	DeleteProject(id int64) error
	DuplicateProject(id int64, newTitle string) (int64, error)
	CreateShot(projectID int64, shot model.StoryboardShot) (int64, error)
	UpdateShot(id int64, shot model.StoryboardShot) error
	DeleteShot(id int64) error
	ReorderShots(projectID int64, shotIDs []int64) error
	GetShotsByProject(projectID int64) ([]model.StoryboardShot, error)
	Close() error
}

// ==================== Settings ====================

type SettingsRepository interface {
	GetSettings() (*model.Settings, error)
	UpdateSettings(s *model.Settings) error
}

// ==================== Access Log ====================

type AccessLogRecord struct {
	ID           int64  `json:"id"`
	Timestamp    string `json:"timestamp"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	Status       int    `json:"status"`
	DurationMs   int    `json:"duration_ms"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent"`
	RequestBody  string `json:"request_body"`
	ResponseBody string `json:"response_body"`
	Error        string `json:"error"`
}

type AccessLogQuery struct {
	Page      int
	PageSize  int
	Method    string
	Path      string
	StatusMin int
	StatusMax int
	From      string
	To        string
	Sort      string
}

type AccessLogQueryResult struct {
	Items []AccessLogRecord `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"page_size"`
}

type AccessLogRepository interface {
	InsertRecord(method, path string, status int, durationMs int, clientIP, userAgent, requestBody, responseBody, errMsg string) (int64, error)
	Query(q AccessLogQuery) (*AccessLogQueryResult, error)
	DeleteRecord(id int64) error
	ClearRecords() error
	StartDailyCleanup(days int)
}

// ==================== Task Queue ====================

type TaskRepository interface {
	Insert(task *model.TaskRecord) error
	Get(id int64) (*model.TaskRecord, error)
	List(page, pageSize int, status string) ([]model.TaskRecord, int, error)
	Update(task *model.TaskRecord) error
	Delete(id int64) error
	FindPending() ([]*model.TaskRecord, error)
}
```

- [ ] **Step 2: Run `go vet ./...` to verify no syntax errors**

- [ ] **Step 3: Commit**

```bash
GIT_MASTER=1 git add backend/internal/repository/interfaces.go
GIT_MASTER=1 git commit -m "refactor: define repository interfaces for data layer decoupling"
```

---

### Task 2: Update Handlers to Use Interfaces

**Files:**
- Modify: `backend/internal/handler/history.go` — `var historyRepo *repository.HistoryRepo` → `var historyRepo repository.HistoryRepository`, `NewHistoryHandler`, `SetHistoryRepo`, `SetRepo`
- Modify: `backend/internal/handler/asset.go` — same pattern
- Modify: `backend/internal/handler/storyboard.go` — `*repository.StoryboardRepo` → `repository.StoryboardRepository`
- Modify: `backend/internal/handler/settings.go` — `*repository.SettingsRepo` → `repository.SettingsRepository`
- Modify: `backend/internal/handler/access_log.go` — `*repository.AccessLogRepo` → `repository.AccessLogRepository`
- Modify: `backend/internal/repository/history.go` — remove `PendingVideoInfo` (moved to interfaces.go), keep `type HistoryRepo struct { db *sql.DB }`
- Modify: `backend/internal/repository/access_log.go` — remove `AccessLogRecord`, `AccessLogQuery`, `AccessLogQueryResult` (moved to interfaces.go)

- [ ] **Step 1: Change HistoryHandler and globals in `history.go`**

Change:
```go
var historyRepo *repository.HistoryRepo
// ...
func SetHistoryRepo(repo *repository.HistoryRepo) { ... }
// ...
type HistoryHandler struct { repo *repository.HistoryRepo }
func NewHistoryHandler(repo *repository.HistoryRepo) *HistoryHandler { ... }
func (h *HistoryHandler) SetRepo(repo *repository.HistoryRepo) { ... }
```

To:
```go
var historyRepo repository.HistoryRepository
// ...
func SetHistoryRepo(repo repository.HistoryRepository) { ... }
// ...
type HistoryHandler struct { repo repository.HistoryRepository }
func NewHistoryHandler(repo repository.HistoryRepository) *HistoryHandler { ... }
func (h *HistoryHandler) SetRepo(repo repository.HistoryRepository) { ... }
```

- [ ] **Step 2: Change AssetHandler in `asset.go`**

Same pattern: `*repository.HistoryRepo` → `repository.HistoryRepository`

- [ ] **Step 3: Change StoryboardHandler in `storyboard.go`**

`*repository.StoryboardRepo` → `repository.StoryboardRepository` (constructors, struct fields, SetRepo)

- [ ] **Step 4: Change SettingsHandler in `settings.go`**

`*repository.SettingsRepo` → `repository.SettingsRepository`

- [ ] **Step 5: Change AccessLogHandler in `access_log.go`**

`*repository.AccessLogRepo` → `repository.AccessLogRepository`

- [ ] **Step 6: Remove duplicated types from `repository/history.go`**

Delete the `type PendingVideoInfo struct` block from `history.go` (now in `interfaces.go`).

- [ ] **Step 7: Remove duplicated types from `repository/access_log.go`**

Delete `AccessLogRecord`, `AccessLogQuery`, `AccessLogQueryResult` from `access_log.go`.

- [ ] **Step 8: Run `go vet ./...` to verify**

Expected: clean output. If errors about unused imports — fix them.

- [ ] **Step 9: Commit**

```bash
GIT_MASTER=1 git add backend/internal/handler/history.go backend/internal/handler/asset.go backend/internal/handler/storyboard.go backend/internal/handler/settings.go backend/internal/handler/access_log.go backend/internal/repository/history.go backend/internal/repository/access_log.go
GIT_MASTER=1 git commit -m "refactor: handlers now depend on repository interfaces"
```

---

### Task 3: Update main.go to Wire Interfaces

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Change `main.go` variable types and `dbReplaceFunc`**

All handler constructors now accept interfaces. Since `*repository.HistoryRepo` implements `repository.HistoryRepository` (Go duck typing), no adapter code needed — just change the type annotations in `dbReplaceFunc` and the `func() *sql.DB` getter.

Key changes:
- `handler.SetHistoryRepo(histRepo)` — `histRepo` is `*repository.HistoryRepo` which satisfies `repository.HistoryRepository`
- `handler.NewHistoryHandler(histRepo)` — same
- `accessLogRepo` passed to `handler.NewAccessLogHandler` — `*repository.AccessLogRepo` satisfies `repository.AccessLogRepository`
- etc.

No actual code changes needed beyond ensuring the import paths are correct.

- [ ] **Step 2: Run `go vet ./...`**

- [ ] **Step 3: Run `go build ./...`**

- [ ] **Step 4: Commit**

```bash
GIT_MASTER=1 git add backend/cmd/server/main.go
GIT_MASTER=1 git commit -m "refactor: wire handlers with repository interfaces in main.go"
```

Phase 1 complete. The code now uses interfaces everywhere while still running on SQLite.

---

### Task 4: Add GORM Dependencies and OpenDB Factory

**Files:**
- Create: `backend/internal/repository/gorm/gorm.go`
- Create: `backend/internal/repository/gorm/models.go`
- Modify: `backend/go.mod` / `backend/go.sum`

- [ ] **Step 1: Install GORM dependencies**

```bash
cd backend
go get gorm.io/gorm
go get gorm.io/driver/sqlite
```

- [ ] **Step 2: Create `gorm/models.go` — GORM model structs**

```go
package gorm

import "time"

type History struct {
	ID     int64   `gorm:"primaryKey"`
	Time   string  `gorm:"index"`
	Mode   string  `gorm:"index"`
	Prompt string
	Images string  // JSON array
	Extra  *string
}

func (History) TableName() string { return "history" }

type Favorite struct {
	HistoryID int64 `gorm:"primaryKey"`
}

func (Favorite) TableName() string { return "favorites" }

type StoryboardProject struct {
	ID        int64           `gorm:"primaryKey"`
	Title     string
	Script    string
	CreatedAt string
	UpdatedAt string
	Shots     []StoryboardShot `gorm:"foreignKey:ProjectID"`
}

func (StoryboardProject) TableName() string { return "storyboard_projects" }

type StoryboardShot struct {
	ID             int64  `gorm:"primaryKey"`
	ProjectID      int64  `gorm:"index"`
	Sequence       int
	Prompt         string
	Type           string
	ReferenceImage string `gorm:"column:reference_image"`
	Status         string
	ResultVideo    string `gorm:"column:result_video"`
	TaskID         string `gorm:"column:task_id"`
	CreatedAt      string
}

func (StoryboardShot) TableName() string { return "storyboard_shots" }

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

func (Setting) TableName() string { return "settings" }

type AccessLog struct {
	ID           int64  `gorm:"primaryKey"`
	Timestamp    string
	Method       string `gorm:"index"`
	Path         string
	Status       int
	DurationMs   int    `gorm:"column:duration_ms"`
	ClientIP     string `gorm:"column:client_ip"`
	UserAgent    string `gorm:"column:user_agent"`
	RequestBody  string `gorm:"column:request_body;type:text"`
	ResponseBody string `gorm:"column:response_body;type:text"`
	Error        string
}

func (AccessLog) TableName() string { return "access_logs" }

type TaskRecord struct {
	ID          int64      `gorm:"primaryKey"`
	Type        string     `gorm:"index"`
	Status      string     `gorm:"index"`
	Params      string     `gorm:"type:text"`
	Result      *string    `gorm:"type:text"`
	Progress    int
	Error       *string    `gorm:"type:text"`
	RetryCount  int        `gorm:"column:retry_count"`
	CreatedAt   string     `gorm:"column:created_at"`
	UpdatedAt   string     `gorm:"column:updated_at"`
	CompletedAt *string    `gorm:"column:completed_at"`
}

func (TaskRecord) TableName() string { return "task_queue" }
```

- [ ] **Step 3: Create `gorm/gorm.go` — OpenDB factory**

```go
package gorm

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DBConfig struct {
	Driver string // "sqlite" | "postgres" | "mysql"
	DSN    string
}

func OpenDB(cfg DBConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN + "?_journal_mode=WAL&_busy_timeout=5000")
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Driver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	if err := db.AutoMigrate(
		&History{}, &Favorite{},
		&StoryboardProject{}, &StoryboardShot{},
		&Setting{}, &AccessLog{}, &TaskRecord{},
	); err != nil {
		return nil, fmt.Errorf("自动迁移失败: %w", err)
	}
	return db, nil
}
```

- [ ] **Step 4: Run `go build ./...` to verify GORM compiles**

- [ ] **Step 5: Commit**

```bash
GIT_MASTER=1 git add backend/internal/repository/gorm/ backend/go.mod backend/go.sum
GIT_MASTER=1 git commit -m "feat: add GORM models and OpenDB factory"
```

---

### Task 5: GORM HistoryRepository Implementation

**Files:**
- Create: `backend/internal/repository/gorm/history.go`

- [ ] **Step 1: Implement `HistoryRepository` interface using GORM**

`backend/internal/repository/gorm/history.go`:

```go
package gorm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
	"gorm.io/gorm"
)

type HistoryRepository struct {
	db *gorm.DB
}

func NewHistoryRepository(db *gorm.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) InsertRecord(prompt string, images []string, mode string, extra any) (int64, error) {
	imagesJSON, _ := json.Marshal(images)
	var extraStr *string
	if extra != nil {
		b, _ := json.Marshal(extra)
		s := string(b)
		extraStr = &s
	}
	h := History{Prompt: prompt, Mode: mode, Images: string(imagesJSON), Extra: extraStr}
	if err := r.db.Create(&h).Error; err != nil {
		return 0, err
	}
	return h.ID, nil
}

func (r *HistoryRepository) GetRecords(limit int) ([]model.HistoryRecord, error) {
	var hs []History
	if err := r.db.Order("id DESC").Limit(limit).Find(&hs).Error; err != nil {
		return nil, err
	}
	return toHistoryRecords(hs), nil
}

func (r *HistoryRepository) GetRecordsPaginated(page, perPage int, assetType, search string, favIDs map[int64]bool) ([]model.HistoryRecord, int, error) {
	var total int64
	query := r.db.Model(&History{})
	if assetType != "" {
		switch assetType {
		case "image":
			query = query.Where("mode IN (?)", []string{"text2image", "image2image", "batch"})
		case "video":
			query = query.Where("mode IN (?)", []string{"text2video", "image2video", "multi_image_video"})
		}
	}
	if search != "" {
		query = query.Where("prompt LIKE ?", "%"+search+"%")
	}
	query.Count(&total)

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var hs []History
	if err := query.Order("id DESC").Limit(perPage).Offset(offset).Find(&hs).Error; err != nil {
		return nil, 0, err
	}
	return toHistoryRecords(hs), int(total), nil
}

func (r *HistoryRepository) GetRecordsByIDs(ids []int64) ([]model.HistoryRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var hs []History
	if err := r.db.Where("id IN ?", ids).Order("id DESC").Find(&hs).Error; err != nil {
		return nil, err
	}
	return toHistoryRecords(hs), nil
}

func (r *HistoryRepository) DeleteRecord(id int64) error {
	return r.db.Delete(&History{}, id).Error
}

func (r *HistoryRepository) DeleteRecords(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("id IN ?", ids).Delete(&History{}).Error
}

func (r *HistoryRepository) ClearRecords() error {
	return r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&History{}).Error
}

func (r *HistoryRepository) UpdateRecordImages(id int64, images []string) error {
	imagesJSON, _ := json.Marshal(images)
	return r.db.Model(&History{}).Where("id = ?", id).Update("images", string(imagesJSON)).Error
}

func (r *HistoryRepository) FindByTaskId(taskId int64) (int64, error) {
	var h History
	err := r.db.Where("json_extract(extra, '$.taskId') = ?", taskId).Or("json_extract(extra, '$.taskId') = ?", fmt.Sprintf("%d", taskId)).Order("id DESC").First(&h).Error
	if err != nil {
		return 0, err
	}
	return h.ID, nil
}

func (r *HistoryRepository) FindPendingVideos() ([]repository.PendingVideoInfo, error) {
	var hs []History
	if err := r.db.Where("images = '[]' AND extra IS NOT NULL AND extra != ''").Order("id DESC").Find(&hs).Error; err != nil {
		return nil, err
	}
	var results []repository.PendingVideoInfo
	for _, h := range hs {
		if !strings.HasSuffix(h.Mode, "video") && h.Mode != "video" {
			continue
		}
		if h.Extra == nil {
			continue
		}
		var extra map[string]any
		if err := json.Unmarshal([]byte(*h.Extra), &extra); err != nil {
			continue
		}
		taskID := ""
		switch v := extra["taskId"].(type) {
		case string:
			taskID = v
		case float64:
			taskID = fmt.Sprintf("%.0f", v)
		}
		if taskID == "" {
			continue
		}
		results = append(results, repository.PendingVideoInfo{
			ID:     h.ID,
			TaskID: taskID,
			Prompt: h.Prompt,
			Mode:   h.Mode,
		})
	}
	return results, nil
}

func (r *HistoryRepository) TrimRecords(max int) error {
	sub := r.db.Select("id").Order("id DESC").Limit(max)
	return r.db.Where("id NOT IN (?)", sub).Delete(&History{}).Error
}

func (r *HistoryRepository) ToggleFavorite(historyID int64, favorite bool) error {
	if favorite {
		return r.db.Create(&Favorite{HistoryID: historyID}).Error
	}
	return r.db.Where("history_id = ?", historyID).Delete(&Favorite{}).Error
}

func (r *HistoryRepository) GetFavoriteIDs() (map[int64]bool, error) {
	var favs []Favorite
	if err := r.db.Find(&favs).Error; err != nil {
		return nil, err
	}
	result := make(map[int64]bool, len(favs))
	for _, f := range favs {
		result[f.HistoryID] = true
	}
	return result, nil
}

// toHistoryRecords 转换 GORM History → model.HistoryRecord
func toHistoryRecords(hs []History) []model.HistoryRecord {
	records := make([]model.HistoryRecord, 0, len(hs))
	for _, h := range hs {
		var images []string
		json.Unmarshal([]byte(h.Images), &images)
		if images == nil {
			images = []string{}
		}
		rec := model.HistoryRecord{
			ID:     h.ID,
			Time:   h.Time,
			Mode:   h.Mode,
			Prompt: h.Prompt,
			Images: images,
		}
		if h.Extra != nil {
			var extra any
			if err := json.Unmarshal([]byte(*h.Extra), &extra); err == nil {
				rec.Extra = extra
			}
		}
		records = append(records, rec)
	}
	return records
}
```

- [ ] **Step 2: Run `go vet ./...`**

- [ ] **Step 3: Commit**

```bash
GIT_MASTER=1 git add backend/internal/repository/gorm/history.go
GIT_MASTER=1 git commit -m "feat: GORM HistoryRepository implementation"
```

---

### Task 6: GORM Remaining Repository Implementations

**Files:**
- Create: `backend/internal/repository/gorm/storyboard.go`
- Create: `backend/internal/repository/gorm/settings.go`
- Create: `backend/internal/repository/gorm/access_log.go`
- Create: `backend/internal/repository/gorm/task.go`

- [ ] **Step 1: Implement `SettingsRepository` in `gorm/settings.go`**

```go
package gorm

import (
	"github.com/agnes-image-tool/backend/internal/model"
	"gorm.io/gorm"
)

type SettingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) GetSettings() (*model.Settings, error) {
	s := &model.Settings{
		StorageTarget:   "local",
		LocalImageDir:   "images",
		LocalVideoDir:   "videos",
		GithubImagePath: "outputs/images",
		GithubVideoPath: "outputs/videos",
	}
	var settings []Setting
	if err := r.db.Find(&settings).Error; err != nil {
		return nil, err
	}
	for _, kv := range settings {
		switch kv.Key {
		case "storage_target":
			s.StorageTarget = kv.Value
		case "local_image_dir":
			s.LocalImageDir = kv.Value
		case "local_video_dir":
			s.LocalVideoDir = kv.Value
		case "github_image_path":
			s.GithubImagePath = kv.Value
		case "github_video_path":
			s.GithubVideoPath = kv.Value
		}
	}
	return s, nil
}

func (r *SettingsRepository) UpdateSettings(s *model.Settings) error {
	pairs := map[string]string{
		"storage_target":    s.StorageTarget,
		"local_image_dir":   s.LocalImageDir,
		"local_video_dir":   s.LocalVideoDir,
		"github_image_path": s.GithubImagePath,
		"github_video_path": s.GithubVideoPath,
	}
	for k, v := range pairs {
		if err := r.db.Where("key = ?", k).Assign(Setting{Value: v}).FirstOrCreate(&Setting{Key: k}).Error; err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 2: Implement `StoryboardRepository` in `gorm/storyboard.go`**

File: `backend/internal/repository/gorm/storyboard.go`
```go
package gorm

import (
	"github.com/agnes-image-tool/backend/internal/model"
	"gorm.io/gorm"
)

type StoryboardRepository struct {
	db *gorm.DB
}

func NewStoryboardRepository(db *gorm.DB) *StoryboardRepository {
	return &StoryboardRepository{db: db}
}

func (r *StoryboardRepository) ListProjects() ([]model.StoryboardProject, error) {
	var projects []StoryboardProject
	if err := r.db.Order("updated_at DESC").Find(&projects).Error; err != nil {
		return nil, err
	}
	result := make([]model.StoryboardProject, len(projects))
	for i, p := range projects {
		var shotCount int64
		r.db.Model(&StoryboardShot{}).Where("project_id = ?", p.ID).Count(&shotCount)
		result[i] = model.StoryboardProject{
			ID:        p.ID,
			Title:     p.Title,
			Script:    p.Script,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
			ShotCount: int(shotCount),
		}
	}
	return result, nil
}

func (r *StoryboardRepository) CreateProject(title, script string) (int64, error) {
	p := StoryboardProject{Title: title, Script: script}
	if err := r.db.Create(&p).Error; err != nil {
		return 0, err
	}
	return p.ID, nil
}

func (r *StoryboardRepository) GetProject(id int64) (*model.StoryboardProject, error) {
	var p StoryboardProject
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &model.StoryboardProject{
		ID:        p.ID,
		Title:     p.Title,
		Script:    p.Script,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}, nil
}

func (r *StoryboardRepository) UpdateProject(id int64, title, script string) error {
	return r.db.Model(&StoryboardProject{}).Where("id = ?", id).Updates(map[string]any{
		"title": title, "script": script, "updated_at": "datetime('now')",
	}).Error
}

func (r *StoryboardRepository) DeleteProject(id int64) error {
	return r.db.Delete(&StoryboardProject{}, id).Error
}

func (r *StoryboardRepository) DuplicateProject(id int64, newTitle string) (int64, error) {
	tx := r.db.Begin()
	var orig StoryboardProject
	if err := tx.First(&orig, id).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	dup := StoryboardProject{Title: newTitle, Script: orig.Script}
	if err := tx.Create(&dup).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	var shots []StoryboardShot
	if err := tx.Where("project_id = ?", id).Find(&shots).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	for _, s := range shots {
		s.ID = 0
		s.ProjectID = dup.ID
		if err := tx.Create(&s).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
	}
	tx.Commit()
	return dup.ID, nil
}

func (r *StoryboardRepository) CreateShot(projectID int64, shot model.StoryboardShot) (int64, error) {
	s := StoryboardShot{
		ProjectID:      projectID,
		Sequence:       shot.Sequence,
		Prompt:         shot.Prompt,
		Type:           shot.Type,
		ReferenceImage: shot.ReferenceImage,
		Status:         "pending",
	}
	if err := r.db.Create(&s).Error; err != nil {
		return 0, err
	}
	return s.ID, nil
}

func (r *StoryboardRepository) UpdateShot(id int64, shot model.StoryboardShot) error {
	return r.db.Model(&StoryboardShot{}).Where("id = ?", id).Updates(map[string]any{
		"prompt":          shot.Prompt,
		"type":            shot.Type,
		"reference_image": shot.ReferenceImage,
		"status":          shot.Status,
		"result_video":    shot.ResultVideo,
		"task_id":         shot.TaskID,
	}).Error
}

func (r *StoryboardRepository) DeleteShot(id int64) error {
	return r.db.Delete(&StoryboardShot{}, id).Error
}

func (r *StoryboardRepository) ReorderShots(projectID int64, shotIDs []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, sid := range shotIDs {
			if err := tx.Model(&StoryboardShot{}).Where("id = ? AND project_id = ?", sid, projectID).Update("sequence", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *StoryboardRepository) GetShotsByProject(projectID int64) ([]model.StoryboardShot, error) {
	var shots []StoryboardShot
	if err := r.db.Where("project_id = ?", projectID).Order("sequence ASC").Find(&shots).Error; err != nil {
		return nil, err
	}
	result := make([]model.StoryboardShot, len(shots))
	for i, s := range shots {
		result[i] = model.StoryboardShot{
			ID:              s.ID,
			ProjectID:       s.ProjectID,
			Sequence:        s.Sequence,
			Prompt:          s.Prompt,
			Type:            s.Type,
			ReferenceImage:  s.ReferenceImage,
			Status:          model.ShotStatus(s.Status),
			ResultVideo:     s.ResultVideo,
			TaskID:          s.TaskID,
			CreatedAt:       s.CreatedAt,
		}
	}
	return result, nil
}

func (r *StoryboardRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
```

- [ ] **Step 3: Implement `AccessLogRepository` in `gorm/access_log.go`**

File: `backend/internal/repository/gorm/access_log.go`
```go
package gorm

import (
	"log"
	"strings"
	"time"

	"github.com/agnes-image-tool/backend/internal/repository"
	"gorm.io/gorm"
)

type AccessLogRepository struct {
	db *gorm.DB
}

func NewAccessLogRepository(db *gorm.DB) *AccessLogRepository {
	return &AccessLogRepository{db: db}
}

func (r *AccessLogRepository) InsertRecord(method, path string, status int, durationMs int, clientIP, userAgent, requestBody, responseBody, errMsg string) (int64, error) {
	al := AccessLog{
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		Method:       method,
		Path:         path,
		Status:       status,
		DurationMs:   durationMs,
		ClientIP:     clientIP,
		UserAgent:    userAgent,
		RequestBody:  requestBody,
		ResponseBody: responseBody,
		Error:        errMsg,
	}
	if err := r.db.Create(&al).Error; err != nil {
		return 0, err
	}
	return al.ID, nil
}

func (r *AccessLogRepository) Query(q repository.AccessLogQuery) (*repository.AccessLogQueryResult, error) {
	var total int64
	query := r.db.Model(&AccessLog{})
	if q.Method != "" {
		query = query.Where("method = ?", q.Method)
	}
	if q.Path != "" {
		query = query.Where("path LIKE ?", "%"+q.Path+"%")
	}
	if q.StatusMin > 0 {
		query = query.Where("status >= ?", q.StatusMin)
	}
	if q.StatusMax > 0 {
		query = query.Where("status <= ?", q.StatusMax)
	}
	if q.From != "" {
		query = query.Where("timestamp >= ?", q.From)
	}
	if q.To != "" {
		query = query.Where("timestamp <= ?", q.To)
	}
	query.Count(&total)

	order := "id DESC"
	if strings.ToLower(q.Sort) == "asc" {
		order = "id ASC"
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 50
	}
	offset := (q.Page - 1) * q.PageSize

	var logs []AccessLog
	if err := query.Order(order).Limit(q.PageSize).Offset(offset).Find(&logs).Error; err != nil {
		return nil, err
	}

	items := make([]repository.AccessLogRecord, len(logs))
	for i, l := range logs {
		items[i] = repository.AccessLogRecord{
			ID:           l.ID,
			Timestamp:    l.Timestamp,
			Method:       l.Method,
			Path:         l.Path,
			Status:       l.Status,
			DurationMs:   l.DurationMs,
			ClientIP:     l.ClientIP,
			UserAgent:    l.UserAgent,
			RequestBody:  l.RequestBody,
			ResponseBody: l.ResponseBody,
			Error:        l.Error,
		}
	}
	return &repository.AccessLogQueryResult{Items: items, Total: int(total), Page: q.Page, Size: q.PageSize}, nil
}

func (r *AccessLogRepository) DeleteRecord(id int64) error {
	return r.db.Delete(&AccessLog{}, id).Error
}

func (r *AccessLogRepository) ClearRecords() error {
	return r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&AccessLog{}).Error
}

func (r *AccessLogRepository) StartDailyCleanup(days int) {
	go func() {
		for {
			cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
			if err := r.db.Where("timestamp < ?", cutoff).Delete(&AccessLog{}).Error; err != nil {
				log.Printf("[AccessLog] 清理过期日志失败: %v", err)
			}
			time.Sleep(24 * time.Hour)
		}
	}()
}
```

- [ ] **Step 4: Implement `TaskRepository` in `gorm/task.go`**

File: `backend/internal/repository/gorm/task.go`
```go
package gorm

import (
	"github.com/agnes-image-tool/backend/internal/model"
	"gorm.io/gorm"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Insert(task *model.TaskRecord) error {
	t := TaskRecord{
		Type:    task.Type,
		Status:  task.Status,
		Params:  task.Params,
		Progress: task.Progress,
	}
	return r.db.Create(&t).Error
}

func (r *TaskRepository) Get(id int64) (*model.TaskRecord, error) {
	var t TaskRecord
	if err := r.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return toTaskRecord(&t), nil
}

func (r *TaskRepository) List(page, pageSize int, status string) ([]model.TaskRecord, int, error) {
	var total int64
	query := r.db.Model(&TaskRecord{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var ts []TaskRecord
	if err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&ts).Error; err != nil {
		return nil, 0, err
	}
	records := make([]model.TaskRecord, len(ts))
	for i, t := range ts {
		records[i] = *toTaskRecord(&t)
	}
	return records, int(total), nil
}

func (r *TaskRepository) Update(task *model.TaskRecord) error {
	return r.db.Model(&TaskRecord{}).Where("id = ?", task.ID).Updates(map[string]any{
		"status":     task.Status,
		"result":     task.Result,
		"progress":   task.Progress,
		"error":      task.Error,
		"retry_count": task.RetryCount,
	}).Error
}

func (r *TaskRepository) Delete(id int64) error {
	return r.db.Delete(&TaskRecord{}, id).Error
}

func (r *TaskRepository) FindPending() ([]*model.TaskRecord, error) {
	var ts []TaskRecord
	if err := r.db.Where("status IN ?", []string{"pending", "processing"}).Order("created_at ASC").Find(&ts).Error; err != nil {
		return nil, err
	}
	result := make([]*model.TaskRecord, len(ts))
	for i, t := range ts {
		result[i] = toTaskRecord(&t)
	}
	return result, nil
}

func toTaskRecord(t *TaskRecord) *model.TaskRecord {
	return &model.TaskRecord{
		ID:         t.ID,
		Type:       t.Type,
		Status:     model.TaskStatus(t.Status),
		Params:     t.Params,
		Result:     t.Result,
		Progress:   t.Progress,
		Error:      t.Error,
		RetryCount: t.RetryCount,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}
```

- [ ] **Step 5: Run `go vet ./...`**

- [ ] **Step 6: Commit**

```bash
GIT_MASTER=1 git add backend/internal/repository/gorm/
GIT_MASTER=1 git commit -m "feat: GORM implementations for all repositories"
```

---

### Task 7: Switch main.go to GORM

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Replace SQLite repo initialization with GORM**

In `main.go`:
- Replace `repository.NewHistoryRepo(dbPath)` → `gorm.OpenDB(gorm.DBConfig{Driver: "sqlite", DSN: dbPath})` to get `*gorm.DB`
- Replace all `repository.NewXxxRepo(...)` with `gorm.NewXxxRepository(db)` calls
- Remove `settingsRepo.InitTable()` (AutoMigrate handles it)
- Remove `taskRepo.InitTable()`
- Remove `histRepo.Close()` / `storyboardRepo.Close()` (GORM DB closed via `db.DB()`)

- [ ] **Step 2: Update `dbReplaceFunc` to use GORM**

The `dbReplaceFunc` in main.go closes old DB, creates new GORM DB, and re-creates all repositories.

- [ ] **Step 3: Run `go vet ./...` and `go build ./...`**

- [ ] **Step 4: Start server and verify existing data loads**

```bash
cd backend && go run ./cmd/server
# Verify: open http://localhost:8080/api/v1/history in browser
# Should return existing records from history.db
```

- [ ] **Step 5: Commit**

```bash
GIT_MASTER=1 git add backend/cmd/server/main.go
GIT_MASTER=1 git commit -m "feat: switch to GORM-backed repositories"
```

---

### Task 8: Multi-Backend Config + Cleanup

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/cmd/server/main.go`
- Delete: `backend/internal/repository/history.go`, `storyboard.go`, `settings.go`, `access_log.go`

- [ ] **Step 1: Add `DB_DRIVER` and `DB_DSN` to config**

In `config.go`:
```go
type Config struct {
    // ... existing fields
    DBDriver string  `json:"db_driver" env:"DB_DRIVER"`
    DBDSN    string  `json:"db_dsn" env:"DB_DSN"`
}
```
Default: `DBDriver = "sqlite"`, `DBDSN = "history.db"`

- [ ] **Step 2: Update main.go to read config and pass to `OpenDB`**

```go
db, err := gorm.OpenDB(gorm.DBConfig{
    Driver: cfg.DBDriver,
    DSN:    cfg.DBDSN,
})
```

- [ ] **Step 3: Remove old SQLite repository files**

Backup as `.bak` then delete: `history.go`, `storyboard.go`, `settings.go`, `access_log.go`.

- [ ] **Step 4: Run full build and verify**

```bash
cd backend
go vet ./...
go build ./...
```

- [ ] **Step 5: Commit**

```bash
GIT_MASTER=1 git add backend/internal/config/ backend/cmd/server/ backend/internal/repository/
GIT_MASTER=1 git commit -m "feat: multi-backend DB config, remove legacy SQLite repos"
```
