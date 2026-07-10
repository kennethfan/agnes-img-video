<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, Link } from '@element-plus/icons-vue'
import { submitImageToVideo } from '../api/video'
import { saveAsset } from '../api/assets'
import { connectTaskSSE } from '../utils/sse'
import TaskProgress from '../components/TaskProgress.vue'
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
const errorMsg = ref('')
const resultVideos = ref<string[]>([])
const showProgress = ref(false)
const taskId = ref<number | string>('')
const savingUrls = ref<Set<string>>(new Set())
const redoStore = useRedoStore()
let cleanupSSE: (() => void) | null = null

function downloadVideo(url: string) {
  window.open('/api/v1/download?url=' + encodeURIComponent(url), '_blank')
}

async function handleSaveToGallery(url: string) {
  savingUrls.value = new Set([...savingUrls.value, url])
  try {
    await saveAsset({ image_url: url, prompt: prompt.value, mode: 'image2video' })
    ElMessage.success('已保存到作品库')
  } catch (e: any) {
    ElMessage.error(e.message || '保存到作品库失败')
  } finally {
    const next = new Set(savingUrls.value)
    next.delete(url)
    savingUrls.value = next
  }
}

onUnmounted(() => {
  cleanupSSE?.()
})

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
  if (filePreviewUrl.value) {
    URL.revokeObjectURL(filePreviewUrl.value)
  }
  file.value = uploadFile.raw || null
  if (file.value) {
    filePreviewUrl.value = URL.createObjectURL(file.value)
  } else {
    filePreviewUrl.value = ''
  }
}

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
  errorMsg.value = ''
  resultVideos.value = []
  taskId.value = ''
  showProgress.value = false

  try {
    const res = await submitImageToVideo(
      source,
      prompt.value,
      duration.value,
      aspectRatio.value,
      frameRate.value
    )
    taskId.value = res.taskId
    showProgress.value = true

    cleanupSSE = connectTaskSSE(res.taskId, {
      onProgress: () => {},
      onComplete: (data) => {
        showProgress.value = false
        try {
          resultVideos.value = JSON.parse(data.result)
        } catch {
          resultVideos.value = [data.result]
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
      <h3 class="gen-title">图生视频</h3>
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
          <el-button type="primary" :loading="loading" size="large" @click="handleGenerate" style="width: 100%">
            生成视频
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gen-preview">
      <div v-if="errorMsg" style="padding: 20px">
        <el-alert title="生成失败" :description="errorMsg" type="error" show-icon />
      </div>
      <TaskProgress v-else-if="showProgress && taskId" :task-id="taskId" />
      <div v-else-if="resultVideos.length > 0" style="padding: 20px">
        <div v-for="(video, idx) in resultVideos" :key="idx" style="margin-bottom: 12px">
          <video
            :src="video"
            controls
            style="width: 100%; max-height: 400px; border-radius: var(--radius-sm)"
          />
          <div style="display: flex; gap: 8px; margin-top: 8px">
            <el-button size="small" type="primary" @click="downloadVideo(video)">
              下载
            </el-button>
            <el-button
              size="small"
              type="success"
              :loading="savingUrls.has(video)"
              :disabled="savingUrls.has(video)"
              @click="handleSaveToGallery(video)"
            >
              保存到作品库
            </el-button>
          </div>
        </div>
      </div>
      <div v-else style="padding: 20px; text-align: center; color: var(--text-muted); font-size: 14px">
        上传图片并填写提示词，视频结果将出现在这里
      </div>
    </div>
  </div>
</template>

<style scoped>
.gen-page { display: flex; gap: 24px; min-height: 500px; }
.gen-input { flex: 1; max-width: 480px; padding: 20px; background: var(--bg-card); border: 1px solid var(--border-default); border-radius: var(--radius-card); }
.gen-preview { flex: 1; padding: 20px; background: var(--bg-subtle); border: 1px solid var(--border-default); border-radius: var(--radius-card); display: flex; flex-direction: column; }
.gen-title { font-size: 14px; font-weight: 500; color: var(--text-muted); margin: 0 0 16px 0; }
</style>
