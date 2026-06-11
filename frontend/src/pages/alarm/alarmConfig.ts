// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import React from 'react'
import { ExclamationCircleOutlined, AlertOutlined, InfoCircleOutlined } from '@ant-design/icons'

export const severityConfig: Record<string, { color: string; key: string; text: string; icon: React.ReactNode }> = {
  critical: { color: 'red', key: 'severityCritical', text: '严重', icon: React.createElement(ExclamationCircleOutlined) },
  warning:  { color: 'orange', key: 'severityWarning', text: '警告', icon: React.createElement(AlertOutlined) },
  info:     { color: 'blue', key: 'severityInfo', text: '提醒', icon: React.createElement(InfoCircleOutlined) },
}

export const categoryConfig: Record<string, { color: string; key: string; text: string }> = {
  disk:     { color: 'purple', key: 'categoryDisk', text: '磁盘' },
  memory:   { color: 'blue', key: 'categoryMemory', text: '内存' },
  cpu:      { color: 'geekblue', key: 'categoryCpu', text: 'CPU' },
  instance: { color: 'green', key: 'categoryInstance', text: '实例存活' },
  network:  { color: 'cyan', key: 'categoryNetwork', text: '网络' },
  load:     { color: 'magenta', key: 'categoryLoad', text: '负载' },
  other:    { color: 'default', key: 'categoryOther', text: '其他' },
}

export const eventStatusConfig: Record<string, { color: string; key: string; text: string }> = {
  firing:       { color: 'error', key: 'eventStatusFiring', text: '告警中' },
  acknowledged: { color: 'warning', key: 'eventStatusAcknowledged', text: '已介入' },
  resolved:     { color: 'success', key: 'eventStatusResolved', text: '已恢复' },
  closed:       { color: 'default', key: 'eventStatusClosed', text: '已关闭' },
}
