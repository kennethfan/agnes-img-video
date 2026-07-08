import client from './client'

export async function uploadToGitHub(url: string, filename?: string): Promise<string> {
  const res = await client.post('/api/v1/upload-to-github', { url, filename })
  return res.data.github_url
}
