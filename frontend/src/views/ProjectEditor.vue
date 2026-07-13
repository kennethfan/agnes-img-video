<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, ArrowRight, Picture, Edit, Check, ChatLineSquare, DataBoard } from '@element-plus/icons-vue'
import { getProject, updateStepProgress, getProjectFiles } from '../api/projects'
import type { Project, ProjectFile } from '../types'
import IdeateStep from '../components/IdeateStep.vue'
import GenStep from '../components/GenStep.vue'
import RefineStep from '../components/RefineStep.vue'
import FinalStep from '../components/FinalStep.vue'

const props = defineProps<{ projectId: number }>()
const emit = defineEmits<{ back: [] }>()

const steps = ['ideate', 'generate', 'refine', 'finalize'] as const
type Step = typeof steps[number]
const stepLabels: Record<Step, string> = { ideate: '创意发想', generate: '生成', refine: '优化', finalize: '定稿' }
const stepIcons: Record<Step, any> = { ideate: ChatLineSquare, generate: Picture, refine: Edit, finalize: Check }

const currentStep = ref<Step>('ideate')
const project = ref<Project | null>(null)
const loading = ref(false)
const latestGenUrls = ref<string[]>([])
const latestBrief = ref('')
const latestGenPrompt = ref('')

function onBriefGenerated(briefText: string, prompt: string) {
  latestBrief.value = briefText
  latestGenPrompt.value = prompt
  // sessionStorage 兜底，避免 v-if 组件重建时 prop 丢失
  sessionStorage.setItem('ideate_prompt', prompt)
}

async function loadProject() {
  loading.value = true
  try {
    project.value = await getProject(props.projectId)
    // 恢复步骤进度：跳转到第一个未完成的步骤
    const sp = project.value.step_progress
    if (sp) {
      try {
        const data = JSON.parse(sp)
        let found = false
        for (const s of steps) {
          if (data[s] !== 'completed') {
            currentStep.value = s
            found = true
            break
          }
        }
        // 全部已完成 → 跳转到定稿页展示完成状态
        if (!found) currentStep.value = 'finalize'
      } catch { /* 忽略解析错误 */ }
    }

    // 加载项目历史图片，供 refine 步骤恢复预览
    try {
      const files: ProjectFile[] = await getProjectFiles(props.projectId)
      const historyImages = files.filter(f => f.source === 'history' && f.type === 'image')
      if (historyImages.length > 0) {
        latestGenUrls.value = historyImages.map(f => f.url)
      }
    } catch { /* 非阻塞 */ }
  } catch (e: any) {
    ElMessage.error('加载项目失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function isStepDone(s: string): boolean {
  if (!project.value?.step_progress) return false
  try {
    const data = JSON.parse(project.value.step_progress)
    return data[s] === 'completed'
  } catch { return false }
}

onMounted(loadProject)

watch(() => props.projectId, loadProject)

function goStep(s: Step) {
  currentStep.value = s
}

function prevStep() {
  const idx = steps.indexOf(currentStep.value)
  if (idx > 0) goStep(steps[idx - 1])
}

const router = useRouter()

async function nextStepWithProgress() {
  if (!project.value) return
  const idx = steps.indexOf(currentStep.value)
  if (idx < steps.length - 1) {
    try {
      await updateStepProgress(project.value.id, currentStep.value, 'completed')
    } catch { /* 非阻塞 */ }
    goStep(steps[idx + 1])
  }
}

function goToDashboard() {
  if (project.value) {
    router.push(`/projects/${project.value.id}/dashboard`)
  }
}
</script>

<template>
  <div v-loading="loading">
    <div class="editor-header">
      <el-button text @click="emit('back')">
        <el-icon><ArrowLeft /></el-icon> 返回列表
      </el-button>
      <el-button text @click="goToDashboard" v-if="project">
        <el-icon><DataBoard /></el-icon> 仪表盘
      </el-button>
      <h3 v-if="project">{{ project.title || '未命名项目' }}</h3>
    </div>

    <!-- 步骤导航 -->
    <div class="step-nav">
      <div
        v-for="(s, i) in steps"
        :key="s"
        class="step-item"
        :class="{ active: currentStep === s, done: isStepDone(s) }"
        @click="goStep(s)"
      >
        <div class="step-dot">
          <el-icon v-if="isStepDone(s)"><Check /></el-icon>
          <el-icon v-else><component :is="stepIcons[s]" /></el-icon>
        </div>
        <span>{{ stepLabels[s] }}</span>
        <div v-if="i < steps.length - 1" class="step-line" />
      </div>
    </div>

    <!-- 步骤内容 -->
    <div class="step-content">
      <!-- 创意发想 -->
      <div v-if="currentStep === 'ideate'" class="step-body">
        <IdeateStep :project="project" @brief-generated="onBriefGenerated" />
      </div>

      <!-- 生成 -->
      <div v-if="currentStep === 'generate'" class="step-body">
        <GenStep :project="project" :initialPrompt="latestGenPrompt" @generated="latestGenUrls = $event" />
      </div>

      <!-- 优化 -->
      <div v-if="currentStep === 'refine'" class="step-body">
        <RefineStep :project="project" :defaultImageUrl="latestGenUrls[0] || ''" />
      </div>

      <!-- 定稿 -->
      <div v-if="currentStep === 'finalize'" class="step-body">
        <FinalStep :project="project" @updated="loadProject" />
      </div>
    </div>

    <!-- 步骤操作 -->
    <div class="step-actions">
      <el-button v-if="currentStep !== 'ideate'" @click="prevStep">
        <el-icon><ArrowLeft /></el-icon> 上一步
      </el-button>
      <el-button v-if="currentStep !== 'finalize'" type="primary" @click="nextStepWithProgress">
        下一步 <el-icon><ArrowRight /></el-icon>
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.editor-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}
.editor-header h3 {
  margin: 0;
  font-size: 18px;
}
.step-nav {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0;
  margin-bottom: 32px;
  padding: 16px 0;
}
.step-item {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  position: relative;
  padding: 0 16px;
  color: #c0c4cc;
  transition: color 0.2s;
}
.step-item.active {
  color: #409eff;
}
.step-item.done {
  color: #67c23a;
}
.step-item:hover {
  color: #409eff;
}
.step-dot {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid currentColor;
  background: #fff;
  font-size: 16px;
}
.step-item.active .step-dot {
  background: #409eff;
  color: #fff;
  border-color: #409eff;
}
.step-item.done .step-dot {
  background: #67c23a;
  color: #fff;
  border-color: #67c23a;
}
.step-line {
  position: absolute;
  left: 100%;
  top: 50%;
  width: 40px;
  height: 2px;
  background: #dcdfe6;
  transform: translateY(-50%);
}
.step-item.active .step-line,
.step-item.done .step-line {
  background: #409eff;
}
.step-content {
  min-height: 300px;
}
.step-body {
  padding: 0 8px;
}
.step-actions {
  display: flex;
  justify-content: space-between;
  margin-top: 32px;
  padding-top: 16px;
  border-top: 1px solid #ebeef5;
}
</style>
