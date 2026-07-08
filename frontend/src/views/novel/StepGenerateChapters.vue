<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { expandIdea } from '../../api/ideas'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()
const generating = ref(false)

async function generateNextChapter() {
  generating.value = true
  const chapterNum = store.novel.chapters.length + 1
  const previousContent = store.novel.chapters.map(c => c.title + '\n' + c.content).join('\n\n')
  try {
    const prompt = `继续写小说「${store.novel.theme}」（${store.novel.genre}风格）。大纲：${store.novel.outline}。已写内容：\n${previousContent || '尚未开始'}\n\n请写第${chapterNum}章，包含标题和正文。用中文回复。`
    const result = await expandIdea(prompt, '', 'zh')
    const text = result || ''
    const lines = text.split('\n')
    const title = lines[0]?.replace(/^#+\s*/, '') || `第${chapterNum}章`
    const content = lines.slice(1).join('\n').trim()
    store.novel.chapters.push({ title, content })
  } catch (e: any) {
    ElMessage.error('生成章节失败: ' + (e.message || ''))
  } finally {
    generating.value = false
  }
}
</script>

<template>
  <div class="step">
    <h4>逐章生成</h4>
    <div v-for="(ch, i) in store.novel.chapters" :key="i" class="chapter-card">
      <div class="chapter-card__title">{{ ch.title }}</div>
      <div class="chapter-card__content">{{ ch.content.slice(0, 200) }}...</div>
    </div>
    <el-button type="primary" :loading="generating" style="margin-top: 16px" @click="generateNextChapter">
      {{ generating ? '生成中...' : `生成第 ${store.novel.chapters.length + 1} 章` }}
    </el-button>
    <el-button v-if="store.novel.chapters.length >= 3" style="margin-top: 16px; margin-left: 8px" @click="store.nextStep()">
      下一步：配插图
    </el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.chapter-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; margin-bottom: 12px; }
.chapter-card__title { font-weight: 600; margin-bottom: 8px; }
.chapter-card__content { font-size: 13px; color: #666; line-height: 1.6; }
</style>
