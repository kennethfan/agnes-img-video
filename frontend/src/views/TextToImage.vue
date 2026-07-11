<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { submitTextToImage } from '../api/image'
import { getHistory } from '../api/history'
import { getTemplates, type PromptTemplate } from '../api/templates'
import { connectTaskSSE } from '../utils/sse'
import ImageResult from '../components/ImageResult.vue'
import TaskProgress from '../components/TaskProgress.vue'
import { useRedoStore } from '../stores/redo'

const prompt = ref('')
const negativePrompt = ref('')
const size = ref('1024x1024')
const n = ref(1)
const model = ref('')
const loading = ref(false)
const showProgress = ref(false)
const taskId = ref<number | string>('')
const images = ref<string[]>([])
const errorMsg = ref('')
const redoStore = useRedoStore()

const showTemplatePicker = ref(false)
const templates = ref<PromptTemplate[]>([])
const selectedPreset = ref<number>()
const presets = ref<PromptTemplate[]>([])
const showHistory = ref(false)
const historyList = ref<any[]>([])

const sizeOptions = [
  { value: '1024x1024', label: '1024x1024 (1:1)' },
  { value: '1024x1792', label: '1024x1792 (9:16)' },
  { value: '1792x1024', label: '1792x1024 (16:9)' },
]

onMounted(async () => {
  templates.value = await getTemplates()
  presets.value = templates.value.filter(t => t.type === 'preset')
  try {
    const res = await getHistory()
    historyList.value = (res as any).records || []
  } catch { /* silent */ }
})

function applyTemplate(row: PromptTemplate) {
  prompt.value = row.prompt
  if (row.negative_prompt) negativePrompt.value = row.negative_prompt
  if (row.size) size.value = row.size
  if (row.model) model.value = row.model
  showTemplatePicker.value = false
}

function applyPreset(presetId: number) {
  const preset = presets.value.find(p => p.id === presetId)
  if (!preset) return
  if (preset.size) size.value = preset.size
  if (preset.negative_prompt) negativePrompt.value = preset.negative_prompt
  if (preset.model) model.value = preset.model
}

function applyHistory(h: any) {
  prompt.value = h.prompt
  showHistory.value = false
}

// 监听重做数据（flush: sync 确保同步触发）
watch(() => redoStore.redoData, (newData) => {
  if (newData && newData.mode === 'text2image') {
    prompt.value = newData.prompt || ''
    negativePrompt.value = newData.negativePrompt || ''
    size.value = newData.size || '1024x1024'
    n.value = newData.n || 1
  }
}, { flush: 'sync' })

async function handleGenerate() {
  if (!prompt.value.trim()) {
    ElMessage.warning('请输入提示词')
    return
  }
  loading.value = true
  errorMsg.value = ''
  images.value = []
  taskId.value = ''
  showProgress.value = false

  try {
    const res = await submitTextToImage({
      prompt: prompt.value,
      size: size.value,
      n: n.value,
      negative_prompt: negativePrompt.value || undefined,
    })
    taskId.value = res.taskId
    showProgress.value = true

    connectTaskSSE(res.taskId, {
      onComplete: (data) => {
        showProgress.value = false
        try {
          images.value = JSON.parse(data.result)
        } catch {
          images.value = data.result ? [data.result] : []
        }
        loading.value = false
      },
      onError: (data) => {
        showProgress.value = false
        errorMsg.value = data.error
        loading.value = false
      },
    })
  } catch (e: any) {
    errorMsg.value = e.message || '提交失败'
    loading.value = false
  }
}
</script>

<template>
  <div class="gen-page">
    <div class="gen-input">
      <h3 class="gen-title">文生图</h3>
      <el-form label-width="100px">
        <el-form-item label="提示词">
          <div style="display: flex; gap: 8px; flex-direction: column;">
            <el-input
              v-model="prompt"
              type="textarea"
              :rows="3"
              placeholder="描述你想要生成的图片..."
            />
            <div style="display: flex; gap: 8px;">
              <el-button size="small" @click="showTemplatePicker = true">从模板</el-button>
              <el-button size="small" @click="showHistory = !showHistory">
                {{ showHistory ? '收起历史' : '历史 Prompt' }}
              </el-button>
            </div>
            <el-collapse-transition>
              <div v-if="showHistory" style="max-height: 300px; overflow-y: auto; border: 1px solid #e0e0e0; border-radius: 4px; padding: 8px;">
                <div v-for="h in historyList" :key="h.id" style="padding: 6px 8px; cursor: pointer; border-bottom: 1px solid #f0f0f0;" @click="applyHistory(h)">
                  <div style="font-size: 12px; color: #999;">{{ h.time }}</div>
                  <div style="font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ h.prompt }}</div>
                </div>
                <div v-if="historyList.length === 0" style="color: #999; text-align: center; padding: 16px;">暂无历史记录</div>
              </div>
            </el-collapse-transition>
          </div>
        </el-form-item>
        <el-form-item label="负面提示词">
          <el-input
            v-model="negativePrompt"
            type="textarea"
            :rows="2"
            placeholder="不想出现在图片中的内容..."
          />
        </el-form-item>
        <el-form-item label="尺寸">
          <el-select v-model="size" style="width: 250px">
            <el-option
              v-for="opt in sizeOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model="n" :min="1" :max="4" />
        </el-form-item>
        <el-form-item label="风格预设">
          <el-select v-model="selectedPreset" placeholder="风格预设" @change="applyPreset" clearable style="width: 100%">
            <el-option v-for="p in presets" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型">
          <el-input v-model="model" placeholder="模型名称（可选）" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" size="large" @click="handleGenerate" style="width: 100%">
            生成图片
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gen-preview">
      <TaskProgress v-if="showProgress && taskId" :task-id="taskId" @error="errorMsg = $event" />
      <el-alert v-if="errorMsg" type="error" :description="errorMsg" show-icon closable class="error-alert" />
      <ImageResult :images="images" :loading="loading && !showProgress" :prompt="prompt" mode="text2image" />

    <el-dialog v-model="showTemplatePicker" title="选择 Prompt 模板" width="600px" :close-on-click-modal="false">
      <el-table :data="templates" style="width: 100%" @row-click="applyTemplate" highlight-current-row>
        <el-table-column prop="name" label="名称" width="150" />
        <el-table-column prop="category" label="分类" width="100" />
        <el-table-column prop="prompt" label="Prompt" show-overflow-tooltip />
        <el-table-column prop="size" label="尺寸" width="80" />
      </el-table>
    </el-dialog>
    </div>
  </div>
</template>

<style scoped>
.gen-page {
  display: flex;
  gap: 24px;
  min-height: 500px;
}
.gen-input {
  flex: 1;
  max-width: 480px;
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
}
.gen-preview {
  flex: 1;
  padding: 20px;
  background: var(--bg-subtle);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  display: flex;
  flex-direction: column;
}
.gen-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-muted);
  margin: 0 0 16px 0;
}
.error-alert {
  margin-bottom: 12px;
}
</style>
