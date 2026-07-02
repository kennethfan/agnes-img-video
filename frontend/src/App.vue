<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useConfigStore } from './stores/config'
import ApiConfig from './components/ApiConfig.vue'
import TextToImage from './views/TextToImage.vue'
import ImageToImage from './views/ImageToImage.vue'
import BatchGen from './views/BatchGen.vue'
import ScriptGen from './views/ScriptGen.vue'
import TextToVideo from './views/TextToVideo.vue'
import ImageToVideo from './views/ImageToVideo.vue'
import MultiImageVideo from './views/MultiImageVideo.vue'
import History from './views/History.vue'

const configStore = useConfigStore()
const activeTab = ref('text2img')

onMounted(async () => {
  await configStore.loadConfig()
})
</script>

<template>
  <el-container style="min-height: 100vh; background: #f5f7fa">
    <el-main style="max-width: 1200px; margin: 0 auto; width: 100%">
      <h1 style="margin-bottom: 8px; font-size: 24px; color: #303133">
        Agnes Creator Studio
      </h1>

      <ApiConfig />

      <el-tabs v-model="activeTab" type="border-card" style="margin-top: 16px">
        <el-tab-pane label="文生图" name="text2img">
          <TextToImage />
        </el-tab-pane>
        <el-tab-pane label="图生图" name="img2img">
          <ImageToImage />
        </el-tab-pane>
        <el-tab-pane label="批量生成" name="batch">
          <BatchGen />
        </el-tab-pane>
        <el-tab-pane label="脚本生成" name="script">
          <ScriptGen />
        </el-tab-pane>
        <el-tab-pane label="文生视频" name="text2vid">
          <TextToVideo />
        </el-tab-pane>
        <el-tab-pane label="图生视频" name="img2vid">
          <ImageToVideo />
        </el-tab-pane>
        <el-tab-pane label="多图视频" name="multi-vid">
          <MultiImageVideo />
        </el-tab-pane>
        <el-tab-pane label="历史记录" name="history">
          <History />
        </el-tab-pane>
      </el-tabs>
    </el-main>
  </el-container>
</template>
