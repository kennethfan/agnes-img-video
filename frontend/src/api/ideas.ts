import client from './client'

export async function expandIdea(title: string, content: string, tags: string): Promise<string> {
  const res = await client.post<{ result: string }>('/api/v1/ideas/expand', {
    title,
    content,
    tags,
  })
  return res.data.result
}
