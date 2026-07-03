package main

import (
	"log"
	"os"
	"path/filepath"

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

	// 确定项目根目录（backend/ 的父目录）
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("无法确定项目根目录: %v", err)
	}

	configPath := filepath.Join(projectRoot, ".config.json")
	dbPath := filepath.Join(projectRoot, "history.db")
	outputsPath := filepath.Join(projectRoot, "outputs")

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
	svc := service.NewAgnesClient(cfg.APIKey, cfg.BaseURL)

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

	// 从 history.json 导入旧数据（如果存在）
	oldJSONPath := filepath.Join(projectRoot, "history.json")
	if n, err := histRepo.ImportFromJSON(oldJSONPath); err != nil {
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

	// 设置视频完成回调（自动保存历史记录）
	handler.SetupVideoHistoryCallback(taskMgr, svc)

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

func findProjectRoot() (string, error) {
	// 尝试从当前工作目录找到项目根（包含 outputs/ 目录）
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 如果当前在 backend/ 目录，返回父目录
	if filepath.Base(wd) == "backend" {
		return filepath.Dir(wd), nil
	}

	// 如果 outputs/ 目录存在，则当前就是根目录
	if _, err := os.Stat(filepath.Join(wd, "outputs")); err == nil {
		return wd, nil
	}

	// 否则使用 CWD
	return wd, nil
}
