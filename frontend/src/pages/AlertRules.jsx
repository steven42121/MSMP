import React, { useEffect, useState, useCallback } from 'react';
import { Table, Button, Modal, Form, Input, InputNumber, Select, Tag, Space, Popconfirm, Switch, message, Card, Typography } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';

const { Text } = Typography;

export default function AlertRules() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await client.get()('/alert-rules');
      setData(Array.isArray(resp) ? resp : resp.data || []);
    } catch (e) {
      setData([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      await client.post()('/alert-rules', values);
      message.success('规则已创建');
      setCreateOpen(false);
      form.resetFields();
      load();
    } catch (e) {
      if (e.errorFields) return;
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggle = async (id, enabled) => {
    try {
      await client.put()(`/alert-rules/${id}`, { enabled });
      load();
    } catch (e) {}
  };

  const handleDelete = async (id) => {
    try {
      await client.delete()(`/alert-rules/${id}`);
      message.success('已删除');
      load();
    } catch (e) {}
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '指标', dataIndex: 'metric', key: 'metric', render: (v) => <Tag>{v}</Tag> },
    { title: '运算符', dataIndex: 'operator', key: 'operator' },
    { title: '阈值', dataIndex: 'threshold', key: 'threshold', render: (v) => `${v}%` },
    {
      title: '级别', dataIndex: 'level', key: 'level',
      render: (l) => <Tag color={l === 'critical' ? 'red' : 'orange'}>{l}</Tag>,
    },
    {
      title: '启用', dataIndex: 'enabled', key: 'enabled',
      render: (v, r) => <Switch checked={v} onChange={(checked) => handleToggle(r.id, checked)} />,
    },
    {
      title: '创建时间', dataIndex: 'created_at', key: 'created_at',
      render: (t) => dayjs(t).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作', key: 'action',
      render: (_, r) => (
        <Popconfirm title="确认删除该规则？" onConfirm={() => handleDelete(r.id)}>
          <Button type="link" danger>删除</Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 20 }}>
        <div>
          <div className="page-title">告警规则</div>
          <Text type="secondary" style={{ fontSize: 13 }}>配置性能指标的告警阈值规则</Text>
        </div>
      </div>
      <Card className="liquid-glass" style={{ marginBottom: 16, borderRadius: 16, border: 'none' }}>
        <Space wrap className="filter-bar">
          <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)} style={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none' }}>新建规则</Button>
        </Space>
      </Card>
      <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}>
        <Table columns={columns} dataSource={data} rowKey="id" loading={loading} scroll={{ x: 'max-content' }} />
      </Card>
      <Modal
        title="新建告警规则"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={handleCreate}
        confirmLoading={submitting}
        okText="创建"
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input placeholder="例如：CPU 过高" />
          </Form.Item>
          <Form.Item name="metric" label="指标" rules={[{ required: true }]}>
            <Select options={[
              { value: 'cpu', label: 'CPU 使用率' },
              { value: 'mem', label: '内存使用率' },
              { value: 'disk', label: '磁盘使用率' },
            ]} />
          </Form.Item>
          <Form.Item name="operator" label="运算符" rules={[{ required: true }]}>
            <Select options={[
              { value: 'gt', label: '大于 (>)' },
              { value: 'gte', label: '大于等于 (>=)' },
              { value: 'lt', label: '小于 (<)' },
              { value: 'lte', label: '小于等于 (<=)' },
            ]} />
          </Form.Item>
          <Form.Item name="threshold" label="阈值(%)" rules={[{ required: true }]}>
            <InputNumber min={0} max={100} style={{ width: 200 }} />
          </Form.Item>
          <Form.Item name="level" label="级别" initialValue="warning">
            <Select options={[
              { value: 'warning', label: '警告' },
              { value: 'critical', label: '严重' },
            ]} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" initialValue={true} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
