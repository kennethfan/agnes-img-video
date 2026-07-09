<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSettings, updateSettings } from '../api/settings'
import type { Settings } from '../types'

const form = ref<Settings>({
  storage_target: 'local',
  local_image_dir: 'images',
  local_video_dir: 'videos',
  github_image_path: 'outputs/images',
  github_video_path: 'outputs/videos',
})
const loading = ref(false)
const saving = ref(false)

// 将 storage_target 字符串转为独立 Checkbox 状态
const targetLocal = ref(true)
const targetGithub = ref(false)

function syncTargetToCheckboxes(target: string) {
  targetLocal.value = target === 'local' || target === 'both'
  targetGithub.value = target === 'github' || target === 'both'
}

onMounted(async () => {
  loading.value = true
  try {
    const s = await getSettings()
    form.value = s
    syncTargetToCheckboxes(s.storage_target)
  } catch (e: any) {
    ElMessage.error(e.message || '加载设置失败')
  } finally {
    loading.value = false
  }
})

function handleSave() {
  // 从 Checkbox 状态计算 storage_target
  if (targetLocal.value && targetGithub.value) {
    form.value.storage_target = 'both'
  } else if (targetGithub.value) {
    form.value.storage_target = 'github'
  } else {
    form.value.storage_target = 'local'
  }
  saving.value = true
  updateSettings(form.value).then(() => {
    ElMessage.success('保存成功')
  }).catch((e: any) => {
    ElMessage.error(e.message || '保存失败')
  }).finally(() => {
    saving.value = false
  })
}
</script>

<template>
  <div class="settings-page">
    <h3>存储设置</h3>

    <el-form :model="form" label-width="140px" v-loading="loading" style="max-width: 520px">
      <el-form-item label="存储目标">
        <el-checkbox v-model="targetLocal">本地 outputs/</el-checkbox>
        <el-checkbox v-model="targetGithub">GitHub</el-checkbox>
      </el-form-item>

      <el-divider>本地存储</el-divider>

      <el-form-item label="图片存放目录">
        <el-input v-model="form.local_image_dir" placeholder="images" />
      </el-form-item>
      <el-form-item label="视频存放目录">
        <el-input v-model="form.local_video_dir" placeholder="videos" />
      </el-form-item>

      <el-divider>GitHub 存储</el-divider>

      <el-form-item label="图片路径">
        <el-input v-model="form.github_image_path" placeholder="outputs/images" />
      </el-form-item>
      <el-form-item label="视频路径">
        <el-input v-model="form.github_video_path" placeholder="outputs/videos" />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<style scoped>
.settings-page { max-width: 600px; padding: 8px 0; }
.settings-page h3 { margin: 0 0 20px; font-size: 16px; }
</style>
