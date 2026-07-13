<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import type { ProjectFile } from '../types'
import { Download, View, FolderAdd, Link } from '@element-plus/icons-vue'
import { saveAsset } from '../api/assets'

const props = defineProps<{ files: ProjectFile[]; projectId?: number }>()
const emit = defineEmits<{ preview: [file: ProjectFile] }>()

const savingUrls = ref<Set<string>>(new Set())

async function handleSaveToGallery(file: ProjectFile) {
  if (savingUrls.value.has(file.url)) return
  savingUrls.value = new Set([...savingUrls.value, file.url])
  try {
    await saveAsset({
      image_url: file.url,
      prompt: file.prompt,
      mode: file.mode || (file.type === 'video' ? 'text2video' : 'text2image'),
      project_id: props.projectId,
    })
    ElMessage.success('已保存到作品库')
  } catch (e: any) {
    ElMessage.error(e.message || '保存到作品库失败')
  } finally {
    const next = new Set(savingUrls.value)
    next.delete(file.url)
    savingUrls.value = next
  }
}

const activeTab = ref<'all' | 'image' | 'video'>('all')

const filteredFiles = computed(() => {
  if (!props.files) return []
  if (activeTab.value === 'all') return props.files
  return props.files.filter(f => f.type === activeTab.value)
})

const stepLabel: Record<string, string> = {
  generate: '生成',
  refine: '优化',
  finalize: '定稿',
}

function getThumbnailUrl(file: ProjectFile): string {
  if (file.type === 'video') {
    return file.url.replace(/\.(mp4|webm)$/, '.jpg') || '/placeholder-video.png'
  }
  return file.url
}

function downloadFile(url: string) {
  window.open('/api/v1/download?url=' + encodeURIComponent(url), '_blank')
}

function copyUrl(url: string) {
  navigator.clipboard.writeText(url).then(() => {
    ElMessage.success('链接已复制')
  }).catch(() => {})
}
</script>

<template>
  <div class="file-grid-panel">
    <div class="file-grid-header">
      <el-radio-group v-model="activeTab" size="small">
        <el-radio-button value="all">全部 ({{ files.length }})</el-radio-button>
        <el-radio-button value="image">图片 ({{ files.filter(f => f.type === 'image').length }})</el-radio-button>
        <el-radio-button value="video">视频 ({{ files.filter(f => f.type === 'video').length }})</el-radio-button>
      </el-radio-group>
    </div>

    <div v-if="filteredFiles.length === 0" class="file-grid-empty">
      <el-empty description="暂无文件" />
    </div>

    <div v-else class="file-grid">
      <div v-for="file in filteredFiles" :key="`${file.source}-${file.id}`" class="file-card">
        <div class="file-thumb" @click="emit('preview', file)">
          <img :src="getThumbnailUrl(file)" :alt="file.prompt" @error="(e: Event) => (e.target as HTMLImageElement).src = '/placeholder.png'" />
          <span v-if="file.step" class="file-step-tag" :class="file.step">{{ stepLabel[file.step] || file.step }}</span>
          <span v-if="file.type === 'video'" class="file-type-badge">视频</span>
        </div>
        <div class="file-info">
          <p class="file-prompt" :title="file.prompt">{{ file.prompt }}</p>
          <div class="file-actions">
            <el-tooltip content="预览">
              <el-button size="small" circle :icon="View" @click="emit('preview', file)" />
            </el-tooltip>
            <el-tooltip content="下载">
              <el-button size="small" circle :icon="Download" @click="downloadFile(file.url)" />
            </el-tooltip>
            <el-tooltip content="复制链接">
              <el-button size="small" circle :icon="Link" @click="copyUrl(file.url)" />
            </el-tooltip>
            <el-tooltip v-if="file.source === 'history'" content="保存到作品库">
              <el-button
                size="small"
                circle
                type="success"
                :icon="FolderAdd"
                :loading="savingUrls.has(file.url)"
                :disabled="savingUrls.has(file.url)"
                @click="handleSaveToGallery(file)"
              />
            </el-tooltip>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.file-grid-panel {
  margin-top: 8px;
}
.file-grid-header {
  margin-bottom: 16px;
}
.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}
.file-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  overflow: hidden;
  transition: box-shadow 0.2s;
  background: #fff;
}
.file-card:hover {
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
}
.file-thumb {
  position: relative;
  aspect-ratio: 1;
  overflow: hidden;
  cursor: pointer;
  background: var(--el-color-info-light-9);
}
.file-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.file-step-tag {
  position: absolute;
  top: 6px;
  left: 6px;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  color: #fff;
  background: var(--el-color-primary);
}
.file-type-badge {
  position: absolute;
  top: 6px;
  right: 6px;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  color: #fff;
  background: var(--el-color-warning);
}
.file-info {
  padding: 8px 10px;
}
.file-prompt {
  font-size: 12px;
  line-height: 1.4;
  margin: 0 0 8px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  color: var(--el-text-color-secondary);
}
.file-actions {
  display: flex;
  gap: 4px;
  justify-content: flex-end;
}
.file-grid-empty {
  padding: 40px 0;
}
</style>
