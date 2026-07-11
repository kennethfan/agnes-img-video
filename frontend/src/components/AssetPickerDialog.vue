<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getAssets } from '../api/assets'
import type { AssetItem } from '../types'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
  selected: [url: string]
}>()

const loading = ref(false)
const assets = ref<AssetItem[]>([])
const selectedUrl = ref('')

async function loadAssets() {
  loading.value = true
  try {
    const res = await getAssets({ type: 'image' })
    assets.value = res.items || []
  } catch (e: any) {
    ElMessage.error('加载作品库失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

watch(() => props.visible, (v) => {
  if (v) {
    selectedUrl.value = ''
    loadAssets()
  }
})

function toggleSelect(url: string) {
  selectedUrl.value = selectedUrl.value === url ? '' : url
}

function confirm() {
  if (!selectedUrl.value) {
    ElMessage.warning('请选择一张图片')
    return
  }
  emit('selected', selectedUrl.value)
  emit('update:visible', false)
}

function cancel() {
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="emit('update:visible', $event)"
    title="从作品库选择图片"
    width="700px"
    top="5vh"
    :close-on-click-modal="false"
  >
    <div v-loading="loading" style="min-height: 200px">
      <div v-if="assets.length === 0 && !loading" style="text-align: center; padding: 60px 0; color: #c0c4cc">
        <p>作品库暂无图片</p>
      </div>
      <el-row :gutter="12" v-else>
        <el-col
          v-for="item in assets"
          :key="item.id"
          :xs="12" :sm="8" :md="6"
          style="margin-bottom: 12px"
        >
          <div
            class="asset-thumb"
            :class="{ selected: selectedUrl === item.original_url }"
            @click="toggleSelect(item.original_url)"
          >
            <el-image
              :src="item.thumbnail || item.original_url"
              fit="cover"
              style="width: 100%; height: 120px"
            />
            <div class="asset-label">{{ item.prompt?.slice(0, 30) }}</div>
          </div>
        </el-col>
      </el-row>
    </div>
    <template #footer>
      <el-button @click="cancel">取消</el-button>
      <el-button type="primary" @click="confirm">确认</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.asset-thumb {
  border: 2px solid #dcdfe6;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  transition: border-color 0.2s;
}
.asset-thumb:hover {
  border-color: #409eff;
}
.asset-thumb.selected {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.3);
}
.asset-label {
  padding: 4px 6px;
  font-size: 12px;
  color: #606266;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
