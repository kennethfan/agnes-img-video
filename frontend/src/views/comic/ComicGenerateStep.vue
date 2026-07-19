<script setup lang="ts">
import { ref, inject, computed } from 'vue'
import { ElButton, ElMessage, ElProgress } from 'element-plus'
import { Picture } from '@element-plus/icons-vue'
import { textToImage } from '../../api/image'
import { generateComicPrompts } from '../../api/comic'
import ImageResult from '../../components/ImageResult.vue'
import type { ComicPanel, ComicData } from '../../types'

const comicData = inject<import('vue').Ref<ComicData | null>>('comicData')
const comicPanels = inject<import('vue').Ref<ComicPanel[]>>('comicPanels')
const projectId = inject<import('vue').Ref<number | undefined>>('projectId')

const generating = ref(false)
const currentProgress = ref(0)
const generatedUrls = ref<string[]>([])

const panels = computed(() => comicPanels?.value || [])

async function batchGenerate() {
  if (!panels.value.length) {
    ElMessage.warning('没有需要生成的图片')
    return
  }
  const emptyPrompts = panels.value.filter(p => !p.prompt.trim())
  if (emptyPrompts.length > 0) {
    ElMessage.warning('存在空的提示词，请先在布局步骤中填写')
    return
  }

  generating.value = true
  currentProgress.value = 0
  generatedUrls.value = []
  const total = panels.value.length

  // 并行提交所有格子的生成任务，等待全部完成
  await Promise.allSettled(
    panels.value.map(async (panel, i) => {
      try {
        const resp = await textToImage({
          prompt: panel.prompt,
          size: '1024x1024',
          n: 1,
          project_id: projectId?.value,
        })
        if (resp.images?.length) {
          panels.value[i].image = resp.images[0]
          generatedUrls.value.push(resp.images[0])
        }
      } catch (e: any) {
        ElMessage.error(`第 ${i + 1} 格生成失败: ${(e.message || '')}`)
      }
      currentProgress.value++
    })
  )

  generating.value = false
  ElMessage.success(`全部 ${total} 张图片生成完成`)
}

/** 重新生成单个格子 */
async function regeneratePanel(index: number) {
  const panel = panels.value[index]
  if (!panel?.prompt.trim()) return

  try {
    const resp = await textToImage({
      prompt: panel.prompt,
      size: '1024x1024',
      n: 1,
      project_id: projectId?.value,
    })
    if (resp.images?.length) {
      panel.image = resp.images[0]
    }
  } catch (e: any) {
    ElMessage.error(`重新生成失败: ${(e.message || '')}`)
  }
}

/** 重新生成所有格子（使用布局步骤的 AI 生成一次提示词） */
async function regenerateAllPrompts() {
  if (!comicData?.value || !projectId?.value) return
  const layout = comicData.value.layout || 'quad'
  const count = panels.value.length || 4
  try {
    const theme = comicData.value.storyline || '漫画'
    const prompts = await generateComicPrompts(theme, layout, count)
    prompts.forEach((p, i) => {
      if (panels.value[i]) panels.value[i].prompt = p
    })
    ElMessage.success('提示词已重新生成，继续生成图片')
  } catch (e: any) {
    ElMessage.warning('AI 提示词生成失败')
  }
}
</script>

<template>
  <div class="comic-generate">
    <div class="step-intro">
      <h3>批量生成漫画图片</h3>
      <p>点击「批量生成」为所有格子创建图片。支持逐格重新生成。</p>
    </div>

    <!-- 批量操作栏 -->
    <div class="batch-bar">
      <el-button type="primary" :loading="generating" :icon="Picture" @click="batchGenerate">
        {{ generating ? `生成中 ${currentProgress}/${panels.length}` : '批量生成' }}
      </el-button>
      <el-button v-if="generating" disabled>
        <el-progress
          :percentage="Math.round((currentProgress / (panels.length || 1)) * 100)"
          :stroke-width="16"
          style="width: 200px"
          :show-text="false"
        />
      </el-button>
      <el-button text @click="regenerateAllPrompts" v-if="!generating && panels.length > 0">
        重新生成提示词
      </el-button>
    </div>

    <!-- 格子列表 -->
    <div v-if="panels.length > 0" class="panels-grid">
      <div v-for="(panel, i) in panels" :key="i" class="panel-card">
        <div class="panel-card__header">
          第 {{ i + 1 }} 格
          <el-button text size="small" :loading="generating" @click="regeneratePanel(i)" :disabled="generating">
            重新生成
          </el-button>
        </div>

        <!-- 图片预览 -->
        <div v-if="panel.image" class="panel-card__image">
          <ImageResult :images="[panel.image]" :loading="false" :prompt="panel.prompt" mode="text2image" />
        </div>
        <div v-else class="panel-card__placeholder">
          <el-icon :size="32" color="#c0c4cc"><Picture /></el-icon>
          <span>待生成</span>
        </div>

        <div class="panel-card__prompt">
          {{ panel.prompt }}
        </div>
      </div>
    </div>

    <!-- 无数据提示 -->
    <div v-else class="empty-hint">
      <p>请先在「布局」步骤中选择分格并配置提示词。</p>
    </div>

    <!-- 全部生成后的结果预览 -->
    <div v-if="generatedUrls.length > 0 && !generating" class="result-summary">
      <h4>全部结果</h4>
      <ImageResult :images="generatedUrls" :loading="false" prompt="漫画格子" mode="text2image" />
    </div>
  </div>
</template>

<style scoped>
.comic-generate {
  max-width: 800px;
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
.batch-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}
.panels-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}
.panel-card {
  border: 1px solid #eaeaea;
  border-radius: 12px;
  overflow: hidden;
}
.panel-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  font-weight: 600;
  font-size: 13px;
  background: #fafafa;
  border-bottom: 1px solid #eaeaea;
}
.panel-card__image {
  min-height: 160px;
  background: #f5f5f5;
}
.panel-card__placeholder {
  min-height: 160px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #c0c4cc;
  background: #fafafa;
}
.panel-card__prompt {
  padding: 8px 12px;
  font-size: 12px;
  color: #666;
  line-height: 1.5;
}
.empty-hint {
  text-align: center;
  padding: 48px 0;
  color: #909399;
}
.result-summary {
  margin-top: 32px;
  padding-top: 24px;
  border-top: 1px solid #ebeef5;
}
.result-summary h4 {
  margin: 0 0 16px;
}
</style>
