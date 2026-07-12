<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { MagicStick, Refresh } from '@element-plus/icons-vue'
import type { Project } from '../types'

const props = defineProps<{
  project: Project | null
  aiResult: string
  loading: boolean
}>()

const emit = defineEmits<{ recommend: [] }>()

function reRecommend() {
  emit('recommend')
}

function copyResult() {
  if (props.aiResult) {
    navigator.clipboard.writeText(props.aiResult).then(() => {
      ElMessage.success('已复制到剪贴板')
    })
  }
}
</script>

<template>
  <div class="ai-panel">
    <div class="panel-header">
      <h3><el-icon><MagicStick /></el-icon> AI 创作顾问</h3>
      <el-button size="small" :icon="Refresh" :loading="loading" @click="reRecommend">
        {{ loading ? 'AI 思考中...' : '重新推荐' }}
      </el-button>
    </div>

    <div v-if="!project" class="empty">请先选择一个项目</div>

    <div v-else-if="!aiResult && !loading" class="empty">
      <p>{{ project.brief ? '点击「AI 推荐」获取创作建议' : '项目暂无创意简报，AI 将基于标题提供建议' }}</p>
      <el-button type="primary" :icon="MagicStick" :loading="loading" @click="reRecommend">
        AI 推荐
      </el-button>
    </div>

    <div v-else-if="loading" class="loading-state">
      <el-skeleton :rows="6" animated />
    </div>

    <div v-else class="result-area">
      <div class="result-content markdown-body">
        <pre>{{ aiResult }}</pre>
      </div>
      <div class="result-actions">
        <el-button size="small" @click="copyResult">复制内容</el-button>
        <el-button size="small" :icon="Refresh" @click="reRecommend">重新生成</el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ai-panel {
  max-width: 800px;
  margin: 0 auto;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.panel-header h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
}
.empty {
  text-align: center;
  padding: 60px 0;
  color: #c0c4cc;
}
.empty p {
  margin-bottom: 16px;
}
.loading-state {
  padding: 20px;
}
.result-area {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 20px;
}
.result-content {
  max-height: 500px;
  overflow-y: auto;
  font-size: 14px;
  line-height: 1.6;
}
.result-content pre {
  white-space: pre-wrap;
  font-family: inherit;
  margin: 0;
}
.result-actions {
  margin-top: 16px;
  display: flex;
  gap: 8px;
}
</style>
