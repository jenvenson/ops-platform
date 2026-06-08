import axios, { type AxiosInstance, type AxiosResponse } from 'axios'

interface AxiosErrorData {
  message?: string
  error?: string
}

// 重写 axios 类型以支持响应拦截器返回数据
declare module 'axios' {
  interface AxiosInstance<T = unknown> {
    get<T = unknown, R = T>(url: string, config?: unknown): Promise<R>
    delete<T = unknown, R = T>(url: string, config?: unknown): Promise<R>
    head<T = unknown, R = T>(url: string, config?: unknown): Promise<R>
    options<T = unknown, R = T>(url: string, config?: unknown): Promise<R>
    post<T = unknown, R = T, D = unknown>(url: string, data?: D, config?: unknown): Promise<R>
    put<T = unknown, R = T, D = unknown>(url: string, data?: D, config?: unknown): Promise<R>
    patch<T = unknown, R = T, D = unknown>(url: string, data?: D, config?: unknown): Promise<R>
  }
}

const baseURL = import.meta.env.VITE_API_BASE_URL ?? '/api'

const apiClient = axios.create({
  baseURL,
  timeout: 7200000, // 增加到2小时（7200000毫秒），适应长时间聚合打包任务
  headers: {
    'Content-Type': 'application/json; charset=utf-8'
  }
}) as AxiosInstance

const redirectToForbidden = () => {
  const currentPath = window.location.pathname
  if (currentPath === '/forbidden' || currentPath === '/login') {
    return
  }

  const params = new URLSearchParams()
  params.set('from', currentPath || '/')
  window.location.href = `/forbidden?${params.toString()}`
}

// 请求拦截器
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  const currentPath = window.location.pathname || '/'
  if (token) {
    config.headers = config.headers || {}
    config.headers.Authorization = `Bearer ${token}`
  }
  config.headers = config.headers || {}
  config.headers['X-Page-Path'] = currentPath
  return config
}, (error) => {
  return Promise.reject(error)
})

// 响应拦截器
apiClient.interceptors.response.use(
  (response: AxiosResponse) => response.data,
  (error: { response?: { status?: number; data?: AxiosErrorData }; request?: unknown; message?: string }) => {
    if (error.response) {
      const status = error.response.status
      const data = error.response.data as AxiosErrorData

      // 401 未授权
      if (status === 401) {
        localStorage.removeItem('token')
        window.location.href = '/login'
      }

      // 403 禁止访问
      if (status === 403) {
        console.error('Access forbidden:', data.message || data.error)
        redirectToForbidden()
      }

      // 404 资源不存在
      if (status === 404) {
        console.error('Resource not found:', data.message || data.error)
      }

      // 5xx 服务器错误
      if (status && status >= 500) {
        console.error('Server error:', data.message || data.error)
      }
    } else if (error.request) {
      // 请求已发出但无响应（网络错误）
      console.error('Network error:', error.message)
    } else {
      // 请求配置错误
      console.error('Request error:', error.message)
    }

    return Promise.reject(error)
  }
)

export default apiClient
