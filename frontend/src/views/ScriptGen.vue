<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { generateScript } from '../api/video'
import { useRedoStore } from '../stores/redo'

const topic = ref('')
const style = ref('')
const duration = ref(30)
const language = ref('zh')
const loading = ref(false)
const script = ref('')
const copied = ref(false)
const redoStore = useRedoStore()

const durationOptions = [15, 30, 60, 90, 120, 180]
const styleOptions = [
  { value: '', label: '默认' },
  { value: '剧情短片', label: '剧情短片' },
  { value: '产品展示', label: '产品展示' },
  { value: '科普教育', label: '科普教育' },
  { value: 'Vlog', label: 'Vlog' },
  { value: '宣传片', label: '宣传片' },
  { value: '教程说明', label: '教程说明' },
]

// 监听重做数据（flush: sync 确保同步触发）
watch(() => redoStore.redoData, (newData) => {
  if (newData && newData.mode === 'script_gen') {
    topic.value = newData.topic || ''
    script.value = newData.script || ''
    style.value = newData.style || ''
    duration.value = newData.duration || 30
    language.value = newData.language || 'zh'
  }
}, { flush: 'sync' })

async function handleGenerate() {
  if (!topic.value.trim()) {
    ElMessage.warning('请输入视频主题')
    return
  }
  loading.value = true
  script.value = ''
  try {
    const res = await generateScript({
      topic: topic.value,
      duration: duration.value,
      style: style.value || undefined,
      language: language.value,
    })
    script.value = res.script
  } catch (e: any) {
    ElMessage.error(e.message || '生成失败')
  } finally {
    loading.value = false
  }
}

function copyScript() {
  navigator.clipboard.writeText(script.value)
  copied.value = true
  setTimeout(() => (copied.value = false), 2000)
  ElMessage.success('已复制到剪贴板')
}
</script>

<template>
  <div>
    <el-form label-width="100px">
      <el-form-item label="视频主题">
        <el-input
          v-model="topic"
          type="textarea"
          :rows="3"
          placeholder="输入视频主题，例如：北京胡同文化探索、智能手机摄影技巧..."
        />
      </el-form-item>
      <el-row :gutter="16">
        <el-col :span="8">
          <el-form-item label="风格">
            <el-select v-model="style" style="width: 100%">
              <el-option
                v-for="s in styleOptions"
                :key="s.value"
                :label="s.label"
                :value="s.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="时长">
            <el-select v-model="duration" style="width: 100%">
              <el-option v-for="d in durationOptions" :key="d" :label="`${d}秒`" :value="d" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="语言">
            <el-select v-model="language" style="width: 100%">
              <el-option label="中文" value="zh" />
              <el-option label="English" value="en" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item>
        <el-button type="primary" :loading="loading" size="large" @click="handleGenerate">
          {{ loading ? '生成中...' : '生成脚本' }}
        </el-button>
      </el-form-item>
    </el-form>

    <div v-if="script" class="script-result">
      <div class="script-header">
        <h3>生成的脚本</h3>
        <el-button size="small" @click="copyScript">
          {{ copied ? '已复制' : '复制脚本' }}
        </el-button>
      </div>
      <el-input
        type="textarea"
        :model-value="script"
        :rows="20"
        readonly
        style="margin-top: 8px"
      />
    </div>
  </div>
</template>

<style scoped>
.script-result {
  margin-top: 16px;
  border-top: 1px solid #e4e7ed;
  padding-top: 16px;
}
.script-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.script-header h3 {
  margin: 0;
  font-size: 16px;
  color: #303133;
}
</style>
