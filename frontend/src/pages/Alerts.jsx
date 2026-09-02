import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Tag, Space, Select, Button, Input, Popconfirm, InputNumber, message } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';

const levelColor = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

const typeLabel = { alert: '告警', error: '错误', offline: '离线' };

export default function Alerts() {
  const [data, setData] = useState([]);
  const [hosts, setHosts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [level, setLevel] = useState('');
  const [hostID, setHostID] = useState('');
  const [ack, setAck] = useState('');

  const loadHosts = async () => {
    try {
      const resp = await client.get('/hosts', { params: { page_size: 100 } });
      setHosts(resp.data || []);
    } catch (e) {}
  };

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await client.get('/alerts', { params: { page, page_size: pageSize, level, host_id: hostID, ack } });
      setData(resp.data || []);
      setTotal(resp.total || 0);
    } catch (e) {
      setData([]);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, level, hostID, ack]);

  useEffect(() => { loadHosts(); }, []);
  useEffect(() => { load(); }, [load]);

  // 自动刷新（30 秒）
  useEffect(() => {
    const timer = setInterval(load, 30000);
    return () => clearInterval(timer);
  }, [load]);

  const hostMap = {};
  hosts.forEach((h) => { hostMap[h.id] = h; });

  const handleAck = async (id) => {
    try {
      await client.post(`/alerts/${id}/ack`);
      message.success('已确认');
      load();
    } catch (e) {}
  };

  const handleSilence = async (id, minutes) => {
    try {
      await client.post(`/alerts/${id}/silence`, { minutes });
      message.success(`已静音 ${minutes} 分钟`);
      load();
    } catch (e) {}
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
    {
      title: '主机', dataIndex: 'host_id', key: 'host',
      render: (id) => hostMap[id]?.hostname || (id ? `#${id}` : '-'),
    },
    {
      title: '类型', dataIndex: 'type', key: 'type',
      render: (v) => <Tag>{typeLabel[v] || v}</Tag>,
    },
    {
      title: '级别', dataIndex: 'level', key: 'level',
      render: (l) => <Tag color={levelColor[l] || 'default'}>{l}</Tag>,
    },
    { title: '消息', dataIndex: 'message', key: 'message', ellipsis: true },
    {
      title: '状态', dataIndex: 'acknowledged', key: 'acknowledged',
      render: (v, r) => v ? <Tag color="green">已确认</Tag> : (r.silenced_until ? <Tag color="orange">静音中</Tag> : <Tag color="red">未处理</Tag>),
    },
    {
      title: '时间', dataIndex: 'created_at', key: 'created_at',
      render: (t) => dayjs(t).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作', key: 'action', width: 180,
      render: (_, r) => (
        <Space>
          {!r.acknowledged && (
            <Popconfirm title="确认该告警？" onConfirm={() => handleAck(r.id)}>
              <Button type="link" size="small">确认</Button>
            </Popconfirm>
          )}
          <Popconfirm title="静音多少分钟？" icon={null} onConfirm={(e) => {
            const m = window.prompt('静音分钟数', '60');
            if (m) handleSilence(r.id, parseInt(m, 10));
          }}>
            <Button type="link" size="small">静音</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <h2>告警</h2>
      <Card style={{ marginBottom: 16 }}>
        <Space wrap>
          <Select
            placeholder="级别筛选"
            allowClear
            style={{ width: 140 }}
            value={level || undefined}
            onChange={(v) => { setLevel(v || ''); setPage(1); }}
            options={[
              { value: 'critical', label: '严重' },
              { value: 'warning', label: '警告' },
              { value: 'info', label: '信息' },
            ]}
          />
          <Select
            placeholder="主机筛选"
            allowClear
            showSearch
            optionFilterProp="label"
            style={{ width: 260 }}
            value={hostID || undefined}
            onChange={(v) => { setHostID(v || ''); setPage(1); }}
            options={hosts.map((h) => ({ value: String(h.id), label: `${h.hostname} (${h.ip})` }))}
          />
          <Select
            placeholder="状态筛选"
            allowClear
            style={{ width: 140 }}
            value={ack || undefined}
            onChange={(v) => { setAck(v || ''); setPage(1); }}
            options={[
              { value: 'false', label: '未处理' },
              { value: 'true', label: '已确认' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
        </Space>
      </Card>
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
