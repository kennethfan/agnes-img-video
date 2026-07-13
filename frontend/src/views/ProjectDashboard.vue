<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import { getProject, getProjectFiles, getProjectStats } from '../api/projects'
import type { Project, ProjectFile, ProjectStats } from '../types'
import StepProgressBar from '../components/StepProgressBar.vue'
import ProjectStatsCards from '../components/ProjectStatsCards.vue'
import ProjectFileGrid from '../components/ProjectFileGrid.vue'

const route = useRoute()
const router = useRouter()
const projectId = Number(route.params.id)

const project = ref<Project | null>(null)
const files = ref<ProjectFile[]>([])
const stats = ref<ProjectStats | null>(null)
const loading = ref(true)

const stepConfig = [
  { key: 'ideate', label: '创意发想', status: 'pending' },
  { key: 'generate', label: '生成', status: 'pending' },
  { key: 'refine', label: '优化', status: 'pending' },
  { key: 'finalize', label: '定稿', status: 'pending' },
]

async function loadData() {
  loading.value = true
  try {
    const [p, f, s] = await Promise.all([
      getProject(projectId),
      getProjectFiles(projectId),
      getProjectStats(projectId),
    ])
    project.value = p
    files.value = f
    stats.value = s

    if (s.step_progress) {
      stepConfig.forEach(st => {
        st.status = s.step_progress[st.key] || 'pending'
      })
    }
  } catch (e: any) {
    ElMessage.error('加载仪表盘失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push(`/project-editor/${projectId}`)
}

const previewFile = ref<ProjectFile | null>(null)
const previewVisible = computed(() => previewFile.value !== null)

function onPreview(file: ProjectFile) {
  previewFile.value = file
}

function closePreview() {
  previewFile.value = null
}

onMounted(loadData)
</script>

<template>
  <div class="dashboard">
    <div class="dashboard-header">
      <el-button text :icon="ArrowLeft" @click="goBack">返回编辑器</el-button>
      <h2 v-if="project" class="dashboard-title">{{ project.title }} — 仪表盘</h2>
      <el-tag v-if="project" :type="project.status === 'completed' ? 'success' : 'warning'">
        {{ project.status === 'completed' ? '已完成' : project.status === 'generating' ? '生成中' : '草稿' }}
      </el-tag>
    </div>

    <div v-loading="loading" class="dashboard-content">
      <el-card shadow="never" class="section-card">
        <template #header><span>步骤进度</span></template>
        <StepProgressBar :steps="stepConfig" />
      </el-card>

      <ProjectStatsCards :stats="stats" />

      <el-card shadow="never" class="section-card">
        <template #header><span>生成文件</span></template>
        <ProjectFileGrid :files="files" :projectId="projectId" @preview="onPreview" />
      </el-card>
    </div>
  </div>

  <el-dialog
    v-model="previewVisible"
    :title="previewFile?.prompt || '预览'"
    width="80%"
    top="5vh"
    destroy-on-close
    @close="closePreview"
  >
    <div style="text-align: center">
      <img
        v-if="previewFile?.type === 'image'"
        :src="previewFile?.url"
        style="max-width: 100%; max-height: 75vh; border-radius: 6px"
      />
      <video
        v-else
        :src="previewFile?.url"
        controls
        style="max-width: 100%; max-height: 75vh; border-radius: 6px"
      />
    </div>
  </el-dialog>
</template>

<style scoped>
.dashboard {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}
.dashboard-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}
.dashboard-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  flex: 1;
}
.dashboard-content {
  min-height: 400px;
}
.section-card {
  margin-bottom: 20px;
}
</style>
