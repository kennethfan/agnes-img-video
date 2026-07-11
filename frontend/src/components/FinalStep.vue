<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Edit, Plus } from '@element-plus/icons-vue'
import { updateProject } from '../api/projects'
import { saveAsset } from '../api/assets'
import ImageResult from './ImageResult.vue'
import type { Project } from '../types'

const props = defineProps<{ project: Project | null }>()
const emit = defineEmits<{ updated: [] }>()

const editingNotes = ref(false)
const notes = ref('')
const finalUrl = ref('')
const saving = ref(false)
const savingSteps = ref<Set<number>>(new Set())

async function saveStepOutput(stepId: number, imageUrl: string) {
  savingSteps.value = new Set([...savingSteps.value, stepId])
  try {
    await saveAsset({ image_url: imageUrl, prompt: '来自创作项目', mode: 'image' })
    ElMessage.success('已保存到作品库')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    const next = new Set(savingSteps.value)
    next.delete(stepId)
    savingSteps.value = next
  }
}

function startEditNotes() {
  if (!props.project) return
  notes.value = props.project.notes || ''
  finalUrl.value = props.project.final_url || ''
  editingNotes.value = true
}

async function saveFinal() {
  if (!props.project) return
  saving.value = true
  try {
    await updateProject(props.project.id, {
      status: 'completed',
      notes: notes.value
    })
    ElMessage.success('已保存定稿信息')
    editingNotes.value = false
    emit('updated')
  } catch (e: any) {
    ElMessage.error('保存失败: ' + (e.message || ''))
  } finally {
    saving.value = false
  }
}

async function markCompleted() {
  if (!props.project) return
  try {
    await ElMessageBox.confirm('确认将此项目标记为已完成？', '定稿确认')
    await updateProject(props.project.id, { status: 'completed' })
    ElMessage.success('项目已标记为已完成')
    emit('updated')
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error('操作失败')
  }
}

function getStepsByType(type: string) {
  return props.project?.steps?.filter(s => s.step_type === type) || []
}
</script>

<template>
  <div class="final-step">
    <div class="step-intro">
      <h3><el-icon><Check /></el-icon> 定稿</h3>
      <p>查看生成结果，添加备注，完成项目。</p>
    </div>

    <div v-if="!project" class="empty">暂无项目数据</div>

    <template v-else>
      <!-- 状态 -->
      <el-alert
        v-if="project.status === 'completed'"
        title="此项目已完成"
        type="success"
        show-icon
        :closable="false"
        class="mb-4"
      />

      <!-- 生成记录 -->
      <div v-if="getStepsByType('generate').length" class="section">
        <h4>生成记录（{{ getStepsByType('generate').length }} 条）</h4>
        <div v-for="step in getStepsByType('generate')" :key="step.id" class="step-record">
          <div class="record-label">输入: {{ step.input?.slice(0, 100) }}</div>
          <div v-if="step.output" class="record-output">
            <ImageResult :images="[step.output]" :loading="false" prompt="" mode="" />
            <div style="margin-top: 8px">
              <el-button
                size="small"
                type="success"
                :icon="Plus"
                :loading="savingSteps.has(step.id)"
                :disabled="savingSteps.has(step.id)"
                @click="saveStepOutput(step.id, step.output)"
              >
                {{ savingSteps.has(step.id) ? '保存中...' : '保存到作品库' }}
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <div v-if="getStepsByType('refine').length" class="section">
        <h4>优化记录（{{ getStepsByType('refine').length }} 条）</h4>
        <div v-for="step in getStepsByType('refine')" :key="step.id" class="step-record">
          <div class="record-label">优化: {{ step.input?.slice(0, 100) }}</div>
          <div v-if="step.output" class="record-output">
            <ImageResult :images="[step.output]" :loading="false" prompt="" mode="" />
            <div style="margin-top: 8px">
              <el-button
                size="small"
                type="success"
                :icon="Plus"
                :loading="savingSteps.has(step.id)"
                :disabled="savingSteps.has(step.id)"
                @click="saveStepOutput(step.id, step.output)"
              >
                {{ savingSteps.has(step.id) ? '保存中...' : '保存到作品库' }}
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <!-- 备注编辑 -->
      <div class="section">
        <h4>项目备注</h4>
        <div v-if="!editingNotes">
          <p v-if="project.notes" class="notes-preview">{{ project.notes }}</p>
          <p v-else class="notes-empty">暂无备注</p>
          <el-button size="small" :icon="Edit" @click="startEditNotes">编辑</el-button>
        </div>
        <div v-else class="notes-edit">
          <el-input v-model="notes" type="textarea" :rows="4" placeholder="定稿备注..." />
          <div class="notes-actions">
            <el-button @click="editingNotes = false">取消</el-button>
            <el-button type="primary" :loading="saving" @click="saveFinal">保存</el-button>
          </div>
        </div>
      </div>

      <!-- 定稿操作 -->
      <div v-if="project.status !== 'completed'" class="final-actions">
        <el-button type="success" :icon="Check" @click="markCompleted">
          标记为已完成
        </el-button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.final-step {
  max-width: 700px;
  margin: 0 auto;
}
.step-intro {
  margin-bottom: 24px;
}
.step-intro h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 8px;
}
.step-intro p {
  margin: 0;
  color: #909399;
}
.empty {
  text-align: center;
  padding: 60px 0;
  color: #c0c4cc;
}
.section {
  margin-bottom: 24px;
}
.section h4 {
  margin: 0 0 12px;
  font-size: 15px;
}
.step-record {
  background: #f5f7fa;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 8px;
}
.record-label {
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
}
.record-output {
  max-width: 300px;
}
.notes-preview {
  white-space: pre-wrap;
  line-height: 1.5;
  color: #606266;
}
.notes-empty {
  color: #c0c4cc;
}
.notes-edit {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.notes-actions {
  display: flex;
  gap: 8px;
}
.final-actions {
  margin-top: 32px;
  text-align: center;
}
.mb-4 {
  margin-bottom: 16px;
}
</style>
