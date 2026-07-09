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
	gormrepo "github.com/agnes-image-tool/backend/internal/repository/gorm"
	"github.com/agnes-image-tool/backend/internal/service"
)

func main() {
	// 加载 .env 文件
	_ = godotenv.Load(".env") // 从 backend/ 目录加载 .env

	// 所有运行时数据都在 backend/ 目录下
	configPath := ".config.json"
	dbPath := "history.db"
	outputsPath := "outputs"

	// 确保 outputs/ 目录存在
	os.MkdirAll(outputsPath, 0755)

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("警告: 加载配置失败: %v", err)
	}
	if cfg.APIKey == "" {
		log.Fatal("AGNES_API_KEY 环境变量未设置")
	}

	log.Printf("配置加载完成: base_url=%s, image_model=%s, video_model=%s, chat_model=%s", cfg.BaseURL, cfg.ImageModel, cfg.VideoModel, cfg.ChatModel)
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

	// 初始化 GORM 数据库（AutoMigrate 自动建表）
	dbDriver := cfg.DBDriver
	dbDSN := cfg.DBDSN
	if dbDriver == "" {
		dbDriver = "sqlite"
	}
	if dbDSN == "" {
		dbDSN = dbPath
	}
	gormDB, err := gormrepo.OpenDB(gormrepo.DBConfig{Driver: dbDriver, DSN: dbDSN})
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("获取底层 sql.DB 失败: %v", err)
	}
	defer sqlDB.Close()

	// 创建 GORM 仓库实例
	histRepo := gormrepo.NewHistoryRepository(gormDB)
	handler.SetHistoryRepo(histRepo)

	accessLogRepo := gormrepo.NewAccessLogRepository(gormDB)
	middleware.SetAccessLogRepo(accessLogRepo)
	accessLogRepo.StartDailyCleanup(7)

	if gs := svc.GetGithubStorage(); gs != nil {
		handler.SetGithubStorage(gs)
	}

	// 创建任务仓库
	taskRepo := gormrepo.NewTaskRepository(gormDB)

	// 创建统一任务队列
	taskQueue := service.NewTaskQueue(taskRepo, svc, 10)

	// 创建 handler
	imageHandler := handler.NewImageHandler(svc, taskQueue)
	videoHandler := handler.NewVideoHandler(svc, taskQueue)
	historyHandler := handler.NewHistoryHandler(histRepo)
	taskHandler := handler.NewTaskHandler(taskQueue)
	ideasHandler := handler.NewIdeasHandler(svc)
	comicHandler := handler.NewComicHandler(svc)
	accessLogHandler := handler.NewAccessLogHandler(accessLogRepo)
	assetHandler := handler.NewAssetHandler(histRepo)

	storyboardRepo := gormrepo.NewStoryboardRepository(gormDB)
	storyboardHandler := handler.NewStoryboardHandler(storyboardRepo)

	settingsRepo := gormrepo.NewSettingsRepository(gormDB)
	settingsHandler := handler.NewSettingsHandler(settingsRepo)

	// 数据库导出与恢复
	dbReplaceFunc := func(tmpPath string) error {
		// 关闭旧连接
		sqlDB.Close()

		// 备份当前数据库
		bakPath := dbPath + ".bak"
		if err := os.Rename(dbPath, bakPath); err != nil {
			return fmt.Errorf("备份数据库失败: %w", err)
		}

		// 替换为新文件
		if err := os.Rename(tmpPath, dbPath); err != nil {
			os.Rename(bakPath, dbPath)
			log.Printf("[DB] 替换数据库文件失败，请重启服务器: %v", err)
			return fmt.Errorf("替换数据库文件失败: %w", err)
		}

		// 重新打开数据库
		newGormDB, err := gormrepo.OpenDB(gormrepo.DBConfig{Driver: "sqlite", DSN: dbPath})
		if err != nil {
			os.Rename(bakPath, dbPath)
			return fmt.Errorf("重新打开数据库失败: %w", err)
		}
		newSQLDB, err := newGormDB.DB()
		if err != nil {
			os.Rename(bakPath, dbPath)
			return fmt.Errorf("获取底层 sql.DB 失败: %w", err)
		}

		// 创建新仓库
		newHistRepo := gormrepo.NewHistoryRepository(newGormDB)
		newStoryRepo := gormrepo.NewStoryboardRepository(newGormDB)
		newAccessLogRepo := gormrepo.NewAccessLogRepository(newGormDB)
		newSettingsRepo := gormrepo.NewSettingsRepository(newGormDB)

		// 更新所有引用
		gormDB = newGormDB
		sqlDB = newSQLDB
		histRepo = newHistRepo
		storyboardRepo = newStoryRepo
		accessLogRepo = newAccessLogRepo
		settingsRepo = newSettingsRepo
		handler.SetHistoryRepo(newHistRepo)
		historyHandler.SetRepo(newHistRepo)
		assetHandler.SetRepo(newHistRepo)
		settingsHandler = handler.NewSettingsHandler(newSettingsRepo)
		middleware.SetAccessLogRepo(newAccessLogRepo)
		storyboardHandler.SetRepo(newStoryRepo)

		os.Remove(bakPath)
		log.Printf("[DB] 数据库恢复完成，连接已刷新")
		return nil
	}

	dbHandler := handler.NewDBHandler(dbPath, dbReplaceFunc, func() *sql.DB { return sqlDB })

	// 设置任务完成回调
	handler.SetupVideoHistoryCallback(taskQueue, svc)

	// 设置路由
	r := gin.Default()
	r.Use(middleware.SetupCORS())
	r.Use(middleware.AccessLogger())

	api := r.Group("/api/v1")
	{
		api.POST("/images/text-to-image", imageHandler.TextToImage)
		api.POST("/images/image-to-image", imageHandler.ImageToImage)
		api.POST("/images/batch", imageHandler.BatchGenerate)

		api.POST("/videos/text-to-video", videoHandler.TextToVideo)
		api.POST("/videos/image-to-video", videoHandler.ImageToVideo)
		api.POST("/videos/multi-image", videoHandler.MultiImageVideo)
		api.POST("/videos/generate-script", videoHandler.GenerateScript)
		api.GET("/videos/:taskId", videoHandler.GetTaskStatus)
		api.GET("/videos/stream/:taskId", videoHandler.StreamSSE)

		api.GET("/history", historyHandler.GetHistory)
		api.DELETE("/history", historyHandler.ClearHistory)
		api.DELETE("/history/:id", historyHandler.DeleteRecord)
		api.POST("/history/delete", historyHandler.DeleteHistory)

		api.POST("/ideas/expand", ideasHandler.ExpandIdea)
		api.POST("/comic/generate-prompts", comicHandler.GeneratePrompts)

		api.GET("/access-logs", accessLogHandler.ListLogs)
		api.DELETE("/access-logs", accessLogHandler.ClearLogs)
		api.DELETE("/access-logs/:id", accessLogHandler.DeleteLog)

		api.GET("/settings", settingsHandler.GetSettings)
		api.PUT("/settings", settingsHandler.UpdateSettings)

		api.GET("/assets", assetHandler.ListAssets)
		api.POST("/assets/favorite", assetHandler.ToggleFavorite)
		api.POST("/assets/batch-download", assetHandler.BatchDownload)
		api.DELETE("/assets", assetHandler.DeleteAssets)

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

		api.GET("/db/export", dbHandler.ExportDB)
		api.POST("/db/restore", dbHandler.RestoreDB)

		api.POST("/upload-to-github", handler.UploadToGitHub)
		api.GET("/download", handler.ProxyDownload)

		api.GET("/tasks", taskHandler.ListTasks)
		api.GET("/tasks/:id", taskHandler.GetTask)
		api.GET("/tasks/:id/stream", taskHandler.StreamSSE)
		api.POST("/tasks/:id/cancel", taskHandler.CancelTask)
		api.POST("/tasks/:id/retry", taskHandler.RetryTask)
	}

	r.Static("/outputs", outputsPath)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("启动服务器在 :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
