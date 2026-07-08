import { ElMessage } from 'element-plus'
import client from './client'

/** 导出数据库文件：支持 .db 或 .sql 格式 */
export async function exportDB(format: 'db' | 'sql' = 'db') {
  const res = await fetch(`/api/v1/db/export?format=${format}`)
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: '导出失败' }))
    ElMessage.error(err.error || '导出失败')
    return
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `history.${format}`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/** 恢复数据库：上传 .db/.sqlite/.sql 文件 */
export async function restoreDB(file: File): Promise<void> {
  const form = new FormData()
  form.append('file', file)
  await client.post('/api/v1/db/restore', form)
}
