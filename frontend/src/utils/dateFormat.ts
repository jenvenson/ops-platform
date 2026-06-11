// SPDX-License-Identifier: MIT
// Copyright (c) 2026 OPS Platform Contributors

import i18next from '../i18n'

/** Get the current locale string based on i18next language for date formatting. */
export function getDateLocale(): string {
  return i18next.language === 'en-US' ? 'en-US' : 'zh-CN'
}

/**
 * Format a date string to a locale-aware string.
 * Falls back to the raw value if the date is unparseable.
 */
export function formatDateTime(value?: string | null): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString(getDateLocale())
}

/**
 * Format an already-constructed Date object to a locale-aware string.
 * Useful when the caller does custom timezone conversion before formatting.
 */
export function formatDate(date: Date): string {
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString(getDateLocale())
}
