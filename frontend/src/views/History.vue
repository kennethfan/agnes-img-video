<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getHistory, clearHistory } from '../api/history'
import type { HistoryRecord } from '../types'

const records = ref<HistoryRecord[]>([])
const loading = ref(false)

async function loadHistory() {
  loading.value = true
  try {
    const res = await getHistory()
    records.value = res.records || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载历史失败')
  } finally {
    loading.value = false
  }
}

async function handleClear() {
  try {
    await clearHistory()
    records.value = []
    ElMessage.success('历史已清空')
  } catch (e: any) {
    ElMessage.error(e.message || '清空失败')
  }
}

onMounted(loadHistory)
</script>

<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px">
      <span style="color: #909399">{{ records.length }} 条记录</span>
      <el-button type="danger" size="small" @click="handleClear">清空历史</el-button>
    </div>

    <el-timeline v-if="records.length > 0">
      <el-timeline-item
        v-for="(rec, idx) in records"
        :key="idx"
        :timestamp="rec.time"
        placement="top"
      >
        <el-card shadow="hover">
          <div style="margin-bottom: 8px">
            <el-tag size="small" type="info">{{ rec.mode }}</el-tag>
            <span style="margin-left: 8px; color: #606266">{{ rec.prompt }}</span>
          </div>
          <div v-if="rec.images && rec.images.length > 0" class="history-images">
            <el-image
              v-for="(img, i) in rec.images.slice(0, 4)"
              :key="i"
              :src="img"
              :preview-src-list="rec.images"
              fit="cover"
              style="width: 100px; height: 100px; border-radius: 4px; cursor: pointer"
            />
          </div>
        </el-card>
      </el-timeline-item>
    </el-timeline>

    <el-empty v-else description="暂无历史记录" />
  </div>
</template>

<style scoped>
.history-images {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
