import React, { useEffect, useState } from 'react';
import { Table, Tag, Button, Space, Modal, Form, Input, Select, InputNumber, Popconfirm, message, Typography } from 'antd';
import { PlusOutlined, ReloadOutlined, PlayCircleOutlined, DeleteOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';

const { Text } = Typography;

export default function Probes() {
  const [probes, setProbes] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [runningId, setRunningId] = useState(null);

  const load = async () => {
    setLoading(true);
    try {
      const resp = await client.get()('/probes');
      setProbes(Array.isArray(resp) ? resp : []);
    } catch (e) { message.error('加载失败'); }
    finally { setLoading(false); }
  };

  useEffect(() => { load(); }, []);

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      await client.post()('/probes', values);
      message.success('已创建');
      form.resetFields();
      setModalOpen(false);
      load();
    } catch (e) { if (e.errorFields) return; message.error('创建失败'); }
  };

  const handleDelete = async (id) => {
    try {
      await client.delete()(`/probes/${id}`);
      message.success('已删除');
      load();
    } catch (e) { message.error('删除失败'); }
  };

  const handleRun = async (id) => {
    setRunningId(id);
    try {
      const resp = await client.post()(`/probes/${id}/run`);
      if (resp.up) message.success(`${resp.latency_ms}ms`);
      else message.error(resp.error || 'down');
    } catch (e) { message.error('探测失败'); }
    finally { setRunningId(null); load(); }
  };

  const typeColor = { http: 'blue', https: 'blue', tcp: 'cyan', ssl: 'purple' };

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    {
      title: '类型', dataIndex: 'type', key: 'type', width: 90,
      render: (v) => <Tag color={typeColor[v] || 'default'}>{v}</Tag>,
    },
    { title: '目标', dataIndex: 'target', key: 'target', ellipsis: true },
    {
      title: '状态', dataIndex: 'last_status', key: 'last_status', width: 90,
      render: (v) => v === 'up' ? <Tag color="green">Up</Tag> : v === 'down' ? <Tag color="red">Down</Tag> : <Tag>-</Tag>,
    },
    { title: '延迟', dataIndex: 'last_latency_ms', key: 'latency', width: 100, render: (v) => v ? `${v}ms` : '-' },
    { title: '间隔', dataIndex: 'interval_sec', key: 'interval', width: 80, render: (v) => `${v}s` },
    {
      title: '启用', dataIndex: 'enabled', key: 'enabled', width: 70,
      render: (v) => v ? <Tag color="green">是</Tag> : <Tag color="default">否</Tag>,
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 160, render: (t) => dayjs(t).format('MM-DD HH:mm') },
    {
      title: '操作', key: 'action', width: 160,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<PlayCircleOutlined />} loading={runningId === r.id} onClick={() => handleRun(r.id)}>探测</Button>
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" danger size="small" icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 16 }}>
        <div className="page-title">可用性探测</div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>新建探测</Button>
        </Space>
      </div>
      <Table columns={columns} dataSource={probes} rowKey="id" loading={loading} size="small" pagination={{ pageSize: 20 }} />

      <Modal title="新建可用性探测" open={modalOpen} onOk={handleCreate} onCancel={() => { setModalOpen(false); form.resetFields(); }} width={500}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input placeholder="如：Google DNS" />
          </Form.Item>
          <Form.Item name="type" label="类型" rules={[{ required: true }]} initialValue="http">
            <Select options={[
              { value: 'http', label: 'HTTP' },
              { value: 'https', label: 'HTTPS' },
              { value: 'tcp', label: 'TCP 端口' },
              { value: 'ssl', label: 'SSL 证书' },
            ]} />
          </Form.Item>
          <Form.Item name="target" label="目标" rules={[{ required: true }]}>
            <Input placeholder="https://example.com 或 8.8.8.8:53" />
          </Form.Item>
          <Form.Item name="interval_sec" label="探测间隔（秒）" rules={[{ required: true }]} initialValue={60}>
            <InputNumber min={10} max={3600} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="timeout_sec" label="超时（秒）" initialValue={10}>
            <InputNumber min={1} max={60} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
