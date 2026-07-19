<script setup lang="ts">
import { ref } from 'vue'
import { ElButton, ElMessage } from 'element-plus'
import type { Project, ComicPanel } from '../../types'

const props = defineProps<{ project: Project | null; panels: ComicPanel[] }>()
const emit = defineEmits<{
  completed: []
}>()

const coverIndex = ref(0)
const exporting = ref(false)

function selectCover(i: number) {
  coverIndex.value = i
}

async function exportHTML() {
  if (props.panels.length === 0) {
    ElMessage.warning('没有可导出的内容')
    return
  }
  exporting.value = true
  try {
    const panelsHtml = props.panels.map((p, i) => `
      <div class="panel" style="border: 1px solid #ddd; border-radius: 8px; padding: 8px; page-break-inside: avoid;${i === coverIndex.value ? ' border-color: #409eff;' : ''}">
        ${p.image ? `<img src="${p.image}" style="width: 100%; border-radius: 4px;" />` : '<div style="height: 100px; background: #f5f5f5; display: flex; align-items: center; justify-content: center; color: #999;">无图片</div>'}
        <div style="margin-top: 8px; font-size: 12px; line-height: 1.5;">
          <div>第 ${i + 1} 格</div>
          ${p.caption ? `<div style="color: #e6a23c; font-style: italic; margin-top: 4px;">「${p.caption}」</div>` : ''}
        </div>
      </div>
    `).join('')

    const html = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>${props.project?.title || '漫画'} - 导出</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, 'Noto Sans SC', sans-serif; padding: 24px; background: #f5f5f5; }
.comic { max-width: 800px; margin: 0 auto; }
h1 { text-align: center; margin-bottom: 24px; font-size: 24px; }
.grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px; }
.panel { background: #fff; }
@media print {
  body { background: #fff; padding: 0; }
  .grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
</head>
<body>
<div class="comic">
  <h1>${props.project?.title || '漫画'}</h1>
  <div class="grid">
    ${panelsHtml}
  </div>
</div>
</body>
</html>`

    const blob = new Blob([html], { type: 'text/html' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${props.project?.title || 'comic'}.html`
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('导出成功！')
  } catch (e: any) {
    ElMessage.error('导出失败: ' + (e.message || ''))
  } finally {
    exporting.value = false
  }
}

function onComplete() {
  emit('completed')
}
</script>

<template>
  <div class="comic-final">
    <div class="step-intro">
      <h3>预览与定稿</h3>
      <p>查看完整漫画，选择封面图片，导出或标记完成。</p>
    </div>

    <!-- 封面选择 -->
    <div class="section">
      <h4>选择封面</h4>
      <div class="cover-grid">
        <div
          v-for="(panel, i) in panels"
          :key="i"
          class="cover-option"
          :class="{ active: coverIndex === i }"
          @click="selectCover(i)"
        >
          <img v-if="panel.image" :src="panel.image" class="cover-thumb" />
          <div v-else class="cover-placeholder">无图</div>
          <div class="cover-label">格子 {{ i + 1 }}</div>
        </div>
      </div>
    </div>

    <!-- 预览 -->
    <div class="section">
      <h4>漫画预览</h4>
      <div class="preview-grid">
        <div v-for="(panel, i) in panels" :key="i" class="preview-card">
          <img v-if="panel.image" :src="panel.image" class="preview-img" />
          <div v-else class="preview-placeholder">第 {{ i + 1 }} 格</div>
          <div v-if="panel.caption" class="preview-caption">{{ panel.caption }}</div>
        </div>
      </div>
    </div>

    <!-- 操作 -->
    <div class="actions">
      <el-button :loading="exporting" @click="exportHTML">
        {{ exporting ? '导出中...' : '导出 HTML' }}
      </el-button>
      <el-button type="primary" @click="onComplete">
        完成
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.comic-final {
  max-width: 700px;
  margin: 0 auto;
}
.step-intro {
  margin-bottom: 24px;
}
.step-intro h3 {
  margin: 0 0 8px;
}
.step-intro p {
  margin: 0;
  color: #909399;
}
.section {
  margin-bottom: 32px;
}
.section h4 {
  margin: 0 0 12px;
  font-size: 15px;
}
.cover-grid {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.cover-option {
  width: 100px;
  cursor: pointer;
  border: 2px solid transparent;
  border-radius: 8px;
  padding: 4px;
  text-align: center;
  transition: border-color 0.2s;
}
.cover-option.active {
  border-color: #409eff;
}
.cover-thumb {
  width: 100%;
  height: 80px;
  object-fit: cover;
  border-radius: 4px;
}
.cover-placeholder {
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  border-radius: 4px;
  color: #c0c4cc;
  font-size: 12px;
}
.cover-label {
  font-size: 12px;
  margin-top: 4px;
  color: #909399;
}
.preview-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}
.preview-card {
  border: 1px solid #eaeaea;
  border-radius: 8px;
  overflow: hidden;
}
.preview-img {
  width: 100%;
  display: block;
}
.preview-placeholder {
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  color: #c0c4cc;
  font-size: 14px;
}
.preview-caption {
  padding: 8px;
  font-size: 13px;
  color: #e6a23c;
  font-style: italic;
  text-align: center;
  border-top: 1px solid #f0f0f0;
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}
</style>
