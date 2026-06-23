<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { createMultiImageVideo } from '../api/video'

const prompt = ref('')
const imageUrlsText = ref('')
const mode = ref('ti2vid')
const duration = ref(5)
const aspectRatio = ref('16:9')
const frameRate = ref(24)
const loading = ref(false)
const taskId = ref('')

const durationOptions = [3, 5, 8, 10, 15, 18]
const ratioOptions = ['16:9', '9:16', '1:1', '4:3', '3:4']
const fpsOptions = [12, 24, 30, 60]
const modeOptions = [
  { value: 'ti2vid', label: '多图过渡' },
  { value: 'keyframes', label: '关键帧动画' },
]

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
  try {
    const res = await createMultiImageVideo({
      prompt: prompt.value,
      image_urls: urls,
      mode: mode.value,
      duration: duration.value,
      aspect_ratio: aspectRatio.value,
      frame_rate: frameRate.value,
    })
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
