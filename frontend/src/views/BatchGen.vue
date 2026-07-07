<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { batchGenerate } from '../api/image'
import ImageResult from '../components/ImageResult.vue'
import { useRedoStore } from '../stores/redo'

const promptsText = ref('')
const size = ref('1024x1024')
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
  if (newData && newData.mode === 'batch') {
    promptsText.value = newData.promptsText || ''
    size.value = newData.size || '1024x1024'
  }
}, { flush: 'sync' })

async function handleGenerate() {
  const prompts = promptsText.value
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
  if (prompts.length === 0) {
    ElMessage.warning('请输入至少一个提示词')
    return
  }
  if (prompts.length > 20) {
    ElMessage.warning('最多支持 20 个提示词')
    return
  }
  loading.value = true
  images.value = []
  try {
    const res = await batchGenerate({ prompts, size: size.value })
    images.value = res.images
  } catch (e: any) {
    ElMessage.error(e.message || '生成失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="gen-page">
    <div class="gen-input">
      <h3 class="gen-title">批量生成</h3>
      <el-form label-width="100px">
        <el-form-item label="提示词列表">
          <el-input
            v-model="promptsText"
            type="textarea"
            :rows="6"
            placeholder="每行一个提示词&#10;例如:&#10;a cute cat&#10;a beautiful landscape&#10;a futuristic city"
          />
        </el-form-item>
        <el-form-item label="尺寸">
          <el-select v-model="size" style="width: 250px">
            <el-option v-for="opt in sizeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" size="large" @click="handleGenerate" style="width: 100%">
            批量生成
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gen-preview">
      <ImageResult :images="images" :loading="loading" />
    </div>
  </div>
</template>

<style scoped>
.gen-page {
  display: flex;
  gap: 24px;
  min-height: 500px;
}
.gen-input {
  flex: 1;
  max-width: 480px;
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
}
.gen-preview {
  flex: 1;
  padding: 20px;
  background: var(--bg-subtle);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  display: flex;
  flex-direction: column;
}
.gen-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-muted);
  margin: 0 0 16px 0;
}
</style>
