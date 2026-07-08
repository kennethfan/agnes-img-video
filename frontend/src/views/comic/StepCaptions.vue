<script setup lang="ts">
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElInput } from 'element-plus'

const store = useWizardStore()
</script>

<template>
  <div class="step">
    <h4>添加台词</h4>
    <div class="captions-grid" :style="{ gridTemplateColumns: 'repeat(2, 1fr)' }">
      <div v-for="(panel, i) in store.comic.panels" :key="i" class="caption-card">
        <img :src="panel.image" style="width: 100%; border-radius: 8px; margin-bottom: 8px" />
        <el-input v-model="panel.caption" placeholder="输入台词/对白..." />
      </div>
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="store.nextStep()">下一步：导出</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.captions-grid { display: grid; gap: 16px; }
.caption-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; }
</style>
