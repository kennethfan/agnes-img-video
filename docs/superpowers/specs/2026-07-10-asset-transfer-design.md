# Asset Transfer (转存) — 服务端统一存储策略

**Date:** 2026-07-10
**Status:** Draft

## 问题

"转存"和"保存到作品库"共享同一套存储决策逻辑（根据 `storage_target` 决定本地下载 / GitHub上传 / 两者都做），但当前代码中这套逻辑耦合在 `AssetHandler.SaveAsset` 里，无法复用。

需要在作品库页面新增"转存"功能：对已入库的 asset，根据当前存储设置补全 `local_path` 和 `github_url`。

## 方案

### 1. 提取共享方法 `AssetHandler.storeFile()`

从 `SaveAsset` 中抽取"下载远程文件 → 按策略本地保存 / GitHub上传"逻辑为独立方法：

```go
// storeFile 下载远程文件并根据 storage_target 处理存储
// 返回 (localPath, githubURL, error)
func (h *AssetHandler) storeFile(imageURL string, assetType string) (string, string, error)
```

内部逻辑（从当前 `SaveAsset` 的远程 URL 分支提取）：
- 读取 `storage_target`（local / github / both）
- 下载远程文件到 `outputs/` 目录
- 根据策略决定是否保留本地文件 → `localPath`
- 根据策略决定是否上传 GitHub → `githubURL`
- 纯 GitHub 模式上传后删除本地临时文件

### 2. 重构 `SaveAsset`

```go
func (h *AssetHandler) SaveAsset(c *gin.Context) {
    // ... 参数校验、类型判断 ...
    // 远程 URL 分支：直接调用 storeFile()
    localPath, githubURL := h.storeFile(req.ImageURL, assetType)
    // ... 写入 assets 表 ...
}
```

行为不变，代码复用。

### 3. 新增转存端点

```
POST /api/v1/assets/:id/transfer
```

Handler 逻辑：
1. 根据 `:id` 查出已有 asset
2. 如果 asset 已有完整的 `local_path` 和 `github_url`（按当前策略判断），直接返回
3. 调用 `storeFile(asset.OriginalURL, asset.Type)`
4. 更新 asset 的 `local_path` / `github_url` 字段
5. 返回更新后的 asset

### 4. 仓库层新增方法

```go
// AssetRepository 增加
UpdateStoragePaths(id int64, localPath, githubURL string) error
```

同时更新 `local_path` 和 `github_url` 两个字段，替代现在单字段的 `UpdateGithubURL`。

### 5. 前端 Assets 页面

在 AssetCard 或 Assets 视图的操作区域增加"转存"按钮：

- 只有 `github_url` 为空的 asset 才显示（或始终显示但禁用状态提示"已转存"）
- 调用 `POST /api/v1/assets/:id/transfer`
- 成功后刷新列表，展示 `github_url` 为共享链接

## 数据流

```
用户点击"转存"
  → POST /api/v1/assets/:id/transfer
  → 查询 asset 记录
  → 读取 storage_target 设置
  → 下载远程文件到 outputs/
  → （可选）上传到 GitHub
  → 更新 asset.local_path / asset.github_url
  → 返回更新后的 asset
  → 前端刷新展示
```

## 边界情况

| 场景 | 行为 |
|------|------|
| asset 已是本地路径（非远程 URL） | 直接返回，无需下载 |
| storage_target = local | 只下载到本地，不传 GitHub |
| storage_target = github | 下载后上传 GitHub，删除本地临时文件 |
| storage_target = both | 下载到本地 + 上传 GitHub，都保留 |
| 本地文件已存在 | 覆盖下载 |
| GitHub 上传失败 | 记录日志，不影响本地保存 |
| asset 的 local_path 已有值 | 覆盖更新 |
| asset 的 github_url 已有值 | 覆盖更新 |

## 依赖与约束

- `storeFile` 需要访问 `githubStorage`（`handler` 包级全局变量）和 `settingsRepo`（`AssetHandler` 已注入）。`githubStorage` 保持全局变量方式调用，不做额外注入。
- 仓库保留 `UpdateGithubURL` 不动，新增 `UpdateStoragePaths`。已有调用方不受影响。
- 不新增 GORM 模型或 DB 迁移。
- 暂不改动 `saveHistoryRecord()` 中的存储逻辑（历史记录和作品库保持独立）。
