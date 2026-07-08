<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { imageToImage } from '../../api/image'
import { ElMessage, ElButton, ElInput, ElSlider, ElSelect, ElOption } from 'element-plus'

const store = useWizardStore()
const loading = ref(false)

async function handleRefine() {
  loading.value = true
  try {
    const source = store.image.sourceFile || store.image.sourceImage
    if (!source) {
      ElMessage.warning('缺少源图片')
      return
    }
    const res = await imageToImage(
      source,
      store.image.refinePrompt,
      store.image.size,
      store.image.strength,
      '',
    )
    store.image.resultImage = res.images?.[0] || ''
    store.nextStep()
  } catch (e: any) {
    ElMessage.error('精修失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="step refine-layout">
    <div class="refine-layout__preview">
      <h4>原图</h4>
      <img :src="store.image.sourceImage" style="max-width: 100%; border-radius: 8px" />
    </div>
    <div class="refine-layout__params">
      <h4>精修参数</h4>
      <div style="margin-bottom: 16px">
        <label>提示词</label>
        <el-input v-model="store.image.refinePrompt" type="textarea" :rows="3" />
      </div>
      <div style="margin-bottom: 16px">
        <label>强度 ({{ store.image.strength }})</label>
        <el-slider v-model="store.image.strength" :min="0" :max="1" :step="0.05" />
      </div>
      <div style="margin-bottom: 16px">
        <label>尺寸</label>
        <el-select v-model="store.image.size">
          <el-option label="1024x1024" value="1024x1024" />
          <el-option label="768x768" value="768x768" />
          <el-option label="512x512" value="512x512" />
        </el-select>
      </div>
      <el-button type="primary" :loading="loading" @click="handleRefine">开始精修</el-button>
    </div>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.refine-layout { display: flex; gap: 24px; }
.refine-layout__preview { flex: 1; }
.refine-layout__params { width: 320px; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
label { display: block; font-size: 13px; color: #666; margin-bottom: 6px; }
</style>
