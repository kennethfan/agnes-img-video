import type { TaskSSEHandlers, SSEHandlers, VideoEvent } from '../types'

export function connectTaskSSE(taskId: number | string, handlers: TaskSSEHandlers): () => void {
  const url = `/api/v1/tasks/${taskId}/stream`
  const source = new EventSource(url)

  source.addEventListener('progress', (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data)
      handlers.onProgress?.(data)
    } catch {
      // ignore parse errors
    }
  })

  source.addEventListener('complete', (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data)
      handlers.onComplete?.(data)
      source.close()
    } catch {
      // ignore parse errors
    }
  })

  source.addEventListener('error', (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data)
      handlers.onError?.(data)
      source.close()
    } catch {
      if (source.readyState === EventSource.CLOSED) {
        handlers.onError?.({ error: 'SSE 连接已断开' })
      }
    }
  })

  return () => {
    source.close()
  }
}

// 保留旧函数名兼容（视频视图）
export function connectSSE(taskId: number | string, handlers: SSEHandlers): () => void {
  const url = `/api/v1/videos/stream/${taskId}`
  const source = new EventSource(url)

  source.addEventListener('progress', (e: MessageEvent) => {
    try {
      const data: VideoEvent = JSON.parse(e.data)
      handlers.onProgress?.(data)
    } catch {
      // ignore parse errors
    }
  })

  source.addEventListener('complete', (e: MessageEvent) => {
    try {
      const data: VideoEvent = JSON.parse(e.data)
      handlers.onComplete?.(data)
      source.close()
    } catch {
      // ignore parse errors
    }
  })

  source.addEventListener('error', (e: MessageEvent) => {
    try {
      const data: VideoEvent = JSON.parse(e.data)
      handlers.onError?.(data)
      source.close()
    } catch {
      if (source.readyState === EventSource.CLOSED) {
        handlers.onError?.({ error: 'SSE 连接已断开' })
      }
    }
  })

  return () => {
    source.close()
  }
}
