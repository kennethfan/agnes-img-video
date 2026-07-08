<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElButton, ElInput, ElSelect, ElOption, ElTable, ElTableColumn, ElTag, ElPagination, ElTooltip, ElDatePicker, ElMessage, ElMessageBox } from 'element-plus'
import { queryAccessLogs, deleteAccessLog, clearAccessLogs } from '../api/access-logs'
import type { AccessLogItem, AccessLogQuery } from '../api/access-logs'

const logs = ref<AccessLogItem[]>([])
const total = ref(0)
const loading = ref(false)

const filterMethod = ref('')
const filterPath = ref('')
const filterStatusMin = ref('')
const filterStatusMax = ref('')
const filterFrom = ref<Date | null>(null)
const filterTo = ref<Date | null>(null)

/** 将 Date 格式化为与后端一致的 RFC3339 本地时间字符串 (2006-01-02T15:04:05+08:00) */
function toLocalRFC3339(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  const offset = -d.getTimezoneOffset()
  const sign = offset >= 0 ? '+' : '-'
  const absOffset = Math.abs(offset)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}${sign}${pad(Math.floor(absOffset / 60))}:${pad(absOffset % 60)}`
}
const page = ref(1)
const pageSize = ref(50)
const expandedRow = ref<Set<number>>(new Set())

async function fetchLogs() {
  loading.value = true
  try {
    const q: AccessLogQuery = {
      page: page.value,
      page_size: pageSize.value,
      sort: 'desc',
    }
    if (filterMethod.value) q.method = filterMethod.value
    if (filterPath.value) q.path = filterPath.value
    if (filterStatusMin.value) q.status_min = Number(filterStatusMin.value)
    if (filterStatusMax.value) q.status_max = Number(filterStatusMax.value)
    if (filterFrom.value) q.from = toLocalRFC3339(filterFrom.value)
    if (filterTo.value) q.to = toLocalRFC3339(filterTo.value)

    const res = await queryAccessLogs(q)
    logs.value = res.items
    total.value = res.total
  } catch (e: any) {
    console.error('Failed to load access logs:', e)
  } finally {
    loading.value = false
  }
}

function toggleRow(id: number) {
  if (expandedRow.value.has(id)) {
    expandedRow.value.delete(id)
  } else {
    expandedRow.value.add(id)
  }
}

function formatTime(ts: string) {
  const d = new Date(ts)
  return d.toLocaleString('zh-CN', { hour12: false })
}

function statusTagType(status: number): 'info' | 'success' | 'warning' | 'danger' {
  if (status >= 200 && status < 300) return 'success'
  if (status >= 300 && status < 400) return 'warning'
  if (status >= 400) return 'danger'
  return 'info'
}

function copyText(text: string) {
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制')
  }).catch(() => {
    ElMessage.warning('复制失败')
  })
}

function handlePageChange(p: number) {
  page.value = p
  fetchLogs()
}

function handleSizeChange(s: number) {
  pageSize.value = s
  page.value = 1
  fetchLogs()
}

function refresh() {
  page.value = 1
  fetchLogs()
}

async function handleDeleteLog(id: number, e: Event) {
  e.stopPropagation()
  try {
    await ElMessageBox.confirm('确定删除此条日志？', '确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteAccessLog(id)
    logs.value = logs.value.filter(r => r.id !== id)
    total.value--
    ElMessage.success('已删除')
  } catch { /* cancelled */ }
}

async function handleClearLogs() {
  try {
    await ElMessageBox.confirm('确定清空所有日志？此操作不可恢复。', '确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await clearAccessLogs()
    logs.value = []
    total.value = 0
    ElMessage.success('日志已清空')
  } catch { /* cancelled */ }
}

onMounted(fetchLogs)
</script>

<template>
  <div class="access-logs">
    <div class="header">
      <h3>接口日志</h3>
      <div>
        <el-button size="small" @click="refresh" :loading="loading">刷新</el-button>
        <el-button size="small" type="danger" plain @click="handleClearLogs">清空日志</el-button>
      </div>
    </div>

    <!-- Filters -->
    <div class="filters">
      <el-select v-model="filterMethod" placeholder="方法" clearable size="small" style="width: 110px" @change="refresh">
        <el-option label="全部" value="" />
        <el-option label="GET" value="GET" />
        <el-option label="POST" value="POST" />
        <el-option label="PUT" value="PUT" />
        <el-option label="DELETE" value="DELETE" />
      </el-select>
      <el-input v-model="filterPath" placeholder="路径关键词" clearable size="small" style="width: 200px" @clear="refresh" @keyup.enter="refresh" />
      <el-input v-model="filterStatusMin" placeholder="状态码≥" clearable size="small" style="width: 100px" @clear="refresh" @keyup.enter="refresh" />
      <el-input v-model="filterStatusMax" placeholder="状态码≤" clearable size="small" style="width: 100px" @clear="refresh" @keyup.enter="refresh" />
      <el-date-picker v-model="filterFrom" type="datetime" placeholder="起始时间" clearable size="small" style="width: 200px" @change="refresh" />
      <el-date-picker v-model="filterTo" type="datetime" placeholder="结束时间" clearable size="small" style="width: 200px" @change="refresh" />
    </div>

    <!-- Table -->
    <el-table
      :data="logs"
      v-loading="loading"
      stripe
      size="small"
      style="width: 100%"
      :row-key="(row: any) => row.id"
      @row-click="(row: any) => toggleRow(row.id)"
      :expand-row-keys="[...expandedRow].map(String)"
    >
      <el-table-column type="expand" width="1">
        <template #default="{ row }: { row: any }">
          <div class="log-detail">
            <div class="detail-section">
              <div class="detail-label">
                请求体
                <el-button v-if="row.request_body" size="small" link type="primary" @click.stop="copyText(row.request_body)">复制</el-button>
              </div>
              <pre class="detail-code">{{ row.request_body || '(空)' }}</pre>
            </div>
            <div class="detail-section">
              <div class="detail-label">
                响应体
                <el-button v-if="row.response_body" size="small" link type="primary" @click.stop="copyText(row.response_body)">复制</el-button>
              </div>
              <pre class="detail-code">{{ row.response_body || '(空)' }}</pre>
            </div>
            <div v-if="row.error" class="detail-section">
              <div class="detail-label">错误</div>
              <pre class="detail-code detail-error">{{ row.error }}</pre>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="timestamp" label="时间" width="170" :formatter="(_: any, __: any, v: string) => formatTime(v)" />
      <el-table-column prop="method" label="方法" width="80">
        <template #default="{ row }: { row: any }">
          <el-tag :type="row.method === 'GET' ? 'info' : row.method === 'POST' ? 'warning' : 'danger'" size="small" effect="plain">
            {{ row.method }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="path" label="路径" min-width="200">
        <template #default="{ row }: { row: any }">
          <el-tooltip :content="row.path" placement="top">
            <span class="log-path">{{ row.path }}</span>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态码" width="80">
        <template #default="{ row }: { row: any }">
          <el-tag :type="statusTagType(row.status)" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="duration_ms" label="耗时(ms)" width="90">
        <template #default="{ row }: { row: any }">
          <span :class="{ 'slow': row.duration_ms > 5000 }">{{ row.duration_ms }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="client_ip" label="客户端IP" width="130" />
      <el-table-column label="操作" width="70" fixed="right">
        <template #default="{ row }: { row: any }">
          <el-button size="small" type="danger" link @click="(e: Event) => handleDeleteLog(row.id, e)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <div class="pagination-wrap">
      <el-pagination
        v-if="total > 0"
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[20, 50, 100]"
        layout="total, sizes, prev, pager, next"
        background
        small
        @current-change="handlePageChange"
        @size-change="handleSizeChange"
      />
    </div>
  </div>
</template>

<style scoped>
.access-logs { padding: 0; }
.header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.header h3 { margin: 0; font-size: 16px; }
.filters { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.log-path { cursor: pointer; }
.log-path:hover { color: #409eff; }
.slow { color: #e6a23c; font-weight: 600; }
.pagination-wrap { margin-top: 16px; display: flex; justify-content: center; }
.log-detail { padding: 12px 24px; }
.detail-section { margin-bottom: 12px; }
.detail-label { font-size: 12px; color: #909399; margin-bottom: 4px; font-weight: 600; }
.detail-code { margin: 0; padding: 8px 12px; background: #f8f8f8; border-radius: 6px; font-size: 12px; line-height: 1.5; max-height: 200px; overflow-y: auto; white-space: pre-wrap; word-break: break-all; }
.detail-error { color: #f56c6c; }
</style>
