<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { saveAsset } from '../api/assets'

const props = defineProps<{
  images: string[]
  loading: boolean
  prompt: string
  mode: string
}>()

const savingUrls = ref<Set<string>>(new Set())

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
          size="small"
          type="success"
          :loading="savingUrls.has(img)"
          :disabled="savingUrls.has(img)"
          @click="handleSaveToGallery(img)"
        >
          保存到作品库
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
