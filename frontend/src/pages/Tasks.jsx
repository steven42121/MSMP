import React, { useEffect, useState, useCallback } from 'react';
import { Table, Button, Modal, Form, Input, Select, Tag, Space, InputNumber, message } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import client from '../api/client';

const statusColor = {
  pending: 'default',
  running: 'processing',
  success: 'success',
  failed: 'error',
  canceled: 'warning',
};

export default function Tasks() {
  const navigate = useNavigate();
  const [data, setData] = useState([]);
  const [hosts, setHosts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [status, setStatus] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();

  const loadHosts = async () => {
    try {
      const resp = await client.get('/hosts', { params: { page_size: 100 } });
      setHosts(resp.data || []);
    } catch (e) {}
  };

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await client.get('/tasks', { params: { page, page_size: pageSize, status } });
      setData(resp.data || []);
      setTotal(resp.total || 0);
    } catch (e) {
      setData([]);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, status]);

  useEffect(() => { loadHosts(); }, []);
  useEffect(() => { load(); }, [load]);

  // 有待执行/执行中任务时自动刷新
  useEffect(() => {
    const hasActive = data.some((t) => t.status === 'pending' || t.status === 'running');
    if (!hasActive) return;
    const timer = setInterval(load, 5000);
    return () => clearInterval(timer);
  }, [data, load]);

  const hostMap = {};
  hosts.forEach((h) => { hostMap[h.id] = h; });

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      await client.post('/tasks', values);
      message.success('任务已创建');
      setModalOpen(false);
      form.resetFields();
      setPage(1);
      load();
    } catch (e) {
      if (e.errorFields) return;
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = async (id) => {
    try {
      await client.put(`/tasks/${id}`, { status: 'canceled' });
      message.success('已取消');
      load();
    } catch (e) {}
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
    {
      title: '主机', dataIndex: 'host_id', key: 'host',
      render: (id) => hostMap[id]?.hostname || `#${id}`,
    },
    { title: '类型', dataIndex: 'type', key: 'type', render: (v) => <Tag>{v}</Tag> },
    { title: '命令', dataIndex: 'command', key: 'command', ellipsis: true },
    {
      title: '状态', dataIndex: 'status', key: 'status',
      render: (s) => <Tag color={statusColor[s] || 'default'}>{s}</Tag>,
    },
    {
      title: '创建时间', dataIndex: 'created_at', key: 'created_at',
      render: (t) => dayjs(t).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作', key: 'action',
      render: (_, r) => (
        <Space>
          <Button type="link" onClick={() => navigate(`/tasks/${r.id}`)}>详情</Button>
          {(r.status === 'pending') && (
            <Button type="link" danger onClick={() => handleCancel(r.id)}>取消</Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <h2>任务管理</h2>
      <Space style={{ marginBottom: 16 }} wrap>
        <Select
          placeholder="状态筛选"
          allowClear
          style={{ width: 140 }}
          value={status || undefined}
          onChange={(v) => { setStatus(v || ''); setPage(1); }}
          options={[
            { value: 'pending', label: '待执行' },
            { value: 'running', label: '执行中' },
            { value: 'success', label: '成功' },
            { value: 'failed', label: '失败' },
            { value: 'canceled', label: '已取消' },
          ]}
        />
        <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>新建任务</Button>
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
      <Modal
        title="新建任务"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleCreate}
        confirmLoading={submitting}
        okText="创建"
      >
        <Form form={form} layout="vertical">
          <Form.Item name="host_uuids" label="目标主机" rules={[{ required: true, message: '请选择主机' }]}>
            <Select
              showSearch
              mode="multiple"
              placeholder="可选择多个主机"
              optionFilterProp="label"
              options={hosts.map((h) => ({ value: h.uuid, label: `${h.hostname} (${h.ip})` }))}
            />
          </Form.Item>
          <Form.Item name="type" label="类型" initialValue="shell" rules={[{ required: true }]}>
            <Select options={[
              { value: 'shell', label: 'Shell 命令' },
              { value: 'restart', label: '重启' },
              { value: 'upgrade', label: '升级 Agent' },
            ]} />
          </Form.Item>
          <Form.Item name="command" label="命令" rules={[{ required: true, message: '请输入命令' }]}>
            <Input.TextArea rows={4} placeholder="例如：ls -la /tmp" />
          </Form.Item>
          <Form.Item name="timeout_sec" label="超时（秒）" initialValue={0} extra="0 表示不限制">
            <InputNumber min={0} max={86400} style={{ width: 200 }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
