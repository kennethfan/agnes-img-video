<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { submitMultiImageVideo } from '../api/video'
import { connectTaskSSE } from '../utils/sse'
import TaskProgress from '../components/TaskProgress.vue'
import { useRedoStore } from '../stores/redo'

const prompt = ref('')
const imageUrlsText = ref('')
const mode = ref('ti2vid')
const duration = ref(5)
const aspectRatio = ref('16:9')
const frameRate = ref(24)
const loading = ref(false)
const errorMsg = ref('')
const resultVideos = ref<string[]>([])
const showProgress = ref(false)
const taskId = ref('')
const redoStore = useRedoStore()
let cleanupSSE: (() => void) | null = null

onUnmounted(() => {
  cleanupSSE?.()
})

const durationOptions = [3, 5, 8, 10, 15, 18]
const ratioOptions = ['16:9', '9:16', '1:1', '4:3', '3:4']
const fpsOptions = [12, 24, 30, 60]
const modeOptions = [
  { value: 'ti2vid', label: '多图过渡' },
  { value: 'keyframes', label: '关键帧动画' },
]

watch(() => redoStore.redoData, (newData) => {
  if (newData && newData.mode === 'multi_image_video') {
    prompt.value = newData.prompt || ''
    imageUrlsText.value = newData.imageUrlsText || ''
    mode.value = newData.videoMode || 'ti2vid'
    duration.value = newData.duration || 5
    aspectRatio.value = newData.aspectRatio || '16:9'
    frameRate.value = newData.frameRate || 24
  }
}, { flush: 'sync' })

async function handleGenerate() {
  const urls = imageUrlsText.value
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
  if (urls.length === 0) {
    ElMessage.warning('请输入至少一个图片 URL')
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
    const res = await submitMultiImageVideo({
      prompt: prompt.value,
      image_urls: urls,
      mode: mode.value,
      duration: duration.value,
      aspect_ratio: aspectRatio.value,
      frame_rate: frameRate.value,
    })
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
      <h3 class="gen-title">多图视频</h3>
      <el-form label-width="100px">
        <el-form-item label="图片 URL">
          <el-input
            v-model="imageUrlsText"
            type="textarea"
            :rows="4"
            placeholder="每行一个图片 URL&#10;支持公网可访问的图片地址"
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
        <el-form-item label="模式">
          <el-select v-model="mode" style="width: 180px">
            <el-option v-for="opt in modeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
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
        <video
          v-for="(video, idx) in resultVideos"
          :key="idx"
          :src="video"
          controls
          style="width: 100%; max-height: 400px; border-radius: var(--radius-sm); margin-bottom: 12px"
        />
      </div>
      <div v-else style="padding: 20px; text-align: center; color: var(--text-muted); font-size: 14px">
        输入图片 URL 并填写提示词，视频结果将出现在这里
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
