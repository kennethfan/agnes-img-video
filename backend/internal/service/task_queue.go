package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
)

const (
	defaultMaxConcurrentTasks = 10
	pollInterval              = 5 * time.Second
	maxPollTime               = 30 * time.Minute
	pollRetryMax              = 10
	defaultMaxRetries         = 3
	retryBackoffBase          = 5 * time.Second
)

// TaskCompleteFunc 任务完成回调（保存历史记录等）
type TaskCompleteFunc func(taskID int64, taskType, prompt, resultURL string)

// TaskQueue 统一异步任务队列
type TaskQueue struct {
	mu          sync.RWMutex
	repo        repository.TaskRepository
	client      *AgnesClient
	workerSem   chan struct{}
	subscribers map[int64]map[string]chan model.TaskEvent
	onComplete  TaskCompleteFunc
	ctx         context.Context
	cancel      context.CancelFunc
	cancelChans map[int64]chan struct{} // per-task 取消信号
}

// NewTaskQueue 创建任务队列
func NewTaskQueue(repo repository.TaskRepository, client *AgnesClient, maxConcurrent int) *TaskQueue {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultMaxConcurrentTasks
	}
	ctx, cancel := context.WithCancel(context.Background())
	tq := &TaskQueue{
		repo:        repo,
		client:      client,
		workerSem:   make(chan struct{}, maxConcurrent),
		subscribers: make(map[int64]map[string]chan model.TaskEvent),
		cancelChans: make(map[int64]chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}
	// 启动时恢复未完成任务
	go tq.recoverPending()
	// 定期清理过期任务
	go tq.cleanupLoop()
	return tq
}

// SetOnComplete 设置完成回调
func (tq *TaskQueue) SetOnComplete(fn TaskCompleteFunc) {
	tq.onComplete = fn
}

// SubmitTask 提交任务：写入 SQLite → 返回 taskId → 启动 Worker
func (tq *TaskQueue) SubmitTask(taskType string, paramsJSON string) (int64, error) {
	id, err := tq.repo.CreateTask(taskType, paramsJSON)
	if err != nil {
		return 0, err
	}

	if err := tq.submitExistingTask(id, taskType, paramsJSON); err != nil {
		return 0, err
	}
	return id, nil
}

// submitExistingTask 启动 worker goroutine（供 SubmitTask 和 RetryTask 共用）
func (tq *TaskQueue) submitExistingTask(id int64, taskType, paramsJSON string) error {
	cancelCh := make(chan struct{})
	tq.mu.Lock()
	tq.cancelChans[id] = cancelCh
	tq.mu.Unlock()

	select {
	case tq.workerSem <- struct{}{}:
		tq.mu.Lock()
		delete(tq.cancelChans, id)
		tq.mu.Unlock()
		go tq.worker(id, taskType, paramsJSON)
	default:
		log.Printf("[TaskQueue] 达到最大并发数，任务 %d 排队等待", id)
		go func() {
			select {
			case tq.workerSem <- struct{}{}:
				tq.mu.Lock()
				delete(tq.cancelChans, id)
				tq.mu.Unlock()
				// 再检查一次是否在排队期间被取消
				rec, _ := tq.repo.GetTask(id)
				if rec != nil && rec.Status == string(model.TaskStatusCancelled) {
					<-tq.workerSem
					return
				}
				tq.worker(id, taskType, paramsJSON)
			case <-cancelCh:
				return
			}
		}()
	}
	return nil
}

// GetTask 查询任务状态
func (tq *TaskQueue) GetTask(id int64) (*model.TaskRecord, error) {
	return tq.repo.GetTask(id)
}

// ListTasks 查询任务列表，按创建时间倒序
func (tq *TaskQueue) ListTasks(taskType, status string, limit, offset int) ([]*model.TaskRecord, error) {
	return tq.repo.ListTasks(taskType, status, limit, offset)
}

// CancelTask 取消 pending 状态的任务
func (tq *TaskQueue) CancelTask(id int64) error {
	rec, err := tq.repo.GetTask(id)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("任务不存在")
	}
	if rec.Status != string(model.TaskStatusPending) {
		return fmt.Errorf("只能取消 pending 状态的任务（当前: %s）", rec.Status)
	}

	cancelled, err := tq.repo.CancelTaskAtomic(id)
	if err != nil {
		return err
	}
	if !cancelled {
		return fmt.Errorf("任务已不在 pending 状态，无法取消")
	}

	// 通知等待中的 goroutine
	tq.mu.Lock()
	ch, ok := tq.cancelChans[id]
	delete(tq.cancelChans, id)
	tq.mu.Unlock()
	if ok {
		close(ch)
	}

	tq.notifySubscribers(id, model.TaskEvent{
		Event:  "progress",
		Status: string(model.TaskStatusCancelled),
	})

	log.Printf("[TaskQueue] 任务已取消: id=%d", id)
	return nil
}

// RetryTask 手动重试失败的任务
func (tq *TaskQueue) RetryTask(id int64) error {
	rec, err := tq.repo.GetTask(id)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("任务不存在")
	}
	if rec.Status != string(model.TaskStatusFailed) {
		return fmt.Errorf("只能重试失败的任务（当前: %s）", rec.Status)
	}

	// 重置状态
	tq.repo.UpdateTaskStatus(id, string(model.TaskStatusPending), 0, "", "")
	tq.repo.UpdateRetryCount(id, 0)

	// 重新提交
	return tq.submitExistingTask(id, rec.Type, rec.Params)
}

// Subscribe 注册 SSE 订阅者
func (tq *TaskQueue) Subscribe(taskID int64, subID string) chan model.TaskEvent {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.subscribers[taskID] == nil {
		tq.subscribers[taskID] = make(map[string]chan model.TaskEvent)
	}
	ch := make(chan model.TaskEvent, 10)
	tq.subscribers[taskID][subID] = ch
	return ch
}

// Unsubscribe 移除 SSE 订阅者
func (tq *TaskQueue) Unsubscribe(taskID int64, subID string) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if subs, ok := tq.subscribers[taskID]; ok {
		if ch, ok := subs[subID]; ok {
			close(ch)
			delete(subs, subID)
		}
		if len(subs) == 0 {
			delete(tq.subscribers, taskID)
		}
	}
}

// notifySubscribers 通知所有订阅者
func (tq *TaskQueue) notifySubscribers(taskID int64, event model.TaskEvent) {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	if subs, ok := tq.subscribers[taskID]; ok {
		for _, ch := range subs {
			select {
			case ch <- event:
			default:
			}
		}
	}
}

// worker 执行任务（goroutine 中运行）
func (tq *TaskQueue) worker(taskID int64, taskType, paramsJSON string) {
	defer func() {
		<-tq.workerSem
	}()

	// 检查是否已被取消
	rec, _ := tq.repo.GetTask(taskID)
	if rec != nil && rec.Status == string(model.TaskStatusCancelled) {
		log.Printf("[TaskQueue] 任务 %d 已取消，跳过执行", taskID)
		return
	}

	log.Printf("[TaskQueue] Worker 开始任务: id=%d type=%s", taskID, taskType)
	tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusProcessing), 0, "", "")
	tq.notifySubscribers(taskID, model.TaskEvent{
		Event:  "progress",
		Status: string(model.TaskStatusProcessing),
	})

	var lastErr error
	maxRetries := defaultMaxRetries

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := retryBackoffBase * time.Duration(1<<uint(attempt-1))
			log.Printf("[TaskQueue] 任务 %d 失败，第 %d/%d 次重试，等待 %v", taskID, attempt, maxRetries, backoff)

			tq.repo.UpdateRetryCount(taskID, attempt)
			tq.notifySubscribers(taskID, model.TaskEvent{
				Event:   "progress",
				Status:  string(model.TaskStatusProcessing),
				Message: fmt.Sprintf("失败后自动重试中 (%d/%d)", attempt, maxRetries),
			})

			time.Sleep(backoff)
		}

		var err error
		switch taskType {
		case string(model.TaskTypeTextToImage):
			err = tq.execTextToImage(taskID, paramsJSON)
		case string(model.TaskTypeImageToImage):
			err = tq.execImageToImage(taskID, paramsJSON)
		case string(model.TaskTypeBatch):
			err = tq.execBatch(taskID, paramsJSON)
		case string(model.TaskTypeTextToVideo):
			err = tq.execTextToVideo(taskID, paramsJSON)
		case string(model.TaskTypeImageToVideo):
			err = tq.execImageToVideo(taskID, paramsJSON)
		case string(model.TaskTypeMultiImageVideo):
			err = tq.execMultiImageVideo(taskID, paramsJSON)
		default:
			err = fmt.Errorf("未知任务类型: %s", taskType)
		}

		if err == nil {
			return // 成功
		}
		lastErr = err
	}

	// 全部重试失败
	errMsg := lastErr.Error()
	log.Printf("[TaskQueue] 任务失败: id=%d err=%s", taskID, errMsg)
	tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusFailed), 0, "", errMsg)
	tq.notifySubscribers(taskID, model.TaskEvent{
		Event: "error",
		Error: errMsg,
	})
}

// ==================== 图片任务执行 ====================

type imageParams struct {
	Prompt         string  `json:"prompt"`
	Size           string  `json:"size"`
	N              int     `json:"n"`
	NegativePrompt string  `json:"negative_prompt"`
	ImageValue     string  `json:"image_value,omitempty"` // base64 data URI 或 URL（image2image）
	Strength       float64 `json:"strength,omitempty"`
}

func (tq *TaskQueue) execTextToImage(taskID int64, paramsJSON string) error {
	var p struct {
		Prompt         string `json:"prompt"`
		Size           string `json:"size"`
		N              int    `json:"n"`
		NegativePrompt string `json:"negative_prompt"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	urls, err := tq.client.TextToImage(p.Prompt, p.Size, p.N, p.NegativePrompt)
	if err != nil {
		return err
	}

	resultJSON, _ := json.Marshal(urls)
	tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusCompleted), 100, string(resultJSON), "")
	tq.notifySubscribers(taskID, model.TaskEvent{
		Event:  "complete",
		Result: string(resultJSON),
	})

	// 触发完成回调
	if tq.onComplete != nil {
		tq.onComplete(taskID, "text2image", p.Prompt, strings.Join(urls, ","))
	}

	return nil
}

func (tq *TaskQueue) execImageToImage(taskID int64, paramsJSON string) error {
	var p struct {
		Prompt         string  `json:"prompt"`
		Size           string  `json:"size"`
		NegativePrompt string  `json:"negative_prompt"`
		ImageValue     string  `json:"image_value"`
		Strength       float64 `json:"strength"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	urls, err := tq.client.ImageToImage(p.ImageValue, p.Prompt, p.Size, 1, p.Strength, p.NegativePrompt)
	if err != nil {
		return err
	}

	resultJSON, _ := json.Marshal(urls)
	tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusCompleted), 100, string(resultJSON), "")
	tq.notifySubscribers(taskID, model.TaskEvent{
		Event:  "complete",
		Result: string(resultJSON),
	})

	if tq.onComplete != nil {
		tq.onComplete(taskID, "image2image", p.Prompt, strings.Join(urls, ","))
	}

	return nil
}

func (tq *TaskQueue) execBatch(taskID int64, paramsJSON string) error {
	var p struct {
		Prompts []string `json:"prompts"`
		Size    string   `json:"size"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	var allResults []string
	total := len(p.Prompts)

	for i, prompt := range p.Prompts {
		select {
		case <-tq.ctx.Done():
			return fmt.Errorf("任务已取消")
		default:
		}

		urls, err := tq.client.TextToImage(prompt, p.Size, 1, "")
		if err != nil {
			return fmt.Errorf("第 %d 个提示词失败: %w", i+1, err)
		}
		allResults = append(allResults, urls...)

		// 更新进度
		progress := (i + 1) * 100 / total
		tq.repo.UpdateTaskProgress(taskID, progress)
		tq.notifySubscribers(taskID, model.TaskEvent{
			Event:    "progress",
			Status:   string(model.TaskStatusProcessing),
			Progress: progress,
		})
	}

	resultJSON, _ := json.Marshal(allResults)
	tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusCompleted), 100, string(resultJSON), "")
	tq.notifySubscribers(taskID, model.TaskEvent{
		Event:  "complete",
		Result: string(resultJSON),
	})

	if tq.onComplete != nil {
		tq.onComplete(taskID, "batch", strings.Join(p.Prompts, "; "), strings.Join(allResults, ","))
	}

	return nil
}

// ==================== 视频任务执行 ====================

type videoTaskExtra struct {
	TaskID    string   `json:"taskId"`
	Mode      string   `json:"mode,omitempty"`
	ImageURLs []string `json:"image_urls,omitempty"`
}

func (tq *TaskQueue) execTextToVideo(taskID int64, paramsJSON string) error {
	var p struct {
		Prompt            string `json:"prompt"`
		Duration          int    `json:"duration"`
		AspectRatio       string `json:"aspect_ratio"`
		FrameRate         int    `json:"frame_rate"`
		NegativePrompt    string `json:"negative_prompt"`
		Seed              *int   `json:"seed,omitempty"`
		NumInferenceSteps *int   `json:"num_inference_steps,omitempty"`
		Width             *int   `json:"width,omitempty"`
		Height            *int   `json:"height,omitempty"`
		NumFrames         *int   `json:"num_frames,omitempty"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	opts := VideoOptions{
		Duration:          p.Duration,
		AspectRatio:       p.AspectRatio,
		FrameRate:         p.FrameRate,
		NegativePrompt:    p.NegativePrompt,
		Seed:              p.Seed,
		NumInferenceSteps: p.NumInferenceSteps,
		Width:             p.Width,
		Height:            p.Height,
		NumFrames:         p.NumFrames,
		RecordType:        "text2video",
	}
	if opts.Duration <= 0 {
		opts.Duration = 5
	}
	if opts.AspectRatio == "" {
		opts.AspectRatio = "16:9"
	}
	if opts.FrameRate <= 0 {
		opts.FrameRate = 24
	}

	payload := tq.client.BuildVideoPayload(p.Prompt, opts)
	videoID, err := tq.client.SubmitVideoTask(payload)
	if err != nil {
		return fmt.Errorf("提交视频任务失败: %w", err)
	}

	log.Printf("[TaskQueue] 视频任务已提交: task=%d videoID=%s", taskID, videoID)
	return tq.pollVideoTask(taskID, videoID, p.Prompt, opts)
}

func (tq *TaskQueue) execImageToVideo(taskID int64, paramsJSON string) error {
	var p struct {
		Prompt            string   `json:"prompt"`
		Duration          int      `json:"duration"`
		AspectRatio       string   `json:"aspect_ratio"`
		FrameRate         int      `json:"frame_rate"`
		NegativePrompt    string   `json:"negative_prompt"`
		Seed              *int     `json:"seed,omitempty"`
		NumInferenceSteps *int     `json:"num_inference_steps,omitempty"`
		Width             *int     `json:"width,omitempty"`
		Height            *int     `json:"height,omitempty"`
		NumFrames         *int     `json:"num_frames,omitempty"`
		ImageValue        string   `json:"image_value"`
		ImageURLs         []string `json:"image_urls,omitempty"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	opts := VideoOptions{
		Duration:          p.Duration,
		AspectRatio:       p.AspectRatio,
		FrameRate:         p.FrameRate,
		NegativePrompt:    p.NegativePrompt,
		Seed:              p.Seed,
		NumInferenceSteps: p.NumInferenceSteps,
		Width:             p.Width,
		Height:            p.Height,
		NumFrames:         p.NumFrames,
		RecordType:        "image2video",
	}
	if opts.Duration <= 0 {
		opts.Duration = 5
	}
	if opts.AspectRatio == "" {
		opts.AspectRatio = "16:9"
	}
	if opts.FrameRate <= 0 {
		opts.FrameRate = 24
	}

	payload := tq.client.BuildVideoPayload(p.Prompt, opts)
	if p.ImageValue != "" {
		payload["image"] = p.ImageValue
	} else if len(p.ImageURLs) > 0 {
		payload["image"] = p.ImageURLs[0]
	}

	videoID, err := tq.client.SubmitVideoTask(payload)
	if err != nil {
		return fmt.Errorf("提交视频任务失败: %w", err)
	}

	return tq.pollVideoTask(taskID, videoID, p.Prompt, opts)
}

func (tq *TaskQueue) execMultiImageVideo(taskID int64, paramsJSON string) error {
	var p struct {
		Prompt            string   `json:"prompt"`
		Duration          int      `json:"duration"`
		AspectRatio       string   `json:"aspect_ratio"`
		FrameRate         int      `json:"frame_rate"`
		NegativePrompt    string   `json:"negative_prompt"`
		Seed              *int     `json:"seed,omitempty"`
		NumInferenceSteps *int     `json:"num_inference_steps,omitempty"`
		Width             *int     `json:"width,omitempty"`
		Height            *int     `json:"height,omitempty"`
		NumFrames         *int     `json:"num_frames,omitempty"`
		ImageURLs         []string `json:"image_urls"`
		Mode              string   `json:"mode"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	if len(p.ImageURLs) == 0 {
		return fmt.Errorf("至少需要一张图片 URL")
	}

	opts := VideoOptions{
		Duration:          p.Duration,
		AspectRatio:       p.AspectRatio,
		FrameRate:         p.FrameRate,
		NegativePrompt:    p.NegativePrompt,
		Seed:              p.Seed,
		NumInferenceSteps: p.NumInferenceSteps,
		Width:             p.Width,
		Height:            p.Height,
		NumFrames:         p.NumFrames,
		RecordType:        "multi_image_video",
		ImageURLs:         p.ImageURLs,
		Mode:              p.Mode,
	}
	if opts.Duration <= 0 {
		opts.Duration = 5
	}
	if opts.AspectRatio == "" {
		opts.AspectRatio = "16:9"
	}
	if opts.FrameRate <= 0 {
		opts.FrameRate = 24
	}

	payload := tq.client.BuildVideoPayload(p.Prompt, opts)
	extraBody := map[string]any{
		"image": p.ImageURLs,
	}
	if p.Mode == "keyframes" {
		extraBody["mode"] = "keyframes"
	}
	payload["extra_body"] = extraBody

	videoID, err := tq.client.SubmitVideoTask(payload)
	if err != nil {
		return fmt.Errorf("提交视频任务失败: %w", err)
	}

	return tq.pollVideoTask(taskID, videoID, p.Prompt, opts)
}

// pollVideoTask 轮询视频任务状态（复用现有逻辑）
func (tq *TaskQueue) pollVideoTask(taskID int64, videoID, prompt string, opts VideoOptions) error {
	startTime := time.Now()
	retryCount := 0

	for {
		select {
		case <-tq.ctx.Done():
			return fmt.Errorf("任务已取消")
		default:
		}

		elapsed := time.Since(startTime)
		if elapsed > maxPollTime {
			return fmt.Errorf("视频生成超时（超过 %d 分钟）", int(maxPollTime.Minutes()))
		}

		status, err := tq.client.CheckVideoStatus(videoID)
		if err != nil {
			retryCount++
			if retryCount <= pollRetryMax {
				backoffSecs := 1 << uint(retryCount)
				if backoffSecs > 30 {
					backoffSecs = 30
				}
				time.Sleep(time.Duration(backoffSecs) * time.Second)
				continue
			}
			return fmt.Errorf("查询视频状态失败，已重试 %d 次: %w", pollRetryMax, err)
		}

		retryCount = 0

		tq.repo.UpdateTaskProgress(taskID, status.Progress)
		tq.notifySubscribers(taskID, model.TaskEvent{
			Event:    "progress",
			Status:   status.Status,
			Progress: status.Progress,
		})

		switch status.Status {
		case "completed":
			log.Printf("[TaskQueue] 视频生成完成: task=%d url=%s", taskID, status.URL)
			resultJSON, _ := json.Marshal([]string{status.URL})
			tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusCompleted), 100, string(resultJSON), "")
			tq.notifySubscribers(taskID, model.TaskEvent{
				Event:  "complete",
				Result: string(resultJSON),
			})

			if tq.onComplete != nil {
				tq.onComplete(taskID, opts.RecordType, prompt, status.URL)
			}
			return nil

		case "failed":
			errMsg := extractError(status.Error)
			return fmt.Errorf("视频生成失败: %s", errMsg)
		}

		// 动态调整轮询间隔
		sleepTime := pollInterval
		if status.Progress == 0 && elapsed > 2*time.Minute {
			sleepTime = pollInterval * 2
		}
		time.Sleep(sleepTime)
	}
}

// ==================== 重启恢复 ====================

// recoverPending 启动时恢复未完成任务
func (tq *TaskQueue) recoverPending() {
	pending, err := tq.repo.FindPendingTasks()
	if err != nil {
		log.Printf("[TaskQueue] 查询待恢复任务失败: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}

	log.Printf("[TaskQueue] 发现 %d 个待恢复任务，开始恢复...", len(pending))
	for _, task := range pending {
		select {
		case tq.workerSem <- struct{}{}:
			go func(t *model.TaskRecord) {
				log.Printf("[TaskQueue] 恢复任务: id=%d type=%s", t.ID, t.Type)
				// 对于视频任务，检查 Agnes 端状态
				if isVideoTask(t.Type) {
					tq.recoverVideoTask(t)
				} else {
					// 图片任务重新执行
					tq.repo.UpdateTaskStatus(t.ID, string(model.TaskStatusPending), 0, "", "")
					tq.worker(t.ID, t.Type, t.Params)
				}
				<-tq.workerSem
			}(task)
		default:
			log.Printf("[TaskQueue] 并发已满，任务 %d 延后恢复", task.ID)
			go func(t *model.TaskRecord) {
				tq.workerSem <- struct{}{}
				if isVideoTask(t.Type) {
					tq.recoverVideoTask(t)
				} else {
					tq.repo.UpdateTaskStatus(t.ID, string(model.TaskStatusPending), 0, "", "")
					tq.worker(t.ID, t.Type, t.Params)
				}
				<-tq.workerSem
			}(task)
		}
	}
}

func isVideoTask(taskType string) bool {
	return taskType == string(model.TaskTypeTextToVideo) ||
		taskType == string(model.TaskTypeImageToVideo) ||
		taskType == string(model.TaskTypeMultiImageVideo)
}

// recoverVideoTask 恢复视频任务：检查 Agnes 端状态
func (tq *TaskQueue) recoverVideoTask(task *model.TaskRecord) {
	// 从 params 中解析 videoID
	var extra struct {
		TaskID string `json:"taskId"`
	}
	if err := json.Unmarshal([]byte(task.Params), &extra); err != nil || extra.TaskID == "" {
		// 无法获取 videoID，重新提交
		log.Printf("[TaskQueue] 任务 %d 无 videoID，重新执行", task.ID)
		tq.worker(task.ID, task.Type, task.Params)
		return
	}

	status, err := tq.client.CheckVideoStatus(extra.TaskID)
	if err != nil {
		log.Printf("[TaskQueue] 查询视频状态失败: task=%d err=%v，重新执行", task.ID, err)
		tq.worker(task.ID, task.Type, task.Params)
		return
	}

	switch status.Status {
	case "completed":
		log.Printf("[TaskQueue] 恢复: 任务 %d 已完成", task.ID)
		resultJSON, _ := json.Marshal([]string{status.URL})
		tq.repo.UpdateTaskStatus(task.ID, string(model.TaskStatusCompleted), 100, string(resultJSON), "")
		if tq.onComplete != nil {
			// 从 params 中获取 prompt
			var p struct{ Prompt string `json:"prompt"` }
			json.Unmarshal([]byte(task.Params), &p)
			tq.onComplete(task.ID, task.Type, p.Prompt, status.URL)
		}
	case "failed":
		errMsg := extractError(status.Error)
		log.Printf("[TaskQueue] 恢复: 任务 %d 已失败: %s", task.ID, errMsg)
		tq.repo.UpdateTaskStatus(task.ID, string(model.TaskStatusFailed), 0, "", errMsg)
	default:
		log.Printf("[TaskQueue] 恢复: 任务 %d 仍在处理中，重启轮询", task.ID)
		// 从 params 中解析 opts
		var p struct {
			Prompt         string `json:"prompt"`
			Duration       int    `json:"duration"`
			AspectRatio    string `json:"aspect_ratio"`
			FrameRate      int    `json:"frame_rate"`
			NegativePrompt string `json:"negative_prompt"`
		}
		json.Unmarshal([]byte(task.Params), &p)
		opts := VideoOptions{
			Duration:       p.Duration,
			AspectRatio:    p.AspectRatio,
			FrameRate:      p.FrameRate,
			NegativePrompt: p.NegativePrompt,
			RecordType:     task.Type,
		}
		if opts.Duration <= 0 {
			opts.Duration = 5
		}
		if opts.AspectRatio == "" {
			opts.AspectRatio = "16:9"
		}
		if opts.FrameRate <= 0 {
			opts.FrameRate = 24
		}
		tq.repo.UpdateTaskStatus(task.ID, string(model.TaskStatusProcessing), 0, "", "")
		tq.pollVideoTask(task.ID, extra.TaskID, p.Prompt, opts)
	}
}

// CreateTask 创建新的任务记录并设置 ID
func (tq *TaskQueue) CreateTask(record *model.TaskRecord) error {
	id, err := tq.repo.CreateTask(record.Type, record.Params)
	if err != nil {
		return err
	}
	record.ID = id
	return nil
}

// EnqueuePolling 注册后台轮询任务，视频完成后执行 onComplete 回调
func (tq *TaskQueue) EnqueuePolling(taskRecordID int64, videoID string, onComplete func(resultURL string)) error {
	go func() {
		opts := VideoOptions{
			Duration:    5,
			AspectRatio: "16:9",
			FrameRate:   24,
			RecordType:  "shot_video",
		}
		// 复用 pollVideoTask 的轮询逻辑（状态更新、订阅者通知等）
		if err := tq.pollVideoTask(taskRecordID, videoID, "", opts); err != nil {
			log.Printf("[TaskQueue] 轮询视频任务失败: task=%d err=%v", taskRecordID, err)
			return
		}
		// 轮询成功后获取结果 URL 并执行回调
		rec, err := tq.repo.GetTask(taskRecordID)
		if err != nil || rec == nil || rec.Result == "" {
			log.Printf("[TaskQueue] 获取任务结果失败: task=%d", taskRecordID)
			return
		}
		var urls []string
		if err := json.Unmarshal([]byte(rec.Result), &urls); err != nil || len(urls) == 0 {
			log.Printf("[TaskQueue] 解析结果 URL 失败: task=%d", taskRecordID)
			return
		}
		onComplete(urls[0])
	}()
	return nil
}

// ==================== 清理 ====================

// cleanupLoop 定期清理过期任务
func (tq *TaskQueue) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-tq.ctx.Done():
			return
		case <-ticker.C:
			n, err := tq.repo.CleanupOlderThan(24)
			if err != nil {
				log.Printf("[TaskQueue] 清理过期任务失败: %v", err)
			} else if n > 0 {
				log.Printf("[TaskQueue] 已清理 %d 个过期任务", n)
			}
		}
	}
}

// Stop 停止任务队列
func (tq *TaskQueue) Stop() {
	tq.cancel()
}

// extractError 从 Agnes API 错误响应中提取错误消息
func extractError(e any) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case string:
		return v
	case map[string]any:
		if msg, ok := v["message"]; ok {
			return fmt.Sprintf("%v", msg)
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
