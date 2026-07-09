<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { submitTextToImage } from '../api/image'
import { connectTaskSSE } from '../utils/sse'
import ImageResult from '../components/ImageResult.vue'
import TaskProgress from '../components/TaskProgress.vue'
import { useRedoStore } from '../stores/redo'

const prompt = ref('')
const negativePrompt = ref('')
const size = ref('1024x1024')
const n = ref(1)
const loading = ref(false)
const showProgress = ref(false)
const taskId = ref('')
const images = ref<string[]>([])
const errorMsg = ref('')
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
  errorMsg.value = ''
  images.value = []
  taskId.value = ''
  showProgress.value = false

  try {
    const res = await submitTextToImage({
      prompt: prompt.value,
      size: size.value,
      n: n.value,
      negative_prompt: negativePrompt.value || undefined,
    })
    taskId.value = res.taskId
    showProgress.value = true

    connectTaskSSE(res.taskId, {
      onComplete: (data) => {
        showProgress.value = false
        try {
          images.value = JSON.parse(data.result)
        } catch {
          images.value = data.result ? [data.result] : []
        }
        loading.value = false
      },
      onError: (data) => {
        showProgress.value = false
        errorMsg.value = data.error
        loading.value = false
      },
    })
  } catch (e: any) {
    errorMsg.value = e.message || '提交失败'
    loading.value = false
  }
}
</script>

<template>
  <div class="gen-page">
    <div class="gen-input">
      <h3 class="gen-title">文生图</h3>
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
          <el-button type="primary" :loading="loading" size="large" @click="handleGenerate" style="width: 100%">
            生成图片
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gen-preview">
      <TaskProgress v-if="showProgress && taskId" :task-id="taskId" @error="errorMsg = $event" />
      <el-alert v-if="errorMsg" type="error" :description="errorMsg" show-icon closable class="error-alert" />
      <ImageResult :images="images" :loading="loading && !showProgress" />
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
.error-alert {
  margin-bottom: 12px;
}
</style>
