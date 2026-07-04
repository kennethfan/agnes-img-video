<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getHistory, clearHistory, deleteHistory, deleteRecord } from '../api/history'
import type { HistoryRecord } from '../types'
import { useRedoStore } from '../stores/redo'

const records = ref<HistoryRecord[]>([])
const loading = ref(false)
const videoDialogVisible = ref(false)
const previewVideoUrl = ref('')
const selectedIds = ref<Set<number>>(new Set())
const deleteFiles = ref(false)
const expandedScripts = ref<Set<number>>(new Set())
const redoStore = useRedoStore()

const modeConfig: Record<string, { label: string; type: string; color: string }> = {
  text2image:        { label: '文生图',     type: 'primary', color: '#409eff' },
  image2image:       { label: '图生图',     type: 'success', color: '#67c23a' },
  batch:             { label: '批量生成',   type: 'warning', color: '#e6a23c' },
  text2video:        { label: '文生视频',   type: 'danger',  color: '#f56c6c' },
  image2video:       { label: '图生视频',   type: 'danger',  color: '#f56c6c' },
  multi_image_video: { label: '多图视频',   type: 'danger',  color: '#f56c6c' },
  script_gen:        { label: '脚本生成',   type: 'success', color: '#722ed1' },
}

function getModeConf(mode: string) {
  return modeConfig[mode] || { label: mode, type: 'info', color: '#909399' }
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

function isImageUrl(url: string): boolean {
  const ext = url.split('?')[0].toLowerCase()
  return /\.(png|jpe?g|gif|webp|bmp|svg)$/.test(ext)
}

function playVideo(url: string) {
  previewVideoUrl.value = url
  videoDialogVisible.value = true
}

function toggleScript(id: number) {
  const next = new Set(expandedScripts.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedScripts.value = next
}

// 根据历史记录构建重做数据
function handleRedo(rec: HistoryRecord) {
  const extra = rec.extra || {}
  
  switch (rec.mode) {
    case 'text2image':
      redoStore.setRedoData({
        mode: rec.mode,
        prompt: rec.prompt,
        negativePrompt: (extra.negative_prompt as string) || '',
        size: (extra.size as string) || '1024x1024',
        n: (extra.n as number) || 1,
      })
      break
    case 'image2image':
      redoStore.setRedoData({
        mode: rec.mode,
        prompt: rec.prompt,
        negativePrompt: (extra.negative_prompt as string) || '',
        size: (extra.size as string) || '1024x1024',
        strength: (extra.strength as number) || 0.75,
        inputMode: 'url',
        imageUrl: rec.images?.[0] || '',
      })
      break
    case 'batch':
      redoStore.setRedoData({
        mode: rec.mode,
        promptsText: rec.prompt,
        size: (extra.size as string) || '1024x1024',
      })
      break
    case 'script_gen':
      redoStore.setRedoData({
        mode: rec.mode,
        topic: rec.prompt,
        script: (extra.script as string) || '',
        style: (extra.style as string) || '',
        duration: (extra.duration as number) || 30,
        language: (extra.language as string) || 'zh',
      })
      break
    case 'text2video':
      redoStore.setRedoData({
        mode: rec.mode,
        prompt: rec.prompt,
        duration: (extra.duration as number) || 5,
        aspectRatio: (extra.aspect_ratio as string) || '16:9',
        frameRate: (extra.frame_rate as number) || 24,
      })
      break
    case 'image2video':
      redoStore.setRedoData({
        mode: rec.mode,
        prompt: rec.prompt,
        inputMode: 'url',
        imageUrl: rec.images?.[0] || '',
        duration: (extra.duration as number) || 5,
        aspectRatio: (extra.aspect_ratio as string) || '16:9',
        frameRate: (extra.frame_rate as number) || 24,
      })
      break
    case 'multi_image_video':
      redoStore.setRedoData({
        mode: rec.mode,
        prompt: rec.prompt,
        imageUrlsText: (extra.image_urls as string[])?.join('\n') || rec.images?.join('\n') || '',
        videoMode: (extra.mode as string) || 'ti2vid',
        duration: (extra.duration as number) || 5,
        aspectRatio: (extra.aspect_ratio as string) || '16:9',
        frameRate: (extra.frame_rate as number) || 24,
      })
      break
    default:
      ElMessage.warning('不支持的模式')
      return
  }
  
  // 触发tab切换（通过自定义事件）
  window.dispatchEvent(new CustomEvent('redo-trigger'))
  ElMessage.success('已加载到对应页面')
}

async function loadHistory() {
  loading.value = true
  try {
    const res = await getHistory()
    records.value = res.records || []
    selectedIds.value = new Set()
  } catch (e: any) {
    ElMessage.error(e.message || '加载历史失败')
  } finally {
    loading.value = false
  }
}

function toggleSelect(id: number) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = next
}

async function handleClear() {
  try {
    await ElMessageBox.confirm('确定清空所有历史记录？', '确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await clearHistory(deleteFiles.value)
    records.value = []
    selectedIds.value = new Set()
    ElMessage.success('历史已清空')
  } catch { /* cancelled */ }
}

async function handleDeleteRecord(id: number) {
  try {
    await ElMessageBox.confirm('确定删除此条记录？', '确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteRecord(id, deleteFiles.value)
    records.value = records.value.filter(r => r.id !== id)
    selectedIds.value.delete(id)
    ElMessage.success('已删除')
  } catch { /* cancelled */ }
}

async function handleBatchDelete() {
  if (selectedIds.value.size === 0) {
    ElMessage.warning('请先选择要删除的记录')
    return
  }
  try {
    const msg = deleteFiles.value
      ? `确定删除选中的 ${selectedIds.value.size} 条记录？关联的文件也将被删除。`
      : `确定删除选中的 ${selectedIds.value.size} 条记录？`
    await ElMessageBox.confirm(msg, '确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteHistory({ ids: Array.from(selectedIds.value), delete_files: deleteFiles.value })
    records.value = records.value.filter(r => !selectedIds.value.has(r.id))
    selectedIds.value = new Set()
    ElMessage.success('已删除')
  } catch { /* cancelled */ }
}

onMounted(loadHistory)
</script>

<template>
  <div class="history-page">
    <!-- Header -->
    <div class="history-header">
      <div class="header-left">
        <h3 class="header-title">历史记录</h3>
        <span class="header-count">{{ records.length }} 条</span>
      </div>
      <div class="header-actions">
        <el-switch v-model="deleteFiles" active-text="同时删除文件" size="small" />
        <el-button
          v-if="selectedIds.size > 0"
          type="danger" size="small" plain
          @click="handleBatchDelete"
        >
          删除选中 ({{ selectedIds.size }})
        </el-button>
        <el-button size="small" :loading="loading" @click="loadHistory">刷新</el-button>
        <el-button type="danger" size="small" plain @click="handleClear">清空历史</el-button>
      </div>
    </div>

    <!-- Timeline -->
    <div v-if="records.length > 0" class="timeline-wrap">
      <div
        v-for="(rec, idx) in records"
        :key="rec.id"
        class="timeline-node"
        :style="{ '--accent': getModeConf(rec.mode).color }"
        :class="{ 'is-selected': selectedIds.has(rec.id) }"
      >
        <div class="timeline-dot"></div>
        <div class="timeline-line" v-if="idx < records.length - 1"></div>

        <div class="rec-card"
          :class="{ 'is-selected': selectedIds.has(rec.id) }"
          @click="toggleSelect(rec.id)"
        >
          <div class="card-top">
            <el-checkbox
              :model-value="selectedIds.has(rec.id)"
              @click.stop
              @change="toggleSelect(rec.id)"
              class="card-checkbox"
            />
            <div class="card-meta">
              <el-tag
                :type="getModeConf(rec.mode).type as any"
                size="small"
                class="mode-tag"
              >
                {{ getModeConf(rec.mode).label }}
              </el-tag>
              <span class="card-prompt">{{ rec.prompt }}</span>
            </div>
            <div class="card-actions" @click.stop>
              <el-tooltip :content="rec.time" placement="top">
                <span class="rec-time">{{ relativeTime(rec.time) }}</span>
              </el-tooltip>
              <el-button size="small" type="primary" link class="redo-btn" @click="handleRedo(rec)">
                重做
              </el-button>
              <el-popconfirm
                title="确定删除？"
                confirm-button-text="确定"
                cancel-button-text="取消"
                @confirm="handleDeleteRecord(rec.id)"
              >
                <template #reference>
                  <el-button size="small" type="danger" link class="del-btn">删除</el-button>
                </template>
              </el-popconfirm>
            </div>
          </div>

          <!-- Script record: show result inline -->
          <div v-if="rec.mode === 'script_gen' && rec.extra?.script" class="card-body" @click.stop>
            <div class="script-result">
              <div class="script-result-label">生成的脚本</div>
              <div
                class="script-body"
                :class="{ 'is-collapsed': !expandedScripts.has(rec.id) }"
              >{{ rec.extra!.script as string }}</div>
              <el-button
                v-if="(rec.extra!.script as string).length > 300"
                size="small"
                text
                type="primary"
                class="script-toggle"
                @click="toggleScript(rec.id)"
              >
                {{ expandedScripts.has(rec.id) ? '收起' : '展开全部' }}
              </el-button>
            </div>
          </div>

          <!-- Image / Video record -->
          <div v-else-if="rec.images && rec.images.length > 0" class="card-body" @click.stop>
            <div class="media-grid">
              <template v-for="(img, i) in rec.images.slice(0, 4)" :key="i">
                <div v-if="isImageUrl(img)" class="media-item">
                  <el-image
                    :src="img"
                    :preview-src-list="rec.images"
                    fit="cover"
                    class="media-thumb"
                  />
                  <div v-if="i === 3 && rec.images.length > 4" class="media-overlay">
                    +{{ rec.images.length - 3 }}
                  </div>
                </div>
                <div v-else class="media-item media-video" @click="playVideo(img)">
                  <div class="video-play-btn">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                      <path d="M8 5v14l11-7z" fill="currentColor"/>
                    </svg>
                  </div>
                  <span class="video-label">视频</span>
                </div>
              </template>
            </div>
          </div>

        </div>
      </div>
    </div>

    <el-empty v-else description="暂无历史记录" />

    <!-- Video dialog -->
    <el-dialog v-model="videoDialogVisible" title="视频预览" width="720px" destroy-on-close>
      <video
        v-if="previewVideoUrl"
        :src="previewVideoUrl"
        controls
        autoplay
        class="video-player"
      />
    </el-dialog>
  </div>
</template>

<style scoped>
/* ==================== Layout ==================== */
.history-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 8px 0 40px;
}

/* ==================== Header ==================== */
.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 28px;
  flex-wrap: wrap;
  gap: 12px;
}
.header-left {
  display: flex;
  align-items: baseline;
  gap: 10px;
}
.header-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #1d2129;
}
.header-count {
  font-size: 13px;
  color: #86909c;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

/* ==================== Timeline ==================== */
.timeline-wrap {
  position: relative;
}
.timeline-node {
  position: relative;
  padding-left: 28px;
  margin-bottom: 0;
}
.timeline-dot {
  position: absolute;
  left: 0;
  top: 22px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--accent, #c9cdd4);
  border: 2px solid #fff;
  box-shadow: 0 0 0 2px var(--accent, #c9cdd4);
  z-index: 1;
  transition: transform 0.2s;
}
.timeline-node:hover .timeline-dot {
  transform: scale(1.25);
}
.timeline-line {
  position: absolute;
  left: 5px;
  top: 36px;
  bottom: -8px;
  width: 2px;
  background: linear-gradient(to bottom, var(--accent, #e5e6eb), #e5e6eb 60%);
  opacity: 0.5;
}

/* ==================== Card ==================== */
.rec-card {
  position: relative;
  padding: 16px 18px;
  margin-bottom: 16px;
  border-radius: 10px;
  background: #fff;
  border: 1px solid #e5e6eb;
  border-left: 3px solid var(--accent, #e5e6eb);
  transition: all 0.25s ease;
  cursor: pointer;
}
.rec-card:hover {
  border-color: #c9cdd4;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
  transform: translateY(-1px);
}
.rec-card.is-selected {
  background: linear-gradient(135deg, #f0f5ff 0%, #fafafa 100%);
  border-color: var(--accent, #409eff);
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.12);
}

/* ==================== Card Top Row ==================== */
.card-top {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.card-checkbox {
  margin-top: 2px;
}
.card-meta {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}
.mode-tag {
  flex-shrink: 0;
  font-weight: 500;
}
.card-prompt {
  color: #1d2129;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.5;
}
.card-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}
.rec-time {
  font-size: 12px;
  color: #86909c;
  white-space: nowrap;
}
.del-btn {
  font-size: 12px;
  opacity: 0;
  transition: opacity 0.2s;
}
.redo-btn {
  font-size: 12px;
  opacity: 0;
  transition: opacity 0.2s;
}
.rec-card:hover .del-btn,
.rec-card:hover .redo-btn {
  opacity: 1;
}

/* ==================== Card Body ==================== */
.card-body {
  margin-top: 12px;
  margin-left: 28px;
}

/* ==================== Script Result (inline, no dialog) ==================== */
.script-result {
  background: #fafafa;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  padding: 14px 16px;
}
.script-result-label {
  font-size: 12px;
  font-weight: 500;
  color: #722ed1;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
}
.script-body {
  font-size: 13px;
  color: #1d2129;
  line-height: 1.8;
  white-space: pre-wrap;
  overflow: hidden;
  transition: max-height 0.3s ease;
}
.script-body.is-collapsed {
  max-height: 120px;
  position: relative;
}
.script-body.is-collapsed::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 40px;
  background: linear-gradient(transparent, #fafafa);
  pointer-events: none;
}
.script-toggle {
  margin-top: 8px;
}

/* ==================== Media Grid ==================== */
.media-grid {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.media-item {
  position: relative;
  width: 88px;
  height: 88px;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s;
}
.media-item:hover {
  transform: scale(1.04);
}
.media-thumb {
  width: 100%;
  height: 100%;
  display: block;
}
.media-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  backdrop-filter: blur(2px);
}
.media-video {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f0f2f5, #e5e6eb);
  border: 1px solid #e5e6eb;
  gap: 4px;
}
.video-play-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: rgba(64, 158, 255, 0.15);
  color: #409eff;
  transition: background 0.2s;
}
.media-video:hover .video-play-btn {
  background: rgba(64, 158, 255, 0.25);
}
.video-label {
  font-size: 11px;
  color: #86909c;
}

/* ==================== Video Dialog ==================== */
.video-player {
  width: 100%;
  max-height: 70vh;
  outline: none;
  border-radius: 6px;
}
</style>
