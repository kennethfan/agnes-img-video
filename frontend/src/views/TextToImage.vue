<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { textToImage } from '../api/image'
import ImageResult from '../components/ImageResult.vue'
import { useRedoStore } from '../stores/redo'

const prompt = ref('')
const negativePrompt = ref('')
const size = ref('1024x1024')
const n = ref(1)
const loading = ref(false)
const images = ref<string[]>([])
const redoStore = useRedoStore()

const sizeOptions = [
  { value: '1024x1024', label: '1024x1024 (1:1)' },
  { value: '1024x1792', label: '1024x1792 (9:16)' },
  { value: '1792x1024', label: '1792x1024 (16:9)' },
]

// 监听重做数据（flush: sync 确保同步触发）
watch(() => redoStore.redoData, (newData) => {
  if (newData && newData.mode === 'text2image') {
    prompt.value = newData.prompt || ''
    negativePrompt.value = newData.negativePrompt || ''
    size.value = newData.size || '1024x1024'
    n.value = newData.n || 1
  }
}, { flush: 'sync' })

async function handleGenerate() {
  if (!prompt.value.trim()) {
    ElMessage.warning('请输入提示词')
    return
  }
  loading.value = true
  images.value = []
  try {
    const res = await textToImage({
      prompt: prompt.value,
      size: size.value,
      n: n.value,
      negative_prompt: negativePrompt.value,
    })
    images.value = res.images
  } catch (e: any) {
    ElMessage.error(e.message || '生成失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <el-form label-width="100px">
      <el-form-item label="提示词">
        <el-input
          v-model="prompt"
          type="textarea"
          :rows="3"
          placeholder="描述你想要生成的图片..."
        />
      </el-form-item>
      <el-form-item label="负面提示词">
        <el-input
          v-model="negativePrompt"
          type="textarea"
          :rows="2"
          placeholder="不想出现在图片中的内容..."
        />
      </el-form-item>
      <el-form-item label="尺寸">
        <el-select v-model="size" style="width: 250px">
          <el-option
            v-for="opt in sizeOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="数量">
        <el-input-number v-model="n" :min="1" :max="4" />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" size="large" @click="handleGenerate">
          生成图片
        </el-button>
      </el-form-item>
    </el-form>

    <ImageResult :images="images" :loading="loading" />
  </div>
</template>
