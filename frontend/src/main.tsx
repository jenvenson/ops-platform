// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import 'antd/dist/reset.css'
import './styles/theme-variables.css'
import './styles/theme.css'
import './styles/menu-overrides.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)