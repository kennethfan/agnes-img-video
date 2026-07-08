import { defineStore } from 'pinia'

export interface NovelData {
  theme: string
  genre: string
  characters: { name: string; personality: string; appearance: string }[]
  outline: string
  chapters: { title: string; content: string; illustration?: string }[]
}

export interface ImageRefineData {
  sourceType: 'generate' | 'upload'
  sourcePrompt: string
  sourceImage: string
  sourceFile?: File
  refinePrompt: string
  strength: number
  size: string
  resultImage: string
}

export interface ComicData {
  theme: string
  layout: 'single' | 'dual' | 'quad' | 'six'
  panels: { prompt: string; image: string; caption: string }[]
}

export type WorkflowType = 'image_refine' | 'comic' | 'novel'

export const useWizardStore = defineStore('wizard', {
  state: () => ({
    workflow: null as WorkflowType | null,
    step: 0,
    totalSteps: 0,
    novel: {
      theme: '',
      genre: '',
      characters: [] as NovelData['characters'],
      outline: '',
      chapters: [] as NovelData['chapters'],
    } as NovelData,
    image: {
      sourceType: 'generate' as ImageRefineData['sourceType'],
      sourcePrompt: '',
      sourceImage: '',
      refinePrompt: '',
      strength: 0.75,
      size: '1024x1024',
      resultImage: '',
    } as ImageRefineData,
    comic: {
      theme: '',
      layout: 'quad' as ComicData['layout'],
      panels: [],
    } as ComicData,
  }),
  getters: {
    stepConfig: (state) => {
      const configs: Record<WorkflowType, string[]> = {
        image_refine: ['选择来源', '精修调参', '对比预览', '导出'],
        comic: ['设定主题', '选择布局', '填写提示词', '批量生成', '添加台词', '导出'],
        novel: ['选题设定', '风格选择', '角色设置', '大纲生成', '确认大纲', '逐章生成', '配插图', '导出'],
      }
      if (!state.workflow) return []
      return configs[state.workflow]
    },
    currentStepLabel(): string {
      const labels = this.stepConfig
      return labels[this.step] || ''
    },
    isFirstStep: (state) => state.step === 0,
    isLastStep: (state) => state.step === state.totalSteps - 1,
  },
  actions: {
    startWorkflow(type: WorkflowType) {
      this.workflow = type
      this.step = 0
      this.totalSteps = this.stepConfig.length
    },
    nextStep() {
      if (this.step < this.totalSteps - 1) this.step++
    },
    prevStep() {
      if (this.step > 0) this.step--
    },
    goToStep(n: number) {
      if (n >= 0 && n < this.totalSteps) this.step = n
    },
    reset() {
      this.workflow = null
      this.step = 0
      this.totalSteps = 0
    },
  },
})
