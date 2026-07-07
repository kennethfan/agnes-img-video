# Agnes Creator Studio

基于 [Agnes AI](https://agnes-ai.com) 的 AI 创作工具，支持文生图、图生图、批量生成、文生视频、图生视频、多图视频、脚本生成、点子库等功能。

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
| **历史记录** | SQLite 持久化，支持重做到任意生成页面 |

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.25 · Gin · SQLite · SSE |
| 前端 | Vue 3 · TypeScript 6 · Vite 8 · Element Plus · Pinia · Axios |
| AI API | Agnes AI (image/video/chat) |

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 20+ (pnpm)
- Agnes AI API Key

### 启动

```bash
# 1. 后端
cd backend
cp .env.example .env      # 编辑 AGNES_API_KEY
go run ./cmd/server        # → http://localhost:8080

# 2. 前端
cd frontend
pnpm install
pnpm dev                   # → http://localhost:5173
```

浏览器打开 `http://localhost:5173` 即可使用。

## 配置

### 环境变量

在 `backend/.env` 中配置（参考 `.env.example`）：

| 变量 | 必填 | 说明 |
|------|------|------|
| `AGNES_API_KEY` | ✅ | Agnes AI API 密钥 |
| `PORT` | 否 | 后端端口，默认 `8080` |
| `AGNES_BASE_URL` | 否 | API 地址，默认 `https://apihub.agnes-ai.com/v1` |
| `GITHUB_TOKEN` | 否 | GitHub Token，启用远程文件存储 |
| `GITHUB_REPO` | 否 | 存储仓库，格式 `owner/repo` |
| `GITHUB_BRANCH` | 否 | 分支名，默认 `main` |
| `IMAGE_MODEL` | 否 | 图片模型，默认 `agnes-image-2.1-flash` |
| `VIDEO_MODEL` | 否 | 视频模型，默认 `agnes-video-v2.0` |
| `CHAT_MODEL` | 否 | 聊天模型，默认 `agnes-2.0-flash` |

### 配置文件

`backend/.config.json`（gitignored）持久化配置，环境变量优先覆盖。

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

### 其他

```
POST /ideas/expand             点子库 AI 完善
GET  /config                   获取配置
PUT  /config                   更新配置
GET  /history                  历史记录
DELETE /history                清空历史
DELETE /history/:id            删除单条
POST /history/delete           批量删除
```

## 项目结构

```
agnes-image-tool/
├── backend/                     # Go 后端
│   ├── cmd/server/main.go       # 入口
│   ├── internal/
│   │   ├── config/              # 配置管理
│   │   ├── handler/             # HTTP handlers
│   │   ├── model/               # 共享类型
│   │   ├── service/             # 业务逻辑（Agnes API 客户端、视频任务管理）
│   │   ├── repository/          # SQLite 存储
│   │   └── middleware/          # CORS
│   ├── makefile
│   └── .env.example
├── frontend/                    # Vue 3 前端
│   ├── src/
│   │   ├── views/               # 9 个页面组件
│   │   ├── components/          # 复用组件
│   │   ├── api/                 # API 封装
│   │   ├── stores/              # Pinia 状态管理
│   │   ├── types/               # TypeScript 类型
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
make build    # 编译到 bin/server
make test     # 运行测试
make run      # 编译并运行
make clean    # 清理
```

### 前端

```bash
cd frontend
pnpm build    # vue-tsc 类型检查 + vite build
pnpm dev      # 开发服务器，代理 /api → :8080
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
