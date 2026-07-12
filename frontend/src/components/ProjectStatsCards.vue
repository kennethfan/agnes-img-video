<script setup lang="ts">
import type { ProjectStats } from '../types'
import { Picture, Edit, Clock, Warning } from '@element-plus/icons-vue'

defineProps<{ stats: ProjectStats | null }>()
</script>

<template>
  <div class="stats-cards">
    <el-card shadow="hover" class="stat-card">
      <el-icon :size="24" color="var(--el-color-primary)"><Picture /></el-icon>
      <div class="stat-info">
        <span class="stat-value">{{ stats?.file_count ?? '-' }}</span>
        <span class="stat-label">文件总数</span>
      </div>
    </el-card>
    <el-card shadow="hover" class="stat-card">
      <el-icon :size="24" color="var(--el-color-success)"><Edit /></el-icon>
      <div class="stat-info">
        <span class="stat-value">{{ stats?.optimized_count ?? '-' }}</span>
        <span class="stat-label">优化次数</span>
      </div>
    </el-card>
    <el-card shadow="hover" class="stat-card">
      <el-icon :size="24" color="var(--el-color-warning)"><Warning /></el-icon>
      <div class="stat-info">
        <span class="stat-value">{{ stats?.running_tasks ?? '-' }}</span>
        <span class="stat-label">进行中任务</span>
      </div>
    </el-card>
    <el-card shadow="hover" class="stat-card">
      <el-icon :size="24" color="var(--el-color-info)"><Clock /></el-icon>
      <div class="stat-info">
        <span class="stat-value">{{ stats?.last_activity ? stats.last_activity.slice(0, 16).replace('T', ' ') : '-' }}</span>
        <span class="stat-label">最后活动</span>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
}
.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}
.stat-info {
  display: flex;
  flex-direction: column;
}
.stat-value {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 12px;
  color: var(--el-color-info);
}
</style>
