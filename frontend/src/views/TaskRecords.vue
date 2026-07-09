<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listTasks, retryTask, cancelTask } from '../api/task'
import type { TaskRecord } from '../types'

const records = ref<TaskRecord[]>([])
const loading = ref(false)

const statusFilter = ref('')
const typeFilter = ref('')

const typeOptions = [
  { value: '', label: '全部类型' },
  { value: 'text2image', label: '文生图' },
  { value: 'image2image', label: '图生图' },
  { value: 'batch', label: '批量生成' },
  { value: 'text2video', label: '文生视频' },
  { value: 'image2video', label: '图生视频' },
  { value: 'multi_image_video', label: '多图视频' },
]

const statusConfig: Record<string, { label: string; type: string }> = {
  pending:    { label: '等待中', type: 'info' },
  processing: { label: '进行中', type: 'primary' },
  completed:  { label: '已完成', type: 'success' },
  failed:     { label: '失败',   type: 'danger' },
  cancelled:  { label: '已取消', type: 'warning' },
}

function getStatusConf(status: string) {
  return statusConfig[status] || { label: status, type: 'info' }
}

function relativeTime(timeStr: string): string {
  const now = Date.now()
  const t = new Date(timeStr.replace(' ', 'T')).getTime()
  if (isNaN(t)) return timeStr
  const diffSec = Math.floor((now - t) / 1000)
  if (diffSec < 60) return '刚刚'
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)} 分钟前`
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)} 小时前`
  if (diffSec < 172800) return '昨天'
  if (diffSec < 2592000) return `${Math.floor(diffSec / 86400)} 天前`
  return timeStr.slice(0, 10)
}

function taskTypeLabel(t: string): string {
  const opt = typeOptions.find(o => o.value === t)
  return opt ? opt.label : t
}

async function loadRecords() {
  loading.value = true
  try {
    const params: { type?: string; status?: string } = {}
    if (typeFilter.value) params.type = typeFilter.value
    if (statusFilter.value) params.status = statusFilter.value
    const res = await listTasks(params)
    records.value = res.records || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载任务列表失败')
  } finally {
    loading.value = false
  }
}

function handleFilterChange() {
  loadRecords()
}

async function handleCancel(taskId: string) {
  try {
    await ElMessageBox.confirm('确定取消此任务？', '确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await cancelTask(taskId)
    ElMessage.success('任务已取消')
    loadRecords()
  } catch { /* cancelled */ }
}

async function handleRetry(taskId: string) {
  try {
    await retryTask(taskId)
    ElMessage.success('任务已重新提交')
    loadRecords()
  } catch (e: any) {
    ElMessage.error(e.message || '重试失败')
  }
}

onMounted(loadRecords)
</script>

<template>
  <div class="task-records-page">
    <div class="page-header">
      <h2>任务记录</h2>
      <div class="filters">
        <el-select
          v-model="typeFilter"
          style="width: 140px"
          @change="handleFilterChange"
        >
          <el-option
            v-for="opt in typeOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>
        <el-select
          v-model="statusFilter"
          placeholder="全部状态"
          style="width: 130px"
          @change="handleFilterChange"
        >
          <el-option label="全部状态" value="" />
          <el-option label="等待中" value="pending" />
          <el-option label="进行中" value="processing" />
          <el-option label="已完成" value="completed" />
          <el-option label="失败" value="failed" />
          <el-option label="已取消" value="cancelled" />
        </el-select>
        <el-button type="primary" @click="loadRecords">刷新</el-button>
      </div>
    </div>

    <el-table
      v-loading="loading"
      :data="records"
      stripe
      style="width: 100%"
      empty-text="暂无任务记录"
    >
      <el-table-column prop="id" label="任务 ID" width="200" show-overflow-tooltip />
      <el-table-column label="类型" width="110">
        <template #default="{ row }">
          {{ taskTypeLabel(row.type) }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-tag :type="getStatusConf(row.status).type as any" size="small">
            {{ getStatusConf(row.status).label }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="进度" width="140">
        <template #default="{ row }">
          <el-progress
            :percentage="row.progress"
            :status="row.status === 'failed' ? 'exception' : row.status === 'completed' ? 'success' : undefined"
            :stroke-width="12"
            :text-inside="true"
            style="width: 120px"
          />
        </template>
      </el-table-column>
      <el-table-column prop="error" label="错误信息" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.error" style="color: #f56c6c; font-size: 13px">{{ row.error }}</span>
          <span v-else style="color: #c0c4cc">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="retry_count" label="重试次数" width="80" align="center" />
      <el-table-column label="创建时间" width="140">
        <template #default="{ row }">
          {{ relativeTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="row.status === 'pending'"
            size="small"
            type="warning"
            @click="handleCancel(row.id)"
          >
            取消
          </el-button>
          <el-button
            v-if="row.status === 'failed'"
            size="small"
            type="primary"
            @click="handleRetry(row.id)"
          >
            重试
          </el-button>
          <span v-else-if="row.status === 'completed'" style="color: #67c23a; font-size: 13px">-</span>
          <span v-else-if="row.status === 'cancelled'" style="color: #909399; font-size: 13px">-</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.task-records-page {
  max-width: 1000px;
  margin: 0 auto;
  padding: 8px 0 40px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.page-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #000;
}
.filters {
  display: flex;
  gap: 10px;
  align-items: center;
}
</style>
