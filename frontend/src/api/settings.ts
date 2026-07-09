import client from './client'

export interface Settings {
  storage_target: string
  local_image_dir: string
  local_video_dir: string
  github_image_path: string
  github_video_path: string
}

export async function getSettings(): Promise<Settings> {
  const res = await client.get('/api/v1/settings')
  return res.data
}

export async function updateSettings(s: Settings): Promise<void> {
  await client.put('/api/v1/settings', s)
}
