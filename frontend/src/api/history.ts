import client from './client'
import type { HistoryRecord } from '../types'

export async function getHistory(): Promise<{ records: HistoryRecord[] }> {
  const res = await client.get('/api/v1/history')
  return res.data
}

export async function clearHistory(): Promise<void> {
  await client.delete('/api/v1/history')
}
