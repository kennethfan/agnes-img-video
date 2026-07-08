<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { uploadToGitHub } from '../api/github'

const props = defineProps<{
  images: string[]
  loading: boolean
}>()

const uploadingUrls = ref<Set<string>>(new Set())

function downloadImage(url: string) {
  window.open('/api/v1/download?url=' + encodeURIComponent(url), '_blank')
}

async function handleUploadToGitHub(url: string) {
  uploadingUrls.value = new Set([...uploadingUrls.value, url])
  try {
    const githubUrl = await uploadToGitHub(url)
    ElMessage.success(`已上传到 GitHub: ${githubUrl}`)
  } catch (e: any) {
    ElMessage.error(e.message || '上传到 GitHub 失败')
  } finally {
    const next = new Set(uploadingUrls.value)
    next.delete(url)
    uploadingUrls.value = next
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
          :loading="uploadingUrls.has(img)"
          :disabled="uploadingUrls.has(img)"
          @click="handleUploadToGitHub(img)"
        >
          转存
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
