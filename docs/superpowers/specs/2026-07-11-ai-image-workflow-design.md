# AI 图片工作流优化设计

> 面向职场工作人员的高频 AI 生图 + 迭代工作流设计
>
> 适用场景：市场营销/社媒运营 + 内容创作/自媒体

## 目标

以作品库为中心枢纽，构建高效的 AI 生图迭代闭环：

1. **出图速度与探索效率** — 快速试错、批量出图、低摩擦
2. **精确控制与迭代质量** — 从作品库选图精修，参数完整继承
3. **素材管理与复用** — 集合组织、Prompt 模板、风格预设
4. **团队协作与品牌一致** — 模板/预设可导出共享

## 架构概览

作品库作为工作流中枢，各生成页作为功能入口，不改变现有导航结构。

```
┌─────────────────────────────────────────────┐
│            作品库（Assets 页面增强）              │
│                                               │
│  网格视图 │ 集合筛选 │ 多选操作 │ 对比模式        │
│                                               │
│  每个图片卡片: [精修] [收藏] [下载] [详情]        │
└──────────┬──────────────────────────────────┘
           │ 点击「精修」
           ▼
┌─────────────────────────────────────────────┐
│      ImageToImage（自动填入选中图的参数）         │
│                                               │
│  输入图片: [已选图片]                            │
│  Prompt: [原始 prompt，可修改]                  │
│  参数: [原始参数，可修改]                        │
│  [开始精修] → 结果自动入库 → 继续迭代            │
└─────────────────────────────────────────────┘
```

## 模块设计

### 模块 A：作品库功能增强

#### A1 集合（Collections）
- 标签式分类，一张图可属于多个集合
- CRUD：创建、重命名、删除集合
- 多选操作 → 「移至集合」
- 视图筛选：按集合过滤网格
- 存储：新增 `collections` 表 + `asset_collections` 关联表（SQLite GORM）
- API: `GET/POST/PUT/DELETE /api/v1/collections` + `POST /api/v1/collections/:id/assets`

#### A2 一键精修入口
- 作品库每张图片卡片上已有「精修」按钮
- 点击后通过 redo store 跳转到 ImageToImage，自动填入：
  - `image_url`（该图的本地/远程 URL）
  - `prompt`（生成时记录的原始 prompt）
  - `negative_prompt`、`size`、`strength`
  - `inputMode: 'url'`
- 复用现有的 `useRedoStore().setRedoData()` 机制

#### A3 批量多选操作
- 勾选多张图 → 底部操作栏
  - 「批量精修」→ 批量图生图（使用同一组 prompt/参数）
  - 「批量下载」
  - 「移至集合」
  - 「批量删除」
- API 复用现有的批量端点

#### A4 图片对比
- 选中 2 张 → 「对比」按钮
- 并排对比模式 + 叠加滑动对比
- 纯前端实现，无后端改动

### 模块 B：生成自动入库

#### B1 结果自动保存
- 所有生成页（TextToImage/ImageToImage/Batch）的结果在展示时自动调用 save API
- 保存内容：图片 URL + 完整 prompt + 参数 + 生成模式
- **不取消结果区的展示**，结果仍在页面上可见
- 可选配置：设置中增加「自动保存到作品库」开关（默认开启）

#### B2 Prompt 历史
- 作品库中每张图记录完整生成参数
- 从作品库精修时自动带入全部参数
- Prompt 输入框旁增加历史侧栏（最近使用的 prompts 列表）
- 存储：复用现有的 history 表（已存 prompt/参数，确保入库时写入完整）

#### B3 两条迭代路径并存
- 作品库 → 点击精修 → 跳到生成页（预填参数）
- 生成页 → 「从作品库选择」按钮 → 选图 → 精修
- 两者都保留，覆盖不同使用习惯

### 模块 C：Prompt 模板与风格预设

#### C1 Prompt 模板库
- 独立页面或作品库侧栏标签
- 每条模板：名称 + 分类 + prompt 正文 + 参数预设（size/strength/model）
- 分类：内置 `人物 / 产品 / 背景 / 封面 / 海报 / 社媒 / 自定义`
- 在生成页 prompt 框旁增加「从模板」按钮 → 弹出选择器 → 选后自动填入
- 可从历史生成记录中「另存为模板」
- 存储：新增 `prompt_templates` 表

#### C2 风格预设
- 聚焦参数组合：model + size + strength + negative_prompt
- 生成页参数区增加「风格预设」下拉选择器
- 预设示例："小红书封面" = `3:4, strength: 0.8`、"公众号配图" = `16:9, strength: 0.7`
- 存储：可共用 `prompt_templates` 表（type 字段区分 template/preset），或独立 `style_presets` 表

#### C3 模板导出/共享
- 导出为 JSON 文件
- 导入 JSON 文件
- 无认证环境下通过文件共享实现团队复用

## 数据存储

### 新增模型

```go
// Collection 集合
type Collection struct {
    ID        uint           `gorm:"primaryKey"`
    Name      string         `gorm:"not null"`
    CreatedAt time.Time
    UpdatedAt time.Time
    Assets    []Asset        `gorm:"many2many:asset_collections;"`
}

// AssetCollection 关联表
type AssetCollection struct {
    AssetID      uint `gorm:"primaryKey"`
    CollectionID uint `gorm:"primaryKey"`
}

// PromptTemplate 模板/预设
type PromptTemplate struct {
    ID             uint      `gorm:"primaryKey"`
    Name           string    `gorm:"not null"`
    Type           string    `gorm:"default:template"` // template | preset
    Category       string    // 分类：人物/产品/背景/封面/海报/社媒/自定义
    Prompt         string    // prompt 正文（template 专用）
    NegativePrompt string
    Size           string
    Strength       float64
    Model          string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

### 现有模型改动

- `History` / `Asset` 表确保入库时 `prompt`、`negative_prompt`、`size`、`strength`、`model` 字段完整写入（当前部分字段可能为空）

## 前端组件改动

| 组件 | 改动 |
|------|------|
| `Assets.vue` | 新增集合筛选、多选操作栏、对比模式、「精修」按钮接入 redo store |
| `Collections.vue`（新） | 集合 CRUD 页面/弹窗 |
| `ImageResult.vue` | 自动保存逻辑（生成展示时触发） |
| `ImageToImage.vue` | 从作品库精修时自动填入参数；新增「从模板」按钮 |
| `TextToImage.vue` | 自动保存；prompt 历史侧栏；模板选择器 |
| `PromptTemplate.vue`（新） | 模板管理页面 |
| `ImageCompare.vue`（新） | 并排/叠加对比组件 |

## 后端 API 新增

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/v1/collections` | 列表 |
| POST | `/api/v1/collections` | 创建 |
| PUT | `/api/v1/collections/:id` | 更新 |
| DELETE | `/api/v1/collections/:id` | 删除 |
| POST | `/api/v1/collections/:id/assets` | 添加图片到集合 |
| DELETE | `/api/v1/collections/:id/assets` | 从集合移除图片 |
| GET | `/api/v1/templates` | 模板列表 |
| POST | `/api/v1/templates` | 创建模板 |
| PUT | `/api/v1/templates/:id` | 更新模板 |
| DELETE | `/api/v1/templates/:id` | 删除模板 |
| POST | `/api/v1/templates/export` | 导出为 JSON |
| POST | `/api/v1/templates/import` | 从 JSON 导入 |
| POST | `/api/v1/history/:id/save-template` | 从历史记录保存为模板 |

## 实施顺序

1. **模块 B（自动入库）** — 工作量最小，收益立即可见，且为后续依赖
2. **模块 A1-A3（集合 + 精修 + 批量操作）** — 作品库核心增强
3. **模块 C（模板/预设）** — 独立可用的生产力功能
4. **模块 A4（图片对比）** — 完善体验的锦上添花
