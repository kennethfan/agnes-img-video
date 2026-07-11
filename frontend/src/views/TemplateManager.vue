<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getTemplates, createTemplate, updateTemplate, deleteTemplate,
  exportTemplates, importTemplates,
  type PromptTemplate,
} from '../api/templates'

const templates = ref<PromptTemplate[]>([])
const loading = ref(false)
const categoryFilter = ref('')
const dialogVisible = ref(false)
const dialogTitle = ref('新建模板')
const editingId = ref<number | null>(null)
const form = ref<Partial<PromptTemplate>>({
  name: '',
  type: 'template',
  category: '',
  prompt: '',
  negative_prompt: '',
  size: '1024x1024',
  strength: 0.75,
  model: '',
})
const saving = ref(false)

const categoryOptions = ['人物', '产品', '背景', '封面', '海报', '社媒', '自定义']
const sizeOptions = [
  { value: '1024x1024', label: '1024x1024 (1:1)' },
  { value: '1024x1792', label: '1024x1792 (9:16)' },
  { value: '1792x1024', label: '1792x1024 (16:9)' },
  { value: '768x768', label: '768x768 (1:1)' },
  { value: '768x1344', label: '768x1344 (9:16)' },
  { value: '1344x768', label: '1344x768 (16:9)' },
  { value: '512x512', label: '512x512 (1:1)' },
  { value: '512x896', label: '512x896 (9:16)' },
  { value: '896x512', label: '896x512 (16:9)' },
]

const filteredTemplates = computed(() => {
  if (!categoryFilter.value) return templates.value
  return templates.value.filter(t => t.category === categoryFilter.value)
})

const categories = computed(() => {
  const set = new Set(templates.value.map(t => t.category))
  return Array.from(set).filter(Boolean)
})

onMounted(async () => {
  await fetchTemplates()
})

async function fetchTemplates() {
  loading.value = true
  try {
    templates.value = await getTemplates()
  } catch (e: any) {
    ElMessage.error('加载模板失败: ' + (e.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  dialogTitle.value = '新建模板'
  form.value = {
    name: '',
    type: 'template',
    category: '',
    prompt: '',
    negative_prompt: '',
    size: '1024x1024',
    strength: 0.75,
    model: '',
  }
  dialogVisible.value = true
}

function openEdit(row: PromptTemplate) {
  editingId.value = row.id
  dialogTitle.value = '编辑模板'
  form.value = { ...row }
  dialogVisible.value = true
}

async function handleSave() {
  if (!form.value.name || !form.value.prompt) {
    ElMessage.warning('名称和 Prompt 为必填项')
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await updateTemplate(editingId.value, form.value)
      ElMessage.success('更新成功')
    } else {
      await createTemplate(form.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await fetchTemplates()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(row: PromptTemplate) {
  try {
    await ElMessageBox.confirm(`确定删除模板「${row.name}」吗？`, '确认删除')
    await deleteTemplate(row.id)
    ElMessage.success('删除成功')
    await fetchTemplates()
  } catch {
    // canceled
  }
}

async function handleExport() {
  try {
    const data = await exportTemplates()
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `templates-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success(`导出成功，共 ${data.length} 条`)
  } catch (e: any) {
    ElMessage.error('导出失败: ' + (e.message || '未知错误'))
  }
}

async function handleImport(file: File) {
  try {
    const text = await file.text()
    const data = JSON.parse(text)
    if (!Array.isArray(data)) {
      ElMessage.error('无效的导入文件格式')
      return
    }
    await importTemplates(data)
    ElMessage.success(`导入成功，共 ${data.length} 条`)
    await fetchTemplates()
  } catch (e: any) {
    ElMessage.error('导入失败: ' + (e.message || '未知错误'))
  }
}

function onImportFileChange(uploadFile: any) {
  if (uploadFile.raw) {
    handleImport(uploadFile.raw)
  }
}
</script>

<template>
  <div class="tmpl-page">
    <div class="tmpl-header">
      <h3 class="tmpl-title">Prompt 模板管理</h3>
      <div class="tmpl-actions">
        <el-upload
          accept=".json"
          :auto-upload="false"
          :show-file-list="false"
          :on-change="onImportFileChange"
          style="display: inline-block; margin-right: 8px;"
        >
          <el-button size="small">导入</el-button>
        </el-upload>
        <el-button size="small" @click="handleExport">导出</el-button>
        <el-button type="primary" size="small" @click="openCreate">新建</el-button>
      </div>
    </div>

    <div style="margin-bottom: 16px;">
      <el-radio-group v-model="categoryFilter" size="small">
        <el-radio-button value="">全部分类</el-radio-button>
        <el-radio-button v-for="c in categories" :key="c" :value="c">{{ c }}</el-radio-button>
      </el-radio-group>
    </div>

    <el-table :data="filteredTemplates" v-loading="loading" stripe style="width: 100%">
      <el-table-column prop="name" label="名称" width="140" />
      <el-table-column prop="type" label="类型" width="90">
        <template #default="{ row }">
          <el-tag :type="row.type === 'template' ? 'primary' : 'success'" size="small">
            {{ row.type === 'template' ? '模板' : '预设' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="category" label="分类" width="80" />
      <el-table-column prop="prompt" label="Prompt" min-width="200" show-overflow-tooltip />
      <el-table-column prop="size" label="尺寸" width="90" />
      <el-table-column prop="strength" label="强度" width="65">
        <template #default="{ row }">{{ row.strength?.toFixed(2) }}</template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="160" />
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button size="small" link @click="openEdit(row)">编辑</el-button>
          <el-button size="small" link type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px" :close-on-click-modal="false">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="模板名称" />
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio value="template">模板</el-radio>
            <el-radio value="preset">预设</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="form.category" allow-create filterable placeholder="选择或输入分类" style="width: 100%">
            <el-option v-for="c in categoryOptions" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="Prompt" required>
          <el-input v-model="form.prompt" type="textarea" :rows="4" placeholder="提示词" />
        </el-form-item>
        <el-form-item label="负面提示词">
          <el-input v-model="form.negative_prompt" type="textarea" :rows="2" placeholder="负面提示词" />
        </el-form-item>
        <el-form-item label="尺寸">
          <el-select v-model="form.size" placeholder="选择尺寸" style="width: 100%">
            <el-option v-for="opt in sizeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="重绘强度">
          <el-slider v-model="form.strength" :min="0" :max="1" :step="0.05" style="width: 100%" />
        </el-form-item>
        <el-form-item label="模型">
          <el-input v-model="form.model" placeholder="模型名称（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.tmpl-page {
  max-width: 1100px;
}
.tmpl-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.tmpl-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
}
.tmpl-actions {
  display: flex;
  align-items: center;
}
</style>
