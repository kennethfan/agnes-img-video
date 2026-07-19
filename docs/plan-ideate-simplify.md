# Ideate 步骤简化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将创意发想从多轮聊天改为单次输入→单次输出，重写系统 prompt 防止 AI 反问

**Architecture:** 
- 后端去掉 `IdeateChat` 路由和方法，`IdeateBrief` 入参改为 `{ idea: string }`，system prompt 重写为「直接输出简报不反问」
- 前端 IdeateStep.vue 从聊天 UI 重写为 textarea + 按钮 + 结果卡片

**Tech Stack:** Go 1.25 + Gin (backend), Vue 3 + TypeScript 6 + Element Plus (frontend)

## Global Constraints

- Go: 遵循 project-layout 标准
- Vue: Composition API + `<script setup>` only
- TypeScript: `erasableSyntaxOnly` — 不用 enum
- 错误信息使用中文
- 不要引入新的依赖

---

### Task 1: Backend — Rewrite IdeateBrief handler

**Files:**
- Modify: `backend/internal/handler/project.go:358-451`
- Test: `go build ./cmd/server`

**Interfaces:**
- Consumes: `ProjectRepository.GetByID(id)`, `AgnesClient.Chat(system, user, temp)`, `extractJSON()`, `project.ai_result` (JSON string)
- Produces: `POST /api/v1/projects/:id/ideate-brief` accepts `{ "idea": "..." }`, returns `{ brief_text, generated_prompt }`

- [ ] **Step 1: Change IdeateBrief request body to accept `{ idea: string }`**

替换 `IdeateBrief` 中的 request struct，从 `Messages` 数组改为单个 `Idea` 字段：

```go
var req struct {
    Idea string `json:"idea" binding:"required"`
}
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
    return
}
```

- [ ] **Step 2: Rewrite system prompt — 禁止反问，直接输出简报**

替换原来的 `systemPrompt` 变量：

```go
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
```

- [ ] **Step 3: 去掉 chat context 拼接逻辑**

删除 `var conversation strings.Builder` 及其后续拼接循环（原 382-389 行），改为直接使用 `req.Idea`。

- [ ] **Step 4: Build 验证**

```bash
cd backend && go build ./cmd/server
```
预期：exit 0

---

### Task 2: Backend — Remove IdeateChat handler and route

**Files:**
- Modify: `backend/internal/handler/project.go:298-355`
- Modify: `backend/cmd/server/main.go`
- Test: `go build ./cmd/server`

**Interfaces:**
- Consumes: (nothing — removing code)
- Produces: Route `POST /projects/:id/ideate-chat` removed

- [ ] **Step 1: 删除 IdeateChat 方法**

从 `project.go` 中删除整个 `IdeateChat` 方法（第 298-355 行）。

- [ ] **Step 2: 删除路由注册**

从 `main.go` 中找到并删除：
```go
projects.POST("/:id/ideate-chat", handler.IdeateChat)
```

- [ ] **Step 3: 检查有其他地方引用了 `IdeateChat`（如测试文件）**

```bash
cd backend && grep -rn "IdeateChat" internal/ --include="*.go"
```
预期：只有类型定义相关引用（如果有的话），不应有调用方。

- [ ] **Step 4: Build 验证**

```bash
cd backend && go build ./cmd/server
```
预期：exit 0

---

### Task 3: Frontend — Update API client

**Files:**
- Modify: `frontend/src/api/ideate.ts`

**Interfaces:**
- Produces: `ideateBrief(projectId, idea)` — 新签名，返回 `{ brief_text, generated_prompt }`

- [ ] **Step 1: 重写 api/ideate.ts**

```typescript
import client from './client'

export async function ideateBrief(
  projectId: number,
  idea: string
): Promise<{ brief_text: string; generated_prompt: string }> {
  const res = await client.post(`/api/v1/projects/${projectId}/ideate-brief`, { idea })
  return res.data
}
```

删除 `ChatMessage` 接口和 `ideateChat` 函数。

- [ ] **Step 2: 检查引用**

```bash
grep -rn "ideateChat\|ChatMessage" frontend/src/ --include="*.ts" --include="*.vue"
```
预期：只应出现在已修改的文件中（或无引用）。

---

### Task 4: Frontend — Rewrite IdeateStep.vue

**Files:**
- Rewrite: `frontend/src/components/IdeateStep.vue`
- Test: `pnpm build`

**Interfaces:**
- Consumes: `project: Project | null` (prop), `ideateBrief(projectId, idea)` (API)
- Produces: `briefGenerated(briefText, prompt)` (emit)

- [ ] **Step 1: 重写组件模板和数据流**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Promotion, EditPen, Check, Refresh } from '@element-plus/icons-vue'
import { ideateBrief } from '../api/ideate'
import type { Project } from '../types'

const props = defineProps<{ project: Project | null }>()
const emit = defineEmits<{
  briefGenerated: [briefText: string, prompt: string]
}>()

const idea = ref('')
const loading = ref(false)
const briefDone = ref(false)
const briefResult = ref<{ brief_text: string; generated_prompt: string } | null>(null)
const editing = ref(true)

async function generateBrief() {
  const text = idea.value.trim()
  if (!text) {
    ElMessage.warning('请输入创作想法')
    return
  }
  if (!props.project) return

  loading.value = true
  try {
    const result = await ideateBrief(props.project.id, text)
    briefResult.value = result
    briefDone.value = true
    editing.value = false
    emit('briefGenerated', result.brief_text, result.generated_prompt)
    ElMessage.success('创作简报已生成')
  } catch (e: any) {
    ElMessage.error('生成失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function copyResult(text: string) {
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制')
  })
}
</script>

<template>
  <div class="ideate-step">
    <!-- 输入区 -->
    <div v-if="!briefDone || editing" class="input-section">
      <label class="section-label">写下你的创作想法</label>
      <el-input
        v-model="idea"
        type="textarea"
        :rows="6"
        placeholder="描述你想创作的内容，比如主题、风格、氛围... 不需要很详细，AI 会帮你完善。"
        :maxlength="500"
        show-word-limit
      />
      <div class="input-actions">
        <el-button type="primary" :loading="loading" :disabled="!idea.trim()" @click="generateBrief">
          <el-icon><Promotion /></el-icon> 生成创作简报
        </el-button>
      </div>
    </div>

    <!-- 结果卡片 -->
    <div v-if="briefDone && briefResult" class="result-section">
      <div class="result-header">
        <span class="section-label">创作简报</span>
        <div class="result-actions">
          <el-button size="small" text @click="copyResult(briefResult.brief_text)">
            <el-icon><Check /></el-icon> 复制简报
          </el-button>
          <el-button size="small" text @click="copyResult(briefResult.generated_prompt)">
            <el-icon><Check /></el-icon> 复制提示词
          </el-button>
          <el-button size="small" text @click="loading ? null : (editing = !editing); briefDone = false; briefResult = null">
            <el-icon><EditPen /></el-icon> 重新编辑
          </el-button>
          <el-button size="small" text :loading="loading" @click="generateBrief">
            <el-icon><Refresh /></el-icon> 重新生成
          </el-button>
        </div>
      </div>
      <el-card class="result-card">
        <div class="brief-text">{{ briefResult.brief_text }}</div>
        <el-divider />
        <div class="prompt-section">
          <div class="prompt-label">AI 生成提示词</div>
          <pre class="prompt-text">{{ briefResult.generated_prompt }}</pre>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.ideate-step {
  max-width: 720px;
  margin: 0 auto;
}
.section-label {
  display: block;
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 12px;
  color: #303133;
}
.input-section {
  margin-bottom: 24px;
}
.input-actions {
  margin-top: 16px;
  display: flex;
  justify-content: center;
}
.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  flex-wrap: wrap;
  gap: 8px;
}
.result-actions {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}
.result-card {
  line-height: 1.8;
}
.brief-text {
  font-size: 14px;
  color: #303133;
  white-space: pre-wrap;
}
.prompt-section {
  margin-top: 8px;
}
.prompt-label {
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
}
.prompt-text {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: #606266;
  margin: 0;
}
</style>
```

- [ ] **Step 2: 验证 build**

```bash
cd frontend && pnpm build
```
预期：exit 0

---

### Task 5: Frontend — Simplify ProjectEditor.vue

**Files:**
- Modify: `frontend/src/views/ProjectEditor.vue`

**Interfaces:**
- Consumes: (tidying up — removing chat state that no component references)

- [ ] **Step 1: 检查是否可以简化**

IdeateStep 不再需要 chat 状态传递，`onBriefGenerated` 回调签名未变（`briefText, prompt`），所以 ProjectEditor.vue 不需要改动。

确认：查看模板中 `onBriefGenerated` 只在第 93 行的 `@brief-generated="onBriefGenerated"` 使用，该 emit 在新组件中保留，所以无改动必要。

无实际代码变更。

---

### Task 6: Verify

**Files:**
- (no file changes — full project verification)

- [ ] **Step 1: 后端 build**

```bash
cd backend && go build ./cmd/server
```
预期：exit 0

- [ ] **Step 2: 前端 build**

```bash
cd frontend && pnpm build
```
预期：exit 0

- [ ] **Step 3: LSP 诊断检查**

```bash
# 项目根目录
cd backend && go vet ./...
```
预期：无 error
