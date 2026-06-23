# Agnes Creator Studio B/S 架构改造 — 实施计划

## 总体策略

分 4 个阶段，每阶段独立可验证。建议按顺序执行：

```
Phase 1: Go 后端基础 (API 客户端 + 图片功能)   ← 先做，独立可测
Phase 2: Go 后端视频 (异步 + SSE)              ← 依赖 Phase 1
Phase 3: Vue 前端 (脚手架 + 图片页面)            ← 可与 Phase 1 并行
Phase 4: Vue 前端 (视频页面 + History + 集成)    ← 依赖 Phase 2+3
```

## Phase 1: Go 后端 — Core + 图片功能

### 文件清单

| 文件 | 职责 |
|------|------|
| `backend/go.mod` | Go 模块定义，依赖 Gin + godotenv |
| `backend/cmd/server/main.go` | 入口：加载 env、配置、启动 Gin |
| `backend/internal/config/config.go` | `.config.json` 读写 + 环境变量 fallback |
| `backend/internal/model/types.go` | 所有共享 struct 定义 |
| `backend/internal/service/agnes.go` | Agnes API 客户端（完整移植 api_client.py） |
| `backend/internal/handler/image.go` | 文生图、图生图、批量 handler |
| `backend/internal/handler/history.go` | 历史记录 GET/DELETE handler |
| `backend/internal/handler/config_handler.go` | 配置 GET/PUT handler |
| `backend/internal/middleware/cors.go` | CORS 中间件 |

### 步骤

1. **Scaffold 项目**
   - `go mod init github.com/you-want/agnes-image-tool/backend`
   - 安装 `gin`, `cors` 等依赖
   - 创建目录结构
   
2. **实现 model/types.go**
   - `Config`, `HistoryRecord`, `ImageRequest`, `ImageResponse`, `BatchRequest`, 所有 video 相关 struct
   
3. **实现 config/config.go**
   - `LoadConfig()`, `SaveConfig()`, `GetConfig()`, `UpdateConfig()`
   - 环境变量 `AGNES_API_KEY` 覆盖

4. **实现 service/agnes.go** — 核心移植
   - `AgnesClient` struct 定义
   - `NewAgnesClient(apiKey, baseURL string)`
   - `TextToImage(prompt, size, n, negativePrompt) → ([]string, error)`
   - `ImageToImage(imagePath, prompt, size, strength, negativePrompt) → ([]string, error)`
   - `DownloadAndSave(url, prefix) → (string, error)` (保存到 outputs/)

5. **实现 handler/image.go**
   - `POST /api/v1/images/text-to-image` — 调用 `TextToImage` → 下载 → 返回本地路径
   - `POST /api/v1/images/image-to-image` — multipart 接收文件 → 保存临时 → 调用 `ImageToImage` → 下载 → 返回
   - `POST /api/v1/images/batch` — 遍历调用 `TextToImage`

6. **实现 handler/history.go**
   - `GET /api/v1/history` — 读取 history.json
   - `DELETE /api/v1/history` — 清空

7. **实现 handler/config_handler.go**
   - `GET /api/v1/config`
   - `PUT /api/v1/config`

8. **实现 middleware/cors.go**
   - 允许开发阶段 localhost:5173

9. **实现 main.go**
   - 加载配置 → 注册路由 → 启动 `:8080`

### 验证

```bash
export AGNES_API_KEY="xxx"
cd backend && go run ./cmd/server
curl http://localhost:8080/api/v1/config
curl -X POST http://localhost:8080/api/v1/images/text-to-image \
  -H "Content-Type: application/json" \
  -d '{"prompt":"a cat","size":"1024x1024","n":1}'
```

## Phase 2: Go 后端 — 视频功能 + SSE

### 文件

| 文件 | 修改/新增 |
|------|-----------|
| `backend/internal/service/agnes.go` | 新增 `TextToVideo`, `ImageToVideo`, `MultiImageVideo`, `CheckVideoStatus` |
| `backend/internal/service/video_manager.go` | 新增：任务管理器 + SSE |
| `backend/internal/handler/video.go` | 新增：所有视频 handler + SSE handler |

### 步骤

1. **agnes.go 新增方法**
   - `TextToVideo(...) → (taskID string, error)` — 提交任务
   - `ImageToVideo(imagePath, ...) → (taskID, error)` — 通过 base64 或 URL
   - `MultiImageVideo(urls, mode, ...) → (taskID, error)`
   - `CheckVideoStatus(videoID) → (*VideoStatus, error)`

2. **实现 video_manager.go**
   ```go
   type TaskManager struct {
       mu    sync.RWMutex
       tasks map[string]*VideoTask
   }
   type VideoTask struct {
       ID          string
       Status      string    // queued / in_progress / completed / failed
       Progress    int
       ResultURL   string
       Error       string
       Subscribers map[string]chan VideoEvent
   }
   ```
   - `CreateTask(...) → *VideoTask` — 提交到 Agnes API，存入 tasks map
   - `StartPolling(taskID)` — goroutine，每 5s 轮询，通过 channel 通知
   - `Subscribe(taskID) → chan VideoEvent` — SSE 注册
   - `Unsubscribe(taskID, subID)`
   - 限制最大 10 个并发 goroutine

3. **实现 handler/video.go**
   - `POST /api/v1/videos/text-to-video`
   - `POST /api/v1/videos/image-to-video`
   - `POST /api/v1/videos/multi-image`
   - `GET /api/v1/videos/:taskId` — 查询状态
   - `GET /api/v1/videos/stream/:taskId` — SSE

4. **SSE 实现**
   - Gin 的 `c.Stream()` 方式
   - 写入 `event: progress\ndata: ...\n\n`
   - 连接断开时自动清理 subscriber

### 验证

```bash
# 创建视频任务
curl -X POST http://localhost:8080/api/v1/videos/text-to-video \
  -H "Content-Type: application/json" \
  -d '{"prompt":"cat walking","duration":5,"aspectRatio":"16:9","frameRate":24}'
# → {"taskId":"task_xxx"}

# SSE 测试
curl -N http://localhost:8080/api/v1/videos/stream/task_xxx
```

## Phase 3: Vue 前端 — 脚手架 + 图片页面

### 步骤

1. **Scaffold 项目**
   ```bash
   npm create vite@latest frontend -- --template vue-ts
   cd frontend
   npm install element-plus @element-plus/icons-vue pinia axios vue-router
   ```

2. **vite.config.ts**
   - 代理 `/api` → `http://localhost:8080`
   - 代理 `/outputs` → `http://localhost:8080`

3. **TypeScript 类型 (`types/index.ts`)**
   - `Config`, `ImageRequest`, `ImageResponse`, `VideoTask`, `VideoEvent`, `HistoryRecord` 等

4. **API 调用层**
   - `api/client.ts` — axios 实例，baseURL 空（Vite 代理处理）
   - `api/image.ts` — `textToImage()`, `imageToImage()`, `batchGenerate()`
   - `api/history.ts` — `getHistory()`, `clearHistory()`
   - `api/video.ts` — `createTextToVideo()`, `createImageToVideo()`, `createMultiImageVideo()`, `getTaskStatus()`

5. **Pinia Store (`stores/config.ts`)**
   - `apiKey`, `baseUrl`, `model`
   - `loadConfig()`, `saveConfig()` — 调用后端 API

6. **SSE 工具 (`utils/sse.ts`)**
   - `connectSSE(taskId, handlers)` — EventSource 封装
   - 自动重连逻辑
   - 事件分发

7. **组件实现**
   - `App.vue` — 页面框架 + ApiConfig + Tabs
   - `components/ApiConfig.vue` — API Key/BaseURL/Model 配置折叠面板
   - `components/ImageResult.vue` — 图片 Gallery 展示 + 下载
   - `views/TextToImage.vue` — 文生图页面
   - `views/ImageToImage.vue` — 图生图页面（文件上传）
   - `views/BatchGen.vue` — 批量生成页面

### 验证

```bash
cd frontend && npm run dev
# 浏览器打开 http://localhost:5173
# 配置 API Key → 文生图 → 应能正常生成
```

## Phase 4: Vue 前端 — 视频页面 + History + 集成

### 步骤

1. **views/TextToVideo.vue** — 文生视频
2. **views/ImageToVideo.vue** — 图生视频（URL/上传）
3. **views/MultiImageVideo.vue** — 多图视频（URL 列表）
4. **views/History.vue** — 历史记录 Gallery + 详情
5. **components/VideoParams.vue** — 视频参数选择器（复用组件）
6. **components/VideoProgress.vue** — SSE 进度条 + 视频播放器

### 集成测试

```bash
# 两个终端
cd backend && go run ./cmd/server
cd frontend && npm run dev
# 完整测试所有功能
```

## 里程碑检查

- [ ] Phase 1: Go 后端可响应图片 API 调用
- [ ] Phase 2: Go 视频 API + SSE 正常工作
- [ ] Phase 3: 前端图片功能完整可用
- [ ] Phase 4: 前端视频 + 历史记录完整可用
- [ ] 配置管理前后端贯通
- [ ] 文件下载功能正常
- [ ] 错误处理覆盖主要边界情况
