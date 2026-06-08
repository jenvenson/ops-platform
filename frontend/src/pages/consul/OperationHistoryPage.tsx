import { useState, useEffect } from 'react'
import {
  Card, Table, Button, Space, Tag, Modal, Descriptions, message, Popconfirm
} from 'antd'
import {
  EyeOutlined, ClockCircleOutlined, DeleteOutlined
} from '@ant-design/icons'
import { consulAPI, CopyOperation } from '../../api/consul'
import dayjs from 'dayjs'

export default function OperationHistoryPage() {
  const [operations, setOperations] = useState<CopyOperation[]>([])
  const [loading, setLoading] = useState(false)
  const [detailModalVisible, setDetailModalVisible] = useState(false)
  const [selectedOperation, setSelectedOperation] = useState<CopyOperation | null>(null)

  const fetchOperations = async () => {
    setLoading(true)
    try {
      const data = await consulAPI.getOperations({ limit: 50 })
      setOperations(data)
    } catch (error) {
      message.error('获取操作历史失败')
    } finally {
      setLoading(false)
    }
  }

  const showOperationDetails = (operation: CopyOperation) => {
    setSelectedOperation(operation)
    setDetailModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await consulAPI.deleteOperation(id)
      message.success('删除成功')
      fetchOperations()
    } catch (error) {
      message.error('删除失败')
    }
  }

  const getStatusInfo = (record: CopyOperation) => {
    if (record.status === 'failed') {
      return { color: 'red', text: '失败' }
    }
    if (record.created_at) {
      return { color: 'green', text: '成功' }
    }
    return { color: 'orange', text: '待处理' }
  }

  useEffect(() => {
    fetchOperations()
  }, [])

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: '源键',
      dataIndex: 'source_key',
      key: 'source_key',
      ellipsis: true,
    },
    {
      title: '目标键',
      dataIndex: 'target_key',
      key: 'target_key',
      ellipsis: true,
    },
    {
      title: '状态',
      key: 'status',
      width: 80,
      render: (_: any, record: CopyOperation) => {
        const info = getStatusInfo(record)
        return <Tag color={info.color}>{info.text}</Tag>
      },
    },
    {
      title: '操作人',
      dataIndex: 'operator',
      key: 'operator',
      width: 100,
      render: (val: string) => val || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (date: string) => date ? dayjs(date).format('YYYY-MM-DD HH:mm:ss') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 140,
      render: (_: any, record: CopyOperation) => (
        <Space>
          <Button
            size="small"
            icon={<EyeOutlined />}
            onClick={() => showOperationDetails(record)}
          >
            详情
          </Button>
          <Popconfirm
            title="确定删除此条记录？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button
              size="small"
              danger
              icon={<DeleteOutlined />}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: 24 }}>
      <Card
        title={
          <Space>
            <ClockCircleOutlined />
            <span>配置操作记录</span>
          </Space>
        }
        size="small"
        extra={
          <Button onClick={fetchOperations}>
            刷新
          </Button>
        }
      >
        <Table
          dataSource={operations}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 20 }}
          size="small"
        />
      </Card>

      <Modal
        title="操作详情"
        open={detailModalVisible}
        onCancel={() => {
          setDetailModalVisible(false)
          setSelectedOperation(null)
        }}
        footer={null}
        width={800}
      >
        {selectedOperation && (
          <Descriptions bordered column={1}>
            <Descriptions.Item label="ID">{selectedOperation.id}</Descriptions.Item>
            <Descriptions.Item label="源键">{selectedOperation.source_key}</Descriptions.Item>
            <Descriptions.Item label="目标键">{selectedOperation.target_key}</Descriptions.Item>
            <Descriptions.Item label="应用的规则">{selectedOperation.rules_applied || '无'}</Descriptions.Item>
            <Descriptions.Item label="状态">
              {(() => {
                const info = getStatusInfo(selectedOperation)
                return <Tag color={info.color}>{info.text}</Tag>
              })()}
            </Descriptions.Item>
            <Descriptions.Item label="消息">{selectedOperation.message || '无'}</Descriptions.Item>
            <Descriptions.Item label="操作人">{selectedOperation.operator || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {selectedOperation.created_at ? dayjs(selectedOperation.created_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  )
}
