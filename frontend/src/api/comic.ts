import client from './client'

export async function generateComicPrompts(
  theme: string,
  layout: string,
  panelCount: number
): Promise<string[]> {
  const res = await client.post('/api/v1/comic/generate-prompts', {
    theme,
    layout,
    panel_count: panelCount,
  })
  return res.data.prompts
}

export async function generateStoryline(theme: string, style?: string): Promise<{
  storyline: string
  characters: string
  style: string
}> {
  const res = await client.post('/api/v1/comic/generate-storyline', { theme, style })
  return res.data
}
