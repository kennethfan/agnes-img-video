package main

import (
	"database/sql"
	"fmt"
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

	// 初始化访问日志仓库（复用 history.db）
	accessLogRepo, err := repository.NewAccessLogRepo(histRepo.DB())
	if err != nil {
		log.Fatalf("初始化访问日志仓库失败: %v", err)
	}
	middleware.SetAccessLogRepo(accessLogRepo)
	accessLogRepo.StartDailyCleanup(7) // 保留 7 天

	if gs := svc.GetGithubStorage(); gs != nil {
		handler.SetGithubStorage(gs)
	}

	// 从 history.json 导入旧数据（如果存在）
	if n, err := histRepo.ImportFromJSON("history.json"); err != nil {
		log.Printf("[Migration] 导入 history.json 失败: %v", err)
	} else if n > 0 {
		log.Printf("[Migration] 成功从 history.json 导入 %d 条记录", n)
	}

	// 创建任务仓库（复用 history.db）
	taskRepo := repository.NewTaskRepository(histRepo.DB())
	if err := taskRepo.InitTable(); err != nil {
		log.Fatalf("初始化任务表失败: %v", err)
	}

	// 创建统一任务队列（替换视频任务管理器）
	taskQueue := service.NewTaskQueue(taskRepo, svc, 10)

	// 创建 handler
	imageHandler := handler.NewImageHandler(svc, taskQueue)
	videoHandler := handler.NewVideoHandler(svc, taskQueue)
	historyHandler := handler.NewHistoryHandler(histRepo)
	taskHandler := handler.NewTaskHandler(taskQueue)
	configHandler := handler.NewConfigHandler(configPath)
	ideasHandler := handler.NewIdeasHandler(svc)
	comicHandler := handler.NewComicHandler(svc)
	accessLogHandler := handler.NewAccessLogHandler(accessLogRepo)
	assetHandler := handler.NewAssetHandler(histRepo)

	// 故事板
	storyboardRepo, err := repository.NewStoryboardRepo(dbPath)
	if err != nil {
		log.Fatalf("初始化故事板数据库失败: %v", err)
	}
	defer storyboardRepo.Close()
	storyboardHandler := handler.NewStoryboardHandler(storyboardRepo)

	// 数据库导出与恢复
	dbReplaceFunc := func(tmpPath string) error {
		// 关闭旧连接
		histRepo.Close()
		storyboardRepo.Close()

		// 备份当前数据库
		bakPath := dbPath + ".bak"
		if err := os.Rename(dbPath, bakPath); err != nil {
			return fmt.Errorf("备份数据库失败: %w", err)
		}

		// 替换为新文件
		if err := os.Rename(tmpPath, dbPath); err != nil {
			os.Rename(bakPath, dbPath) // 恢复备份
			log.Printf("[DB] 替换数据库文件失败，请重启服务器: %v", err)
			return fmt.Errorf("替换数据库文件失败: %w", err)
		}

		// 重新打开数据库
		newHistRepo, err := repository.NewHistoryRepo(dbPath)
		if err != nil {
			os.Rename(bakPath, dbPath) // 恢复备份
			return fmt.Errorf("重新打开数据库失败: %w", err)
		}

		// 更新访问日志仓库的 db 引用
		accessLogRepo.SetDB(newHistRepo.DB())

		newStoryRepo, err := repository.NewStoryboardRepo(dbPath)
		if err != nil {
			newHistRepo.Close()
			os.Rename(bakPath, dbPath) // 恢复备份
			return fmt.Errorf("重新打开故事板数据库失败: %w", err)
		}

		// 更新所有引用
		histRepo = newHistRepo
		storyboardRepo = newStoryRepo
		handler.SetHistoryRepo(newHistRepo)
		historyHandler.SetRepo(newHistRepo)
		assetHandler.SetRepo(newHistRepo)
		middleware.SetAccessLogRepo(accessLogRepo)
		storyboardHandler.SetRepo(newStoryRepo)

		// 删除备份（恢复成功）
		os.Remove(bakPath)
		log.Printf("[DB] 数据库恢复完成，连接已刷新")
		return nil
	}



	dbHandler := handler.NewDBHandler(dbPath, dbReplaceFunc, func() *sql.DB { return histRepo.DB() })

	// 设置任务完成回调（自动保存历史记录）
	handler.SetupVideoHistoryCallback(taskQueue, svc)

	// 设置路由
	r := gin.Default()
	r.Use(middleware.SetupCORS())
	r.Use(middleware.AccessLogger())

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

		// 访问日志
		api.GET("/access-logs", accessLogHandler.ListLogs)
		api.DELETE("/access-logs", accessLogHandler.ClearLogs)
		api.DELETE("/access-logs/:id", accessLogHandler.DeleteLog)

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

		// 数据库导出与恢复
		api.GET("/db/export", dbHandler.ExportDB)
		api.POST("/db/restore", dbHandler.RestoreDB)

		// 手动上传到 GitHub
		api.POST("/upload-to-github", handler.UploadToGitHub)

		// 代理下载（解决跨域下载问题）
		api.GET("/download", handler.ProxyDownload)

		// 统一任务查询与进度推送
	api.GET("/tasks/:id", taskHandler.GetTask)
	api.GET("/tasks/:id/stream", taskHandler.StreamSSE)
	api.POST("/tasks/:id/cancel", taskHandler.CancelTask)
	api.POST("/tasks/:id/retry", taskHandler.RetryTask)
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


