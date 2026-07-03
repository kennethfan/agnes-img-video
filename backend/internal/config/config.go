package config

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/agnes-image-tool/backend/internal/model"
)

const (
	DefaultBaseURL    = "https://apihub.agnes-ai.com/v1"
	DefaultModel      = "agnes-image-2.1-flash"
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
func LoadConfig(configPath, apiKeyEnv string) (*model.Config, error) {
	cfg := &model.Config{
		BaseURL: DefaultBaseURL,
		Model:   DefaultModel,
	}

	data, err := os.ReadFile(configPath)
	if err == nil {
		var fileCfg model.Config
		if err := json.Unmarshal(data, &fileCfg); err == nil {
			if fileCfg.BaseURL != "" {
				cfg.BaseURL = fileCfg.BaseURL
			}
			if fileCfg.Model != "" {
				cfg.Model = fileCfg.Model
			}
			if fileCfg.APIKey != "" {
				cfg.APIKey = fileCfg.APIKey
			}
		}
	}

	// 环境变量覆盖
	if envKey := os.Getenv(apiKeyEnv); envKey != "" {
		cfg.APIKey = envKey
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
	if v := os.Getenv("IMAGE_MODEL"); v != "" {
		cfg.ImageModel = v
	} else if cfg.ImageModel == "" {
		cfg.ImageModel = DefaultImageModel
	}
	if v := os.Getenv("VIDEO_MODEL"); v != "" {
		cfg.VideoModel = v
	} else if cfg.VideoModel == "" {
		cfg.VideoModel = DefaultVideoModel
	}
	if v := os.Getenv("CHAT_MODEL"); v != "" {
		cfg.ChatModel = v
	} else if cfg.ChatModel == "" {
		cfg.ChatModel = DefaultChatModel
	}

	mu.Lock()
	cached = cfg
	mu.Unlock()

	return cfg, nil
}

// GetConfig 返回当前配置缓存
func GetConfig() *model.Config {
	mu.RLock()
	defer mu.RUnlock()
	if cached == nil {
		return &model.Config{
			BaseURL: DefaultBaseURL,
			Model:   DefaultModel,
		}
	}
	cp := *cached
	return &cp
}

// UpdateConfig 更新配置文件并刷新缓存
func UpdateConfig(configPath string, cfg *model.Config) error {
	mu.Lock()
	defer mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	cached = cfg
	return nil
}

// SaveConfig 保存配置（兼容外部调用）
func SaveConfig(configPath string, cfg *model.Config) error {
	return UpdateConfig(configPath, cfg)
}
