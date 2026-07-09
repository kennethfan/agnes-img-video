import client from './client'

export async function retryTask(taskId: string): Promise<void> {
  await client.post(`/api/v1/tasks/${taskId}/retry`)
}

export async function cancelTask(taskId: string): Promise<void> {
  await client.post(`/api/v1/tasks/${taskId}/cancel`)
}
