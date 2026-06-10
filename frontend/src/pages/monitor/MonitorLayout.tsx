// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useEffect } from 'react'

export default function MonitorLayout() {
  const location = useLocation()
  const navigate = useNavigate()

  // 访问 /monitor 时自动跳转到监控大屏
  useEffect(() => {
    if (location.pathname === '/monitor') {
      navigate('/monitor/bigscreen', { replace: true })
    }
  }, [location.pathname, navigate])

  return <Outlet />
}