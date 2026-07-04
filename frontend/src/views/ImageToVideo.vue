<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, Link } from '@element-plus/icons-vue'
import { createImageToVideo } from '../api/video'
import { useRedoStore } from '../stores/redo'

const inputMode = ref<'upload' | 'url'>('upload')
const prompt = ref('')
const file = ref<File | null>(null)
const imageUrl = ref('')
const filePreviewUrl = ref('')
const duration = ref(5)
const aspectRatio = ref('16:9')
const frameRate = ref(24)
const loading = ref(false)
const taskId = ref('')
const redoStore = useRedoStore()

const durationOptions = [3, 5, 8, 10, 15, 18]
const ratioOptions = ['16:9', '9:16', '1:1', '4:3', '3:4']
const fpsOptions = [12, 24, 30, 60]

const previewUrl = computed(() => {
  if (inputMode.value === 'upload') {
    return filePreviewUrl.value
  }
  return imageUrl.value.trim()
})

// 监听重做数据（flush: sync 确保同步触发）
watch(() => redoStore.redoData, (newData) => {
  if (newData && newData.mode === 'image2video') {
    prompt.value = newData.prompt || ''
    inputMode.value = newData.inputMode || 'url'
    imageUrl.value = newData.imageUrl || ''
    duration.value = newData.duration || 5
    aspectRatio.value = newData.aspectRatio || '16:9'
    frameRate.value = newData.frameRate || 24
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
    ElMessage.warning('请输入提示词')
    return
  }
  loading.value = true
  try {
    const res = await createImageToVideo(
      source,
      prompt.value,
      duration.value,
      aspectRatio.value,
      frameRate.value
    )
    taskId.value = res.taskId
  } catch (e: any) {
    ElMessage.error(e.message || '提交失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
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
          style="max-width: 300px; max-height: 200px; border-radius: 8px; border: 1px solid #e5e6eb"
          :preview-src-list="[previewUrl]"
        />
      </el-form-item>
      <el-form-item label="提示词">
        <el-input
          v-model="prompt"
          type="textarea"
          :rows="3"
          placeholder="描述视频内容..."
        />
      </el-form-item>
      <el-form-item label="时长">
        <el-select v-model="duration" style="width: 120px">
          <el-option v-for="d in durationOptions" :key="d" :label="`${d}秒`" :value="d" />
        </el-select>
      </el-form-item>
      <el-form-item label="宽高比">
        <el-select v-model="aspectRatio" style="width: 120px">
          <el-option v-for="r in ratioOptions" :key="r" :label="r" :value="r" />
        </el-select>
      </el-form-item>
      <el-form-item label="帧率">
        <el-select v-model="frameRate" style="width: 120px">
          <el-option v-for="f in fpsOptions" :key="f" :label="`${f}fps`" :value="f" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" size="large" @click="handleGenerate">
          生成视频
        </el-button>
      </el-form-item>
    </el-form>

    <div v-if="taskId" style="margin-top: 16px">
      <el-alert title="任务已提交" type="success" show-icon>
        <p>任务 ID: {{ taskId }}</p>
      </el-alert>
    </div>
  </div>
</template>
