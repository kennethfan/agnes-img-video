import client from './client'

export async function ideateBrief(
  projectId: number,
  idea: string
): Promise<{ brief_text: string; generated_prompt: string }> {
  const res = await client.post(`/api/v1/projects/${projectId}/ideate-brief`, { idea })
  return res.data
}
