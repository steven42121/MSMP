import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Button, Modal, Form, Input, Space, message, Typography } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';

const { Text } = Typography;

export default function Tenants() {
  const [tenants, setTenants] = useState([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await client.get()('/tenants');
      setTenants(Array.isArray(resp) ? resp : resp.data || []);
    } catch (e) {
      setTenants([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      await client.post()('/tenants', values);
      message.success('租户已创建');
      setCreateOpen(false);
      form.resetFields();
      load();
    } catch (e) {
      if (e.errorFields) return;
    } finally {
      setSubmitting(false);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '标识', dataIndex: 'slug', key: 'slug' },
    {
      title: '创建时间', dataIndex: 'created_at', key: 'created_at',
      render: (t) => dayjs(t).format('YYYY-MM-DD HH:mm:ss'),
    },
  ];

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 20 }}>
        <div>
          <div className="page-title">租户管理</div>
          <Text type="secondary" style={{ fontSize: 13 }}>多租户隔离与资源管理</Text>
        </div>
      </div>
      <Card className="liquid-glass" style={{ marginBottom: 16, borderRadius: 16, border: 'none' }}>
        <Space wrap className="filter-bar">
          <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)} style={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none' }}>新建租户</Button>
        </Space>
      </Card>
      <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}>
        <Table columns={columns} dataSource={tenants} rowKey="id" loading={loading} />
      </Card>
      <Modal
        title="新建租户"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={handleCreate}
        confirmLoading={submitting}
        okText="创建"
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="slug" label="标识" rules={[{ required: true }]}>
            <Input placeholder="例如：acme" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
