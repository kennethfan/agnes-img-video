<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { textToImage } from '../../api/image'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()
const currentChapter = ref(-1)

async function generateIllustration(chapterIndex: number) {
  currentChapter.value = chapterIndex
  const ch = store.novel.chapters[chapterIndex]
  try {
    const prompt = `为小说章节绘制插图。小说主题：${store.novel.theme}。章节：${ch.title}。内容提要：${ch.content.slice(0, 100)}`
    const res = await textToImage({ prompt, size: '1024x1024' })
    ch.illustration = res.images?.[0] || ''
  } catch (e: any) {
    ElMessage.error(`插图生成失败: ${e.message || ''}`)
  } finally {
    currentChapter.value = -1
  }
}
</script>

<template>
  <div class="step">
    <h4>配插图</h4>
    <div v-for="(ch, i) in store.novel.chapters" :key="i" class="chapter-card">
      <div class="chapter-card__header">
        <span class="chapter-card__title">{{ ch.title }}</span>
        <el-button
          size="small"
          :loading="currentChapter === i"
          :disabled="!!ch.illustration"
          @click="generateIllustration(i)"
        >
          {{ ch.illustration ? '已生成' : '生成插图' }}
        </el-button>
      </div>
      <img v-if="ch.illustration" :src="ch.illustration" style="max-width: 200px; border-radius: 8px; margin-top: 8px" />
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="store.nextStep()">下一步：导出</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.chapter-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; margin-bottom: 12px; }
.chapter-card__header { display: flex; justify-content: space-between; align-items: center; }
.chapter-card__title { font-weight: 600; }
</style>
