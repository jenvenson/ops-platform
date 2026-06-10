// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useEffect } from 'react'

export default function AlarmLayout() {
  const location = useLocation()
  const navigate = useNavigate()

  // 访问 /alarm 时自动跳转到告警中心默认页
  useEffect(() => {
    if (location.pathname === '/alarm') {
      navigate('/alarm/events', { replace: true })
    }
  }, [location.pathname, navigate])

  return <Outlet />
}