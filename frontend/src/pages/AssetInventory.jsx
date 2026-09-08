import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Table, Tag, Input, Button, Space, Spin, Typography, Tabs } from 'antd';
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';
import { formatBytes } from '../utils/format';

const { Text } = Typography;

export default function AssetInventory() {
  const { uuid } = useParams();
  const [activeKey, setActiveKey] = useState('processes');
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState('');
  const [host, setHost] = useState(null);

  const loadHost = async () => {
    try {
      const resp = await client.get()(`/hosts/${uuid}`);
      setHost(resp);
    } catch (e) {}
  };

  const loadData = async () => {
    setLoading(true);
    try {
      let resp;
      if (activeKey === 'processes') {
        resp = await client.get()(`/hosts/${uuid}/assets/processes?limit=200`);
        setData(resp.processes || []);
        setTotal(resp.total || 0);
      } else if (activeKey === 'ports') {
        resp = await client.get()(`/hosts/${uuid}/assets/ports?limit=500`);
        setData(resp.ports || []);
        setTotal(resp.total || 0);
      } else {
        const params = keyword ? `?keyword=${encodeURIComponent(keyword)}&limit=500` : '?limit=500';
        resp = await client.get()(`/hosts/${uuid}/assets/packages${params}`);
        setData(resp.packages || []);
        setTotal(resp.total || 0);
      }
    } catch (e) {
      setData([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadHost(); }, [uuid]);
  useEffect(() => { loadData(); }, [activeKey, uuid]);

  const processColumns = [
    { title: 'PID', dataIndex: 'pid', key: 'pid', width: 80 },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '用户', dataIndex: 'username', key: 'username', width: 100 },
    { title: 'CPU%', dataIndex: 'cpu_percent', key: 'cpu_percent', width: 90, render: (v) => v?.toFixed(1) },
    { title: '内存%', dataIndex: 'mem_percent', key: 'mem_percent', width: 90, render: (v) => v?.toFixed(1) },
    { title: '采集时间', dataIndex: 'collected_at', key: 'collected_at', width: 160, render: (t) => dayjs(t).format('MM-DD HH:mm:ss') },
  ];

  const portColumns = [
    { title: '协议', dataIndex: 'type', key: 'type', width: 70, render: (v) => <Tag color={v === 'tcp' ? 'blue' : 'cyan'}>{v.toUpperCase()}</Tag> },
    { title: '地址族', dataIndex: 'family', key: 'family', width: 80, render: (v) => <Tag>{v}</Tag> },
    { title: '本地地址', dataIndex: 'local_addr', key: 'local_addr', width: 160 },
    { title: '端口', dataIndex: 'local_port', key: 'local_port', width: 80, render: (v) => <Text code>{v}</Text> },
    { title: '状态', dataIndex: 'state', key: 'state', width: 100, render: (v) => v === 'LISTEN' ? <Tag color="green">监听</Tag> : <Tag>{v || '-'}</Tag> },
    { title: 'PID', dataIndex: 'pid', key: 'pid', width: 80, render: (v) => v > 0 ? v : '-' },
    { title: '采集时间', dataIndex: 'collected_at', key: 'collected_at', width: 160, render: (t) => dayjs(t).format('MM-DD HH:mm:ss') },
  ];

  const packageColumns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '版本', dataIndex: 'version', key: 'version', width: 140 },
    { title: '大小', dataIndex: 'size_bytes', key: 'size', width: 100, render: formatBytes },
    { title: '来源', dataIndex: 'source', key: 'source', width: 80, render: (v) => <Tag color="geekblue">{v || '-'}</Tag> },
    { title: '采集时间', dataIndex: 'collected_at', key: 'collected_at', width: 160, render: (t) => dayjs(t).format('MM-DD HH:mm:ss') },
  ];

  const tabItems = [
    {
      key: 'processes',
      label: `进程（${total}）`,
      children: (
        <Spin spinning={loading}>
          <Space direction="vertical" style={{ width: '100%' }} size={8}>
            <Button icon={<ReloadOutlined />} onClick={loadData}>刷新</Button>
            <Table columns={processColumns} dataSource={data} rowKey="pid" size="small"
              pagination={{ pageSize: 50, total }} scroll={{ x: 600 }} />
          </Space>
        </Spin>
      ),
    },
    {
      key: 'ports',
      label: `端口（${total}）`,
      children: (
        <Spin spinning={loading}>
          <Space direction="vertical" style={{ width: '100%' }} size={8}>
            <Button icon={<ReloadOutlined />} onClick={loadData}>刷新</Button>
            <Table columns={portColumns} dataSource={data} rowKey={(r) => `${r.local_port}-${r.type}`} size="small"
              pagination={{ pageSize: 50, total }} scroll={{ x: 700 }} />
          </Space>
        </Spin>
      ),
    },
    {
      key: 'packages',
      label: `软件包（${total}）`,
      children: (
        <Spin spinning={loading}>
          <Space direction="vertical" style={{ width: '100%' }} size={8}>
            <Space>
              <Button icon={<ReloadOutlined />} onClick={loadData}>刷新</Button>
              <Input prefix={<SearchOutlined />} placeholder="搜索名称或版本..." value={keyword}
                onChange={(e) => setKeyword(e.target.value)} style={{ width: 280 }}
                onPressEnter={loadData} allowClear />
            </Space>
            <Table columns={packageColumns} dataSource={data} rowKey={(r) => `${r.name}-${r.version}`} size="small"
              pagination={{ pageSize: 50, total }} scroll={{ x: 600 }} />
          </Space>
        </Spin>
      ),
    },
  ];

  if (!host) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 16 }}>
        <Space>
          <Text type="secondary" style={{ fontSize: 13 }}>← </Text>
          <div className="page-title">{host.hostname} — 资产清单</div>
        </Space>
        <Text type="secondary" style={{ fontSize: 12 }}>UUID: {host.uuid} &nbsp;|&nbsp; {host.ip}</Text>
      </div>
      <Tabs activeKey={activeKey} onChange={setActiveKey} items={tabItems} size="large" />
    </div>
  );
}
