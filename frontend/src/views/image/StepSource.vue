<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { textToImage } from '../../api/image'
import { ElMessage, ElButton, ElInput, ElRadioGroup, ElRadioButton } from 'element-plus'

const store = useWizardStore()
const sourceMode = ref<'generate' | 'upload'>('generate')
const prompt = ref('')
const loading = ref(false)

async function handleGenerate() {
  if (!prompt.value) { ElMessage.warning('请输入提示词'); return }
  loading.value = true
  try {
    const res = await textToImage({ prompt: prompt.value, size: store.image.size })
    store.image.sourceImage = res.images?.[0] || ''
    store.image.sourcePrompt = prompt.value
    store.image.refinePrompt = prompt.value
  } catch (e: any) {
    ElMessage.error('生成失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function handleFileUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  store.image.sourceFile = file
  const reader = new FileReader()
  reader.onload = () => {
    store.image.sourceImage = reader.result as string
  }
  reader.readAsDataURL(file)
  store.image.sourceType = 'upload'
}

function proceedWithUpload() {
  if (!store.image.sourceImage) { ElMessage.warning('请先上传图片'); return }
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>选择图片来源</h4>
    <el-radio-group v-model="sourceMode">
      <el-radio-button value="generate">文生图</el-radio-button>
      <el-radio-button value="upload">上传图片</el-radio-button>
    </el-radio-group>

    <div v-if="sourceMode === 'generate'" style="margin-top: 16px">
      <el-input v-model="prompt" type="textarea" :rows="4" placeholder="描述你想生成的图片内容..." />
      <div style="display: flex; gap: 8px; margin-top: 12px">
        <el-button type="primary" :loading="loading" @click="handleGenerate">
          生成图片
        </el-button>
        <el-button v-if="store.image.sourceImage" type="success" @click="store.nextStep()">
          下一步：精修
        </el-button>
      </div>
      <img v-if="store.image.sourceImage" :src="store.image.sourceImage" style="max-width: 300px; margin-top: 12px; border-radius: 8px" />
    </div>

    <div v-else style="margin-top: 16px">
      <input type="file" accept="image/*" @change="handleFileUpload" />
      <img v-if="store.image.sourceImage" :src="store.image.sourceImage" style="max-width: 300px; margin-top: 12px; border-radius: 8px" />
      <el-button v-if="store.image.sourceImage" type="primary" style="margin-top: 12px" @click="proceedWithUpload">
        下一步：精修
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
