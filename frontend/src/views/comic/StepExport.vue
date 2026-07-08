<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElMessage, ElRadioGroup, ElRadioButton } from 'element-plus'

const store = useWizardStore()
const format = ref('html')

function exportComic() {
  if (format.value === 'html') {
    let html = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>${store.comic.theme}</title><style>body{font-family:sans-serif;max-width:800px;margin:0 auto;padding:20px}.grid{display:grid;grid-template-columns:repeat(2,1fr);gap:16px}.panel{border:1px solid #eaeaea;border-radius:12px;padding:12px;text-align:center}.panel img{width:100%;border-radius:8px}.caption{margin-top:8px;font-size:14px;color:#333}</style></head><body><h1>${store.comic.theme}</h1><div class="grid">`
    store.comic.panels.forEach(p => {
      html += `<div class="panel"><img src="${p.image}"><div class="caption">${p.caption || ''}</div></div>`
    })
    html += `</div></body></html>`
    const blob = new Blob([html], { type: 'text/html' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = `${store.comic.theme}.html`; a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } else {
    ElMessage.info('其他格式导出待实现')
  }
}
</script>

<template>
  <div class="step" style="text-align: center; padding: 40px 0">
    <h4>漫画完成！</h4>
    <div class="export-preview">
      <div v-for="panel in store.comic.panels" :key="panel.prompt" class="export-panel">
        <img :src="panel.image" style="width: 100%; border-radius: 8px" />
        <div v-if="panel.caption" class="export-caption">{{ panel.caption }}</div>
      </div>
    </div>
    <div style="margin-top: 16px">
      <label>导出格式：</label>
      <el-radio-group v-model="format">
        <el-radio-button value="html">HTML</el-radio-button>
        <el-radio-button value="png">PNG</el-radio-button>
      </el-radio-group>
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="exportComic">导出</el-button>
    <el-button style="margin-top: 16px; margin-left: 8px" @click="store.reset()">重新开始</el-button>
  </div>
</template>

<style scoped>
.step h4 { margin: 0 0 16px; font-size: 16px; }
.export-preview { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; max-width: 500px; margin: 0 auto; }
.export-panel { border: 1px solid #eaeaea; border-radius: 12px; padding: 8px; }
.export-caption { margin-top: 8px; font-size: 13px; color: #333; text-align: center; }
</style>
