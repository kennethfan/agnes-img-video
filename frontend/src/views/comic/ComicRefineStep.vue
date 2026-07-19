<script setup lang="ts">
import { ref } from 'vue'
import { ElButton, ElInput, ElMessage, ElDialog } from 'element-plus'
import { textToImage } from '../../api/image'
import type { Project, ComicPanel } from '../../types'

const props = defineProps<{ project: Project | null; panels: ComicPanel[] }>()
const emit = defineEmits<{
  refined: [panels: ComicPanel[]]
}>()

const editingIndex = ref(-1)
const editPanel = ref<ComicPanel>({ prompt: '', image: '', caption: '', refImage: '' })
const regenerating = ref(false)

function openEditor(index: number) {
  editingIndex.value = index
  editPanel.value = { ...props.panels[index] }
}

function closeEditor() {
  editingIndex.value = -1
}

function saveEdit() {
  if (editingIndex.value >= 0 && editingIndex.value < props.panels.length) {
    const copy = [...props.panels]
    copy[editingIndex.value] = { ...editPanel.value }
    emit('refined', copy)
  }
  closeEditor()
}

async function regenerateImage() {
  if (!editPanel.value.prompt.trim()) {
    ElMessage.warning('请输入提示词')
    return
  }
  regenerating.value = true
  try {
    const res = await textToImage({ prompt: editPanel.value.prompt, size: '1024x1024', n: 1 })
    if (res.images?.length) {
      editPanel.value.image = res.images[0]
      ElMessage.success('已重新生成')
    }
  } catch (e: any) {
    ElMessage.error('生成失败: ' + (e.message || ''))
  } finally {
    regenerating.value = false
  }
}

function proceed() {
  emit('refined', [...props.panels])
}
</script>

<template>
  <div class="comic-refine">
    <div class="step-intro">
      <h3>精修漫画格子</h3>
      <p>点击每个格子编辑提示词、重新生成图片、添加台词语句。</p>
    </div>

    <div class="panels-grid">
      <div
        v-for="(panel, i) in panels"
        :key="i"
        class="panel-card"
        @click="openEditor(i)"
      >
        <div class="panel-card__header">第 {{ i + 1 }} 格</div>
        <img v-if="panel.image" :src="panel.image" class="panel-card__img" />
        <div v-else class="panel-card__placeholder">无图片</div>
        <div class="panel-card__prompt">{{ panel.prompt.slice(0, 60) }}{{ panel.prompt.length > 60 ? '...' : '' }}</div>
        <div v-if="panel.caption" class="panel-card__caption">台词: {{ panel.caption }}</div>
      </div>
    </div>

    <div class="actions">
      <el-button type="primary" @click="proceed">
        下一步：定稿
      </el-button>
    </div>

    <!-- 编辑弹窗 -->
    <el-dialog
      :model-value="editingIndex >= 0"
      title="编辑格子"
      width="500px"
      @close="closeEditor"
    >
      <div v-if="editingIndex >= 0" class="edit-form">
        <el-image
          v-if="editPanel.image"
          :src="editPanel.image"
          style="width: 100%; max-height: 300px; object-fit: contain; border-radius: 8px; margin-bottom: 16px;"
          fit="contain"
        />
        <el-input
          v-model="editPanel.prompt"
          type="textarea"
          :rows="3"
          placeholder="画面提示词"
          style="margin-bottom: 12px"
        />
        <el-button :loading="regenerating" @click="regenerateImage" style="margin-bottom: 12px">
          {{ regenerating ? '生成中...' : '重新生成图片' }}
        </el-button>
        <el-input
          v-model="editPanel.caption"
          placeholder="台词文本（可选）"
          style="margin-bottom: 16px"
        />
      </div>
      <template #footer>
        <el-button @click="closeEditor">取消</el-button>
        <el-button type="primary" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.comic-refine {
  max-width: 700px;
  margin: 0 auto;
}
.step-intro {
  margin-bottom: 24px;
}
.step-intro h3 {
  margin: 0 0 8px;
}
.step-intro p {
  margin: 0;
  color: #909399;
}
.panels-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}
.panel-card {
  border: 1px solid #eaeaea;
  border-radius: 12px;
  padding: 12px;
  cursor: pointer;
  transition: border-color 0.2s;
}
.panel-card:hover {
  border-color: #409eff;
}
.panel-card__header {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 4px;
}
.panel-card__img {
  width: 100%;
  border-radius: 8px;
  margin-bottom: 8px;
}
.panel-card__placeholder {
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  border-radius: 8px;
  color: #c0c4cc;
  font-size: 13px;
  margin-bottom: 8px;
}
.panel-card__prompt {
  font-size: 12px;
  color: #606266;
  margin-bottom: 4px;
}
.panel-card__caption {
  font-size: 12px;
  color: #e6a23c;
  font-style: italic;
}
.actions {
  margin-top: 24px;
  text-align: right;
}
.edit-form {
  display: flex;
  flex-direction: column;
}
</style>
