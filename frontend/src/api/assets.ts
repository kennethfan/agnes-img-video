import client from './client'
import type { AssetListResponse, AssetFavoriteRequest, AssetDeleteRequest, AssetItem } from '../types'

export async function saveAsset(data: { image_url: string; prompt: string; mode: string; project_id?: number }): Promise<{ id: number }> {
  const res = await client.post('/api/v1/assets', data)
  return res.data
}

export interface AssetQuery {
  page?: number
  per_page?: number
  type?: 'image' | 'video' | 'all'
  sort?: 'newest' | 'oldest'
  search?: string
  favorite?: string
}

export async function getAssets(params: AssetQuery = {}): Promise<AssetListResponse> {
  const res = await client.get('/api/v1/assets', { params })
  return res.data
}

export async function toggleFavorite(data: AssetFavoriteRequest): Promise<void> {
  await client.post('/api/v1/assets/favorite', data)
}

export async function batchDownload(ids: number[]): Promise<Blob> {
  const res = await client.post('/api/v1/assets/batch-download', { ids }, {
    responseType: 'blob',
  })
  return res.data
}

export async function transferAsset(id: number): Promise<AssetItem> {
  const res = await client.post(`/api/v1/assets/${id}/transfer`)
  return res.data.asset || res.data
}

export async function deleteAssets(data: AssetDeleteRequest): Promise<void> {
  await client.delete('/api/v1/assets', { data })
}
