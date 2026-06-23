import { defineStore } from 'pinia'
import { ref } from 'vue'
import client from '../api/client'
import type { Config } from '../types'

export const useConfigStore = defineStore('config', () => {
  const apiKey = ref('')
  const baseUrl = ref('')
  const model = ref('')

  async function loadConfig() {
    try {
      const res = await client.get('/api/v1/config')
      baseUrl.value = res.data.base_url || ''
      model.value = res.data.model || ''
    } catch (e) {
      console.error('加载配置失败:', e)
    }
  }

  async function saveConfig() {
    try {
      const payload: Config = {
        base_url: baseUrl.value || undefined,
        model: model.value || undefined,
      }
      if (apiKey.value) {
        payload.api_key = apiKey.value
      }
      await client.put('/api/v1/config', payload)
    } catch (e) {
      console.error('保存配置失败:', e)
      throw e
    }
  }

  return { apiKey, baseUrl, model, loadConfig, saveConfig }
})
