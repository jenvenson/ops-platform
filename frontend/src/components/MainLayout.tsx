// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

// @ts-nocheck
import React, { useState, useEffect, useRef, useCallback } from 'react'
const AIChatbot = React.lazy(() => import('./AIChatbot'))
import { Layout, Menu, Dropdown, Space, Tabs, message, Input, Modal } from 'antd'
import type { MenuProps } from 'antd'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import apiClient from '../api/client';
import { MENU_UPDATED_EVENT, notifyMenusChanged, readStoredMenus, readAllowedPaths, hasMenuAccess, readStoredUserInfo } from '../utils/menuAccess';
import { useTheme } from '../contexts/ThemeContext';
import { useTranslation } from 'react-i18next'
import i18next from '../i18n'

import { UserOutlined, LogoutOutlined, DashboardOutlined, ProjectOutlined, CloudOutlined, SettingOutlined, RocketOutlined, MonitorOutlined, HistoryOutlined, InboxOutlined, AppstoreOutlined, DesktopOutlined, CloseOutlined, FullscreenOutlined, FullscreenExitOutlined, TeamOutlined, BellOutlined, SafetyOutlined, ToolOutlined, MenuOutlined, MenuFoldOutlined, MenuUnfoldOutlined, IdcardOutlined, SendOutlined, AlertOutlined, FundProjectionScreenOutlined, LineChartOutlined, ScanOutlined, BugOutlined, DatabaseOutlined, FileTextOutlined, ApiOutlined, CopyOutlined, ReadOutlined, SyncOutlined, KeyOutlined, SearchOutlined, SunOutlined, MoonOutlined } from '@ant-design/icons'

const { Header, Sider, Content } = Layout

const hiddenMenuItems = new Set(['security-tickets', 'cmdb-agent'])
const hiddenMenuPaths = new Set(['/security/tickets', '/cmdb/agent'])

// 静态页面配置映射
function getStaticPageConfig(): Record<string, { title: string; icon: React.ReactNode; path: string }> {
  const t = (key: string, fallback: string) => i18next.t(key, fallback)
  return {
    dashboard: { title: t('menu:dashboard', '工作台'), icon: <DashboardOutlined />, path: '/' },
    'cmdb-projects': { title: t('menu:cmdb-projects', '项目管理'), icon: <ProjectOutlined />, path: '/cmdb/projects' },
    'cmdb-environments': { title: t('menu:cmdb-environments', '环境管理'), icon: <CloudOutlined />, path: '/cmdb/environments' },
    'cmdb-servers': { title: t('menu:cmdb-servers', '主机管理'), icon: <DesktopOutlined />, path: '/cmdb/servers' },
    'cmdb-applications': { title: t('menu:cmdb-applications', '应用流水线管理'), icon: <AppstoreOutlined />, path: '/cmdb/applications' },
    'deploy-release': { title: t('menu:deploy-release', '迭代部署'), icon: <RocketOutlined />, path: '/deploy/release' },
    'deploy-history': { title: t('menu:deploy-history', '部署记录'), icon: <HistoryOutlined />, path: '/deploy/history' },
    'deploy-archive': { title: t('menu:deploy-archive', '归档打包'), icon: <InboxOutlined />, path: '/deploy/archive' },
    'deploy-archived': { title: t('menu:deploy-archived', '归档历史'), icon: <HistoryOutlined />, path: '/deploy/archived' },
    'deploy-aggregate-package': { title: t('menu:deploy-aggregate-package', '聚合打包'), icon: <InboxOutlined />, path: '/deploy/aggregate-package' },
    'aggregated-history': { title: t('menu:aggregated-history', '聚合历史'), icon: <HistoryOutlined />, path: '/deploy/aggregated-history' },
    'monitor-bigscreen': { title: t('menu:monitor-bigscreen', '监控大屏'), icon: <MonitorOutlined />, path: '/monitor/bigscreen' },
    'monitor-overview': { title: t('menu:monitor-overview', '监控概览'), icon: <MonitorOutlined />, path: '/monitor/overview' },
    'monitor-dashboards': { title: t('menu:monitor-dashboards', 'Grafana仪表盘'), icon: <DashboardOutlined />, path: '/monitor/dashboards' },
    'alarm-center': { title: t('menu:alarm', '告警中心'), icon: <BellOutlined />, path: '/alarm' },
    'alarm-events': { title: t('menu:alarm-events', '告警事件'), icon: <BellOutlined />, path: '/alarm/events' },
    'alarm-rules': { title: t('menu:alarm-rules', '告警规则'), icon: <BellOutlined />, path: '/alarm/rules' },
    'alarm-contacts': { title: t('menu:alarm-contacts', '联系人管理'), icon: <TeamOutlined />, path: '/alarm/contacts' },
    'alarm-channels': { title: t('menu:alarm-channels', '通知渠道'), icon: <SendOutlined />, path: '/alarm/channels' },
    'alarm-templates': { title: t('menu:alarm-templates', '通知模板'), icon: <AlertOutlined />, path: '/alarm/templates' },
    'admin-users': { title: t('menu:admin-users', '用户管理'), icon: <TeamOutlined />, path: '/admin/users' },
    'admin-roles': { title: t('menu:admin-roles', '角色管理'), icon: <SafetyOutlined />, path: '/admin/roles' },
    'admin-menus': { title: t('menu:admin-menus', '菜单管理'), icon: <MenuOutlined />, path: '/admin/menus' },
    'admin-settings': { title: t('menu:admin-settings', '系统设置'), icon: <ToolOutlined />, path: '/admin/settings' },
    'security-overview': { title: t('menu:security-overview', '安全概览'), icon: <SafetyOutlined />, path: '/security/overview' },
    'security-fim-policies': { title: t('menu:security-fim-policies', '巡检策略'), icon: <SafetyOutlined />, path: '/security/fim/policies' },
    'security-fim-executions': { title: t('menu:security-fim-executions', '执行记录'), icon: <SyncOutlined />, path: '/security/fim/executions' },
    'security-fim-events': { title: t('menu:security-fim-events', '文件变更事件'), icon: <FileTextOutlined />, path: '/security/fim/events' },
    'security-fim-alerts': { title: t('menu:security-fim-alerts', '完整性告警'), icon: <AlertOutlined />, path: '/security/fim/alerts' },
    'security-fim-known-hosts': { title: t('menu:security-fim-known-hosts', 'SSH主机密钥'), icon: <KeyOutlined />, path: '/security/fim/known-hosts' },
    'security-tasks': { title: t('menu:security-tasks', '扫描任务'), icon: <ScanOutlined />, path: '/security/tasks' },
    'security-assets': { title: t('menu:security-assets', '安全资产'), icon: <DatabaseOutlined />, path: '/security/assets' },
    'security-vulnerabilities': { title: t('menu:security-vulnerabilities', '漏洞管理'), icon: <BugOutlined />, path: '/security/vulnerabilities' },
    'security-vuln-db': { title: t('menu:security-vuln-db', '漏洞知识库'), icon: <FundProjectionScreenOutlined />, path: '/security/vuln-db' },
    'consul-batch-all': { title: t('menu:consul-batch-all', '批量配置下发'), icon: <CopyOutlined />, path: '/consul/batch-all' },
    'consul-config': { title: t('menu:consul-config', '配置管理'), icon: <ApiOutlined />, path: '/consul/config' },
    'consul-operations': { title: t('menu:consul-operations', '配置操作记录'), icon: <ApiOutlined />, path: '/consul/operations' },
    'jenkins-views': { title: t('menu:jenkins-views', '视图管理'), icon: <AppstoreOutlined />, path: '/jenkins/views' },
    'platform-audit': { title: t('menu:platform-audit', '平台审计'), icon: <FileTextOutlined />, path: '/platform/audit' },
    'platform-events': { title: t('menu:platform-events', '平台事件中心'), icon: <BellOutlined />, path: '/platform/events' },
    'user-manual': { title: t('menu:user-manual', '用户手册'), icon: <ReadOutlined />, path: '/user-manual' },
    'profile': { title: t('menu:profile', '我的资料'), icon: <IdcardOutlined />, path: '/profile' },
  }
}

// 页面配置映射 - 优先使用动态菜单数据，如果没有则使用静态配置
const getPageConfig = (): Record<string, { title: string; icon: React.ReactNode; path: string }> => {
  // 获取动态菜单配置
  let dynamicPageConfig: Record<string, { title: string; icon: React.ReactNode; path: string }> = {};
  const dynamicMenus = readStoredMenus() as MenuConfig[];

  // 递归遍历菜单，构建映射（只添加有有效 path 的菜单项）
  const buildPageConfig = (menus: MenuConfig[]) => {
    menus.forEach(menu => {
      if (hiddenMenuItems.has(menu.key) || hiddenMenuPaths.has(menu.path)) {
        return
      }
      // 只有有有效 path 的菜单项才添加到配置中，用于搜索
      // 过滤条件：path 存在、不为空、不是根路径 /、不是 # 开头的锚点
      const hasValidPath = menu.path && menu.path.trim() !== '' && menu.path !== '/' && !menu.path.startsWith('#');
      if (hasValidPath) {
        dynamicPageConfig[menu.key] = {
          title: i18next.t(`menu:${menu.key}`, menu.title),
          icon: getAntdIconByName(menu.icon),
          path: menu.path
        };
      }

      if (menu.children && menu.children.length > 0) {
        buildPageConfig(menu.children);
      }
    });
  };

  buildPageConfig(dynamicMenus);

  // 合并静态配置和动态配置，优先使用动态配置
  return { ...getStaticPageConfig(), ...dynamicPageConfig };
};

// 辅助函数：根据图标名称获取对应的 Ant Design 图标组件
function getAntdIconByName(iconName: string): React.ReactNode {
  switch (iconName) {
    case 'DashboardOutlined': return <DashboardOutlined />;
    case 'ProjectOutlined': return <ProjectOutlined />;
    case 'CloudOutlined': return <CloudOutlined />;
    case 'SettingOutlined': return <SettingOutlined />;
    case 'RocketOutlined': return <RocketOutlined />;
    case 'MonitorOutlined': return <MonitorOutlined />;
    case 'HistoryOutlined': return <HistoryOutlined />;
    case 'InboxOutlined': return <InboxOutlined />;
    case 'AppstoreOutlined': return <AppstoreOutlined />;
    case 'DesktopOutlined': return <DesktopOutlined />;
    case 'FullscreenOutlined': return <FullscreenOutlined />;
    case 'FullscreenExitOutlined': return <FullscreenExitOutlined />;
    case 'TeamOutlined': return <TeamOutlined />;
    case 'BellOutlined': return <BellOutlined />;
    case 'SafetyOutlined': return <SafetyOutlined />;
    case 'ToolOutlined': return <ToolOutlined />;
    case 'MenuOutlined': return <MenuOutlined />;
    case 'IdcardOutlined': return <IdcardOutlined />;
    case 'SendOutlined': return <SendOutlined />;
    case 'AlertOutlined': return <AlertOutlined />;
    case 'FundProjectionScreenOutlined': return <FundProjectionScreenOutlined />;
    case 'LineChartOutlined': return <LineChartOutlined />;
    case 'ScanOutlined': return <ScanOutlined />;
    case 'BugOutlined': return <BugOutlined />;
    case 'DatabaseOutlined': return <DatabaseOutlined />;
    case 'FileTextOutlined': return <FileTextOutlined />;
    case 'ApiOutlined': return <ApiOutlined />;
    case 'CopyOutlined': return <CopyOutlined />;
    case 'ReadOutlined': return <ReadOutlined />;
    default: return <SettingOutlined />;
  }
}

// 静态菜单（登录前或无权限时使用）
function renderStaticMenuItems(): React.ReactNode {
  const t = (key: string, fallback: string) => i18next.t(key, fallback)
  return (
    <>
      <Menu.Item key="dashboard" icon={<DashboardOutlined />}>{t('menu:dashboard', '工作台')}</Menu.Item>
      <Menu.SubMenu key="cmdb" icon={<DatabaseOutlined />} title={t('menu:cmdb', '资产中心')}>
        <Menu.Item key="cmdb-projects" icon={<ProjectOutlined />}>{t('menu:cmdb-projects', '项目管理')}</Menu.Item>
        <Menu.Item key="cmdb-environments" icon={<CloudOutlined />}>{t('menu:cmdb-environments', '环境管理')}</Menu.Item>
        <Menu.Item key="cmdb-servers" icon={<DesktopOutlined />}>{t('menu:cmdb-servers', '主机管理')}</Menu.Item>
        <Menu.Item key="cmdb-applications" icon={<AppstoreOutlined />}>{t('menu:cmdb-applications', '应用流水线管理')}</Menu.Item>
      </Menu.SubMenu>
      <Menu.SubMenu key="deploy" icon={<RocketOutlined />} title={t('menu:deploy', '变更发布')}>
        <Menu.Item key="deploy-release" icon={<RocketOutlined />}>{t('menu:deploy-release', '迭代部署')}</Menu.Item>
        <Menu.Item key="deploy-history" icon={<HistoryOutlined />}>{t('menu:deploy-history', '部署记录')}</Menu.Item>
        <Menu.Item key="deploy-archive" icon={<InboxOutlined />}>{t('menu:deploy-archive', '归档打包')}</Menu.Item>
        <Menu.Item key="deploy-archived" icon={<HistoryOutlined />}>{t('menu:deploy-archived', '归档历史')}</Menu.Item>
        <Menu.Item key="deploy-aggregate-package" icon={<InboxOutlined />}>{t('menu:deploy-aggregate-package', '聚合打包')}</Menu.Item>
        <Menu.SubMenu key="consul" icon={<ApiOutlined />} title={t('menu:consul', 'Consul配置变更')}>
          <Menu.Item key="consul-config" icon={<ApiOutlined />}>{t('menu:consul-config', '配置管理')}</Menu.Item>
          <Menu.Item key="consul-batch-all" icon={<CopyOutlined />}>{t('menu:consul-batch-all', '批量配置下发')}</Menu.Item>
          <Menu.Item key="consul-operations" icon={<ApiOutlined />}>{t('menu:consul-operations', '配置操作记录')}</Menu.Item>
        </Menu.SubMenu>
        <Menu.SubMenu key="jenkins" icon={<AppstoreOutlined />} title={t('menu:jenkins', 'Jenkins任务')}>
          <Menu.Item key="jenkins-views" icon={<AppstoreOutlined />}>{t('menu:jenkins-views', '视图管理')}</Menu.Item>
        </Menu.SubMenu>
      </Menu.SubMenu>
      <Menu.SubMenu key="monitor" icon={<MonitorOutlined />} title={t('menu:monitor', '监控中心')}>
        <Menu.Item key="monitor-bigscreen" icon={<FundProjectionScreenOutlined />}>{t('menu:monitor-bigscreen', '监控大屏')}</Menu.Item>
        <Menu.Item key="monitor-overview" icon={<LineChartOutlined />}>{t('menu:monitor-overview', '监控概览')}</Menu.Item>
        <Menu.Item key="monitor-dashboards" icon={<DashboardOutlined />}>{t('menu:monitor-dashboards', 'Grafana仪表盘')}</Menu.Item>
      </Menu.SubMenu>
      <Menu.Item key="platform-events" icon={<BellOutlined />}>{t('menu:platform-events', '平台事件中心')}</Menu.Item>
      <Menu.SubMenu key="alarm" icon={<BellOutlined />} title={t('menu:alarm', '告警中心')}>
        <Menu.Item key="alarm-events" icon={<AlertOutlined />}>{t('menu:alarm-events', '告警事件')}</Menu.Item>
        <Menu.Item key="alarm-rules" icon={<BellOutlined />}>{t('menu:alarm-rules', '告警规则')}</Menu.Item>
        <Menu.Item key="alarm-contacts" icon={<TeamOutlined />}>{t('menu:alarm-contacts', '联系人管理')}</Menu.Item>
        <Menu.Item key="alarm-channels" icon={<SendOutlined />}>{t('menu:alarm-channels', '通知渠道')}</Menu.Item>
        <Menu.Item key="alarm-templates" icon={<AlertOutlined />}>{t('menu:alarm-templates', '通知模板')}</Menu.Item>
      </Menu.SubMenu>
      <Menu.SubMenu key="security" icon={<SafetyOutlined />} title={t('menu:security', '安全中心')}>
        <Menu.Item key="security-overview" icon={<SafetyOutlined />}>{t('menu:security-overview', '安全概览')}</Menu.Item>
        <Menu.SubMenu key="security-fim" icon={<SafetyOutlined />} title={t('menu:security-fim', '文件完整性巡检')}>
          <Menu.Item key="security-fim-policies" icon={<SafetyOutlined />}>{t('menu:security-fim-policies', '巡检策略')}</Menu.Item>
          <Menu.Item key="security-fim-executions" icon={<SyncOutlined />}>{t('menu:security-fim-executions', '执行记录')}</Menu.Item>
          <Menu.Item key="security-fim-events" icon={<FileTextOutlined />}>{t('menu:security-fim-events', '文件变更事件')}</Menu.Item>
          <Menu.Item key="security-fim-alerts" icon={<AlertOutlined />}>{t('menu:security-fim-alerts', '完整性告警')}</Menu.Item>
          <Menu.Item key="security-fim-known-hosts" icon={<KeyOutlined />}>{t('menu:security-fim-known-hosts', 'SSH主机密钥')}</Menu.Item>
        </Menu.SubMenu>
        <Menu.Item key="security-tasks" icon={<ScanOutlined />}>{t('menu:security-tasks', '扫描任务')}</Menu.Item>
        <Menu.Item key="security-assets" icon={<DatabaseOutlined />}>{t('menu:security-assets', '安全资产')}</Menu.Item>
        <Menu.Item key="security-vulnerabilities" icon={<BugOutlined />}>{t('menu:security-vulnerabilities', '漏洞管理')}</Menu.Item>
        <Menu.Item key="security-vuln-db" icon={<FundProjectionScreenOutlined />}>{t('menu:security-vuln-db', '漏洞知识库')}</Menu.Item>
      </Menu.SubMenu>
      <Menu.SubMenu key="admin" icon={<SettingOutlined />} title={t('menu:admin', '系统管理')}>
        <Menu.Item key="admin-users" icon={<TeamOutlined />}>{t('menu:admin-users', '用户管理')}</Menu.Item>
        <Menu.Item key="admin-roles" icon={<SafetyOutlined />}>{t('menu:admin-roles', '角色管理')}</Menu.Item>
        <Menu.Item key="admin-menus" icon={<MenuOutlined />}>{t('menu:admin-menus', '菜单管理')}</Menu.Item>
        <Menu.Item key="platform-audit" icon={<FileTextOutlined />}>{t('menu:platform-audit', '平台审计')}</Menu.Item>
        <Menu.Item key="admin-settings" icon={<ToolOutlined />}>{t('menu:admin-settings', '系统设置')}</Menu.Item>
      </Menu.SubMenu>
    </>
  )
}

// 动态渲染菜单（基于后端获取的数据）
interface MenuConfig {
  key: string;
  path: string;
  title: string;
  icon: string;
  children?: MenuConfig[];
}

function renderDynamicMenuItems(menus: MenuConfig[]): React.ReactNode {
  return menus.filter((menu) => !hiddenMenuItems.has(menu.key) && !hiddenMenuPaths.has(menu.path)).map((menu) => {
    const title = i18next.t(`menu:${menu.key}`, menu.title)
    if (menu.children && menu.children.length > 0) {
      const getIconComponent = (iconName: string) => {
        switch (iconName) {
          case 'DashboardOutlined': return <DashboardOutlined />;
          case 'ProjectOutlined': return <ProjectOutlined />;
          case 'CloudOutlined': return <CloudOutlined />;
          case 'SettingOutlined': return <SettingOutlined />;
          case 'RocketOutlined': return <RocketOutlined />;
          case 'MonitorOutlined': return <MonitorOutlined />;
          case 'HistoryOutlined': return <HistoryOutlined />;
          case 'InboxOutlined': return <InboxOutlined />;
          case 'AppstoreOutlined': return <AppstoreOutlined />;
          case 'DesktopOutlined': return <DesktopOutlined />;
          case 'FullscreenOutlined': return <FullscreenOutlined />;
          case 'FullscreenExitOutlined': return <FullscreenExitOutlined />;
          case 'TeamOutlined': return <TeamOutlined />;
          case 'BellOutlined': return <BellOutlined />;
          case 'SafetyOutlined': return <SafetyOutlined />;
          case 'ToolOutlined': return <ToolOutlined />;
          case 'MenuOutlined': return <MenuOutlined />;
          case 'IdcardOutlined': return <IdcardOutlined />;
          case 'SendOutlined': return <SendOutlined />;
          case 'AlertOutlined': return <AlertOutlined />;
          case 'FundProjectionScreenOutlined': return <FundProjectionScreenOutlined />;
          case 'LineChartOutlined': return <LineChartOutlined />;
          case 'ScanOutlined': return <ScanOutlined />;
          case 'SyncOutlined': return <SyncOutlined />;
          case 'BugOutlined': return <BugOutlined />;
          case 'DatabaseOutlined': return <DatabaseOutlined />;
          case 'FileTextOutlined': return <FileTextOutlined />;
          case 'ApiOutlined': return <ApiOutlined />;
          case 'CopyOutlined': return <CopyOutlined />;
          case 'ReadOutlined': return <ReadOutlined />;
          default: return <SettingOutlined />;
        }
      };

      return (
        <Menu.SubMenu
          key={menu.key}
          icon={getIconComponent(menu.icon)}
          title={title}
        >
          {renderDynamicMenuItems(menu.children)}
        </Menu.SubMenu>
      );
    } else {
      const getIconComponent = (iconName: string) => {
        switch (iconName) {
          case 'DashboardOutlined': return <DashboardOutlined />;
          case 'ProjectOutlined': return <ProjectOutlined />;
          case 'CloudOutlined': return <CloudOutlined />;
          case 'SettingOutlined': return <SettingOutlined />;
          case 'RocketOutlined': return <RocketOutlined />;
          case 'MonitorOutlined': return <MonitorOutlined />;
          case 'HistoryOutlined': return <HistoryOutlined />;
          case 'InboxOutlined': return <InboxOutlined />;
          case 'AppstoreOutlined': return <AppstoreOutlined />;
          case 'DesktopOutlined': return <DesktopOutlined />;
          case 'FullscreenOutlined': return <FullscreenOutlined />;
          case 'FullscreenExitOutlined': return <FullscreenExitOutlined />;
          case 'TeamOutlined': return <TeamOutlined />;
          case 'BellOutlined': return <BellOutlined />;
          case 'SafetyOutlined': return <SafetyOutlined />;
          case 'ToolOutlined': return <ToolOutlined />;
          case 'MenuOutlined': return <MenuOutlined />;
          case 'IdcardOutlined': return <IdcardOutlined />;
          case 'SendOutlined': return <SendOutlined />;
          case 'AlertOutlined': return <AlertOutlined />;
          case 'FundProjectionScreenOutlined': return <FundProjectionScreenOutlined />;
          case 'LineChartOutlined': return <LineChartOutlined />;
          case 'ScanOutlined': return <ScanOutlined />;
          case 'SyncOutlined': return <SyncOutlined />;
          case 'BugOutlined': return <BugOutlined />;
          case 'DatabaseOutlined': return <DatabaseOutlined />;
          case 'FileTextOutlined': return <FileTextOutlined />;
          case 'ApiOutlined': return <ApiOutlined />;
          case 'CopyOutlined': return <CopyOutlined />;
          case 'ReadOutlined': return <ReadOutlined />;
          default: return <SettingOutlined />;
        }
      };

      return (
        <Menu.Item
          key={menu.key}
          icon={getIconComponent(menu.icon)}
        >
          {title}
        </Menu.Item>
      );
    }
  });
}

interface TabItem {
  key: string
  label: React.ReactNode
  closable?: boolean
}

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { isDarkMode, toggleTheme } = useTheme()
  const { t } = useTranslation('menu')
  const { t: tc } = useTranslation('common')

  const [selectedKeys, setSelectedKeys] = useState<string[]>([])
  const [openKeys, setOpenKeys] = useState<string[]>([])
  const [activeTab, setActiveTab] = useState<string>('dashboard')
  const [tabItems, setTabItems] = useState<TabItem[]>([])
  const tabListRef = useRef<Record<string, boolean>>({})
  const [userRole, setUserRole] = useState<string>('')
  const [username, setUsername] = useState<string>('')
  const [realName, setRealName] = useState<string>('')
  const [menuVersion, setMenuVersion] = useState(0)
  const deniedPathRef = useRef<string>('')
  const [collapsed, setCollapsed] = useState(false)
  const [siderHovered, setSiderHovered] = useState(false)
  const [toggleHovered, setToggleHovered] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchValue, setSearchValue] = useState('')

  const refreshMenus = useCallback(async () => {
    try {
      const response = await fetch('/api/user/menus', {
        headers: {
          'Accept': 'application/json; charset=utf-8',
          'Content-Type': 'application/json; charset=utf-8',
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        }
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();

      if (data.menus) {
        const enhancedMenus = data.menus as MenuConfig[]
        console.log('获取到的菜单数据:', enhancedMenus);
        localStorage.setItem('user_menus', JSON.stringify(enhancedMenus));
        notifyMenusChanged();
        setMenuVersion(v => v + 1);
      }
    } catch (error) {
      console.error('获取菜单失败:', error);
    }
  }, []);

  useEffect(() => {
    const loadUserInfo = () => {
      const userInfo = readStoredUserInfo()
      setUsername(userInfo?.username || '')
      setRealName(userInfo?.real_name || '')
      setUserRole(userInfo?.role || 'user')
    }

    loadUserInfo()
    refreshMenus()

    const handleStorageChange = () => {
      loadUserInfo();
      setTabItems(prev => [...prev]);
    };

    window.addEventListener('storage', handleStorageChange);

    const handleMenuUpdate = () => {
      refreshMenus()
    };

    window.addEventListener(MENU_UPDATED_EVENT, handleMenuUpdate);

    return () => {
      window.removeEventListener('storage', handleStorageChange)
      window.removeEventListener(MENU_UPDATED_EVENT, handleMenuUpdate);
    }
  }, [refreshMenus])

  // 根据路径获取页面key
  const getPageKey = (pathname: string): string => {
    if (pathname === '/') return 'dashboard'
    if (pathname.startsWith('/cmdb/projects')) return 'cmdb-projects'
    if (pathname.startsWith('/cmdb/environments')) return 'cmdb-environments'
    if (pathname.startsWith('/cmdb/servers')) return 'cmdb-servers'
    if (pathname.startsWith('/cmdb/applications')) return 'cmdb-applications'
    if (pathname.startsWith('/deploy/release')) return 'deploy-release'
    if (pathname.startsWith('/deploy/history')) return 'deploy-history'
    if (pathname.startsWith('/deploy/archived')) return 'deploy-archived'
    if (pathname.startsWith('/deploy/aggregate-package')) return 'deploy-aggregate-package'
    if (pathname.startsWith('/deploy/aggregated-history')) return 'aggregated-history'
    if (pathname.startsWith('/deploy/archive')) return 'deploy-archive'
    if (pathname.startsWith('/monitor/bigscreen')) return 'monitor-bigscreen'
    if (pathname.startsWith('/monitor/overview')) return 'monitor-overview'
    if (pathname.startsWith('/monitor/dashboards')) return 'monitor-dashboards'
    if (pathname.startsWith('/monitor')) return 'monitor-bigscreen'
    if (pathname.startsWith('/alarm/events')) return 'alarm-events'
    if (pathname.startsWith('/alarm/rules')) return 'alarm-rules'
    if (pathname.startsWith('/alarm/contacts')) return 'alarm-contacts'
    if (pathname.startsWith('/alarm/channels')) return 'alarm-channels'
    if (pathname.startsWith('/alarm/templates')) return 'alarm-templates'
    if (pathname.startsWith('/alarm')) return 'alarm-events'
    if (pathname.startsWith('/security/overview')) return 'security-overview'
    if (pathname.startsWith('/security/fim/policies')) return 'security-fim-policies'
    if (pathname.startsWith('/security/fim/executions')) return 'security-fim-executions'
    if (pathname.startsWith('/security/fim/events')) return 'security-fim-events'
    if (pathname.startsWith('/security/fim/alerts')) return 'security-fim-alerts'
    if (pathname.startsWith('/security/fim/known-hosts')) return 'security-fim-known-hosts'
    if (pathname.startsWith('/security/fim')) return 'security-fim-policies'
    if (pathname.startsWith('/security/tasks')) return 'security-tasks'
    if (pathname.startsWith('/security/assets')) return 'security-assets'
    if (pathname.startsWith('/security/vulnerabilities')) return 'security-vulnerabilities'
    if (pathname.startsWith('/security/vuln-db')) return 'security-vuln-db'
    if (pathname.startsWith('/security')) return 'security-overview'
    if (pathname.startsWith('/consul/operations')) return 'consul-operations'
    if (pathname.startsWith('/consul/config')) return 'consul-config'
    if (pathname.startsWith('/consul/batch-all')) return 'consul-batch-all'
    if (pathname.startsWith('/jenkins/views')) return 'jenkins-views'
    if (pathname.startsWith('/jenkins')) return 'jenkins-views'
    if (pathname.startsWith('/consul')) return 'consul-operations'
    if (pathname.startsWith('/platform/audit')) return 'platform-audit'
    if (pathname.startsWith('/platform/events')) return 'platform-events'
    if (pathname.startsWith('/admin/users')) return 'admin-users'
    if (pathname.startsWith('/admin/roles')) return 'admin-roles'
    if (pathname.startsWith('/admin/menus')) return 'admin-menus'
    if (pathname.startsWith('/admin/settings')) return 'admin-settings'
    if (pathname.startsWith('/user-manual')) return 'user-manual'
    if (pathname.startsWith('/profile')) return 'profile'
    return 'dashboard'
  }

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (!token) {
      navigate('/login')
      return
    }

    const currentPageKey = getPageKey(location.pathname)
    const pageConfig = getPageConfig()
    const canonicalPath = pageConfig[currentPageKey]?.path || location.pathname
    const alwaysAllowedPaths = new Set(['/', '/profile', '/user-manual', '/forbidden'])
    const storedMenus = readStoredMenus() as MenuConfig[]

    if (alwaysAllowedPaths.has(canonicalPath)) {
      deniedPathRef.current = ''
      return
    }

    if (storedMenus.length === 0) {
      deniedPathRef.current = ''
      return
    }

    const allowedPaths = readAllowedPaths()
    if (hasMenuAccess(canonicalPath, allowedPaths)) {
      deniedPathRef.current = ''
      return
    }

    if (deniedPathRef.current !== canonicalPath) {
      deniedPathRef.current = canonicalPath
      message.warning(t('noPermission', '当前账号没有该页面权限'))
    }
    navigate('/forbidden', { replace: true, state: { from: canonicalPath } })
  }, [location.pathname, navigate, menuVersion])

  // 添加标签
  const addTab = (key: string) => {
    const config = getPageConfig()[key]
    if (!config) return

    if (tabListRef.current[key]) return

    const newTab: TabItem = {
      key: key,
      label: (
        <span>
          {config.icon}
          <span style={{ marginLeft: 4 }}>{config.title}</span>
        </span>
      ),
      closable: key !== 'dashboard',
    }

    setTabItems(prev => [...prev, newTab])
    tabListRef.current[key] = true
    setActiveTab(key)
  }

  // 关闭标签
  const removeTab = (key: string) => {
    if (key === 'dashboard') return

    const index = tabItems.findIndex(tab => tab.key === key)
    if (index === -1) return

    const newTabs = tabItems.filter(tab => tab.key !== key)
    setTabItems(newTabs)
    delete tabListRef.current[key]

    if (activeTab === key) {
      const newIndex = Math.max(0, index - 1)
      const targetKey = newTabs[newIndex]?.key || 'dashboard'
      setActiveTab(targetKey)
      navigate(getPageConfig()[targetKey]?.path || '/')
    }
  }

  // 根据URL获取应该展开的父菜单key
  const getOpenKeys = (pathname: string): string[] => {
    if (pathname.startsWith('/cmdb')) return ['cmdb']
    if (pathname.startsWith('/deploy')) return ['deploy']
    if (pathname.startsWith('/monitor')) return ['monitor']
    if (pathname.startsWith('/alarm')) return ['alarm']
    if (pathname.startsWith('/security/fim')) return ['security', 'security-fim']
    if (pathname.startsWith('/security')) return ['security']
    if (pathname.startsWith('/admin')) return ['admin']
    if (pathname.startsWith('/jenkins')) return ['deploy', 'jenkins']
    if (pathname.startsWith('/consul')) return ['deploy', 'consul']
    return []
  }

  // URL变化时同步菜单状态和标签
  useEffect(() => {
    const pageKey = getPageKey(location.pathname)
    setSelectedKeys([pageKey])
    setOpenKeys(getOpenKeys(location.pathname))
    setActiveTab(pageKey)
    addTab(pageKey)

  }, [location.pathname])

  // 菜单点击处理
  const handleMenuClick: MenuProps['onClick'] = (info) => {
    const { key } = info

    const pageConfig = getPageConfig();
    const path = pageConfig[key]?.path;

    if (path) {
      navigate(path);
    } else if (key === 'dashboard') {
      navigate('/');
    } else if (key.startsWith('cmdb-')) {
      const subKey = key.replace('cmdb-', '')
      navigate(`/cmdb/${subKey}`)
    } else if (key.startsWith('deploy-')) {
      const subKey = key.replace('deploy-', '')
      navigate(`/deploy/${subKey}`)
    } else if (key.startsWith('monitor-')) {
      const subKey = key.replace('monitor-', '')
      navigate(`/monitor/${subKey}`)
    } else if (key.startsWith('alarm-')) {
      const subKey = key.replace('alarm-', '')
      navigate(`/alarm/${subKey}`)
    } else if (key.startsWith('security-')) {
      const subKey = key.replace('security-', '')
      navigate(`/security/${subKey}`)
    } else if (key.startsWith('consul-')) {
      const subKey = key.replace('consul-', '')
      navigate(`/consul/${subKey}`)
    } else if (key.startsWith('jenkins-')) {
      const subKey = key.replace('jenkins-', '')
      navigate(`/jenkins/${subKey}`)
    } else if (key.startsWith('admin-')) {
      const subKey = key.replace('admin-', '')
      navigate(`/admin/${subKey}`)
    }
  }

  // 菜单展开变化处理
  const handleOpenChange = (keys: string[]) => {
    setOpenKeys(keys)
  }

  // Tab 切换
  const handleTabChange = (key: string) => {
    setActiveTab(key)
    navigate(getPageConfig()[key]?.path || '/')
  }

  // 关闭标签
  const handleTabClose = (key: string | any, action: 'add' | 'remove') => {
    if (action === 'remove' && typeof key === 'string') {
      removeTab(key)
    }
  }

  // 右键菜单
  const [contextMenuVisible, setContextMenuVisible] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [fullscreenTabKey, setFullscreenTabKey] = useState<string>('')
  const [menuPosition, setMenuPosition] = useState<{ x: number; y: number } | null>(null)

  // 切换全屏
  const toggleFullscreen = (key: string) => {
    setIsFullscreen(true)
    setFullscreenTabKey(key)
    setContextMenuVisible(false)
  }

  // 退出全屏
  const exitFullscreen = () => {
    setIsFullscreen(false)
    setFullscreenTabKey('')
  }

  // 右键菜单项点击
  const handleContextMenuClick: MenuProps['onClick'] = (info) => {
    const { key } = info
    if (key === 'close') {
      removeTab(rightClickTabKey)
    } else if (key === 'fullscreen') {
      toggleFullscreen(rightClickTabKey)
    } else if (key === 'closeOthers') {
      const newTabs = tabItems.filter(tab => tab.key === 'dashboard' || tab.key === rightClickTabKey)
      setTabItems(newTabs)
      Object.keys(tabListRef.current).forEach(k => {
        if (k !== 'dashboard' && k !== rightClickTabKey) {
          delete tabListRef.current[k]
        }
      })
    } else if (key === 'closeAll') {
      setTabItems([tabItems[0]])
      Object.keys(tabListRef.current).forEach(k => {
        if (k !== 'dashboard') {
          delete tabListRef.current[k]
        }
      })
      setActiveTab('dashboard')
      navigate('/')
    }
    setContextMenuVisible(false)
  }

  const contextMenuItems: MenuProps['items'] = [
    { key: 'fullscreen', icon: <FullscreenOutlined />, label: t('fullscreen', '全屏显示') },
    { type: 'divider' as const },
    { key: 'close', icon: <CloseOutlined />, label: t('closeTab', '关闭标签') },
    { key: 'closeOthers', label: t('closeOtherTabs', '关闭其他标签') },
    { key: 'closeAll', label: t('closeAllTabs', '关闭所有标签') },
  ]

  // 点击其他地方关闭右键菜单
  useEffect(() => {
    const handleClick = () => {
      if (contextMenuVisible) {
        setContextMenuVisible(false)
      }
    }
    document.addEventListener('click', handleClick)
    return () => document.removeEventListener('click', handleClick)
  }, [contextMenuVisible])

  // 自定义 Tab 渲染，添加右键菜单支持
  const [rightClickTabKey, setRightClickTabKey] = useState<string>('')

  const renderTabBar = (tabBarProps: any) => {
    return (
      <div
        onContextMenu={(e) => {
          e.preventDefault()
          e.stopPropagation()
          setRightClickTabKey(activeTab)
          setMenuPosition({ x: e.clientX, y: e.clientY })
          setContextMenuVisible(true)
        }}
        style={{ display: 'flex', alignItems: 'center', overflow: 'hidden' }}
      >
        <Tabs
          {...tabBarProps}
          activeKey={activeTab}
          onChange={handleTabChange}
          onEdit={handleTabClose}
          items={tabItems}
          type="editable-card"
          size="small"
          style={{ marginBottom: 0, flex: 1, overflow: 'hidden' }}
          hideAdd
          tabBarStyle={{ overflowX: 'auto', flexWrap: 'nowrap' }}
        />
        {/* 右键菜单 Dropdown */}
        <Dropdown
          menu={{ items: contextMenuItems, onClick: handleContextMenuClick }}
          trigger={['click']}
          open={contextMenuVisible}
          placement="bottomLeft"
          getPopupContainer={() => document.body}
          overlayStyle={{ zIndex: 10000 }}
        >
          <div
            style={{
              position: 'fixed',
              left: menuPosition?.x || 0,
              top: menuPosition?.y || 0,
              width: 1,
              height: 1,
              pointerEvents: 'none',
            }}
          />
        </Dropdown>
      </div>
    )
  }

  const handleLogout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user_menus')
    localStorage.removeItem('user_info')
    navigate('/login')
  }

  const handleProfile = () => {
    navigate('/profile')
  }

  const handleUserManual = () => {
    navigate('/user-manual')
  }

  const userMenuItems: MenuProps['items'] = [
    { key: 'user-manual', icon: <ReadOutlined />, label: t('user-manual', '用户手册'), onClick: handleUserManual },
    { type: 'divider' as const },
    { key: 'profile', icon: <IdcardOutlined />, label: t('profile', '我的资料'), onClick: handleProfile },
    { type: 'divider' as const },
    { key: 'logout', icon: <LogoutOutlined />, label: t('logout', '退出登录'), onClick: handleLogout },
  ]

  // 获取角色显示名称
  const getRoleDisplayName = (roleCode: string): string => {
    return tc(`roleNames.${roleCode}`, roleCode)
  }

  // 渲染菜单项 - 动态获取后端菜单数据
  function getMenuContent(): React.ReactNode {
    const menus = readStoredMenus() as MenuConfig[]
    if (menus.length > 0) {
      return (
        <>
          {renderDynamicMenuItems(menus)}
        </>
      )
    }
    return renderStaticMenuItems()
  }

  // 获取所有菜单项用于搜索 - 只包含有效的子页面
  function getAllMenuItems(): Array<{ key: string; title: string; path: string; icon: React.ReactNode }> {
    const pageConfig = getPageConfig()
    const validPathPatterns = [
      '/cmdb/projects', '/cmdb/environments', '/cmdb/servers', '/cmdb/applications',
      '/deploy/release', '/deploy/history', '/deploy/archive', '/deploy/archived',
      '/deploy/aggregate-package', '/deploy/aggregated-history',
      '/consul/config', '/consul/batch-all', '/consul/operations',
      '/jenkins/views',
      '/monitor/bigscreen', '/monitor/overview', '/monitor/dashboards',
      '/alarm', '/alarm/events', '/alarm/rules', '/alarm/contacts', '/alarm/channels', '/alarm/templates',
      '/security/overview', '/security/tasks', '/security/assets', '/security/vulnerabilities',
      '/security/fim/policies', '/security/fim/executions', '/security/fim/events',
      '/security/fim/alerts', '/security/fim/known-hosts',
      '/platform/audit', '/platform/events',
      '/admin/users', '/admin/roles', '/admin/menus', '/admin/settings',
      '/user-manual', '/profile'
    ]
    return Object.entries(pageConfig)
      .filter(([key, config]) => {
        if (!config.path) return false
        if (config.path === '/') return false
        return validPathPatterns.includes(config.path)
      })
      .map(([key, config]) => ({
        key,
        title: config.title,
        path: config.path,
        icon: config.icon,
      }))
  }

  // 搜索过滤
  const filteredMenuItems = searchValue.trim()
    ? getAllMenuItems().filter(item =>
        item.title.toLowerCase().includes(searchValue.toLowerCase())
      )
    : getAllMenuItems()

  // 键盘快捷键 Cmd+K
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setSearchOpen(true)
      }
      if (e.key === 'Escape' && searchOpen) {
        setSearchOpen(false)
        setSearchValue('')
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [searchOpen])

  // 搜索跳转
  const handleSearchSelect = (path: string) => {
    setSearchOpen(false)
    setSearchValue('')
    navigate(path)
  }

  return (
    <>
      {/* 主布局 */}
      <Layout style={{ minHeight: '100vh' }}>
        {/* 非全屏时显示侧边栏 */}
        {!isFullscreen && (
          <Sider
            width={240}
            collapsedWidth={68}
            collapsed={collapsed}
            collapsible
            trigger={null}
            className="ant-layout-sider-custom"
            onMouseEnter={() => setSiderHovered(true)}
            onMouseLeave={() => setSiderHovered(false)}
            style={{
              background: '#0B0E14',
              height: '100vh',
              position: 'fixed',
              left: 0,
              top: 0,
              bottom: 0,
              zIndex: 100,
              overflow: siderHovered ? 'visible' : 'hidden',
            }}
          >
            {/* Logo 区域 */}
            <div
              style={{
                height: 56,
                display: 'flex',
                alignItems: 'center',
                justifyContent: collapsed ? 'center' : 'flex-start',
                padding: collapsed ? '0' : '0 1rem',
                borderBottom: '1px solid rgba(255, 255, 255, 0.06)',
                gap: '0.75rem',
              }}
            >
              <div
                style={{
                  width: 32,
                  height: 32,
                  minWidth: 32,
                  background: 'linear-gradient(135deg, #40a9ff 0%, #096dd9 100%)',
                  borderRadius: 8,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '0.75rem',
                  fontWeight: 800,
                  color: '#fff',
                  letterSpacing: '0.05em',
                }}
              >
                OPS
              </div>
              {!collapsed && (
                <span style={{ color: '#fff', fontSize: '0.95rem', fontWeight: 700, letterSpacing: '0.02em' }}>
                  {t('siteTitle', '运维管理平台')}
                </span>
              )}
            </div>

            <Menu
              theme="dark"
              mode="inline"
              selectedKeys={selectedKeys}
              openKeys={openKeys}
              onClick={handleMenuClick}
              onOpenChange={handleOpenChange}
              style={{
                background: '#0B0E14',
                borderRight: 'none',
                marginTop: 8,
              }}
              inlineIndent={20}
              forceSubMenuRender
              collapsed={collapsed}
            >
              {getMenuContent()}
            </Menu>
          </Sider>
        )}

        {/* 侧边栏切换按钮 */}
        {!isFullscreen && (
          <div
            className="sidebar-toggle"
            onClick={() => setCollapsed(!collapsed)}
            onMouseEnter={() => setToggleHovered(true)}
            onMouseLeave={() => setToggleHovered(false)}
            style={{
              position: 'fixed',
              left: collapsed ? 50 : 222,
              top: 68,
              width: 32,
              height: 32,
              background: '#0B0E14',
              borderRadius: '50%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              zIndex: 101,
              boxShadow: toggleHovered
                ? '0 0 0 2px #3D9BFF, 0 4px 12px rgba(61, 155, 255, 0.3), 0 2px 8px rgba(0, 0, 0, 0.3)'
                : '0 2px 8px rgba(0, 0, 0, 0.3)',
              opacity: siderHovered || toggleHovered ? 1 : 0,
              pointerEvents: siderHovered || toggleHovered ? 'auto' : 'none',
              transition: 'left 0.3s ease, box-shadow 0.2s ease, opacity 0.2s ease',
            }}
          >
            <span
              style={{
                fontSize: 18,
                color: '#E5E7EB',
                fontWeight: 300,
                display: 'block',
                marginTop: -1,
              }}
            >
              {collapsed ? '›' : '‹'}
            </span>
          </div>
        )}

        <Layout style={{ marginLeft: isFullscreen ? 0 : collapsed ? 68 : 240 }}>
          {/* 非全屏时显示头部 */}
          {!isFullscreen && (
            <Header
              style={{
                background: 'var(--card-bg)',
                padding: '0 24px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                position: 'sticky',
                top: 0,
                zIndex: 9,
                boxShadow: '0 2px 8px rgba(0,0,0,0.09)',
                height: 44,
              }}
            >
              <div
                onClick={() => setSearchOpen(true)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  background: 'var(--content-bg)',
                  border: '1px solid var(--border-color)',
                  borderRadius: 6,
                  padding: '0 12px',
                  cursor: 'pointer',
                  minWidth: 220,
                  height: 32,
                }}
              >
                <SearchOutlined style={{ color: 'var(--text-muted)', fontSize: 13 }} />
                <span style={{ color: 'var(--text-muted)', fontSize: 13, flex: 1 }}>{t('searchMenu', '搜索菜单...')}</span>
                <span style={{
                  background: 'var(--card-bg)',
                  border: '1px solid var(--border-color)',
                  borderRadius: 4,
                  padding: '1px 5px',
                  fontSize: 11,
                  color: 'var(--text-secondary)',
                  lineHeight: 1.4,
                }}>
                  ⌘K
                </span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                {/* 工具图标区域 */}
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  {/* 主题切换按钮 */}
                  <div
                    onClick={toggleTheme}
                    style={{
                      width: 32,
                      height: 32,
                      borderRadius: 6,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      cursor: 'pointer',
                      border: '1px solid var(--border-color)',
                      background: 'var(--card-bg)',
                      transition: 'all 0.2s ease',
                    }}
                    title={isDarkMode ? t('switchToLight', '切换到浅色模式') : t('switchToDark', '切换到深色模式')}
                  >
                    {isDarkMode ? (
                      <SunOutlined style={{ fontSize: 16, color: '#faad14' }} />
                    ) : (
                      <MoonOutlined style={{ fontSize: 16, color: '#6B7280' }} />
                    )}
                  </div>
                </div>
                {/* 垂直分割线 */}
                <div style={{ width: 1, height: 24, background: 'var(--border-color)' }} />
                <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
                  <Space style={{ cursor: 'pointer', color: '#333' }}>
                    <UserOutlined />
                    <span>
                      {realName || username || t('user', '用户')}
                      {userRole && (
                        <span style={{ color: '#faad14', fontSize: 12, marginLeft: 4 }}>
                          ({getRoleDisplayName(userRole)})
                        </span>
                      )}
                    </span>
                  </Space>
                </Dropdown>
              </div>
            </Header>
          )}

          {/* 非全屏时显示标签栏 */}
          {!isFullscreen && (
            <div
              style={{
                background: 'var(--card-bg)',
                padding: '0 24px',
                borderBottom: '1px solid var(--border-color)',
              }}
            >
              {renderTabBar({})}
            </div>
          )}

          {/* 全屏时显示顶部栏 */}
          {isFullscreen && (
            <div
              style={{
                height: 40,
                background: '#001529',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '0 16px',
                position: 'sticky',
                top: 0,
                zIndex: 9,
              }}
            >
              <span style={{ color: '#fff', fontSize: 14 }}>
                {getPageConfig()[fullscreenTabKey]?.title || ''}
              </span>
              <button
                onClick={exitFullscreen}
                onKeyDown={(e) => e.key === 'Enter' && exitFullscreen()}
                tabIndex={0}
                type="button"
                style={{
                  background: 'transparent',
                  border: 'none',
                  color: '#fff',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 4,
                  outline: 'none',
                  padding: '4px 8px',
                  borderRadius: 4,
                }}
              >
                <FullscreenExitOutlined />
                <span>{t('exitFullscreen', '退出全屏')}</span>
              </button>
            </div>
          )}

          <Content
            style={{
              margin: isFullscreen ? 0 : 16,
              padding: isFullscreen ? 16 : 0,
              minHeight: isFullscreen ? 'calc(100vh - 40px)' : 'calc(100vh - 64px - 48px - 32px)',
              background: isFullscreen ? 'var(--content-bg)' : undefined,
            }}
          >
            <Outlet />
          <React.Suspense fallback={null}>
            <AIChatbot />
          </React.Suspense>
          </Content>
        </Layout>
      </Layout>

      {/* 搜索命令面板 */}
      <Modal
        open={searchOpen}
        onCancel={() => {
          setSearchOpen(false)
          setSearchValue('')
        }}
        footer={null}
        closable={false}
        width={480}
        style={{ top: 60 }}
        styles={{
          body: { padding: 0 },
          mask: { backgroundColor: 'rgba(0, 0, 0, 0.5)' },
        }}
      >
        <div style={{ padding: '12px 16px', borderBottom: '1px solid #F3F4F6' }}>
          <Input
            prefix={<SearchOutlined style={{ color: '#9CA3AF' }} />}
            placeholder={t('searchMenu', '搜索菜单...')}
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            autoFocus
            bordered={false}
            style={{ fontSize: 16 }}
          />
        </div>
        <div style={{ maxHeight: 400, overflow: 'auto', background: '#FFFFFF' }}>
          {filteredMenuItems.length > 0 ? (
            <div>
              {filteredMenuItems.map((item) => (
                <div
                  key={item.key}
                  onClick={() => handleSearchSelect(item.path)}
                  style={{
                    cursor: 'pointer',
                    padding: '10px 16px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 12,
                    textAlign: 'left',
                    background: '#FFFFFF',
                    transition: 'background 0.15s ease',
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.background = '#E8F4FF'}
                  onMouseLeave={(e) => e.currentTarget.style.background = '#FFFFFF'}
                >
                  <span style={{ color: '#6B7280', display: 'flex', alignItems: 'center' }}>{item.icon}</span>
                  <span style={{ color: '#111827', fontSize: 14 }}>{item.title}</span>
                </div>
              ))}
            </div>
          ) : (
            <div style={{ padding: 24, textAlign: 'center', color: '#9CA3AF' }}>
              {t('noMatchMenu', '未找到匹配菜单')}
            </div>
          )}
        </div>
      </Modal>
    </>
  )
}
