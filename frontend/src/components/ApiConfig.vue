<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useConfigStore } from '../stores/config'

const configStore = useConfigStore()
const visible = ref(false)
const saving = ref(false)

async function handleSave() {
  saving.value = true
  try {
    await configStore.saveConfig()
    ElMessage.success('配置已保存')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-collapse v-model="visible" style="margin-bottom: 8px">
    <el-collapse-item title="API 配置" name="config">
      <el-form label-width="100px" size="small">
        <el-form-item label="API Key">
          <el-input
            v-model="configStore.apiKey"
            type="password"
            placeholder="留空则使用环境变量"
            show-password
          />
        </el-form-item>
        <el-form-item label="Base URL">
          <el-input v-model="configStore.baseUrl" placeholder="https://apihub.agnes-ai.com/v1" />
        </el-form-item>
        <el-form-item label="Model">
          <el-input v-model="configStore.model" placeholder="agnes-image-2.1-flash" />
        </el-form-item>

        <el-divider content-position="left">GitHub 存储（可选）</el-divider>

        <el-form-item label="Token">
          <el-input
            v-model="configStore.githubToken"
            type="password"
            placeholder="GitHub Personal Access Token"
            show-password
          />
        </el-form-item>
        <el-form-item label="仓库">
          <el-input v-model="configStore.githubRepo" placeholder="owner/repo" />
        </el-form-item>
        <el-form-item label="分支">
          <el-input v-model="configStore.githubBranch" placeholder="main" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="handleSave">
            保存配置
          </el-button>
        </el-form-item>
      </el-form>
    </el-collapse-item>
  </el-collapse>
</template>
