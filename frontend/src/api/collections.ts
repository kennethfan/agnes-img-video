import client from './client'

export interface Collection {
  id: number
  name: string
  created_at: string
  updated_at: string
  assets?: { id: number }[]
}

export async function getCollections(): Promise<Collection[]> {
  const res = await client.get('/api/v1/collections')
  return res.data
}

export async function createCollection(name: string): Promise<Collection> {
  const res = await client.post('/api/v1/collections', { name })
  return res.data
}

export async function updateCollection(id: number, name: string): Promise<void> {
  await client.put(`/api/v1/collections/${id}`, { name })
}

export async function deleteCollection(id: number): Promise<void> {
  await client.delete(`/api/v1/collections/${id}`)
}

export async function addAssetsToCollection(collectionId: number, assetIds: number[]): Promise<void> {
  await client.post(`/api/v1/collections/${collectionId}/assets`, { asset_ids: assetIds })
}

export async function removeAssetsFromCollection(collectionId: number, assetIds: number[]): Promise<void> {
  await client.delete(`/api/v1/collections/${collectionId}/assets`, {
    data: { asset_ids: assetIds }
  })
}
