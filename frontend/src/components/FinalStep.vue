<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check } from '@element-plus/icons-vue'
import { updateProject, getProjectFiles } from '../api/projects'
import type { Project, ProjectFile } from '../types'

const props = defineProps<{ project: Project | null }>()
const emit = defineEmits<{ updated: [] }>()

const router = useRouter()
const notes = ref('')
const selectedCover = ref('')
const completing = ref(false)
const projectFiles = ref<ProjectFile[]>([])

interface OutputItem {
  url: string
  label: string
  stepType: string
}

const allOutputs = computed<OutputItem[]>(() => {
  const items: OutputItem[] = []
  // 收集 ProjectStep 输出
  if (props.project?.steps) {
    for (const step of props.project.steps) {
      if (step.output && (step.step_type === 'generate' || step.step_type === 'refine')) {
        const pos = step.position || step.id
        const label = step.step_type === 'generate' ? `生成 #${pos}` : `优化 #${pos}`
        items.push({ url: step.output, label, stepType: step.step_type })
      }
    }
  }
  // 补充 getProjectFiles 中的图片（去重）
  const seen = new Set(items.map(i => i.url))
  for (const f of projectFiles.value) {
    if (f.type === 'image' && f.url && !seen.has(f.url)) {
      items.push({ url: f.url, label: f.step || '历史记录', stepType: 'generate' })
      seen.add(f.url)
    }
  }
  return items
})

function getStepsByType(type: string) {
  return props.project?.steps?.filter(s => s.step_type === type) || []
}

async function loadFiles() {
  if (!props.project) return
  try {
    projectFiles.value = await getProjectFiles(props.project.id)
  } catch { /* 非阻塞 */ }
}

onMounted(loadFiles)

async function completeProject() {
  if (!props.project) return
  if (!selectedCover.value && allOutputs.value.length > 0) {
    try {
      await ElMessageBox.confirm('尚未选择封面图片，确定要完成吗？', '提示', {
        type: 'warning',
        confirmButtonText: '确定完成',
        cancelButtonText: '去选择',
      })
    } catch {
      return
    }
  }
  completing.value = true
  try {
    await updateProject(props.project.id, {
      status: 'completed',
      notes: notes.value || undefined,
      cover_url: selectedCover.value || undefined,
    })
    ElMessage.success('🎉 项目已完成！')
    emit('updated')
    router.push(`/projects/${props.project.id}/dashboard`)
  } catch (e: any) {
    ElMessage.error('完成失败: ' + (e.message || ''))
  } finally {
    completing.value = false
  }
}
</script>

<template>
  <div class="final-step">
    <!-- 页头 -->
    <div class="step-header">
      <h3>🎯 定稿 — 成果画廊</h3>
      <p v-if="project" class="step-summary">
        共 {{ allOutputs.length }} 张生成结果
        <template v-if="getStepsByType('refine').length">
          ｜优化 {{ getStepsByType('refine').length }} 次
        </template>
      </p>
    </div>

    <div v-if="!project" class="empty">暂无项目数据</div>

    <template v-else>
      <!-- 已完成标记 -->
      <el-alert
        v-if="project.status === 'completed'"
        title="此项目已完成"
        type="success"
        show-icon
        :closable="false"
        class="mb-4"
      />

      <!-- 成果画廊（有图片时展示） -->
      <div v-if="allOutputs.length" class="section">
        <h4>🖼️ 成果画廊</h4>
        <p class="section-tip">点击图片设为项目封面</p>
        <div class="gallery-grid">
          <div
            v-for="(item, idx) in allOutputs"
            :key="idx"
            class="gallery-card"
            :class="{ 'is-cover': selectedCover === item.url }"
            @click="selectedCover = item.url"
          >
            <el-image :src="item.url" fit="cover" loading="lazy" />
            <div class="card-badge" :class="item.stepType">
              {{ item.stepType === 'generate' ? '生成' : '优化' }}
            </div>
            <div v-if="selectedCover === item.url" class="card-cover-badge">
              <el-icon><Check /></el-icon> 封面
            </div>
            <div class="card-label">{{ item.label }}</div>
          </div>
        </div>
      </div>

      <!-- 封面预览 -->
      <div v-if="selectedCover" class="section">
        <h4>📌 封面预览</h4>
        <div class="cover-preview-wrap">
          <el-image :src="selectedCover" fit="contain" class="cover-preview" />
          <el-button
            size="small"
            text
            type="warning"
            @click="selectedCover = ''"
            style="margin-top: 8px"
          >
            取消选择
          </el-button>
        </div>
      </div>

      <!-- 定稿备注 -->
      <div class="section">
        <h4>📝 定稿备注</h4>
        <el-input
          v-model="notes"
          type="textarea"
          :rows="3"
          placeholder="添加完成备注…"
        />
      </div>

      <!-- 完成操作 -->
      <div class="final-actions">
        <el-button
          type="primary"
          size="large"
          :icon="Check"
          :loading="completing"
          :disabled="project.status === 'completed'"
          @click="completeProject"
        >
          完成项目
        </el-button>
        <p class="action-hint">
          标记项目完成并跳转到仪表盘
        </p>
      </div>
    </template>
  </div>
</template>

<style scoped>
.final-step {
  max-width: 800px;
  margin: 0 auto;
}
.step-header {
  margin-bottom: 24px;
}
.step-header h3 {
  margin: 0 0 8px;
  font-size: 18px;
}
.step-summary {
  margin: 0;
  color: #909399;
  font-size: 14px;
}
.empty {
  text-align: center;
  padding: 60px 0;
  color: #c0c4cc;
}
.section {
  margin-bottom: 28px;
}
.section h4 {
  margin: 0 0 8px;
  font-size: 15px;
}
.section-tip {
  margin: 0 0 12px;
  font-size: 13px;
  color: #c0c4cc;
}

/* 画廊网格 */
.gallery-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
}
.gallery-card {
  position: relative;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  border: 3px solid transparent;
  transition: border-color 0.2s, transform 0.15s;
  background: #f5f7fa;
}
.gallery-card:hover {
  transform: translateY(-2px);
}
.gallery-card.is-cover {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2);
}
.gallery-card :deep(.el-image) {
  width: 100%;
  height: 160px;
  display: block;
}
.card-badge {
  position: absolute;
  top: 6px;
  left: 6px;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  color: #fff;
}
.card-badge.generate { background: #67c23a; }
.card-badge.refine  { background: #e6a23c; }
.card-cover-badge {
  position: absolute;
  top: 6px;
  right: 6px;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  background: #409eff;
  color: #fff;
  display: flex;
  align-items: center;
  gap: 2px;
}
.card-label {
  padding: 6px 8px;
  font-size: 12px;
  color: #606266;
  text-align: center;
}

/* 封面预览 */
.cover-preview-wrap {
  text-align: center;
}
.cover-preview {
  max-width: 400px;
  max-height: 300px;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

/* 统计摘要 */
.stats-summary {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 16px;
}
.stat-card {
  text-align: center;
}
.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
  line-height: 1.2;
}
.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 4px;
}

/* 完成操作 */
.final-actions {
  text-align: center;
  padding: 24px 0 12px;
}
.final-actions .el-button--large {
  padding: 16px 48px;
  font-size: 16px;
}
.action-hint {
  margin: 12px 0 0;
  font-size: 13px;
  color: #c0c4cc;
}
.mb-4 {
  margin-bottom: 16px;
}
</style>
