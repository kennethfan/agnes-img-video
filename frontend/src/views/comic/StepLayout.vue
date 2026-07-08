<script setup lang="ts">
import { useWizardStore } from '../../stores/wizard'

const store = useWizardStore()
const layouts = [
  { id: 'single', label: '单格', cols: 1, rows: 1, desc: '一张图' },
  { id: 'dual', label: '双格', cols: 1, rows: 2, desc: '上下两格' },
  { id: 'quad', label: '四格', cols: 2, rows: 2, desc: '田字四格' },
  { id: 'six', label: '六格', cols: 2, rows: 3, desc: '2x3 六格' },
] as const

function selectLayout(id: string) {
  store.comic.layout = id as typeof store.comic.layout
  const layout = layouts.find(l => l.id === id)!
  const count = layout.cols * layout.rows
  store.comic.panels = Array.from({ length: count }, () => ({ prompt: '', image: '', caption: '' }))
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>选择分格布局</h4>
    <div class="layout-grid">
      <div
        v-for="layout in layouts"
        :key="layout.id"
        class="layout-card"
        @click="selectLayout(layout.id)"
      >
        <div class="layout-card__preview" :style="{ gridTemplateColumns: `repeat(${layout.cols}, 1fr)`, gridTemplateRows: `repeat(${layout.rows}, 1fr)` }">
          <div v-for="n in layout.cols * layout.rows" :key="n" class="layout-card__cell">{{ n }}</div>
        </div>
        <div class="layout-card__label">{{ layout.label }}</div>
        <div class="layout-card__desc">{{ layout.desc }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.layout-grid { display: flex; gap: 16px; flex-wrap: wrap; }
.layout-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 16px; cursor: pointer; transition: border-color 0.2s; width: 160px; text-align: center; }
.layout-card:hover { border-color: #000; }
.layout-card__preview { display: grid; gap: 4px; width: 120px; height: 120px; margin: 0 auto 8px; }
.layout-card__cell { border: 1px solid #ddd; border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 12px; color: #909399; background: #fafafa; }
.layout-card__label { font-weight: 600; font-size: 14px; }
.layout-card__desc { font-size: 12px; color: #909399; }
</style>
