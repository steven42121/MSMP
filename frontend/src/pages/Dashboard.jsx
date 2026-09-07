import React, { useEffect, useState, useMemo, useCallback } from 'react';
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
import { useThemeStore } from '../store/theme';

const { Text } = Typography;

// ── 常量 ────────────────────────────────────────────────────────────────────
const STATUS_META = {
  online: { label: '在线', color: '#52c41a' },
  pending: { label: '待接入', color: '#faad14' },
  offline: { label: '离线', color: '#ff4d4f' },
};

const OS_COLORS = ['#667eea', '#764ba2', '#52c41a', '#faad14', '#ff4d4f', '#1890ff', '#13c2c2'];

const STAT_CARDS = [
  { title: '主机总数', key: 'total', icon: <DesktopOutlined />, color: '#667eea' },
  { title: '在线主机', key: 'online', icon: <CheckCircleOutlined />, color: '#52c41a' },
  { title: '离线主机', key: 'offline', icon: <CloseCircleOutlined />, color: '#ff4d4f' },
  { title: '告警数量', key: 'alert', icon: <WarningOutlined />, color: '#faad14' },
];

const HOST_COLUMNS = [
  { title: '主机名', dataIndex: 'hostname', key: 'hostname' },
  { title: '操作系统', dataIndex: 'os', key: 'os' },
  { title: 'IP', dataIndex: 'ip', key: 'ip' },
  {
    title: '状态', dataIndex: 'status', key: 'status',
    render: (s) => <span className={`status-dot ${s}`}>{STATUS_META[s]?.label || s}</span>,
  },
];

const EXPORT_COLUMNS = [
  { title: '主机名', dataIndex: 'hostname' },
  { title: '操作系统', dataIndex: 'os' },
  { title: 'IP', dataIndex: 'ip' },
  { title: 'CPU', dataIndex: 'cpu_model' },
  { title: '内存(B)', dataIndex: 'memory_total' },
  { title: '状态', dataIndex: 'status' },
];

// ── 工具函数 ────────────────────────────────────────────────────────────────
function exportCSV(rows, columns, filename) {
  const header = columns.map((c) => c.title).join(',');
  const lines = rows.map((r) =>
    columns.map((c) => `"${String(r[c.dataIndex] ?? '').replace(/"/g, '""')}"`).join(','));
  const blob = new Blob([[header, ...lines].join('\n')], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

// 环形图统一配置：中心总数 + 外部标签（名称 + 百分比），适配明暗主题
function buildPieOption(title, data, dark) {
  const total = data.reduce((s, d) => s + (d.value || 0), 0);
  const hasData = total > 0;
  const cTitle = dark ? 'rgba(255,255,255,0.9)' : 'rgba(0,0,0,0.78)';
  const cText = dark ? 'rgba(255,255,255,0.7)' : 'rgba(0,0,0,0.65)';
  const cLine = dark ? 'rgba(255,255,255,0.25)' : 'rgba(0,0,0,0.3)';
  const cBorder = dark ? 'rgba(22,22,28,0.9)' : '#fff';
  return {
    title: { text: title, left: 'center', top: 0, textStyle: { fontSize: 14, fontWeight: 600, color: cTitle } },
    tooltip: { trigger: 'item', formatter: '{b}: {c}（{d}%）' },
    legend: {
      bottom: 0, left: 'center', icon: 'circle',
      itemWidth: 10, itemHeight: 10, itemGap: 16,
      textStyle: { fontSize: 12, color: cText },
    },
    series: [{
      type: 'pie',
      radius: ['36%', '50%'],
      center: ['50%', '53%'],
      avoidLabelOverlap: true,
      padAngle: 2,
      itemStyle: { borderRadius: 6, borderColor: cBorder, borderWidth: 2 },
      label: { show: hasData, formatter: '{b} {d}%', fontSize: 11, color: cText },
      labelLine: { length: 10, length2: 6, lineStyle: { color: cLine } },
      emphasis: {
        scaleSize: 6,
        itemStyle: { shadowBlur: 12, shadowColor: 'rgba(0,0,0,0.25)' },
        label: { fontSize: 12, fontWeight: 600, color: cTitle },
      },
      data: hasData
        ? data
        : [{ name: '暂无数据', value: 1, itemStyle: { color: dark ? 'rgba(255,255,255,0.08)' : '#e8e8e8' } }],
    }],
    graphic: hasData ? {
      type: 'text', left: 'center', top: '48%',
      style: { text: String(total), textAlign: 'center', fontSize: 22, fontWeight: 700, fill: cTitle },
    } : undefined,
  };
}

// ── 主页面 ──────────────────────────────────────────────────────────────────
export default function Dashboard() {
  const { dark } = useThemeStore();
  const [stats, setStats] = useState({ total: 0, online: 0, pending: 0, offline: 0, alert: 0 });
  const [hosts, setHosts] = useState([]);
  const [recentAlerts, setRecentAlerts] = useState([]);

  const loadStats = useCallback(async () => {
    try {
      const [all, online, pending, alerts] = await Promise.all([
        client.get()('/hosts', { page_size: 1 }),
        client.get()('/hosts', { status: 'online', page_size: 1 }),
        client.get()('/hosts', { status: 'pending', page_size: 1 }),
        client.get()('/alerts', { page_size: 1 }),
      ]);
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
  }, []);

  const loadHosts = useCallback(async () => {
    try {
      const resp = await client.get()('/hosts', { page_size: 100 });
      setHosts(resp.data || []);
    } catch (e) {
      console.error('Failed to load hosts:', e);
    }
  }, []);

  const loadAlerts = useCallback(async () => {
    try {
      const resp = await client.get()('/alerts', { page_size: 5 });
      setRecentAlerts(resp.data || []);
    } catch (e) {}
  }, []);

  useEffect(() => {
    loadStats();
    loadHosts();
    loadAlerts();
    const timer = setInterval(() => { loadStats(); loadAlerts(); }, 30_000);
    return () => clearInterval(timer);
  }, [loadStats, loadHosts, loadAlerts]);

  const recentHosts = useMemo(() => hosts.slice(0, 10), [hosts]);
  const osDist = useMemo(() => {
    const dist = {};
    hosts.forEach((h) => { const k = h.os || '未知'; dist[k] = (dist[k] || 0) + 1; });
    return dist;
  }, [hosts]);

  const statusOption = useMemo(() => buildPieOption('主机状态分布', [
    { name: '在线', value: stats.online, itemStyle: { color: STATUS_META.online.color } },
    { name: '待接入', value: stats.pending, itemStyle: { color: STATUS_META.pending.color } },
    { name: '离线', value: stats.offline, itemStyle: { color: STATUS_META.offline.color } },
  ], dark), [stats, dark]);

  const osOption = useMemo(() => buildPieOption('操作系统分布',
    Object.entries(osDist).map(([name, value], i) => ({ name, value, itemStyle: { color: OS_COLORS[i % OS_COLORS.length] } })),
    dark), [osDist, dark]);

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 24 }}>
        <div>
          <div className="page-title">仪表盘</div>
          <Text type="secondary" style={{ fontSize: 13 }}>实时监控主机状态与告警概览</Text>
        </div>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        {STAT_CARDS.map((card) => (
          <Col key={card.key} xs={12} sm={12} md={6}>
            <Card className="liquid-glass" style={{ borderRadius: 20, border: 'none', cursor: 'default' }}>
              <Statistic
                title={
                  <Space>
                    <span style={{ color: card.color }}>{card.icon}</span>
                    <span style={{ fontSize: 13, fontWeight: 500 }}>{card.title}</span>
                  </Space>
                }
                value={stats[card.key]}
                valueStyle={{ color: card.color, fontWeight: 700, fontSize: 32 }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      {/* 状态分布 + 告警 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} md={12}>
          <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}>
            <ReactECharts option={statusOption} style={{ height: 260 }} />
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}
            title={<Space><AlertOutlined style={{ color: '#faad14' }} /><span>最新告警</span></Space>}
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
                    <span style={{ maxWidth: 260, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
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

      {/* 操作系统分布 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} md={12}>
          <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}>
            <ReactECharts option={osOption} style={{ height: 260 }} />
          </Card>
        </Col>
      </Row>

      {/* 最近主机 */}
      <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}
        title={<Space><DesktopOutlined style={{ color: '#667eea' }} /><span>最近主机</span></Space>}
        extra={
          <Button type="primary" ghost onClick={() => exportCSV(hosts, EXPORT_COLUMNS, 'hosts.csv')}
            icon={<ClusterOutlined />} style={{ borderRadius: 8 }}>
            导出 CSV
          </Button>
        }
      >
        <Table
          columns={HOST_COLUMNS}
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