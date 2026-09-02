import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Table, Button, List, Tag, Space, Typography } from 'antd';
import {
  DesktopOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  ClusterOutlined,
  AlertOutlined,
} from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import dayjs from 'dayjs';
import client from '../api/client';

const { Title, Text } = Typography;

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

const statCards = [
  { title: '主机总数', value: 'total', icon: <DesktopOutlined />, color: '#667eea', bgColor: 'rgba(102,126,234,0.1)', textColor: '#667eea' },
  { title: '在线主机', value: 'online', icon: <CheckCircleOutlined />, color: '#52c41a', bgColor: 'rgba(82,196,26,0.1)', textColor: '#52c41a' },
  { title: '离线主机', value: 'offline', icon: <CloseCircleOutlined />, color: '#ff4d4f', bgColor: 'rgba(255,77,79,0.1)', textColor: '#ff4d4f' },
  { title: '告警数量', value: 'alert', icon: <WarningOutlined />, color: '#faad14', bgColor: 'rgba(250,173,20,0.1)', textColor: '#faad14' },
];

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
    } catch (e) { console.error('Failed to load hosts:', e); }
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
        const dotClass = s === 'online' ? 'status-dot online' : s === 'pending' ? 'status-dot pending' : 'status-dot offline';
        const label = s === 'online' ? '在线' : s === 'pending' ? '待接入' : '离线';
        return <span className={dotClass}>{label}</span>;
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
    title: { text: '主机状态分布', left: 'center', textStyle: { fontSize: 13, color: 'rgba(0,0,0,0.65)' } },
    tooltip: { trigger: 'item' },
    legend: { bottom: 4, textStyle: { fontSize: 12 } },
    series: [{
      type: 'pie', radius: ['38%', '68%'], center: ['50%', '46%'],
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: { show: true, formatter: '{b}: {c}', fontSize: 12 },
      data: [
        { name: '在线', value: stats.online, itemStyle: { color: '#52c41a' } },
        { name: '待接入', value: stats.pending, itemStyle: { color: '#faad14' } },
        { name: '离线', value: stats.offline, itemStyle: { color: '#ff4d4f' } },
      ],
    }],
  };

  const osOption = {
    title: { text: '操作系统分布', left: 'center', textStyle: { fontSize: 13, color: 'rgba(0,0,0,0.65)' } },
    tooltip: { trigger: 'item' },
    legend: { bottom: 4, textStyle: { fontSize: 12 } },
    series: [{
      type: 'pie', radius: ['38%', '68%'], center: ['50%', '46%'],
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: { show: true, formatter: '{b}: {c}', fontSize: 12 },
      data: Object.entries(osDist).map(([name, value], i) => ({
        name, value,
        itemStyle: { color: ['#667eea', '#764ba2', '#52c41a', '#faad14', '#ff4d4f', '#1890ff', '#13c2c2'][i % 7] },
      })),
    }],
  };

  return (
    <div>
      {/* Page header */}
      <div className="page-header" style={{ marginBottom: 24 }}>
        <div>
          <div className="page-title">仪表盘</div>
          <Text type="secondary" style={{ fontSize: 13 }}>实时监控主机状态与告警概览</Text>
        </div>
      </div>

      {/* Stats cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        {statCards.map((card) => (
          <Col key={card.value} xs={12} sm={12} md={6}>
            <Card
              style={{
                borderRadius: 12, border: 'none',
                background: card.bgColor,
                transition: 'transform 0.2s ease, box-shadow 0.2s ease',
                cursor: 'default',
              }}
              onMouseEnter={(e) => { e.currentTarget.style.transform = 'translateY(-2px)'; }}
              onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0)'; }}
            >
              <Statistic
                title={
                  <Space>
                    <span style={{ color: card.color }}>{card.icon}</span>
                    <span>{card.title}</span>
                  </Space>
                }
                value={stats[card.value]}
                valueStyle={{ color: card.textColor, fontWeight: 700, fontSize: 32 }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      {/* Charts row */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} md={12}>
          <Card
            style={{ borderRadius: 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
          >
            <ReactECharts option={statusOption} style={{ height: 260 }} />
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card
            style={{ borderRadius: 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
            title={
              <Space>
                <AlertOutlined style={{ color: '#faad14' }} />
                <span>最新告警</span>
              </Space>
            }
          >
            <List
              size="small"
              dataSource={recentAlerts}
              renderItem={(item) => (
                <List.Item style={{ padding: '8px 0' }}>
                  <Space>
                    <Tag color={item.level === 'critical' ? 'red' : item.level === 'warning' ? 'orange' : 'blue'} style={{ borderRadius: 4 }}>
                      {item.level.toUpperCase()}
                    </Tag>
                    <span style={{ maxWidth: 260, overflow: 'hidden', textOverflow: ellipsis, whiteSpace: 'nowrap' }}>
                      {item.message}
                    </span>
                    <span style={{ color: 'rgba(0,0,0,0.35)', fontSize: 12, flexShrink: 0 }}>
                      {dayjs(item.created_at).format('MM-DD HH:mm')}
                    </span>
                  </Space>
                </List.Item>
              )}
              locale={{ emptyText: <Text type="secondary" style={{ textAlign: 'center', display: 'block', padding: '24px 0' }}>暂无告警数据</Text> }}
            />
          </Card>
        </Col>
      </Row>

      {/* OS distribution */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} md={12}>
          <Card
            style={{ borderRadius: 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
          >
            <ReactECharts option={osOption} style={{ height: 260 }} />
          </Card>
        </Col>
      </Row>

      {/* Host table */}
      <Card
        style={{ borderRadius: 12, border: 'none', boxShadow: '0 2px 8px rgba(0,0,0,0.04)' }}
        title={
          <Space>
            <DesktopOutlined style={{ color: '#667eea' }} />
            <span>最近主机</span>
          </Space>
        }
        extra={
          <Button
            type="primary" ghost
            onClick={() => exportCSV(allHosts, exportColumns, 'hosts.csv')}
            icon={<ClusterOutlined />}
            style={{ borderRadius: 8 }}
          >
            导出 CSV
          </Button>
        }
      >
        <Table
          columns={hostColumns}
          dataSource={recentHosts}
          rowKey="uuid"
          pagination={false}
          scroll={{ x: 'max-content' }}
          size="middle"
          locale={{ emptyText: <Text type="secondary">暂无主机数据</Text> }}
        />
      </Card>
    </div>
  );
}
