interface ErrorPayload {
  message?: string
  error?: string
}

interface ErrorLike {
  message?: string
  response?: {
    data?: ErrorPayload
  }
}

export function getErrorMessage(error: unknown, fallback: string): string {
  if (!error) {
    return fallback
  }
  if (typeof error === 'string') {
    return error || fallback
  }
  if (typeof error === 'object') {
    const payload = error as ErrorLike
    const backendMessage = payload.response?.data?.error || payload.response?.data?.message
    if (backendMessage && backendMessage.trim() !== '') {
      return backendMessage
    }
    if (payload.message && payload.message.trim() !== '') {
      return payload.message
    }
  }
  return fallback
}

function mapFIMErrorMessage(raw: string): string {
  const text = raw.trim()
  const lower = text.toLowerCase()

  if (lower.includes('fim execution already running')) {
    return '当前策略与主机已有执行任务进行中，请稍后重试'
  }
  if (lower.includes('fim target not found')) {
    return '当前主机不在策略绑定范围内或绑定已失效'
  }
  if (lower.includes('has no watch paths configured')) {
    return '当前策略尚未配置巡检目录，请先完成目录配置'
  }
  if (lower.includes('record not found')) {
    return '相关策略或主机记录不存在，请刷新后重试'
  }
  if (lower.includes('invalid request')) {
    return '请求参数不合法，请检查后重试'
  }
  if (lower.includes('invalid policy id') || lower.includes('invalid snapshot id')) {
    return '请求对象无效，请刷新页面后重试'
  }
  if (lower.includes('ssh dial') || lower.includes('ssh command failed') || lower.includes('missing fim_ssh')) {
    return 'SSH 执行失败，请检查主机连通性和 FIM SSH 配置'
  }
  if (lower.includes('failed to run fim scan')) {
    return '执行巡检失败，请稍后重试'
  }
  if (lower.includes('failed to build fim baseline')) {
    return '构建基线失败，请稍后重试'
  }
  return text
}

export function getFIMErrorMessage(error: unknown, fallback: string): string {
  const base = getErrorMessage(error, fallback)
  return mapFIMErrorMessage(base)
}
