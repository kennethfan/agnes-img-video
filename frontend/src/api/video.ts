import client from './client'
import type { ScriptGenRequest, ScriptGenResponse, VideoCreateRequest, VideoTaskResponse, VideoStatus } from '../types'

export async function generateScript(data: ScriptGenRequest): Promise<ScriptGenResponse> {
  const res = await client.post('/api/v1/videos/generate-script', data, { timeout: 120000 })
  return res.data
}

export async function createTextToVideo(data: VideoCreateRequest): Promise<VideoTaskResponse> {
  const res = await client.post('/api/v1/videos/text-to-video', data, { timeout: 300000 })
  return res.data
}

export async function createImageToVideo(
  imageFile: File,
  prompt: string,
  duration: number = 5,
  aspectRatio: string = '16:9',
  frameRate: number = 24,
  negativePrompt: string = ''
): Promise<VideoTaskResponse> {
  const formData = new FormData()
  formData.append('image', imageFile)
  formData.append('prompt', prompt)
  formData.append('duration', String(duration))
  formData.append('aspect_ratio', aspectRatio)
  formData.append('frame_rate', String(frameRate))
  if (negativePrompt) {
    formData.append('negative_prompt', negativePrompt)
  }

  const res = await client.post('/api/v1/videos/image-to-video', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 300000,
  })
  return res.data
}

export async function createMultiImageVideo(data: VideoCreateRequest): Promise<VideoTaskResponse> {
  const res = await client.post('/api/v1/videos/multi-image', data, { timeout: 300000 })
  return res.data
}

export async function getTaskStatus(taskId: string): Promise<VideoStatus> {
  const res = await client.get(`/api/v1/videos/${taskId}`)
  return res.data
}
