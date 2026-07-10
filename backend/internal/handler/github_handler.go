package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// UploadToGitHub 手动上传文件到 GitHub
// POST /api/v1/upload-to-github
// Body: {"url": "https://...", "filename": "optional_name.png"}
func UploadToGitHub(c *gin.Context) {
	if githubStorage == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置 GitHub 存储"})
		return
	}

	var req struct {
		URL      string `json:"url" binding:"required"`
		Filename string `json:"filename"`
		AssetID  int64  `json:"asset_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 下载文件到临时目录
	resp, err := http.Get(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "下载文件失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	tmpDir := "tmp"
	os.MkdirAll(tmpDir, 0755)

	ext := filepath.Ext(req.URL)
	if ext == "" {
		ext = ".png"
	}
	if req.Filename == "" {
		req.Filename = fmt.Sprintf("github_upload_%d%s", time.Now().UnixNano(), ext)
	}

	tmpPath := filepath.Join(tmpDir, req.Filename)
	f, err := os.Create(tmpPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时文件失败: " + err.Error()})
		return
	}
	defer os.Remove(tmpPath)

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入临时文件失败: " + err.Error()})
		return
	}
	f.Close()

	remotePath := fmt.Sprintf("outputs/%s", req.Filename)
	githubURL, err := githubStorage.UploadFile(tmpPath, remotePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传到 GitHub 失败: " + err.Error()})
		return
	}

	// 如果指定了 asset_id，将 github_url 回写到资产记录
	if req.AssetID > 0 && assetRepo != nil {
		if err := assetRepo.UpdateGithubURL(req.AssetID, githubURL); err != nil {
			log.Printf("[GitHub] 回写 github_url 到资产 %d 失败: %v", req.AssetID, err)
			// 不阻塞响应，仅记录日志
		}
	}

	c.JSON(http.StatusOK, gin.H{"github_url": githubURL})
}

// ProxyDownload 代理下载外部资源（解决跨域下载问题）
// GET /api/v1/download?url=...
func ProxyDownload(c *gin.Context) {
	urlStr := c.Query("url")
	if urlStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 url 参数"})
		return
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "下载失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上游返回异常状态码: " + http.StatusText(resp.StatusCode)})
		return
	}

	filename := filepath.Base(urlStr)
	if filename == "" || !strings.Contains(filename, ".") {
		filename = "download"
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Status(http.StatusOK)
	io.Copy(c.Writer, resp.Body)
}
