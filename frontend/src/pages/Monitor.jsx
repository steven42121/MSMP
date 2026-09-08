import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { Card, Select, Space, Spin, Empty, Row, Col, Button, Typography, Popconfirm } from 'antd';
import { LineChartOutlined, DownloadOutlined, ReloadOutlined, BgColorsOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import dayjs from 'dayjs';
import client from '../api/client';
import { formatBytes, formatNetSpeed } from '../utils/format';

const { Text } = Typography;

// ── 常量 ────────────────────────────────────────────────────────────────────
const DURATIONS = [
  { value: '30m', label: '30 分钟' },
  { value: '1h',  label: '1 小时'  },
  { value: '3h',  label: '3 小时'  },
  { value: '6h',  label: '6 小时'  },
  { value: '12h', label: '12 小时' },
  { value: '24h', label: '24 小时' },
  { value: '7d',  label: '7 天'    },
  { value: '30d', label: '30 天'   },
];

const METRIC_COLORS = ['#667eea', '#764ba2', '#52c41a', '#faad14', '#ff4d4f', '#1890ff', '#722ed1', '#a6ee3c'];

// 生成单线图 ECharts 配置
function buildLineOption(title, data, field, unit, color) {
  const times = data.map((d) => dayjs(d.timestamp).format('HH:mm:ss'));
  return {
    title: { text: title, left: 'center', textStyle: { fontSize: 13 } },
    tooltip: { trigger: 'axis' },
    grid: { left: 48, right: 24, top: 40, bottom: 32 },
    xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
    yAxis: {
      type: 'value',
      axisLabel: { formatter: (v) => `${v}${unit || ''}` },
      splitLine: { lineStyle: { type: 'dashed', color: 'rgba(0,0,0,0.06)' } },
    },
    series: [{
      name: title, type: 'line', data: data.map((d) => d[field]),
      smooth: true, showSymbol: false,
      areaStyle: { opacity: 0.1 },
      itemStyle: { color },
      lineStyle: { width: 2.5 },
    }],
  };
}

// ── 子组件：监控卡片统一外壳（标题 + 彩色图标 + 可选 extra 操作） ───────────
function ChartCard({ title, iconColor, loading, onRefresh, extra, children }) {
  return (
    <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}
      title={<Space><LineChartOutlined style={{ color: iconColor }} /><span>{title}</span></Space>}
      loading={loading}
      extra={extra || (onRefresh && <Button size="small" onClick={onRefresh} loading={loading}>刷新</Button>)}
    >
      {children}
    </Card>
  );
}

// GPU 列表内容
function GpuInfo({ gpus }) {
  if (!gpus.length) return <Text type="secondary">未检测到 GPU</Text>;
  return (
    <div style={{ maxHeight: 220, overflowY: 'auto' }}>
      {gpus.map((gpu, i) => (
        <div key={i} style={{ padding: '10px 12px', background: 'rgba(82,196,26,0.08)', borderRadius: 8, marginBottom: 8 }}>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>{gpu.name || `GPU ${i + 1}`}</div>
          <div style={{ fontSize: 12, color: 'rgba(0,0,0,0.65)' }}>
            <div>厂商: {gpu.vendor || '-'}</div>
            <div>显存: {gpu.memory_total ? formatBytes(gpu.memory_total) : '-'} / {gpu.memory_used ? formatBytes(gpu.memory_used) : '-'}</div>
            <div>温度: {gpu.temperature_c != null ? `${gpu.temperature_c}°C` : '-'}</div>
            <div>利用率: {gpu.utilization_gpu != null ? `${gpu.utilization_gpu}%` : '-'}</div>
            {gpu.driver_version && <div>驱动: {gpu.driver_version}</div>}
          </div>
        </div>
      ))}
    </div>
  );
}

// 温度传感器列表内容
function TempInfo({ temps }) {
  if (!temps.length) return <Text type="secondary">未检测到温度传感器</Text>;
  return (
    <div style={{ maxHeight: 220, overflowY: 'auto' }}>
      {temps.map((t, i) => {
        const isHigh = t.temp > (t.critical || 90);
        const isWarn = t.temp > (t.high || 70);
        return (
          <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 12px',
            background: isHigh ? 'rgba(255,77,79,0.1)' : isWarn ? 'rgba(250,173,20,0.1)' : 'rgba(82,196,26,0.08)',
            borderRadius: 6, marginBottom: 4 }}>
            <span style={{ fontSize: 13 }}>{t.sensor_key || `Sensor ${i + 1}`}</span>
            <span style={{ fontSize: 13, fontWeight: 600, color: isHigh ? '#ff4d4f' : isWarn ? '#faad14' : '#52c41a' }}>
              {t.temp != null ? `${t.temp}°C` : '-'}
            </span>
          </div>
        );
      })}
    </div>
  );
}

// ── 主页面 ──────────────────────────────────────────────────────────────────
export default function Monitor() {
  const [hosts, setHosts] = useState([]);
  const [hostUUID, setHostUUID] = useState();
  const [duration, setDuration] = useState('24h');
  const [metrics, setMetrics] = useState([]);
  const [loading, setLoading] = useState(false);
  const [lastUpdate, setLastUpdate] = useState(null);
  const [currentHost, setCurrentHost] = useState(null);
  const [assets, setAssets] = useState([]);
  const [assetLoading, setAssetLoading] = useState(false);
  const [flushing, setFlushing] = useState(false);

  // ── 数据加载 ────────────────────────────────────────────────────────────
  const loadHosts = useCallback(() => {
    client.get()('/hosts', { params: { page_size: 100 } })
      .then((resp) => {
        const data = resp.data || resp || [];
        setHosts(data);
        if (data.length) {
          setHostUUID(data[0].uuid);
          setCurrentHost(data[0]);
        }
      })
      .catch(() => setHosts([]));
  }, []);

  const loadAssets = useCallback(() => {
    if (!hostUUID) return;
    setAssetLoading(true);
    client.get()(`/hosts/${hostUUID}/assets`)
      .then((resp) => setAssets(Array.isArray(resp) ? resp : (resp.data || [])))
      .catch(() => setAssets([]))
      .finally(() => setAssetLoading(false));
  }, [hostUUID]);

  const loadMetrics = useCallback(() => {
    if (!hostUUID) return;
    setLoading(true);
    client.get()('/metrics', { host_uuid: hostUUID, duration })
      .then((resp) => { setMetrics(resp.data || []); setLastUpdate(new Date()); })
      .catch(() => setMetrics([]))
      .finally(() => setLoading(false));
  }, [hostUUID, duration]);

  // 初始化：加载主机列表，选中第一台后自动加载指标和资产
  useEffect(() => {
    loadHosts();
  }, [loadHosts]);

  useEffect(() => {
    if (!hostUUID) return;
    loadMetrics();
    loadAssets();
  }, [hostUUID, loadMetrics, loadAssets]);

  // 定时刷新
  useEffect(() => {
    if (!hostUUID) return;
    const mt = setInterval(loadMetrics, 30_000);
    const at = setInterval(loadAssets, 5 * 60_000);
    return () => { clearInterval(mt); clearInterval(at); };
  }, [hostUUID, loadMetrics, loadAssets]);

  // ── 缓存清理 ────────────────────────────────────────────────────────────
  const handleFlushCaches = async () => {
    if (!hostUUID || !currentHost) return;
    setFlushing(true);
    try {
      const resp = await client.post()('/maintenance/flush-caches', {
        host_uuid: hostUUID, cache_type: 'all',
      });
      const taskId = resp.task_id;
      message.loading({ content: '清理缓存任务已提交，等待执行...', key: 'flush', duration: 0 });
      for (let i = 0; i < 20; i++) {
        await new Promise((r) => setTimeout(r, 3_000));
        try {
          const task = await client.get()(`/tasks/${taskId}`);
          if (task.status === 'success') {
            message.success({ content: '缓存清理完成', key: 'flush', duration: 4 });
            break;
          }
          if (task.status === 'failed') {
            message.error({ content: '清理失败：' + (task.result || '未知错误'), key: 'flush', duration: 8 });
            break;
          }
        } catch {}
      }
      loadMetrics();
    } catch (e) {
      message.error('提交失败：' + (e.message || '请重试'));
    } finally {
      setFlushing(false);
    }
  };

  // ── 图表选项（memo） ────────────────────────────────────────────────────
  const charts = useMemo(() => {
    if (!metrics.length) return null;
    const mk = (title, field, unit, color) => buildLineOption(title, metrics, field, unit, color);
    const times = metrics.map((d) => dayjs(d.timestamp).format('HH:mm:ss'));

    const diskIO = {
      title: { text: '磁盘 IO (累计)', left: 'center', textStyle: { fontSize: 13 } },
      tooltip: { trigger: 'axis' }, legend: { data: ['读取', '写入'], bottom: 4 },
      grid: { left: 60, right: 24, top: 40, bottom: 40 },
      xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
      yAxis: { type: 'value', axisLabel: { formatter: formatBytes } },
      series: [
        { name: '读取', type: 'line', data: metrics.map((d) => d.disk_read_bytes), smooth: true, showSymbol: false, lineStyle: { width: 2 }, itemStyle: { color: '#52c41a' }, areaStyle: { opacity: 0.1 } },
        { name: '写入', type: 'line', data: metrics.map((d) => d.disk_write_bytes), smooth: true, showSymbol: false, lineStyle: { width: 2 }, itemStyle: { color: '#faad14' }, areaStyle: { opacity: 0.1 } },
      ],
    };

    const netPkts = {
      title: { text: '网络包计数 (累计)', left: 'center', textStyle: { fontSize: 13 } },
      tooltip: { trigger: 'axis' }, legend: { data: ['收包', '发包'], bottom: 4 },
      grid: { left: 60, right: 24, top: 40, bottom: 40 },
      xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
      yAxis: { type: 'value' },
      series: [
        { name: '收包', type: 'line', data: metrics.map((d) => d.net_pkts_recv), smooth: true, showSymbol: false, lineStyle: { width: 2 }, itemStyle: { color: '#1890ff' } },
        { name: '发包', type: 'line', data: metrics.map((d) => d.net_pkts_sent), smooth: true, showSymbol: false, lineStyle: { width: 2 }, itemStyle: { color: '#722ed1' } },
      ],
    };

    const net = {
      title: { text: '网络流量', left: 'center', textStyle: { fontSize: 13 } },
      tooltip: {
        trigger: 'axis',
        formatter: (params) => {
          let s = `<span style="font-weight:600">${params[0].axisValue}</span><br/>`;
          params.forEach((p) => { s += `${p.marker}${p.seriesName}: <b>${formatNetSpeed(p.value)}</b><br/>`; });
          return s;
        },
      },
      legend: { data: ['入站', '出站'], bottom: 4, textStyle: { fontSize: 11 } },
      grid: { left: 60, right: 24, top: 40, bottom: 40 },
      xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
      yAxis: { type: 'value', axisLabel: { formatter: formatNetSpeed }, splitLine: { lineStyle: { type: 'dashed', color: 'rgba(0,0,0,0.06)' } } },
      series: [
        { name: '入站', type: 'line', data: metrics.map((d) => d.net_rx_bps), smooth: true, showSymbol: false, lineStyle: { width: 2.5 }, itemStyle: { color: '#667eea' }, areaStyle: { opacity: 0.1 } },
        { name: '出站', type: 'line', data: metrics.map((d) => d.net_tx_bps), smooth: true, showSymbol: false, lineStyle: { width: 2.5 }, itemStyle: { color: '#764ba2' }, areaStyle: { opacity: 0.1 } },
      ],
    };

    return { cpu: mk('CPU 使用率 (%)', 'cpu_percent', '%', METRIC_COLORS[0]),
             mem: mk('内存使用率 (%)', 'mem_percent', '%', METRIC_COLORS[1]),
             load: mk('系统负载 (1min)', 'load1', '', METRIC_COLORS[3]),
             proc: mk('进程数', 'process_count', '', METRIC_COLORS[5]),
             diskIO, netPkts, net };
  }, [metrics]);

  // ── 实时统计条 ──────────────────────────────────────────────────────────
  const stats = useMemo(() => {
    if (!metrics.length) return null;
    const l = metrics[metrics.length - 1];
    return {
      cpu: l.cpu_percent?.toFixed(1) + '%',
      mem: l.mem_percent?.toFixed(1) + '%',
      swap: (l.swap_used && l.swap_total) ? ((l.swap_used / l.swap_total) * 100).toFixed(1) + '%' : '-',
      disk: (l.disk_used && l.disk_total) ? ((l.disk_used / l.disk_total) * 100).toFixed(1) + '%' : '-',
      load: l.load1?.toFixed(2) || '-',
      procs: l.process_count?.toString() || '-',
      diskR: formatBytes(l.disk_read_bytes),
      diskW: formatBytes(l.disk_write_bytes),
      rx: formatNetSpeed(l.net_rx_bps),
      tx: formatNetSpeed(l.net_tx_bps),
      pktsR: l.net_pkts_recv?.toLocaleString() || '-',
      pktsT: l.net_pkts_sent?.toLocaleString() || '-',
    };
  }, [metrics]);

  // ── CSV 导出 ────────────────────────────────────────────────────────────
  const handleExport = useCallback(() => {
    if (!metrics.length) return;
    const header = 'timestamp,cpu_percent,mem_percent,mem_used,mem_total,disk_used,disk_total,net_rx_bps,net_tx_bps,load1\n';
    const lines = metrics.map((d) => [d.timestamp, d.cpu_percent, d.mem_percent, d.mem_used, d.mem_total,
      d.disk_used, d.disk_total, d.net_rx_bps, d.net_tx_bps, d.load1].join(',')).join('\n');
    const blob = new Blob([header + lines], { type: 'text/csv;charset=utf-8;' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = `metrics_${hostUUID || 'unknown'}.csv`;
    a.click();
    URL.revokeObjectURL(a.href);
  }, [metrics, hostUUID]);

  // ── 渲染 ────────────────────────────────────────────────────────────────
  const latestAsset = assets[0];

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 24 }}>
        <div>
          <div className="page-title">监控</div>
          <Text type="secondary" style={{ fontSize: 13 }}>主机实时性能指标与趋势分析</Text>
        </div>
      </div>

      {/* 控制面板 */}
      <Card className="liquid-glass" style={{ marginBottom: 16, borderRadius: 16, border: 'none' }}>
        <Space wrap className="filter-bar">
          <Select showSearch style={{ width: 320 }} placeholder="选择主机" optionFilterProp="label"
            value={hostUUID} onChange={(v) => { setHostUUID(v); setMetrics([]); setCurrentHost(hosts.find(h => h.uuid === v) || null); }}
            options={hosts.map((h) => ({ value: h.uuid, label: `${h.hostname || '未知主机'} (${h.ip || '-'})` }))} />
          <Select style={{ width: 140 }} value={duration} onChange={setDuration} options={DURATIONS} />
          <Button onClick={loadMetrics} loading={loading} icon={<ReloadOutlined />} style={{ borderRadius: 8 }}>刷新</Button>
          <Button onClick={handleExport} disabled={!metrics.length} icon={<DownloadOutlined />} style={{ borderRadius: 8 }}>导出 CSV</Button>
          {lastUpdate && <Text type="secondary" style={{ fontSize: 12 }}>最后更新：{dayjs(lastUpdate).format('HH:mm:ss')}</Text>}
        </Space>
      </Card>

      {/* 实时统计条 */}
      {stats && (
        <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
          {[
            { label: 'CPU',    value: stats.cpu,    color: '#667eea' },
            { label: '内存',   value: stats.mem,    color: '#764ba2' },
            { label: '交换',   value: stats.swap,   color: '#faad14' },
            { label: '磁盘',   value: stats.disk,   color: '#52c41a' },
            { label: '进程',   value: stats.procs,  color: '#1890ff' },
            { label: '负载',   value: stats.load,   color: '#ff4d4f' },
            { label: '磁盘读', value: stats.diskR,  color: '#a6ee3c' },
            { label: '磁盘写', value: stats.diskW,  color: '#f9e2af' },
            { label: '入站',   value: stats.rx,     color: '#667eea' },
            { label: '出站',   value: stats.tx,     color: '#764ba2' },
            { label: '收包',   value: stats.pktsR,  color: '#1890ff' },
            { label: '发包',   value: stats.pktsT,  color: '#722ed1' },
          ].map(({ label, value, color }) => (
            <Col key={label} xs={12} sm={8} md={4} lg={3}>
              <div style={{ textAlign: 'center', padding: '8px 4px', background: color + '10', borderRadius: 8, border: `1px solid ${color}30` }}>
                <div style={{ fontSize: 10, color: 'var(--text-secondary)', marginBottom: 2 }}>{label}</div>
                <div style={{ fontSize: 14, fontWeight: 700, color, wordBreak: 'break-all' }}>{value}</div>
              </div>
            </Col>
          ))}
        </Row>
      )}

      {/* 内容区：所有卡片统一依赖主机选择 */}
      {!hostUUID ? (
        <Empty description="请选择主机" style={{ margin: '80px 0' }} />
      ) : (
        <>
          {/* 指标图表：加载中 / 空数据 / 有数据三种状态 */}
          {loading && !metrics.length ? (
            <div style={{ textAlign: 'center', padding: '80px 0' }}><Spin size="large" /></div>
          ) : !metrics.length ? (
            <Empty description="暂无监控数据" style={{ margin: '60px 0' }} />
          ) : (
            <Row gutter={[16, 16]}>
              <Col xs={24} md={12}>
                <ChartCard title="CPU 使用率" iconColor="#667eea">
                  <ReactECharts option={charts.cpu} style={{ height: 260 }} />
                </ChartCard>
              </Col>
              <Col xs={24} md={12}>
                <ChartCard title="内存使用率" iconColor="#764ba2"
                  extra={
                    <Popconfirm title="确认清理内存缓存？" description="将释放页缓存、目录缓存和索引节点缓存"
                      onConfirm={handleFlushCaches} okText="清理" cancelText="取消" disabled={flushing}>
                      <Button type="primary" size="small" icon={<BgColorsOutlined />} loading={flushing} disabled={flushing}>清理缓存</Button>
                    </Popconfirm>
                  }
                >
                  <ReactECharts option={charts.mem} style={{ height: 260 }} />
                </ChartCard>
              </Col>
              <Col xs={24} md={12}>
                <ChartCard title="系统负载" iconColor="#faad14">
                  <ReactECharts option={charts.load} style={{ height: 260 }} />
                </ChartCard>
              </Col>
              <Col xs={24} md={12}>
                <ChartCard title="网络流量" iconColor="#52c41a">
                  <ReactECharts option={charts.net} style={{ height: 260 }} />
                </ChartCard>
              </Col>
              <Col xs={24} md={8}>
                <ChartCard title="进程数" iconColor="#1890ff">
                  <ReactECharts option={charts.proc} style={{ height: 220 }} />
                </ChartCard>
              </Col>
              <Col xs={24} md={8}>
                <ChartCard title="磁盘 IO" iconColor="#faad14">
                  <ReactECharts option={charts.diskIO} style={{ height: 220 }} />
                </ChartCard>
              </Col>
              <Col xs={24} md={8}>
                <ChartCard title="网络包" iconColor="#722ed1">
                  <ReactECharts option={charts.netPkts} style={{ height: 220 }} />
                </ChartCard>
              </Col>
            </Row>
          )}

          {/* GPU & 温度（资产快照，选中主机即渲染） */}
          <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
            <Col xs={24} md={12}>
              <ChartCard title="GPU 信息" iconColor="#52c41a" loading={assetLoading} onRefresh={loadAssets}>
                <GpuInfo gpus={latestAsset?.gpus || []} />
              </ChartCard>
            </Col>
            <Col xs={24} md={12}>
              <ChartCard title="温度传感器" iconColor="#faad14" loading={assetLoading} onRefresh={loadAssets}>
                <TempInfo temps={latestAsset?.temperatures || []} />
              </ChartCard>
            </Col>
          </Row>
        </>
      )}
    </div>
  );
}
