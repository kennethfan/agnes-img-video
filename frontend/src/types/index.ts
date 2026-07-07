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
  taskId: string
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
