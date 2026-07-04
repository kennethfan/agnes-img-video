package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

// GithubStorage 将文件上传到 GitHub 仓库
type GithubStorage struct {
	token  string
	repo   string // "owner/repo"
	branch string
	client *http.Client
}

func NewGithubStorage(token, repo, branch string) *GithubStorage {
	if branch == "" {
		branch = "main"
	}
	return &GithubStorage{
		token:  token,
		repo:   strings.TrimSuffix(repo, ".git"),
		branch: branch,
		client: &http.Client{},
	}
}

// UploadFile 通过 GitHub Contents API 上传文件
// remotePath: 仓库中的路径，如 "images/20260703_foo.png"
// 返回 download_url
func (g *GithubStorage) UploadFile(localPath, remotePath string) (string, error) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("读取本地文件失败: %w", err)
	}

	content := base64.StdEncoding.EncodeToString(data)

	// 检测 MIME 类型
	var mimeType string
	switch {
	case strings.HasSuffix(remotePath, ".png"):
		mimeType = "image/png"
	case strings.HasSuffix(remotePath, ".jpg"), strings.HasSuffix(remotePath, ".jpeg"):
		mimeType = "image/jpeg"
	case strings.HasSuffix(remotePath, ".gif"):
		mimeType = "image/gif"
	case strings.HasSuffix(remotePath, ".webp"):
		mimeType = "image/webp"
	case strings.HasSuffix(remotePath, ".mp4"):
		mimeType = "video/mp4"
	default:
		mimeType = "application/octet-stream"
	}

	payload := map[string]any{
		"message":     fmt.Sprintf("upload %s", remotePath),
		"content":     content,
		"branch":      g.branch,
		"content-type": mimeType,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", g.repo, remotePath)
	req, err := http.NewRequest("PUT", url, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GitHub API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应获取 download_url
	var result struct {
		Content struct {
			DownloadURL string `json:"download_url"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.Content.DownloadURL != "" {
		return result.Content.DownloadURL, nil
	}

	return url, nil
}

func (g *GithubStorage) DeleteFile(remotePath string) error {
	infoURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", g.repo, remotePath)
	req, err := http.NewRequest("GET", infoURL, nil)
	if err != nil {
		return fmt.Errorf("创建查询请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("查询文件信息失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("查询文件返回 %d: %s", resp.StatusCode, string(body))
	}

	var info struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return fmt.Errorf("解析文件信息失败: %w", err)
	}

	payload := map[string]any{
		"message": fmt.Sprintf("delete %s", remotePath),
		"branch":  g.branch,
		"sha":     info.SHA,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化删除请求失败: %w", err)
	}

	delURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", g.repo, remotePath)
	delReq, err := http.NewRequest("DELETE", delURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("创建删除请求失败: %w", err)
	}
	delReq.Header.Set("Authorization", "Bearer "+g.token)
	delReq.Header.Set("Content-Type", "application/json")
	delReq.Header.Set("Accept", "application/vnd.github.v3+json")

	delResp, err := g.client.Do(delReq)
	if err != nil {
		return fmt.Errorf("删除请求失败: %w", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode < 200 || delResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(delResp.Body)
		return fmt.Errorf("GitHub API 删除返回 %d: %s", delResp.StatusCode, string(respBody))
	}

	return nil
}

func (g *GithubStorage) DeleteByURL(rawURL string) error {
	prefix := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/", g.repo, g.branch)
	if !strings.HasPrefix(rawURL, prefix) {
		return fmt.Errorf("不是本仓库的 GitHub URL: %s", rawURL)
	}
	remotePath := path.Clean(strings.TrimPrefix(rawURL, prefix))
	return g.DeleteFile(remotePath)
}
