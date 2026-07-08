<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { expandIdea } from '../../api/ideas'
import { ElButton, ElInput } from 'element-plus'

const store = useWizardStore()
const editing = ref(false)
const editedOutline = ref(store.novel.outline)
const reloading = ref(false)

async function regenerate() {
  reloading.value = true
  try {
    const charDesc = store.novel.characters.map(c => `${c.name}（${c.personality}，${c.appearance}）`).join('；')
    const prompt = `请重新为一部小说创作大纲。主题：${store.novel.theme}。风格：${store.novel.genre}。角色：${charDesc}。请提供全新的章节结构。用中文回复。`
    const result = await expandIdea(prompt, '', 'zh')
    editedOutline.value = result || ''
    store.novel.outline = editedOutline.value
  } finally {
    reloading.value = false
  }
}

function confirm() {
  store.novel.outline = editedOutline.value
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>确认大纲</h4>
    <el-button text @click="editing = !editing">{{ editing ? '预览' : '手动编辑' }}</el-button>
    <el-button text :loading="reloading" @click="regenerate">重新生成</el-button>
    <div v-if="editing">
      <el-input v-model="editedOutline" type="textarea" :rows="12" />
    </div>
    <div v-else class="outline-content">{{ editedOutline }}</div>
    <el-button type="primary" style="margin-top: 16px" @click="confirm">确认，开始创作</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.outline-content { white-space: pre-wrap; background: #fafafa; border-radius: 8px; padding: 16px; margin-top: 16px; line-height: 1.8; font-size: 14px; }
</style>
