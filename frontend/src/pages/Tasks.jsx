import React, { useEffect, useState, useCallback } from 'react';
import { Table, Button, Modal, Form, Input, Select, Tag, Space, InputNumber, message, Typography, Card } from 'antd';
import { PlusOutlined, ReloadOutlined, DesktopOutlined, CheckCircleOutlined, CloseCircleOutlined, ClockCircleOutlined, FileTextOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import client from '../api/client';

const { Text } = Typography;

const statusColor = {
  pending: 'default',
  running: 'processing',
  success: 'success',
  failed: 'error',
  canceled: 'warning',
};

const statusIcon = {
  pending: <ClockCircleOutlined />,
  running: <ReloadOutlined spin />,
  success: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
  failed: <CloseCircleOutlined style={{ color: '#ff4d4f' }} />,
  canceled: <ClockCircleOutlined style={{ color: '#faad14' }} />,
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
    { title: 'ID', dataIndex: 'id', key: 'id', width: 70, render: (v) => <Text type="secondary">{v}</Text> },
    {
      title: '主机', dataIndex: 'host_id', key: 'host', width: 160,
      render: (id) => {
        const h = hostMap[id];
        return h ? <Space><DesktopOutlined style={{ color: '#667eea' }} /><span style={{ fontWeight: 500 }}>{h.hostname}</span></Space> : <Text type="secondary">#{id}</Text>;
      },
    },
    {
      title: '类型', dataIndex: 'type', key: 'type', width: 100,
      render: (v) => <Tag style={{ borderRadius: 4, padding: '1px 8px' }}>{v}</Tag>,
    },
    { title: '命令', dataIndex: 'command', key: 'command', ellipsis: true, render: (v) => <Text code style={{ fontSize: 12 }}>{v}</Text> },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (s) => (
        <Space>
          {statusIcon[s] || statusIcon.pending}
          <Tag color={statusColor[s] || 'default'} style={{ borderRadius: 4 }}>{s}</Tag>
        </Space>
      ),
    },
    {
      title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170,
      render: (t) => (
        <Space direction="vertical" size={0}>
          <Text style={{ fontSize: 12 }}>{dayjs(t).format('YYYY-MM-DD')}</Text>
          <Text type="secondary" style={{ fontSize: 11 }}>{dayjs(t).format('HH:mm:ss')}</Text>
        </Space>
      ),
    },
    {
      title: '操作', key: 'action', width: 120,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<FileTextOutlined />} onClick={() => navigate(`/tasks/${r.id}`)}>详情</Button>
          {(r.status === 'pending') && (
            <Button type="link" size="small" danger onClick={() => handleCancel(r.id)}>取消</Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 20 }}>
        <div>
          <div className="page-title">任务管理</div>
          <Text type="secondary" style={{ fontSize: 13 }}>远程执行 Shell 命令与重启任务</Text>
        </div>
      </div>

      <Card style={{ marginBottom: 16, borderRadius: 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}>
        <Space wrap size={12}>
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
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)} style={{ marginLeft: 'auto', background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none' }}>
            新建任务
          </Button>
        </Space>
      </Card>

      <Card style={{ borderRadius: 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}>
        <Table
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          scroll={{ x: 'max-content' }}
          size="middle"
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50'],
            showTotal: (t) => `共 ${t} 个任务`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps); },
          }}
          locale={{ emptyText: <Text type="secondary">暂无任务记录</Text> }}
        />
      </Card>

      <Modal
        title={<Space><PlusOutlined style={{ color: '#667eea' }} /><span>新建任务</span></Space>}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleCreate}
        confirmLoading={submitting}
        okText="创建"
        width={560}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="host_uuids" label="目标主机" rules={[{ required: true, message: '请选择主机' }]}>
            <Select
              showSearch
              mode="multiple"
              placeholder="可选择多个主机"
              optionFilterProp="label"
              options={hosts.map((h) => ({ value: h.uuid, label: `${h.hostname} (${h.ip})` }))}
            />
          </Form.Item>
          <Form.Item name="type" label="任务类型" initialValue="shell" rules={[{ required: true }]}>
            <Select options={[
              { value: 'shell', label: 'Shell 命令' },
              { value: 'restart', label: '重启 Agent' },
              { value: 'upgrade', label: '升级 Agent' },
            ]} />
          </Form.Item>
          <Form.Item name="command" label="命令" rules={[{ required: true, message: '请输入命令' }]} extra="可在目标主机上执行的命令">
            <Input.TextArea rows={4} placeholder="例如：ls -la /tmp" />
          </Form.Item>
          <Form.Item name="timeout_sec" label="超时（秒）" initialValue={0} extra="0 表示不限制，建议设置合理超时时间">
            <InputNumber min={0} max={86400} style={{ width: 200 }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
