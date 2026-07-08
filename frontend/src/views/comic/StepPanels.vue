<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { generateComicPrompts } from '../../api/comic'
import { ElButton, ElInput, ElMessage } from 'element-plus'

const store = useWizardStore()
const generating = ref(false)

function proceed() {
  const empty = store.comic.panels.some(p => !p.prompt)
  if (empty) { ElMessage.warning('请填写所有分格的提示词'); return }
  store.nextStep()
}

async function aiGenerate() {
  if (!store.comic.theme) {
    ElMessage.warning('请先在"设定主题"步骤填写漫画主题')
    return
  }
  generating.value = true
  try {
    const prompts = await generateComicPrompts(
      store.comic.theme,
      store.comic.layout,
      store.comic.panels.length
    )
    for (let i = 0; i < store.comic.panels.length; i++) {
      if (prompts[i]) {
        store.comic.panels[i].prompt = prompts[i]
      }
    }
    ElMessage.success('AI 提示词已生成')
  } catch (e: any) {
    ElMessage.error(e.message || 'AI 生成失败')
  } finally {
    generating.value = false
  }
}
</script>

<template>
    <div class="step">
    <div class="step__header">
      <h4>填写每格提示词</h4>
      <el-button type="warning" :loading="generating" @click="aiGenerate" size="small">
        {{ generating ? '生成中...' : 'AI 生成提示词' }}
      </el-button>
    </div>
    <div class="panels-grid" :style="{ gridTemplateColumns: store.comic.layout === 'six' ? 'repeat(2, 1fr)' : 'repeat(2, 1fr)' }">
      <div v-for="(panel, i) in store.comic.panels" :key="i" class="panel-card">
        <div class="panel-card__header">第 {{ i + 1 }} 格</div>
        <el-input v-model="panel.prompt" type="textarea" :rows="3" placeholder="描述这个格子的画面内容" />
      </div>
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="proceed">下一步：批量生成</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step__header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.step__header h4 { margin: 0; font-size: 16px; }
.panels-grid { display: grid; gap: 16px; }
.panel-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; }
.panel-card__header { font-weight: 600; font-size: 13px; margin-bottom: 8px; }
</style>
