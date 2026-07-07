<script setup lang="ts">
import { computed } from 'vue'
import { VideoCameraFilled, PictureFilled } from '@element-plus/icons-vue'
import type { StoryboardShot } from '../types'

const props = defineProps<{
  shot: StoryboardShot
}>()

const emit = defineEmits<{
  edit: [shot: StoryboardShot]
  delete: [shot: StoryboardShot]
  generate: [shot: StoryboardShot]
  preview: [shot: StoryboardShot]
}>()

const statusConfig: Record<string, { type: string; label: string }> = {
  pending: { type: 'info', label: '待生成' },
  generating: { type: 'primary', label: '生成中' },
  completed: { type: 'success', label: '已完成' },
}

const statusInfo = computed(() => statusConfig[props.shot.status] || { type: 'info', label: props.shot.status })

const typeLabel = computed(() => {
  if (props.shot.type === 'text2video') return '文生视频'
  if (props.shot.type === 'image2video') return '图生视频'
  return props.shot.type
})

const truncatedPrompt = computed(() => {
  if (props.shot.prompt.length <= 50) return props.shot.prompt
  return props.shot.prompt.slice(0, 50) + '...'
})

const isVideo = computed(() => props.shot.type === 'text2video' || props.shot.type === 'image2video')

const hasResult = computed(() => !!props.shot.result_video)
</script>

<template>
  <div class="shot-card" :class="`shot-card--${shot.status}`">
    <div class="shot-card__header">
      <span class="shot-card__seq">#{{ shot.sequence }}</span>
      <el-tag :type="statusInfo.type as any" size="small" effect="dark">
        {{ statusInfo.label }}
      </el-tag>
    </div>

    <div class="shot-card__body">
      <div class="shot-card__type">
        <el-icon :size="20" color="#909399">
          <component :is="isVideo ? VideoCameraFilled : PictureFilled" />
        </el-icon>
        <span class="shot-card__type-label">{{ typeLabel }}</span>
      </div>
      <div class="shot-card__prompt" :title="shot.prompt">
        {{ truncatedPrompt }}
      </div>
    </div>

    <div v-if="hasResult" class="shot-card__result">
      <video
        v-if="isVideo"
        :src="shot.result_video"
        controls
        class="shot-card__video"
      />
      <el-image
        v-else
        :src="shot.result_video"
        fit="cover"
        class="shot-card__image"
      >
        <template #error>
          <div class="shot-card__placeholder">
            <el-icon :size="28"><PictureFilled /></el-icon>
          </div>
        </template>
      </el-image>
    </div>

    <div v-else class="shot-card__preview-placeholder">
      <el-icon :size="36" color="#c0c4cc">
        <component :is="isVideo ? VideoCameraFilled : PictureFilled" />
      </el-icon>
      <span class="shot-card__preview-hint">暂无结果</span>
    </div>

    <div class="shot-card__actions">
      <el-button size="small" text @click="emit('edit', shot)">编辑</el-button>
      <el-button
        v-if="shot.status === 'pending'"
        size="small"
        type="primary"
        text
        @click="emit('generate', shot)"
      >
        生成
      </el-button>
      <el-button
        v-if="hasResult"
        size="small"
        text
        @click="emit('preview', shot)"
      >
        预览
      </el-button>
      <el-button size="small" text type="danger" @click="emit('delete', shot)">删除</el-button>
    </div>
  </div>
</template>

<style scoped>
.shot-card {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  overflow: hidden;
  transition: border-color 0.2s, background 0.2s;
}

.shot-card:hover {
  border-color: #d0d0d0;
}

.shot-card--generating {
  border-color: #409eff;
  background: #f0f7ff;
}

.shot-card--completed {
  border-color: #67c23a;
  background: #f0f9eb;
}

.shot-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px 0;
}

.shot-card__seq {
  font-weight: 700;
  font-size: 15px;
  color: #303133;
}

.shot-card__body {
  padding: 8px 12px;
}

.shot-card__type {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 4px;
}

.shot-card__type-label {
  font-size: 12px;
  color: #909399;
}

.shot-card__prompt {
  font-size: 13px;
  color: #606266;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.shot-card__result {
  margin: 0 12px;
}

.shot-card__video {
  width: 100%;
  border-radius: 4px;
  max-height: 200px;
}

.shot-card__image {
  width: 100%;
  height: 120px;
  border-radius: 4px;
}

.shot-card__preview-placeholder {
  margin: 0 12px;
  height: 100px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: #f5f7fa;
  border-radius: 4px;
}

.shot-card__preview-hint {
  font-size: 12px;
  color: #c0c4cc;
}

.shot-card__placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #c0c4cc;
  background: #f5f7fa;
}

.shot-card__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0;
  padding: 8px 8px 6px;
  border-top: 1px solid var(--border-light);
  margin-top: 8px;
}
</style>
