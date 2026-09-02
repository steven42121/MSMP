import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Descriptions, Card, Spin, Tabs, Table, Tag, Button, Space, Input, Form, Popconfirm, message, Modal, Select, InputNumber, Typography, Switch } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import dayjs from 'dayjs';
import client from '../api/client';

function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '-';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function buildOption(title, data, field, unit) {
  const times = data.map((d) => dayjs(d.timestamp).format('HH:mm:ss'));
  return {
    title: { text: title, left: 'center', textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'axis' },
    grid: { left: 48, right: 24, top: 40, bottom: 40 },
    xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
    yAxis: { type: 'value', axisLabel: { formatter: `{value}${unit || ''}` } },
    series: [{ name: title, type: 'line', data: data.map((d) => d[field]), smooth: true, showSymbol: false, areaStyle: { opacity: 0.1 } }],
  };
}

export default function HostDetail() {
  const { uuid } = useParams();
  const [host, setHost] = useState(null);
  const [metrics, setMetrics] = useState([]);
  const [tags, setTags] = useState([]);
  const [events, setEvents] = useState([]);
  const [assets, setAssets] = useState([]);
  const [channels, setChannels] = useState([]);
  const [channelModal, setChannelModal] = useState(false);
  const [channelForm] = Form.useForm();
  const [loading, setLoading] = useState(true);
  const [activeKey, setActiveKey] = useState('info');
  const [tagForm] = Form.useForm();

  const loadHost = async () => {
    try {
      const resp = await client.get()(`/hosts/${uuid}`);
      setHost(resp);
    } catch (e) {
      setHost(null);
    } finally {
      setLoading(false);
    }
  };

  const loadMetrics = async () => {
    try {
      const resp = await client.get()(`/hosts/${uuid}/metrics`, { params: { limit: 120 } });
      setMetrics(Array.isArray(resp) ? resp : resp.data || []);
    } catch (e) {}
  };

  const loadTags = async () => {
    try {
      const resp = await client.get()(`/hosts/${uuid}/tags`);
      setTags(Array.isArray(resp) ? resp : resp.data || []);
    } catch (e) {}
  };

  const loadEvents = async () => {
    try {
      const resp = await client.get()(`/hosts/${uuid}/events`);
      setEvents(Array.isArray(resp) ? resp : resp.data || []);
    } catch (e) {}
  };

  const loadAssets = async () => {
    try {
      const resp = await client.get()(`/hosts/${uuid}/assets`);
      setAssets(Array.isArray(resp) ? resp : resp.data || []);
    } catch (e) {}
  };

  const loadChannels = async () => {
    try {
      const resp = await client.get()(`/hosts/${uuid}/channels`);
      setChannels(Array.isArray(resp) ? resp : resp.data || []);
    } catch (e) {}
  };

  useEffect(() => { loadHost(); }, [uuid]);

  // 切换到监控/资产/事件标签时自动加载
  useEffect(() => {
    if (activeKey === 'metrics') loadMetrics();
    if (activeKey === 'assets') loadAssets();
    if (activeKey === 'events') loadEvents();
    if (activeKey === 'channels') loadChannels();
    // tags 已在初始加载? 这里也刷新
    if (activeKey === 'tags') loadTags();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeKey, uuid]);

  const handleAddTag = async () => {
    try {
      const values = await tagForm.validateFields();
      await client.post()(`/hosts/${uuid}/tags`, values);
      message.success('标签已添加');
      tagForm.resetFields();
      loadTags();
    } catch (e) {
      if (e.errorFields) return;
    }
  };

  const handleDeleteTag = async (id) => {
    try {
      await client.delete()(`/hosts/${uuid}/tags/${id}`);
      message.success('已删除');
      loadTags();
    } catch (e) {}
  };

  const handleCreateChannel = async () => {
    try {
      const values = await channelForm.validateFields();
      await client.post()(`/hosts/${uuid}/channels`, values);
      message.success('渠道已创建，正在自动探测');
      channelForm.resetFields();
      setChannelModal(false);
      loadChannels();
    } catch (e) {
      if (e.errorFields) return;
    }
  };

  const handleProbeChannel = async (id) => {
    try {
      const resp = await client.post()(`/channels/${id}/probe`);
      if (resp.ok) {
        message.success(`探测成功：${resp.os || ''} ${resp.host || ''}`);
      } else {
        message.error(`探测失败：${resp.error || ''} ${resp.detail || ''}`);
      }
      loadChannels();
    } catch (e) {
      message.error('探测请求失败');
    }
  };

  const handleToggleChannel = async (id, enabled) => {
    try {
      await client.put()(`/channels/${id}`, { enabled: !enabled });
      message.success(enabled ? '已禁用' : '已启用');
      loadChannels();
    } catch (e) {}
  };

  const handleDeleteChannel = async (id) => {
    try {
      await client.delete()(`/channels/${id}`);
      message.success('已删除');
      loadChannels();
    } catch (e) {}
  };

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  if (!host) return <div>主机不存在</div>;

  const tagColumns = [
    { title: '键', dataIndex: 'key', key: 'key' },
    { title: '值', dataIndex: 'value', key: 'value' },
    {
      title: '操作', key: 'action',
      render: (_, r) => (
        <Popconfirm title="确认删除标签？" onConfirm={() => handleDeleteTag(r.id)}>
          <Button type="link" danger>删除</Button>
        </Popconfirm>
      ),
    },
  ];

  const eventColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
    { title: '类型', dataIndex: 'type', key: 'type', render: (v) => <Tag>{v}</Tag> },
    { title: '级别', dataIndex: 'level', key: 'level', render: (v) => <Tag color={v === 'critical' ? 'red' : v === 'warning' ? 'orange' : 'blue'}>{v}</Tag> },
    { title: '消息', dataIndex: 'message', key: 'message', ellipsis: true },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', render: (t) => dayjs(t).format('YYYY-MM-DD HH:mm:ss') },
  ];

  const items = [
    {
      key: 'info',
      label: '基本信息',
      children: (
        <Descriptions bordered column={{ xs: 1, sm: 2 }}>
          <Descriptions.Item label="主机名">{host.hostname}</Descriptions.Item>
          <Descriptions.Item label="UUID">{host.uuid}</Descriptions.Item>
          <Descriptions.Item label="操作系统">{host.os} {host.os_version}</Descriptions.Item>
          <Descriptions.Item label="架构">{host.arch}</Descriptions.Item>
          <Descriptions.Item label="IP">{host.ip}</Descriptions.Item>
          <Descriptions.Item label="公网IP">{host.public_ip || '-'}</Descriptions.Item>
          <Descriptions.Item label="CPU">{host.cpu_model} ({host.cpu_cores}核)</Descriptions.Item>
          <Descriptions.Item label="内存">{formatBytes(host.memory_total)}</Descriptions.Item>
          <Descriptions.Item label="磁盘">{formatBytes(host.disk_total)}</Descriptions.Item>
          <Descriptions.Item label="Agent版本">{host.agent_version}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={host.status === 'online' ? 'green' : 'red'}>{host.status === 'online' ? '在线' : '离线'}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="最后心跳">{host.last_heartbeat ? dayjs(host.last_heartbeat).format('YYYY-MM-DD HH:mm:ss') : '-'}</Descriptions.Item>
        </Descriptions>
      ),
    },
    {
      key: 'metrics',
      label: '监控',
      children: metrics.length === 0 ? (
        <Button onClick={loadMetrics}>加载监控数据</Button>
      ) : (
        <Space direction="vertical" style={{ width: '100%' }} size={16}>
          <Button onClick={loadMetrics}>刷新</Button>
          <ReactECharts option={buildOption('CPU 使用率', metrics, 'cpu_percent', '%')} style={{ height: 280 }} />
          <ReactECharts option={buildOption('内存使用率', metrics, 'mem_percent', '%')} style={{ height: 280 }} />
          <ReactECharts option={buildOption('系统负载', metrics, 'load1', '')} style={{ height: 280 }} />
        </Space>
      ),
    },
    {
      key: 'tags',
      label: '标签',
      children: (
        <Space direction="vertical" style={{ width: '100%' }} size={16}>
          <Form form={tagForm} layout="inline" onFinish={handleAddTag}>
            <Form.Item name="key" rules={[{ required: true, message: '键必填' }]}>
              <Input placeholder="键" />
            </Form.Item>
            <Form.Item name="value">
              <Input placeholder="值" />
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" icon={<PlusOutlined />}>添加</Button>
            </Form.Item>
            <Form.Item>
              <Button onClick={loadTags}>刷新</Button>
            </Form.Item>
          </Form>
          <Table columns={tagColumns} dataSource={tags} rowKey="id" size="small" pagination={false} />
        </Space>
      ),
    },
    {
      key: 'assets',
      label: '资产明细',
      children: (() => {
        const latest = assets[0];
        let detail = null;
        if (latest) {
          try { detail = JSON.parse(latest.payload); } catch (e) {}
        }
        const diskCols = [
          { title: '设备', dataIndex: 'device', key: 'device' },
          { title: '挂载点', dataIndex: 'mountpoint', key: 'mountpoint' },
          { title: '类型', dataIndex: 'fstype', key: 'fstype' },
          { title: '总量', dataIndex: 'total', key: 'total', render: formatBytes },
          { title: '已用', dataIndex: 'used', key: 'used', render: formatBytes },
          { title: '使用率', dataIndex: 'used_percent', key: 'used_percent', render: (v) => `${v?.toFixed(1)}%` },
        ];
        const netCols = [
          { title: '网卡', dataIndex: 'name', key: 'name' },
          { title: 'IP', dataIndex: 'ip', key: 'ip' },
          { title: 'MAC', dataIndex: 'mac', key: 'mac' },
          { title: '入站', dataIndex: 'bytes_recv', key: 'bytes_recv', render: formatBytes },
          { title: '出站', dataIndex: 'bytes_sent', key: 'bytes_sent', render: formatBytes },
        ];
        const procCols = [
          { title: 'PID', dataIndex: 'pid', key: 'pid' },
          { title: '名称', dataIndex: 'name', key: 'name' },
          { title: '用户', dataIndex: 'username', key: 'username' },
          { title: 'CPU%', dataIndex: 'cpu_percent', key: 'cpu_percent', render: (v) => v?.toFixed(1) },
          { title: '内存%', dataIndex: 'mem_percent', key: 'mem_percent', render: (v) => v?.toFixed(1) },
        ];
        return (
          <Space direction="vertical" style={{ width: '100%' }} size={16}>
            <Button onClick={loadAssets}>刷新资产快照</Button>
            {assets.length === 0 ? (
              <span>暂无资产快照</span>
            ) : (
              <>
                <div style={{ color: '#888' }}>最近快照：{dayjs(latest.created_at).format('YYYY-MM-DD HH:mm:ss')}</div>
                {detail?.disk_partitions && (
                  <Table title="磁盘分区" columns={diskCols} dataSource={detail.disk_partitions} rowKey="mountpoint" size="small" pagination={false} />
                )}
                {detail?.network_interfaces && (
                  <Table title="网络接口" columns={netCols} dataSource={detail.network_interfaces} rowKey="name" size="small" pagination={false} />
                )}
                {detail?.processes && (
                  <Table title="进程（前 50）" columns={procCols} dataSource={detail.processes} rowKey="pid" size="small" pagination={{ pageSize: 10 }} />
                )}
              </>
            )}
          </Space>
        );
      })(),
    },
    {
      key: 'channels',
      label: '采集渠道',
      children: (
        <Space direction="vertical" style={{ width: '100%' }} size={16}>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setChannelModal(true)}>添加渠道</Button>
            <Button onClick={loadChannels}>刷新</Button>
          </Space>
          {channels.length === 0 ? (
            <Typography.Text type="secondary">
              暂无采集渠道。当主机无法安装 Agent 时，可配置 SSH / Windows Admin Center / 宝塔面板 / Prometheus / SNMP / WinRM 渠道远程采集指标。
            </Typography.Text>
          ) : (
            <Table
              size="small"
              rowKey="id"
              dataSource={channels}
              pagination={false}
              columns={[
                { title: '类型', dataIndex: 'type', key: 'type', width: 130, render: (v) => <Tag color={v === 'ssh' ? 'blue' : v === 'wac' ? 'purple' : v === 'baota' ? 'green' : v === 'prometheus' ? 'cyan' : v === 'snmp' ? 'orange' : 'magenta'}>{v}</Tag> },
                { title: '地址', dataIndex: 'address', key: 'address' },
                { title: '接入方式', dataIndex: 'auth_mode', key: 'auth_mode', width: 120 },
                { title: '优先级', dataIndex: 'priority', key: 'priority', width: 80 },
                {
                  title: '状态', dataIndex: 'last_status', key: 'last_status', width: 110,
                  render: (v) => {
                    const map = { ok: ['green', '正常'], unreachable: ['red', '不可达'], auth_failed: ['orange', '认证失败'], denied: ['volcano', '权限不足'], unsupported: ['default', '不兼容'], parse_error: ['gold', '解析错误'] };
                    const m = map[v] || ['default', v || '未探测'];
                    return <Tag color={m[0]}>{m[1]}</Tag>;
                  },
                },
                { title: '最近探测', dataIndex: 'last_probe_at', key: 'last_probe_at', width: 170, render: (t) => t ? dayjs(t).format('MM-DD HH:mm:ss') : '-' },
                { title: '失败次数', dataIndex: 'fail_count', key: 'fail_count', width: 90 },
                { title: '启用', dataIndex: 'enabled', key: 'enabled', width: 70, render: (v, r) => <Switch checked={v} size="small" onChange={() => handleToggleChannel(r.id, v)} /> },
                {
                  title: '操作', key: 'action', width: 140,
                  render: (_, r) => (
                    <Space size={0}>
                      <Button type="link" size="small" onClick={() => handleProbeChannel(r.id)}>探测</Button>
                      <Popconfirm title="确认删除该渠道？" onConfirm={() => handleDeleteChannel(r.id)}>
                        <Button type="link" danger size="small">删除</Button>
                      </Popconfirm>
                    </Space>
                  ),
                },
              ]}
            />
          )}
        </Space>
      ),
    },
    {
      key: 'events',
      label: '事件',
      children: (
        <Space direction="vertical" style={{ width: '100%' }} size={16}>
          <Button onClick={loadEvents}>加载事件</Button>
          <Table columns={eventColumns} dataSource={events} rowKey="id" size="small" pagination={{ pageSize: 20 }} />
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 16 }}>
        <Space>
          <Button type="link" style={{ padding: 0, fontSize: 13, color: 'rgba(0,0,0,0.45)' }} onClick={() => navigate(-1)}>← 返回</Button>
          <div className="page-title">{host.hostname}</div>
          <Tag color={host.status === 'online' ? 'green' : 'red'} style={{ borderRadius: 20, padding: '2px 12px', fontWeight: 600 }}>
            {host.status === 'online' ? '在线' : '离线'}
          </Tag>
        </Space>
        <Text type="secondary" style={{ fontSize: 12 }}>UUID: {host.uuid} &nbsp;|&nbsp; {host.ip}</Text>
      </div>
      <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }} styles={{ body: { padding: 16 } }}>
        <Tabs
          activeKey={activeKey}
          onChange={setActiveKey}
          items={items}
          size="large"
          style={{ marginTop: -8 }}
        />
      </Card>

      <Modal
        title="添加采集渠道"
        open={channelModal}
        onOk={handleCreateChannel}
        onCancel={() => { setChannelModal(false); channelForm.resetFields(); }}
        okText="创建并探测"
        cancelText="取消"
        width={520}
      >
        <Form form={channelForm} layout="vertical" initialValues={{ type: 'ssh', auth_mode: 'password', priority: 100 }}>
          <Form.Item name="type" label="渠道类型" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'ssh', label: 'SSH（Linux 远程命令）' },
                { value: 'wac', label: 'Windows Admin Center' },
                { value: 'baota', label: '宝塔面板' },
                { value: 'prometheus', label: 'Prometheus / Node Exporter' },
                { value: 'snmp', label: 'SNMP（网络设备）' },
                { value: 'winrm', label: 'WinRM（Windows 远程管理）' },
              ]}
            />
          </Form.Item>
          <Form.Item name="address" label="地址" rules={[{ required: true, message: '地址必填' }]}>
              <Input placeholder="SSH: 10.0.0.1:22  /  其余: https://host:port 或 host:port" />
          </Form.Item>
          <Form.Item shouldUpdate noStyle>
            {(f) => f.getFieldValue('type') === 'ssh' ? (
              <Form.Item name="auth_mode" label="接入方式" rules={[{ required: true }]}>
                <Select
                  options={[
                    { value: 'password', label: '密码直连' },
                    { value: 'generated_key', label: '平台生成密钥对（推荐）' },
                    { value: 'private_key', label: '用户自带私钥' },
                  ]}
                />
              </Form.Item>
            ) : f.getFieldValue('type') === 'baota' ? (
              <Form.Item name="auth_mode" label="接入方式" rules={[{ required: true }]}>
                <Select
                  options={[
                    { value: 'api_key', label: 'API Key 直连' },
                    { value: 'gateway', label: '面板引导生成' },
                  ]}
                />
              </Form.Item>
            ) : f.getFieldValue('type') === 'snmp' ? (
              <Form.Item name="auth_mode" label="SNMP 版本" rules={[{ required: true }]}>
                <Select
                  options={[
                    { value: 'community', label: 'Community V2c' },
                    { value: 'v3', label: 'SNMPv3（用户认证）' },
                  ]}
                />
              </Form.Item>
            ) : f.getFieldValue('type') === 'prometheus' ? (
              <Form.Item name="auth_mode" label="认证方式" rules={[{ required: true }]}>
                <Select
                  options={[
                    { value: 'none', label: '无需认证' },
                    { value: 'basic', label: 'Basic Auth（用户名:密码）' },
                    { value: 'bearer', label: 'Bearer Token' },
                  ]}
                />
              </Form.Item>
            ) : f.getFieldValue('type') === 'winrm' ? (
              <Form.Item name="auth_mode" label="认证方式" rules={[{ required: true }]}>
                <Select
                  options={[
                    { value: 'basic', label: 'Basic 认证' },
                    { value: 'ntlm', label: 'NTLM 认证' },
                  ]}
                />
              </Form.Item>
            ) : (
              <Form.Item name="auth_mode" label="接入方式" rules={[{ required: true }]}>
                <Select options={[{ value: 'gateway', label: '网关凭据直连' }]} />
              </Form.Item>
            )}
          </Form.Item>
          <Form.Item name="username" label="用户名">
            <Input placeholder="SSH 登录用户 / 宝塔面板标识" />
          </Form.Item>
          <Form.Item name="secret" label="凭据（密码 / 私钥 / API Key）" rules={[{ required: true, message: '凭据必填' }]}>
            <Input.TextArea rows={3} placeholder="密码或私钥 PEM 内容或 API Key" />
          </Form.Item>
          <Form.Item name="priority" label="优先级（数字小优先）">
            <InputNumber min={1} max={999} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
