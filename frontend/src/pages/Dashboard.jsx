import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Table, Button, List, Tag, Space } from 'antd';
import {
  DesktopOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import dayjs from 'dayjs';
import client from '../api/client';

function exportCSV(rows, columns, filename) {
  const header = columns.map((c) => c.title).join(',');
  const lines = rows.map((r) => columns.map((c) => {
    const v = c.render ? c.render(r[c.dataIndex], r) : r[c.dataIndex];
    return `"${String(v ?? '').replace(/"/g, '""')}"`;
  }).join(','));
  const csv = [header, ...lines].join('\n');
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export default function Dashboard() {
  const [stats, setStats] = useState({ total: 0, online: 0, pending: 0, offline: 0, alert: 0 });
  const [recentHosts, setRecentHosts] = useState([]);
  const [allHosts, setAllHosts] = useState([]);
  const [recentAlerts, setRecentAlerts] = useState([]);

  useEffect(() => {
    loadStats();
    loadRecentHosts();
    loadAllHosts();
    loadRecentAlerts();
    const timer = setInterval(() => { loadStats(); loadRecentAlerts(); }, 30000);
    return () => clearInterval(timer);
  }, []);

  const loadStats = async () => {
    try {
      const [all, online, pending] = await Promise.all([
        client.get('/hosts?page_size=1'),
        client.get('/hosts?status=online&page_size=1'),
        client.get('/hosts?status=pending&page_size=1'),
      ]);
      const alerts = await client.get('/alerts?page_size=1').catch(() => ({ total: 0 }));
      const total = all.total || 0;
      const onlineCount = online.total || 0;
      const pendingCount = pending.total || 0;
      setStats({
        total,
        online: onlineCount,
        pending: pendingCount,
        offline: total - onlineCount - pendingCount,
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

  const loadAllHosts = async () => {
    try {
      const resp = await client.get('/hosts', { params: { page_size: 100 } });
      setAllHosts(resp.data || []);
    } catch (e) {}
  };

  const loadRecentAlerts = async () => {
    try {
      const resp = await client.get('/alerts', { params: { page_size: 5 } });
      setRecentAlerts(resp.data || []);
    } catch (e) {}
  };

  const osDist = {};
  allHosts.forEach((h) => { const k = h.os || '未知'; osDist[k] = (osDist[k] || 0) + 1; });

  const hostColumns = [
    { title: '主机名', dataIndex: 'hostname', key: 'hostname' },
    { title: '操作系统', dataIndex: 'os', key: 'os' },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    {
      title: '状态', dataIndex: 'status', key: 'status',
      render: (s) => {
        if (s === 'online') return <span style={{ color: '#52c41a' }}>在线</span>;
        if (s === 'pending') return <span style={{ color: '#faad14' }}>待接入</span>;
        return <span style={{ color: '#ff4d4f' }}>离线</span>;
      },
    },
  ];

  const exportColumns = [
    { title: '主机名', dataIndex: 'hostname' },
    { title: '操作系统', dataIndex: 'os' },
    { title: 'IP', dataIndex: 'ip' },
    { title: 'CPU', dataIndex: 'cpu_model' },
    { title: '内存', dataIndex: 'memory_total' },
    { title: '状态', dataIndex: 'status' },
  ];

  const statusOption = {
    title: { text: '主机状态', left: 'center', textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'item' },
    series: [{
      type: 'pie', radius: ['40%', '70%'],
      data: [
        { name: '在线', value: stats.online, itemStyle: { color: '#52c41a' } },
        { name: '待接入', value: stats.pending, itemStyle: { color: '#faad14' } },
        { name: '离线', value: stats.offline, itemStyle: { color: '#ff4d4f' } },
      ],
    }],
  };

  const osOption = {
    title: { text: '操作系统分布', left: 'center', textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'item' },
    series: [{
      type: 'pie', radius: ['40%', '70%'],
      data: Object.entries(osDist).map(([name, value]) => ({ name, value })),
    }],
  };

  return (
    <div>
      <h2>仪表盘</h2>
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={12} sm={12} md={6}>
          <Card><Statistic title="主机总数" value={stats.total} prefix={<DesktopOutlined />} /></Card>
        </Col>
        <Col xs={12} sm={12} md={6}>
          <Card><Statistic title="在线主机" value={stats.online} valueStyle={{ color: '#52c41a' }} prefix={<CheckCircleOutlined />} /></Card>
        </Col>
        <Col xs={12} sm={12} md={6}>
          <Card><Statistic title="离线主机" value={stats.offline} valueStyle={{ color: '#ff4d4f' }} prefix={<CloseCircleOutlined />} /></Card>
        </Col>
        <Col xs={12} sm={12} md={6}>
          <Card><Statistic title="告警" value={stats.alert} valueStyle={{ color: '#faad14' }} prefix={<WarningOutlined />} /></Card>
        </Col>
      </Row>
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} md={12}><Card><ReactECharts option={statusOption} style={{ height: 260 }} /></Card></Col>
        <Col xs={24} md={12}>
          <Card title="最新告警" style={{ height: '100%' }}>
            <List
              size="small"
              dataSource={recentAlerts}
              renderItem={(item) => (
                <List.Item>
                  <Space>
                    <Tag color={item.level === 'critical' ? 'red' : item.level === 'warning' ? 'orange' : 'blue'}>{item.level}</Tag>
                    <span>{item.message}</span>
                    <span style={{ color: '#888', fontSize: 12 }}>{dayjs(item.created_at).format('MM-DD HH:mm')}</span>
                  </Space>
                </List.Item>
              )}
              locale={{ emptyText: '暂无告警' }}
            />
          </Card>
        </Col>
      </Row>
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} md={12}><Card><ReactECharts option={osOption} style={{ height: 260 }} /></Card></Col>
      </Row>
      <Card
        title="最近主机"
        extra={<Button onClick={() => exportCSV(allHosts, exportColumns, 'hosts.csv')}>导出 CSV</Button>}
      >
        <Table columns={hostColumns} dataSource={recentHosts} rowKey="uuid" pagination={false} scroll={{ x: 'max-content' }} />
      </Card>
    </div>
  );
}
