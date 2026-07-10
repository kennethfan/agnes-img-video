import client from './client'
import type { HistoryRecord, DeleteHistoryRequest } from '../types'

export async function getHistory(): Promise<{ records: HistoryRecord[] }> {
  const res = await client.get('/api/v1/history')
  return res.data
}

export async function clearHistory(): Promise<void> {
  await client.delete('/api/v1/history')
}

export async function deleteHistory(data: DeleteHistoryRequest): Promise<void> {
  await client.post('/api/v1/history/delete', data)
}

export async function deleteRecord(id: number): Promise<void> {
  await client.delete(`/api/v1/history/${id}`)
}
