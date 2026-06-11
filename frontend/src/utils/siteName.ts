// SPDX-License-Identifier: MIT
// Copyright (c) 2026 OPS Platform Contributors

import apiClient from '../api/client'
import i18next from '../i18n'

// 默认站点名跟随界面语言翻译；管理员自定义的名称保持原样
const DEFAULT_NAMES = ['OPS Platform', 'OPS Management Platform', '运维管理平台']

let rawSiteName = ''
let fetched: Promise<string> | null = null

export function isCustomSiteName(name?: string | null): boolean {
  return !!name && !DEFAULT_NAMES.includes(name.trim())
}

export function resolveSiteName(raw?: string | null): string {
  const name = raw ?? rawSiteName
  if (isCustomSiteName(name)) return (name as string).trim()
  return i18next.t('login:title', '运维管理平台')
}

export function applyDocumentTitle(): void {
  document.title = resolveSiteName()
}

export function setRawSiteName(name: string): void {
  rawSiteName = name || ''
  applyDocumentTitle()
}

export function loadSiteName(): Promise<string> {
  if (!fetched) {
    fetched = apiClient
      .get<{ site_name: string }>('/site-name')
      .then(res => {
        setRawSiteName(res.site_name || '')
        return rawSiteName
      })
      .catch(() => {
        applyDocumentTitle()
        return ''
      })
  }
  return fetched
}
