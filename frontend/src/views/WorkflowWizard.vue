<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useWizardStore } from '../stores/wizard'
import type { WorkflowType } from '../stores/wizard'
import { ArrowLeft, ArrowRight, Close } from '@element-plus/icons-vue'
import { ElSteps, ElStep, ElButton } from 'element-plus'

// Image refine steps
import StepSource from './image/StepSource.vue'
import StepRefine from './image/StepRefine.vue'
import StepCompare from './image/StepCompare.vue'
import StepExport from './image/StepExport.vue'

// Comic steps
import ComicStepTheme from './comic/StepTheme.vue'
import ComicStepLayout from './comic/StepLayout.vue'
import ComicStepPanels from './comic/StepPanels.vue'
import ComicStepGenerate from './comic/StepGenerate.vue'
import ComicStepCaptions from './comic/StepCaptions.vue'
import ComicStepExport from './comic/StepExport.vue'

// Novel steps
import NovelStepTheme from './novel/StepTheme.vue'
import NovelStepGenre from './novel/StepGenre.vue'
import NovelStepCharacters from './novel/StepCharacters.vue'
import NovelStepOutline from './novel/StepOutline.vue'
import NovelStepOutlineConfirm from './novel/StepOutlineConfirm.vue'
import NovelStepGenerateChapters from './novel/StepGenerateChapters.vue'
import NovelStepIllustrate from './novel/StepIllustrate.vue'
import NovelStepExport from './novel/StepExport.vue'

const route = useRoute()
const store = useWizardStore()

const WIZARD_TYPES = ['image_refine', 'comic', 'novel'] as const

function syncWorkflow(type: unknown) {
  if (typeof type !== 'string') return
  if (!WIZARD_TYPES.includes(type as any)) return
  if (store.workflow !== type) {
    store.startWorkflow(type as WorkflowType)
  }
}

// 路由切换时 route.name 可能滞后于 activePage，等路由就绪后再同步
watch(() => route.name, (name) => syncWorkflow(name), { immediate: true })

const workType = computed(() => store.workflow)
const step = computed(() => store.step)
const steps = computed(() => store.stepConfig)
</script>

<template>
  <div class="wizard">
    <div class="wizard__header">
      <h3>创作工作流</h3>
      <el-button text :icon="Close" @click="store.reset()">关闭</el-button>
    </div>

    <div class="wizard__steps">
      <el-steps :active="step" align-center>
        <el-step v-for="(label, i) in steps" :key="i" :title="label" />
      </el-steps>
    </div>

    <div class="wizard__body">
      <!-- Image Refine -->
      <template v-if="workType === 'image_refine'">
        <StepSource v-if="step === 0" />
        <StepRefine v-else-if="step === 1" />
        <StepCompare v-else-if="step === 2" />
        <StepExport v-else-if="step === 3" />
      </template>

      <!-- Comic -->
      <template v-if="workType === 'comic'">
        <ComicStepTheme v-if="step === 0" />
        <ComicStepLayout v-else-if="step === 1" />
        <ComicStepPanels v-else-if="step === 2" />
        <ComicStepGenerate v-else-if="step === 3" />
        <ComicStepCaptions v-else-if="step === 4" />
        <ComicStepExport v-else-if="step === 5" />
      </template>

      <!-- Novel -->
      <template v-if="workType === 'novel'">
        <NovelStepTheme v-if="step === 0" />
        <NovelStepGenre v-else-if="step === 1" />
        <NovelStepCharacters v-else-if="step === 2" />
        <NovelStepOutline v-else-if="step === 3" />
        <NovelStepOutlineConfirm v-else-if="step === 4" />
        <NovelStepGenerateChapters v-else-if="step === 5" />
        <NovelStepIllustrate v-else-if="step === 6" />
        <NovelStepExport v-else-if="step === 7" />
      </template>
    </div>

    <div class="wizard__footer">
      <el-button :disabled="store.isFirstStep" :icon="ArrowLeft" @click="store.prevStep()">
        上一步
      </el-button>
      <el-button v-if="!store.isLastStep" type="primary" :icon="ArrowRight" @click="store.nextStep()">
        下一步
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.wizard { max-width: 800px; margin: 0 auto; }
.wizard__header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 24px; }
.wizard__steps { margin-bottom: 32px; }
.wizard__body { min-height: 300px; }
.wizard__footer { display: flex; justify-content: space-between; margin-top: 24px; padding-top: 16px; border-top: 1px solid #f0f0f0; }
</style>
