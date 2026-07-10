<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listTasks, retryTask, cancelTask } from '../api/task'
import type { TaskRecord } from '../types'

const records = ref<TaskRecord[]>([])
const loading = ref(false)
const detailDialogVisible = ref(false)
const detailRecord = ref<TaskRecord | null>(null)

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

function formatJSON(str: string | undefined | null): string {
  if (!str) return '-'
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

function handleViewDetail(row: TaskRecord) {
  detailRecord.value = row
  detailDialogVisible.value = true
}

async function copyText(text: string | undefined | null, label: string) {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(`${label}已复制`)
  } catch {
    ElMessage.error('复制失败')
  }
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
      <div class="page-header-left">
        <h2>任务记录</h2>
        <span class="page-desc">查看和管理异步任务的执行状态</span>
      </div>
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
      class="task-table"
      header-cell-class-name="task-table-header"
    >
      <el-table-column prop="id" label="ID" width="80" align="center" />
      <el-table-column label="类型" width="100">
        <template #default="{ row }">
          <el-tag size="small" effect="plain" round>
            {{ taskTypeLabel(row.type) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="getStatusConf(row.status).type as any" size="small" effect="light" round>
            {{ getStatusConf(row.status).label }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="进度" width="150">
        <template #default="{ row }">
          <el-progress
            :percentage="row.progress"
            :status="row.status === 'failed' ? 'exception' : row.status === 'completed' ? 'success' : undefined"
            :stroke-width="14"
            :text-inside="true"
            style="width: 130px"
          />
        </template>
      </el-table-column>
      <el-table-column label="错误信息" min-width="100">
        <template #default="{ row }">
          <span v-if="row.error" class="error-text">
            <el-popover
              trigger="click"
              placement="bottom"
              :width="380"
              popper-class="error-popover"
            >
              <template #reference>
                <span class="error-truncate">{{ row.error }}</span>
              </template>
              <div class="error-full">{{ row.error }}</div>
            </el-popover>
          </span>
          <span v-else class="no-data">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="retry_count" label="重试" width="60" align="center" />
      <el-table-column label="创建时间" width="130">
        <template #default="{ row }">
          <span class="time-text">{{ relativeTime(row.created_at) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <div class="action-btns">
            <el-button size="small" @click="handleViewDetail(row)">详情</el-button>
            <el-button
              v-if="row.status === 'pending'"
              size="small"
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
          </div>
        </template>
      </el-table-column>
    </el-table>

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="任务详情"
      width="640px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <template v-if="detailRecord">
        <div class="detail-grid">
          <div class="detail-field">
            <span class="detail-label">任务 ID</span>
            <span class="detail-value">{{ detailRecord.id }}</span>
          </div>
          <div class="detail-field">
            <span class="detail-label">类型</span>
            <span class="detail-value">{{ taskTypeLabel(detailRecord.type) }}</span>
          </div>
          <div class="detail-field">
            <span class="detail-label">状态</span>
            <span class="detail-value">
              <el-tag :type="getStatusConf(detailRecord.status).type as any" size="small">
                {{ getStatusConf(detailRecord.status).label }}
              </el-tag>
            </span>
          </div>
          <div class="detail-field">
            <span class="detail-label">进度</span>
            <span class="detail-value">
              <el-progress
                :percentage="detailRecord.progress"
                :status="detailRecord.status === 'failed' ? 'exception' : detailRecord.status === 'completed' ? 'success' : undefined"
                :stroke-width="14"
                :text-inside="true"
                style="width: 200px"
              />
            </span>
          </div>
          <div class="detail-field">
            <span class="detail-label">重试次数</span>
            <span class="detail-value">{{ detailRecord.retry_count }}</span>
          </div>
          <div class="detail-field">
            <span class="detail-label">创建时间</span>
            <span class="detail-value">{{ detailRecord.created_at }}</span>
          </div>
          <div class="detail-field">
            <span class="detail-label">更新时间</span>
            <span class="detail-value">{{ detailRecord.updated_at }}</span>
          </div>
          <div v-if="detailRecord.completed_at" class="detail-field">
            <span class="detail-label">完成时间</span>
            <span class="detail-value">{{ detailRecord.completed_at }}</span>
          </div>
        </div>

        <div v-if="detailRecord.error" class="detail-section">
          <div class="detail-section-title">
            错误信息
            <el-button size="small" text @click="copyText(detailRecord!.error, '错误信息')">复制</el-button>
          </div>
          <div class="detail-code error-bg">{{ detailRecord.error }}</div>
        </div>

        <div class="detail-section">
          <div class="detail-section-title">
            请求参数
            <el-button size="small" text @click="copyText(formatJSON(detailRecord.params), '请求参数')">复制</el-button>
          </div>
          <pre class="detail-code">{{ formatJSON(detailRecord.params) }}</pre>
        </div>

        <div v-if="detailRecord.result" class="detail-section">
          <div class="detail-section-title">
            返回结果
            <el-button size="small" text @click="copyText(formatJSON(detailRecord.result), '返回结果')">复制</el-button>
          </div>
          <pre class="detail-code">{{ formatJSON(detailRecord.result) }}</pre>
        </div>
      </template>
    </el-dialog>
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
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 12px;
}
.page-header-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
}
.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #1d2129;
}
.page-desc {
  font-size: 13px;
  color: #86909c;
}
.filters {
  display: flex;
  gap: 10px;
  align-items: center;
}

/* ==================== Table 美化 ==================== */
.task-table {
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
}
.task-table :deep(.el-table__header-wrapper) {
  border-bottom: 1px solid #e5e6eb;
}
.task-table :deep(.el-table__body tr:hover td) {
  background-color: #f5f7fa;
}
.task-table :deep(.el-table__body tr.el-table__row--striped:hover td) {
  background-color: #eef1f6;
}

/* ==================== Column specifics ==================== */
.error-text {
  color: #f56c6c;
  font-size: 13px;
  cursor: pointer;
}
.error-truncate {
  display: inline-block;
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
}
.error-full {
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow-y: auto;
}
.no-data {
  color: #c0c4cc;
}
.time-text {
  font-size: 13px;
  color: #606266;
}
.action-btns {
  display: flex;
  gap: 6px;
}
.done-text {
  color: #67c23a;
  font-size: 13px;
}
.cancelled-text {
  color: #909399;
  font-size: 13px;
}

/* ==================== Detail Dialog ==================== */
.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 24px;
  margin-bottom: 16px;
}
.detail-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.detail-label {
  font-size: 12px;
  color: #86909c;
  font-weight: 500;
}
.detail-value {
  font-size: 14px;
  color: #1d2129;
}
.detail-section {
  margin-top: 16px;
}
.detail-section-title {
  font-size: 13px;
  font-weight: 600;
  color: #4e5969;
  margin-bottom: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid #e5e6eb;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.detail-code {
  background: #f7f8fa;
  border: 1px solid #e5e6eb;
  border-radius: 6px;
  padding: 12px 14px;
  font-size: 12px;
  line-height: 1.6;
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', Menlo, monospace;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 400px;
  overflow: auto;
  margin: 0;
}
.error-bg {
  background: #fef0f0;
  border-color: #fcdcdc;
  color: #f56c6c;
}
</style>
