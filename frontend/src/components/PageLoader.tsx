// SPDX-License-Identifier: MIT
// Copyright (c) 2026 OPS Platform Contributors

import { useTranslation } from 'react-i18next'

export default function PageLoader() {
  const { t } = useTranslation('common')
  return <div style={{ padding: 24 }}>{t('loading')}</div>
}
