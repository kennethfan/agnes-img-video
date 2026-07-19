# Ideate 步骤简化设计文档

> 将创意发想步骤从多轮聊天改为单次输入 → 单次输出，解决 API 超时和用户体验臃肿问题。

---

## 现状问题

1. **API 频繁超时**：Agnes Chat API 响应不稳定，单次请求常超 180s
2. **对话流程冗余**：用户需要多次输入 → 等待回复，超时体验更差
3. **步骤间重叠**：AI Recommend → Ideate → Generate 三步内容有重复，用户需要反复描述想法

## 设计目标

1. 将 Ideate 步骤从多轮聊天改为「一次想法输入 → 一次 AI 生成简报」
2. 消除聊天状态的复杂性（消息列表、滚动、上下文截断）
3. 保留核心价值：用户仍能获得结构化的创作简报

## 方案

### IdeateStep.vue（重写）

**旧**：消息列表 + 输入框 + 发送按钮 + "生成简报"按钮

**新**：文本域 + "生成简报"按钮 + 结果卡片

**输入框占位文案**（引导用户自由书写，无需太长）：

> 描述你想创作的内容，比如主题、风格、氛围... 不需要很详细，AI 会帮你完善。

布局：

```
┌────────────────────────────────────────┐
│  创意发想                               │
│  ┌────────────────────────────────────┐│
│  │ 描述你想创作的内容，比如主题、风格、   ││
│  │ 氛围... 不需要很详细，AI 会帮你完善  ││
│  │                                    ││
│  └────────────────────────────────────┘│
│                                         │
│  [ 🎯 生成创作简报 ]  ← loading 状态     │
│                                         │
│  ── 生成结果（成功后） ──               │
│  ┌────────────────────────────────────┐│
│  │ 📋 创作主题：xxx                   ││
│  │ 风格方向：xxx                      ││
│  │ 视觉要点：xxx                      ││
│  │ 建议提示词：xxx                    ││
│  │                                    ││
│  │ [复制简报] [重新生成] [继续编辑]     ││
│  └────────────────────────────────────┘│
│                                         │
│  [ 下一步：生成图片 → ]                  │
└────────────────────────────────────────┘
```

### 后端改动

**删除**：
- `IdeateChat` 方法（不再需要多轮对话）
- 路由 `POST /projects/:id/ideate-chat`

**修改**：
- `IdeateBrief` 改为直接接收 `{ idea: string }` 而不是消息数组
- **System prompt 重写**：明确要求 AI"不要反问用户，缺失的维度用通用描述填充，直接输出 JSON 简报"
- 去掉 chat context 拼接逻辑

**持久化**：
- ideate 结果存入 `project.ai_result`，字段 `ideate_brief` / `ideate_prompt` / `ideate_time`（与现有逻辑一致）

### 前端改动

| 文件 | 改动 |
|---|---|
| `IdeateStep.vue` | 重写：textarea + button + result card，去掉 message list、chat bubbles、auto-scroll |
| `ProjectEditor.vue` | `onBriefGenerated` 逻辑简化，去掉 chat 相关状态 |
| `GenStep.vue` | 不变（已支持 `initialPrompt` prop） |
| `api/ideate.ts` | 去掉 `ideateChat`，修改 `ideateBrief` 参数签名 |

### 数据流

```
用户输入想法 → ideateBrief({ idea }) → POST /api/v1/projects/:id/ideate-brief
  → ProjectHandler.IdeateBrief()
    → 构建 system prompt + 用户想法 → AgnesClient.ChatWithHistory()
    → AI 返回 JSON { brief_text, generated_prompt }
    → 存入 project.ai_result（可选）
    → 返回前端
  → 前端渲染结果卡片
  → 用户点击「下一步」→ GenStep 接收 generated_prompt 作为 initialPrompt
```

### 错误处理

- 调用超时 → 显示友好错误 + 重试按钮
- AI 返回非 JSON → 走现有 extractJSON 兜底
- 网络错误 → ElMessage.error + 按钮恢复可点击

## 边界情况

- **输入为空**：按钮置灰，提示"请输入创作想法"
- **输入过长**：限制 500 字，超出时提示
- **快速连续点击**：按钮 loading 期间 disable
- **已生成简报后再次编辑**：清空简报结果，重新生成

## 不做的改动

- 不对 ProjectEditor 的步骤数做修改（保留 4 步：ideate → generate → refine → finalize）
- 不引入 SSE / 流式
- 不修改数据库 schema
