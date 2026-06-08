export interface StoredMenuItem {
  key: string
  path?: string
  title?: string
  icon?: string
  children?: StoredMenuItem[]
}

export interface StoredUserInfo {
  username: string
  real_name?: string
  role?: string
}

export const MENU_UPDATED_EVENT = 'menuUpdated'
export const MENU_CHANGED_EVENT = 'menuChanged'

const normalizeMenuItem = (value: unknown): StoredMenuItem | null => {
  if (!value || typeof value !== 'object') {
    return null
  }

  const item = value as Record<string, unknown>
  const children = Array.isArray(item.children)
    ? item.children.map(normalizeMenuItem).filter((child): child is StoredMenuItem => child !== null)
    : undefined

  return {
    key: typeof item.key === 'string' ? item.key : '',
    path: typeof item.path === 'string' ? item.path : undefined,
    title: typeof item.title === 'string' ? item.title : undefined,
    icon: typeof item.icon === 'string' ? item.icon : undefined,
    children,
  }
}

export const readStoredMenus = (): StoredMenuItem[] => {
  if (typeof window === 'undefined') {
    return []
  }

  try {
    const savedMenus = localStorage.getItem('user_menus')
    if (!savedMenus) {
      return []
    }
    const parsed = JSON.parse(savedMenus)
    if (!Array.isArray(parsed)) {
      console.warn('菜单数据格式无效，已忽略本地缓存')
      return []
    }
    return parsed
      .map(normalizeMenuItem)
      .filter((item): item is StoredMenuItem => item !== null)
  } catch (error) {
    console.warn('解析菜单数据失败:', error)
    return []
  }
}

export const collectMenuPaths = (menus: StoredMenuItem[]): Set<string> => {
  const paths = new Set<string>()

  const walk = (items: StoredMenuItem[]) => {
    items.forEach((item) => {
      if (item.path) {
        paths.add(item.path)
      }
      if (item.children?.length) {
        walk(item.children)
      }
    })
  }

  walk(menus)
  return paths
}

export const readAllowedPaths = (): Set<string> => collectMenuPaths(readStoredMenus())

export const hasMenuAccess = (path: string, allowedPaths?: Set<string>): boolean => {
  const paths = allowedPaths || readAllowedPaths()
  return paths.has(path)
}

export const notifyMenusChanged = (): void => {
  if (typeof window === 'undefined') {
    return
  }
  window.dispatchEvent(new CustomEvent(MENU_CHANGED_EVENT))
}

export const readStoredUserInfo = (): StoredUserInfo | null => {
  if (typeof window === 'undefined') {
    return null
  }

  try {
    const savedUserInfo = localStorage.getItem('user_info')
    if (!savedUserInfo) {
      return null
    }

    const parsed = JSON.parse(savedUserInfo)
    if (!parsed || typeof parsed !== 'object') {
      console.warn('用户信息格式无效，已忽略本地缓存')
      return null
    }

    const userInfo = parsed as Record<string, unknown>
    return {
      username: typeof userInfo.username === 'string' ? userInfo.username : '',
      real_name: typeof userInfo.real_name === 'string' ? userInfo.real_name : undefined,
      role: typeof userInfo.role === 'string' ? userInfo.role : undefined,
    }
  } catch (error) {
    console.warn('解析用户信息失败:', error)
    return null
  }
}

// 权限检查：是否为管理员角色
export const isAdmin = (): boolean => {
  const userInfo = readStoredUserInfo()
  if (!userInfo?.role) return false
  // 管理员角色可以是 admin 或包含 admin 关键字的角色
  return userInfo.role.toLowerCase() === 'admin' || userInfo.role.toLowerCase().includes('administrator')
}

// 权限检查：是否可以编辑（编辑/删除/配置）- 需要管理员角色
export const canEdit = (): boolean => {
  return isAdmin()
}
