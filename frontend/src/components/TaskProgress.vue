<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { connectTaskSSE } from '../utils/sse'
import { cancelTask, retryTask } from '../api/task'

const props = defineProps<{
  taskId: number | string
}>()

const emit = defineEmits<{
  complete: [result: string]
  error: [message: string]
  retry: [taskId: number | string]
}>()

const progress = ref(0)
const status = ref('pending')
const loading = ref(true)
const cancelling = ref(false)
const retrying = ref(false)
let cleanup: (() => void) | null = null

async function handleCancel() {
  cancelling.value = true
  try {
    await cancelTask(props.taskId)
    status.value = 'cancelled'
    loading.value = false
    cleanup?.()
  } catch {
    // 错误由组件外部显示
  } finally {
    cancelling.value = false
  }
}

async function handleRetry() {
  retrying.value = true
  try {
    cleanup?.()
    await retryTask(props.taskId)
    // 重新连接 SSE
    status.value = 'pending'
    progress.value = 0
    loading.value = true
    cleanup = connectTaskSSE(props.taskId, {
      onProgress: (data) => {
        progress.value = data.progress
        status.value = data.status
      },
      onComplete: (data) => {
        progress.value = 100
        status.value = 'completed'
        loading.value = false
        cleanup?.()
        emit('complete', data.result)
      },
      onError: (data) => {
        status.value = 'failed'
        loading.value = false
        emit('error', data.error)
      },
    })
  } catch {
    // 错误由组件外部显示
  } finally {
    retrying.value = false
  }
}

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
    <div v-if="status === 'cancelled'" class="cancelled-message">
      <el-alert title="任务已取消" type="info" show-icon :closable="false" />
    </div>
    <div v-else-if="loading || status === 'processing' || status === 'pending'" class="progress-bar-wrapper">
      <el-progress
        :percentage="progress"
        :status="status === 'failed' ? 'exception' : undefined"
        :stroke-width="16"
        :text-inside="true"
      />
      <p class="status-text">
        {{ status === 'pending' ? '排队中...' : status === 'processing' ? `生成中 ${progress}%` : '' }}
      </p>
      <el-button
        v-if="status === 'pending'"
        type="danger"
        size="small"
        :loading="cancelling"
        @click="handleCancel"
        style="margin-top: 8px"
      >
        {{ cancelling ? '取消中...' : '取消' }}
      </el-button>
    </div>
    <div v-else-if="status === 'failed'" class="error-message">
      <el-alert title="生成失败" type="error" show-icon :closable="false" />
      <el-button
        type="primary"
        size="small"
        :loading="retrying"
        @click="handleRetry"
        style="margin-top: 8px"
      >
        {{ retrying ? '重试中...' : '重试' }}
      </el-button>
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
.cancelled-message {
  margin-top: 8px;
}
</style>
