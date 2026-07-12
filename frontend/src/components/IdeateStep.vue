<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Promotion, EditPen, Check, Refresh } from '@element-plus/icons-vue'
import { ideateBrief } from '../api/ideate'
import type { Project } from '../types'

const props = defineProps<{ project: Project | null }>()
const emit = defineEmits<{
  briefGenerated: [briefText: string, prompt: string]
}>()

const idea = ref('')
const loading = ref(false)
const briefDone = ref(false)
const briefResult = ref<{ brief_text: string; generated_prompt: string } | null>(null)
const editing = ref(true)

async function generateBrief() {
  const text = idea.value.trim()
  if (!text) {
    ElMessage.warning('请输入创作想法')
    return
  }
  if (!props.project) return

  loading.value = true
  try {
    const result = await ideateBrief(props.project.id, text)
    briefResult.value = result
    briefDone.value = true
    editing.value = false
    emit('briefGenerated', result.brief_text, result.generated_prompt)
    ElMessage.success('创作简报已生成')
  } catch (e: any) {
    ElMessage.error('生成失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function copyResult(text: string) {
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制')
  }).catch(() => {
    ElMessage.warning('复制失败，请手动复制')
  })
}
</script>

<template>
  <div class="ideate-step">
    <!-- 输入区 -->
    <div v-if="!briefDone || editing" class="input-section">
      <label class="section-label">写下你的创作想法</label>
      <el-input
        v-model="idea"
        type="textarea"
        :rows="6"
        placeholder="描述你想创作的内容，比如主题、风格、氛围... 不需要很详细，AI 会帮你完善。"
        :maxlength="500"
        show-word-limit
      />
      <div class="input-actions">
        <el-button type="primary" :loading="loading" :disabled="!idea.trim()" @click="generateBrief">
          <el-icon><Promotion /></el-icon> 生成创作简报
        </el-button>
      </div>
    </div>

    <!-- 结果卡片 -->
    <div v-if="briefDone && briefResult" class="result-section">
      <div class="result-header">
        <span class="section-label">创作简报</span>
        <div class="result-actions">
          <el-button size="small" text @click="copyResult(briefResult.brief_text)">
            <el-icon><Check /></el-icon> 复制简报
          </el-button>
          <el-button size="small" text @click="copyResult(briefResult.generated_prompt)">
            <el-icon><Check /></el-icon> 复制提示词
          </el-button>
          <el-button size="small" text @click="loading ? null : (editing = !editing); briefDone = false; briefResult = null">
            <el-icon><EditPen /></el-icon> 重新编辑
          </el-button>
          <el-button size="small" text :loading="loading" @click="generateBrief">
            <el-icon><Refresh /></el-icon> 重新生成
          </el-button>
        </div>
      </div>
      <el-card class="result-card">
        <div class="brief-text">{{ briefResult.brief_text }}</div>
        <el-divider />
        <div class="prompt-section">
          <div class="prompt-label">AI 生成提示词</div>
          <pre class="prompt-text">{{ briefResult.generated_prompt }}</pre>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.ideate-step {
  max-width: 720px;
  margin: 0 auto;
}
.section-label {
  display: block;
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 12px;
  color: #303133;
}
.input-section {
  margin-bottom: 24px;
}
.input-actions {
  margin-top: 16px;
  display: flex;
  justify-content: center;
}
.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  flex-wrap: wrap;
  gap: 8px;
}
.result-actions {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}
.result-card {
  line-height: 1.8;
}
.brief-text {
  font-size: 14px;
  color: #303133;
  white-space: pre-wrap;
}
.prompt-section {
  margin-top: 8px;
}
.prompt-label {
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
}
.prompt-text {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: #606266;
  margin: 0;
}
</style>
