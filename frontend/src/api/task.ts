import client from './client'
import type { TaskRecord } from '../types'

export async function listTasks(params?: {
  type?: string
  status?: string
}): Promise<{ records: TaskRecord[] }> {
  const query = new URLSearchParams()
  if (params?.type) query.set('type', params.type)
  if (params?.status) query.set('status', params.status)
  const res = await client.get(`/api/v1/tasks?${query.toString()}`)
  return res.data
}

export async function getTask(taskId: string): Promise<TaskRecord> {
  const res = await client.get(`/api/v1/tasks/${taskId}`)
  return res.data
}

export async function retryTask(taskId: string): Promise<void> {
  await client.post(`/api/v1/tasks/${taskId}/retry`)
}

export async function cancelTask(taskId: string): Promise<void> {
  await client.post(`/api/v1/tasks/${taskId}/cancel`)
}
