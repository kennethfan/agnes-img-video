package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/agnes-image-tool/backend/internal/model"
)

const (
	DefaultBaseURL    = "https://apihub.agnes-ai.com/v1"
	DefaultImageModel = "agnes-image-2.1-flash"
	DefaultVideoModel = "agnes-video-v2.0"
	DefaultChatModel  = "agnes-2.0-flash"
)

var (
	mu     sync.RWMutex
	cached *model.Config
)

func resolvePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func loadKeyFromFile(path string) (string, error) {
	resolved := resolvePath(path)
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", nil
	}
	return key, nil
}

// LoadConfig 从环境变量加载配置，API Key 通过 API_KEY_PATH 指向的文件读取
func LoadConfig() (*model.Config, error) {
	cfg := &model.Config{
		BaseURL:    DefaultBaseURL,
		ImageModel: DefaultImageModel,
		VideoModel: DefaultVideoModel,
		ChatModel:  DefaultChatModel,
	}

	if envPath := os.Getenv("API_KEY_PATH"); envPath != "" {
		cfg.ApiKeyPath = envPath
		log.Printf("[config] 从环境变量 API_KEY_PATH 读取 Key 路径: %s", envPath)
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}
	if envImageModel := os.Getenv("IMAGE_MODEL"); envImageModel != "" {
		cfg.ImageModel = envImageModel
	}
	if envVideoModel := os.Getenv("VIDEO_MODEL"); envVideoModel != "" {
		cfg.VideoModel = envVideoModel
	}
	if envChatModel := os.Getenv("CHAT_MODEL"); envChatModel != "" {
		cfg.ChatModel = envChatModel
	}
	if envToken := os.Getenv("GITHUB_TOKEN"); envToken != "" {
		cfg.GithubToken = envToken
	}
	if envRepo := os.Getenv("GITHUB_REPO"); envRepo != "" {
		cfg.GithubRepo = envRepo
	}
	if envBranch := os.Getenv("GITHUB_BRANCH"); envBranch != "" {
		cfg.GithubBranch = envBranch
	}
	if envDriver := os.Getenv("DB_DRIVER"); envDriver != "" {
		cfg.DBDriver = envDriver
	}
	if envDSN := os.Getenv("DB_DSN"); envDSN != "" {
		cfg.DBDSN = envDSN
	}

	if cfg.ApiKeyPath != "" {
		if key, err := loadKeyFromFile(cfg.ApiKeyPath); err != nil {
			log.Printf("警告: 从 %s 读取 API Key 失败: %v", cfg.ApiKeyPath, err)
		} else if key != "" {
			cfg.APIKey = key
			log.Printf("[config] 从 %s 读取 API Key", resolvePath(cfg.ApiKeyPath))
		}
	}

	mu.Lock()
	cached = cfg
	mu.Unlock()

	return cfg, nil
}
