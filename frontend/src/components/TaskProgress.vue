<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { connectTaskSSE } from '../utils/sse'

const props = defineProps<{
  taskId: string
}>()

const emit = defineEmits<{
  complete: [result: string]
  error: [message: string]
}>()

const progress = ref(0)
const status = ref('pending')
const loading = ref(true)
let cleanup: (() => void) | null = null

onMounted(() => {
  cleanup = connectTaskSSE(props.taskId, {
    onProgress: (data) => {
      progress.value = data.progress
      status.value = data.status
    },
    onComplete: (data) => {
      progress.value = 100
      status.value = 'completed'
      loading.value = false
      emit('complete', data.result)
    },
    onError: (data) => {
      status.value = 'failed'
      loading.value = false
      emit('error', data.error)
    },
  })
})

onUnmounted(() => {
  cleanup?.()
})
</script>

<template>
  <div class="task-progress">
    <div v-if="loading || status === 'processing' || status === 'pending'" class="progress-bar-wrapper">
      <el-progress
        :percentage="progress"
        :status="status === 'failed' ? 'exception' : undefined"
        :stroke-width="16"
        :text-inside="true"
      />
      <p class="status-text">
        {{ status === 'pending' ? '排队中...' : status === 'processing' ? `生成中 ${progress}%` : '' }}
      </p>
    </div>
    <div v-else-if="status === 'failed'" class="error-message">
      <el-alert title="生成失败" type="error" show-icon />
    </div>
  </div>
</template>

<style scoped>
.task-progress {
  margin: 16px 0;
}
.progress-bar-wrapper {
  text-align: center;
}
.status-text {
  margin-top: 8px;
  color: #909399;
  font-size: 14px;
}
.error-message {
  margin-top: 8px;
}
</style>
