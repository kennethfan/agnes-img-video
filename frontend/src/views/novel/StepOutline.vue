<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { expandIdea } from '../../api/ideas'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()
const loading = ref(false)
const outline = ref(store.novel.outline)

async function generateOutline() {
  loading.value = true
  try {
    const charDesc = store.novel.characters.map(c => `${c.name}（${c.personality}，${c.appearance}）`).join('；')
    const prompt = `请为一部小说创作详细的大纲。主题：${store.novel.theme}。风格：${store.novel.genre}。角色：${charDesc}。请包含：故事背景、主要冲突、章节概要（5-8章）。用中文回复。`
    const result = await expandIdea(prompt, '', 'zh')
    outline.value = result || ''
    store.novel.outline = outline.value
  } catch (e: any) {
    ElMessage.error('生成大纲失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function proceed() {
  if (!outline.value) { ElMessage.warning('请先生成大纲'); return }
  store.novel.outline = outline.value
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>大纲生成</h4>
    <el-button :loading="loading" @click="generateOutline">生成大纲</el-button>
    <div v-if="outline" class="outline-content">{{ outline }}</div>
    <el-button v-if="outline" type="primary" style="margin-top: 16px" @click="proceed">确认大纲</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.outline-content { white-space: pre-wrap; background: #fafafa; border-radius: 8px; padding: 16px; margin-top: 16px; line-height: 1.8; font-size: 14px; }
</style>
