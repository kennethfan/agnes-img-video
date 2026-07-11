<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Edit } from '@element-plus/icons-vue'
import { imageToImage } from '../api/image'
import ImageResult from './ImageResult.vue'
import type { Project } from '../types'

const props = defineProps<{
  project: Project | null
  defaultImageUrl?: string
}>()

const sourceImage = ref('')
const prompt = ref('')

watch(() => props.defaultImageUrl, (url) => {
  if (url) {
    sourceImage.value = url
  }
})
const strength = ref(0.6)
const size = ref('1024x1024')

const refining = ref(false)
const resultUrls = ref<string[]>([])

async function refine() {
  if (!sourceImage.value) {
    ElMessage.warning('请输入源图片 URL')
    return
  }
  if (!prompt.value.trim()) {
    ElMessage.warning('请输入优化提示词')
    return
  }
  refining.value = true
  resultUrls.value = []
  try {
    const resp = await imageToImage(sourceImage.value, prompt.value, size.value, strength.value)
    if (resp.images?.length) {
      resultUrls.value = resp.images
    }
  } catch (e: any) {
    ElMessage.error('优化失败: ' + (e.message || ''))
  } finally {
    refining.value = false
  }
}
</script>

<template>
  <div class="refine-step">
    <div class="step-intro">
      <h3><el-icon><Edit /></el-icon> 图生图优化</h3>
      <p>对生成的图片进一步调整，通过变化强度控制与原图的差异程度。</p>
    </div>

    <div class="refine-form">
      <el-input
        v-model="sourceImage"
        placeholder="源图片 URL（从生成结果复制或输入）"
      />

      <el-input
        v-model="prompt"
        type="textarea"
        :rows="3"
        placeholder="描述希望调整的方向..."
      />

      <div class="form-row">
        <el-select v-model="size" style="width: 140px">
          <el-option label="1024x1024" value="1024x1024" />
          <el-option label="1152x768" value="1152x768" />
          <el-option label="768x1152" value="768x1152" />
        </el-select>
        <div class="slider-group">
          <span>强度:</span>
          <el-slider
            v-model="strength"
            :min="0"
            :max="1"
            :step="0.05"
            style="width: 200px"
            show-input
          />
        </div>
      </div>

      <el-button type="primary" :loading="refining" :icon="Edit" @click="refine">
        {{ refining ? '优化中...' : '开始优化' }}
      </el-button>
    </div>

    <div v-if="resultUrls.length" class="result-section">
      <h4>优化结果</h4>
      <ImageResult :images="resultUrls" :loading="false" prompt="" mode="" />
    </div>
  </div>
</template>

<style scoped>
.refine-step {
  max-width: 700px;
  margin: 0 auto;
}
.step-intro {
  margin-bottom: 24px;
}
.step-intro h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 8px;
}
.step-intro p {
  margin: 0;
  color: #909399;
}
.refine-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.form-row {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
}
.slider-group {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #606266;
}
.result-section {
  margin-top: 32px;
}
.result-section h4 {
  margin-bottom: 12px;
}
</style>
