import type { SSEHandlers, VideoEvent } from '../types'

export function connectSSE(taskId: string, handlers: SSEHandlers): () => void {
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
      // connection error (no data yet)
      if (source.readyState === EventSource.CLOSED) {
        handlers.onError?.({ error: 'SSE 连接已断开' })
      }
    }
  })

  return () => {
    source.close()
  }
}
