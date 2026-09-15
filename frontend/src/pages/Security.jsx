import React, { useEffect, useState } from 'react';
import { Table, Tag, Button, Space, Tabs, Select, Input, InputNumber, message, Typography, Empty, Alert } from 'antd';
import { ReloadOutlined, PlayCircleOutlined, UnlockOutlined, LockOutlined } from '@ant-design/icons';
import client from '../api/client';

const { Text } = Typography;

const levelColor = { critical: 'red', warning: 'orange' };
const baselineColor = { pass: 'green', fail: 'red', warn: 'orange', na: 'default' };
const baselineLabel = { pass: '通过', fail: '未通过', warn: '警告', na: '不适用' };

export default function Security() {
  const [hosts, setHosts] = useState([]);
  const [risks, setRisks] = useState([]);
  const [loading, setLoading] = useState(false);

  const [baseHost, setBaseHost] = useState(null);
  const [baseItems, setBaseItems] = useState([]);
  const [baseRunning, setBaseRunning] = useState(false);

  const [fwHost, setFwHost] = useState(null);
  const [fwRules, setFwRules] = useState('');
  const [fwLoading, setFwLoading] = useState(false);
  const [fwPort, setFwPort] = useState(null);
  const [fwProto, setFwProto] = useState('tcp');
  const [fwBusy, setFwBusy] = useState(false);

  const loadHosts = async () => {
    try {
      const resp = await client.get()('/hosts');
      const arr = Array.isArray(resp) ? resp : (resp?.data || []);
      setHosts(arr);
    } catch (e) {}
  };

  const loadRisks = async () => {
    setLoading(true);
    try { setRisks(await client.get()('/security/risks') || []); }
    catch (e) { message.error('加载风险报告失败'); }
    finally { setLoading(false); }
  };

  useEffect(() => { loadHosts(); loadRisks(); }, []);

  const runBaseline = async () => {
    if (!baseHost) { message.warning('请先选择主机'); return; }
    setBaseRunning(true);
    try { setBaseItems(await client.post()(`/security/baseline/${baseHost.id}`) || []); }
    catch (e) { message.error(e?.response ? '基线检查失败' : '基线检查失败（需主机配置 SSH 渠道）'); setBaseItems([]); }
    finally { setBaseRunning(false); }
  };

  const loadFirewall = async () => {
    if (!fwHost) { message.warning('请先选择主机'); return; }
    setFwLoading(true);
    try {
      const resp = await client.get()(`/security/firewall/${fwHost.id}`);
      setFwRules(resp?.rules || '');
    } catch (e) { message.error('获取防火墙规则失败（需 SSH 渠道）'); setFwRules(''); }
    finally { setFwLoading(false); }
  };

  const doFirewall = async (action) => {
    if (!fwHost) { message.warning('请先选择主机'); return; }
    if (!fwPort) { message.warning('请输入端口'); return; }
    setFwBusy(true);
    try {
      const resp = await client.post()(`/security/firewall/${fwHost.id}`, { action, port: fwPort, proto: fwProto });
      message.success('操作成功');
      loadFirewall();
    } catch (e) { message.error('操作失败'); }
    finally { setFwBusy(false); }
  };

  const riskColumns = [
    { title: '主机', dataIndex: 'hostname', key: 'hostname', render: (v, r) => <><Text strong>{v}</Text>{r.public_ip && <div style={{ fontSize: 12, color: '#888' }}>{r.public_ip}</div>}</> },
    { title: '端口', key: 'port', width: 120, render: (_, r) => `${r.port}/${r.proto}` },
    { title: '服务', dataIndex: 'service', key: 'service', width: 140 },
    { title: '等级', dataIndex: 'level', key: 'level', width: 100, render: (v) => <Tag color={levelColor[v]}>{v === 'critical' ? '严重' : '警告'}</Tag> },
    { title: '原因', dataIndex: 'reason', key: 'reason', ellipsis: true },
    { title: '加固建议', dataIndex: 'suggestion', key: 'suggestion', ellipsis: true },
  ];

  const baselineColumns = [
    { title: '检查项', dataIndex: 'name', key: 'name', width: 200 },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100, render: (v) => <Tag color={baselineColor[v]}>{baselineLabel[v] || v}</Tag> },
    { title: '详情', dataIndex: 'detail', key: 'detail', ellipsis: true },
    { title: '建议', dataIndex: 'suggestion', key: 'suggestion', ellipsis: true },
  ];

  const hostOptions = hosts.map((h) => ({ value: h.id, label: `${h.hostname}${h.ip ? ' (' + h.ip + ')' : ''}` }));

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 16 }}>
        <div className="page-title">网络安全</div>
        <Button icon={<ReloadOutlined />} onClick={loadRisks} loading={loading}>刷新风险</Button>
      </div>

      <Tabs
        items={[
          {
            key: 'risks',
            label: `端口暴露风险（${risks.length}）`,
            children: (
              risks.length === 0 ? (
                <Empty description="未发现端口暴露风险" style={{ padding: 40 }} />
              ) : (
                <Table rowKey={(r) => `${r.host_id}-${r.port}-${r.proto}`} dataSource={risks} columns={riskColumns} pagination={false} size="small" />
              )
            ),
          },
          {
            key: 'baseline',
            label: '安全基线检查',
            children: (
              <div>
                <Space style={{ marginBottom: 16 }}>
                  <Select style={{ width: 280 }} placeholder="选择主机" options={hostOptions} value={baseHost?.id} onChange={(v) => setBaseHost(hosts.find((h) => h.id === v))} />
                  <Button type="primary" icon={<PlayCircleOutlined />} onClick={runBaseline} loading={baseRunning}>执行检查</Button>
                </Space>
                {baseItems.length > 0 && (
                  <>
                    <Alert style={{ marginBottom: 12 }} type={baseItems.some((i) => i.status === 'fail') ? 'warning' : 'success'} showIcon
                      message={`通过 ${baseItems.filter((i) => i.status === 'pass').length} / ${baseItems.length} 项`} />
                    <Table rowKey="name" dataSource={baseItems} columns={baselineColumns} pagination={false} size="small" />
                  </>
                )}
                {baseItems.length === 0 && <Empty description="选择主机后执行基线检查" />}
              </div>
            ),
          },
          {
            key: 'firewall',
            label: '防火墙管理',
            children: (
              <div>
                <Space style={{ marginBottom: 16 }}>
                  <Select style={{ width: 280 }} placeholder="选择主机" options={hostOptions} value={fwHost?.id} onChange={(v) => setFwHost(hosts.find((h) => h.id === v))} />
                  <Button onClick={loadFirewall} loading={fwLoading}>查看规则</Button>
                </Space>
                {fwRules && <pre style={{ background: '#f6f6f6', padding: 12, borderRadius: 6, whiteSpace: 'pre-wrap' }}>{fwRules}</pre>}
                <Space style={{ marginTop: 16 }}>
                  <InputNumber min={1} max={65535} placeholder="端口" value={fwPort} onChange={setFwPort} style={{ width: 120 }} />
                  <Select style={{ width: 90 }} value={fwProto} onChange={setFwProto} options={[{ value: 'tcp', label: 'TCP' }, { value: 'udp', label: 'UDP' }]} />
                  <Button type="primary" icon={<UnlockOutlined />} onClick={() => doFirewall('open')} loading={fwBusy}>放行</Button>
                  <Button danger icon={<LockOutlined />} onClick={() => doFirewall('close')} loading={fwBusy}>封禁</Button>
                </Space>
              </div>
            ),
          },
        ]}
      />
    </div>
  );
}