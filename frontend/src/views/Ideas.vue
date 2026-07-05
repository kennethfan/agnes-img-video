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

interface Template {
  id: string
  name: string
  title: string
  content: string
  tags: string[]
}

const templates: Template[] = [
  {
    id: 'video-story',
    name: '📹 视频故事',
    title: '',
    content: `## 视频主题
[在这里写视频主题]

## 目标受众
[目标观众是谁]

## 核心内容
- 开场：
- 发展：
- 高潮：
- 结尾：

## 视觉风格
[描述画面风格]

## 时长规划
预计时长：__秒`,
    tags: ['视频', '故事'],
  },
  {
    id: 'image-concept',
    name: '🎨 图片创意',
    title: '',
    content: `## 创意概念
[描述图片的核心概念]

## 画面描述
- 主体：
- 背景：
- 色调：
- 光线：

## 风格参考
[描述期望的风格]

## 文字/标语
[如果需要添加文字]`,
    tags: ['图片', '创意'],
  },
  {
    id: 'script-outline',
    name: '📝 脚本大纲',
    title: '',
    content: `## 故事背景
[设定故事发生的世界]

## 主要角色
- 角色1：
- 角色2：

## 剧情梗概
[一句话概括故事]

## 分幕结构
### 第一幕：开端
### 第二幕：发展
### 第三幕：高潮
### 第四幕：结局

## 对白要点
[关键对白或旁白]`,
    tags: ['脚本', '剧情'],
  },
  {
    id: 'social-post',
    name: '📱 社交内容',
    title: '',
    content: `## 平台
[小红书/抖音/微博/...]

## 内容主题
[要发布什么内容]

## 文案草稿
[第一版文案]

## 标签
#标签1 #标签2 #标签3

## 发布时间
[计划发布时间]

## 配图/视频要求
[需要什么样的素材]`,
    tags: ['社交', '文案'],
  },
  {
    id: 'product-showcase',
    name: '🛍️ 产品展示',
    title: '',
    content: `## 产品名称
[产品名]

## 核心卖点
- 卖点1：
- 卖点2：
- 卖点3：

## 目标场景
[使用场景描述]

## 素材需求
- 主图：
- 细节图：
- 场景图：

## 文案风格
[专业/活泼/高端/...]`,
    tags: ['产品', '展示'],
  },
]

const ideas = ref<Idea[]>([])
const newTitle = ref('')
const newContent = ref('')
const newTags = ref('')
const showDialog = ref(false)
const selectedTemplate = ref<Template | null>(null)

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
  selectedTemplate.value = null
  showDialog.value = false
}

function selectTemplate(template: Template) {
  selectedTemplate.value = template
  newTitle.value = template.title
  newContent.value = template.content
  newTags.value = template.tags.join(', ')
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
      width="600px"
      @close="resetForm"
    >
      <!-- 模板选择 -->
      <div class="template-section">
        <div class="template-label">选择模板（可选）：</div>
        <div class="template-list">
          <el-button
            v-for="tpl in templates"
            :key="tpl.id"
            :type="selectedTemplate?.id === tpl.id ? 'primary' : 'default'"
            size="small"
            @click="selectTemplate(tpl)"
          >
            {{ tpl.name }}
          </el-button>
          <el-button
            v-if="selectedTemplate"
            type="info"
            text
            size="small"
            @click="selectTemplate({ id: '', name: '', title: '', content: '', tags: [] })"
          >
            清除模板
          </el-button>
        </div>
      </div>

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
            :rows="8"
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

.template-section {
  margin-bottom: 20px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 8px;
}

.template-label {
  font-size: 14px;
  color: #606266;
  margin-bottom: 8px;
}

.template-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
