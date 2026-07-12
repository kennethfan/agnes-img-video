<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import NavSidebar from './components/NavSidebar.vue'
import TextToImage from './views/TextToImage.vue'
import ImageToImage from './views/ImageToImage.vue'
import BatchGen from './views/BatchGen.vue'
import ScriptGen from './views/ScriptGen.vue'
import TextToVideo from './views/TextToVideo.vue'
import ImageToVideo from './views/ImageToVideo.vue'
import MultiImageVideo from './views/MultiImageVideo.vue'
import History from './views/History.vue'
import Ideas from './views/Ideas.vue'
import Assets from './views/Assets.vue'
import Storyboard from './views/Storyboard.vue'
import AccessLogs from './views/AccessLogs.vue'
import DBManage from './views/DBManage.vue'
import TaskRecords from './views/TaskRecords.vue'
import Settings from './views/Settings.vue'
import WorkflowWizard from './views/WorkflowWizard.vue'
import TemplateManager from './views/TemplateManager.vue'
import ProjectList from './views/ProjectList.vue'
import ProjectEditor from './views/ProjectEditor.vue'
import type { Project } from './types'

const route = useRoute()
const router = useRouter()
const activePage = ref('text2img')
const currentProjectId = ref<number>(0)

// 路由变化 → 同步 activePage（浏览器前进/后退 & 刷新恢复）
watch(() => route.name, (name) => {
  if (name && typeof name === 'string') {
    activePage.value = name
    if (name === 'project_editor') {
      currentProjectId.value = Number(route.params.id) || 0
    }
  }
}, { immediate: true })

// 页面切换：同步 activePage + URL
function navigateTo(page: string) {
  activePage.value = page
  if (page === 'project_editor') return // project_editor 通过 router.push 带参数导航
  router.push({ name: page })
}

function openProjectEditor(project: Project) {
  currentProjectId.value = project.id
  activePage.value = 'project_editor'
  router.push({ name: 'project_editor', params: { id: project.id } })
}

function backToProjects() {
  activePage.value = 'projects'
  router.push({ name: 'projects' })
}
</script>

<template>
  <div class="app-layout">
    <header class="top-bar">
      <span class="app-title">Agnes Creator Studio</span>
      <span class="app-subtitle">AI Image &amp; Video Studio</span>
    </header>

    <div class="app-body">
      <NavSidebar :active-page="activePage" @navigate="navigateTo" />

      <main class="main-content">
        <TextToImage v-if="activePage === 'text2img'" />
        <ImageToImage v-else-if="activePage === 'img2img'" />
        <BatchGen v-else-if="activePage === 'batch'" />
        <ScriptGen v-else-if="activePage === 'script_gen'" />
        <TextToVideo v-else-if="activePage === 'text2vid'" />
        <ImageToVideo v-else-if="activePage === 'img2vid'" />
        <MultiImageVideo v-else-if="activePage === 'multi_vid'" />
        <Ideas v-else-if="activePage === 'ideas'" />
        <Storyboard v-else-if="activePage === 'storyboard'" />
        <Assets v-else-if="activePage === 'assets'" />
        <TaskRecords v-else-if="activePage === 'tasks'" />
        <History v-else-if="activePage === 'history'" />
        <AccessLogs v-else-if="activePage === 'access_logs'" />
        <DBManage v-else-if="activePage === 'db_manage'" />
        <Settings v-else-if="activePage === 'settings'" />
        <ProjectList v-else-if="activePage === 'projects'" @edit-project="openProjectEditor" />
        <ProjectEditor v-else-if="activePage === 'project_editor'" :project-id="currentProjectId" @back="backToProjects" />
        <WorkflowWizard v-else-if="activePage === 'image_refine'" />
        <WorkflowWizard v-else-if="activePage === 'comic'" />
        <WorkflowWizard v-else-if="activePage === 'novel'" />
        <TemplateManager v-else-if="activePage === 'templates'" />
      </main>
    </div>
  </div>
</template>

<style>
* {
  box-sizing: border-box;
}
:root {
  --bg-page: #ffffff;
  --bg-subtle: #fafafa;
  --bg-card: #ffffff;
  --border-default: #eaeaea;
  --border-light: #f0f0f0;
  --text-primary: #000000;
  --text-secondary: #666666;
  --text-muted: #909399;
  --accent: #000000;
  --accent-hover: #333333;
  --radius-card: 12px;
  --radius-sm: 8px;
  --shadow-card: none;
}
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
    'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
  background: var(--bg-page);
  color: var(--text-primary);
}
</style>

<style scoped>
.app-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.top-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 20px;
  border-bottom: 1px solid #f0f0f0;
  background: #ffffff;
}
.app-title {
  font-weight: 600;
  font-size: 15px;
  color: #000;
}
.app-subtitle {
  font-size: 12px;
  color: #909399;
}
.app-body {
  display: flex;
  flex: 1;
}
.main-content {
  flex: 1;
  padding: 24px;
  max-width: 1200px;
  overflow-y: auto;
}
</style>
