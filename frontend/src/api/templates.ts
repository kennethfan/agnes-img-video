import client from './client'

export interface PromptTemplate {
  id: number
  name: string
  type: 'template' | 'preset'
  category: string
  prompt: string
  negative_prompt: string
  size: string
  strength: number
  model: string
  created_at: string
  updated_at: string
}

export async function getTemplates(category?: string): Promise<PromptTemplate[]> {
  const params = category ? { category } : {}
  const res = await client.get('/api/v1/templates', { params })
  return res.data
}

export async function createTemplate(data: Partial<PromptTemplate>): Promise<PromptTemplate> {
  const res = await client.post('/api/v1/templates', data)
  return res.data
}

export async function updateTemplate(id: number, data: Partial<PromptTemplate>): Promise<void> {
  await client.put(`/api/v1/templates/${id}`, data)
}

export async function deleteTemplate(id: number): Promise<void> {
  await client.delete(`/api/v1/templates/${id}`)
}

export async function exportTemplates(): Promise<PromptTemplate[]> {
  const res = await client.post('/api/v1/templates/export')
  return res.data
}

export async function importTemplates(data: PromptTemplate[]): Promise<void> {
  await client.post('/api/v1/templates/import', data)
}
