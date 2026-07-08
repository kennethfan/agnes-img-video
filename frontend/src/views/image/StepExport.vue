<script setup lang="ts">
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()

function downloadImage(url: string) {
  const a = document.createElement('a')
  a.href = url
  a.download = `refined_${Date.now()}.png`
  a.click()
  ElMessage.success('下载已开始')
}

function startNew() {
  store.reset()
}
</script>

<template>
  <div class="step" style="text-align: center; padding: 40px 0">
    <h4>精修完成</h4>
    <img :src="store.image.resultImage" style="max-width: 400px; border-radius: 8px; margin: 16px auto" />
    <div style="display: flex; gap: 12px; justify-content: center; margin-top: 16px">
      <el-button type="primary" @click="downloadImage(store.image.resultImage)">下载图片</el-button>
      <el-button @click="startNew">重新开始</el-button>
    </div>
  </div>
</template>

<style scoped>
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
