<script setup lang="ts">
import { ref, computed } from 'vue'

const activeGroup = ref('image')
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
]

const isOpen = ref(false)

function toggleGroup(groupId: string) {
  if (activeGroup.value === groupId && isOpen.value) {
    isOpen.value = false
  } else {
    activeGroup.value = groupId
    isOpen.value = true
  }
}

function selectPage(pageId: string) {
  activePage.value = pageId
  isOpen.value = false
}

function closeFlyout() {
  isOpen.value = false
}

const currentGroup = computed(() => groups.find(g => g.id === activeGroup.value))
</script>

<template>
  <div class="nav-sidebar" @mouseleave="closeFlyout">
    <div class="icon-bar">
      <button
        v-for="g in groups"
        :key="g.id"
        class="icon-btn"
        :class="{ active: activeGroup === g.id && isOpen }"
        @click="toggleGroup(g.id)"
        :title="g.label"
      >
        <span class="icon-btn__icon">{{ g.icon }}</span>
        <span class="icon-btn__label">{{ g.label }}</span>
      </button>
    </div>

    <Transition name="flyout">
      <div v-if="isOpen && currentGroup" class="flyout">
        <div class="flyout-header">{{ currentGroup.label }}</div>
        <button
          v-for="item in currentGroup.items"
          :key="item.id"
          class="flyout-item"
          :class="{ active: activePage === item.id }"
          @click="selectPage(item.id)"
        >
          {{ item.label }}
        </button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.nav-sidebar {
  position: relative;
  display: flex;
  align-items: flex-start;
}
.icon-bar {
  width: 76px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 20px 0;
  background: #ffffff;
  border-right: 1px solid #f0f0f0;
}
.icon-btn {
  width: 60px;
  min-height: 56px;
  border: none;
  border-radius: 12px;
  background: #f5f5f5;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 6px 0;
  transition: background 0.15s;
}
.icon-btn__icon {
  font-size: 22px;
  line-height: 1;
}
.icon-btn__label {
  font-size: 10px;
  color: #666;
  line-height: 1;
  white-space: nowrap;
}
.icon-btn.active .icon-btn__label {
  color: #fff;
}
.icon-btn.active {
  background: #000;
  color: #fff;
}
.flyout {
  position: absolute;
  left: 80px;
  top: 20px;
  width: 220px;
  background: #ffffff;
  border: 1px solid #eaeaea;
  border-radius: 14px;
  padding: 12px;
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.08);
  z-index: 100;
}
.flyout-header {
  font-size: 14px;
  font-weight: 600;
  color: #909399;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 8px 12px 12px;
}
.flyout-item {
  display: block;
  width: 100%;
  padding: 12px 14px;
  border: none;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  font-size: 15px;
  color: #000;
  text-align: left;
  transition: background 0.15s;
}
.flyout-item:hover {
  background: #f5f5f5;
}
.flyout-item.active {
  background: #f5f5f5;
  font-weight: 500;
}
.flyout-enter-active,
.flyout-leave-active {
  transition: opacity 0.15s, transform 0.15s;
}
.flyout-enter-from,
.flyout-leave-to {
  opacity: 0;
  transform: translateX(-6px);
}
</style>
