<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'

interface Idea {
  id: number
  title: string
  content: string
  tags: string[]
  createdAt: string
}

const ideas = ref<Idea[]>([])
const newTitle = ref('')
const newContent = ref('')
const newTags = ref('')
const showDialog = ref(false)

function addIdea() {
  if (!newTitle.value.trim()) {
    ElMessage.warning('请输入标题')
    return
  }

  const idea: Idea = {
    id: Date.now(),
    title: newTitle.value,
    content: newContent.value,
    tags: newTags.value.split(',').map(t => t.trim()).filter(Boolean),
    createdAt: new Date().toLocaleString('zh-CN'),
  }

  ideas.value.unshift(idea)
  saveIdeas()
  resetForm()
  ElMessage.success('点子已添加')
}

function deleteIdea(id: number) {
  ideas.value = ideas.value.filter(i => i.id !== id)
  saveIdeas()
  ElMessage.success('点子已删除')
}

function resetForm() {
  newTitle.value = ''
  newContent.value = ''
  newTags.value = ''
  showDialog.value = false
}

function saveIdeas() {
  localStorage.setItem('agnes-ideas', JSON.stringify(ideas.value))
}

function loadIdeas() {
  const saved = localStorage.getItem('agnes-ideas')
  if (saved) {
    try {
      ideas.value = JSON.parse(saved)
    } catch {
      ideas.value = []
    }
  }
}

// 初始加载
loadIdeas()
</script>

<template>
  <div>
    <div style="margin-bottom: 16px">
      <el-button type="primary" @click="showDialog = true">
        添加点子
      </el-button>
    </div>

    <!-- 点子列表 -->
    <div v-if="ideas.length === 0" style="text-align: center; padding: 40px; color: #909399">
      <p>还没有点子，点击上方按钮添加第一个点子吧！</p>
    </div>

    <div v-else class="ideas-grid">
      <el-card
        v-for="idea in ideas"
        :key="idea.id"
        class="idea-card"
        shadow="hover"
      >
        <template #header>
          <div class="card-header">
            <span class="idea-title">{{ idea.title }}</span>
            <el-button
              type="danger"
              text
              size="small"
              @click="deleteIdea(idea.id)"
            >
              删除
            </el-button>
          </div>
        </template>
        <div class="idea-content">{{ idea.content }}</div>
        <div class="idea-meta">
          <span class="idea-time">{{ idea.createdAt }}</span>
          <div class="idea-tags">
            <el-tag
              v-for="tag in idea.tags"
              :key="tag"
              size="small"
              type="info"
            >
              {{ tag }}
            </el-tag>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 添加点子对话框 -->
    <el-dialog
      v-model="showDialog"
      title="添加点子"
      width="500px"
      @close="resetForm"
    >
      <el-form label-width="80px">
        <el-form-item label="标题">
          <el-input
            v-model="newTitle"
            placeholder="点子标题"
          />
        </el-form-item>
        <el-form-item label="内容">
          <el-input
            v-model="newContent"
            type="textarea"
            :rows="4"
            placeholder="详细描述你的点子..."
          />
        </el-form-item>
        <el-form-item label="标签">
          <el-input
            v-model="newTags"
            placeholder="用逗号分隔，如：创意,设计,视频"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetForm">取消</el-button>
        <el-button type="primary" @click="addIdea">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.ideas-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.idea-card {
  height: fit-content;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.idea-title {
  font-weight: 600;
  font-size: 16px;
  color: #303133;
}

.idea-content {
  color: #606266;
  line-height: 1.6;
  margin-bottom: 12px;
  white-space: pre-wrap;
}

.idea-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: #909399;
}

.idea-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}
</style>
