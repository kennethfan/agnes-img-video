<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { textToImage } from '../../api/image'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()
const generating = ref(false)
const currentIndex = ref(-1)

async function generateAll() {
  generating.value = true
  for (let i = 0; i < store.comic.panels.length; i++) {
    currentIndex.value = i
    const panel = store.comic.panels[i]
    if (panel.image) continue
    try {
      const res = await textToImage({ prompt: panel.prompt, size: '1024x1024' })
      panel.image = res.images?.[0] || ''
    } catch (e: any) {
      ElMessage.error(`第 ${i + 1} 格生成失败: ${e.message || ''}`)
    }
  }
  currentIndex.value = -1
  generating.value = false
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>批量生成</h4>
    <div class="panels-grid" :style="{ gridTemplateColumns: 'repeat(2, 1fr)' }">
      <div v-for="(panel, i) in store.comic.panels" :key="i" class="panel-card" :class="{ generating: currentIndex === i }">
        <div class="panel-card__header">第 {{ i + 1 }} 格</div>
        <div class="panel-card__prompt">{{ panel.prompt }}</div>
        <img v-if="panel.image" :src="panel.image" style="width: 100%; border-radius: 8px; margin-top: 8px" />
        <div v-else class="panel-card__placeholder">{{ currentIndex === i ? '生成中...' : '待生成' }}</div>
      </div>
    </div>
    <el-button type="primary" :loading="generating" style="margin-top: 16px" @click="generateAll">
      {{ generating ? `正在生成第 ${currentIndex + 1}/${store.comic.panels.length} 格...` : '全部生成' }}
    </el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.panels-grid { display: grid; gap: 16px; }
.panel-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; }
.panel-card.generating { border-color: #409eff; background: #f0f7ff; }
.panel-card__header { font-weight: 600; font-size: 13px; margin-bottom: 4px; }
.panel-card__prompt { font-size: 12px; color: #666; margin-bottom: 8px; }
.panel-card__placeholder { height: 100px; display: flex; align-items: center; justify-content: center; background: #fafafa; border-radius: 8px; color: #c0c4cc; font-size: 13px; }
</style>
