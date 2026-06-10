// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import apiClient from './client'

export interface PlatformEventListItem {
  id: number
  event_id: string
  event_type: string
  event_category: string
  source_system: string
  source_table: string
  source_id: string
  object_type: string
  object_id: string
  title: string
  summary: string
  status: string
  severity: string
  operator_id: string
  operator_name: string
  trigger_mode: string
  started_at?: string
  finished_at?: string
  occurred_at: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface PlatformEventListParams {
  page?: number
  limit?: number
  q?: string
  event_category?: string
  source_system?: string
  event_type?: string
  object_type?: string
  object_id?: string
  status?: string
  severity?: string
  operator?: string
  trigger_mode?: string
  occurred_from?: string
  occurred_to?: string
}

export interface PlatformEventListResponse {
  data: PlatformEventListItem[]
  page: number
  limit: number
  total: number
}

export interface PlatformTimelineResponse {
  data: PlatformEventListItem[]
  object_type: string
  object_id: string
  limit: number
}

export const platformEventsAPI = {
  getEvents: (params: PlatformEventListParams) => {
    return apiClient.get<PlatformEventListResponse>('/platform/events', { params })
  },
  getTimeline: (params: { object_type: string; object_id: string; limit?: number }) => {
    return apiClient.get<PlatformTimelineResponse>('/platform/timeline', { params })
  },
}