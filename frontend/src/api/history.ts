import client from './client'
import type { HistoryRecord, DeleteHistoryRequest } from '../types'

export async function getHistory(): Promise<{ records: HistoryRecord[] }> {
  const res = await client.get('/api/v1/history')
  return res.data
}

export async function clearHistory(deleteFiles?: boolean): Promise<void> {
  const params = deleteFiles ? '?delete_files=true' : ''
  await client.delete(`/api/v1/history${params}`)
}

export async function deleteHistory(data: DeleteHistoryRequest): Promise<void> {
  await client.post('/api/v1/history/delete', data)
}

export async function deleteRecord(id: number, deleteFiles?: boolean): Promise<void> {
  const params = deleteFiles ? '?delete_files=true' : ''
  await client.delete(`/api/v1/history/${id}${params}`)
}
