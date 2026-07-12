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
	GetRecordsByProjectID(projectID int64) ([]model.HistoryRecord, error)
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
	DuplicateProject(id int64) (int64, error)
	ListShots(projectID int64) ([]model.StoryboardShot, error)
	CreateShot(projectID int64, seq int, prompt, shotType, refImage string) (int64, error)
	UpdateShot(id int64, prompt, shotType, refImage string) error
	DeleteShot(id int64) error
	ReorderShots(ids []int64) error
	GetShot(id int64) (*model.StoryboardShot, error)
	UpdateShotStatus(id int64, status, taskID string, taskRecordID int64) error
	UpdateShotResult(id int64, resultVideo string) error
	BatchCreateShots(projectID int64, prompts []string, shotType string) ([]model.StoryboardShot, error)
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
	Insert(record *AccessLogRecord) error
	Query(q AccessLogQuery) (*AccessLogQueryResult, error)
	Delete(id int64) error
	ClearAll() error
	StartDailyCleanup(retentionDays int)
}

// ==================== Asset ====================

type AssetRepository interface {
	Insert(asset *model.Asset) (int64, error)
	List(page, perPage int, assetType, search string, favoriteFilter bool) ([]model.Asset, int, error)
	GetByIDs(ids []int64) ([]model.Asset, error)
	GetByProjectID(projectID int64) ([]model.Asset, error)
	ToggleFavorite(id int64, favorite bool) error
	UpdateGithubURL(id int64, githubURL string) error
	UpdateStoragePaths(id int64, localPath, githubURL string) error
	Delete(ids []int64) error
}

// ==================== Task Queue ====================

type TaskRepository interface {
	InitTable() error
	CreateTask(taskType, params string) (int64, error)
	GetTask(id int64) (*model.TaskRecord, error)
	UpdateTaskStatus(id int64, status string, progress int, result, errMsg string) error
	UpdateTaskProgress(id int64, progress int) error
	UpdateRetryCount(id int64, count int) error
	CancelTaskAtomic(id int64) (bool, error)
	FindPendingTasks() ([]*model.TaskRecord, error)
	ListTasks(taskType, status string, limit, offset int) ([]*model.TaskRecord, error)
	ListByProjectID(projectID int64) ([]*model.TaskRecord, error)
	CleanupOlderThan(hours int) (int64, error)
}
