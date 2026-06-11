// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { Form, Input, Select, Button, Space } from 'antd'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

export interface SearchField {
  name: string
  label: string
  type?: 'text' | 'select'
  options?: { value: string | number; label: string }[]
}

interface SearchBarProps {
  fields: SearchField[]
  onSearch: (values: Record<string, string>) => void
  onReset: () => void
  style?: React.CSSProperties
}

export default function SearchBar({ fields, onSearch, onReset, style }: SearchBarProps) {
  const { t: tc } = useTranslation('common')
  const [form] = Form.useForm()

  // 默认字段
  // 直接使用传入的字段，不添加"全部"选项
  const defaultFields: SearchField[] = fields

  const handleValuesChange = (_changedValues: Record<string, unknown>, allValues: Record<string, string>) => {
    onSearch(allValues)
  }

  const handleReset = () => {
    form.resetFields()
    onReset()
  }

  return (
    <Form
      form={form}
      layout="inline"
      style={{ marginBottom: 16, ...style }}
      onValuesChange={handleValuesChange}
    >
      {defaultFields.map((field) => (
        <Form.Item key={field.name} name={field.name} style={{ marginBottom: 0 }}>
          {field.type === 'select' ? (
            <Select
              placeholder={field.label}
              allowClear
              style={{ width: 120 }}
              options={field.options}
            />
          ) : (
            <Input
              placeholder={field.label}
              allowClear
              style={{ width: 200 }}
            />
          )}
        </Form.Item>
      ))}
      <Form.Item style={{ marginBottom: 0 }}>
        <Space>
          <Button type="primary" icon={<SearchOutlined />} onClick={() => onSearch(form.getFieldsValue())}>
            {tc('search', '搜索')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            {tc('reset', '重置')}
          </Button>
        </Space>
      </Form.Item>
    </Form>
  )
}