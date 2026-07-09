<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, Link } from '@element-plus/icons-vue'
import { submitImageToImage } from '../api/image'
import { connectTaskSSE } from '../utils/sse'
import ImageResult from '../components/ImageResult.vue'
import TaskProgress from '../components/TaskProgress.vue'
import { useRedoStore } from '../stores/redo'

const inputMode = ref<'upload' | 'url'>('upload')
const prompt = ref('')
const negativePrompt = ref('')
const size = ref('1024x1024')
const strength = ref(0.75)
const loading = ref(false)
const showProgress = ref(false)
const taskId = ref('')
const images = ref<string[]>([])
const errorMsg = ref('')
const file = ref<File | null>(null)
const imageUrl = ref('')
const filePreviewUrl = ref('')
const redoStore = useRedoStore()

const sizeOptions = [
  { value: '1024x1024', label: '1024x1024 (1:1)' },
  { value: '1024x1792', label: '1024x1792 (9:16)' },
  { value: '1792x1024', label: '1792x1024 (16:9)' },
]

const previewUrl = computed(() => {
  if (inputMode.value === 'upload') {
    return filePreviewUrl.value
  }
  return imageUrl.value.trim()
})

// 监听重做数据（flush: sync 确保同步触发）
watch(() => redoStore.redoData, (newData) => {
  if (newData && newData.mode === 'image2image') {
    prompt.value = newData.prompt || ''
    negativePrompt.value = newData.negativePrompt || ''
    size.value = newData.size || '1024x1024'
    strength.value = newData.strength || 0.75
    inputMode.value = newData.inputMode || 'url'
    imageUrl.value = newData.imageUrl || ''
  }
}, { flush: 'sync' })

function handleFileChange(uploadFile: any) {
  // 清理旧的预览URL
  if (filePreviewUrl.value) {
    URL.revokeObjectURL(filePreviewUrl.value)
  }
  file.value = uploadFile.raw || null
  // 创建新的预览URL
  if (file.value) {
    filePreviewUrl.value = URL.createObjectURL(file.value)
  } else {
    filePreviewUrl.value = ''
  }
}

// 切换输入模式时清理预览
watch(inputMode, () => {
  if (inputMode.value === 'url' && filePreviewUrl.value) {
    URL.revokeObjectURL(filePreviewUrl.value)
    filePreviewUrl.value = ''
  }
})

async function handleGenerate() {
  const source = inputMode.value === 'upload' ? file.value : imageUrl.value.trim()
  if (!source) {
    ElMessage.warning(inputMode.value === 'upload' ? '请上传图片' : '请输入图片 URL')
    return
  }
  if (!prompt.value.trim()) {
    ElMessage.warning('请输入风格描述')
    return
  }
  loading.value = true
  errorMsg.value = ''
  images.value = []
  taskId.value = ''
  showProgress.value = false

  try {
    const res = await submitImageToImage(
      source,
      prompt.value,
      size.value,
      strength.value,
      negativePrompt.value
    )
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
      <h3 class="gen-title">图生图</h3>
      <el-form label-width="100px">
        <el-form-item label="输入方式">
          <el-radio-group v-model="inputMode">
            <el-radio-button value="upload">
              <el-icon style="vertical-align: middle"><upload-filled /></el-icon>
              <span style="vertical-align: middle">上传图片</span>
            </el-radio-button>
            <el-radio-button value="url">
              <el-icon style="vertical-align: middle"><Link /></el-icon>
              <span style="vertical-align: middle">图片 URL</span>
            </el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="inputMode === 'upload'" label="上传图片">
          <el-upload
            drag
            accept="image/*"
            :auto-upload="false"
            :limit="1"
            :on-change="handleFileChange"
          >
            <el-icon class="el-icon--upload" style="font-size: 48px">
              <upload-filled />
            </el-icon>
            <div class="el-upload__text">拖拽图片到此处或 <em>点击上传</em></div>
          </el-upload>
        </el-form-item>

        <el-form-item v-if="inputMode === 'url'" label="图片 URL">
          <el-input
            v-model="imageUrl"
            placeholder="请输入图片公网 URL，如 https://example.com/image.png"
            clearable
          />
        </el-form-item>

        <el-form-item v-if="previewUrl" label="预览">
          <el-image
            :src="previewUrl"
            fit="contain"
            style="max-width: 100%; max-height: 200px; border-radius: var(--radius-sm); border: 1px solid var(--border-default)"
            :preview-src-list="[previewUrl]"
          />
        </el-form-item>
        <el-form-item label="风格描述">
          <el-input
            v-model="prompt"
            type="textarea"
            :rows="3"
            placeholder="描述你想要的风格变化..."
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
            <el-option v-for="opt in sizeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="重绘强度">
          <el-slider v-model="strength" :min="0" :max="1" :step="0.05" style="width: 100%" />
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
