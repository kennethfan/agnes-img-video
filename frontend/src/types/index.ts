export interface Config {
  api_key?: string
  base_url?: string
  model?: string
  github_token?: string
  github_repo?: string
  github_branch?: string
}

export interface TextToImageRequest {
  prompt: string
  size?: string
  n?: number
  negative_prompt?: string
}

export interface BatchRequest {
  prompts: string[]
  size?: string
}

export interface ImageResponse {
  images: string[]
}

export interface VideoCreateRequest {
  prompt: string
  duration?: number
  aspect_ratio?: string
  frame_rate?: number
  negative_prompt?: string
  seed?: number | null
  num_inference_steps?: number | null
  width?: number | null
  height?: number | null
  num_frames?: number | null
  image_urls?: string[]
  mode?: string
}

export interface VideoTaskResponse {
  taskId: number
}

export interface VideoStatus {
  status: string
  progress: number
  url?: string
  error?: string
  seconds?: string
}

export interface VideoEvent {
  progress?: number
  status?: string
  url?: string
  seconds?: string
  error?: string
}

export interface HistoryRecord {
  id: number
  time: string
  mode: string
  prompt: string
  images: string[]
  extra?: Record<string, unknown>
}

export interface DeleteHistoryRequest {
  ids: number[]
  delete_files?: boolean
}

export interface ScriptGenRequest {
  topic: string
  duration?: number
  style?: string
  language?: string
}

export interface ScriptGenResponse {
  script: string
}

export interface SSEHandlers {
  onProgress?: (event: VideoEvent) => void
  onComplete?: (event: VideoEvent) => void
  onError?: (event: VideoEvent) => void
}

export interface AssetItem {
  id: number
  mode: string
  prompt: string
  files: string[]
  thumbnail: string
  type: 'image' | 'video'
  time: string
  favorite: boolean
}

export interface AssetListResponse {
  items: AssetItem[]
  total: number
  page: number
}

export interface AssetFavoriteRequest {
  history_id: number
  favorite: boolean
}

export interface AssetDeleteRequest {
  ids: number[]
  delete_files?: boolean
}

export interface StoryboardProject {
  id: number
  title: string
  script: string
  created_at: string
  updated_at: string
  shot_count: number
}

export interface StoryboardShot {
  id: number
  project_id: number
  sequence: number
  prompt: string
  type: string
  reference_image: string
  status: 'pending' | 'generating' | 'completed'
  result_video: string
  task_id: string
  created_at: string
}

export interface CreateProjectRequest {
  title: string
  script?: string
}

export interface UpdateProjectRequest {
  title?: string
  script?: string
}

export interface CreateShotRequest {
  prompt: string
  type: string
  reference_image?: string
}

export interface UpdateShotRequest {
  prompt?: string
  type?: string
  reference_image?: string
}

// ==================== 异步任务队列 ====================

export interface TaskRecord {
  id: number
  type: string
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled'
  params: string
  result?: string
  progress: number
  error?: string
  retry_count: number
  created_at: string
  updated_at: string
  completed_at?: string
}

export interface TaskCreateResponse {
  taskId: number
}

export interface TaskSSEHandlers {
  onProgress?: (data: { progress: number; status: string }) => void
  onComplete?: (data: { result: string }) => void
  onError?: (data: { error: string }) => void
}

// ==================== 存储设置 ====================

export interface Settings {
  storage_target: string
  local_image_dir: string
  local_video_dir: string
  github_image_path: string
  github_video_path: string
}
