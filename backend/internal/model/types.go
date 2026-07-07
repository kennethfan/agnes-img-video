package model

// ==================== 配置 ====================

type Config struct {
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url"`
	Model        string `json:"model"`
	GithubToken  string `json:"github_token"`
	GithubRepo   string `json:"github_repo"`
	GithubBranch string `json:"github_branch"`
	ImageModel   string `json:"image_model,omitempty"`
	VideoModel   string `json:"video_model,omitempty"`
	ChatModel    string `json:"chat_model,omitempty"`
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
	IDs         []int64 `json:"ids" binding:"required"`
	DeleteFiles bool    `json:"delete_files"`
}

// ==================== 视频 ====================

type VideoCreateRequest struct {
	Prompt             string   `json:"prompt" binding:"required"`
	Duration           int      `json:"duration"`
	AspectRatio        string   `json:"aspect_ratio"`
	FrameRate          int      `json:"frame_rate"`
	NegativePrompt     string   `json:"negative_prompt"`
	Seed               *int     `json:"seed,omitempty"`
	NumInferenceSteps  *int     `json:"num_inference_steps,omitempty"`
	Width              *int     `json:"width,omitempty"`
	Height             *int     `json:"height,omitempty"`
	NumFrames          *int     `json:"num_frames,omitempty"`
	ImageURLs          []string `json:"image_urls,omitempty"`
	Mode               string   `json:"mode,omitempty"`
}

type VideoTaskResponse struct {
	TaskID string `json:"taskId"`
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
	Event   string `json:"-"` // progress / complete / error
	Progress int   `json:"progress,omitempty"`
	Status  string `json:"status,omitempty"`
	URL     string `json:"url,omitempty"`
	Seconds string `json:"seconds,omitempty"`
	Error   string `json:"error,omitempty"`
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
	Topic       string `json:"topic" binding:"required"`
	Duration    int    `json:"duration"`
	Style       string `json:"style"`
	Language    string `json:"language"`
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
	Model       string              `json:"model"`
	Messages    []ChatMessage       `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
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

// ==================== 资产管理 ====================

type AssetItem struct {
	ID        int64    `json:"id"`
	Mode      string   `json:"mode"`
	Prompt    string   `json:"prompt"`
	Files     []string `json:"files"`
	Thumbnail string   `json:"thumbnail"`
	Type      string   `json:"type"`
	Time      string   `json:"time"`
	Favorite  bool     `json:"favorite"`
}

type AssetListResponse struct {
	Items []AssetItem `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
}

type AssetFavoriteRequest struct {
	HistoryID int64 `json:"history_id" binding:"required"`
	Favorite  bool  `json:"favorite"`
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
	ID            int64  `json:"id"`
	ProjectID     int64  `json:"project_id"`
	Sequence      int    `json:"sequence"`
	Prompt        string `json:"prompt"`
	Type          string `json:"type"`
	ReferenceImage string `json:"reference_image"`
	Status        string `json:"status"`
	ResultVideo   string `json:"result_video"`
	TaskID        string `json:"task_id"`
	CreatedAt     string `json:"created_at"`
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
