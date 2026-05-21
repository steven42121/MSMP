import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Table } from 'antd';
import {
  DesktopOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import client from '../api/client';

export default function Dashboard() {
  const [stats, setStats] = useState({ total: 0, online: 0, offline: 0, alert: 0 });
  const [recentHosts, setRecentHosts] = useState([]);

  useEffect(() => {
    loadStats();
    loadRecentHosts();
    const timer = setInterval(loadStats, 30000);
    return () => clearInterval(timer);
  }, []);

  const loadStats = async () => {
    try {
      const [all, online, offline] = await Promise.all([
        client.get('/hosts?page_size=1'),
        client.get('/hosts?status=online&page_size=1'),
        client.get('/hosts?status=offline&page_size=1'),
      ]);
      const alerts = await client.get('/alerts?page_size=1').catch(() => ({ total: 0 }));
      setStats({
        total: all.total || 0,
        online: online.total || 0,
        offline: offline.total || 0,
        alert: alerts.total || 0,
      });
    } catch (e) {
      console.error('Failed to load stats:', e);
    }
  };

  const loadRecentHosts = async () => {
    try {
      const resp = await client.get('/hosts?page_size=10');
      setRecentHosts(resp.data || []);
    } catch (e) {
      console.error('Failed to load hosts:', e);
    }
  };

  const hostColumns = [
    { title: '主机名', dataIndex: 'hostname', key: 'hostname' },
    { title: '操作系统', dataIndex: 'os', key: 'os' },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    {
      title: '状态', dataIndex: 'status', key: 'status',
      render: (s) => (
        <span style={{ color: s === 'online' ? '#52c41a' : '#ff4d4f' }}>
          {s === 'online' ? '在线' : '离线'}
        </span>
      ),
    },
  ];

  return (
    <div>
      <h2>仪表盘</h2>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card><Statistic title="主机总数" value={stats.total} prefix={<DesktopOutlined />} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="在线主机" value={stats.online} valueStyle={{ color: '#52c41a' }} prefix={<CheckCircleOutlined />} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="离线主机" value={stats.offline} valueStyle={{ color: '#ff4d4f' }} prefix={<CloseCircleOutlined />} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="告警" value={stats.alert} valueStyle={{ color: '#faad14' }} prefix={<WarningOutlined />} /></Card>
        </Col>
      </Row>
      <Card title="最近主机">
        <Table columns={hostColumns} dataSource={recentHosts} rowKey="uuid" pagination={false} />
      </Card>
    </div>
  );
}