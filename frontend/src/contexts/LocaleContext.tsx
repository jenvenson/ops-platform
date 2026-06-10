// SPDX-License-Identifier: MIT
// Copyright (c) 2026 OPS Platform Contributors

import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'
import zhCN from 'antd/locale/zh_CN'
import enUS from 'antd/locale/en_US'

const LOCALE_MAP: Record<string, typeof zhCN> = {
  'zh-CN': zhCN,
  'en-US': enUS,
}

function getStoredLang(): string {
  try {
    const lang = localStorage.getItem('app_language')
    if (lang && lang in LOCALE_MAP) return lang
  } catch { /* noop */ }
  return 'zh-CN'
}

interface LocaleContextType {
  locale: typeof zhCN
  lang: string
  setLang: (lang: string) => void
}

const LocaleContext = createContext<LocaleContextType>({
  locale: zhCN,
  lang: 'zh-CN',
  setLang: () => {},
})

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<string>(getStoredLang)

  const setLang = useCallback((newLang: string) => {
    if (LOCALE_MAP[newLang]) {
      localStorage.setItem('app_language', newLang)
      setLangState(newLang)
    }
  }, [])

  return (
    <LocaleContext.Provider value={{ locale: LOCALE_MAP[lang] || zhCN, lang, setLang }}>
      {children}
    </LocaleContext.Provider>
  )
}

export function useLocale() {
  return useContext(LocaleContext)
}
