<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Edit, Delete, CopyDocument, EditPen } from '@element-plus/icons-vue'
import { listProjects, createProject, updateProject, deleteProject, duplicateProject } from '../api/projects'
import type { Project } from '../types'

const emit = defineEmits<{
  editProject: [project: Project]
}>()

const projects = ref<Project[]>([])
const loading = ref(false)

const showDialog = ref(false)
const isEditing = ref(false)
const editingId = ref<number | null>(null)
const form = ref({ title: '', brief: '' })
const projectType = ref<'project' | 'comic'>('project')

async function loadProjects() {
  loading.value = true
  try {
    projects.value = await listProjects()
  } catch (e: any) {
    ElMessage.error('加载项目列表失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

onMounted(loadProjects)

function openNew() {
  isEditing.value = false
  editingId.value = null
  form.value = { title: '', brief: '' }
  projectType.value = 'project'
  showDialog.value = true
}

function openEdit(p: Project) {
  isEditing.value = true
  editingId.value = p.id
  form.value = { title: p.title, brief: p.brief }
  showDialog.value = true
}

async function saveProject() {
  if (!form.value.title.trim()) {
    ElMessage.warning('请输入项目标题')
    return
  }
  try {
    if (isEditing.value && editingId.value) {
      await updateProject(editingId.value, form.value)
      ElMessage.success('保存成功')
    } else {
      await createProject({ ...form.value, type: projectType.value })
      ElMessage.success('创建成功')
    }
    showDialog.value = false
    await loadProjects()
  } catch (e: any) {
    ElMessage.error('操作失败: ' + (e.message || ''))
  }
}

async function deleteById(id: number) {
  try {
    await ElMessageBox.confirm('确定删除此项目？所有步骤将一同删除。', '确认删除', { type: 'warning' })
    await deleteProject(id)
    ElMessage.success('删除成功')
    await loadProjects()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

async function duplicateById(id: number) {
  try {
    const p = await duplicateProject(id)
    ElMessage.success(`已复制为: ${p.title}`)
    await loadProjects()
  } catch (e: any) {
    ElMessage.error('复制失败: ' + (e.message || ''))
  }
}

function goToEditor(project: Project) {
  emit('editProject', project)
}

const statusLabels: Record<string, string> = {
  draft: '草稿',
  generating: '生成中',
  refining: '优化中',
  completed: '已完成'
}

const statusTypes: Record<string, string> = {
  draft: 'info',
  generating: 'warning',
  refining: '',
  completed: 'success'
}

function statusLabel(s: string) {
  return statusLabels[s] || s
}

function stepProgressSummary(sp: string | null | undefined): string {
  if (!sp) return ''
  try {
    const data = JSON.parse(sp)
    const hasLayout = 'layout' in data
    const steps: Record<string, string> = hasLayout
      ? { ideate: '构思', layout: '布局', generate: '生成', refine: '精修', finalize: '定稿' }
      : { ideate: '发想', generate: '生成', refine: '优化', finalize: '定稿' }
    return Object.entries(steps)
      .filter(([k]) => data[k])
      .map(([k, v]) => {
        if (data[k] === 'completed') return v + ' ✓'
        if (data[k] === 'in_progress') return v + ' ●'
        return v + ' ○'
      })
      .join(' | ')
  } catch { return '' }
}
</script>

<template>
  <div>
    <div class="project-header">
      <h3>创作项目</h3>
      <el-button type="primary" :icon="Plus" @click="openNew">新建项目</el-button>
    </div>

    <div v-loading="loading">
      <div v-if="projects.length === 0 && !loading" class="empty-state">
        <el-icon :size="48"><EditPen /></el-icon>
        <p>暂无创作项目</p>
        <el-button type="primary" @click="openNew">创建第一个项目</el-button>
      </div>

      <el-row :gutter="16" v-else>
        <el-col
          v-for="p in projects"
          :key="p.id"
          :xs="24" :sm="12" :md="8" :lg="6"
          class="mb-4"
        >
          <el-card shadow="hover" class="project-card" @click="goToEditor(p)">
            <div class="card-header">
              <div class="card-title-row">
                <h3>{{ p.title || '未命名项目' }}</h3>
                <el-tag :type="statusTypes[p.status] || 'info'" size="small">
                  {{ statusLabel(p.status) }}
                </el-tag>
                <el-tag v-if="p.type === 'comic'" type="warning" size="small">漫画</el-tag>
                <el-tag v-else-if="p.type === 'project'" type="" size="small">创作</el-tag>
              </div>
              <div class="card-actions" @click.stop>
                <el-button text @click="openEdit(p)"><el-icon><Edit /></el-icon></el-button>
                <el-button text @click="duplicateById(p.id)"><el-icon><CopyDocument /></el-icon></el-button>
                <el-button text type="danger" @click="deleteById(p.id)"><el-icon><Delete /></el-icon></el-button>
              </div>
            </div>
            <div class="card-brief" v-if="p.brief">
              {{ p.brief.length > 80 ? p.brief.slice(0, 80) + '...' : p.brief }}
            </div>
            <div class="card-meta" v-if="p.step_progress">
              <span class="step-summary">{{ stepProgressSummary(p.step_progress) }}</span>
            </div>
            <div class="card-meta" v-else>
              <span>{{ p.steps?.length || 0 }} 个步骤</span>
              <span>{{ p.updated_at?.slice(0, 10) }}</span>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <el-dialog
      v-model="showDialog"
      :title="isEditing ? '编辑项目' : '新建项目'"
      width="500px"
    >
      <el-form :model="form" label-width="60px">
        <el-form-item label="标题">
          <el-input v-model="form.title" placeholder="项目名称" />
        </el-form-item>
        <el-form-item label="类型" v-if="!isEditing">
          <el-radio-group v-model="projectType">
            <el-radio value="project">创作项目</el-radio>
            <el-radio value="comic">漫画项目</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="简报">
          <el-input
            v-model="form.brief"
            type="textarea"
            :rows="4"
            placeholder="创意简报 — AI 将根据此内容提供建议（可选）"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="saveProject">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.project-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--spacing-md, 20px);
}
.empty-state {
  text-align: center;
  padding: 60px 0;
  color: #c0c4cc;
}
.empty-state p {
  margin-top: 12px;
}
.project-card {
  cursor: pointer;
  transition: transform 0.2s;
}
.project-card:hover {
  transform: translateY(-2px);
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}
.card-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}
.card-title-row h3 {
  margin: 0;
  font-size: 15px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.card-brief {
  margin-top: 8px;
  font-size: 13px;
  color: #909399;
  line-height: 1.5;
}
.card-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 12px;
  color: #c0c4cc;
  font-size: 12px;
}
.step-summary {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
}
.card-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}
.mb-4 {
  margin-bottom: 16px;
}
</style>
