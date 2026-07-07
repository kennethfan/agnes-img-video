# Workflow Wizard — 创作工作流向导设计

## 概述

在 Agnes Creator Studio 中新增**创作工作流**功能，以分步向导（Step-by-Step Wizard）模式引导用户完成复杂的多步创作流程。用户无需手动在各个页面之间切换，向导自动串联相关 API，降低创作门槛。

### 核心设计原则

- **分步引导**：每步一个明确任务，用户完成后自动推进
- **步骤可逆**：用户可以回退到上一步修改参数
- **状态持久**：每一步的结果保存在本地 ref 中，最后统一提交
- **复用现有 API**：不新增后端接口，全部复用现有的 image/video/chat API

### 导航位置

在 NavSidebar 中新增 `workflow` 分组，图标 ⚡，包含工作流入库入口。

## 工作流一：小说生成 (Novel Generation)

### 流程

```
步骤1: 选题设定 → 步骤2: 风格/流派 → 步骤3: 角色设置 → 步骤4: 大纲生成
→ 步骤5: 确认大纲 → 步骤6: 逐章生成（用户逐章触发）→ 步骤7: 配插图
→ 步骤8: 导出
```

### 步骤详情

| 步骤 | UI | 后端调用 |
|---|---|---|
| 1. 选题 | 文本输入框，placeholder "输入小说主题或一句话灵感" | 无 |
| 2. 风格流派 | 标签选择器：玄幻/科幻/言情/悬疑/现实主义/历史，可选带"选择风格灵感"入口 | `POST /ideas/expand` 扩展风格建议 |
| 3. 角色设置 | 动态表单：角色名 + 性格描述 + 外观，可添加/删除角色 | 无 |
| 4. 大纲生成 | 加载状态 → 展示生成的大纲 | `POST /ideas/expand` 生成大纲，prompt 包含主题+风格+角色 |
| 5. 确认大纲 | 展示大纲全文 + "重新生成"按钮 + "确认"按钮 + "手动编辑" | 无 |
| 6. 逐章生成 | 显示章节列表 + "生成下一章"按钮，每章生成后展示全文 | `POST /ideas/expand` 逐章生成，携带前文作为 context |
| 7. 配插图 | 每章旁 "生成插图" 按钮，选择风格后生成 | `POST /images/text-to-image` |
| 8. 导出 | 选择格式：纯文本 / Markdown / HTML | 前端拼接 |

### 数据流

```
WizardStore {
  currentStep: number
  novelData: {
    theme: string
    genre: string
    characters: Array<{ name, personality, appearance }>
    outline: string
    chapters: Array<{ title, content, illustration? }>
  }
}
```

所有步骤数据存储在 Pinia store 中，刷新页面时通过 sessionStorage 恢复。

### 前端组件

- `views/WorkflowWizard.vue` — 向导容器，管理步骤切换和进度指示
- `views/novel/` 目录下各步骤组件

---

## 工作流二：图片精修 (Image Refinement)

### 流程

```
双入口选择
├── 文生图入口: 输入提示词 → 文生图 → 进入精修
└── 上传入口: 上传/拖拽图片 → 进入精修

精修步骤:
  步骤1: 选择/生成原始图片
  步骤2: 精修调参（显示原图 + 参数面板）
  步骤3: 预览对比（原图 vs 精修结果）
  步骤4: 确认导出
```

### 步骤详情

| 步骤 | UI | 后端调用 |
|---|---|---|
| 1a. 文生图 | 提示词输入 + 尺寸选择 + "生成"按钮 | `POST /images/text-to-image` |
| 1b. 上传 | 拖拽上传区 / 点击选择文件 | 无（前端读取 File） |
| 2. 精修调参 | 双栏：左原图预览、右参数面板（提示词、强度 0-1 滑块、尺寸选择）+ "开始精修"按钮 | `POST /images/image-to-image`（multipart 或 image_url） |
| 3. 预览对比 | 左右对比/滑动对比：原图 vs 结果 + "继续精修" + "满意" | 无 |
| 4. 导出 | 选择导出格式 + 下载按钮 | 无（前端触发下载） |

### 入口复用

精修步骤直接复用现有的 `ImageResult.vue` 组件展示结果，参数面板复用当前 `ImageToImage.vue` 中的表单控件。

---

## 工作流三：漫画生成 (Comic Generation)

### 流程

```
步骤1: 设定主题 → 步骤2: 选择分格布局 → 步骤3: 逐格填写提示词
→ 步骤4: 批量生成 → 步骤5: 添加台词/气泡 → 步骤6: 预览导出
```

### 步骤详情

| 步骤 | UI | 后端调用 |
|---|---|---|
| 1. 设定主题 | 文本输入框，输入漫画主题/描述 | 无 |
| 2. 分格布局 | 网格选择器：1格(单图) / 2格(上下) / 4格(田字) / 6格(2x3) | 无 |
| 3. 逐格填写 | 布局预览网格，每格可输入独立的提示词。初始提示词基于主题自动生成，用户可修改 | 调用 `POST /ideas/expand` 为每格生成初始提示词建议 |
| 4. 批量生成 | "全部生成"按钮，每格显示生成进度 | `POST /images/batch` 批量生成，或逐格 `POST /images/text-to-image` |
| 5. 添加台词 | 每格上方叠加文本输入框，可拖拽调整位置 | 无（前端 Canvas 合成） |
| 6. 预览导出 | 完整漫画预览 + 导出为 HTML/PNG 序列/PDF | 无（前端渲染） |

### 组件复用

- 分格布局使用 CSS Grid 渲染
- 每格复用已有的图片展示样式
- 台词叠加使用绝对定位 + 文本输入

---

## 工作流四：视频生成 (Video Generation)

> ⚠️ **暂缓设计，待进一步讨论。**

---

## 技术实现

### 前端新增文件

```
frontend/src/
  views/
    WorkflowWizard.vue          # 向导容器 + 工作流选择
    novel/
      StepTheme.vue
      StepGenre.vue
      StepCharacters.vue
      StepOutline.vue
      StepOutlineConfirm.vue
      StepGenerateChapters.vue
      StepIllustrate.vue
      StepExport.vue
    image/
      StepSource.vue            # 文生图 / 上传入口
      StepRefine.vue            # 精修调参
      StepCompare.vue           # 对比预览
      StepExport.vue
    comic/
      StepTheme.vue
      StepLayout.vue
      StepPanels.vue            # 逐格填提示词
      StepGenerate.vue          # 批量生成
      StepCaptions.vue          # 添加台词
      StepExport.vue
  stores/
    wizard.ts                   # 工作流状态管理（Pinia）
```

### NavSidebar 变更

新增 `workflow` 分组，包含 4 个入口项（视频入口暂时 disabled）：

```typescript
{
  id: 'workflow',
  icon: '⚡',
  label: '创作',
  items: [
    { id: 'novel', label: '小说生成' },
    { id: 'image_refine', label: '图片精修' },
    { id: 'comic', label: '漫画生成' },
    { id: 'video_wizard', label: '视频生成', disabled: true },
  ],
}
```

### App.vue 变更

新增 `WorkflowWizard` 组件引用和条件渲染：

```vue
<WorkflowWizard v-else-if="activePage === 'novel'" />
<WorkflowWizard v-else-if="activePage === 'image_refine'" />
<WorkflowWizard v-else-if="activePage === 'comic'" />
```

### 后端变更

**无需新增后端接口**。所有能力复用现有 API：

| 功能 | 复用接口 |
|---|---|
| 小说大纲/章节生成 | `POST /ideas/expand`（chat completions） |
| 小说插图 | `POST /images/text-to-image` |
| 图片精修 | `POST /images/image-to-image` |
| 图片上传转 base64 | 已实现（multipart → `extra_body.image`） |
| 漫画批量生成 | `POST /images/batch` 或逐张 `POST /images/text-to-image` |

---

## 实现计划建议

建议分 3 个里程碑实现：

1. **MVP（图片精修 + 漫画）**：实现 WorkflowWizard 容器 + 图片精修完整流程 + 漫画完整流程
2. **小说生成**：实现小说完整流程（工作量最大，纯文本生成）
3. **视频生成**：待设计确认后实现
