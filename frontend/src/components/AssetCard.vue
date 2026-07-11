<script setup lang="ts">
import { computed } from 'vue'
import { StarFilled, Star, PictureFilled, VideoCameraFilled } from '@element-plus/icons-vue'
import type { AssetItem } from '../types'

const props = defineProps<{
  asset: AssetItem
  selected: boolean
}>()

const emit = defineEmits<{
  'toggle-favorite': []
  'toggle-select': []
  'refine': []
  click: []
}>()

const modeLabels: Record<string, string> = {
  text2image: '文生图',
  image2image: '图生图',
  batch: '批量',
  text2video: '文生视频',
  image2video: '图生视频',
  multi_image_video: '多图视频',
}

const modeColors: Record<string, string> = {
  text2image: '#409eff',
  image2image: '#67c23a',
  batch: '#e6a23c',
  text2video: '#f56c6c',
  image2video: '#f56c6c',
  multi_image_video: '#f56c6c',
}

const modeLabel = computed(() => modeLabels[props.asset.mode] || props.asset.mode)

const modeColor = computed(() => modeColors[props.asset.mode] || '#909399')

const truncatedPrompt = computed(() => {
  if (props.asset.prompt.length <= 40) return props.asset.prompt
  return props.asset.prompt.slice(0, 40) + '...'
})

const formattedTime = computed(() => props.asset.time.slice(5, 16))
</script>

<template>
  <div
    class="asset-card"
    :class="{ 'asset-card--selected': selected }"
    @click="emit('click')"
  >
    <div class="asset-card__select" @click.stop="emit('toggle-select')">
      <el-checkbox :model-value="selected" />
    </div>

    <div class="asset-card__favorite" @click.stop="emit('toggle-favorite')">
      <el-icon :color="asset.favorite ? '#e6a23c' : '#c0c4cc'" :size="18">
        <component :is="asset.favorite ? StarFilled : Star" />
      </el-icon>
    </div>

    <div class="asset-card__thumb">
      <el-image
        v-if="asset.type === 'image' && asset.thumbnail"
        :src="asset.thumbnail"
        fit="cover"
        style="width: 100%; height: 100%"
      >
        <template #error>
          <div class="asset-card__placeholder">
            <el-icon :size="32"><PictureFilled /></el-icon>
          </div>
        </template>
      </el-image>

      <div v-else class="asset-card__placeholder">
        <el-icon :size="32">
          <component :is="asset.type === 'video' ? VideoCameraFilled : PictureFilled" />
        </el-icon>
      </div>
    </div>

    <div class="asset-card__badge" :style="{ background: modeColor }">
      {{ modeLabel }}
    </div>

    <div class="asset-card__prompt" :title="asset.prompt">
      {{ truncatedPrompt }}
    </div>

    <div class="asset-card__time">
      {{ formattedTime }}
    </div>
    <div class="asset-card__actions" @click.stop>
      <el-button size="small" type="primary" @click="emit('refine')">精修</el-button>
    </div>
  </div>
</template>

<style scoped>
.asset-card {
  position: relative;
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  overflow: hidden;
  cursor: pointer;
  transition: border-color 0.2s;
}

.asset-card:hover {
  border-color: #d0d0d0;
}

.asset-card--selected {
  border-color: #409eff;
}

.asset-card__select {
  position: absolute;
  top: 8px;
  left: 8px;
  z-index: 2;
}

.asset-card__favorite {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 2;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.8);
  transition: background 0.2s;
}

.asset-card__favorite:hover {
  background: rgba(255, 255, 255, 1);
}

.asset-card__thumb {
  width: 100%;
  height: 160px;
  overflow: hidden;
  background: #f5f7fa;
}

.asset-card__placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #c0c4cc;
}

.asset-card__badge {
  position: absolute;
  top: 34px;
  right: 8px;
  z-index: 2;
  font-size: 11px;
  color: #fff;
  padding: 2px 6px;
  border-radius: 4px;
  line-height: 1.4;
}

.asset-card__prompt {
  padding: 8px 10px 4px;
  font-size: 13px;
  color: #303133;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.asset-card__time {
  padding: 0 10px 4px;
  font-size: 12px;
  color: #909399;
}

.asset-card__actions {
  padding: 0 10px 8px;
}
</style>
