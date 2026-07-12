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
  project_id?: number
}

export interface DeleteHistoryRequest {
  ids: number[]
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
  type: string
  time: string
  favorite: boolean
  original_url: string
  local_path: string
  github_url: string
  thumbnail: string
}

export interface AssetListResponse {
  items: AssetItem[]
  total: number
  page: number
}

export interface AssetFavoriteRequest {
  asset_id: number
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

export interface GenerateShotsResponse {
  submitted: number
  total: number
  failed: number
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
  project_id?: number
}

export interface TaskCreateResponse {
  taskId: number
}

export interface TaskSSEHandlers {
  onProgress?: (data: { progress: number; status: string }) => void
  onComplete?: (data: { result: string }) => void
  onError?: (data: { error: string }) => void
}

// ==================== 创作项目 ====================

export interface ProjectStep {
  id: number
  project_id: number
  step_type: string
  position: number
  input: string
  output: string
  created_at: string
}

export interface Project {
  id: number
  title: string
  brief: string
  ai_result: string
  status: 'draft' | 'generating' | 'refining' | 'completed'
  cover_url: string
  final_url: string
  asset_ids: string
  notes: string
  step_progress: string  // JSON string {"ideate":"completed",...}
  created_at: string
  updated_at: string
  steps: ProjectStep[]
}

export interface CreateProjectRequest {
  title: string
  brief?: string
}

export interface UpdateProjectRequest {
	title?: string
	brief?: string
	status?: string
	notes?: string
	cover_url?: string
}

export interface ProjectStepRequest {
  step_type: string
  position?: number
}

// ==================== 项目仪表盘 ====================

export interface ProjectFile {
  id: number
  type: 'image' | 'video'
  source: 'history' | 'asset'
  url: string
  prompt: string
  step: string
  created_at: string
}

export interface ProjectStats {
  file_count: number
  optimized_count: number
  running_tasks: number
  last_activity: string
  step_progress: Record<string, string>
}

// ==================== 存储设置 ====================

export interface Settings {
  storage_target: string
  local_image_dir: string
  local_video_dir: string
  github_image_path: string
  github_video_path: string
}
