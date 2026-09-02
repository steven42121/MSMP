import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Tag, Space, Select, Button } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';

export default function AuditLogs() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [action, setAction] = useState('');
  const [keyword, setKeyword] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await client.get()('/audit-logs', { params: { page, page_size: pageSize, action } });
      setData(resp.data || []);
      setTotal(resp.total || 0);
    } catch (e) {
      setData([]);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, action]);

  useEffect(() => { load(); }, [load]);

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
    { title: '用户', dataIndex: 'username', key: 'username' },
    { title: '动作', dataIndex: 'action', key: 'action', render: (v) => <Tag>{v}</Tag> },
    { title: '资源', dataIndex: 'resource', key: 'resource', ellipsis: true },
    { title: '方法', dataIndex: 'method', key: 'method' },
    {
      title: '状态', dataIndex: 'status', key: 'status',
      render: (s) => <Tag color={s < 300 ? 'green' : s < 400 ? 'blue' : 'red'}>{s}</Tag>,
    },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    {
      title: '时间', dataIndex: 'created_at', key: 'created_at',
      render: (t) => dayjs(t).format('YYYY-MM-DD HH:mm:ss'),
    },
  ];

  return (
    <div>
      <h2>审计日志</h2>
      <Space style={{ marginBottom: 16 }}>
        <Select
          placeholder="动作筛选"
          allowClear
          style={{ width: 160 }}
          value={action || undefined}
          onChange={(v) => { setAction(v || ''); setPage(1); }}
          options={[
            { value: 'manage', label: '管理操作' },
          ]}
        />
        <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
      </Space>
      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        scroll={{ x: 'max-content' }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          onChange: (p, ps) => { setPage(p); setPageSize(ps); },
        }}
      />
    </div>
  );
}
