// SPDX-License-Identifier: MIT
// Copyright (c) 2026 OPS Platform Contributors

import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'

import menuZh from './resources/zh-CN/menu.json'
import commonZh from './resources/zh-CN/common.json'
import dashboardZh from './resources/zh-CN/dashboard.json'
import cmdbZh from './resources/zh-CN/cmdb.json'
import deployZh from './resources/zh-CN/deploy.json'
import monitorZh from './resources/zh-CN/monitor.json'
import alarmZh from './resources/zh-CN/alarm.json'
import securityZh from './resources/zh-CN/security.json'
import adminZh from './resources/zh-CN/admin.json'
import platformZh from './resources/zh-CN/platform.json'
import loginZh from './resources/zh-CN/login.json'
import errorsZh from './resources/zh-CN/errors.json'
import chatbotZh from './resources/zh-CN/chatbot.json'

import menuEn from './resources/en-US/menu.json'
import commonEn from './resources/en-US/common.json'
import dashboardEn from './resources/en-US/dashboard.json'
import cmdbEn from './resources/en-US/cmdb.json'
import deployEn from './resources/en-US/deploy.json'
import monitorEn from './resources/en-US/monitor.json'
import alarmEn from './resources/en-US/alarm.json'
import securityEn from './resources/en-US/security.json'
import adminEn from './resources/en-US/admin.json'
import platformEn from './resources/en-US/platform.json'
import loginEn from './resources/en-US/login.json'
import errorsEn from './resources/en-US/errors.json'
import chatbotEn from './resources/en-US/chatbot.json'

const STORAGE_KEY = 'app_language'

function getStoredLang(): string {
  try {
    const lang = localStorage.getItem(STORAGE_KEY)
    if (lang && ['zh-CN', 'en-US'].includes(lang)) return lang
  } catch { /* noop */ }
  return 'zh-CN'
}

i18next.use(initReactI18next).init({
  resources: {
    'zh-CN': {
      menu: menuZh,
      common: commonZh,
      dashboard: dashboardZh,
      cmdb: cmdbZh,
      deploy: deployZh,
      monitor: monitorZh,
      alarm: alarmZh,
      security: securityZh,
      admin: adminZh,
      platform: platformZh,
      login: loginZh,
      errors: errorsZh,
      chatbot: chatbotZh,
    },
    'en-US': {
      menu: menuEn,
      common: commonEn,
      dashboard: dashboardEn,
      cmdb: cmdbEn,
      deploy: deployEn,
      monitor: monitorEn,
      alarm: alarmEn,
      security: securityEn,
      admin: adminEn,
      platform: platformEn,
      login: loginEn,
      errors: errorsEn,
      chatbot: chatbotEn,
    },
  },
  lng: getStoredLang(),
  fallbackLng: 'zh-CN',
  defaultNS: 'common',
  fallbackNS: 'common',
  interpolation: { escapeValue: false },
  returnNull: false,
  returnEmptyString: false,
})

export default i18next
