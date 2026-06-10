// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import React from 'react'
import { ExclamationCircleOutlined, AlertOutlined, InfoCircleOutlined } from '@ant-design/icons'

export const severityConfig: Record<string, { color: string; text: string; icon: React.ReactNode }> = {
  critical: { color: 'red', text: '严重', icon: React.createElement(ExclamationCircleOutlined) },
  warning:  { color: 'orange', text: '警告', icon: React.createElement(AlertOutlined) },
  info:     { color: 'blue', text: '提醒', icon: React.createElement(InfoCircleOutlined) },
}

export const categoryConfig: Record<string, { color: string; text: string }> = {
  disk:     { color: 'purple', text: '磁盘' },
  memory:   { color: 'blue', text: '内存' },
  cpu:      { color: 'geekblue', text: 'CPU' },
  instance: { color: 'green', text: '实例存活' },
  network:  { color: 'cyan', text: '网络' },
  load:     { color: 'magenta', text: '负载' },
  other:    { color: 'default', text: '其他' },
}

export const eventStatusConfig: Record<string, { color: string; text: string }> = {
  firing:       { color: 'error', text: '告警中' },
  acknowledged: { color: 'warning', text: '已介入' },
  resolved:     { color: 'success', text: '已恢复' },
  closed:       { color: 'default', text: '已关闭' },
}