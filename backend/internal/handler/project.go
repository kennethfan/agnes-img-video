package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	gormrepo "github.com/agnes-image-tool/backend/internal/repository/gorm"
	"github.com/agnes-image-tool/backend/internal/service"
)

type ProjectHandler struct {
	repo *gormrepo.ProjectRepository
	svc  *service.AgnesClient
}

func NewProjectHandler(repo *gormrepo.ProjectRepository, svc *service.AgnesClient) *ProjectHandler {
	return &ProjectHandler{repo: repo, svc: svc}
}

type createProjectRequest struct {
	Title string `json:"title" binding:"required"`
	Brief string `json:"brief"`
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	project := &gormrepo.Project{
		Title:  req.Title,
		Brief:  req.Brief,
		Status: "draft",
	}
	if err := h.repo.Create(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建项目失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": project})
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	projects, err := h.repo.List()
	if err != nil {
		log.Printf("[Project] 查询失败: %v", err)
		projects = []gormrepo.Project{}
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (h *ProjectHandler) GetProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}
	project, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": project})
}

type updateProjectRequest struct {
	Title  *string `json:"title"`
	Brief  *string `json:"brief"`
	Status *string `json:"status"`
	Notes  *string `json:"notes"`
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}
	project, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	var req updateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if req.Title != nil {
		project.Title = *req.Title
	}
	if req.Brief != nil {
		project.Brief = *req.Brief
	}
	if req.Status != nil {
		project.Status = *req.Status
	}
	if req.Notes != nil {
		project.Notes = *req.Notes
	}

	if err := h.repo.Update(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新项目失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": project})
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除项目失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ProjectHandler) DuplicateProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}
	project, err := h.repo.Duplicate(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "复制项目失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": project})
}

// AIRecommend 调用 AI 根据创意简报推荐创作方案（资产选择、布局建议等）
func (h *ProjectHandler) AIRecommend(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	project, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	// 收集用户的作品库资产，作为 AI 推荐的上下文
	assetContext := ""
	if repo := GetAssetRepo(); repo != nil {
		assets, _, _ := repo.List(1, 50, "", "", false)
		if len(assets) > 0 {
			var descriptions []string
			for _, a := range assets {
				desc := a.Prompt
				if desc == "" {
					desc = a.OriginalURL
				}
				descriptions = append(descriptions, fmt.Sprintf("[%s] %s", a.Type, desc))
			}
			assetContext = "用户作品库中的可用资产:\n" + strings.Join(descriptions, "\n") + "\n\n"
		}
	}

	totalTime := time.Now()

	systemPrompt := `你是一个创作项目顾问。用户会提供项目标题和创意简报，以及他们作品库中的可用资产列表。

请根据这些信息，提供以下内容的建议：
1. 创作方向分析（简要分析简报的核心创意和可行性）
2. 推荐使用哪些作品库资产（按相关性排序）
3. 建议的创作步骤（生成 -> 优化 -> 定稿）
4. 可能需要的扩展或变体

请用 Markdown 格式输出，结构清晰。`

	userPrompt := fmt.Sprintf("项目标题：%s\n\n创意简报：%s\n\n%s请根据以上信息提供创作建议。",
		project.Title, project.Brief, assetContext)

	result, err := h.svc.Chat(systemPrompt, userPrompt, 0.7)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 推荐失败: " + err.Error()})
		return
	}

	// 保存 AI 推荐结果到项目
	aiResult := map[string]any{
		"result":    result,
		"timestamp": totalTime.Format(time.RFC3339),
	}
	jsonBytes, _ := json.Marshal(aiResult)
	project.AIResult = string(jsonBytes)
	if err := h.repo.Update(project); err != nil {
		log.Printf("[Project] 保存 AI 推荐结果失败: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"project_id": project.ID,
		"result":     result,
	})
}

// AddStep 添加步骤
func (h *ProjectHandler) AddStep(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	var req struct {
		StepType string `json:"step_type" binding:"required"`
		Position int    `json:"position"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	step := &gormrepo.ProjectStep{
		ProjectID: projectID,
		StepType:  req.StepType,
		Position:  req.Position,
	}
	if err := h.repo.AddStep(step); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加步骤失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"step": step})
}

// UpdateStep 更新步骤
func (h *ProjectHandler) UpdateStep(c *gin.Context) {
	stepID, err := strconv.ParseInt(c.Param("stepId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的步骤 ID"})
		return
	}

	var req struct {
		Input  *string `json:"input"`
		Output *string `json:"output"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	step, err := h.repo.GetStepByID(stepID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "步骤不存在"})
		return
	}

	if req.Input != nil {
		step.Input = *req.Input
	}
	if req.Output != nil {
		step.Output = *req.Output
	}

	if err := h.repo.UpdateStep(step); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新步骤失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"step": step})
}

// DeleteStep 删除步骤
func (h *ProjectHandler) DeleteStep(c *gin.Context) {
	stepID, err := strconv.ParseInt(c.Param("stepId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的步骤 ID"})
		return
	}
	if err := h.repo.DeleteStep(stepID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除步骤失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// IdeateBrief 根据用户的想法生成结构化创作简报
func (h *ProjectHandler) IdeateBrief(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	project, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	var req struct {
		Idea string `json:"idea" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	systemPrompt := `你是一个创意总结专家。根据用户提供的创作想法，直接生成一份结构化创作简报。

规则：
1. 直接输出简报，不要反问用户任何问题
2. 如果用户描述中缺失某些维度（如风格、色调等），使用通用描述或合理默认值填充
3. 严格按照以下 JSON 格式输出，不要包含 markdown 代码块：

{
  "brief_text": "完整的创作简报文本（中文），包括主题、风格、氛围、构图、关键元素等描述",
  "generated_prompt": "一段可直接用于 AI 文生图的提示词（英文），包含所有确定的关键信息"
}`

	userPrompt := fmt.Sprintf("项目标题：%s\n原始简报：%s\n\n用户想法：\n%s\n\n请根据以上内容生成创作简报。", project.Title, project.Brief, req.Idea)

	summary, err := h.svc.Chat(systemPrompt, userPrompt, 0.3)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成简报失败: " + err.Error()})
		return
	}

	// 提取 JSON：AI 经常在 markdown 代码块中返回 JSON
	jsonStr := extractJSON(summary)

	var briefData struct {
		BriefText       string `json:"brief_text"`
		GeneratedPrompt string `json:"generated_prompt"`
	}
	briefText := summary
	generatedPrompt := ""

	if jsonStr != "" {
		if err := json.Unmarshal([]byte(jsonStr), &briefData); err == nil {
			briefText = briefData.BriefText
			generatedPrompt = briefData.GeneratedPrompt
		}
	}

	var aiResult map[string]any
	if project.AIResult != "" {
		json.Unmarshal([]byte(project.AIResult), &aiResult)
	}
	if aiResult == nil {
		aiResult = make(map[string]any)
	}
	aiResult["ideate_brief"] = briefText
	aiResult["ideate_prompt"] = generatedPrompt
	aiResult["ideate_time"] = time.Now().Format(time.RFC3339)

	jsonBytes, _ := json.Marshal(aiResult)
	project.AIResult = string(jsonBytes)
	if err := h.repo.Update(project); err != nil {
		log.Printf("[Project] 保存创作简报失败: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"brief_text":       briefText,
		"generated_prompt": generatedPrompt,
	})
}

// extractJSON 从 AI 响应中提取 JSON，支持剥离 markdown 代码块标记和前后文本
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if json.Valid([]byte(s)) {
		return s
	}
	// 剥离 markdown 代码块 ```json ... ``` 或 ```JSON ... ```
	for _, marker := range []string{"```json", "```JSON"} {
		if idx := strings.Index(s, marker); idx >= 0 {
			rest := s[idx+len(marker):]
			if end := strings.Index(rest, "```"); end >= 0 {
				return strings.TrimSpace(rest[:end])
			}
		}
	}
	// 剥离普通代码块 ``` ... ```
	if idx := strings.Index(s, "```"); idx >= 0 {
		rest := s[idx+3:]
		if end := strings.Index(rest, "```"); end >= 0 {
			return strings.TrimSpace(rest[:end])
		}
	}
	// 无代码块：找第一个 { 和最后一个 } 尝试提取
	if start := strings.Index(s, "{"); start >= 0 {
		if end := strings.LastIndex(s, "}"); end > start {
			candidate := s[start : end+1]
			if json.Valid([]byte(candidate)) {
				return candidate
			}
		}
	}
	return ""
}
