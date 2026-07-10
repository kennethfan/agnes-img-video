<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Edit, Delete, CopyDocument, VideoPlay, Document } from '@element-plus/icons-vue'
import { listProjects, getProject, createProject, updateProject, deleteProject, duplicateProject, createShot, updateShot, deleteShot, generateShots, reorderShots, batchCreateShots } from '../api/storyboard'
import type { StoryboardProject, StoryboardShot, GenerateShotsResponse } from '../types'
import ShotCard from '../components/ShotCard.vue'

const view = ref<'list' | 'detail'>('list')

const projects = ref<StoryboardProject[]>([])
const loading = ref(false)

const currentProject = ref<StoryboardProject | null>(null)
const shots = ref<StoryboardShot[]>([])

const showProjectDialog = ref(false)
const isEditingProject = ref(false)
const projectForm = ref({ title: '', script: '' })

const showShotDialog = ref(false)
const editingShot = ref<number | null>(null)
const shotForm = ref({ prompt: '', type: 'text2video' as string, reference_image: '' })

async function loadProjects() {
  loading.value = true
  try {
    projects.value = await listProjects()
  } catch (e: any) {
    ElMessage.error('加载失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

onMounted(loadProjects)

function openNewProject() {
  isEditingProject.value = false
  projectForm.value = { title: '', script: '' }
  showProjectDialog.value = true
}

async function saveProject() {
  try {
    if (isEditingProject.value && currentProject.value) {
      await updateProject(currentProject.value.id, projectForm.value)
      ElMessage.success('保存成功')
    } else {
      const id = await createProject(projectForm.value)
      currentProject.value = { id, title: projectForm.value.title, script: projectForm.value.script, created_at: '', updated_at: '', shot_count: 0 }
      view.value = 'detail'
      ElMessage.success('创建成功')
    }
    showProjectDialog.value = false
    await loadProjects()
  } catch (e: any) {
    ElMessage.error('操作失败')
  }
}

async function openProject(project: StoryboardProject) {
  loading.value = true
  try {
    const data = await getProject(project.id)
    currentProject.value = data.project
    shots.value = data.shots
    view.value = 'detail'
  } catch (e: any) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function backToList() {
  view.value = 'list'
  currentProject.value = null
  shots.value = []
}

function editProject() {
  if (!currentProject.value) return
  isEditingProject.value = true
  projectForm.value = { title: currentProject.value.title, script: currentProject.value.script }
  showProjectDialog.value = true
}

async function deleteCurrentProject() {
  if (!currentProject.value) return
  try {
    await ElMessageBox.confirm('确定删除此分镜项目？所有镜头将一同删除。', '确认删除', { type: 'warning' })
    await deleteProject(currentProject.value.id)
    ElMessage.success('删除成功')
    backToList()
    await loadProjects()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

async function duplicateCurrentProject() {
  if (!currentProject.value) return
  try {
    await duplicateProject(currentProject.value.id)
    ElMessage.success('复制成功')
    await loadProjects()
  } catch (e: any) {
    ElMessage.error('复制失败')
  }
}

async function duplicateProjectById(id: number) {
  try {
    await duplicateProject(id)
    ElMessage.success('复制成功')
    await loadProjects()
  } catch (e: any) {
    ElMessage.error('复制失败: ' + (e.message || ''))
  }
}

async function deleteProjectById(id: number) {
  try {
    await deleteProject(id)
    ElMessage.success('删除成功')
    await loadProjects()
  } catch (e: any) {
    ElMessage.error('删除失败: ' + (e.message || ''))
  }
}

function editProjectByObj(project: StoryboardProject) {
  isEditingProject.value = true
  projectForm.value = { title: project.title, script: project.script }
  showProjectDialog.value = true
}

function openNewShot() {
  editingShot.value = null
  shotForm.value = { prompt: '', type: 'text2video', reference_image: '' }
  showShotDialog.value = true
}

function editShot(shot: StoryboardShot) {
  editingShot.value = shot.id
  shotForm.value = { prompt: shot.prompt, type: shot.type, reference_image: shot.reference_image || '' }
  showShotDialog.value = true
}

async function saveShotEdit() {
  if (!editingShot.value) { await addShot(); return }
  try {
    await updateShot(editingShot.value, shotForm.value)
    ElMessage.success('更新成功')
    showShotDialog.value = false
    editingShot.value = null
    if (currentProject.value) {
      const data = await getProject(currentProject.value.id)
      shots.value = data.shots
    }
  } catch (e: any) {
    ElMessage.error('更新失败')
  }
}

function generateSingleShot() {
  if (!currentProject.value) return
  ElMessage.info('请使用"批量生成"按钮触发所有待生成镜头')
}

async function addShot() {
  if (!currentProject.value) return
  if (!shotForm.value.prompt) {
    ElMessage.warning('请输入提示词')
    return
  }
  try {
    await createShot(currentProject.value.id, shotForm.value)
    ElMessage.success('添加成功')
    showShotDialog.value = false
    const data = await getProject(currentProject.value.id)
    currentProject.value = data.project
    shots.value = data.shots
  } catch (e: any) {
    ElMessage.error('添加失败')
  }
}

async function deleteShotById(id: number) {
  try {
    await deleteShot(id)
    shots.value = shots.value.filter(s => s.id !== id)
    ElMessage.success('删除成功')
  } catch (e: any) {
    ElMessage.error('删除失败')
  }
}

async function onShotDrop(fromIndex: number, toIndex: number) {
  if (!currentProject.value) return
  const arr = [...shots.value]
  const [moved] = arr.splice(fromIndex, 1)
  arr.splice(toIndex, 0, moved)
  arr.forEach((s, i) => { s.sequence = i + 1 })
  shots.value = arr
  try {
    await reorderShots(currentProject.value.id, arr.map(s => s.id))
  } catch (e: any) {
    ElMessage.error('排序保存失败: ' + (e.message || ''))
  }
}

const generating = ref(false)
const generateResult = ref<GenerateShotsResponse | null>(null)

async function handleGenerateShots() {
  if (!currentProject.value) return
  generating.value = true
  generateResult.value = null
  try {
    const result = await generateShots(currentProject.value.id)
    generateResult.value = result
    ElMessage.success(`已提交 ${result.submitted} 个镜头生成任务`)
    if (result.submitted > 0) {
      startPollingShots(currentProject.value.id)
    }
  } catch (e: any) {
    ElMessage.error('批量生成失败: ' + (e.message || ''))
  } finally {
    generating.value = false
  }
}

let pollTimer: ReturnType<typeof setInterval> | null = null

function startPollingShots(projectId: number) {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = setInterval(async () => {
    try {
      const resp = await getProject(projectId)
      shots.value = resp.shots
      const hasGenerating = resp.shots.some(s => s.status === 'generating')
      if (!hasGenerating) {
        if (pollTimer) {
          clearInterval(pollTimer)
          pollTimer = null
        }
        ElMessage.success('所有镜头生成完毕')
      }
    } catch (e: any) {
      console.warn('[Storyboard] 轮询镜头状态失败:', e)
    }
  }, 3000)
}

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

const showImportDialog = ref(false)
const importing = ref(false)
const importScript = ref('')
const splitMode = ref<'line' | 'paragraph'>('paragraph')

const previewCount = computed(() => {
  if (!importScript.value.trim()) return 0
  if (splitMode.value === 'line') {
    return importScript.value.split('\n').filter(l => l.trim()).length
  }
  return importScript.value.split(/\n\n+/).filter(p => p.trim()).length
})

async function doImport() {
  if (!currentProject.value) return
  if (importing.value) return
  const text = importScript.value.trim()
  if (!text) {
    ElMessage.warning('请输入脚本内容')
    return
  }

  const prompts = splitMode.value === 'line'
    ? text.split('\n').filter(l => l.trim()).map(l => l.trim())
    : text.split(/\n\n+/).filter(p => p.trim()).map(p => p.trim())

  if (prompts.length === 0) {
    ElMessage.warning('未能解析出镜头内容')
    return
  }

  importing.value = true
  try {
    const resp = await batchCreateShots(currentProject.value.id, prompts)
    const projectResp = await getProject(currentProject.value.id)
    shots.value = projectResp.shots
    showImportDialog.value = false
    importScript.value = ''
    ElMessage.success(`成功导入 ${resp.shots.length} 个镜头`)
  } catch (e: any) {
    ElMessage.error('导入失败: ' + (e.message || ''))
  } finally {
    importing.value = false
  }
}

const previewUrl = ref('')
const showPreview = ref(false)

function previewVideo(url: string) {
  previewUrl.value = url
  showPreview.value = true
}
</script>

<template>
  <div>
    <template v-if="view === 'list'">
      <div class="storyboard-header">
        <h3>分镜项目</h3>
        <el-button type="primary" :icon="Plus" @click="openNewProject">新建项目</el-button>
      </div>

      <div v-loading="loading">
        <div v-if="projects.length === 0 && !loading" class="empty-state">
          <el-icon :size="48"><VideoPlay /></el-icon>
          <p>暂无分镜项目</p>
          <el-button type="primary" @click="openNewProject">创建第一个项目</el-button>
        </div>

        <el-row :gutter="16" v-else>
          <el-col
            v-for="project in projects"
            :key="project.id"
            :xs="24" :sm="12" :md="8" :lg="6"
            class="mb-4"
          >
            <el-card shadow="hover" class="project-card" @click="openProject(project)">
              <div class="project-card-header">
                <h3>{{ project.title || '未命名项目' }}</h3>
                <div class="project-card-actions" @click.stop>
                  <el-button text @click="editProjectByObj(project)"><el-icon><Edit /></el-icon></el-button>
                  <el-button text @click="duplicateProjectById(project.id)"><el-icon><CopyDocument /></el-icon></el-button>
                  <el-button text type="danger" @click="deleteProjectById(project.id)"><el-icon><Delete /></el-icon></el-button>
                </div>
              </div>
              <div class="project-card-meta">
                <span>{{ project.shot_count }} 个镜头</span>
                <span>{{ project.updated_at?.slice(0, 10) }}</span>
              </div>
            </el-card>
          </el-col>
        </el-row>
      </div>
    </template>

    <template v-else-if="view === 'detail' && currentProject">
      <div class="detail-header">
        <el-button text @click="backToList">&lt; 返回</el-button>
        <h3>{{ currentProject.title || '未命名项目' }}</h3>
        <div class="detail-actions">
          <el-button size="small" :icon="Document" @click="showImportDialog = true">从脚本导入</el-button>
          <el-button size="small" :icon="Edit" @click="editProject">编辑</el-button>
          <el-button size="small" :icon="CopyDocument" @click="duplicateCurrentProject">复制</el-button>
          <el-button size="small" type="danger" :icon="Delete" @click="deleteCurrentProject">删除</el-button>
          <el-button
            v-if="shots.some(s => s.status === 'pending')"
            type="primary"
            size="small"
            :loading="generating"
            @click="handleGenerateShots"
            :disabled="shots.filter(s => s.status === 'pending').length === 0"
          >
            {{ generating ? '生成中...' : `批量生成 (${shots.filter(s => s.status === 'pending').length})` }}
          </el-button>
        </div>
      </div>

      <div v-if="generateResult" class="generate-result">
        <el-alert
          :title="`已提交 ${generateResult.submitted}/${generateResult.total} 个镜头生成任务`"
          :type="generateResult.failed > 0 ? 'warning' : 'success'"
          show-icon
          closable
        />
      </div>

      <div v-if="currentProject.script" class="script-preview">
        <h3>脚本</h3>
        <p>{{ currentProject.script }}</p>
      </div>

      <div v-loading="loading">
        <div v-if="shots.length === 0 && !loading" style="text-align: center; padding: 40px; color: #c0c4cc">
          <p>暂无镜头，添加第一个镜头开始策划</p>
        </div>

        <div v-else class="shots-grid">
          <ShotCard
            v-for="(shot, index) in shots"
            :key="shot.id"
            :shot="shot"
            :index="index"
            @edit="editShot(shot)"
            @delete="deleteShotById(shot.id)"
            @generate="generateSingleShot"
            @preview="shot.result_video ? previewVideo(shot.result_video) : undefined"
            @drop="onShotDrop"
          />
        </div>

        <div class="add-shot">
          <el-button type="primary" plain :icon="Plus" @click="openNewShot">添加镜头</el-button>
        </div>
      </div>
    </template>

    <el-dialog
      v-model="showProjectDialog"
      :title="isEditingProject ? '编辑项目' : '新建项目'"
      width="500px"
    >
      <el-form :model="projectForm" label-width="60px">
        <el-form-item label="标题">
          <el-input v-model="projectForm.title" placeholder="分镜项目名称" />
        </el-form-item>
        <el-form-item label="脚本">
          <el-input
            v-model="projectForm.script"
            type="textarea"
            :rows="6"
            placeholder="可选：粘贴完整的脚本内容，后续可拆分为镜头"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showProjectDialog = false">取消</el-button>
        <el-button type="primary" @click="saveProject">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showShotDialog"
      :title="editingShot ? '编辑镜头' : '添加镜头'"
      width="500px"
    >
      <el-form :model="shotForm" label-width="80px">
        <el-form-item label="提示词">
          <el-input
            v-model="shotForm.prompt"
            type="textarea"
            :rows="4"
            placeholder="描述这个镜头的画面内容"
          />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="shotForm.type">
            <el-option label="文生视频" value="text2video" />
            <el-option label="图生视频" value="image2video" />
          </el-select>
        </el-form-item>
        <el-form-item label="参考图">
          <el-input v-model="shotForm.reference_image" placeholder="图片 URL（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showShotDialog = false">取消</el-button>
        <el-button type="primary" @click="saveShotEdit">{{ editingShot ? '保存' : '添加' }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showImportDialog" title="从脚本导入镜头" width="600px">
      <el-alert
        title="每段文字将生成一个镜头，支持按段落或按行分割"
        type="info"
        show-icon
        :closable="false"
        class="mb-4"
      />
      <el-input
        v-model="importScript"
        type="textarea"
        :rows="10"
        placeholder="在此输入完整的脚本内容..."
      />
      <div class="mt-3" style="display: flex; align-items: center">
        <el-radio-group v-model="splitMode">
          <el-radio value="paragraph">按段落分割</el-radio>
          <el-radio value="line">按行分割</el-radio>
        </el-radio-group>
        <span style="font-size: 13px; color: #c0c4cc; margin-left: 12px">
          将生成 {{ previewCount }} 个镜头
        </span>
      </div>
      <template #footer>
        <el-button @click="showImportDialog = false">取消</el-button>
        <el-button type="primary" @click="doImport" :disabled="previewCount === 0" :loading="importing">
          导入并创建 {{ previewCount }} 个镜头
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showPreview" title="视频预览" width="600px">
      <video v-if="previewUrl" :src="previewUrl" controls style="width: 100%; max-height: 400px" />
    </el-dialog>
  </div>
</template>

<style scoped>
.storyboard-header {
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
.project-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}
.project-card-header h3 {
  margin: 0;
  font-size: 16px;
}
.project-card-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 12px;
  color: #909399;
  font-size: 13px;
}
.project-card-actions {
  display: flex;
  gap: 2px;
}
.detail-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.detail-header h3 {
  flex: 1;
  margin: 0;
}
.detail-actions {
  display: flex;
  gap: 8px;
}
.shots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
  margin-top: 16px;
}
.generate-result {
  margin-bottom: 16px;
}
.script-preview {
  background: #f5f7fa;
  padding: 12px 16px;
  border-radius: 8px;
  margin-bottom: 16px;
}
.script-preview h3 {
  margin: 0 0 8px;
  font-size: 14px;
  color: #606266;
}
.script-preview p {
  margin: 0;
  white-space: pre-wrap;
  color: #303133;
  font-size: 14px;
  line-height: 1.6;
}
.add-shot {
  margin-top: 24px;
  text-align: center;
}
.mb-4 {
  margin-bottom: 16px;
}
</style>
