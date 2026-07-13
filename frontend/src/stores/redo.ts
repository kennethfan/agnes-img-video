import { defineStore } from 'pinia'
import { ref } from 'vue'
import router from '../router'

// 重做数据接口
export interface RedoData {
  mode: string
  // 文生图参数
  prompt?: string
  negativePrompt?: string
  size?: string
  n?: number
  // 图生图/图生视频参数
  inputMode?: 'upload' | 'url'
  imageUrl?: string
  strength?: number
  // 批量生成参数
  promptsText?: string
  // 脚本生成参数
  topic?: string
  style?: string
  language?: string
  script?: string
  // 视频参数
  duration?: number
  aspectRatio?: string
  frameRate?: number
  // 多图视频参数
  imageUrlsText?: string
  videoMode?: string
}

// 模式到路由 name 的映射
export const modeToTab: Record<string, string> = {
  text2image: 'text2img',
  image2image: 'img2img',
  batch: 'batch',
  script_gen: 'script_gen',
  text2video: 'text2vid',
  image2video: 'img2vid',
  multi_image_video: 'multi_vid',
  image_refine: 'image_refine',
  comic: 'comic',
  novel: 'novel',
}

export const useRedoStore = defineStore('redo', () => {
  const redoData = ref<RedoData | null>(null)
  const targetTab = ref('')

  // 设置重做数据
  function setRedoData(data: RedoData) {
    redoData.value = data
    targetTab.value = modeToTab[data.mode] || ''

    // 保存到 sessionStorage（各页面 onMounted 时消费）
    sessionStorage.setItem('redoData', JSON.stringify(data))

    // 路由跳转到目标页面
    const routeName = modeToTab[data.mode]
    if (routeName) {
	    router.push({ name: routeName })
    }
  }

  // 获取并清除重做数据（一次性消费）
  function consumeRedoData(): { data: RedoData; tab: string } | null {
    // 优先从 sessionStorage 读取
    const stored = sessionStorage.getItem('redoData')
    if (stored) {
      sessionStorage.removeItem('redoData')
      const data = JSON.parse(stored) as RedoData
      const tab = modeToTab[data.mode] || ''
      return { data, tab }
    }

    if (!redoData.value || !targetTab.value) return null
    const result = {
      data: redoData.value,
      tab: targetTab.value,
    }
    redoData.value = null
    targetTab.value = ''
    return result
  }

  // 清除重做数据
  function clearRedoData() {
    redoData.value = null
    targetTab.value = ''
    sessionStorage.removeItem('redoData')
  }

  return {
    redoData,
    targetTab,
    setRedoData,
    consumeRedoData,
    clearRedoData,
  }
})
