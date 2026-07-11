<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox, ElInput } from 'element-plus'
import { Search, Star, StarFilled, Download, Delete, PictureFilled } from '@element-plus/icons-vue'
import { getAssets, toggleFavorite, batchDownload, deleteAssets, transferAsset } from '../api/assets'
import { getCollections, createCollection, type Collection } from '../api/collections'
import type { AssetItem } from '../types'
import AssetCard from '../components/AssetCard.vue'

const items = ref<AssetItem[]>([])
const total = ref(0)
const page = ref(1)
const perPage = 24
const loading = ref(false)
const search = ref('')
const assetType = ref('all')
const showFavorites = ref(false)
const selectionMode = ref(false)
const selectedIds = ref<Set<number>>(new Set())
const deleteFiles = ref(false)
const drawerVisible = ref(false)
const detailAsset = ref<AssetItem | null>(null)
const uploadingUrl = ref('')
const collections = ref<Collection[]>([])
const selectedCollectionId = ref<number | undefined>(undefined)
const showCreateCollectionDialog = ref(false)
const newCollectionName = ref('')

const collectionAssetIds = computed(() => {
  if (!selectedCollectionId.value) return null
  const c = collections.value.find(c => c.id === selectedCollectionId.value)
  return new Set(c?.assets?.map(a => a.id) || [])
})

const displayAssets = computed(() => {
  if (!collectionAssetIds.value) return items.value
  return items.value.filter(item => collectionAssetIds.value!.has(item.id))
})

// assetSrc 按 localPath > originalURL > githubURL 返回可展示的 URL，并修正本地路径为 HTTP 路径
function assetSrc(item: AssetItem): string {
  if (item.local_path) {
    return item.local_path.startsWith('outputs/') ? '/' + item.local_path : item.local_path
  }
  return item.original_url || item.github_url || ''
}

async function loadAssets() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page: page.value,
      per_page: perPage,
      type: assetType.value === 'all' ? undefined : assetType.value,
      sort: 'newest',
    }
    if (search.value) params.search = search.value
    if (showFavorites.value) params.favorite = 'true'

    const res = await getAssets(params)
    items.value = res.items || []
    total.value = res.total || 0
    selectedIds.value = new Set()
  } catch (e: any) {
    ElMessage.error(e.message || '加载作品失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadAssets()
}

function handleTypeChange() {
  page.value = 1
  loadAssets()
}

function toggleShowFavorites() {
  showFavorites.value = !showFavorites.value
  page.value = 1
  loadAssets()
}

function toggleSelectionMode() {
  selectionMode.value = !selectionMode.value
  if (!selectionMode.value) selectedIds.value = new Set()
}

function handleToggleFavorite(item: AssetItem) {
  toggleFavorite({ asset_id: item.id, favorite: !item.favorite })
  item.favorite = !item.favorite
}

function handleToggleSelect(item: AssetItem) {
  const next = new Set(selectedIds.value)
  if (next.has(item.id)) next.delete(item.id)
  else next.add(item.id)
  selectedIds.value = next
}

function handleCardClick(item: AssetItem) {
  detailAsset.value = item
  drawerVisible.value = true
}

async function handleTransfer() {
  if (!detailAsset.value) return
  uploadingUrl.value = detailAsset.value.original_url
  try {
    const updated = await transferAsset(detailAsset.value.id)
    detailAsset.value = updated
    const idx = items.value.findIndex(i => i.id === updated.id)
    if (idx >= 0) items.value[idx] = updated
    ElMessage.success('转存完成')
  } catch (e: any) {
    ElMessage.error(e.message || '转存失败')
  } finally {
    uploadingUrl.value = ''
  }
}

function handleCopyLink() {
  if (!detailAsset.value) return
  const url = assetSrc(detailAsset.value)
  if (!url) {
    ElMessage.warning('无可用的链接')
    return
  }
  const fullUrl = url.startsWith('/') ? window.location.origin + url : url
  navigator.clipboard.writeText(fullUrl)
  ElMessage.success('链接已复制')
}

function handleDownload() {
  if (!detailAsset.value) return
  const url = assetSrc(detailAsset.value)
  if (!url) {
    ElMessage.warning('无可用的下载地址')
    return
  }
  // 本地地址直接下载，跨域地址走后端代理
  if (url.startsWith('/outputs/') || url.startsWith('outputs/')) {
    const a = document.createElement('a')
    a.href = url
    a.download = url.split('/').pop() || `asset_${detailAsset.value.id}`
    a.click()
  } else {
    window.open('/api/v1/download?url=' + encodeURIComponent(url), '_blank')
  }
}

async function handleBatchDownload() {
  if (selectedIds.value.size === 0) return
  try {
    const blob = await batchDownload(Array.from(selectedIds.value))
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = 'assets.zip'
    a.click()
    URL.revokeObjectURL(blobUrl)
  } catch (e: any) {
    ElMessage.error(e.message || '下载失败')
  }
}

async function loadCollections() {
  collections.value = await getCollections()
}

async function handleCreateCollection() {
  if (!newCollectionName.value.trim()) return
  await createCollection(newCollectionName.value.trim())
  newCollectionName.value = ''
  showCreateCollectionDialog.value = false
  await loadCollections()
}

async function handleBatchDelete() {
  if (selectedIds.value.size === 0) return
  try {
    const msg = deleteFiles.value
      ? `确定删除选中的 ${selectedIds.value.size} 项？关联的文件也将被删除。`
      : `确定删除选中的 ${selectedIds.value.size} 项？`
    await ElMessageBox.confirm(msg, '确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteAssets({ ids: Array.from(selectedIds.value), delete_files: deleteFiles.value })
    items.value = items.value.filter(r => !selectedIds.value.has(r.id))
    total.value -= selectedIds.value.size
    selectedIds.value = new Set()
    ElMessage.success('已删除')
  } catch { /* cancelled */ }
}

onMounted(() => {
  loadAssets()
  loadCollections()
})
</script>

<template>
  <div class="assets-page">
    <div class="assets-toolbar">
      <div class="toolbar-left">
        <el-input
          v-model="search"
          placeholder="搜索作品"
          :prefix-icon="Search"
          clearable
          style="width: 240px"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <el-select v-model="assetType" style="width: 120px" @change="handleTypeChange">
          <el-option label="全部" value="all" />
          <el-option label="图片" value="image" />
          <el-option label="视频" value="video" />
        </el-select>
      </div>
      <div class="toolbar-right">
        <el-button
          :icon="showFavorites ? StarFilled : Star"
          @click="toggleShowFavorites"
        >
          {{ showFavorites ? '仅收藏' : '全部' }}
        </el-button>
        <el-button
          :type="selectionMode ? 'primary' : 'default'"
          @click="toggleSelectionMode"
        >
          {{ selectionMode ? '取消选择' : '多选' }}
        </el-button>
      </div>
    </div>

    <div style="margin-bottom: 16px; display: flex; align-items: center; gap: 8px; flex-wrap: wrap;">
      <el-tag
        :type="selectedCollectionId === undefined ? 'primary' : 'info'"
        style="cursor: pointer"
        @click="selectedCollectionId = undefined"
      >
        全部
      </el-tag>
      <el-tag
        v-for="c in collections"
        :key="c.id"
        :type="selectedCollectionId === c.id ? 'primary' : 'info'"
        style="cursor: pointer"
        @click="selectedCollectionId = c.id"
      >
        {{ c.name }}
      </el-tag>
      <el-button size="small" type="default" @click="showCreateCollectionDialog = true">
        + 新建集合
      </el-button>
    </div>

    <div v-if="selectionMode && selectedIds.size > 0" class="batch-bar">
      <span class="batch-count">已选 {{ selectedIds.size }} 项</span>
      <el-button type="primary" :icon="Download" @click="handleBatchDownload">
        下载 ({{ selectedIds.size }})
      </el-button>
      <el-checkbox v-model="deleteFiles" style="margin: 0 8px">删除文件</el-checkbox>
      <el-button type="danger" :icon="Delete" @click="handleBatchDelete">
        删除 ({{ selectedIds.size }})
      </el-button>
    </div>

    <div v-loading="loading" class="grid-wrap">
      <div v-if="displayAssets.length > 0" class="asset-grid">
        <AssetCard
          v-for="item in displayAssets"
          :key="item.id"
          :asset="item"
          :selected="selectedIds.has(item.id)"
          @toggle-favorite="handleToggleFavorite(item)"
          @toggle-select="handleToggleSelect(item)"
          @click="selectionMode ? handleToggleSelect(item) : handleCardClick(item)"
        />
      </div>
      <el-empty v-else-if="!loading" description="暂无作品">
        <el-icon :size="48" style="color: #c0c4cc; margin-bottom: 12px">
          <PictureFilled />
        </el-icon>
      </el-empty>
    </div>

    <div v-if="total > perPage" class="pagination-wrap">
      <el-pagination
        v-model:current-page="page"
        :page-size="perPage"
        :total="total"
        layout="prev, pager, next"
        @current-change="loadAssets"
      />
    </div>

    <el-drawer
      v-model="drawerVisible"
      :size="500"
      :title="detailAsset?.prompt?.slice(0, 50) || ''"
      destroy-on-close
    >
      <template v-if="detailAsset">
        <div class="detail-preview">
	          <el-image
	            v-if="detailAsset.type === 'image'"
	            :src="detailAsset.thumbnail || assetSrc(detailAsset)"
	            fit="contain"
	            style="width: 100%; max-height: 400px"
	          />
	          <video
	            v-else
	            :src="assetSrc(detailAsset)"
	            controls
	            style="width: 100%; max-height: 400px"
	          />
	        </div>
        <div class="detail-meta">
          <p class="detail-prompt">{{ detailAsset.prompt }}</p>
          <p class="detail-info">
            <span>模式: {{ detailAsset.mode }}</span>
            <span>{{ detailAsset.time }}</span>
          </p>
        </div>
        <div class="detail-actions">
          <el-button
            :type="detailAsset.favorite ? 'warning' : 'default'"
            :icon="detailAsset.favorite ? StarFilled : Star"
            @click="handleToggleFavorite(detailAsset)"
          >
            {{ detailAsset.favorite ? '已收藏' : '收藏' }}
          </el-button>
          <el-button
            :loading="uploadingUrl === (detailAsset.original_url || '')"
            @click="handleTransfer"
          >
            转存
          </el-button>
          <el-button @click="handleCopyLink">复制链接</el-button>
          <el-button @click="handleDownload">下载</el-button>
          <el-button @click="drawerVisible = false">关闭</el-button>
        </div>
      </template>
    </el-drawer>
    <el-dialog v-model="showCreateCollectionDialog" title="新建集合" width="400px">
      <el-input v-model="newCollectionName" placeholder="输入集合名称" />
      <template #footer>
        <el-button @click="showCreateCollectionDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreateCollection">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.assets-page {
  max-width: 1000px;
  margin: 0 auto;
  padding: 8px 0 40px;
}

.assets-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  gap: 12px;
  flex-wrap: wrap;
}
.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.batch-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  margin-bottom: 16px;
  background: #f0f5ff;
  border: 1px solid #d6e4ff;
  border-radius: 8px;
}
.batch-count {
  font-size: 14px;
  font-weight: 500;
  color: #409eff;
}

.grid-wrap {
  min-height: 200px;
}
.asset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

.detail-preview {
  margin-bottom: 16px;
  border-radius: 8px;
  overflow: hidden;
  background: #f5f7fa;
}
.detail-meta {
  margin-bottom: 16px;
}
.detail-prompt {
  font-size: 14px;
  color: #303133;
  line-height: 1.6;
  margin: 0 0 8px;
}
.detail-info {
  font-size: 13px;
  color: #909399;
  display: flex;
  gap: 16px;
  margin: 0;
}
.detail-actions {
  display: flex;
  gap: 10px;
}
</style>
