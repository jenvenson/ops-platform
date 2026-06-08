import { useEffect } from 'react'

type AssistantPageContextInput = {
  objectType?: string
  objectId?: string | number
  selectedRecordIds?: Array<string | number | null | undefined>
  filters?: Record<string, string | number | boolean | null | undefined>
}

const normalizeText = (value: string | number | boolean | null | undefined) => {
  if (value === null || value === undefined) {
    return ''
  }
  return String(value).trim()
}

const normalizeFilters = (filters?: AssistantPageContextInput['filters']) => {
  const result: Record<string, string> = {}
  if (!filters) {
    return result
  }

  Object.entries(filters).forEach(([key, value]) => {
    const normalizedKey = normalizeText(key)
    const normalizedValue = normalizeText(value)
    if (!normalizedKey || !normalizedValue) {
      return
    }
    result[normalizedKey] = normalizedValue
  })

  return result
}

export default function useAssistantPageContext(input: AssistantPageContextInput) {
  const objectType = normalizeText(input.objectType)
  const objectId = normalizeText(input.objectId)
  const selectedRecordIds = (input.selectedRecordIds || [])
    .map((value) => normalizeText(value))
    .filter(Boolean)
  const filters = normalizeFilters(input.filters)
  const filtersJSON = Object.keys(filters).length > 0 ? JSON.stringify(filters) : ''

  useEffect(() => {
    const body = document?.body
    if (!body) {
      return undefined
    }

    if (objectType) {
      body.dataset.assistantObjectType = objectType
    } else {
      delete body.dataset.assistantObjectType
    }

    if (objectId) {
      body.dataset.assistantObjectId = objectId
    } else {
      delete body.dataset.assistantObjectId
    }

    if (selectedRecordIds.length > 0) {
      body.dataset.assistantRecordId = selectedRecordIds[0]
    } else {
      delete body.dataset.assistantRecordId
    }

    if (filtersJSON) {
      body.dataset.assistantFilters = filtersJSON
    } else {
      delete body.dataset.assistantFilters
    }

    return () => {
      delete body.dataset.assistantObjectType
      delete body.dataset.assistantObjectId
      delete body.dataset.assistantRecordId
      delete body.dataset.assistantFilters
    }
  }, [filtersJSON, objectId, objectType, selectedRecordIds.join(',')])
}
