# 独立作品库设计文档

**日期**: 2026-07-10
**状态**: 已确认

## 背景

作品库（Asset Gallery）当前与 `history` 表耦合 —— `AssetHandler` 直接依赖 `HistoryRepository`，作品数据实际存储在 `history` 表中。这与业务需求不符：

- **History** = 任务运行记录（自动生成，用户无感）
- **作品库** = 用户精选存档（只有用户主动保存才写入）

需要将它们彻底解耦，使作品库成为独立的数据系统。

## 数据模型

### Asset 表（新增）

```go
type Asset struct {
    ID          int64  `gorm:"primaryKey"`
    Mode        string `gorm:"index"`       // text2image / image2video / batch / ...
    Prompt      string
    Type        string                     // "image" | "video"
    Time        string                     // 保存时间
    Favorite    bool                       // 是否收藏
    OriginalURL string `gorm:"column:original_url"`  // 原始生成 URL
    LocalPath   string `gorm:"column:local_path"`     // 本地文件路径
    GitHubURL   string `gorm:"column:github_url"`     // GitHub 转存后地址
}
```

- `OriginalURL`：生成完成后前端展示的图片 URL（如 `/outputs/xxx.png`）
- `LocalPath`：服务端解析出的本地绝对路径（用于批量下载/删除文件）
- `GitHubURL`：用户主动转存到 GitHub 后的地址（一期暂不实现自动关联）

### Favorite 模型移除

收藏功能内联到 `Asset.Favorite` 布尔字段，移除原有的独立 `Favorite` 表。

### 前端 AssetItem

```ts
export interface AssetItem {
  id: number
  mode: string
  prompt: string
  type: 'image' | 'video'
  time: string
  favorite: boolean
  original_url: string
  local_path: string
  github_url: string
  thumbnail: string
}
```

## API 设计

### 新增：保存到作品库

```
POST /api/v1/assets
Content-Type: application/json

{ "image_url": "/outputs/xxx_20260710_123456.png", "prompt": "一只猫", "mode": "text2image" }

→ 200
{ "id": 1 }
```

### 重构：现有资产接口

| 接口 | 数据源迁移 |
|------|-----------|
| `GET /api/v1/assets` | `HistoryRepo.GetRecordsPaginated()` → `AssetRepo.List()` |
| `POST /api/v1/assets/favorite` | `HistoryRepo.ToggleFavorite()` → `AssetRepo.ToggleFavorite()` |
| `POST /api/v1/assets/batch-download` | `HistoryRepo.GetRecordsByIDs()` → `AssetRepo.GetByIDs()` |
| `DELETE /api/v1/assets` | `HistoryRepo.DeleteRecords()` → `AssetRepo.Delete()` |

接口 URL 和请求/响应格式不变，前端无需修改现有调用代码。

## 后端实现

### 新增 AssetRepository

**接口** (repository/interfaces.go):

```go
type AssetRepository interface {
    Insert(asset *model.Asset) (int64, error)
    List(page, perPage int, assetType, search string) ([]model.Asset, int, error)
    GetByIDs(ids []int64) ([]model.Asset, error)
    ToggleFavorite(id int64, favorite bool) error
    Delete(ids []int64) error
}
```

**GORM 实现** (repository/gorm/asset.go)：
- 直接操作 `assets` 表
- List 支持按 type 过滤、按 prompt 模糊搜索、分页
- Delete 支持批量删除

### 重构 AssetHandler

- `NewAssetHandler(repo)` 参数从 `HistoryRepository` 改为 `AssetRepository`
- 移除 `SetRepo()` 方法（不再需要动态切换）
- `ListAssets`：改用 AssetRepo.List，字段映射保持与 Assets.vue 兼容
- `ToggleFavorite`：改用 AssetRepo.ToggleFavorite
- `BatchDownload`：改用 AssetRepo.GetByIDs
- `DeleteAssets`：改用 AssetRepo.Delete

### 保存逻辑

```go
func (h *AssetHandler) SaveAsset(c *gin.Context) {
    var req struct {
        ImageURL string `json:"image_url"`
        Prompt   string `json:"prompt"`
        Mode     string `json:"mode"`
    }
    // bind → localPath 解析 → 文件存在性校验 → Asset 写入 → 返回 id
}
```

边界情况：
- 本地文件不存在：返回 400 "图片文件不存在"
- 远程 URL（非本地路径）：local_path 留空，仍可保存
- 重复保存同一图片：每次都创建新记录（不做去重）

### 路由和 main.go

- 新增 route: `assets.POST("", h.SaveAsset)`
- `main.go`：AssetHandler 初始化从 `NewAssetHandler(historyRepo)` 改为 `NewAssetHandler(assetRepo)`
- `db.AutoMigrate` 追加 `&Asset{}`

## 前端实现

### ImageResult.vue

新增 props: `prompt: string`, `mode: string`

按钮组改为：

```
[下载] [保存到作品库] [转存到 GitHub]
```

保存按钮调 `saveAsset()` API，成功后提示"已保存到作品库"。

### assets.ts

新增 API 函数：

```ts
export async function saveAsset(data: { image_url: string; prompt: string; mode: string }): Promise<{ id: number }>
```

### 调用方更新

三个视图补充 prompt/mode prop：

| 视图 | prompt | mode |
|------|--------|------|
| TextToImage.vue | `prompt` | `'text2image'` |
| ImageToImage.vue | `prompt` | `'image2image'` |
| BatchGen.vue | `prompts.join('; ')` | `'batch'` |

Wizard / Storyboard 暂不加入保存功能，后续拓展。

## 不变的部分

- 视频历史记录机制不变（自动入库已有逻辑）
- History 表的现有行为和接口不变
- 作品库的列表/分页/搜索/收藏/批量下载/删除功能体验不变
- 设置页面的存储配置不变

## 测试策略

- AssetRepository 单元测试：Insert → List → ToggleFavorite → GetByIDs → Delete
- 手动验证：生成图片 → 保存 → 作品库可见 → 收藏/取消 → 删除
