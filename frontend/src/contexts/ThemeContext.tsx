// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { ThemeConfig } from 'antd'

// 浅色主题配置
const lightAntdTheme: ThemeConfig = {
  token: {
    colorPrimary: '#40a9ff',
    colorPrimaryHover: '#1890ff',
    colorPrimaryBg: 'rgba(64, 169, 255, 0.12)',
    colorBgContainer: '#FFFFFF',
    colorBgLayout: '#F9FAFB',
    colorBgElevated: '#FFFFFF',
    colorText: '#111827',
    colorTextSecondary: '#6B7280',
    colorTextTertiary: '#9CA3AF',
    colorTextQuaternary: '#D1D5DB',
    colorBorder: '#E5E7EB',
    colorBorderSecondary: '#F3F4F6',
    borderRadius: 8,
    borderRadiusLG: 12,
    borderRadiusSM: 6,
    fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    fontSize: 14,
    wireframe: false,
  },
  components: {
    Layout: {
      siderBg: '#0B0E14',
      headerBg: '#FFFFFF',
      bodyBg: '#F9FAFB',
      triggerBg: '#0B0E14',
      triggerColor: '#D1D5DB',
    },
    Menu: {
      darkItemBg: '#0B0E14',
      darkItemColor: '#D1D5DB',
      darkItemHoverBg: 'rgba(255, 255, 255, 0.05)',
      darkItemHoverColor: '#FFFFFF',
      darkItemSelectedBg: 'rgba(0, 97, 175, 0.15)',
      darkItemSelectedColor: '#FFFFFF',
      darkSubMenuItemBg: '#0B0E14',
      itemMarginInline: 8,
      itemPaddingInline: 16,
      itemBorderRadius: 8,
      iconSize: 18,
      iconMarginInlineEnd: 8,
    },
    Card: {
      headerBg: '#FFFFFF',
    },
    Table: {
      headerBg: '#FAFAFA',
      headerColor: '#374151',
      rowHoverBg: '#F5F7FA',
      borderColor: '#E5E7EB',
      cellPaddingBlock: 16,
      cellPaddingInline: 16,
      borderRadiusLG: 12,
    },
    Input: {
      activeShadow: '0 0 0 2px rgba(0, 97, 175, 0.1)',
    },
    Select: {
      optionSelectedBg: 'rgba(0, 97, 175, 0.15)',
    },
  },
}

// 深色主题配置
const darkAntdTheme: ThemeConfig = {
  token: {
    colorPrimary: '#3b82f6',
    colorPrimaryHover: '#60a5fa',
    colorPrimaryBg: 'rgba(59, 130, 246, 0.15)',
    colorBgContainer: '#1a1f2e',
    colorBgLayout: '#0f1219',
    colorBgElevated: '#232938',
    colorText: '#e2e8f0',
    colorTextSecondary: '#94a3b8',
    colorTextTertiary: '#64748b',
    colorTextQuaternary: '#475569',
    colorBorder: '#2d3548',
    colorBorderSecondary: '#1e2433',
    borderRadius: 8,
    borderRadiusLG: 12,
    borderRadiusSM: 6,
    fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    fontSize: 14,
    wireframe: false,
  },
  components: {
    Layout: {
      siderBg: '#0B0E14',
      headerBg: '#1a1f2e',
      bodyBg: '#0f1219',
      triggerBg: '#0B0E14',
      triggerColor: '#D1D5DB',
    },
    Menu: {
      darkItemBg: '#0B0E14',
      darkItemColor: '#D1D5DB',
      darkItemHoverBg: 'rgba(255, 255, 255, 0.05)',
      darkItemHoverColor: '#FFFFFF',
      darkItemSelectedBg: 'rgba(59, 130, 246, 0.2)',
      darkItemSelectedColor: '#FFFFFF',
      darkSubMenuItemBg: '#0B0E14',
      itemMarginInline: 8,
      itemPaddingInline: 16,
      itemBorderRadius: 8,
      iconSize: 18,
      iconMarginInlineEnd: 8,
    },
    Card: {
      headerBg: '#1a1f2e',
    },
    Table: {
      headerBg: '#232938',
      headerColor: '#e2e8f0',
      rowHoverBg: '#232938',
      borderColor: '#2d3548',
      cellPaddingBlock: 16,
      cellPaddingInline: 16,
      borderRadiusLG: 12,
    },
    Input: {
      activeShadow: '0 0 0 2px rgba(59, 130, 246, 0.2)',
    },
    Select: {
      optionSelectedBg: 'rgba(59, 130, 246, 0.2)',
    },
  },
}

interface ThemeContextType {
  isDarkMode: boolean
  toggleTheme: () => void
  antdTheme: ThemeConfig
}

const ThemeContext = createContext<ThemeContextType>({
  isDarkMode: false,
  toggleTheme: () => {},
  antdTheme: lightAntdTheme,
})

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [isDarkMode, setIsDarkMode] = useState(() => {
    return localStorage.getItem('theme') === 'dark'
  })

  useEffect(() => {
    if (isDarkMode) {
      document.documentElement.setAttribute('data-theme', 'dark')
    } else {
      document.documentElement.setAttribute('data-theme', 'light')
    }
  }, [isDarkMode])

  const toggleTheme = () => {
    const newTheme = !isDarkMode
    setIsDarkMode(newTheme)
    localStorage.setItem('theme', newTheme ? 'dark' : 'light')
  }

  const antdTheme = isDarkMode ? darkAntdTheme : lightAntdTheme

  return (
    <ThemeContext.Provider value={{ isDarkMode, toggleTheme, antdTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  return useContext(ThemeContext)
}

export { lightAntdTheme, darkAntdTheme }