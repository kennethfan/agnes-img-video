# 创作项目工作流体验优化

**日期**: 2026-07-11
**状态**: 设计稿

## 1. 目标

解决创作项目编辑器中三个核心体验问题：

1. 文生图/图生图生成的结果无法直接保存到作品库
2. 图生图/优化步骤无法从作品库选择参考图
3. 生成步骤的图片 URL 不能自动带到优化步骤

## 2. 改动清单

涉及 4 个 Vue 组件，1 个新组件，无后端改动（已有 API 全覆盖）。

### 2.1 跨步骤自动传图

**文件**: `ProjectEditor.vue`, `GenStep.vue`, `RefineStep.vue`

**实现**:

- `GenStep.vue` emit `generated(urls: string[])` 事件，在生成完成后触发
- `ProjectEditor.vue` 增加 `latestGenUrls` 响应式变量，监听 `@generated` 事件
- 用户从 generate 步骤点击「下一步」到 refine 步骤时，将 `latestGenUrls[0]` 传入 RefineStep
- `RefineStep.vue` 新增 `defaultImageUrl` prop，有值时自动填充源图 URL 输入框
- 用户仍可手动修改/清空源图 URL

**边界情况**:
- GenStep 未生成图片直接点下一步 → RefineStep 不传 defaultImageUrl，保持空白
- 用户从 refine 回到 generate 重新生成 → 覆盖 latestGenUrls
- RefineStep 已手动修改过 URL 后，再次从 generate 下一步 → 覆盖为用户新生成的图片（最近一次操作为主）

### 2.2 作品库选择源图

**新组件**: `AssetPickerDialog.vue`（`frontend/src/components/`）

**实现**:

- 以 `el-dialog` 弹窗形式呈现，宽度 70vw/700px，高度 80vh
- 加载时调用 `getAssets({ type: 'image' })` 获取作品库图片
- 网格布局 (el-row/el-col) 展示缩略图,每行 4 列
- 点击缩略图选中（高亮边框），同一时间只允许选中一张
- 底部「确认」和「取消」按钮
- 确认后通过 emit `selected(url: string)` 返回选中资产的 `original_url`

**集成位置**:

- `GenStep.vue` 图生图模式：`imageUrl` 输入框右侧增加 «从作品库» 按钮
- `RefineStep.vue`：`sourceImage` 输入框右侧增加同样按钮

**边界情况**:
- 作品库为空时，弹窗显示空状态提示，按钮置灰
- 用户打开弹窗后取消 → 不影响现有输入
- 选中后再次点击同一资产可取消选中

### 2.3 生成结果显式保存到作品库

**文件改动**: `GenStep.vue`, `RefineStep.vue`, `FinalStep.vue`

**实现**:

- 每张生成结果图片下方增加 «保存到作品库» 按钮（使用已有的 `saveAsset` API）
- 按钮位置在图片右下角或正下方，使用图标 `Plus` + 文字「保存到作品库」
- 点击后调用 `saveAsset({ image_url, prompt, mode })`，成功后按钮变为 ✓ 已入库（禁用态）
- 保存失败显示错误提示但不阻塞用户操作
- **删除** `ImageResult.vue` 中的自动静默保存逻辑（`autoSaveEnabled` 相关代码），改为全手动控制

**FinalStep.vue 的保存**: 生成记录和优化记录中的图片来自 `step.output`（单条 URL）。在每个 `step.output` 展示区下方增加保存按钮，调用 `saveAsset({ image_url: step.output, prompt: step.input?.[输入截取], mode: 'image' })`。prompt 从 `step.input` JSON 中解析，若解析失败回退为「来自创作项目」。

**ImageResult 移除自动保存**: 删除以下代码块:
- `const autoSaveEnabled = ref(...)` 及 `localStorage` 读取
- `const autoSavedUrls = ref<Set<string>>(new Set())`
- `watch(() => props.images, ...)` 整个 watch 自动保存逻辑
- 保留 `savingUrls` 和手动保存逻辑(`saveCurrentAsset` 方法)供后续使用

**边界情况**:
- 同一张图片可多次点击保存（多次调用 API，服务端幂等或新建记录）
- 已保存的图片在页面刷新后状态丢失（按钮恢复为可点击）— 合理，因为入库状态不持久化到本地

## 3. 文件改动汇总

| 文件 | 改动类型 | 改动内容 |
|------|----------|----------|
| `frontend/src/components/AssetPickerDialog.vue` | **新增** | 作品库选择器组件 |
| `frontend/src/components/GenStep.vue` | 修改 | emit `generated` 事件 + 作品库按钮 + 保存按钮 |
| `frontend/src/components/RefineStep.vue` | 修改 | 新增 `defaultImageUrl` prop + 作品库按钮 + 保存按钮 |
| `frontend/src/components/FinalStep.vue` | 修改 | 生成记录下方增加保存按钮 |
| `frontend/src/components/ImageResult.vue` | 修改 | 移除自动静默保存逻辑 |
| `frontend/src/views/ProjectEditor.vue` | 修改 | 管理 `latestGenUrls` 跨步骤传图 |

无后端改动，无路由变动，无新增 API。

## 4. 数据流

```
GenStep 生成完成
  → emit('generated', urls)                 // 上报图片到编辑器
  → url 图片下方 «保存到作品库» → saveAsset API

用户点击「下一步」(generate → refine)
  → ProjectEditor 将 latestGenUrls[0] → RefineStep.defaultImageUrl
  → RefineStep 自动填充 sourceImage 输入框

GenStep/RefineStep 的「从作品库」按钮
  → 弹出 AssetPickerDialog → 选择 → 填充 URL
```

## 5. 未纳入范围

- 步骤状态持久化到 ProjectStep（刷新后保留）— 后续优化考虑
- 多结果选择（勾选多张图批量传给下一步）— 后续优化考虑
- 步骤导航的状态记忆（回到上一步保留输入）— 后续优化考虑
