import React, { useEffect, useState } from 'react';
import { Table, Tag, Button, Space, Modal, Form, Input, InputNumber, Popconfirm, message, Typography } from 'antd';
import { PlusOutlined, ReloadOutlined, PlayCircleOutlined, DeleteOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';

const { Text } = Typography;

export default function CronJobs() {
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [runningId, setRunningId] = useState(null);

  const load = async () => {
    setLoading(true);
    try {
      const resp = await client.get()('/cron-jobs');
      setJobs(Array.isArray(resp) ? resp : []);
    } catch (e) { message.error('加载失败'); }
    finally { setLoading(false); }
  };

  useEffect(() => { load(); }, []);

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      await client.post()('/cron-jobs', values);
      message.success('已创建');
      form.resetFields();
      setModalOpen(false);
      load();
    } catch (e) { if (e.errorFields) return; message.error('创建失败'); }
  };

  const handleDelete = async (id) => {
    try {
      await client.delete()(`/cron-jobs/${id}`);
      message.success('已删除');
      load();
    } catch (e) { message.error('删除失败'); }
  };

  const handleRun = async (id) => {
    setRunningId(id);
    try {
      await client.post()(`/cron-jobs/${id}/run`);
      message.success('已触发');
      load();
    } catch (e) { message.error('触发失败'); }
    finally { setRunningId(null); }
  };

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: 'Cron 表达式', dataIndex: 'expression', key: 'expression', render: (v) => <Text code>{v}</Text> },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '启用', dataIndex: 'enabled', key: 'enabled', width: 70,
      render: (v) => v ? <Tag color="green">是</Tag> : <Tag color="default">否</Tag>,
    },
    { title: '下次运行', dataIndex: 'next_run_at', key: 'next_run_at', width: 160, render: (t) => t ? dayjs(t).format('MM-DD HH:mm:ss') : '-' },
    { title: '最后运行', dataIndex: 'last_run_at', key: 'last_run_at', width: 160, render: (t) => t ? dayjs(t).format('MM-DD HH:mm:ss') : '-' },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 160, render: (t) => dayjs(t).format('MM-DD HH:mm') },
    {
      title: '操作', key: 'action', width: 140,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<PlayCircleOutlined />} loading={runningId === r.id} onClick={() => handleRun(r.id)}>执行</Button>
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
        <div className="page-title">定时任务</div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>新建任务</Button>
        </Space>
      </div>
      <Table columns={columns} dataSource={jobs} rowKey="id" loading={loading} size="small" pagination={{ pageSize: 20 }} />

      <Modal title="新建定时任务" open={modalOpen} onOk={handleCreate} onCancel={() => { setModalOpen(false); form.resetFields(); }} width={500}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input placeholder="如：每日数据清理" />
          </Form.Item>
          <Form.Item name="expression" label="Cron 表达式" rules={[{ required: true }]}
            extra="格式：秒 分 时 日 月 周（如 0 30 8 * * * = 每天 8:30）">
            <Input placeholder="0 30 8 * * *" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
