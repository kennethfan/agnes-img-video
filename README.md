# Agnes Creator Studio

基于 [Agnes AI](https://agnes-ai.com) 的 AI 创作工具，支持文生图、图生图、批量生成、文生视频、图生视频、多图视频、脚本生成、点子库、故事板、项目管理、漫画生成、提示词模板、作品库等功能。

MIT License · Built with Go + Vue 3

---

## 功能

| 功能 | 说明 |
|------|------|
| **文生图** | 根据提示词生成高质量图片 |
| **图生图** | 基于参考图 + 提示词进行风格转换、局部优化 |
| **批量生成** | 一次提交多个提示词，并行生成图片 |
| **文生视频** | 文本提示词生成视频，SSE 实时推送进度 |
| **图生视频** | 静态图片动画化为动态视频 |
| **多图视频** | 多张参考图 + 关键帧模式生成视频 |
| **脚本生成** | AI 生成视频脚本（支持中/英文） |
| **点子库** | 创意点子管理 + AI 完善，内置 5 种创作模板 |
| **漫画生成** | AI 生成漫画剧本 + 分镜提示词，支持多步骤创作向导 |
| **故事板** | 分镜项目管理，支持镜头 CRUD + 批量创建 + 拖拽排序 |
| **项目管理** | 创作项目全生命周期管理（4 步闭环：创意→生成→优化→定稿） |
| **项目仪表盘** | 项目文件聚合展示、进度追踪、统计概览 |
| **作品库** | 所有生成作品的统一管理，支持收藏、批量下载、删除 |
| **作品集合** | 自定义作品集合，支持跨项目整理归类 |
| **提示词模板** | 模板 CRUD + 导入/导出 + 从历史记录保存模板 |
| **历史记录** | SQLite 持久化，支持重做到任意生成页面、批量删除 |
| **任务队列** | 统一任务管理，支持查看进度、取消、重试 |
| **访问日志** | 请求日志记录与查询 |
| **数据库管理** | 数据库导出/恢复（JSON 格式） |

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.25 · Gin · SQLite · SSE · GORM |
| 前端 | Vue 3 · TypeScript 6 · Vite 8 · Element Plus · Pinia · Axios · Vue Router |
| AI API | Agnes AI (image/video/chat) |

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 20+ (pnpm)
- Agnes AI API Key

### 获取 Agnes API Key

1. 打开 [Agnes AI Platform](https://platform.agnes-ai.com)，注册或登录账号。
2. 进入 Developer Dashboard，点击 **Create API Key** 生成密钥。
3. 将密钥保存到单独文件（如 `~/.agnes/api-key`），建议 `chmod 600`。

> **免费政策**：Agnes AI 自 2026 年 6 月起，核心模型（文本/图像/视频）API 永久免费开放，注册即可使用。

### 启动

```bash
# 1. 后端
cd backend
cp .env.example .env      # 编辑 API_KEY_PATH 指向 key 文件
go run ./cmd/server        # → http://localhost:8080

# 2. 前端
cd frontend
pnpm install
pnpm dev                   # → http://localhost:5173
```

浏览器打开 `http://localhost:5173` 即可使用。

## 配置

所有配置通过环境变量读取，无需配置文件。在 `backend/.env` 中设置（参考 `.env.example`）：

| 环境变量 | 必需 | 说明 |
|----------|------|------|
| `API_KEY_PATH` | ✅ | API Key 文件路径（如 `~/.agnes/api-key`） |
| `BASE_URL` | 否 | API 地址，默认 `https://apihub.agnes-ai.com/v1` |
| `IMAGE_MODEL` | 否 | 图片模型，默认 `agnes-image-2.1-flash` |
| `VIDEO_MODEL` | 否 | 视频模型，默认 `agnes-video-v2.0` |
| `CHAT_MODEL` | 否 | 对话模型，默认 `agnes-2.0-flash` |
| `GITHUB_TOKEN` | 否 | GitHub Token，启用远程文件存储 |
| `GITHUB_REPO` | 否 | 存储仓库，格式 `owner/repo` |
| `GITHUB_BRANCH` | 否 | 分支名，默认 `master` |
| `DB_DRIVER` | 否 | 数据库驱动，默认 `sqlite` |
| `DB_DSN` | 否 | 数据库 DSN，默认 `history.db` |
| `PORT` | 否 | 后端端口，默认 `8080` |

> API Key **仅通过 `API_KEY_PATH` 指向的文件读取**，不支持直接写入 `.env`。key 文件可设 `chmod 600` 确保安全。

## API

所有接口以 `/api/v1` 为前缀。

### 图片

```
POST /images/text-to-image     文生图
POST /images/image-to-image    图生图（multipart 或 JSON）
POST /images/batch             批量生成
```

### 视频

```
POST /videos/text-to-video     文生视频
POST /videos/image-to-video    图生视频
POST /videos/multi-image       多图视频
POST /videos/generate-script   脚本生成
GET  /videos/:taskId           查询任务状态
GET  /videos/stream/:taskId    SSE 实时进度
```

### 点子 & 漫画

```
POST /ideas/expand             点子库 AI 完善
POST /comic/generate-prompts   生成漫画分镜提示词
POST /comic/generate-storyline 生成漫画剧本（故事线、角色、画风）
```

### 历史记录

```
GET    /history                列表
DELETE /history                清空
DELETE /history/:id            删除单条
POST   /history/delete         批量删除
POST   /history/:id/save-template  保存为提示词模板
```

### 作品库

```
GET    /assets                 列表
POST   /assets                 保存资产
POST   /assets/favorite        切换收藏
POST   /assets/batch-download  批量下载
POST   /assets/:id/transfer    转移到 outputs/
DELETE /assets                 删除
```

### 故事板

```
GET    /storyboard/projects                   项目列表
POST   /storyboard/projects                   创建项目
GET    /storyboard/projects/:id               项目详情
PUT    /storyboard/projects/:id               更新项目
DELETE /storyboard/projects/:id               删除项目
POST   /storyboard/projects/:id/duplicate     复制项目
POST   /storyboard/projects/:id/shots         创建镜头
POST   /storyboard/projects/:id/shots/batch   批量创建镜头
PUT    /storyboard/projects/:id/shots/reorder 镜头排序
PUT    /storyboard/shots/:id                  更新镜头
DELETE /storyboard/shots/:id                  删除镜头
POST   /storyboard/projects/:id/generate      生成镜头图片
```

### 项目管理

```
GET    /projects                 列表
POST   /projects                 创建（支持 project/comic 类型）
GET    /projects/:id             详情
PUT    /projects/:id             更新
DELETE /projects/:id             删除
POST   /projects/:id/duplicate   复制
POST   /projects/:id/ai-recommend AI 推荐
POST   /projects/:id/steps       添加步骤
PUT    /steps/:stepId            更新步骤
DELETE /steps/:stepId            删除步骤
POST   /projects/:id/ideate-brief 生成创意简报
GET    /projects/:id/files       项目文件聚合
GET    /projects/:id/stats       项目统计
PUT    /projects/:id/step-progress 更新步骤进度
```

### 集合 & 模板

```
GET    /collections              集合列表
POST   /collections              创建集合
PUT    /collections/:id          更新集合
DELETE /collections/:id          删除集合
POST   /collections/:id/assets   添加资产到集合
DELETE /collections/:id/assets   从集合移除资产
GET    /templates                模板列表
POST   /templates                创建模板
PUT    /templates/:id            更新模板
DELETE /templates/:id            删除模板
POST   /templates/export         导出模板
POST   /templates/import         导入模板
```

### 任务 & 系统

```
GET    /tasks                    任务列表
GET    /tasks/:id                任务详情
GET    /tasks/:id/stream         SSE 进度
POST   /tasks/:id/cancel         取消任务
POST   /tasks/:id/retry          重试任务
GET    /config                   获取配置
PUT    /config                   更新配置
GET    /outputs/*filepath        静态文件服务
```

### 设置 & 工具

```
GET  /settings                  获取设置
PUT  /settings                  更新设置
GET  /access-logs               访问日志
DELETE /access-logs/:id         删除日志
DELETE /access-logs/clear       清空日志
GET  /db/export                 导出数据库
POST /db/restore                恢复数据库
POST /github/upload             GitHub 文件上传
GET  /github/fetch?path=xxx     GitHub 文件读取
```

## 项目结构

```
agnes-image-tool/
├── backend/                     # Go 后端
│   ├── cmd/server/main.go       # 入口
│   ├── internal/
│   │   ├── config/              # 环境变量加载
│   │   ├── handler/             # HTTP handlers（14 个 handler 文件）
│   │   ├── model/               # 共享类型定义
│   │   ├── service/             # 业务逻辑（AgnesClient、TaskQueue、GitHub 存储）
│   │   ├── repository/gorm/     # GORM 持久化（8 个 Repository）
│   │   └── middleware/          # CORS
│   ├── makefile
│   └── .env.example
├── frontend/                    # Vue 3 前端
│   ├── src/
│   │   ├── views/               # 20+ 页面组件（含 comic/novel/image 向导子目录）
│   │   ├── components/          # 复用组件（NavSidebar、ImageResult、ShotCard 等）
│   │   ├── api/                 # 17 个 API 封装模块
│   │   ├── stores/              # Pinia 状态管理（redo、wizard）
│   │   ├── types/               # TypeScript 类型定义
│   │   └── utils/               # SSE 等工具函数
│   ├── vite.config.ts
│   └── package.json
├── docs/                        # 设计文档
├── image-api.md                 # Agnes Image API 参考
├── video-api.md                 # Agnes Video API 参考
└── LICENSE                      # MIT
```

## 构建

### 后端

```bash
cd backend
make build    # 编译到 bin/server（-s -w ldflags）
make test     # go test ./...
make run      # 编译并运行
make clean    # rm -rf bin/
```

### 前端

```bash
cd frontend
pnpm build    # vue-tsc 类型检查 + vite build
pnpm dev      # 开发服务器，代理 /api + /outputs → :8080
```

## 已知问题

- **pnpm + lightningcss**: macOS 上 `lightningcss-darwin-x64` 可能未自动下载。构建失败时执行：
  ```bash
  curl -sL https://registry.npmjs.org/lightningcss-darwin-x64/-/lightningcss-darwin-x64-1.32.0.tgz | tar xz -C node_modules/.pnpm/lightningcss-darwin-x64@1.32.0/node_modules/lightningcss-darwin-x64/ --strip-components=1
  ```
- **视频帧数约束**: 必须满足 `8n + 1`（1080p 最大 169 帧，720p 最大 409 帧，480p 最大 961 帧）
- **视频状态查询**: 状态接口会去掉 baseURL 中的 `/v1`，查询 `{baseDomain}/agnesapi?video_id={id}`
- **无鉴权**: API 仅用于本地开发，无认证中间件

## 致谢

本项目受 [agnes-image-tool](https://github.com/you-want/agnes-image-tool.git) 的启发，并借鉴了其中的大量设计和代码实现。特此致谢！

## 相关文档

- [B/S 架构设计](docs/design-bs-architecture.md)
- [实施计划](docs/implementation-plan.md)
- [Agnes Image API](image-api.md)
- [Agnes Video API](video-api.md)
