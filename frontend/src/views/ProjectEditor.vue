<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowLeft, ArrowRight, MagicStick, Picture, Edit, Check } from '@element-plus/icons-vue'
import { getProject, aiRecommend } from '../api/projects'
import type { Project } from '../types'
import AIPanel from '../components/AIPanel.vue'
import GenStep from '../components/GenStep.vue'
import RefineStep from '../components/RefineStep.vue'
import FinalStep from '../components/FinalStep.vue'

const props = defineProps<{ projectId: number }>()
const emit = defineEmits<{ back: [] }>()

const steps = ['ai', 'generate', 'refine', 'finalize'] as const
type Step = typeof steps[number]
const stepLabels: Record<Step, string> = { ai: 'AI 推荐', generate: '生成', refine: '优化', finalize: '定稿' }
const stepIcons: Record<Step, any> = { ai: MagicStick, generate: Picture, refine: Edit, finalize: Check }

const currentStep = ref<Step>('ai')
const project = ref<Project | null>(null)
const loading = ref(false)
const aiResult = ref('')
const aiLoading = ref(false)
const latestGenUrls = ref<string[]>([])

async function loadProject() {
  loading.value = true
  try {
    project.value = await getProject(props.projectId)
    if (project.value.ai_result) {
      try {
        const parsed = JSON.parse(project.value.ai_result)
        aiResult.value = parsed.result || project.value.ai_result
      } catch {
        aiResult.value = project.value.ai_result
      }
    }
  } catch (e: any) {
    ElMessage.error('加载项目失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

onMounted(loadProject)

watch(() => props.projectId, loadProject)

async function runAIRecommend() {
  if (!project.value) return
  aiLoading.value = true
  try {
    const res = await aiRecommend(project.value.id)
    aiResult.value = res.result
    ElMessage.success('AI 推荐完成')
    await loadProject()
  } catch (e: any) {
    ElMessage.error('AI 推荐失败: ' + (e.message || ''))
  } finally {
    aiLoading.value = false
  }
}

function goStep(s: Step) {
  currentStep.value = s
}

function prevStep() {
  const idx = steps.indexOf(currentStep.value)
  if (idx > 0) goStep(steps[idx - 1])
}

function nextStep() {
  const idx = steps.indexOf(currentStep.value)
  if (idx < steps.length - 1) goStep(steps[idx + 1])
}
</script>

<template>
  <div v-loading="loading">
    <div class="editor-header">
      <el-button text @click="emit('back')">
        <el-icon><ArrowLeft /></el-icon> 返回列表
      </el-button>
      <h3 v-if="project">{{ project.title || '未命名项目' }}</h3>
    </div>

    <!-- 步骤导航 -->
    <div class="step-nav">
      <div
        v-for="(s, i) in steps"
        :key="s"
        class="step-item"
        :class="{ active: currentStep === s, done: steps.indexOf(currentStep) > i }"
        @click="goStep(s)"
      >
        <div class="step-dot">
          <el-icon v-if="steps.indexOf(currentStep) > i"><Check /></el-icon>
          <el-icon v-else><component :is="stepIcons[s]" /></el-icon>
        </div>
        <span>{{ stepLabels[s] }}</span>
        <div v-if="i < steps.length - 1" class="step-line" />
      </div>
    </div>

    <!-- 步骤内容 -->
    <div class="step-content">
      <!-- AI 推荐 -->
      <div v-if="currentStep === 'ai'" class="step-body">
        <AIPanel
          :project="project"
          :aiResult="aiResult"
          :loading="aiLoading"
          @recommend="runAIRecommend"
        />
      </div>

      <!-- 生成 -->
      <div v-if="currentStep === 'generate'" class="step-body">
        <GenStep :project="project" @generated="latestGenUrls = $event" />
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
      <el-button v-if="currentStep !== 'ai'" @click="prevStep">
        <el-icon><ArrowLeft /></el-icon> 上一步
      </el-button>
      <el-button v-if="currentStep !== 'finalize'" type="primary" @click="nextStep">
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
