package config

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/agnes-image-tool/backend/internal/model"
)

const (
	DefaultBaseURL    = "https://apihub.agnes-ai.com/v1"
	DefaultImageModel = "agnes-image-2.1-flash"
	DefaultVideoModel = "agnes-video-v2.0"
	DefaultChatModel  = "agnes-2.0-flash"
	ConfigFileName    = ".config.json"
)

var (
	mu     sync.RWMutex
	cached *model.Config
)

// LoadConfig 从文件加载配置，环境变量覆盖
func LoadConfig(configPath string) (*model.Config, error) {
	cfg := &model.Config{
		BaseURL:    DefaultBaseURL,
		ImageModel: DefaultImageModel,
		VideoModel: DefaultVideoModel,
		ChatModel:  DefaultChatModel,
	}

	data, err := os.ReadFile(configPath)
	if err == nil {
		var fileCfg model.Config
		if err := json.Unmarshal(data, &fileCfg); err == nil {
			if fileCfg.APIKey != "" {
				cfg.APIKey = fileCfg.APIKey
			}
			if fileCfg.BaseURL != "" {
				cfg.BaseURL = fileCfg.BaseURL
			}
			if fileCfg.ImageModel != "" {
				cfg.ImageModel = fileCfg.ImageModel
			}
			if fileCfg.VideoModel != "" {
				cfg.VideoModel = fileCfg.VideoModel
			}
			if fileCfg.ChatModel != "" {
				cfg.ChatModel = fileCfg.ChatModel
			}
			if fileCfg.GithubToken != "" {
				cfg.GithubToken = fileCfg.GithubToken
			}
			if fileCfg.GithubRepo != "" {
				cfg.GithubRepo = fileCfg.GithubRepo
			}
		if fileCfg.GithubBranch != "" {
			cfg.GithubBranch = fileCfg.GithubBranch
		}
		if fileCfg.DBDriver != "" {
			cfg.DBDriver = fileCfg.DBDriver
		}
		if fileCfg.DBDSN != "" {
			cfg.DBDSN = fileCfg.DBDSN
		}
	}
	}

	// 环境变量覆盖
	if envKey := os.Getenv("AGNES_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
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

	mu.Lock()
	cached = cfg
	mu.Unlock()

	return cfg, nil
}
