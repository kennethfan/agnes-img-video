<script setup lang="ts">
defineProps<{
  images: string[]
  loading: boolean
}>()

function downloadImage(url: string) {
  const a = document.createElement('a')
  a.href = url
  a.download = url.split('/').pop() || 'image.png'
  a.click()
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
      <div style="text-align: center; margin-top: 8px">
        <el-button type="primary" size="small" @click="downloadImage(img)">
          下载
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
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 12px;
}
</style>
