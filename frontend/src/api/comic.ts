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
