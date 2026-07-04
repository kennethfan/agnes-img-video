package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agnes-image-tool/backend/internal/model"
)

const (
	OutputDir   = "outputs"
	DownloadDir = "../outputs" // 相对于 backend/ 目录
)

type AgnesClient struct {
	apiKey     string
	baseURL    string
	client     *http.Client
	github     *GithubStorage
	imageModel string
	videoModel string
	chatModel  string
}

func (c *AgnesClient) SetGithubStorage(gs *GithubStorage) {
	c.github = gs
}

func (c *AgnesClient) GetGithubStorage() *GithubStorage {
	return c.github
}

func NewAgnesClient(apiKey, baseURL, imageModel, videoModel, chatModel string) *AgnesClient {
	return &AgnesClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		imageModel: imageModel,
		videoModel: videoModel,
		chatModel:  chatModel,
	}
}

// ==================== 图片生成 ====================

func (c *AgnesClient) TextToImage(prompt, size string, n int, negativePrompt string) ([]string, error) {
	if size == "" {
		size = "1024x1024"
	}
	if n <= 0 {
		n = 1
	}

	payload := map[string]any{
		"model":  c.imageModel,
		"prompt": prompt,
		"size":   size,
		"n":      n,
	}
	if negativePrompt != "" {
		payload["negative_prompt"] = negativePrompt
	}

	var resp model.AgnesImageResponse
	if err := c.doRequest("POST", "/images/generations", payload, &resp); err != nil {
		return nil, fmt.Errorf("文生图失败: %w", err)
	}

	urls := make([]string, 0, len(resp.Data))
	for _, d := range resp.Data {
		if d.URL != "" {
			urls = append(urls, d.URL)
		}
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("API 返回中未找到图片 URL")
	}
	return urls, nil
}

func (c *AgnesClient) ImageToImage(imageValue, prompt, size string, n int, strength float64, negativePrompt string) ([]string, error) {
	if size == "" {
		size = "1024x1024"
	}
	if n <= 0 {
		n = 1
	}
	if strength <= 0 {
		strength = 0.75
	}

	payload := map[string]any{
		"model":  c.imageModel,
		"prompt": prompt,
		"size":   size,
		"n":      n,
		"extra_body": map[string]any{
			"image":    []string{imageValue},
			"strength": strength,
		},
	}
	if negativePrompt != "" {
		payload["extra_body"].(map[string]any)["negative_prompt"] = negativePrompt
	}

	var resp model.AgnesImageResponse
	if err := c.doRequest("POST", "/images/generations", payload, &resp); err != nil {
		return nil, fmt.Errorf("图生图失败: %w", err)
	}

	urls := make([]string, 0, len(resp.Data))
	for _, d := range resp.Data {
		if d.URL != "" {
			urls = append(urls, d.URL)
		}
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("API 返回中未找到图片 URL")
	}
	return urls, nil
}

// ==================== 下载 ====================

// DownloadAndSave 下载文件到 outputs/ 目录
func (c *AgnesClient) DownloadAndSave(url, prefix string) (string, error) {
	// 确定目标目录
	outputDir := OutputDir
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		outputDir = DownloadDir
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405_000000")
	filename := fmt.Sprintf("%s_%s.png", prefix, timestamp)
	filepath := filepath.Join(outputDir, filename)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载返回非 200: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 如果配置了 GitHub 存储，同步上传并返回 GitHub URL
	if c.github != nil {
		remotePath := fmt.Sprintf("images/%s", filename)
		dlURL, err := c.github.UploadFile(filepath, remotePath)
		if err != nil {
			log.Printf("[GitHub] 上传图片失败: %v", err)
		} else {
			log.Printf("[GitHub] 图片已上传: %s", dlURL)
			return dlURL, nil
		}
	}

	return filepath, nil
}

// DownloadVideo 下载视频到本地
func (c *AgnesClient) DownloadVideo(url, prefix string) (string, error) {
	outputDir := OutputDir
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		outputDir = DownloadDir
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405_000000")
	filename := fmt.Sprintf("%s_%s.mp4", prefix, timestamp)
	filepath := filepath.Join(outputDir, filename)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("下载视频失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载视频返回非 200: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("创建视频文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("写入视频文件失败: %w", err)
	}

	// 如果配置了 GitHub 存储，同步上传并返回 GitHub URL
	if c.github != nil {
		remotePath := fmt.Sprintf("videos/%s", filename)
		dlURL, err := c.github.UploadFile(filepath, remotePath)
		if err != nil {
			log.Printf("[GitHub] 上传视频失败: %v", err)
		} else {
			log.Printf("[GitHub] 视频已上传: %s", dlURL)
			return dlURL, nil
		}
	}

	return filepath, nil
}

// ==================== 脚本生成 ====================

// GenerateScript 调用聊天 API 生成视频脚本
func (c *AgnesClient) GenerateScript(topic string, duration int, style, language string) (string, error) {
	systemPrompt := "你是一个专业的视频脚本撰写专家。请根据用户提供的主题，生成一份结构化的视频脚本。" +
		"脚本应包含：场景描述、旁白文案、镜头建议、时长分配。请用 Markdown 格式输出。"

	if language == "en" {
		systemPrompt = "You are a professional video script writer. Generate a structured video script based on the user's topic. " +
			"Include: scene descriptions, narration text, camera suggestions, and duration allocation. Output in Markdown format."
	}

	userPrompt := fmt.Sprintf("主题：%s\n视频时长：%d秒\n风格：%s",
		topic, duration, style)
	if style == "" {
		userPrompt = fmt.Sprintf("主题：%s\n视频时长：%d秒", topic, duration)
	}

	req := model.ChatCompletionRequest{
		Model: c.chatModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   2048,
	}

	var resp model.ChatCompletionResponse
	if err := c.doRequest("POST", "/chat/completions", req, &resp); err != nil {
		return "", fmt.Errorf("生成脚本失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("API 返回中未找到脚本内容")
	}

	return resp.Choices[0].Message.Content, nil
}

// ==================== 视频生成 ====================

// SubmitVideoTask 提交视频生成任务到 Agnes API
func (c *AgnesClient) SubmitVideoTask(payload map[string]any) (string, error) {
	url := c.baseURL + "/videos"
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化视频请求体失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建视频请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 视频提交可能需要更长时间
	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("提交视频任务失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取视频响应失败: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("视频API返回 %d: %s", resp.StatusCode, string(respBody))
	}

	var submitResp model.AgnesVideoSubmitResponse
	if err := json.Unmarshal(respBody, &submitResp); err != nil {
		return "", fmt.Errorf("解析视频响应失败: %w", err)
	}

	videoID := submitResp.VideoID
	if videoID == "" {
		videoID = submitResp.ID
	}
	if videoID == "" {
		return "", fmt.Errorf("API 返回中未找到视频任务 ID: %s", string(respBody))
	}

	return videoID, nil
}

// CheckVideoStatus 查询视频生成状态
func (c *AgnesClient) CheckVideoStatus(videoID string) (*model.AgnesVideoStatusResponse, error) {
	// 视频状态查询 URL 需要去掉 /v1 前缀
	baseDomain := strings.TrimSuffix(c.baseURL, "/v1")
	pollURL := fmt.Sprintf("%s/agnesapi?video_id=%s", baseDomain, videoID)

	req, err := http.NewRequest("GET", pollURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建状态查询请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询视频状态失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取状态响应失败: %w", err)
	}

	var status model.AgnesVideoStatusResponse
	if err := json.Unmarshal(respBody, &status); err != nil {
		return nil, fmt.Errorf("解析状态响应失败: %w", err)
	}

	// 兼容：Agnes API 有时将视频 URL 放在 remixed_from_video_id 字段
	if status.URL == "" && status.RemixedFromVideoID != "" {
		status.URL = status.RemixedFromVideoID
	}

	return &status, nil
}

// BuildVideoPayload 构建视频请求 payload（共享逻辑）
func (c *AgnesClient) BuildVideoPayload(prompt string, opts VideoOptions) map[string]any {
	payload := map[string]any{
		"model":  c.videoModel,
		"prompt": prompt,
	}

	if opts.Width != nil && opts.Height != nil {
		payload["width"] = *opts.Width
		payload["height"] = *opts.Height
	} else {
		size := ratioToSize(opts.AspectRatio)
		payload["width"] = size["width"]
		payload["height"] = size["height"]
	}

	// 计算帧数
	maxFrames := maxFramesForResolution(payload["width"].(int), payload["height"].(int))
	if opts.NumFrames != nil {
		payload["num_frames"] = minInt(*opts.NumFrames, maxFrames)
	} else {
		targetFrames := opts.Duration * opts.FrameRate
		n := (targetFrames - 1 + 4) / 8 // round(n)
		calculatedFrames := 8*n + 1
		payload["num_frames"] = minInt(maxInt(calculatedFrames, 9), maxFrames)
	}

	if opts.FrameRate > 0 {
		payload["frame_rate"] = opts.FrameRate
	}
	if opts.NegativePrompt != "" {
		payload["negative_prompt"] = opts.NegativePrompt
	}
	if opts.Seed != nil {
		payload["seed"] = *opts.Seed
	}
	if opts.NumInferenceSteps != nil {
		payload["num_inference_steps"] = *opts.NumInferenceSteps
	}

	return payload
}

// VideoOptions 视频参数
type VideoOptions struct {
	Duration          int
	AspectRatio       string
	FrameRate         int
	NegativePrompt    string
	Seed              *int
	NumInferenceSteps *int
	Width             *int
	Height            *int
	NumFrames         *int
	ImageURLs         []string
	Mode              string
	RecordType        string // "text2video" / "image2video" / "multi_image_video" — 用于历史记录
}

// VideoCompleteFunc 视频完成回调
type VideoCompleteFunc func(taskID, prompt, resultURL string, opts VideoOptions)

func ratioToSize(aspectRatio string) map[string]int {
	sizes := map[string]map[string]int{
		"16:9": {"width": 1152, "height": 768},
		"9:16": {"width": 768, "height": 1152},
		"1:1":  {"width": 768, "height": 768},
		"4:3":  {"width": 1024, "height": 768},
		"3:4":  {"width": 768, "height": 1024},
	}
	if s, ok := sizes[aspectRatio]; ok {
		return s
	}
	return sizes["16:9"]
}

func maxFramesForResolution(width, height int) int {
	maxDim := width
	if height > maxDim {
		maxDim = height
	}
	switch {
	case maxDim >= 1920:
		return 169
	case maxDim >= 1280:
		return 409
	default:
		return 961
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ==================== 内部方法 ====================

func (c *AgnesClient) doRequest(method, path string, payload, result any) error {
	url := c.baseURL + path
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
	}

	return nil
}

// UploadFileAsMultipart 以 multipart/form-data 上传文件
func UploadFileAsMultipart(filePath, fieldName, url, apiKey string) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	fw, err := w.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("创建 form file 失败: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return nil, fmt.Errorf("写入文件数据失败: %w", err)
	}
	w.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
