import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { Card, Select, Space, Spin, Empty, Row, Col, Button } from 'antd';
import ReactECharts from 'echarts-for-react';
import dayjs from 'dayjs';
import client from '../api/client';

const durations = [
  { value: '1h', label: '最近 1 小时' },
  { value: '3h', label: '最近 3 小时' },
  { value: '6h', label: '最近 6 小时' },
  { value: '24h', label: '最近 24 小时' },
];

function buildOption(title, data, field, unit) {
  const times = data.map((d) => dayjs(d.timestamp).format('HH:mm:ss'));
  const values = data.map((d) => d[field]);
  return {
    title: { text: title, left: 'center', textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'axis' },
    grid: { left: 48, right: 24, top: 40, bottom: 40 },
    xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
    yAxis: { type: 'value', axisLabel: { formatter: `{value}${unit || ''}` } },
    series: [{ name: title, type: 'line', data: values, smooth: true, showSymbol: false, areaStyle: { opacity: 0.1 } }],
  };
}

function formatNet(v) {
  if (v >= 1e6) return (v / 1e6).toFixed(2) + ' MB/s';
  if (v >= 1e3) return (v / 1e3).toFixed(2) + ' KB/s';
  return v + ' B/s';
}

export default function Monitor() {
  const [hosts, setHosts] = useState([]);
  const [hostUUID, setHostUUID] = useState();
  const [duration, setDuration] = useState('1h');
  const [metrics, setMetrics] = useState([]);
  const [loading, setLoading] = useState(false);
  const [lastUpdate, setLastUpdate] = useState(null);

  useEffect(() => {
    client.get('/hosts', { params: { page_size: 100 } })
      .then((resp) => {
        setHosts(resp.data || []);
        if (resp.data?.length) setHostUUID(resp.data[0].uuid);
      })
      .catch(() => setHosts([]));
  }, []);

  const loadMetrics = useCallback(() => {
    if (!hostUUID) return;
    setLoading(true);
    client.get('/metrics', { params: { host_uuid: hostUUID, duration } })
      .then((resp) => { setMetrics(resp.data || []); setLastUpdate(new Date()); })
      .catch(() => setMetrics([]))
      .finally(() => setLoading(false));
  }, [hostUUID, duration]);

  useEffect(() => { loadMetrics(); }, [loadMetrics]);

  // 自动刷新（30 秒）
  useEffect(() => {
    if (!hostUUID) return;
    const timer = setInterval(loadMetrics, 30000);
    return () => clearInterval(timer);
  }, [hostUUID, loadMetrics]);

  const cpuOption = useMemo(() => buildOption('CPU 使用率', metrics, 'cpu_percent', '%'), [metrics]);
  const memOption = useMemo(() => buildOption('内存使用率', metrics, 'mem_percent', '%'), [metrics]);
  const loadOption = useMemo(() => buildOption('系统负载', metrics, 'load1', ''), [metrics]);
  const netOption = useMemo(() => {
    const times = metrics.map((d) => dayjs(d.timestamp).format('HH:mm:ss'));
    return {
      title: { text: '网络流量', left: 'center', textStyle: { fontSize: 14 } },
      tooltip: { trigger: 'axis', formatter: (params) => {
        let s = params[0].axisValue + '<br/>';
        params.forEach((p) => { s += `${p.marker}${p.seriesName}: ${formatNet(p.value)}<br/>`; });
        return s;
      } },
      legend: { data: ['入站', '出站'], bottom: 0 },
      grid: { left: 60, right: 24, top: 40, bottom: 40 },
      xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
      yAxis: { type: 'value', axisLabel: { formatter: (v) => formatNet(v) } },
      series: [
        { name: '入站', type: 'line', data: metrics.map((d) => d.net_rx_bps), smooth: true, showSymbol: false },
        { name: '出站', type: 'line', data: metrics.map((d) => d.net_tx_bps), smooth: true, showSymbol: false },
      ],
    };
  }, [metrics]);

  return (
    <div>
      <h2>监控</h2>
      <Card style={{ marginBottom: 16 }}>
        <Space wrap>
          <Select
            showSearch
            style={{ width: 320 }}
            placeholder="选择主机"
            optionFilterProp="label"
            value={hostUUID}
            onChange={setHostUUID}
            options={hosts.map((h) => ({ value: h.uuid, label: `${h.hostname} (${h.ip})` }))}
          />
          <Select
            style={{ width: 160 }}
            value={duration}
            onChange={setDuration}
            options={durations}
          />
          <Button onClick={loadMetrics} loading={loading}>刷新</Button>
          {lastUpdate && (
            <span style={{ color: '#888', fontSize: 12 }}>
              最后更新：{dayjs(lastUpdate).format('HH:mm:ss')}
            </span>
          )}
        </Space>
      </Card>

      {!hostUUID ? (
        <Empty description="请选择主机" />
      ) : loading ? (
        <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />
      ) : metrics.length === 0 ? (
        <Empty description="暂无监控数据" />
      ) : (
        <Space direction="vertical" style={{ width: '100%' }} size={16}>
          <Button onClick={() => {
            const header = 'timestamp,cpu_percent,mem_percent,mem_used,mem_total,disk_used,disk_total,net_rx_bps,net_tx_bps,load1\n';
            const lines = metrics.map((d) => [
              d.timestamp, d.cpu_percent, d.mem_percent, d.mem_used, d.mem_total,
              d.disk_used, d.disk_total, d.net_rx_bps, d.net_tx_bps, d.load1,
            ].join(',')).join('\n');
            const blob = new Blob([header + lines], { type: 'text/csv;charset=utf-8;' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'metrics.csv';
            a.click();
            URL.revokeObjectURL(url);
          }}>导出监控数据 CSV</Button>
          <Row gutter={[16, 16]}>
            <Col xs={24} md={12}><Card><ReactECharts option={cpuOption} style={{ height: 280 }} /></Card></Col>
            <Col xs={24} md={12}><Card><ReactECharts option={memOption} style={{ height: 280 }} /></Card></Col>
            <Col xs={24} md={12}><Card><ReactECharts option={loadOption} style={{ height: 280 }} /></Card></Col>
            <Col xs={24} md={12}><Card><ReactECharts option={netOption} style={{ height: 280 }} /></Card></Col>
          </Row>
        </Space>
      )}
    </div>
  );
}
