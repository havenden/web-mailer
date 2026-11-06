import axios from 'axios'
import { message } from 'ant-design-vue'

const api = axios.create({ baseURL: '/api/v1', timeout: 30000 })

api.interceptors.request.use(cfg => {
  const token = localStorage.getItem('token')
  if (token) cfg.headers.Authorization = 'Bearer ' + token
  return cfg
})

api.interceptors.response.use(
  resp => {
    const d = resp.data
    if (d.code !== 200) {
      message.error(d.message || '操作失败')
      return Promise.reject(d)
    }
    return d.data !== undefined ? d.data : d
  },
  err => {
    const status = err.response?.status
    const msg = err.response?.data?.message || err.message
    if (status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      if (!location.pathname.includes('/login')) location.href = '/login'
    }
    message.error(msg)
    return Promise.reject(err)
  }
)

export default api
