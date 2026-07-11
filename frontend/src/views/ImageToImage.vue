<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, Link, Picture } from '@element-plus/icons-vue'
import { submitImageToImage } from '../api/image'
import { getAssets, saveAsset } from '../api/assets'
import type { AssetItem } from '../types'
import { connectTaskSSE } from '../utils/sse'
import ImageResult from '../components/ImageResult.vue'
import TaskProgress from '../components/TaskProgress.vue'
import { useRedoStore } from '../stores/redo'

const inputMode = ref<'upload' | 'url'>('upload')
const prompt = ref('')
const negativePrompt = ref('')
const size = ref('1024x1024')
const strength = ref(0.75)
const loading = ref(false)
const showProgress = ref(false)
const taskId = ref<number | string>('')
const images = ref<string[]>([])
const errorMsg = ref('')
const file = ref<File | null>(null)
const imageUrl = ref('')
const filePreviewUrl = ref('')
const galleryDialogVisible = ref(false)
const assetList = ref<AssetItem[]>([])
const assetLoading = ref(false)
const savingInput = ref(false)
const redoStore = useRedoStore()

const sizeOptions = [
  { value: '1024x1024', label: '1024x1024 (1:1)' },
  { value: '1024x1792', label: '1024x1792 (9:16)' },
  { value: '1792x1024', label: '1792x1024 (16:9)' },
]

const previewUrl = computed(() => {
  if (inputMode.value === 'upload') {
    return filePreviewUrl.value
  }
  return imageUrl.value.trim()
})

// 监听重做数据（flush: sync 确保同步触发）
watch(() => redoStore.redoData, (newData) => {
  if (newData && newData.mode === 'image2image') {
    prompt.value = newData.prompt || ''
    negativePrompt.value = newData.negativePrompt || ''
    size.value = newData.size || '1024x1024'
    strength.value = newData.strength || 0.75
    inputMode.value = newData.inputMode || 'url'
    imageUrl.value = newData.imageUrl || ''
  }
}, { flush: 'sync' })

function handleFileChange(uploadFile: any) {
  // 清理旧的预览URL
  if (filePreviewUrl.value) {
    URL.revokeObjectURL(filePreviewUrl.value)
  }
  file.value = uploadFile.raw || null
  // 创建新的预览URL
  if (file.value) {
    filePreviewUrl.value = URL.createObjectURL(file.value)
  } else {
    filePreviewUrl.value = ''
  }
}

// 切换输入模式时清理预览
watch(inputMode, () => {
  if (inputMode.value === 'url' && filePreviewUrl.value) {
    URL.revokeObjectURL(filePreviewUrl.value)
    filePreviewUrl.value = ''
  }
})

async function openGallery() {
  galleryDialogVisible.value = true
  if (assetList.value.length > 0) return // 已有缓存不重复加载
  assetLoading.value = true
  try {
    const res = await getAssets({ type: 'image', per_page: 50 })
    assetList.value = res.items || []
  } catch (e: any) {
    ElMessage.error('加载作品库失败: ' + (e.message || '未知错误'))
  } finally {
    assetLoading.value = false
  }
}

function selectAsset(item: AssetItem) {
  const url = item.local_path || item.github_url || item.original_url
  if (!url) {
    ElMessage.warning('该图片没有可用的 URL')
    return
  }
  imageUrl.value = url
  inputMode.value = 'url'
  galleryDialogVisible.value = false
  ElMessage.success('已选择图片')
}

async function saveInputImage() {
  const url = imageUrl.value.trim() || (inputMode.value === 'upload' && filePreviewUrl.value ? filePreviewUrl.value : '')
  if (!url) return
  savingInput.value = true
  try {
    await saveAsset({ image_url: url, prompt: prompt.value, mode: 'image2image' })
    ElMessage.success('已保存到作品库')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    savingInput.value = false
  }
}

async function handleGenerate() {
  const source = inputMode.value === 'upload' ? file.value : imageUrl.value.trim()
  if (!source) {
    ElMessage.warning(inputMode.value === 'upload' ? '请上传图片' : '请输入图片 URL')
    return
  }
  if (!prompt.value.trim()) {
    ElMessage.warning('请输入风格描述')
    return
  }
  loading.value = true
  errorMsg.value = ''
  images.value = []
  taskId.value = ''
  showProgress.value = false

  try {
    const res = await submitImageToImage(
      source,
      prompt.value,
      size.value,
      strength.value,
      negativePrompt.value
    )
    taskId.value = res.taskId
    showProgress.value = true

    connectTaskSSE(res.taskId, {
      onComplete: (data) => {
        showProgress.value = false
        try {
          images.value = JSON.parse(data.result)
        } catch {
          images.value = data.result ? [data.result] : []
        }
        loading.value = false
      },
      onError: (data) => {
        showProgress.value = false
        errorMsg.value = data.error
        loading.value = false
      },
    })
  } catch (e: any) {
    errorMsg.value = e.message || '提交失败'
    loading.value = false
  }
}
</script>

<template>
  <div class="gen-page">
    <div class="gen-input">
      <h3 class="gen-title">图生图</h3>
      <el-form label-width="100px">
        <el-form-item label="输入方式">
          <el-radio-group v-model="inputMode">
            <el-radio-button value="upload">
              <el-icon style="vertical-align: middle"><upload-filled /></el-icon>
              <span style="vertical-align: middle">上传图片</span>
            </el-radio-button>
            <el-radio-button value="url">
              <el-icon style="vertical-align: middle"><Link /></el-icon>
              <span style="vertical-align: middle">图片 URL</span>
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label=" ">
          <el-button @click="openGallery" :icon="Picture" size="small">
            从作品库选择
          </el-button>
        </el-form-item>

        <el-form-item v-if="inputMode === 'upload'" label="上传图片">
          <el-upload
            drag
            accept="image/*"
            :auto-upload="false"
            :limit="1"
            :on-change="handleFileChange"
          >
            <el-icon class="el-icon--upload" style="font-size: 48px">
              <upload-filled />
            </el-icon>
            <div class="el-upload__text">拖拽图片到此处或 <em>点击上传</em></div>
          </el-upload>
        </el-form-item>

        <el-form-item v-if="inputMode === 'url'" label="图片 URL">
          <el-input
            v-model="imageUrl"
            placeholder="请输入图片公网 URL，如 https://example.com/image.png"
            clearable
          />
        </el-form-item>

        <el-form-item v-if="previewUrl" label="预览">
          <div style="display: flex; flex-direction: column; align-items: center; gap: 8px; width: 100%;">
            <el-image
              :src="previewUrl"
              fit="contain"
              style="max-width: 100%; max-height: 200px; border-radius: var(--radius-sm); border: 1px solid var(--border-default)"
              :preview-src-list="[previewUrl]"
            />
            <el-button size="small" type="success" :loading="savingInput" :disabled="savingInput" @click="saveInputImage">
              保存到作品库
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="风格描述">
          <el-input
            v-model="prompt"
            type="textarea"
            :rows="3"
            placeholder="描述你想要的风格变化..."
          />
        </el-form-item>
        <el-form-item label="负面提示词">
          <el-input
            v-model="negativePrompt"
            type="textarea"
            :rows="2"
            placeholder="不想出现在图片中的内容..."
          />
        </el-form-item>
        <el-form-item label="尺寸">
          <el-select v-model="size" style="width: 250px">
            <el-option v-for="opt in sizeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="重绘强度">
          <el-slider v-model="strength" :min="0" :max="1" :step="0.05" style="width: 100%" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" size="large" @click="handleGenerate" style="width: 100%">
            生成图片
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gen-preview">
      <TaskProgress v-if="showProgress && taskId" :task-id="taskId" @error="errorMsg = $event" />
      <el-alert v-if="errorMsg" type="error" :description="errorMsg" show-icon closable class="error-alert" />
      <ImageResult :images="images" :loading="loading && !showProgress" :prompt="prompt" mode="image2image" />
    </div>

    <el-dialog
      v-model="galleryDialogVisible"
      title="从作品库选择图片"
      width="700px"
      :close-on-click-modal="false"
    >
      <div v-loading="assetLoading" style="min-height: 200px">
        <div v-if="assetList.length === 0 && !assetLoading" style="text-align: center; padding: 40px; color: #909399">
          作品库暂无图片
        </div>
        <div v-else style="display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 12px;">
          <div
            v-for="item in assetList"
            :key="item.id"
            style="cursor: pointer; border: 2px solid transparent; border-radius: 8px; overflow: hidden; transition: border-color 0.2s;"
            @click="selectAsset(item)"
          >
            <el-image
              :src="item.thumbnail || item.local_path || item.original_url"
              fit="cover"
              style="width: 100%; height: 140px; display: block;"
            />
            <div style="padding: 4px 6px; font-size: 12px; color: #666; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">
              {{ item.prompt || '无描述' }}
            </div>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.gen-page {
  display: flex;
  gap: 24px;
  min-height: 500px;
}
.gen-input {
  flex: 1;
  max-width: 480px;
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
}
.gen-preview {
  flex: 1;
  padding: 20px;
  background: var(--bg-subtle);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  display: flex;
  flex-direction: column;
}
.gen-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-muted);
  margin: 0 0 16px 0;
}
.error-alert {
  margin-bottom: 12px;
}
</style>
