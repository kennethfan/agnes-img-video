<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Picture, VideoCamera } from '@element-plus/icons-vue'
import { textToImage, imageToImage } from '../api/image'
import ImageResult from './ImageResult.vue'
import AssetPickerDialog from './AssetPickerDialog.vue'
import type { Project } from '../types'

const props = defineProps<{ project: Project | null; initialPrompt?: string }>()
const emit = defineEmits<{
  generated: [urls: string[]]
}>()

const prompt = ref(props.initialPrompt || sessionStorage.getItem('ideate_prompt') || '')
// 读完后清理，避免影响后续项目
sessionStorage.removeItem('ideate_prompt')

// 响应式同步外部传入的 prompt
watch(() => props.initialPrompt, (val) => {
  if (val) prompt.value = val
})
const mode = ref<'text2image' | 'image2image'>('text2image')
const imageUrl = ref('')
const size = ref('1024x1024')
const strength = ref(0.75)
const showAssetPicker = ref(false)

function onAssetSelected(url: string) {
  imageUrl.value = url
}

const generating = ref(false)
const resultUrls = ref<string[]>([])

async function generate() {
  if (!prompt.value.trim()) {
    ElMessage.warning('请输入提示词')
    return
  }
  generating.value = true
  resultUrls.value = []
  try {
    if (mode.value === 'text2image') {
      const resp = await textToImage({ prompt: prompt.value, size: size.value, n: 1 })
      if (resp.images?.length) {
        resultUrls.value = resp.images
      }
    } else {
      if (!imageUrl.value) {
        ElMessage.warning('请输入参考图片 URL')
        return
      }
      const resp = await imageToImage(imageUrl.value, prompt.value, size.value, strength.value)
      if (resp.images?.length) {
        resultUrls.value = resp.images
      }
    }
    if (resultUrls.value.length > 0) {
      emit('generated', resultUrls.value)
    }
  } catch (e: any) {
    ElMessage.error('生成失败: ' + (e.message || ''))
  } finally {
    generating.value = false
  }
}
</script>

<template>
  <div class="gen-step">
    <div class="step-intro">
      <h3>生成内容</h3>
      <p>使用 AI 生成图片，作为创作项目的内容素材。</p>
    </div>

    <div class="gen-form">
      <el-radio-group v-model="mode" class="mode-toggle">
        <el-radio-button value="text2image">
          <el-icon><Picture /></el-icon> 文生图
        </el-radio-button>
        <el-radio-button value="image2image">
          <el-icon><VideoCamera /></el-icon> 图生图
        </el-radio-button>
      </el-radio-group>

      <el-input
        v-model="prompt"
        type="textarea"
        :rows="3"
        placeholder="输入画面描述提示词..."
      />

      <div v-if="mode === 'image2image'" class="form-row">
        <el-input v-model="imageUrl" placeholder="参考图片 URL" />
        <el-button @click="showAssetPicker = true" :icon="Picture">从作品库</el-button>
        <div class="form-row-inner">
          <el-select v-model="size" style="width: 140px">
            <el-option label="1024x1024" value="1024x1024" />
            <el-option label="1152x768" value="1152x768" />
            <el-option label="768x1152" value="768x1152" />
          </el-select>
          <el-slider
            v-model="strength"
            :min="0"
            :max="1"
            :step="0.05"
            style="width: 160px"
            show-input
          />
        </div>
      </div>

      <div v-else class="form-row">
        <el-select v-model="size" style="width: 140px">
          <el-option label="1024x1024" value="1024x1024" />
          <el-option label="1152x768" value="1152x768" />
          <el-option label="768x1152" value="768x1152" />
        </el-select>
      </div>

      <el-button type="primary" :loading="generating" @click="generate" :icon="Picture">
        {{ generating ? '生成中...' : '生成' }}
      </el-button>
    </div>

    <div v-if="resultUrls.length" class="result-section">
      <h4>生成结果</h4>
      <ImageResult :images="resultUrls" :loading="false" prompt="" mode="" />
    </div>
  </div>

  <AssetPickerDialog
    v-model:visible="showAssetPicker"
    @selected="onAssetSelected"
  />
</template>

<style scoped>
.gen-step {
  max-width: 700px;
  margin: 0 auto;
}
.step-intro {
  margin-bottom: 24px;
}
.step-intro h3 {
  margin: 0 0 8px;
}
.step-intro p {
  margin: 0;
  color: #909399;
}
.gen-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.mode-toggle {
  align-self: flex-start;
}
.form-row {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}
.form-row-inner {
  display: flex;
  gap: 12px;
  align-items: center;
}
.result-section {
  margin-top: 32px;
}
.result-section h4 {
  margin-bottom: 12px;
}
</style>
