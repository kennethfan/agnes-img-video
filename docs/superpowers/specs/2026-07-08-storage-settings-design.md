# 存储设置页面设计

## 背景

「转存」按钮目前硬编码为上传到 GitHub。需要将存储目标配置化，让用户可选择转存到本地目录或 GitHub，并可自定义目录路径。

## 配置项

所有配置存入 SQLite `history.db` 的 `settings` 表，单行记录（`id=1`），启动时自动建表并写入默认值。

| 字段 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `storage_target` | string | `local` | 转存目标：`local` / `github` |
| `local_image_dir` | string | `images` | 本地图片子目录（相对 `outputs/`） |
| `local_video_dir` | string | `videos` | 本地视频子目录（相对 `outputs/`） |
| `github_image_path` | string | `outputs/images` | GitHub 图片路径前缀 |
| `github_video_path` | string | `outputs/videos` | GitHub 视频路径前缀 |

> GitHub Token/Repo/Branch 等敏感信息仍在 `.config.json` / 环境变量中，不入库。

## 架构

### 后端

**repository/settings.go** — 新增文件
- `InitSettingsTable(db)` — 建表（`CREATE TABLE IF NOT EXISTS settings`）+ 写入默认行
- `GetSettings() (*model.Settings, error)` — 读取配置，找不到则返回默认值
- `UpdateSettings(s *model.Settings) error` — 更新

**model/types.go** — 新增 `Settings` 结构体

**handler/settings.go** — 新增文件
- `GET /api/v1/settings` → 返回当前配置
- `PUT /api/v1/settings` → 更新配置

**handler/github_handler.go** — 改造 `UploadToGitHub`
- 根据 `settings.storage_target` 分流：
  - `local`: 下载到 `backend/outputs/{local_image_dir|local_video_dir}/`，保存文件，返回 `/outputs/...` 本地 URL
  - `github`: 现有流程不变（下载→上传 GitHub→返回 GitHub URL）

**cmd/server/main.go** — 注册 `/api/v1/settings` 路由，初始化 settings 表

### 前端

**api/settings.ts** — 新增
- `getSettings()` → GET `/api/v1/settings`
- `updateSettings(data)` → PUT `/api/v1/settings`

**views/Settings.vue** — 新增
- 表单：`storage_target` 单选（本地 / GitHub）+ 各自目录路径输入框
- 保存按钮调用 `updateSettings`
- 保存后 `ElMessage.success`

**components/NavSidebar.vue** — 系统组下新增「存储设置」菜单（`id: 'settings'`）

**App.vue** — 注册 `settings` 页面的条件渲染

## 转存流程

1. 前端点击「转存」→ `POST /api/v1/upload-to-github { url, filename }`
2. 后端 `UploadToGitHub` 读取 `settings`
3. 如果 `storage_target == "local"`：
   - 下载文件到 `backend/outputs/{dir}/`（目录不存在则自动创建）
   - 返回 `{"local_url": "/outputs/{dir}/{filename}"}`
4. 如果 `storage_target == "github"`（且 GitHub 已配置）：
   - 现有流程不变
5. 如果 `storage_target == "github"` 但未配置 GitHub：
   - 返回 400 "未配置 GitHub 存储"
6. 前端根据返回的 URL 类型显示提示（本地路径可点击预览，GitHub URL 可复制）

## 文件名规则

- 本地存储：`{timestamp}_{basename}.{ext}`，如 `20260708_143021_sdxyz.png`
- GitHub 存储：保持现有规则（`github_upload_{timestamp}{ext}`）

## 目录路径规则

- `local_image_dir` / `local_video_dir` 是相对路径，基准目录为 `backend/outputs/`
- 支持多级子目录，如 `images/portrait`、`videos/shorts`
- 前后端统一：前端通过 `/outputs/{dir}/{filename}` 访问
- 目录不存在自动 `os.MkdirAll`
