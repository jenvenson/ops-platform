/**
 * Ant Design 主题配置
 * 基于 theme-variables.css 中的变量定义
 */

import { ThemeConfig } from 'antd';

export const antdThemeConfig: ThemeConfig = {
  token: {
    // 主色调（登录页风格渐变）
    colorPrimary: '#40a9ff',
    colorPrimaryHover: '#1890ff',
    colorPrimaryBg: 'rgba(64, 169, 255, 0.12)',

    // 背景色
    colorBgContainer: '#FFFFFF',
    colorBgLayout: '#F9FAFB',
    colorBgElevated: '#FFFFFF',

    // 文字色
    colorText: '#111827',
    colorTextSecondary: '#6B7280',
    colorTextTertiary: '#9CA3AF',
    colorTextQuaternary: '#D1D5DB',

    // 边框
    colorBorder: '#E5E7EB',
    colorBorderSecondary: '#F3F4F6',

    // 圆角
    borderRadius: 8,
    borderRadiusLG: 12,
    borderRadiusSM: 6,

    // 字体
    fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    fontSize: 14,

    // 其他
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
    Button: {
      primaryShadow: '0 2px 4px rgba(0, 97, 175, 0.2)',
      defaultShadow: '0 1px 2px rgba(0, 0, 0, 0.05)',
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
};
