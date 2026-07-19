import client from './client'
import type { ScriptGenRequest, ScriptGenResponse, VideoCreateRequest, TaskCreateResponse } from '../types'

export async function generateScript(data: ScriptGenRequest): Promise<ScriptGenResponse> {
  const res = await client.post('/api/v1/videos/generate-script', data, { timeout: 120000 })
  return res.data
}

export async function submitTextToVideo(data: VideoCreateRequest): Promise<TaskCreateResponse> {
  const res = await client.post('/api/v1/videos/text-to-video', data, { timeout: 30000 })
  return res.data
}

export async function submitImageToVideo(
  image: File | string,
  prompt: string,
  duration: number = 5,
  aspectRatio: string = '16:9',
  frameRate: number = 24,
  negativePrompt: string = '',
  projectId?: number
): Promise<TaskCreateResponse> {
  if (typeof image === 'string') {
    const res = await client.post('/api/v1/videos/image-to-video', {
      image_url: image,
      prompt,
      duration,
      aspect_ratio: aspectRatio,
      frame_rate: frameRate,
      negative_prompt: negativePrompt || undefined,
      project_id: projectId,
    }, { timeout: 30000 })
    return res.data
  }
  const formData = new FormData()
  formData.append('image', image)
  formData.append('prompt', prompt)
  formData.append('duration', String(duration))
  formData.append('aspect_ratio', aspectRatio)
  formData.append('frame_rate', String(frameRate))
  if (negativePrompt) {
    formData.append('negative_prompt', negativePrompt)
  }
  if (projectId !== undefined) {
    formData.append('project_id', String(projectId))
  }

  const res = await client.post('/api/v1/videos/image-to-video', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 30000,
  })
  return res.data
}

export async function submitMultiImageVideo(data: VideoCreateRequest): Promise<TaskCreateResponse> {
  const res = await client.post('/api/v1/videos/multi-image', data, { timeout: 30000 })
  return res.data
}
