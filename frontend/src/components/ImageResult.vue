<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { saveAsset } from '../api/assets'
import { useRedoStore } from '../stores/redo'
const props = defineProps<{
  images: string[]
  loading: boolean
  prompt: string
  mode: string
}>()

const savingUrls = ref<Set<string>>(new Set())
const redoStore = useRedoStore()

// 自动保存开关（默认开启，通过 localStorage 持久化）
const autoSaveEnabled = ref(localStorage.getItem('autoSaveToGallery') !== 'false')
// 已自动保存的图片 URL 集合（避免重复保存）
const autoSavedUrls = ref<Set<string>>(new Set())

// images 变化时自动静默保存
watch(() => props.images, (newImages) => {
  if (!autoSaveEnabled.value) return
  newImages.forEach(img => {
    if (!autoSavedUrls.value.has(img)) {
      autoSavedUrls.value = new Set([...autoSavedUrls.value, img])
      saveAsset({ image_url: img, prompt: props.prompt, mode: props.mode }).catch(() => {
        // 静默失败，不影响用户体验
      })
    }
  })
}, { immediate: true })

function handleRefine(img: string) {
  redoStore.setRedoData({
    mode: 'image2image',
    imageUrl: img,
    prompt: props.prompt,
    inputMode: 'url',
  })
}

function downloadImage(url: string) {
  window.open('/api/v1/download?url=' + encodeURIComponent(url), '_blank')
}

async function handleSaveToGallery(url: string) {
  savingUrls.value = new Set([...savingUrls.value, url])
  try {
    await saveAsset({ image_url: url, prompt: props.prompt, mode: props.mode })
    ElMessage.success('已保存到作品库')
  } catch (e: any) {
    ElMessage.error(e.message || '保存到作品库失败')
  } finally {
    const next = new Set(savingUrls.value)
    next.delete(url)
    savingUrls.value = next
  }
}
</script>

<template>
  <div v-if="loading" style="text-align: center; padding: 40px">
    <el-skeleton :rows="3" animated />
    <p style="color: #909399; margin-top: 12px">生成中...</p>
  </div>

  <div v-else-if="images.length > 0" class="image-gallery">
    <div v-for="(img, idx) in images" :key="idx" class="image-card">
      <el-image
        :src="img"
        :preview-src-list="images"
        fit="contain"
        style="width: 100%; height: 300px"
      />
      <div class="image-actions">
        <el-button type="primary" size="small" @click="downloadImage(img)">
          下载
        </el-button>
        <el-button
          v-if="autoSaveEnabled && autoSavedUrls.has(img)"
          size="small"
          type="info"
          disabled
        >
          已保存
        </el-button>
        <el-button
          v-else
          size="small"
          type="success"
          :loading="savingUrls.has(img)"
          :disabled="savingUrls.has(img)"
          @click="handleSaveToGallery(img)"
        >
          保存到作品库
        </el-button>
        <el-button
          v-if="props.mode === 'image2image'"
          size="small"
          @click="handleRefine(img)"
        >
          继续精修
        </el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.image-gallery {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
  margin-top: 16px;
}
.image-card {
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  padding: 12px;
}
.image-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
  margin-top: 8px;
}
</style>
