package service

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
)

// GenerateResult 批量生成结果
type GenerateResult struct {
	Submitted int `json:"submitted"`
	Total     int `json:"total"`
	Failed    int `json:"failed"`
}

// StoryboardGenerator 分镜镜头视频生成器
type StoryboardGenerator struct {
	client    *AgnesClient
	taskQueue *TaskQueue
	repo      repository.StoryboardRepository
	sem       chan struct{} // 限制并发生成数
}

// NewStoryboardGenerator 创建 StoryboardGenerator
func NewStoryboardGenerator(client *AgnesClient, taskQueue *TaskQueue, repo repository.StoryboardRepository) *StoryboardGenerator {
	return &StoryboardGenerator{
		client:    client,
		taskQueue: taskQueue,
		repo:      repo,
		sem:       make(chan struct{}, 3), // 最多 3 个并发
	}
}

// GenerateAll 批量提交项目下所有 pending shots 到视频生成管线
func (g *StoryboardGenerator) GenerateAll(ctx context.Context, projectID int64) (*GenerateResult, error) {
	project, err := g.repo.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("获取项目失败: %w", err)
	}
	_ = project

	shots, err := g.repo.ListShots(projectID)
	if err != nil {
		return nil, fmt.Errorf("获取镜头列表失败: %w", err)
	}

	result := &GenerateResult{Total: len(shots)}

	var wg sync.WaitGroup
	errCh := make(chan error, len(shots))

	for i := range shots {
		if shots[i].Status != "pending" {
			continue
		}

		wg.Add(1)
		go func(s model.StoryboardShot) {
			defer wg.Done()

			g.sem <- struct{}{}
			defer func() { <-g.sem }()

			if err := g.GenerateOne(ctx, s.ID); err != nil {
				log.Printf("[StoryboardGenerator] 镜头 %d 生成失败: %v", s.ID, err)
				errCh <- err
				return
			}
			result.Submitted++
		}(shots[i])
	}

	wg.Wait()
	close(errCh)

	for range errCh {
		result.Failed++
	}

	return result, nil
}

// GenerateOne 提交单个 shot 到视频生成管线
func (g *StoryboardGenerator) GenerateOne(ctx context.Context, shotID int64) error {
	shot, err := g.repo.GetShot(shotID)
	if err != nil {
		return fmt.Errorf("获取镜头失败: %w", err)
	}

	if shot.Status != "pending" {
		return fmt.Errorf("镜头 %d 状态不是 pending（当前: %s）", shotID, shot.Status)
	}

	opts := VideoOptions{
		Duration:    5,
		AspectRatio: "16:9",
		FrameRate:   24,
		RecordType:  "text2video",
	}

	payload := g.client.BuildVideoPayload(shot.Prompt, opts)

	if shot.ReferenceImage != "" {
		opts.ImageURLs = []string{shot.ReferenceImage}
		payload["extra_body"] = map[string]any{
			"image": opts.ImageURLs,
		}
		opts.RecordType = "image2video"
	}

	videoID, err := g.client.SubmitVideoTask(payload)
	if err != nil {
		return fmt.Errorf("提交视频任务失败: %w", err)
	}

	taskRecord := &model.TaskRecord{
		Type:   "shot_video",
		Status: "pending",
		Params: fmt.Sprintf(`{"shot_id":%d,"project_id":%d,"video_id":"%s"}`, shot.ID, shot.ProjectID, videoID),
	}
	if err := g.taskQueue.CreateTask(taskRecord); err != nil {
		return fmt.Errorf("创建任务记录失败: %w", err)
	}

	if err := g.repo.UpdateShotStatus(shotID, "generating", videoID, taskRecord.ID); err != nil {
		return fmt.Errorf("更新镜头状态失败: %w", err)
	}

	// 注册后台轮询任务，完成后下载视频并更新 shot 结果
	go g.pollVideoStatus(taskRecord.ID, videoID, shotID)

	return nil
}

// pollVideoStatus 轮询视频生成状态（在后台 goroutine 中运行）
func (g *StoryboardGenerator) pollVideoStatus(taskRecordID int64, videoID string, shotID int64) {
	err := g.taskQueue.EnqueuePolling(taskRecordID, videoID, func(resultURL string) {
		localPath, err := g.client.DownloadVideo(resultURL, fmt.Sprintf("shot_%d", shotID))
		if err != nil {
			log.Printf("[StoryboardGenerator] 下载视频失败 shot=%d: %v", shotID, err)
			return
		}
		if err := g.repo.UpdateShotResult(shotID, localPath); err != nil {
			log.Printf("[StoryboardGenerator] 更新镜头结果失败 shot=%d: %v", shotID, err)
		}
	})
	if err != nil {
		log.Printf("[StoryboardGenerator] 注册轮询失败 task=%d: %v", taskRecordID, err)
	}
}
