<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { expandIdea } from '../../api/ideas'
import { ElButton, ElTag, ElMessage } from 'element-plus'

const store = useWizardStore()
const genres = ['玄幻', '科幻', '言情', '悬疑', '现实主义', '历史', '武侠', '恐怖', '喜剧', '冒险']
const selected = ref(store.novel.genre)
const suggestions = ref<string[]>([])

async function getSuggestions() {
  try {
    const result = await expandIdea(
      `基于主题"${store.novel.theme}"的创意写作风格建议`,
      '请推荐3个适合这个主题的写作风格方向，每个方向用一行描述',
      'zh',
    )
    suggestions.value = result.split('\n').filter(Boolean)
  } catch { /* ignore */ }
}

function selectGenre(genre: string) {
  selected.value = genre
  store.novel.genre = genre
}

function proceed() {
  if (!selected.value) { ElMessage.warning('请选择一个风格'); return }
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>选择风格/流派</h4>
    <div style="display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 16px">
      <el-tag
        v-for="g in genres"
        :key="g"
        :type="selected === g ? 'primary' : 'info'"
        style="cursor: pointer; padding: 6px 16px; font-size: 14px"
        @click="selectGenre(g)"
      >
        {{ g }}
      </el-tag>
    </div>
    <el-button text @click="getSuggestions">💡 灵感建议</el-button>
    <div v-if="suggestions.length" style="margin-top: 8px; color: #666; font-size: 13px">
      <p v-for="(s, i) in suggestions" :key="i">{{ s }}</p>
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="proceed">下一步：角色设置</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
