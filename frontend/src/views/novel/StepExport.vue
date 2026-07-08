<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElMessage, ElRadioGroup, ElRadioButton } from 'element-plus'

const store = useWizardStore()
const format = ref('markdown')

function exportNovel() {
  let content = ''
  if (format.value === 'markdown') {
    content = `# ${store.novel.theme}\n\n**风格：** ${store.novel.genre}\n\n---\n\n`
    store.novel.chapters.forEach(ch => {
      content += `## ${ch.title}\n\n${ch.content}\n\n`
      if (ch.illustration) content += `![插图](${ch.illustration})\n\n`
    })
  } else {
    content = `${store.novel.theme}\n${'='.repeat(store.novel.theme.length)}\n\n`
    store.novel.chapters.forEach(ch => {
      content += `${ch.title}\n${'-'.repeat(ch.title.length)}\n\n${ch.content}\n\n`
    })
  }
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = `${store.novel.theme}.${format.value === 'markdown' ? 'md' : 'txt'}`; a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('导出成功')
}
</script>

<template>
  <div class="step" style="text-align: center; padding: 40px 0">
    <h4>小说完成！</h4>
    <p style="color: #666; margin: 8px 0">共 {{ store.novel.chapters.length }} 章</p>
    <div style="margin-top: 16px">
      <el-radio-group v-model="format">
        <el-radio-button value="markdown">Markdown</el-radio-button>
        <el-radio-button value="text">纯文本</el-radio-button>
      </el-radio-group>
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="exportNovel">导出</el-button>
    <el-button style="margin-top: 16px; margin-left: 8px" @click="store.reset()">重新开始</el-button>
  </div>
</template>

<style scoped>
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
