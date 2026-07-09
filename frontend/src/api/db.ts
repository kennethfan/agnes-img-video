import { ElMessage } from 'element-plus'

/** 导出数据库备份（JSON 格式） */
export async function exportDB() {
  const res = await fetch('/api/v1/db/export')
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: '导出失败' }))
    ElMessage.error(err.error || '导出失败')
    return
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'history.json'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/** 恢复数据库：上传 .json 文件 */
export async function restoreDB(file: File): Promise<void> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch('/api/v1/db/restore', { method: 'POST', body: form })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: '恢复失败' }))
    throw new Error(err.error || '恢复失败')
  }
}
