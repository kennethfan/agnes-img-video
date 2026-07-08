<script setup lang="ts">
import { ref } from 'vue'

const activePage = defineModel<string>('activePage', { required: true })

const groups = [
  {
    id: 'image',
    icon: '🖼',
    label: '图片',
    items: [
      { id: 'text2img', label: '文生图' },
      { id: 'img2img', label: '图生图' },
      { id: 'batch', label: '批量生成' },
    ],
  },
  {
    id: 'video',
    icon: '🎬',
    label: '视频',
    items: [
      { id: 'text2vid', label: '文生视频' },
      { id: 'img2vid', label: '图生视频' },
      { id: 'multi_vid', label: '多图视频' },
    ],
  },
  {
    id: 'tools',
    icon: '📝',
    label: '工具',
    items: [
      { id: 'script_gen', label: '脚本生成' },
      { id: 'ideas', label: '灵感' },
      { id: 'storyboard', label: '分镜' },
    ],
  },
  {
    id: 'workflow',
    icon: '⚡',
    label: '创作',
    items: [
      { id: 'image_refine', label: '图片精修' },
      { id: 'comic', label: '漫画生成' },
      { id: 'novel', label: '小说生成' },
    ],
  },
  {
    id: 'works',
    icon: '🖥',
    label: '作品',
    items: [
      { id: 'assets', label: '作品库' },
      { id: 'history', label: '历史记录' },
    ],
  },
  {
    id: 'system',
    icon: '⚙️',
    label: '系统',
    items: [
      { id: 'access_logs', label: '接口日志' },
    ],
  },
]

const openGroup = ref<string | null>(null)

function toggleGroup(groupId: string) {
  openGroup.value = openGroup.value === groupId ? null : groupId
}

function selectPage(pageId: string) {
  activePage.value = pageId
}
</script>

<template>
  <div class="nav-sidebar">
    <div class="sidebar-groups">
      <div v-for="g in groups" :key="g.id" class="group">
        <button class="group-header" @click="toggleGroup(g.id)">
          <span class="group-header__icon">{{ g.icon }}</span>
          <span class="group-header__label">{{ g.label }}</span>
          <span class="group-header__arrow" :class="{ expanded: openGroup === g.id }">▶</span>
        </button>
        <div v-if="openGroup === g.id" class="group-items">
          <button
            v-for="item in g.items"
            :key="item.id"
            class="group-item"
            :class="{ active: activePage === item.id }"
            @click="selectPage(item.id)"
          >
            {{ item.label }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.nav-sidebar {
  width: 200px;
  min-width: 200px;
  background: #ffffff;
  border-right: 1px solid #f0f0f0;
  overflow-y: auto;
  padding: 8px 0;
  height: 100%;
}

.group + .group {
  border-top: 1px solid #f5f5f5;
}

.group-header {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  color: #000;
  text-align: left;
  transition: background 0.15s;
}
.group-header:hover {
  background: #f8f8f8;
}
.group-header__icon {
  font-size: 18px;
  width: 24px;
  text-align: center;
  flex-shrink: 0;
}
.group-header__label {
  flex: 1;
  font-weight: 500;
}
.group-header__arrow {
  font-size: 11px;
  color: #bbb;
  transition: transform 0.2s;
  flex-shrink: 0;
}
.group-header__arrow.expanded {
  transform: rotate(90deg);
}

.group-items {
  padding-bottom: 4px;
}

.group-item {
  display: block;
  width: 100%;
  padding: 10px 16px 10px 50px;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  color: #555;
  text-align: left;
  transition: background 0.12s, color 0.12s;
  border-radius: 0 6px 6px 0;
}
.group-item:hover {
  background: #f5f5f5;
  color: #000;
}
.group-item.active {
  background: #f0f0f0;
  font-weight: 500;
  color: #000;
}
</style>
