# 图片精修（Image-to-Image）流程改进设计

**日期**: 2026-07-11
**状态**: 待审批

## 1. 动机

当前图生图流程存在以下问题：
- **无迭代闭环**：精修结果无法直接作为输入继续精修，用户需手动重新上传/粘贴 URL
- **输入源受限**：只能上传文件或输入 URL，无法从作品库中选取已有图片

## 2. 范围

仅涉及前端修改，无后端变更。

| 文件 | 改动类型 |
|------|---------|
| `frontend/src/components/ImageResult.vue` | 新增按钮 + redo 触发 |
| `frontend/src/views/ImageToImage.vue` | 新增按钮 + 作品库选择弹窗 |
| `frontend/src/stores/redo.ts` | `RedoData.inputMode` 类型扩展（`url` 已支持，无需修改） |

## 3. 功能设计

### 3.1 迭代闭环 — 「继续精修」按钮

**位置**: `ImageResult.vue`，每个结果图操作区

**触发条件**: `props.mode === 'image2image'` 时显示该按钮

**点击行为**:
1. 构造 `RedoData` 对象：
   ```ts
   {
     mode: 'image2image',
     imageUrl: resultUrl,      // 当前精修结果图片 URL
     prompt,                   // 保留原始 prompt
     negativePrompt,           // 保留原始负面提示词
     size,                     // 保留原始尺寸
     strength,                 // 保留原始重绘强度
     inputMode: 'url',         // 切换到 URL 模式
   }
   ```
2. 调用 `redoStore.setRedoData(data)`
3. `setRedoData` 通过 Vue Router 导航到 ImageToImage 页面（路由名 `img2img`）
4. ImageToImage 页面 `watch(redoStore.redoData, { flush: 'sync' })` 接收到数据，填充输入表单

**现有机制复用**: `redo.ts` 中的 `modeToTab.image2image === 'img2img'` 映射已存在，`ImageToImage.vue` 的 `watch(redoData)` 也已有对应处理逻辑（第 40-49 行）。

### 3.2 输入源扩展 — 「从作品库选择」按钮

**位置**: `ImageToImage.vue`，`inputMode` radio-group 下方或旁边

**UI 组件**:
- `el-button` 按钮，文案「从作品库选择」
- 点击弹出 `el-dialog`，标题「选择图片」

**弹窗内容**:
- 调用现有 `/api/v1/assets` 接口获取作品库图片列表
- 展示缩略图网格（参考 Assets.vue 的画廊布局，可简化）
- 每张图可点击选中，点击后突出显示

**确认行为**:
- 点击某张图片 → 弹窗关闭
- 图片 URL 填入 `imageUrl` ref
- `inputMode` 自动切换为 `'url'`
- 预览区展示该图片

**数据获取**: 复用 `src/api/assets.ts` 中的 `getAssets()` 接口即可。

## 4. UI 变化

### ImageResult.vue 操作区（追加后）

```
[ 下载 ]  [ 保存到作品库 ]  [ 继续精修 ]   ← 新增按钮，仅 mode === 'image2image' 时显示
```

### ImageToImage.vue 表单区

```
输入方式: [上传图片] [图片 URL]              ← 原 radio-group
         [从作品库选择]                     ← 新增按钮

上传/URL 区域 ...                           ← 不变

风格描述 / 负面提示词 / 尺寸 / 重绘强度       ← 不变
```

## 5. 数据流

```
ImageResult "继续精修" 点击
    │
    ▼
redoStore.setRedoData({ mode: 'image2image', imageUrl, ... })
    │
    ├─ sessionStorage.setItem('redoData', ...)
    └─ router.push({ name: 'img2img' })
          │
          ▼
ImageToImage.vue watch(redoData)
    │
    └─ 填充: prompt, negativePrompt, size, strength, imageUrl, inputMode='url'
```

```
ImageToImage "从作品库选择" 点击
    │
    ▼
el-dialog 弹窗
    │
    └─ GET /api/v1/assets → 缩略图网格
          │
          └─ 选中图片 → imageUrl = selected.url, inputMode = 'url'
```

## 6. 不做的事（明确排除）

- 不修改后端接口
- 不修改作品库 / history 的 CRUD
- 不处理视频精修
- 不做会话内版本历史
- 不做原图对比

## 7. 测试检查点

- [ ] ImageResult 上「继续精修」按钮只对 `mode === 'image2image'` 显示，对其他模式隐藏
- [ ] 点击「继续精修」后，目标页面正确填充所有参数（prompt、negativePrompt、size、strength、imageUrl）
- [ ] 从作品库选择图片后，预览正确显示，`inputMode` 正确切换到 `url`
- [ ] 「保存到作品库」按钮在精修结果图下正常保存
