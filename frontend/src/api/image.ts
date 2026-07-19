import client from './client'
import type { TextToImageRequest, BatchRequest, TaskCreateResponse, ImageResponse } from '../types'

export async function submitTextToImage(data: TextToImageRequest): Promise<TaskCreateResponse> {
  const res = await client.post('/api/v1/images/text-to-image', data)
  return res.data
}

export async function submitImageToImage(
  image: File | string,
  prompt: string,
  size: string = '1024x1024',
  strength: number = 0.75,
  negativePrompt: string = '',
  projectId?: number
): Promise<TaskCreateResponse> {
  if (typeof image === 'string') {
    const res = await client.post('/api/v1/images/image-to-image', {
      image_url: image,
      prompt,
      size,
      strength,
      negative_prompt: negativePrompt || undefined,
      project_id: projectId,
    }, { timeout: 60000 })
    return res.data
  }
  const formData = new FormData()
  formData.append('image', image)
  formData.append('prompt', prompt)
  formData.append('size', size)
  formData.append('strength', String(strength))
  if (negativePrompt) {
    formData.append('negative_prompt', negativePrompt)
  }
  if (projectId !== undefined) {
    formData.append('project_id', String(projectId))
  }

  const res = await client.post('/api/v1/images/image-to-image', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 60000,
  })
  return res.data
}

export async function submitBatchGenerate(data: BatchRequest): Promise<TaskCreateResponse> {
  const res = await client.post('/api/v1/images/batch', data)
  return res.data
}

/**
 * 提交图片任务并等待完成（用于向后兼容 wizard 视图等同步调用场景）
 * 内部使用轮询等待，不依赖 SSE
 */
async function submitAndPoll(submitFn: (data: any) => Promise<TaskCreateResponse>, data: any): Promise<any> {
  const { taskId } = await submitFn(data)
  // 轮询等待任务完成
  const maxRetries = 60
  for (let i = 0; i < maxRetries; i++) {
    await new Promise(r => setTimeout(r, 2000))
    const res = await client.get(`/api/v1/tasks/${taskId}`)
    const task = res.data
    if (task.status === 'completed') {
      try {
        return { images: JSON.parse(task.result) }
      } catch {
        return { images: [task.result] }
      }
    }
    if (task.status === 'failed') {
      throw new Error(task.error || '任务失败')
    }
  }
  throw new Error('任务超时')
}

export async function textToImage(data: TextToImageRequest): Promise<ImageResponse> {
  return submitAndPoll(submitTextToImage, data)
}

export async function imageToImage(
  image: File | string,
  prompt: string,
  size?: string,
  strength?: number,
  negativePrompt?: string,
  projectId?: number
): Promise<ImageResponse> {
  return submitAndPoll(
    (d: any) => submitImageToImage(d.image, d.prompt, d.size, d.strength, d.negativePrompt, d.projectId),
    { image, prompt, size, strength, negativePrompt, projectId }
  )
}
