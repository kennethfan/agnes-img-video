import client from './client'

export async function uploadToGitHub(url: string, filename?: string, assetId?: number): Promise<string> {
  const res = await client.post('/api/v1/upload-to-github', { url, filename, asset_id: assetId })
  return res.data.github_url
}
