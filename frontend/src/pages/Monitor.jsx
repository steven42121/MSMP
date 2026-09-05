import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { Card, Select, Space, Spin, Empty, Row, Col, Button, Typography } from 'antd';
import { LineChartOutlined, DownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import dayjs from 'dayjs';
import client from '../api/client';

const { Text } = Typography;

const durations = [
  { value: '30m', label: '最近 30 分钟' },
  { value: '1h', label: '最近 1 小时' },
  { value: '3h', label: '最近 3 小时' },
  { value: '6h', label: '最近 6 小时' },
  { value: '12h', label: '最近 12 小时' },
  { value: '24h', label: '最近 24 小时' },
  { value: '7d', label: '最近 7 天' },
  { value: '30d', label: '最近 30 天' },
];

const metricColors = ['#667eea', '#764ba2', '#52c41a', '#faad14', '#ff4d4f', '#1890ff'];

function buildOption(title, data, field, unit) {
  const times = data.map((d) => dayjs(d.timestamp).format('HH:mm:ss'));
  const values = data.map((d) => d[field]);
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
      name: title, type: 'line', data: values, smooth: true,
      showSymbol: false,
      areaStyle: { opacity: 0.12 },
      itemStyle: { color: metricColors[0] },
      lineStyle: { width: 2.5 },
    }],
  };
}

function formatNet(v) {
  if (!v) return '0 B/s';
  if (v >= 1e6) return (v / 1e6).toFixed(2) + ' MB/s';
  if (v >= 1e3) return (v / 1e3).toFixed(2) + ' KB/s';
  return v.toFixed(0) + ' B/s';
}

export default function Monitor() {
  const [hosts, setHosts] = useState([]);
  const [hostUUID, setHostUUID] = useState();
  const [duration, setDuration] = useState('1h');
  const [metrics, setMetrics] = useState([]);
  const [loading, setLoading] = useState(false);
  const [lastUpdate, setLastUpdate] = useState(null);
  const [currentHost, setCurrentHost] = useState(null);
  const [assets, setAssets] = useState([]);
  const [assetLoading, setAssetLoading] = useState(false);

  useEffect(() => {
    client.get()('/hosts', { params: { page_size: 100 } })
      .then((resp) => {
        const data = resp.data || [];
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
      .then((resp) => {
        const data = Array.isArray(resp) ? resp : (resp.data || []);
        setAssets(data);
      })
      .catch(() => setAssets([]))
      .finally(() => setAssetLoading(false));
  }, [hostUUID]);

  useEffect(() => { loadAssets(); }, [loadAssets]);
  useEffect(() => {
    const timer = setInterval(loadAssets, 5 * 60 * 1000); // 5分钟刷新
    return () => clearInterval(timer);
  }, [loadAssets]);

  const loadMetrics = useCallback(() => {
    if (!hostUUID) return;
    setLoading(true);
    client.get()('/metrics', { params: { host_uuid: hostUUID, duration } })
      .then((resp) => { setMetrics(resp.data || []); setLastUpdate(new Date()); })
      .catch(() => setMetrics([]))
      .finally(() => setLoading(false));
  }, [hostUUID, duration]);

  useEffect(() => { loadMetrics(); }, [loadMetrics]);

  useEffect(() => {
    if (!hostUUID) return;
    const timer = setInterval(loadMetrics, 30000);
    return () => clearInterval(timer);
  }, [hostUUID, loadMetrics]);

  const cpuOption = useMemo(() => buildOption('CPU 使用率 (%)', metrics, 'cpu_percent', '%'), [metrics]);
  const memOption = useMemo(() => buildOption('内存使用率 (%)', metrics, 'mem_percent', '%'), [metrics]);
  const loadOption = useMemo(() => buildOption('系统负载 (1min)', metrics, 'load1', ''), [metrics]);
  const procOption = useMemo(() => buildOption('进程数', metrics, 'process_count', ''), [metrics]);
  const diskIOOption = useMemo(() => {
    const times = metrics.map((d) => dayjs(d.timestamp).format('HH:mm:ss'));
    return {
      title: { text: '磁盘 IO (累计)', left: 'center', textStyle: { fontSize: 13 } },
      tooltip: { trigger: 'axis' },
      legend: { data: ['读取', '写入'], bottom: 4 },
      grid: { left: 60, right: 24, top: 40, bottom: 40 },
      xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
      yAxis: { type: 'value', axisLabel: { formatter: formatBytes } },
      series: [
        { name: '读取', type: 'line', data: metrics.map((d) => d.disk_read_bytes), smooth: true, showSymbol: false, lineStyle: { width: 2 }, itemStyle: { color: '#52c41a' }, areaStyle: { opacity: 0.1 } },
        { name: '写入', type: 'line', data: metrics.map((d) => d.disk_write_bytes), smooth: true, showSymbol: false, lineStyle: { width: 2 }, itemStyle: { color: '#faad14' }, areaStyle: { opacity: 0.1 } },
      ],
    };
  }, [metrics]);
  const netPktsOption = useMemo(() => {
    const times = metrics.map((d) => dayjs(d.timestamp).format('HH:mm:ss'));
    return {
      title: { text: '网络包计数 (累计)', left: 'center', textStyle: { fontSize: 13 } },
      tooltip: { trigger: 'axis' },
      legend: { data: ['收包', '发包'], bottom: 4 },
      grid: { left: 60, right: 24, top: 40, bottom: 40 },
      xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
      yAxis: { type: 'value' },
      series: [
        { name: '收包', type: 'line', data: metrics.map((d) => d.net_pkts_recv), smooth: true, showSymbol: false, lineStyle: { width: 2 }, itemStyle: { color: '#1890ff' } },
        { name: '发包', type: 'line', data: metrics.map((d) => d.net_pkts_sent), smooth: true, showSymbol: false, lineStyle: { width: 2 }, itemStyle: { color: '#722ed1' } },
      ],
    };
  }, [metrics]);
  const netOption = useMemo(() => {
    const times = metrics.map((d) => dayjs(d.timestamp).format('HH:mm:ss'));
    return {
      title: { text: '网络流量', left: 'center', textStyle: { fontSize: 13 } },
      tooltip: {
        trigger: 'axis',
        formatter: (params) => {
          let s = `<span style="font-weight:600">${params[0].axisValue}</span><br/>`;
          params.forEach((p) => {
            s += `${p.marker}${p.seriesName}: <b>${formatNet(p.value)}</b><br/>`;
          });
          return s;
        },
      },
      legend: { data: ['入站', '出站'], bottom: 4, textStyle: { fontSize: 11 } },
      grid: { left: 60, right: 24, top: 40, bottom: 40 },
      xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
      yAxis: {
        type: 'value', axisLabel: { formatter: formatNet },
        splitLine: { lineStyle: { type: 'dashed', color: 'rgba(0,0,0,0.06)' } },
      },
      series: [
        { name: '入站', type: 'line', data: metrics.map((d) => d.net_rx_bps), smooth: true, showSymbol: false, lineStyle: { width: 2.5 }, itemStyle: { color: '#667eea' }, areaStyle: { opacity: 0.1 } },
        { name: '出站', type: 'line', data: metrics.map((d) => d.net_tx_bps), smooth: true, showSymbol: false, lineStyle: { width: 2.5 }, itemStyle: { color: '#764ba2' }, areaStyle: { opacity: 0.1 } },
      ],
    };
  }, [metrics]);

  const currentStats = useMemo(() => {
    if (!metrics.length) return null;
    const last = metrics[metrics.length - 1];
    return {
      cpu: last.cpu_percent?.toFixed(1) + '%',
      mem: last.mem_percent?.toFixed(1) + '%',
      swap: last.swap_used && last.swap_total
        ? ((last.swap_used / last.swap_total) * 100).toFixed(1) + '%'
        : '-',
      disk: last.disk_used && last.disk_total
        ? ((last.disk_used / last.disk_total) * 100).toFixed(1) + '%'
        : '-',
      load: last.load1?.toFixed(2) || '-',
      procs: last.process_count?.toString() || '-',
      diskR: formatBytes(last.disk_read_bytes),
      diskW: formatBytes(last.disk_write_bytes),
      rx: formatNet(last.net_rx_bps),
      tx: formatNet(last.net_tx_bps),
      pktsR: last.net_pkts_recv?.toLocaleString() || '-',
      pktsT: last.net_pkts_sent?.toLocaleString() || '-',
    };
  }, [metrics]);

  const handleExport = () => {
    if (!metrics.length) return;
    const header = 'timestamp,cpu_percent,mem_percent,mem_used,mem_total,disk_used,disk_total,net_rx_bps,net_tx_bps,load1\n';
    const lines = metrics.map((d) => [
      d.timestamp, d.cpu_percent, d.mem_percent, d.mem_used, d.mem_total,
      d.disk_used, d.disk_total, d.net_rx_bps, d.net_tx_bps, d.load1,
    ].join(',')).join('\n');
    const blob = new Blob([header + lines], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `metrics_${hostUUID || 'unknown'}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div>
      {/* Page header */}
      <div className="page-header" style={{ marginBottom: 24 }}>
        <div>
          <div className="page-title">监控</div>
          <Text type="secondary" style={{ fontSize: 13 }}>主机实时性能指标与趋势分析</Text>
        </div>
      </div>

      {/* Controls */}
      <Card className="liquid-glass" style={{ marginBottom: 16, borderRadius: 16, border: 'none' }}>
        <Space wrap className="filter-bar">
          <Select
            showSearch
            style={{ width: 320 }}
            placeholder="选择主机"
            optionFilterProp="label"
            value={hostUUID}
            onChange={(v) => {
              setHostUUID(v);
              setMetrics([]);
              setCurrentHost(hosts.find(h => h.uuid === v) || null);
            }}
            options={hosts.map((h) => ({
              value: h.uuid,
              label: `${h.hostname || '未知主机'} (${h.ip || '-'})`,
            }))}
          />
          <Select
            style={{ width: 160 }}
            value={duration}
            onChange={setDuration}
            options={durations}
          />
          <Button onClick={loadMetrics} loading={loading} icon={<ReloadOutlined />} style={{ borderRadius: 8 }}>
            刷新
          </Button>
          <Button
            onClick={handleExport} disabled={!metrics.length}
            icon={<DownloadOutlined />} style={{ borderRadius: 8 }}
          >
            导出 CSV
          </Button>
          {lastUpdate && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              最后更新：{dayjs(lastUpdate).format('HH:mm:ss')}
            </Text>
          )}
        </Space>
      </Card>

      {/* Current stats bar */}
      {currentStats && metrics.length > 0 && (
        <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
          {[
            { label: 'CPU', value: currentStats.cpu, color: '#667eea' },
            { label: '内存', value: currentStats.mem, color: '#764ba2' },
            { label: '交换', value: currentStats.swap, color: '#faad14' },
            { label: '磁盘', value: currentStats.disk, color: '#52c41a' },
            { label: '进程', value: currentStats.procs, color: '#1890ff' },
            { label: '负载', value: currentStats.load, color: '#ff4d4f' },
            { label: '磁盘读', value: currentStats.diskR, color: '#a6ee3c' },
            { label: '磁盘写', value: currentStats.diskW, color: '#f9e2af' },
            { label: '入站', value: currentStats.rx, color: '#667eea' },
            { label: '出站', value: currentStats.tx, color: '#764ba2' },
            { label: '收包', value: currentStats.pktsR, color: '#1890ff' },
            { label: '发包', value: currentStats.pktsT, color: '#722ed1' },
          ].map((s) => (
            <Col key={s.label} xs={12} sm={8} md={4} lg={3}>
              <div style={{
                textAlign: 'center', padding: '8px 4px',
                background: s.color + '10', borderRadius: 8,
                border: `1px solid ${s.color}30`,
              }}>
                <div style={{ fontSize: 10, color: 'rgba(0,0,0,0.4)', marginBottom: 2 }}>{s.label}</div>
                <div style={{ fontSize: 14, fontWeight: 700, color: s.color, wordBreak: 'break-all' }}>{s.value}</div>
              </div>
            </Col>
          ))}
        </Row>
      )}

      {!hostUUID ? (
        <Empty description="请选择主机" style={{ margin: '80px 0' }} />
      ) : loading && metrics.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '100px 0' }}>
          <Spin size="large" />
        </div>
      ) : metrics.length === 0 ? (
        <Empty description="暂无监控数据" style={{ margin: '80px 0' }} />
       ) : (
         <>
         <Row gutter={[16, 16]}>
           <Col xs={24} md={12}>
             <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}
               title={<Space><LineChartOutlined style={{ color: '#667eea' }} /><span>CPU 使用率</span></Space>}
             >
               <ReactECharts option={cpuOption} style={{ height: 260 }} />
             </Card>
           </Col>
           <Col xs={24} md={12}>
             <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}
               title={<Space><LineChartOutlined style={{ color: '#764ba2' }} /><span>内存使用率</span></Space>}
             >
               <ReactECharts option={memOption} style={{ height: 260 }} />
             </Card>
           </Col>
           <Col xs={24} md={12}>
             <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}
               title={<Space><LineChartOutlined style={{ color: '#faad14' }} /><span>系统负载</span></Space>}
             >
               <ReactECharts option={loadOption} style={{ height: 260 }} />
             </Card>
           </Col>
           <Col xs={24} md={12}>
             <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}
               title={<Space><LineChartOutlined style={{ color: '#52c41a' }} /><span>网络流量</span></Space>}
             >
               <ReactECharts option={netOption} style={{ height: 260 }} />
             </Card>
           </Col>
         </Row>
         <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
           <Col xs={24} md={8}>
             <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}
               title={<Space><LineChartOutlined style={{ color: '#1890ff' }} /><span>进程数</span></Space>}
             >
               <ReactECharts option={procOption} style={{ height: 220 }} />
             </Card>
           </Col>
           <Col xs={24} md={8}>
             <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}
               title={<Space><LineChartOutlined style={{ color: '#faad14' }} /><span>磁盘 IO</span></Space>}
             >
               <ReactECharts option={diskIOOption} style={{ height: 220 }} />
             </Card>
           </Col>
           <Col xs={24} md={8}>
             <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}
               title={<Space><LineChartOutlined style={{ color: '#722ed1' }} /><span>网络包</span></Space>}
             >
               <ReactECharts option={netPktsOption} style={{ height: 220 }} />
             </Card>
           </Col>
         </Row>
          </>
        )}

        {/* GPU & Temperature Assets */}
        {hostUUID && (
          <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
            <Col xs={24} md={12}>
              <Card
                className="liquid-glass"
                style={{ borderRadius: 16, border: 'none' }}
                title={<Space><LineChartOutlined style={{ color: '#52c41a' }} /><span>GPU 信息</span></Space>}
                loading={assetLoading}
                extra={<Button size="small" onClick={loadAssets} loading={assetLoading}>刷新</Button>}
              >
                {assets.length === 0 ? (
                  <Empty description="暂无资产数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                ) : (
                  (() => {
                    const latest = assets[0];
                    const gpus = latest?.gpus || [];
                    if (gpus.length === 0) {
                      return <Text type="secondary">未检测到 GPU</Text>;
                    }
                    return (
                      <div style={{ maxHeight: 200, overflowY: 'auto' }}>
                        {gpus.map((gpu, idx) => (
                          <div key={idx} style={{
                            padding: '8px 12px',
                            background: 'rgba(82,196,26,0.08)',
                            borderRadius: 8,
                            marginBottom: 8,
                          }}>
                            <div style={{ fontWeight: 600, marginBottom: 4 }}>{gpu.name || `GPU ${idx + 1}`}</div>
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
                  })()
                )}
              </Card>
            </Col>
            <Col xs={24} md={12}>
              <Card
                className="liquid-glass"
                style={{ borderRadius: 16, border: 'none' }}
                title={<Space><LineChartOutlined style={{ color: '#faad14' }} /><span>温度传感器</span></Space>}
                loading={assetLoading}
                extra={<Button size="small" onClick={loadAssets} loading={assetLoading}>刷新</Button>}
              >
                {assets.length === 0 ? (
                  <Empty description="暂无资产数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                ) : (
                  (() => {
                    const latest = assets[0];
                    const temps = latest?.temperatures || [];
                    if (temps.length === 0) {
                      return <Text type="secondary">未检测到温度传感器</Text>;
                    }
                    return (
                      <div style={{ maxHeight: 200, overflowY: 'auto' }}>
                        {temps.map((t, idx) => (
                          <div key={idx} style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            padding: '6px 12px',
                            background: t.temp > (t.critical || 90) ? 'rgba(255,77,79,0.1)' :
                                       t.temp > (t.high || 70) ? 'rgba(250,173,20,0.1)' : 'rgba(82,196,26,0.08)',
                            borderRadius: 6,
                            marginBottom: 4,
                          }}>
                            <span style={{ fontSize: 13 }}>{t.sensor_key || `Sensor ${idx + 1}`}</span>
                            <span style={{
                              fontSize: 13,
                              fontWeight: 600,
                              color: t.temp > (t.critical || 90) ? '#ff4d4f' :
                                     t.temp > (t.high || 70) ? '#faad14' : '#52c41a',
                            }}>
                              {t.temp != null ? `${t.temp}°C` : '-'}
                            </span>
                          </div>
                        ))}
                      </div>
                    );
                  })()
                )}
              </Card>
            </Col>
          </Row>
        )}
     </div>
  );
}
