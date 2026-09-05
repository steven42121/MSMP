import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Tag, Space, Select, Button, Input, Popconfirm, InputNumber, message, Typography, Modal, Row, Col, Statistic } from 'antd';
import { ReloadOutlined, AlertOutlined, SearchOutlined, BellOutlined, PauseOutlined, RobotOutlined, FireOutlined, WarningOutlined, InfoCircleOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';

const { Text } = Typography;

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
  const [analyzing, setAnalyzing] = useState(null);
  const [analysisResult, setAnalysisResult] = useState(null);
  const [analysisVisible, setAnalysisVisible] = useState(false);
  const [stats, setStats] = useState(null);
  const [statsLoading, setStatsLoading] = useState(false);

  const loadStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const resp = await client.get()('/alert-stats');
      setStats(resp);
    } catch (e) {
      setStats(null);
    } finally {
      setStatsLoading(false);
    }
  }, []);

  useEffect(() => { loadStats(); }, [loadStats]);

  const handleAnalyze = async (id, hostId) => {
    setAnalyzing(id);
    setAnalysisVisible(true);
    setAnalysisResult(null);
    try {
      const resp = await client.post()( `/ai/analyze-alert/${id}?host_id=${hostId}` );
      setAnalysisResult(resp.analysis || '暂无分析结果');
    } catch (e) {
      setAnalysisResult('分析失败：' + (e.message || '请检查 LLM 配置'));
    } finally {
      setAnalyzing(null);
    }
  };

  const loadHosts = async () => {
    try {
      const resp = await client.get()('/hosts', { params: { page_size: 100 } });
      setHosts(resp.data || []);
    } catch (e) {}
  };

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await client.get()('/alerts', { params: { page, page_size: pageSize, level, host_id: hostID, ack } });
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

  useEffect(() => {
    const timer = setInterval(load, 30000);
    return () => clearInterval(timer);
  }, [load]);

  const hostMap = {};
  hosts.forEach((h) => { hostMap[h.id] = h; });

  const handleAck = async (id) => {
    try {
      await client.post()(`/alerts/${id}/ack`);
      message.success('已确认');
      load();
    } catch (e) {}
  };

  const handleSilence = async (id, minutes) => {
    try {
      await client.post()(`/alerts/${id}/silence`, { minutes });
      message.success(`已静音 ${minutes} 分钟`);
      load();
    } catch (e) {}
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 70, render: (v) => <Text type="secondary">{v}</Text> },
    {
      title: '主机', dataIndex: 'host_id', key: 'host',
      render: (id) => {
        const h = hostMap[id];
        return h ? (
          <Space>
            <span style={{ fontWeight: 500 }}>{h.hostname}</span>
            <Text type="secondary" style={{ fontSize: 12 }}>{h.ip}</Text>
          </Space>
          ) : (id ? <Text type="secondary">#{id}</Text> : <Text type="secondary">-</Text>);
      },
    },
    {
      title: '类型', dataIndex: 'type', key: 'type', width: 90,
      render: (v) => <Tag style={{ borderRadius: 4, padding: '1px 8px' }}>{typeLabel[v] || v}</Tag>,
    },
    {
      title: '级别', dataIndex: 'level', key: 'level', width: 80,
      render: (l) => (
        <Tag
          color={levelColor[l]}
          style={{ borderRadius: 4, padding: '2px 8px', fontWeight: 600, textTransform: 'uppercase', fontSize: 11 }}
        >
          {l || '-'}
        </Tag>
      ),
    },
    {
      title: '消息', dataIndex: 'message', key: 'message',
      render: (v) => <Text style={{ maxWidth: 360, display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{v || '-'}</Text>,
    },
    {
      title: '状态', dataIndex: 'acknowledged', key: 'acknowledged', width: 100,
      render: (v, r) => {
        if (v) return <Tag color="green" style={{ borderRadius: 4, padding: '2px 8px' }}>已确认</Tag>;
        if (r.silenced_until) return <Tag color="orange" style={{ borderRadius: 4, padding: '2px 8px' }}>静音中</Tag>;
        return <Tag color="red" style={{ borderRadius: 4, padding: '2px 8px' }}>未处理</Tag>;
      },
    },
    {
      title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170,
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
        <Space size={4} wrap>
          {!r.acknowledged && (
            <Popconfirm title="确认该告警？" onConfirm={() => handleAck(r.id)}>
              <Button type="link" size="small" icon={<AlertOutlined />}>确认</Button>
            </Popconfirm>
          )}
          <Popconfirm title="静音多少分钟？" icon={null} onConfirm={(e) => {
            const m = window.prompt('静音分钟数', '60');
            if (m) handleSilence(r.id, parseInt(m, 10));
          }}>
            <Button type="link" size="small" icon={<PauseOutlined />}>静音</Button>
          </Popconfirm>
          <Button
            type="link"
            size="small"
            icon={<RobotOutlined />}
            loading={analyzing === r.id}
            onClick={() => handleAnalyze(r.id, r.host_id)}
          >
            AI 分析
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      {/* 告警统计面板 */}
      {stats && (
        <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
          <Col xs={12} sm={8} md={6}>
            <Card className="liquid-glass" style={{ borderRadius: 12, border: 'none' }}>
              <Statistic
                title={<Space><FireOutlined style={{ color: '#ff4d4f' }} />严重告警</Space>}
                value={stats.by_level?.critical || 0}
                valueStyle={{ color: '#ff4d4f' }}
                prefix={<FireOutlined />}
              />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={6}>
            <Card className="liquid-glass" style={{ borderRadius: 12, border: 'none' }}>
              <Statistic
                title={<Space><WarningOutlined style={{ color: '#faad14' }} />警告</Space>}
                value={stats.by_level?.warning || 0}
                valueStyle={{ color: '#faad14' }}
                prefix={<WarningOutlined />}
              />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={6}>
            <Card className="liquid-glass" style={{ borderRadius: 12, border: 'none' }}>
              <Statistic
                title={<Space><InfoCircleOutlined style={{ color: '#1890ff' }} />信息</Space>}
                value={stats.by_level?.info || 0}
                valueStyle={{ color: '#1890ff' }}
                prefix={<InfoCircleOutlined />}
              />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={6}>
            <Card className="liquid-glass" style={{ borderRadius: 12, border: 'none' }}>
              <Statistic
                title="未确认"
                value={stats.unacknowledged || 0}
                valueStyle={{ color: '#722ed1' }}
                suffix={<BellOutlined />}
              />
            </Card>
          </Col>
        </Row>
      )}

      <Card className="liquid-glass" style={{ marginBottom: 16, borderRadius: 16, border: 'none' }}>
        <Space wrap size={12} className="filter-bar">
          <Select
            placeholder="级别筛选"
            allowClear
            style={{ width: 130 }}
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
            style={{ width: 240 }}
            value={hostID || undefined}
            onChange={(v) => { setHostID(v || ''); setPage(1); }}
            options={hosts.map((h) => ({ value: String(h.id), label: `${h.hostname} (${h.ip})` }))}
          />
          <Select
            placeholder="处理状态"
            allowClear
            style={{ width: 130 }}
            value={ack || undefined}
            onChange={(v) => { setAck(v || ''); setPage(1); }}
            options={[
              { value: 'false', label: '未处理' },
              { value: 'true', label: '已确认' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>刷新</Button>
        </Space>
      </Card>

      <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}>
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
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (t) => `共 ${t} 条告警`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps); },
          }}
          locale={{ emptyText: <Text type="secondary">暂无告警数据</Text> }}
        />
      </Card>

      <Modal
        title={<Space><RobotOutlined style={{ color: '#667eea' }} /><span>AI 根因分析</span></Space>}
        open={analysisVisible}
        onCancel={() => setAnalysisVisible(false)}
        footer={null}
        width={680}
        styles={{ body: { maxHeight: '60vh', overflowY: 'auto' } }}
      >
        {analysisResult ? (
          <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.8, color: '#d0d0d0', fontSize: 14 }}>
            {analysisResult}
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: 24, color: '#888' }}>
            <RobotOutlined style={{ fontSize: 32, marginBottom: 12, color: '#667eea' }} />
            <div>正在分析告警根因，请稍候...</div>
          </div>
        )}
      </Modal>
    </div>
  );
}
