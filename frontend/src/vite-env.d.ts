// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}