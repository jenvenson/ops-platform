import apiClient from './client'

export interface User {
  id: number
  username: string
  real_name?: string
  email?: string
  role: 'admin' | 'user'
  created_at: string
  updated_at: string
}

export interface Role {
  id: number
  name: string
  code: string
  description?: string
  status: number
  created_at: string
  updated_at: string
}

export interface Menu {
  id: number
  title: string
  key: string
  path?: string
  icon?: string
  parent_id: number
  sort: number
  status: number
  created_at: string
  updated_at: string
}

export interface CreateUserRequest {
  username: string
  password: string
  real_name?: string
  email?: string
  role?: 'admin' | 'user'
}

export interface UpdateUserRequest {
  username?: string
  password?: string
  real_name?: string
  email?: string
  role?: 'admin' | 'user'
}

export interface CreateRoleRequest {
  name: string
  code: string
  description?: string
  status?: number
}

export interface UpdateRoleRequest {
  name?: string
  code?: string
  description?: string
  status?: number
}

export interface CreateMenuRequest {
  title: string
  key: string
  path?: string
  icon?: string
  parent_id?: number
  sort?: number
  status?: number
}

export interface UpdateMenuRequest {
  title?: string
  key?: string
  path?: string
  icon?: string
  parent_id?: number
  sort?: number
  status?: number
}

export interface ChangePasswordRequest {
  old_password: string
  new_password: string
}

export interface UpdateProfileRequest {
  real_name?: string
  email?: string
}

export interface FIMSSHSetting {
  auth_mode: 'password' | 'private_key'
  ssh_user: string
  timeout_sec: number
  password_configured: boolean
  private_key_configured: boolean
}

export interface UpdateFIMSSHSettingRequest {
  auth_mode: 'password' | 'private_key'
  ssh_user: string
  password?: string
  private_key?: string
  timeout_sec: number
}

export interface TestFIMSSHConnectionRequest {
  host: string
  port?: number
}

export interface TestFIMSSHConnectionResponse {
  success: boolean
  message: string
  output?: string
}

export interface AuditLogSetting {
  access_log_enabled: boolean
  operation_log_enabled: boolean
  login_log_enabled: boolean
}

export interface AssistantModelSetting {
  provider: string
  enabled: boolean
  api_key_configured: boolean
  base_url: string
  chat_model: string
  embed_model: string
  temperature: number
  timeout_sec: number
}

export interface UpdateAssistantModelSettingRequest {
  provider: string
  enabled: boolean
  api_key?: string
  base_url?: string
  chat_model?: string
  embed_model?: string
  temperature?: number
  timeout_sec?: number
}

export interface SystemGeneralSetting {
  site_name: string
  timezone: string
  language: string
}

export interface LicenseStatus {
  customer: string
  expires_at: string
  features: string[]
  valid: boolean
  valid_error?: string
}

export const adminAPI = {
  // 用户管理
  getUsers: (): Promise<User[]> =>
    apiClient.get<User[]>('/admin/users'),

  createUser: (data: CreateUserRequest): Promise<User> =>
    apiClient.post<User>('/admin/users', data),

  updateUser: (id: number, data: UpdateUserRequest): Promise<User> =>
    apiClient.put<User>(`/admin/users/${id}`, data),

  deleteUser: (id: number): Promise<{ message: string }> =>
    apiClient.delete<{ message: string }>(`/admin/users/${id}`),

  // 管理员重置用户密码
  resetUserPassword: (id: number, password: string): Promise<User> =>
    apiClient.put<User>(`/admin/users/${id}`, { password }),

  // 当前用户相关
  getCurrentUser: (): Promise<User> =>
    apiClient.get<User>('/user/me'),

  changePassword: (data: ChangePasswordRequest): Promise<{ message: string }> =>
    apiClient.put<{ message: string }>('/user/password', data),

  updateProfile: (data: UpdateProfileRequest): Promise<User> =>
    apiClient.put<User>('/user/profile', data),

  // 角色管理
  getRoles: (): Promise<Role[]> =>
    apiClient.get<Role[]>('/admin/roles'),

  createRole: (data: CreateRoleRequest): Promise<Role> =>
    apiClient.post<Role>('/admin/roles', data),

  updateRole: (id: number, data: UpdateRoleRequest): Promise<Role> =>
    apiClient.put<Role>(`/admin/roles/${id}`, data),

  deleteRole: (id: number): Promise<{ message: string }> =>
    apiClient.delete<{ message: string }>(`/admin/roles/${id}`),

  // 角色菜单关联
  getRoleMenus: (roleId: number): Promise<{ menu_ids: number[]; menus: Menu[] }> =>
    apiClient.get<{ menu_ids: number[]; menus: Menu[] }>(`/admin/roles/${roleId}/menus`),

  updateRoleMenus: (roleId: number, menuIds: number[]): Promise<{ message: string }> =>
    apiClient.put<{ message: string }>(`/admin/roles/${roleId}/menus`, { menu_ids: menuIds }),

  // 菜单管理
  getMenus: (): Promise<Menu[]> =>
    apiClient.get<Menu[]>('/admin/menus'),

  createMenu: (data: CreateMenuRequest): Promise<Menu> =>
    apiClient.post<Menu>('/admin/menus', data),

  updateMenu: (id: number, data: UpdateMenuRequest): Promise<Menu> =>
    apiClient.put<Menu>(`/admin/menus/${id}`, data),

  deleteMenu: (id: number): Promise<{ message: string }> =>
    apiClient.delete<{ message: string }>(`/admin/menus/${id}`),

  getAuditLogSetting: (): Promise<AuditLogSetting> =>
    apiClient.get<AuditLogSetting>('/admin/settings/audit-logs'),

  updateAuditLogSetting: (data: AuditLogSetting): Promise<AuditLogSetting> =>
    apiClient.put<AuditLogSetting>('/admin/settings/audit-logs', data),

  getFIMSSHSetting: (): Promise<FIMSSHSetting> =>
    apiClient.get<FIMSSHSetting>('/admin/settings/fim-ssh'),

  updateFIMSSHSetting: (data: UpdateFIMSSHSettingRequest): Promise<FIMSSHSetting> =>
    apiClient.put<FIMSSHSetting>('/admin/settings/fim-ssh', data),

  testFIMSSHConnection: (data: TestFIMSSHConnectionRequest): Promise<TestFIMSSHConnectionResponse> =>
    apiClient.post<TestFIMSSHConnectionResponse>('/admin/settings/fim-ssh/test', data),

  getAssistantModelSetting: (): Promise<AssistantModelSetting> =>
    apiClient.get<AssistantModelSetting>('/admin/settings/assistant-model'),

  updateAssistantModelSetting: (data: UpdateAssistantModelSettingRequest): Promise<{ message: string }> =>
    apiClient.put<{ message: string }>('/admin/settings/assistant-model', data),

  // 系统通用配置
  getSystemGeneralSetting: (): Promise<SystemGeneralSetting> =>
    apiClient.get<SystemGeneralSetting>('/admin/settings/general'),

  updateSystemGeneralSetting: (data: SystemGeneralSetting): Promise<SystemGeneralSetting> =>
    apiClient.put<SystemGeneralSetting>('/admin/settings/general', data),

  // License
  getLicenseStatus: (): Promise<LicenseStatus> =>
    apiClient.get<LicenseStatus>('/admin/settings/license'),
}
