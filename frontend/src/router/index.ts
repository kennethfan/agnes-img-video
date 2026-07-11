import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/text2img' },
  { path: '/text2img',    name: 'text2img',    component: () => import('../views/TextToImage.vue') },
  { path: '/img2img',     name: 'img2img',     component: () => import('../views/ImageToImage.vue') },
  { path: '/batch',       name: 'batch',       component: () => import('../views/BatchGen.vue') },
  { path: '/script-gen',  name: 'script_gen',  component: () => import('../views/ScriptGen.vue') },
  { path: '/text2vid',    name: 'text2vid',    component: () => import('../views/TextToVideo.vue') },
  { path: '/img2vid',     name: 'img2vid',     component: () => import('../views/ImageToVideo.vue') },
  { path: '/multi-vid',   name: 'multi_vid',   component: () => import('../views/MultiImageVideo.vue') },
  { path: '/ideas',       name: 'ideas',       component: () => import('../views/Ideas.vue') },
  { path: '/storyboard',  name: 'storyboard',  component: () => import('../views/Storyboard.vue') },
  { path: '/assets',      name: 'assets',      component: () => import('../views/Assets.vue') },
  { path: '/tasks',       name: 'tasks',       component: () => import('../views/TaskRecords.vue') },
  { path: '/history',     name: 'history',     component: () => import('../views/History.vue') },
  { path: '/access-logs', name: 'access_logs', component: () => import('../views/AccessLogs.vue') },
  { path: '/db-manage',   name: 'db_manage',   component: () => import('../views/DBManage.vue') },
  { path: '/settings',    name: 'settings',    component: () => import('../views/Settings.vue') },
  { path: '/templates',   name: 'templates',   component: () => import('../views/TemplateManager.vue') },
  { path: '/image-refine', name: 'image_refine', component: () => import('../views/WorkflowWizard.vue') },
  { path: '/comic',        name: 'comic',        component: () => import('../views/WorkflowWizard.vue') },
  { path: '/novel',        name: 'novel',        component: () => import('../views/WorkflowWizard.vue') },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
