package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/agnes-image-tool/backend/internal/config"
	"github.com/agnes-image-tool/backend/internal/handler"
	"github.com/agnes-image-tool/backend/internal/middleware"
	"github.com/agnes-image-tool/backend/internal/repository"
	"github.com/agnes-image-tool/backend/internal/service"
)

func main() {
	// 加载 .env 文件
	_ = godotenv.Load(".env") // 从 backend/ 目录加载 .env

	apiKey := os.Getenv("AGNES_API_KEY")
	if apiKey == "" {
		log.Fatal("AGNES_API_KEY 环境变量未设置")
	}

	// 所有运行时数据都在 backend/ 目录下
	configPath := ".config.json"
	dbPath := "history.db"
	outputsPath := "outputs"

	// 确保 outputs/ 目录存在
	os.MkdirAll(outputsPath, 0755)

	// 加载配置
	cfg, err := config.LoadConfig(configPath, "AGNES_API_KEY")
	if err != nil {
		log.Printf("警告: 加载配置失败: %v", err)
	}
	if cfg.APIKey == "" {
		cfg.APIKey = apiKey
	}

	log.Printf("配置加载完成: base_url=%s, model=%s", cfg.BaseURL, cfg.Model)
	log.Printf("配置文件: %s", configPath)
	log.Printf("数据库: %s", dbPath)
	log.Printf("输出目录: %s", outputsPath)

	// 创建服务
	svc := service.NewAgnesClient(cfg.APIKey, cfg.BaseURL, cfg.ImageModel, cfg.VideoModel, cfg.ChatModel)

	// GitHub 文件存储（如果配置了）
	if cfg.GithubToken != "" && cfg.GithubRepo != "" {
		gs := service.NewGithubStorage(cfg.GithubToken, cfg.GithubRepo, cfg.GithubBranch)
		svc.SetGithubStorage(gs)
		log.Printf("[GitHub] 文件存储已启用: %s (branch: %s)", cfg.GithubRepo, cfg.GithubBranch)
	}

	// 初始化 SQLite 数据库
	histRepo, err := repository.NewHistoryRepo(dbPath)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer histRepo.Close()
	handler.SetHistoryRepo(histRepo)
	if gs := svc.GetGithubStorage(); gs != nil {
		handler.SetGithubStorage(gs)
	}

	// 从 history.json 导入旧数据（如果存在）
	if n, err := histRepo.ImportFromJSON("history.json"); err != nil {
		log.Printf("[Migration] 导入 history.json 失败: %v", err)
	} else if n > 0 {
		log.Printf("[Migration] 成功从 history.json 导入 %d 条记录", n)
	}

	// 创建视频任务管理器
	taskMgr := service.NewTaskManager(svc)

	// 创建 handler
	imageHandler := handler.NewImageHandler(svc)
	videoHandler := handler.NewVideoHandler(svc, taskMgr)
	historyHandler := handler.NewHistoryHandler(histRepo)
	configHandler := handler.NewConfigHandler(configPath)
	ideasHandler := handler.NewIdeasHandler(svc)
	comicHandler := handler.NewComicHandler(svc)
	assetHandler := handler.NewAssetHandler(histRepo)

	// 故事板
	storyboardRepo, err := repository.NewStoryboardRepo(dbPath)
	if err != nil {
		log.Fatalf("初始化故事板数据库失败: %v", err)
	}
	defer storyboardRepo.Close()
	storyboardHandler := handler.NewStoryboardHandler(storyboardRepo)

	// 设置视频完成回调（自动保存历史记录）
	handler.SetupVideoHistoryCallback(taskMgr, svc)

	// 启动时恢复未完成的视频任务
	go recoverPendingVideoTasks(svc, histRepo)

	// 设置路由
	r := gin.Default()
	r.Use(middleware.SetupCORS())

	api := r.Group("/api/v1")
	{
		// 图片
		api.POST("/images/text-to-image", imageHandler.TextToImage)
		api.POST("/images/image-to-image", imageHandler.ImageToImage)
		api.POST("/images/batch", imageHandler.BatchGenerate)

		// 配置
		api.GET("/config", configHandler.GetConfig)
		api.PUT("/config", configHandler.UpdateConfig)

		// 视频 & 脚本
		api.POST("/videos/text-to-video", videoHandler.TextToVideo)
		api.POST("/videos/image-to-video", videoHandler.ImageToVideo)
		api.POST("/videos/multi-image", videoHandler.MultiImageVideo)
		api.POST("/videos/generate-script", videoHandler.GenerateScript)
		api.GET("/videos/:taskId", videoHandler.GetTaskStatus)
		api.GET("/videos/stream/:taskId", videoHandler.StreamSSE)

		// 历史
		api.GET("/history", historyHandler.GetHistory)
		api.DELETE("/history", historyHandler.ClearHistory)
		api.DELETE("/history/:id", historyHandler.DeleteRecord)
		api.POST("/history/delete", historyHandler.DeleteHistory)

		// 点子库
		api.POST("/ideas/expand", ideasHandler.ExpandIdea)

		// 漫画
		api.POST("/comic/generate-prompts", comicHandler.GeneratePrompts)

		// 资产管理
		api.GET("/assets", assetHandler.ListAssets)
		api.POST("/assets/favorite", assetHandler.ToggleFavorite)
		api.POST("/assets/batch-download", assetHandler.BatchDownload)
		api.DELETE("/assets", assetHandler.DeleteAssets)

		// 故事板
		storyboard := api.Group("/storyboard")
		{
			storyboard.GET("/projects", storyboardHandler.ListProjects)
			storyboard.POST("/projects", storyboardHandler.CreateProject)
			storyboard.GET("/projects/:id", storyboardHandler.GetProject)
			storyboard.PUT("/projects/:id", storyboardHandler.UpdateProject)
			storyboard.DELETE("/projects/:id", storyboardHandler.DeleteProject)
			storyboard.POST("/projects/:id/duplicate", storyboardHandler.DuplicateProject)
			storyboard.POST("/projects/:id/shots", storyboardHandler.CreateShot)
			storyboard.PUT("/projects/:id/shots/reorder", storyboardHandler.ReorderShots)
			storyboard.PUT("/shots/:id", storyboardHandler.UpdateShot)
			storyboard.DELETE("/shots/:id", storyboardHandler.DeleteShot)
			storyboard.POST("/projects/:id/generate", storyboardHandler.GenerateShots)
		}
	}

	// 静态文件服务 - outputs/ 目录
	r.Static("/outputs", outputsPath)

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("启动服务器在 :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// recoverPendingVideoTasks 启动时检查未完成的视频任务，更新历史记录
func recoverPendingVideoTasks(svc *service.AgnesClient, repo *repository.HistoryRepo) {
	pending, err := repo.FindPendingVideos()
	if err != nil {
		log.Printf("[Recovery] 查询待处理视频任务失败: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	log.Printf("[Recovery] 发现 %d 个待处理视频任务，开始检查状态...", len(pending))
	for _, p := range pending {
		status, err := svc.CheckVideoStatus(p.TaskID)
		if err != nil {
			log.Printf("[Recovery] 查询任务 %s 状态失败: %v", p.TaskID, err)
			continue
		}
		switch status.Status {
		case "completed":
			log.Printf("[Recovery] 任务 %s 已完成，更新历史记录", p.TaskID)
			paths := []string{status.URL}
			localPath, err := svc.DownloadVideo(status.URL, "video_recover_"+p.Mode)
			if err != nil {
				log.Printf("[Recovery] 下载视频 %s 失败: %v", p.TaskID, err)
			} else {
				paths = []string{localPath}
			}
			repo.UpdateRecordImages(p.ID, paths)
		case "failed":
			log.Printf("[Recovery] 任务 %s 已失败，跳过", p.TaskID)
		default:
			log.Printf("[Recovery] 任务 %s 仍在处理中（%s），跳过", p.TaskID, status.Status)
		}
	}
}
