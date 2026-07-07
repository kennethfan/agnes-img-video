<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import NavSidebar from './components/NavSidebar.vue'
import TextToImage from './views/TextToImage.vue'
import ImageToImage from './views/ImageToImage.vue'
import BatchGen from './views/BatchGen.vue'
import ScriptGen from './views/ScriptGen.vue'
import TextToVideo from './views/TextToVideo.vue'
import ImageToVideo from './views/ImageToVideo.vue'
import MultiImageVideo from './views/MultiImageVideo.vue'
import History from './views/History.vue'
import Ideas from './views/Ideas.vue'
import Assets from './views/Assets.vue'
import Storyboard from './views/Storyboard.vue'
import { useRedoStore } from './stores/redo'

const activePage = ref('text2img')
const redoStore = useRedoStore()

function handleRedoTrigger() {
  const tab = redoStore.targetTab
  if (tab) {
    activePage.value = tab
  }
}

onMounted(() => {
  window.addEventListener('redo-trigger', handleRedoTrigger)
})

onUnmounted(() => {
  window.removeEventListener('redo-trigger', handleRedoTrigger)
})
</script>

<template>
  <div class="app-layout">
    <header class="top-bar">
      <span class="app-title">Agnes Creator Studio</span>
      <span class="app-subtitle">AI Image &amp; Video Studio</span>
    </header>

    <div class="app-body">
      <NavSidebar v-model:active-page="activePage" />

      <main class="main-content">
        <TextToImage v-if="activePage === 'text2img'" />
        <ImageToImage v-else-if="activePage === 'img2img'" />
        <BatchGen v-else-if="activePage === 'batch'" />
        <ScriptGen v-else-if="activePage === 'script_gen'" />
        <TextToVideo v-else-if="activePage === 'text2vid'" />
        <ImageToVideo v-else-if="activePage === 'img2vid'" />
        <MultiImageVideo v-else-if="activePage === 'multi_vid'" />
        <Ideas v-else-if="activePage === 'ideas'" />
        <Storyboard v-else-if="activePage === 'storyboard'" />
        <Assets v-else-if="activePage === 'assets'" />
        <History v-else-if="activePage === 'history'" />
      </main>
    </div>
  </div>
</template>

<style>
* {
  box-sizing: border-box;
}
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
    'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
  background: #ffffff;
  color: #000;
}
</style>

<style scoped>
.app-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.top-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 20px;
  border-bottom: 1px solid #f0f0f0;
  background: #ffffff;
}
.app-title {
  font-weight: 600;
  font-size: 15px;
  color: #000;
}
.app-subtitle {
  font-size: 12px;
  color: #909399;
}
.app-body {
  display: flex;
  flex: 1;
}
.main-content {
  flex: 1;
  padding: 24px;
  max-width: 1200px;
  overflow-y: auto;
}
</style>
