<script setup lang="ts">
import { ref, inject, type Ref } from 'vue'
import { ElButton, ElInput, ElMessage } from 'element-plus'
import { generateComicPrompts } from '../../api/comic'
import type { Project, ComicPanel, ComicData } from '../../types'

const props = defineProps<{ project: Project | null }>()
const injectedComicData = inject<Ref<ComicData | null>>('comicData')!
const emit = defineEmits<{
  panelsReady: [panels: ComicPanel[]]
}>()

const layouts = [
  { id: 'single', label: '单格', cols: 1, rows: 1, desc: '一张图', count: 1 },
  { id: 'dual', label: '双格', cols: 1, rows: 2, desc: '上下两格', count: 2 },
  { id: 'quad', label: '四格', cols: 2, rows: 2, desc: '田字四格', count: 4 },
  { id: 'six', label: '六格', cols: 2, rows: 3, desc: '2x3 六格', count: 6 },
] as const

const selectedLayout = ref<string | null>(null)
const panels = ref<ComicPanel[]>([])
const loading = ref(false)

async function selectLayout(id: string) {
  selectedLayout.value = id
  const layout = layouts.find(l => l.id === id)!
  panels.value = Array.from({ length: layout.count }, () => ({
    prompt: '',
    image: '',
    caption: '',
    refImage: '',
  }))

  loading.value = true
  try {
    // 优先使用构思步骤生成的故事线/角色/画风作为提示词上下文
    const cd = injectedComicData?.value
    const theme = cd?.storyline || props.project?.brief || '漫画'
    const styleHint = cd?.style ? `，画风：${cd.style}` : ''
    const charHint = cd?.characters ? `，角色：${cd.characters}` : ''
    const fullTheme = theme + styleHint + charHint
    const prompts = await generateComicPrompts(fullTheme, id, layout.count)
    prompts.forEach((p, i) => {
      if (panels.value[i]) panels.value[i].prompt = p
    })
    ElMessage.success('提示词已生成')
  } catch (e: any) {
    ElMessage.warning('AI 提示词生成失败，请手动填写: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function proceed() {
  const empty = panels.value.some(p => !p.prompt.trim())
  if (empty) {
    ElMessage.warning('请为所有格子填写画面提示词')
    return
  }
  emit('panelsReady', panels.value)
}
</script>

<template>
  <div class="comic-layout">
    <div class="step-intro">
      <h3>选择分格布局</h3>
      <p>选择漫画的分格方式，AI 将为每格生成画面提示词。</p>
    </div>

    <div class="layout-grid">
      <div
        v-for="layout in layouts"
        :key="layout.id"
        class="layout-card"
        :class="{ active: selectedLayout === layout.id }"
        @click="selectLayout(layout.id)"
      >
        <div class="layout-card__preview" :style="{ gridTemplateColumns: `repeat(${layout.cols}, 1fr)` }">
          <div v-for="n in layout.count" :key="n" class="layout-card__cell">{{ n }}</div>
        </div>
        <div class="layout-card__label">{{ layout.label }}</div>
        <div class="layout-card__desc">{{ layout.desc }}</div>
      </div>
    </div>

    <div v-if="panels.length > 0" class="panels-section">
      <h4>画面提示词</h4>
      <div class="panels-list">
        <div v-for="(panel, i) in panels" :key="i" class="panel-item">
          <div class="panel-item__header">第 {{ i + 1 }} 格</div>
          <el-input
            v-model="panel.prompt"
            type="textarea"
            :rows="2"
            placeholder="输入画面描述"
            :disabled="loading"
          />
        </div>
      </div>

      <el-button type="primary" :loading="loading" style="margin-top: 16px" @click="proceed">
        {{ loading ? '生成中...' : '下一步：生成图片' }}
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.comic-layout {
  max-width: 700px;
  margin: 0 auto;
}
.step-intro {
  margin-bottom: 24px;
}
.step-intro h3 {
  margin: 0 0 8px;
}
.step-intro p {
  margin: 0;
  color: #909399;
}
.layout-grid {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 24px;
}
.layout-card {
  border: 1px solid #eaeaea;
  border-radius: 12px;
  padding: 16px;
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s;
  width: 140px;
  text-align: center;
}
.layout-card:hover {
  border-color: #409eff;
}
.layout-card.active {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2);
}
.layout-card__preview {
  display: grid;
  gap: 4px;
  width: 100px;
  height: 100px;
  margin: 0 auto 8px;
}
.layout-card__cell {
  border: 1px solid #ddd;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: #909399;
  background: #fafafa;
}
.layout-card__label {
  font-weight: 600;
  font-size: 14px;
}
.layout-card__desc {
  font-size: 12px;
  color: #909399;
}
.panels-section {
  margin-top: 24px;
}
.panels-section h4 {
  margin: 0 0 12px;
}
.panels-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.panel-item {
  border: 1px solid #eaeaea;
  border-radius: 8px;
  padding: 12px;
}
.panel-item__header {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 8px;
}
</style>
