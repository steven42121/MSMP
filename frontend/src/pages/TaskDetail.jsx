import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Descriptions, Card, Spin, Tag, Button, Space, message } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';

const statusColor = {
  pending: 'default',
  running: 'processing',
  success: 'success',
  failed: 'error',
  canceled: 'warning',
};

export default function TaskDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [task, setTask] = useState(null);
  const [loading, setLoading] = useState(true);

  const load = async () => {
    setLoading(true);
    try {
      const resp = await client.get()(`/tasks/${id}`);
      setTask(resp);
    } catch (e) {
      setTask(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [id]);

  // 待执行/执行中时自动刷新
  useEffect(() => {
    if (!task || (task.status !== 'pending' && task.status !== 'running')) return;
    const timer = setInterval(load, 3000);
    return () => clearInterval(timer);
  }, [task]);

  const handleCancel = async () => {
    try {
      await client.put()(`/tasks/${id}`, { status: 'canceled' });
      message.success('已取消');
      load();
    } catch (e) {}
  };

  const handleDelete = async () => {
    try {
      await client.delete()(`/tasks/${id}`);
      message.success('已删除');
      navigate('/tasks');
    } catch (e) {}
  };

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  if (!task) return <div>任务不存在</div>;

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/tasks')}>返回</Button>
        {task.status === 'pending' && (
          <Button danger onClick={handleCancel}>取消任务</Button>
        )}
        <Button danger type="primary" onClick={handleDelete}>删除</Button>
      </Space>
      <h2>任务详情 #{task.id}</h2>
      <Card>
        <Descriptions bordered column={{ xs: 1, sm: 2 }}>
          <Descriptions.Item label="ID">{task.id}</Descriptions.Item>
          <Descriptions.Item label="主机ID">{task.host_id}</Descriptions.Item>
          <Descriptions.Item label="类型"><Tag>{task.type}</Tag></Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={statusColor[task.status] || 'default'}>{task.status}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="命令" span={2}>
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{task.command}</pre>
          </Descriptions.Item>
          <Descriptions.Item label="创建人">{task.created_by || '-'}</Descriptions.Item>
          <Descriptions.Item label="创建时间">{dayjs(task.created_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
          <Descriptions.Item label="开始时间">
            {task.started_at ? dayjs(task.started_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="完成时间">
            {task.finished_at ? dayjs(task.finished_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="执行结果" span={2}>
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap', maxHeight: 400, overflow: 'auto' }}>
              {task.result || '-'}
            </pre>
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  );
}
