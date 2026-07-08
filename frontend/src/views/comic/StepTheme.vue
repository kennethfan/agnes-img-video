<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElInput, ElMessage } from 'element-plus'

const store = useWizardStore()
const theme = ref(store.comic.theme)

function proceed() {
  if (!theme.value) { ElMessage.warning('请输入漫画主题'); return }
  store.comic.theme = theme.value
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>设定漫画主题</h4>
    <el-input v-model="theme" type="textarea" :rows="4" placeholder="描述你想创作的漫画主题，例如「一只猫的太空冒险」" />
    <el-button type="primary" style="margin-top: 16px" @click="proceed">下一步：选择布局</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
