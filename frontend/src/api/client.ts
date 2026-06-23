import axios from 'axios'

const client = axios.create({
  baseURL: '',
  timeout: 120000,
})

client.interceptors.response.use(
  (response) => response,
  (error) => {
    const message = error.response?.data?.error || error.message || '请求失败'
    console.error('API Error:', message)
    return Promise.reject(new Error(message))
  }
)

export default client
