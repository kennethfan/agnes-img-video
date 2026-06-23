<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'
import { createImageToVideo } from '../api/video'

const prompt = ref('')
const file = ref<File | null>(null)
const duration = ref(5)
const aspectRatio = ref('16:9')
const frameRate = ref(24)
const loading = ref(false)
const taskId = ref('')

const durationOptions = [3, 5, 8, 10, 15, 18]
const ratioOptions = ['16:9', '9:16', '1:1', '4:3', '3:4']
const fpsOptions = [12, 24, 30, 60]

function handleFileChange(uploadFile: any) {
  file.value = uploadFile.raw || null
}

async function handleGenerate() {
  if (!file.value) {
    ElMessage.warning('请上传图片')
    return
  }
  if (!prompt.value.trim()) {
    ElMessage.warning('请输入提示词')
    return
  }
  loading.value = true
  try {
    const res = await createImageToVideo(
      file.value,
      prompt.value,
      duration.value,
      aspectRatio.value,
      frameRate.value
    )
    taskId.value = res.taskId
  } catch (e: any) {
    ElMessage.error(e.message || '提交失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <el-form label-width="100px">
      <el-form-item label="上传图片">
        <el-upload
          drag
          accept="image/*"
          :auto-upload="false"
          :limit="1"
          :on-change="handleFileChange"
        >
          <el-icon class="el-icon--upload" style="font-size: 48px">
            <upload-filled />
          </el-icon>
          <div class="el-upload__text">拖拽图片到此处或 <em>点击上传</em></div>
        </el-upload>
      </el-form-item>
      <el-form-item label="提示词">
        <el-input
          v-model="prompt"
          type="textarea"
          :rows="3"
          placeholder="描述视频内容..."
        />
      </el-form-item>
      <el-form-item label="时长">
        <el-select v-model="duration" style="width: 120px">
          <el-option v-for="d in durationOptions" :key="d" :label="`${d}秒`" :value="d" />
        </el-select>
      </el-form-item>
      <el-form-item label="宽高比">
        <el-select v-model="aspectRatio" style="width: 120px">
          <el-option v-for="r in ratioOptions" :key="r" :label="r" :value="r" />
        </el-select>
      </el-form-item>
      <el-form-item label="帧率">
        <el-select v-model="frameRate" style="width: 120px">
          <el-option v-for="f in fpsOptions" :key="f" :label="`${f}fps`" :value="f" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" size="large" @click="handleGenerate">
          生成视频
        </el-button>
      </el-form-item>
    </el-form>

    <div v-if="taskId" style="margin-top: 16px">
      <el-alert title="任务已提交" type="success" show-icon>
        <p>任务 ID: {{ taskId }}</p>
      </el-alert>
    </div>
  </div>
</template>
