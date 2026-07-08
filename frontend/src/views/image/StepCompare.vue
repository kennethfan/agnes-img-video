<script setup lang="ts">
import { useWizardStore } from '../../stores/wizard'
import { ElButton } from 'element-plus'

const store = useWizardStore()
</script>

<template>
  <div class="step">
    <h4>对比预览</h4>
    <div class="compare-layout">
      <div class="compare-layout__item">
        <h5>原图</h5>
        <img :src="store.image.sourceImage" style="width: 100%; border-radius: 8px" />
      </div>
      <div class="compare-layout__item">
        <h5>精修结果</h5>
        <img :src="store.image.resultImage" style="width: 100%; border-radius: 8px" />
      </div>
    </div>
    <div style="display: flex; gap: 12px; margin-top: 16px; justify-content: center">
      <el-button @click="store.goToStep(1)">继续精修</el-button>
      <el-button type="primary" @click="store.nextStep()">满意，下一步</el-button>
    </div>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.compare-layout { display: flex; gap: 24px; }
.compare-layout__item { flex: 1; }
.compare-layout__item h5 { margin: 0 0 8px; font-size: 14px; color: #666; }
</style>
