package model

// ==================== 配置 ====================

type Config struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
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
	Time   string   `json:"time"`
	Mode   string   `json:"mode"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images"`
	Extra  any      `json:"extra,omitempty"`
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
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	URL      string `json:"url,omitempty"`
	Error    any    `json:"error,omitempty"`
	Seconds  string `json:"seconds,omitempty"`
	Size     string `json:"size,omitempty"`
}
