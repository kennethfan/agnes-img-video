<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { createTextToVideo } from '../api/video'
import { useRedoStore } from '../stores/redo'

const prompt = ref('')
const duration = ref(5)
const aspectRatio = ref('16:9')
const frameRate = ref(24)
const loading = ref(false)
const taskId = ref('')
const redoStore = useRedoStore()

const durationOptions = [3, 5, 8, 10, 15, 18]
const ratioOptions = ['16:9', '9:16', '1:1', '4:3', '3:4']
const fpsOptions = [12, 24, 30, 60]

// 监听重做数据（flush: sync 确保同步触发）
watch(() => redoStore.redoData, (newData) => {
  if (newData && newData.mode === 'text2video') {
    prompt.value = newData.prompt || ''
    duration.value = newData.duration || 5
    aspectRatio.value = newData.aspectRatio || '16:9'
    frameRate.value = newData.frameRate || 24
  }
}, { flush: 'sync' })

async function handleGenerate() {
  if (!prompt.value.trim()) {
    ElMessage.warning('请输入提示词')
    return
  }
  loading.value = true
  try {
    const res = await createTextToVideo({
      prompt: prompt.value,
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
  <div class="gen-page">
    <div class="gen-input">
      <h3 class="gen-title">文生视频</h3>
      <el-form label-width="100px">
        <el-form-item label="提示词">
          <el-input
            v-model="prompt"
            type="textarea"
            :rows="3"
            placeholder="描述你想要生成的视频..."
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
      <div v-if="taskId" style="padding: 20px">
        <el-alert title="任务已提交" type="success" show-icon>
          <template #default>
            <p>任务 ID: {{ taskId }}</p>
            <p>视频生成可能需要几分钟，请在「历史记录」中查看进度。</p>
          </template>
        </el-alert>
      </div>
      <div v-else style="padding: 20px; text-align: center; color: var(--text-muted); font-size: 14px">
        填写左侧表单并点击生成，视频结果将出现在这里
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
