export interface AccessLogItem {
  id: number
  timestamp: string
  method: string
  path: string
  status: number
  duration_ms: number
  client_ip: string
  user_agent: string
  request_body: string
  response_body: string
  error: string
}

export interface AccessLogQueryResult {
  items: AccessLogItem[]
  total: number
  page: number
  page_size: number
}

export interface AccessLogQuery {
  page?: number
  page_size?: number
  method?: string
  path?: string
  status_min?: number
  status_max?: number
  from?: string
  to?: string
  sort?: 'asc' | 'desc'
}

import client from './client'

export async function queryAccessLogs(q: AccessLogQuery = {}): Promise<AccessLogQueryResult> {
  const params = new URLSearchParams()
  if (q.page) params.set('page', String(q.page))
  if (q.page_size) params.set('page_size', String(q.page_size))
  if (q.method) params.set('method', q.method)
  if (q.path) params.set('path', q.path)
  if (q.status_min) params.set('status_min', String(q.status_min))
  if (q.status_max) params.set('status_max', String(q.status_max))
  if (q.from) params.set('from', q.from)
  if (q.to) params.set('to', q.to)
  if (q.sort) params.set('sort', q.sort)

  const res = await client.get(`/api/v1/access-logs?${params.toString()}`)
  return res.data
}

export async function deleteAccessLog(id: number): Promise<void> {
  await client.delete(`/api/v1/access-logs/${id}`)
}

export async function clearAccessLogs(): Promise<void> {
  await client.delete('/api/v1/access-logs')
}
