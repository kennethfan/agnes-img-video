package model

// ==================== 存储设置 ====================

type Settings struct {
	StorageTarget   string `json:"storage_target"`
	LocalImageDir   string `json:"local_image_dir"`
	LocalVideoDir   string `json:"local_video_dir"`
	GithubImagePath string `json:"github_image_path"`
	GithubVideoPath string `json:"github_video_path"`
}

// ==================== 配置 ====================

type Config struct {
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url"`
	ImageModel   string `json:"image_model,omitempty"`
	VideoModel   string `json:"video_model,omitempty"`
	ChatModel    string `json:"chat_model,omitempty"`
	GithubToken  string `json:"github_token"`
	GithubRepo   string `json:"github_repo"`
	GithubBranch string `json:"github_branch"`
	DBDriver     string `json:"db_driver,omitempty"`
	DBDSN        string `json:"db_dsn,omitempty"`
}

// ==================== 图片请求/响应 ====================

type TextToImageRequest struct {
	Prompt         string `json:"prompt" binding:"required"`
	Size           string `json:"size"`
	N              int    `json:"n"`
	NegativePrompt string `json:"negative_prompt"`
}

type ImageToImageRequest struct {
	Prompt         string  `json:"prompt" binding:"required"`
	Size           string  `json:"size"`
	N              int     `json:"n"`
	Strength       float64 `json:"strength"`
	NegativePrompt string  `json:"negative_prompt"`
}

type BatchRequest struct {
	Prompts []string `json:"prompts" binding:"required"`
	Size    string   `json:"size"`
}

type ImageResponse struct {
	Images []string `json:"images"`
}

// ==================== 历史记录 ====================

type HistoryRecord struct {
	ID     int64    `json:"id"`
	Time   string   `json:"time"`
	Mode   string   `json:"mode"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images"`
	Extra  any      `json:"extra,omitempty"`
}

type BatchDeleteRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

// ==================== 视频 ====================

type VideoCreateRequest struct {
	Prompt            string   `json:"prompt" binding:"required"`
	Duration          int      `json:"duration"`
	AspectRatio       string   `json:"aspect_ratio"`
	FrameRate         int      `json:"frame_rate"`
	NegativePrompt    string   `json:"negative_prompt"`
	Seed              *int     `json:"seed,omitempty"`
	NumInferenceSteps *int     `json:"num_inference_steps,omitempty"`
	Width             *int     `json:"width,omitempty"`
	Height            *int     `json:"height,omitempty"`
	NumFrames         *int     `json:"num_frames,omitempty"`
	ImageURLs         []string `json:"image_urls,omitempty"`
	Mode              string   `json:"mode,omitempty"`
}

type VideoTaskResponse struct {
	TaskID int64 `json:"taskId"`
}

type VideoStatus struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	URL      string `json:"url,omitempty"`
	Error    string `json:"error,omitempty"`
	Seconds  string `json:"seconds,omitempty"`
}

// ==================== SSE 事件 ====================

type VideoEvent struct {
	Event    string `json:"-"` // progress / complete / error
	Progress int    `json:"progress,omitempty"`
	Status   string `json:"status,omitempty"`
	URL      string `json:"url,omitempty"`
	Seconds  string `json:"seconds,omitempty"`
	Error    string `json:"error,omitempty"`
}

// ==================== Agnes API 请求/响应 ====================

type AgnesImagePayload struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size"`
	N              int    `json:"n"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	ExtraBody      any    `json:"extra_body,omitempty"`
}

type AgnesImageData struct {
	URL string `json:"url"`
}

type AgnesImageResponse struct {
	Data []AgnesImageData `json:"data"`
}

type AgnesVideoSubmitResponse struct {
	VideoID string `json:"video_id"`
	ID      string `json:"id,omitempty"`
}

type AgnesVideoStatusResponse struct {
	Status             string `json:"status"`
	Progress           int    `json:"progress"`
	URL                string `json:"url,omitempty"`
	RemixedFromVideoID string `json:"remixed_from_video_id,omitempty"`
	Error              any    `json:"error,omitempty"`
	Seconds            string `json:"seconds,omitempty"`
	Size               string `json:"size,omitempty"`
}

// ==================== 脚本生成 ====================

type ScriptGenRequest struct {
	Topic    string `json:"topic" binding:"required"`
	Duration int    `json:"duration"`
	Style    string `json:"style"`
	Language string `json:"language"`
}

type ScriptGenResponse struct {
	Script string `json:"script"`
}

// ==================== 点子库 ====================

type ExpandIdeaRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Tags    string `json:"tags"`
}

type ExpandIdeaResponse struct {
	Result string `json:"result"`
}

// ChatCompletionRequest OpenAI 兼容的聊天请求
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	Choices []ChatChoice `json:"choices"`
}

type ChatChoice struct {
	Message ChatMessage `json:"message"`
}

// ==================== 资产模型 ====================

// Asset 作品库模型（GORM + JSON 双用途）
type Asset struct {
	ID          int64  `json:"id" gorm:"primaryKey"`
	Mode        string `json:"mode" gorm:"index"`
	Prompt      string `json:"prompt"`
	Type        string `json:"type"`    // "image" | "video"
	Time        string `json:"time"`    // 保存时间
	Favorite    bool   `json:"favorite"`
	OriginalURL string `json:"original_url" gorm:"column:original_url"`
	LocalPath   string `json:"local_path" gorm:"column:local_path"`
	GitHubURL   string `json:"github_url" gorm:"column:github_url"`
}

// ==================== 资产管理 ====================

type AssetItem struct {
	ID          int64  `json:"id"`
	Mode        string `json:"mode"`
	Prompt      string `json:"prompt"`
	Type        string `json:"type"`
	Time        string `json:"time"`
	Favorite    bool   `json:"favorite"`
	OriginalURL string `json:"original_url"`
	LocalPath   string `json:"local_path"`
	GitHubURL   string `json:"github_url"`
	Thumbnail   string `json:"thumbnail"`
}

type AssetListResponse struct {
	Items []AssetItem `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
}

type AssetFavoriteRequest struct {
	AssetID  int64 `json:"asset_id" binding:"required"`
	Favorite bool  `json:"favorite"`
}

type AssetDeleteRequest struct {
	IDs         []int64 `json:"ids" binding:"required"`
	DeleteFiles bool    `json:"delete_files"`
}

// ==================== 故事板 ====================

type StoryboardProject struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Script    string `json:"script"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	ShotCount int    `json:"shot_count"`
}

type StoryboardShot struct {
	ID             int64  `json:"id"`
	ProjectID      int64  `json:"project_id"`
	Sequence       int    `json:"sequence"`
	Prompt         string `json:"prompt"`
	Type           string `json:"type"`
	ReferenceImage string `json:"reference_image"`
	Status         string `json:"status"`
	ResultVideo    string `json:"result_video"`
	TaskID         string `json:"task_id"`
	TaskRecordID   int64  `json:"task_record_id"`
	CreatedAt      string `json:"created_at"`
}

type CreateProjectRequest struct {
	Title  string `json:"title" binding:"required"`
	Script string `json:"script"`
}

type UpdateProjectRequest struct {
	Title  string `json:"title"`
	Script string `json:"script"`
}

type CreateShotRequest struct {
	Prompt         string `json:"prompt" binding:"required"`
	Type           string `json:"type"`
	ReferenceImage string `json:"reference_image"`
}

type UpdateShotRequest struct {
	Prompt         string `json:"prompt"`
	Type           string `json:"type"`
	ReferenceImage string `json:"reference_image"`
}

type ReorderShotsRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

// ==================== 异步任务队列 ====================

type TaskRecord struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Params      string `json:"params"`
	Result      string `json:"result,omitempty"`
	Progress    int    `json:"progress"`
	Error       string `json:"error,omitempty"`
	RetryCount  int    `json:"retry_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	CompletedAt string `json:"completed_at,omitempty"`
}

type TaskEvent struct {
	Event    string `json:"-"` // progress / complete / error
	Progress int    `json:"progress,omitempty"`
	Status   string `json:"status,omitempty"`
	Result   string `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
	Message  string `json:"message,omitempty"` // 重试说明等
}

type TaskCreateResponse struct {
	TaskID int64 `json:"taskId"`
}
