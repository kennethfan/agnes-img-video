<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElInput, ElMessage } from 'element-plus'

const store = useWizardStore()
const theme = ref(store.novel.theme)

function proceed() {
  if (!theme.value) { ElMessage.warning('请输入小说主题'); return }
  store.novel.theme = theme.value
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>选题设定</h4>
    <el-input v-model="theme" type="textarea" :rows="4" placeholder="输入小说主题或一句话灵感，例如「一个程序员穿越到魔法世界」" />
    <el-button type="primary" style="margin-top: 16px" @click="proceed">下一步：选择风格</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
