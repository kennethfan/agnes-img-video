package gorm

import "time"

type History struct {
	ID        int64   `gorm:"primaryKey"`
	Time      string  `gorm:"index"`
	Mode      string  `gorm:"index"`
	Prompt    string
	Images    string  // JSON array
	Extra     *string
	ProjectID int64   `json:"project_id" gorm:"column:project_id;index;default:0"`
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
	TaskRecordID   int64  `gorm:"column:task_record_id"`
	CreatedAt      string
}

func (StoryboardShot) TableName() string { return "storyboard_shots" }

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

func (Setting) TableName() string { return "settings" }

type Asset struct {
	ID          int64  `gorm:"primaryKey"`
	Mode        string `gorm:"index"`
	Prompt      string
	Type        string
	Time        string
	Favorite    bool
	OriginalURL string `gorm:"column:original_url"`
	LocalPath   string `gorm:"column:local_path"`
	GitHubURL   string `gorm:"column:github_url"`
	ProjectID   int64  `json:"project_id" gorm:"column:project_id;index;default:0"`
}

func (Asset) TableName() string { return "assets" }

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
	ID          int64   `gorm:"primaryKey"`
	Type        string  `gorm:"index"`
	Status      string  `gorm:"index"`
	Params      string  `gorm:"type:text"`
	Result      *string `gorm:"type:text"`
	Progress    int
	Error       *string `gorm:"type:text"`
	RetryCount  int     `gorm:"column:retry_count"`
	CreatedAt   string  `gorm:"column:created_at"`
	UpdatedAt   string  `gorm:"column:updated_at"`
	CompletedAt *string `gorm:"column:completed_at"`
	ProjectID   int64   `json:"project_id" gorm:"column:project_id;index;default:0"`
}

func (TaskRecord) TableName() string { return "task_queue" }

type Collection struct {
	ID        int64     `gorm:"primaryKey"`
	Name      string    `gorm:"not null;size:100"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Assets    []Asset   `gorm:"many2many:asset_collections;"`
}

func (Collection) TableName() string { return "collections" }

type AssetCollection struct {
	AssetID      int64 `gorm:"primaryKey"`
	CollectionID int64 `gorm:"primaryKey"`
}

func (AssetCollection) TableName() string { return "asset_collections" }

// PromptTemplate Prompt 模板/风格预设
type PromptTemplate struct {
	ID             int64     `gorm:"primaryKey"`
	Name           string    `gorm:"not null;size:200"`
	Type           string    `gorm:"default:template;size:20"` // template | preset
	Category       string    `gorm:"size:50"`                   // 人物/产品/背景/封面/海报/社媒/自定义
	Prompt         string    `gorm:"type:text"`
	NegativePrompt string    `gorm:"type:text"`
	Size           string    `gorm:"size:20"`
	Strength       float64   `gorm:"default:0.75"`
	Model          string    `gorm:"size:100"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (PromptTemplate) TableName() string { return "prompt_templates" }

// Project 创作项目
type Project struct {
	ID           int64         `gorm:"primaryKey" json:"id"`
	Title        string        `gorm:"size:200" json:"title"`
	Type         string        `gorm:"size:10;default:project" json:"type"`
	Brief        string        `gorm:"type:text" json:"brief"`
	AIResult     string        `gorm:"type:text" json:"ai_result"`
	Status       string        `gorm:"size:20;default:draft" json:"status"`
	CoverURL     string        `gorm:"type:text" json:"cover_url"`
	FinalURL     string        `gorm:"type:text" json:"final_url"`
	AssetIDs     string        `gorm:"type:text" json:"asset_ids"`
	Notes        string        `gorm:"type:text" json:"notes"`
	StepProgress string        `gorm:"type:text" json:"step_progress"`
	ComicData    string        `gorm:"type:text" json:"comic_data"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Steps        []ProjectStep `gorm:"foreignKey:ProjectID" json:"steps"`
}

func (Project) TableName() string { return "projects" }

// ProjectStep 项目步骤
type ProjectStep struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	ProjectID int64     `gorm:"index" json:"project_id"`
	StepType  string    `gorm:"size:20" json:"step_type"` // ideate | layout | generate | refine | finalize
	Position  int       `json:"position"`
	Input     string    `gorm:"type:text" json:"input"`
	Output    string    `gorm:"type:text" json:"output"`
	CreatedAt time.Time `json:"created_at"`
}

func (ProjectStep) TableName() string { return "project_steps" }
