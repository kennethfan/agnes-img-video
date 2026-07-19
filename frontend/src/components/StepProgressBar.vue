<script setup lang="ts">
import { Check } from '@element-plus/icons-vue'

const props = defineProps<{
  steps: { key: string; label: string; status: string }[]
}>()

const statusColor = (status: string): string => {
  if (status === 'completed') return 'var(--el-color-success)'
  if (status === 'in_progress') return 'var(--el-color-primary)'
  return 'var(--el-color-info)'
}
</script>

<template>
  <div class="step-progress-bar">
    <div
      v-for="(step, index) in steps"
      :key="step.key"
      class="step-item"
      :class="{ active: step.status === 'in_progress', done: step.status === 'completed' }"
    >
      <div class="step-indicator" :style="{ borderColor: statusColor(step.status) }">
        <el-icon v-if="step.status === 'completed'" :color="statusColor(step.status)">
          <Check />
        </el-icon>
        <span v-else class="step-num">{{ index + 1 }}</span>
      </div>
      <span class="step-label" :style="{ color: statusColor(step.status) }">{{ step.label }}</span>
      <div v-if="index < steps.length - 1" class="step-line" :class="{ done: step.status === 'completed' }" />
    </div>
  </div>
</template>

<style scoped>
.step-progress-bar {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 16px 0;
}
.step-item {
  display: flex;
  align-items: center;
  position: relative;
  gap: 8px;
}
.step-indicator {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 2px solid var(--el-color-info);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
  background: #fff;
  flex-shrink: 0;
}
.step-item.done .step-indicator {
  background: var(--el-color-success);
  border-color: var(--el-color-success);
  color: #fff;
}
.step-item.active .step-indicator {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.step-num {
  font-size: 13px;
  line-height: 1;
}
.step-label {
  font-size: 13px;
  white-space: nowrap;
}
.step-line {
  width: 60px;
  height: 2px;
  background: var(--el-color-info-light-5);
  margin: 0 12px;
  flex-shrink: 0;
}
.step-line.done {
  background: var(--el-color-success);
}
</style>
