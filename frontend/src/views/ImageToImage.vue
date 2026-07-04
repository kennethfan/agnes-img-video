<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, Link } from '@element-plus/icons-vue'
import { imageToImage } from '../api/image'
import ImageResult from '../components/ImageResult.vue'

const inputMode = ref<'upload' | 'url'>('upload')
const prompt = ref('')
const negativePrompt = ref('')
const size = ref('1024x1024')
const strength = ref(0.75)
const loading = ref(false)
const images = ref<string[]>([])
const file = ref<File | null>(null)
const imageUrl = ref('')

const sizeOptions = [
  { value: '1024x1024', label: '1024x1024 (1:1)' },
  { value: '1024x1792', label: '1024x1792 (9:16)' },
  { value: '1792x1024', label: '1792x1024 (16:9)' },
]

function handleFileChange(uploadFile: any) {
  file.value = uploadFile.raw || null
}

async function handleGenerate() {
  const source = inputMode.value === 'upload' ? file.value : imageUrl.value.trim()
  if (!source) {
    ElMessage.warning(inputMode.value === 'upload' ? '请上传图片' : '请输入图片 URL')
    return
  }
  if (!prompt.value.trim()) {
    ElMessage.warning('请输入风格描述')
    return
  }
  loading.value = true
  images.value = []
  try {
    const res = await imageToImage(
      source,
      prompt.value,
      size.value,
      strength.value,
      negativePrompt.value
    )
    images.value = res.images
  } catch (e: any) {
    ElMessage.error(e.message || '生成失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <el-form label-width="100px">
      <el-form-item label="输入方式">
        <el-radio-group v-model="inputMode">
          <el-radio-button value="upload">
            <el-icon style="vertical-align: middle"><upload-filled /></el-icon>
            <span style="vertical-align: middle">上传图片</span>
          </el-radio-button>
          <el-radio-button value="url">
            <el-icon style="vertical-align: middle"><Link /></el-icon>
            <span style="vertical-align: middle">图片 URL</span>
          </el-radio-button>
        </el-radio-group>
      </el-form-item>

      <el-form-item v-if="inputMode === 'upload'" label="上传图片">
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

      <el-form-item v-if="inputMode === 'url'" label="图片 URL">
        <el-input
          v-model="imageUrl"
          placeholder="请输入图片公网 URL，如 https://example.com/image.png"
          clearable
        />
      </el-form-item>
      <el-form-item label="风格描述">
        <el-input
          v-model="prompt"
          type="textarea"
          :rows="3"
          placeholder="描述你想要的风格变化..."
        />
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
          <el-option v-for="opt in sizeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="重绘强度">
        <el-slider v-model="strength" :min="0" :max="1" :step="0.05" style="width: 300px" />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" size="large" @click="handleGenerate">
          生成图片
        </el-button>
      </el-form-item>
    </el-form>

    <ImageResult :images="images" :loading="loading" />
  </div>
</template>
