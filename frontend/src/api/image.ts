import client from './client'
import type { TextToImageRequest, BatchRequest, ImageResponse } from '../types'

export async function textToImage(data: TextToImageRequest): Promise<ImageResponse> {
  const res = await client.post('/api/v1/images/text-to-image', data)
  return res.data
}

export async function imageToImage(
  image: File | string,
  prompt: string,
  size: string = '1024x1024',
  strength: number = 0.75,
  negativePrompt: string = ''
): Promise<ImageResponse> {
  if (typeof image === 'string') {
    const res = await client.post('/api/v1/images/image-to-image', {
      image_url: image,
      prompt,
      size,
      strength,
      negative_prompt: negativePrompt || undefined,
    }, { timeout: 180000 })
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

  const res = await client.post('/api/v1/images/image-to-image', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 180000,
  })
  return res.data
}

export async function batchGenerate(data: BatchRequest): Promise<ImageResponse> {
  const res = await client.post('/api/v1/images/batch', data)
  return res.data
}
