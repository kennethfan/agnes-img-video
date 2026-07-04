import { defineStore } from 'pinia'
import { ref } from 'vue'

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

// 模式到tab名的映射
export const modeToTab: Record<string, string> = {
  text2image: 'text2img',
  image2image: 'img2img',
  batch: 'batch',
  script_gen: 'script',
  text2video: 'text2vid',
  image2video: 'img2vid',
  multi_image_video: 'multi-vid',
}

export const useRedoStore = defineStore('redo', () => {
  const redoData = ref<RedoData | null>(null)
  const targetTab = ref('')

  // 设置重做数据
  function setRedoData(data: RedoData) {
    redoData.value = data
    targetTab.value = modeToTab[data.mode] || ''
  }

  // 获取并清除重做数据（一次性消费）
  function consumeRedoData(): { data: RedoData; tab: string } | null {
    if (!redoData.value || !targetTab.value) return null
    const result = {
      data: redoData.value,
      tab: targetTab.value,
    }
    // 清除数据，防止重复使用
    redoData.value = null
    targetTab.value = ''
    return result
  }

  // 清除重做数据
  function clearRedoData() {
    redoData.value = null
    targetTab.value = ''
  }

  return {
    redoData,
    targetTab,
    setRedoData,
    consumeRedoData,
    clearRedoData,
  }
})
