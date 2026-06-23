# Agnes Creator Studio — B/S 架构改造设计

## 概述

将当前单模块 Gradio (Python) 应用改造为 **Browser/Server 架构**：
- **后端**: Go (Gin) — REST API + SSE 实时推送
- **前端**: Vue 3 + Vite + Element Plus + TypeScript — SPA
- **部署**: 前后端分离，无 Docker 依赖，简化开发流程

## 技术选型

| 层 | 技术 | 理由 |
|---|---|---|
| 前端框架 | Vue 3 + Vite | 用户偏好，TypeScript 原生支持 |
| UI 组件库 | Element Plus | 用户已有使用经验 |
| 后端框架 | Go + Gin | 高性能，单 binary 部署 |
| 状态管理 | Pinia | Vue 3 官方推荐 |
| HTTP 客户端 | Axios | 拦截器、类型安全 |
| 异步通信 | SSE (Server-Sent Events) | 视频进度实时推送 |
| 构建工具 | Vite | 快速 HMR，代理后端 API |

## 项目结构

```
agnes-image-tool/
├── backend/                          # Go 后端
│   ├── cmd/server/main.go            # 入口
│   ├── internal/
│   │   ├── config/config.go          # 配置管理（JSON 文件 I/O）
│   │   ├── handler/
│   │   │   ├── image.go              # 文生图、图生图、批量
│   │   │   ├── video.go              # 文/图/多图视频 + SSE
│   │   │   ├── history.go            # 历史记录
│   │   │   └── config_handler.go     # API Key 等配置
│   │   ├── service/
│   │   │   └── agnes.go              # Agnes API 客户端
│   │   ├── model/types.go            # 共享数据类型
│   │   └── middleware/
│   │       └── cors.go               # CORS 中间件
│   ├── go.mod / go.sum
│   └── .env.example
├── frontend/                         # Vue 3 + Vite 前端
│   ├── src/
│   │   ├── api/                      # API 封装
│   │   │   ├── client.ts             # Axios 实例
│   │   │   ├── image.ts              # 图片 API
│   │   │   ├── video.ts              # 视频 API + SSE
│   │   │   └── history.ts            # 历史 API
│   │   ├── views/                    # 页面
│   │   │   ├── TextToImage.vue
│   │   │   ├── ImageToImage.vue
│   │   │   ├── TextToVideo.vue
│   │   │   ├── ImageToVideo.vue
│   │   │   ├── MultiImageVideo.vue
│   │   │   ├── BatchGen.vue
│   │   │   └── History.vue
│   │   ├── components/               # 复用组件
│   │   │   ├── ApiConfig.vue         # API 配置面板
│   │   │   ├── VideoParams.vue       # 视频参数选择器
│   │   │   ├── ImageResult.vue       # 图片结果展示
│   │   │   └── VideoProgress.vue     # 视频生成进度(SSE)
│   │   ├── stores/
│   │   │   └── config.ts             # 全局配置 store
│   │   ├── types/index.ts
│   │   ├── utils/sse.ts              # SSE 连接工具
│   │   ├── App.vue
│   │   └── main.ts
│   ├── package.json
│   ├── vite.config.ts
│   └── index.html
├── outputs/                          # 生成的图片/视频
├── .config.json                      # 持久化配置
├── history.json                      # 历史记录
└── .env                              # AGNES_API_KEY 环境变量
```

## API 设计

所有 API 以 `/api/v1` 为前缀。

### 配置管理

| 方法 | 路径 | 请求 | 响应 |
|------|------|------|------|
| GET | `/api/v1/config` | — | `{api_key, base_url, model}` |
| PUT | `/api/v1/config` | `{api_key, base_url, model}` | `{ok}` |

### 图片生成

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/images/text-to-image` | `{prompt, size, n, negative_prompt}` → `{images: string[]}` |
| POST | `/api/v1/images/image-to-image` | multipart: `image`(file) + `prompt` + `size` + `strength` → `{images: string[]}` |
| POST | `/api/v1/images/batch` | `{prompts, size}` → `{images: string[]}` |

同步请求。后端下载到 `outputs/` 后返回本地可访问 URL。

### 视频生成（异步 + SSE）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/videos/text-to-video` | `{prompt, duration, ...}` → `{taskId}` |
| POST | `/api/v1/videos/image-to-video` | multipart: `image` + `prompt` → `{taskId}` |
| POST | `/api/v1/videos/multi-image` | `{prompt, imageUrls, mode}` → `{taskId}` |
| GET | `/api/v1/videos/:taskId` | 查询状态 → `{status, progress, url}` |
| GET | `/api/v1/videos/stream/:taskId` | SSE 实时推送进度 |

SSE 事件：
```
event: progress  data: {"progress": 45, "status": "in_progress"}
event: complete  data: {"url": "/outputs/video_xxx.mp4", "seconds": "10.0"}
event: error     data: {"error": "生成失败: ..."}
```

### 历史记录

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/history` | 获取历史列表 |
| DELETE | `/api/v1/history` | 清空历史 |

### 输出文件

后端在 `/outputs/*` 路径上提供静态文件服务（开发阶段由 Go serve，生产可交由 Nginx/任何静态服务器）。

## Go 后端核心设计

### 入口 (`cmd/server/main.go`)

```
加载配置 → 创建 AgnesClient → 注册路由 → 启动 Gin
```

路由：
```
/api/v1/config          GET, PUT
/api/v1/images/*        POST
/api/v1/videos/*        POST, GET
/api/v1/videos/stream/* GET (SSE)
/api/v1/history         GET, DELETE
/outputs/*              Static
```

### Service 层 (`service/agnes.go`)

保持与当前 `api_client.py` 一致的接口签名：

- `TextToImage(prompt, size, n, negativePrompt) → []string` (URLs)
- `ImageToImage(imagePath, prompt, size, strength, negativePrompt) → []string`
- `TextToVideo(prompt, duration, aspectRatio, frameRate, ...) → taskID`
- `ImageToVideo(imagePath, prompt, ...) → taskID`
- `MultiImageVideo(prompt, imageUrls, mode, ...) → taskID`
- `CheckVideoStatus(videoID) → VideoStatus`
- `DownloadAndSave(url, prefix) → localPath`

### 视频任务管理器

```go
type TaskManager struct {
    mu    sync.RWMutex
    tasks map[string]*VideoTask
}

type VideoTask struct {
    ID          string
    Status      string
    Progress    int
    ResultURL   string
    Error       string
    subscribers map[string]chan VideoEvent  // SSE
}
```

流程：
1. POST 创建任务 → 返回 taskId，启动 goroutine 异步轮询 Agnes API
2. goroutine 每 5s 查询状态，通过 channel 更新任务状态
3. SSE 连接时注册 subscriber，通过 channel 实时推送事件
4. 任务完成后 goroutine 退出，subscriber 收到 complete 事件

### 与当前 Python 版本兼容

- 读写 `.config.json` 格式一致
- 读写 `history.json` 格式一致
- `outputs/` 目录结构一致

## 前端设计

### 页面结构

```
App.vue
├── ApiConfig.vue          # ⚙️ API 配置
├── Tabs
│   ├── TextToImage        # 文生图
│   │   ├── 提示词 + 负面提示词
│   │   ├── 尺寸 + 数量
│   │   └── Gallery
│   ├── ImageToImage       # 图生图
│   │   ├── 图片上传
│   │   ├── 风格描述 + 负面提示词
│   │   ├── 尺寸 + 重绘强度
│   │   └── Gallery
│   ├── TextToVideo        # 文生视频
│   │   ├── VideoParams
│   │   └── VideoProgress (SSE)
│   ├── ImageToVideo       # 图生视频
│   │   ├── 图片上传 / URL 输入
│   │   ├── VideoParams
│   │   └── VideoProgress
│   ├── MultiImageVideo    # 多图视频
│   │   ├── URL 列表 + 模式
│   │   ├── VideoParams
│   │   └── VideoProgress
│   ├── BatchGen           # 批量
│   │   └── Gallery
│   └── History            # 历史
│       └── Gallery + 详情
```

### 关键组件

**VideoParams.vue** — 视频参数选择器（复用）:
- 分辨率档位 (480p/720p/1080p)
- 宽高比 (16:9/9:16/1:1/4:3/3:4)
- 自定义宽高（可选）
- 时长选择 (3/5/8/10/15/18s)
- 帧率 (12/24/30/60)
- 总帧数（可选，覆盖时长）
- 种子 + 推理步数

**VideoProgress.vue** — SSE 进度组件:
- 连接 `/api/v1/videos/stream/:taskId`
- 进度条显示生成进度
- 完成时显示视频播放器
- 错误时显示错误信息

**ImageResult.vue** — 图片结果展示:
- Element Plus Image 组件 + Gallery
- 点击预览大图
- 下载按钮

### 数据流

```
用户操作 → API 调用 (Axios) → Go 后端 → Agnes API
                                    ↓
                             下载到 outputs/
                                    ↓
                             返回本地 URL
                                    ↓
                             前端显示结果
```

视频生成：
```
用户操作 → POST 创建任务 → 返回 taskId
            ↓
        连接 SSE /stream/:taskId
            ↓
        实时接收 progress / complete / error 事件
            ↓
        完成 → 显示视频播放器
```

## 开发流程

```bash
# 终端 1: 后端
cd backend
export AGNES_API_KEY="xxx"
go run ./cmd/server

# 终端 2: 前端
cd frontend
npm install
npm run dev    # Vite 自动代理 /api → localhost:8080
```

Vite 配置开发代理：
```ts
// vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/outputs': 'http://localhost:8080',
    }
  }
})
```

## 边界情况与约束

1. **图片上传大小**: 限制了 10MB，超时返回错误
2. **视频轮询超时**: 30 分钟，超时返回 timeout 错误
3. **并发限制**: 视频生成 goroutine 限制最大 10 个并发任务
4. **视频帧数**: 必须满足 8n+1 约束，由后端强制执行
5. **API Key 安全**: 仅保存在服务端 `.config.json`，不传递给前端
6. **输出文件清理**: 支持按天/按数量自动清理旧文件（可选功能）
