<script setup lang="ts">
import { ref } from 'vue'
import { ElButton, ElInput, ElMessage } from 'element-plus'
import { generateStoryline } from '../../api/comic'
import type { Project, ComicData } from '../../types'

const props = defineProps<{ project: Project | null }>()
const emit = defineEmits<{
  storylineGenerated: [data: ComicData]
}>()

const theme = ref('')
const style = ref('')
const loading = ref(false)
const storylineText = ref('')
const charactersText = ref('')
const styleText = ref('')

async function onGenerate() {
  if (!theme.value.trim()) {
    ElMessage.warning('请输入漫画主题')
    return
  }

  loading.value = true
  try {
    const res = await generateStoryline(theme.value, style.value || undefined)
    storylineText.value = res.storyline
    charactersText.value = res.characters
    styleText.value = res.style
    // 生成完成后立即写入 sessionStorage，供底部"下一步"按钮兜底
    const data: ComicData = {
      storyline: res.storyline,
      characters: res.characters,
      style: res.style,
      layout: '',
      panels: [],
    }
    sessionStorage.setItem('comic_ideate_data', JSON.stringify(data))
    ElMessage.success('故事线生成成功！')
  } catch (e: any) {
    ElMessage.error('生成失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function proceed() {
  if (!storylineText.value) return
  const data: ComicData = {
    storyline: storylineText.value,
    characters: charactersText.value,
    style: styleText.value,
    layout: '',
    panels: [],
  }
  emit('storylineGenerated', data)
}
</script>

<template>
  <div class="comic-ideate">
    <div class="step-intro">
      <h3>构思漫画故事</h3>
      <p>输入漫画主题，AI 将为你生成故事线、角色和画风建议。</p>
    </div>

    <div class="form">
      <el-input
        v-model="theme"
        type="textarea"
        :rows="4"
        placeholder="描述你想创作的漫画主题，例如「一只猫的太空冒险」"
      />
      <el-input
        v-model="style"
        placeholder="画风偏好（可选）如：日漫画风、水墨风、美式卡通"
      />
      <el-button type="primary" :loading="loading" @click="onGenerate">
        {{ loading ? 'AI 构思中...' : 'AI 生成故事线' }}
      </el-button>
    </div>

    <div v-if="storylineText" class="result">
      <div class="result-section">
        <h4>故事线</h4>
        <p>{{ storylineText }}</p>
      </div>
      <div class="result-section">
        <h4>角色</h4>
        <p>{{ charactersText }}</p>
      </div>
      <div class="result-section">
        <h4>画风</h4>
        <p>{{ styleText }}</p>
      </div>

      <el-button type="primary" @click="proceed">
        下一步：选择布局
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.comic-ideate {
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
.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.result {
  margin-top: 32px;
}
.result-section {
  margin-bottom: 20px;
}
.result-section h4 {
  margin: 0 0 8px;
  font-size: 15px;
  color: #303133;
}
.result-section p {
  margin: 0;
  color: #606266;
  line-height: 1.6;
  white-space: pre-wrap;
}
</style>
