<script setup lang="ts">
import { ref } from 'vue'
import { ElButton, ElMessage, ElMessageBox, ElUpload } from 'element-plus'
import { exportDB, restoreDB } from '../api/db'

const uploading = ref(false)

async function handleExportDB() {
  await exportDB('db')
}

async function handleExportSQL() {
  await exportDB('sql')
}

async function handleExportJSON() {
  await exportDB('json')
}

async function handleRestore(rawFile: File) {
  const file = rawFile
  if (!file) return false

  if (file.name.endsWith('.json')) {
    try {
      await ElMessageBox.confirm(
        `确定用「${file.name}」恢复数据？\n\n当前所有数据将被覆盖，此操作不可撤销。`,
        '确认恢复',
        { confirmButtonText: '确定恢复', cancelButtonText: '取消', type: 'warning' },
      )
    } catch {
      return false
    }
  } else if (file.name.endsWith('.sql')) {
    try {
      await ElMessageBox.confirm(
        `确定执行「${file.name}」？\n\nSQL 文件中的语句会直接作用于当前数据库，建议先导出备份。`,
        '确认执行 SQL',
        { confirmButtonText: '确定执行', cancelButtonText: '取消', type: 'warning' },
      )
    } catch {
      return false
    }
  } else if (file.name.endsWith('.db') || file.name.endsWith('.sqlite')) {
    try {
      await ElMessageBox.confirm(
        `确定用「${file.name}」替换当前数据库？\n\n当前所有数据将被覆盖，此操作不可撤销。`,
        '确认恢复',
        { confirmButtonText: '确定恢复', cancelButtonText: '取消', type: 'warning' },
      )
    } catch {
      return false
    }
  } else {
    ElMessage.warning('请选择 .db、.sqlite、.sql 或 .json 文件')
    return false
  }

  uploading.value = true
  try {
    const msg = file.name.endsWith('.json') ? 'JSON 恢复成功'
      : file.name.endsWith('.sql') ? 'SQL 执行成功'
      : '数据库恢复成功'
    await restoreDB(file)
    ElMessage.success(msg)
  } catch (e: any) {
    ElMessage.error(e?.message || '恢复失败')
  } finally {
    uploading.value = false
  }
  return false
}
</script>

<template>
  <div class="db-manage">
    <h3>备份恢复</h3>
    <p class="desc">history.db 包含所有历史记录、作品库、故事板数据。支持 .db / .sql / .json 格式。JSON 格式兼容 SQLite/MySQL/PostgreSQL 所有数据库驱动。</p>

    <div class="cards">
      <div class="card">
        <div class="card-icon">⬇</div>
        <div class="card-title">导出</div>
        <div class="card-desc">下载当前数据库，用于备份或迁移。</div>
        <div class="card-actions">
          <el-button type="primary" @click="handleExportDB">导出 .db</el-button>
          <el-button @click="handleExportSQL">导出 .sql</el-button>
          <el-button @click="handleExportJSON">导出 JSON</el-button>
        </div>
      </div>

      <div class="card">
        <div class="card-icon">⬆</div>
        <div class="card-title">恢复</div>
        <div class="card-desc">上传 .db / .sqlite / .sql / .json 文件。<br>操作不可恢复，建议先导出备份。</div>
        <el-upload
          :show-file-list="false"
          :before-upload="handleRestore"
          accept=".db,.sqlite,.sql,.json"
          :disabled="uploading"
        >
          <el-button type="danger" :loading="uploading" plain>上传并恢复</el-button>
        </el-upload>
      </div>
    </div>
  </div>
</template>

<style scoped>
.db-manage { padding: 0; }
.db-manage h3 { margin: 0 0 8px; font-size: 16px; }
.desc { margin: 0 0 24px; font-size: 13px; color: #909399; }
.cards { display: flex; gap: 24px; }
.card {
  flex: 1;
  border: 1px solid #eaeaea;
  border-radius: 12px;
  padding: 32px 24px;
  text-align: center;
  background: #fafafa;
}
.card-icon { font-size: 36px; margin-bottom: 12px; }
.card-title { font-size: 15px; font-weight: 600; margin-bottom: 8px; }
.card-desc { font-size: 12px; color: #909399; margin-bottom: 24px; line-height: 1.6; }
.card-actions { display: flex; gap: 8px; justify-content: center; }
</style>
