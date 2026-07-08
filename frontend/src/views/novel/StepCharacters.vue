<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElInput, ElMessage } from 'element-plus'

const store = useWizardStore()
const characters = ref(store.novel.characters.length ? store.novel.characters : [{ name: '', personality: '', appearance: '' }])

function addCharacter() {
  characters.value.push({ name: '', personality: '', appearance: '' })
}

function removeCharacter(i: number) {
  characters.value.splice(i, 1)
}

function proceed() {
  const empty = characters.value.some(c => !c.name)
  if (empty) { ElMessage.warning('请填写所有角色名'); return }
  store.novel.characters = JSON.parse(JSON.stringify(characters.value))
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>角色设置</h4>
    <div v-for="(char, i) in characters" :key="i" class="char-card">
      <div class="char-card__header">角色 {{ i + 1 }} <el-button text type="danger" size="small" @click="removeCharacter(i)">删除</el-button></div>
      <el-input v-model="char.name" placeholder="角色名" style="margin-bottom: 8px" />
      <el-input v-model="char.personality" placeholder="性格描述" style="margin-bottom: 8px" />
      <el-input v-model="char.appearance" placeholder="外观描述" />
    </div>
    <el-button text style="margin-top: 8px" @click="addCharacter">+ 添加角色</el-button>
    <div style="margin-top: 16px">
      <el-button type="primary" @click="proceed">下一步：生成大纲</el-button>
    </div>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.char-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; margin-bottom: 12px; }
.char-card__header { display: flex; justify-content: space-between; font-weight: 600; margin-bottom: 8px; }
</style>
