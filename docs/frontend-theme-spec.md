# 专业商务主题设计规范 (Professional Business Theme)

## 1. 概述

### 设计理念
- **深色侧边栏 + 浅色内容区**：形成明确的视觉分区，减少干扰
- **极简线框图标**：统一使用几何符号，简洁专业
- **扁平化设计**：减少阴影和渐变，强调内容层次
- **商务蓝色调**：传达专业、可信赖的品牌形象

### 色彩对比
| 区域 | 背景色 | 用途 |
|------|--------|------|
| 侧边栏 | `#0B0E14` | 深色导航区 |
| 内容区 | `#F9FAFB` | 浅色工作区 |
| 卡片 | `#FFFFFF` | 内容容器 |

---

## 2. 色彩系统 (Color Palette)

### CSS 变量定义

```css
:root {
  /* 主色调 */
  --primary: #0061AF;
  --primary-hover: #005090;
  --primary-light: rgba(0, 97, 175, 0.15);

  /* 侧边栏 */
  --sidebar-bg: #0B0E14;
  --sidebar-hover: rgba(255, 255, 255, 0.05);
  --sidebar-active-bg: rgba(0, 97, 175, 0.15);
  --sidebar-border: rgba(255, 255, 255, 0.06);

  /* 内容区 */
  --content-bg: #F9FAFB;
  --card-bg: #FFFFFF;
  --border-color: #E5E7EB;

  /* 文字色 */
  --text-primary: #111827;
  --text-secondary: #6B7280;
  --text-muted: #9CA3AF;
  --text-sidebar: #D1D5DB;
  --text-sidebar-muted: #6B7280;

  /* 状态色 */
  --success: #059669;
  --success-bg: #ECFDF5;
  --warning: #D97706;
  --warning-bg: #FEF3C7;
  --danger: #DC2626;
  --danger-bg: #FEF3C7;

  /* 组件 */
  --badge-bg: rgba(255, 255, 255, 0.08);
}
```

### 色彩使用规范

| 用途 | 变量 | 色值 |
|------|------|------|
| 主按钮、链接 | `--primary` | `#0061AF` |
| 侧边栏背景 | `--sidebar-bg` | `#0B0E14` |
| 内容区背景 | `--content-bg` | `#F9FAFB` |
| 卡片背景 | `--card-bg` | `#FFFFFF` |
| 一级标题 | `--text-primary` | `#111827` |
| 正文文字 | `--text-secondary` | `#6B7280` |
| 次要文字 | `--text-muted` | `#9CA3AF` |
| 侧边栏菜单文字 | `--text-sidebar` | `#D1D5DB` |
| 成功状态 | `--success` | `#059669` |
| 警告状态 | `--warning` | `#D97706` |
| 危险状态 | `--danger` | `#DC2626` |

---

## 3. 字体系统 (Typography)

### 字体规范

```css
/* 字体族 */
--font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;

/* 字号 */
--font-size-xs: 0.65rem;    /* 10px - 菜单分组标题 */
--font-size-sm: 0.75rem;     /* 12px - 辅助文字 */
--font-size-base: 0.875rem;  /* 14px - 正文 */
--font-size-md: 0.9rem;      /* 15px - 菜单项 */
--font-size-lg: 1rem;        /* 16px - 卡片标题 */
--font-size-xl: 1.25rem;     /* 20px - 页面标题 */
--font-size-2xl: 1.5rem;     /* 24px - 大标题 */

/* 行高 */
--line-height-tight: 1.25;
--line-height-normal: 1.5;
--line-height-relaxed: 1.75;

/* 字重 */
--font-weight-normal: 400;
--font-weight-medium: 500;
--font-weight-semibold: 600;
--font-weight-bold: 700;
```

### 文字使用规范

| 元素 | 字号 | 字重 | 颜色 |
|------|------|------|------|
| 页面大标题 | 1.5rem | 700 | `--text-primary` |
| 页面副标题 | 0.875rem | 400 | `--text-secondary` |
| 卡片标题 | 1rem | 600 | `--text-primary` |
| 菜单分组标题 | 0.7rem | 600 | `--text-sidebar-muted` |
| 菜单项 | 0.9rem | 500 | `--text-sidebar` |
| 正文 | 0.875rem | 400 | `--text-secondary` |
| 辅助文字 | 0.75rem | 400 | `--text-muted` |

---

## 4. 侧边栏规范 (Sidebar)

### 尺寸规范

```css
/* 展开宽度 */
--sidebar-width: 240px;

/* 折叠宽度 */
--sidebar-collapsed-width: 68px;

/* 高度 */
--sidebar-height: 100vh;

/* Logo 区域 */
--sidebar-logo-height: 56px;
--sidebar-logo-padding: 1rem;

/* 菜单区域 */
--sidebar-menu-padding: 0.75rem 0;

/* 切换按钮 */
--sidebar-toggle-size: 24px;
--sidebar-toggle-offset: -12px; /* 按钮超出边界 */
--sidebar-toggle-top: 72px;      /* 按钮距顶部位置 */
```

### 侧边栏样式

```css
.sidebar {
  width: var(--sidebar-width);
  background: var(--sidebar-bg);
  height: var(--sidebar-height);
  position: fixed;
  left: 0;
  top: 0;
  display: flex;
  flex-direction: column;
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: visible;
  z-index: 100;
}

.sidebar.collapsed {
  width: var(--sidebar-collapsed-width);
  overflow: visible;
}
```

### Logo 区域

```css
.sidebar-logo {
  padding: var(--sidebar-logo-padding);
  display: flex;
  align-items: center;
  gap: 0.75rem;
  border-bottom: 1px solid var(--sidebar-border);
  min-height: var(--sidebar-logo-height);
}

.sidebar-logo-icon {
  width: 32px;
  height: 32px;
  min-width: 32px;
  background: var(--primary);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1rem;
  opacity: 0.9;
}

.sidebar-logo-text {
  font-size: 0.95rem;
  font-weight: 700;
  color: #FFFFFF;
  letter-spacing: 0.02em;
  white-space: nowrap;
  opacity: 1;
  transition: opacity 0.2s ease;
}

.sidebar.collapsed .sidebar-logo-text {
  opacity: 0;
  width: 0;
  overflow: hidden;
}
```

### 切换按钮

```css
.sidebar-toggle {
  position: absolute;
  right: var(--sidebar-toggle-offset);
  top: var(--sidebar-toggle-top);
  width: var(--sidebar-toggle-size);
  height: var(--sidebar-toggle-size);
  background: var(--sidebar-bg);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  z-index: 101;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
}

.sidebar-toggle:hover {
  background: #1a1f2e;
  border-color: rgba(255, 255, 255, 0.2);
  transform: scale(1.08);
}

.sidebar-toggle .arrow {
  font-size: 0.5rem;
  color: var(--text-muted);
  transition: transform 0.3s ease;
  line-height: 1;
}

/* 展开时箭头向左，折叠时箭头向右 */
.sidebar:not(.collapsed) .sidebar-toggle .arrow {
  transform: rotate(180deg);
}

.sidebar.collapsed .sidebar-toggle .arrow {
  transform: rotate(0deg);
}
```

---

## 5. 菜单规范 (Menu)

### 菜单分组

```css
/* 一级菜单分组 */
.menu-group {
  margin-bottom: 0.25rem;
}

.menu-group-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.6rem 1rem;
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-sidebar-muted);
  cursor: pointer;
  transition: all 0.15s ease;
  user-select: none;
}

.menu-group-title:hover {
  color: var(--text-sidebar);
}

.menu-group-title .arrow {
  font-size: 0.35rem;
  transition: transform 0.2s ease;
  opacity: 0.4;
  font-weight: 300;
}

.menu-group.collapsed .menu-group-title .arrow {
  transform: rotate(90deg);
}

.sidebar.collapsed .menu-group-title {
  justify-content: center;
  padding: 0.5rem 0.5rem;
}

.sidebar.collapsed .menu-group-title span {
  display: none;
}

/* 子菜单容器 */
.menu-items {
  overflow: hidden;
  max-height: 500px;
  transition: max-height 0.3s ease;
}

.menu-group.collapsed .menu-items {
  max-height: 0;
}
```

### 菜单项

```css
.menu-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 1rem;
  color: var(--text-sidebar);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition: all 0.15s ease;
  position: relative;
  margin: 2px 8px;
  border-radius: 8px;
}

.menu-item:hover {
  background: var(--sidebar-hover);
  color: #FFFFFF;
}

.menu-item.active {
  background: var(--sidebar-active-bg);
  color: #FFFFFF;
}

.menu-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 20px;
  background: var(--primary);
  border-radius: 0 2px 2px 0;
}

/* 折叠状态 */
.sidebar.collapsed .menu-item {
  justify-content: center;
  padding: 0.625rem;
  margin: 2px 4px;
}

.sidebar.collapsed .menu-item.active::before {
  display: none;
}

/* 图标 */
.menu-item .icon {
  width: 18px;
  height: 18px;
  min-width: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.9rem;
  opacity: 0.7;
}

.menu-item .text {
  flex: 1;
  white-space: nowrap;
  font-size: var(--font-size-md);
  letter-spacing: 0.01em;
}

.sidebar.collapsed .menu-item .text {
  display: none;
}

/* 徽章 */
.menu-item .badge {
  background: var(--badge-bg);
  color: var(--text-muted);
  font-size: 0.6rem;
  padding: 0.1rem 0.35rem;
  border-radius: 8px;
  font-weight: 500;
  min-width: 16px;
  text-align: center;
}

.menu-item.active .badge {
  background: var(--primary);
  color: #FFFFFF;
}

.sidebar.collapsed .menu-item .badge {
  opacity: 0;
}
```

### Tooltip（折叠状态提示）

```css
.sidebar.collapsed .menu-item:hover::after {
  content: attr(data-tooltip);
  position: absolute;
  left: 100%;
  top: 50%;
  transform: translateY(-50%);
  background: #1F2937;
  color: #FFFFFF;
  padding: 0.375rem 0.75rem;
  border-radius: 6px;
  font-size: 0.8rem;
  white-space: nowrap;
  z-index: 1000;
  margin-left: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.sidebar.collapsed .menu-group-title:hover::after {
  content: attr(data-tooltip);
  position: absolute;
  left: 100%;
  top: 50%;
  transform: translateY(-50%);
  background: #1F2937;
  color: #FFFFFF;
  padding: 0.375rem 0.75rem;
  border-radius: 6px;
  font-size: 0.8rem;
  white-space: nowrap;
  z-index: 1000;
  margin-left: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}
```

---

## 6. 图标规范 (Icons)

### 图标使用原则
- 使用 **SVG 线框图标**（Outline Style）
- 统一使用 `stroke="currentColor"` 继承文字颜色
- 保持 **一致的线条粗细**（stroke-width: 1.5）
- **圆角**线条（stroke-linecap: round, stroke-linejoin: round）

### SVG 图标模板

```html
<span class="icon">
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
    <!-- 图标路径 -->
  </svg>
</span>
```

### 推荐图标库

| 用途 | SVG 路径 |
|------|----------|
| **仪表盘** | `<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>` |
| **监控大屏** | `<rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/>` |
| **主机管理** | `<rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><circle cx="6" cy="6" r="1"/><circle cx="6" cy="18" r="1"/>` |
| **部署发布** | `<path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/>` |
| **归档管理** | `<path d="M21 8v13H3V8"/><path d="M1 3h22v5H1z"/><path d="M10 12h4"/>` |
| **FIM 巡检** | `<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="M9 12l2 2 4-4"/>` |
| **漏洞管理** | `<path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>` |
| **系统设置** | `<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>` |
| **用户管理** | `<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>` |
| **搜索** | `<circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>` |
| **通知** | `<path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/>` |
| **主题** | `<circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>` |
| **Logo** | `<path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/>` |

### 图标样式

```css
.menu-item .icon {
  width: 20px;
  height: 20px;
  min-width: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0.5;
  transition: opacity 0.15s ease;
}

.menu-item .icon svg {
  width: 18px;
  height: 18px;
  stroke: currentColor;
  stroke-width: 1.5;
  stroke-linecap: round;
  stroke-linejoin: round;
  fill: none;
}

.menu-item:hover .icon {
  opacity: 0.8;
}

.menu-item.active .icon {
  opacity: 1;
}

.menu-item.active .icon svg {
  stroke: var(--primary);
}
```

---

## 7. 主内容区规范 (Main Content)

### 布局结构

```css
.main-content {
  margin-left: var(--sidebar-width);
  padding: 1.5rem 2rem;
  min-height: 100vh;
  transition: margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  border-left: 1px solid rgba(0, 0, 0, 0.05);
}

.main-content.expanded {
  margin-left: var(--sidebar-collapsed-width);
}
```

### 顶部栏

```css
.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.topbar-left h1 {
  font-size: var(--font-size-2xl);
  font-weight: var(--font-weight-bold);
  color: var(--text-primary);
  letter-spacing: -0.025em;
}

.topbar-left p {
  font-size: var(--font-size-base);
  color: var(--text-secondary);
  margin-top: 0.25rem;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 1rem;
}

/* 搜索框 */
.search-box {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: #F3F4F6;
  padding: 0.5rem 0.875rem;
  border-radius: 8px;
  font-size: var(--font-size-base);
  color: var(--text-secondary);
}

.search-box span {
  color: var(--text-muted);
  font-size: var(--font-size-sm);
  background: #fff;
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  border: 1px solid var(--border-color);
}

/* 图标按钮 */
.icon-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
  font-size: 1rem;
}

.icon-btn:hover {
  background: #F3F4F6;
  color: var(--text-primary);
}

/* 头像 */
.avatar {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #667EEA, #764BA2);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: var(--font-weight-semibold);
  font-size: var(--font-size-base);
}
```

---

## 8. 卡片规范 (Cards)

### 统计卡片

```css
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.stat-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.stat-label {
  font-size: var(--font-size-base);
  color: var(--text-secondary);
  margin-bottom: 0.5rem;
}

.stat-value {
  font-size: 1.75rem;
  font-weight: var(--font-weight-bold);
  color: var(--text-primary);
  letter-spacing: -0.025em;
}

.stat-trend {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  margin-top: 0.5rem;
  padding: 0.25rem 0.5rem;
  border-radius: 9999px;
}

.stat-trend.up {
  background: var(--success-bg);
  color: var(--success);
}

.stat-trend.down {
  background: var(--warning-bg);
  color: var(--warning);
}
```

### 通用卡片

```css
.card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.card-title {
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--text-primary);
}

.card-subtitle {
  font-size: var(--font-size-sm);
  color: var(--text-muted);
  margin-top: 0.25rem;
}

.card-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 1.5rem;
  margin-bottom: 2rem;
}
```

---

## 9. 按钮规范 (Buttons)

### 按钮样式

```css
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border-radius: 8px;
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-medium);
  padding: 0.5rem 1rem;
  transition: all 0.15s ease;
  cursor: pointer;
  border: none;
}

.btn-primary {
  background: var(--primary);
  color: #FFFFFF;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.btn-primary:hover {
  background: var(--primary-hover);
}

.btn-outline {
  background: #FFFFFF;
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-outline:hover {
  background: var(--content-bg);
  border-color: #D1D5DB;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: var(--font-size-sm);
}
```

### 标签页

```css
.tabs {
  display: flex;
  gap: 4px;
  background: #F3F4F6;
  padding: 4px;
  border-radius: 8px;
}

.tab {
  padding: 0.375rem 0.75rem;
  border-radius: 6px;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab:hover {
  color: var(--text-primary);
}

.tab.active {
  background: #FFFFFF;
  color: var(--primary);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}
```

---

## 10. 表格规范 (Tables)

```css
.table {
  width: 100%;
  border-collapse: collapse;
}

.table th {
  text-align: left;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--border-color);
  background: #FAFAFA;
}

.table td {
  padding: 1rem;
  border-bottom: 1px solid var(--border-color);
  font-size: var(--font-size-base);
}

.table tr:hover td {
  background: var(--content-bg);
}

/* 状态徽章 */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.625rem;
  border-radius: 9999px;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}

.status-badge.success {
  background: var(--success-bg);
  color: var(--success);
}

.status-badge.warning {
  background: var(--warning-bg);
  color: var(--warning);
}

.status-badge.danger {
  background: var(--danger-bg);
  color: var(--danger);
}

/* 进度条 */
.progress-bar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.progress-bar-track {
  width: 60px;
  height: 6px;
  background: #F3F4F6;
  border-radius: 9999px;
  overflow: hidden;
}

.progress-bar-fill {
  height: 100%;
  border-radius: 9999px;
}

.progress-bar-label {
  font-size: var(--font-size-sm);
  color: var(--text-muted);
}
```

---

## 11. 间距系统 (Spacing)

```css
/* 间距变量 */
--spacing-xs: 0.25rem;   /* 4px */
--spacing-sm: 0.5rem;    /* 8px */
--spacing-md: 1rem;       /* 16px */
--spacing-lg: 1.5rem;     /* 24px */
--spacing-xl: 2rem;       /* 32px */
--spacing-2xl: 3rem;      /* 48px */

/* 圆角 */
--radius-sm: 4px;
--radius-md: 8px;
--radius-lg: 12px;
--radius-full: 9999px;
```

---

## 12. 动画规范 (Animations)

```css
/* 过渡时间 */
--transition-fast: 0.15s ease;
--transition-normal: 0.2s ease;
--transition-slow: 0.3s ease;

/* 缓动函数 */
--ease-default: cubic-bezier(0.4, 0, 0.2, 1);
--ease-in: cubic-bezier(0.4, 0, 1, 1);
--ease-out: cubic-bezier(0, 0, 0.2, 1);
--ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);

/* 侧边栏动画 */
.sidebar {
  transition: width var(--transition-slow) var(--ease-default);
}

.main-content {
  transition: margin-left var(--transition-slow) var(--ease-default);
}

/* 菜单项动画 */
.menu-item {
  transition: all var(--transition-fast) ease;
}

/* 按钮动画 */
.btn {
  transition: all var(--transition-fast) ease;
}

/* 折叠动画 */
.menu-items {
  transition: max-height var(--transition-slow) ease;
}
```

---

## 13. 响应式断点 (Responsive Breakpoints)

```css
/* 断点 */
--breakpoint-sm: 640px;
--breakpoint-md: 768px;
--breakpoint-lg: 1024px;
--breakpoint-xl: 1280px;

/* 移动端适配 */
@media (max-width: 768px) {
  .sidebar {
    transform: translateX(-100%);
  }

  .sidebar.mobile-open {
    transform: translateX(0);
  }

  .main-content {
    margin-left: 0;
  }
}
```

---

## 14. 菜单结构规范 (Menu Structure)

### 菜单层级定义

```typescript
interface MenuItem {
  id: string;
  title: string;           // 菜单名称
  icon?: string;           // SVG 图标
  path?: string;           // 路由路径
  badge?: number;          // 徽章数量
  children?: MenuItem[];   // 子菜单
}
```

### 标准菜单结构

```
├── 工作台
│   └── 工作台首页
├── 资产管理
│   ├── 项目管理
│   ├── 环境管理
│   ├── 主机管理
│   └── 应用管理
├── 部署中心
│   ├── 部署发布
│   ├── 归档管理
│   └── 聚合打包
├── 安全中心
│   ├── FIM巡检
│   ├── 漏洞管理
│   ├── 安全工单
│   └── 安全资产
├── 监控中心
│   ├── 监控大屏
│   └── 仪表盘
├── 告警中心
│   ├── 告警事件
│   ├── 规则管理
│   └── 联系人管理
├── 平台管理
│   ├── 平台事件
│   └── 审计日志
└── 系统设置
    ├── 用户管理
    ├── 角色管理
    └── 菜单管理
```

### 图标映射表

| 菜单 | 图标 SVG |
|------|----------|
| 工作台 | `<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/>` |
| 项目管理 | `<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>` |
| 环境管理 | `<circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>` |
| 主机管理 | `<rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><circle cx="6" cy="6" r="1"/><circle cx="6" cy="18" r="1"/>` |
| 应用管理 | `<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>` |
| 部署发布 | `<path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/>` |
| 归档管理 | `<path d="M21 8v13H3V8"/><path d="M1 3h22v5H1z"/><path d="M10 12h4"/>` |
| 聚合打包 | `<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>` |
| FIM巡检 | `<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="M9 12l2 2 4-4"/>` |
| 漏洞管理 | `<path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>` |
| 安全工单 | `<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>` |
| 安全资产 | `<rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/>` |
| 监控大屏 | `<rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/>` |
| 仪表盘 | `<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>` |
| 告警事件 | `<path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/>` |
| 规则管理 | `<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33..."/>` |
| 联系人管理 | `<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>` |
| 平台事件 | `<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>` |
| 审计日志 | `<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>` |
| 用户管理 | `<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>` |
| 角色管理 | `<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>` |
| 菜单管理 | `<line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="18" x2="21" y2="18"/>` |

---

## 15. React 组件示例

### MenuItem 组件

```tsx
// MenuItem.tsx
import './MenuItem.css';

interface MenuItemProps {
  icon: string;
  text: string;
  badge?: number;
  active?: boolean;
  collapsed?: boolean;
  onClick?: () => void;
}

export const MenuItem: React.FC<MenuItemProps> = ({
  icon,
  text,
  badge,
  active = false,
  collapsed = false,
  onClick
}) => {
  return (
    <div
      className={`menu-item ${active ? 'active' : ''} ${collapsed ? 'collapsed' : ''}`}
      data-tooltip={collapsed ? text : undefined}
      onClick={onClick}
    >
      <span className="icon">{icon}</span>
      <span className="text">{text}</span>
      {badge !== undefined && <span className="badge">{badge}</span>}
    </div>
  );
};
```

### Ant Design 主题配置

```tsx
// theme.config.ts
export const theme = {
  token: {
    colorPrimary: '#0061AF',
    colorSuccess: '#059669',
    colorWarning: '#D97706',
    colorError: '#DC2626',
    colorBgContainer: '#FFFFFF',
    colorBgLayout: '#F9FAFB',
    colorText: '#111827',
    colorTextSecondary: '#6B7280',
    fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    borderRadius: 8,
    fontSize: 14,
  },
  components: {
    Menu: {
      darkItemBg: '#0B0E14',
      darkItemColor: '#D1D5DB',
      darkItemHoverBg: 'rgba(255, 255, 255, 0.05)',
      darkItemSelectedBg: 'rgba(0, 97, 175, 0.15)',
    },
    Layout: {
      siderBg: '#0B0E14',
      bodyBg: '#F9FAFB',
    },
  },
};
```

---

## 15. 文件结构

```
frontend/src/
├── styles/
│   ├── variables.css      # CSS 变量定义
│   ├── theme.css         # 主题样式
│   └── components/       # 组件样式
│       ├── sidebar.css
│       ├── menu.css
│       ├── cards.css
│       └── buttons.css
├── components/
│   ├── Sidebar/
│   ├── Topbar/
│   └── ...
└── App.tsx
```

---

## 16. 更新日志

| 日期 | 版本 | 更新内容 |
|------|------|----------|
| 2026-04-01 | v1.0 | 初始版本，定义专业商务主题规范 |
| 2026-04-01 | v1.1 | 新增菜单结构规范，添加完整菜单树及图标映射 |
| 2026-04-01 | v1.2 | 完成折叠侧边栏功能，新增 MainLayout 改造、CSS 主题文件 |

---

## 17. 开发进度

### ✅ 已完成

1. **主题变量文件** - `frontend/src/styles/theme-variables.css`
2. **主题样式文件** - `frontend/src/styles/theme.css`
3. **Ant Design 主题配置** - `frontend/src/styles/antd-theme-config.ts`
4. **SVG 图标映射** - `frontend/src/components/Sidebar/SvgIconMap.tsx`
5. **菜单样式覆盖** - `frontend/src/styles/menu-overrides.css`
6. **MainLayout 折叠改造** - `frontend/src/components/MainLayout.tsx`
   - 添加 `collapsed` 状态
   - 添加圆形切换按钮（顶部右侧）
   - 修改 Sider 支持折叠/展开
   - Logo 区域适配折叠状态

### 📋 待完成

1. **SVG 图标替换** - 将 Ant Design 图标替换为 SVG
2. **主题配置应用** - 在 App.tsx 中应用 antdThemeConfig
3. **细化样式调整** - 根据实际效果微调样式

### 📁 已创建/修改的文件

| 文件 | 状态 |
|------|------|
| `frontend/src/styles/theme-variables.css` | ✅ 已创建 |
| `frontend/src/styles/theme.css` | ✅ 已创建 |
| `frontend/src/styles/antd-theme-config.ts` | ✅ 已创建 |
| `frontend/src/styles/menu-overrides.css` | ✅ 已创建 |
| `frontend/src/components/Sidebar/SvgIconMap.tsx` | ✅ 已创建 |
| `frontend/src/components/MainLayout.tsx` | ✅ 已修改 |
| `frontend/src/main.tsx` | ✅ 已修改 |
