import React, { useEffect, useState, useCallback } from 'react';
import { Table, Button, Input, Space, Tag, Select, Popconfirm, message, Modal, Form, Typography, Card } from 'antd';
import { ReloadOutlined, PlusOutlined, DesktopOutlined, SearchOutlined, DownloadOutlined, DeleteOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import client from '../api/client';

const { Text } = Typography;

function exportCSV(rows, columns, filename) {
  const header = columns.map((c) => c.title).join(',');
  const lines = rows.map((r) => columns.map((c) => {
    const v = r[c.dataIndex];
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

const statusColor = { online: 'green', offline: 'red', pending: 'gold' };

const osOptions = [
  { label: 'Windows', options: [
    { value: 'Windows Server 2022', label: 'Windows Server 2022' },
    { value: 'Windows Server 2019', label: 'Windows Server 2019' },
    { value: 'Windows Server 2016', label: 'Windows Server 2016' },
    { value: 'Windows 11', label: 'Windows 11' },
    { value: 'Windows 10', label: 'Windows 10' },
  ]},
  { label: 'Linux', options: [
    { value: 'Ubuntu Server', label: 'Ubuntu Server' },
    { value: 'Debian', label: 'Debian' },
    { value: 'CentOS', label: 'CentOS' },
    { value: 'Rocky Linux', label: 'Rocky Linux' },
    { value: 'AlmaLinux', label: 'AlmaLinux' },
    { value: 'openSUSE', label: 'openSUSE' },
    { value: 'Arch Linux', label: 'Arch Linux' },
  ]},
  { label: '其他', options: [
    { value: 'macOS', label: 'macOS' },
    { value: 'ESXi', label: 'VMware ESXi' },
    { value: 'FreeBSD', label: 'FreeBSD' },
    { value: 'Other', label: '其他' },
  ]},
];

function formatBytes(bytes) {
  if (!bytes) return '-';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export default function HostList() {
  const navigate = useNavigate();
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState('');
  const [osFilter, setOsFilter] = useState('');
  const [createOpen, setCreateOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [createdToken, setCreatedToken] = useState(null);
  const [selectedIDs, setSelectedIDs] = useState([]);
  const [form] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await client.get()('/hosts', {
        params: { page, page_size: pageSize, keyword, status, os: osFilter },
      });
      setData(resp.data || []);
      setTotal(resp.total || 0);
    } catch (e) {
      setData([]);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, keyword, status, osFilter]);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      const resp = await client.post()('/hosts', values);
      message.success('主机已添加');
      setCreatedToken(resp.agent_token);
      form.resetFields();
      load();
    } catch (e) {
      if (e.errorFields) return;
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (uuid) => {
    try {
      await client.delete()(`/hosts/${uuid}`);
      message.success('已删除');
      load();
    } catch (e) {}
  };

  const handleBatchDelete = async () => {
    try {
      await client.delete()('/hosts', { data: { ids: selectedIDs } });
      message.success(`已删除 ${selectedIDs.length} 台主机`);
      setSelectedIDs([]);
      load();
    } catch (e) {}
  };

  const columns = [
    {
      title: '主机名', dataIndex: 'hostname', key: 'hostname',
      render: (t, r) => (
        <Button type="link" style={{ fontWeight: 600 }} onClick={() => navigate(`/hosts/${r.uuid}`)}>{t || '-'}</Button>
      ),
    },
    {
      title: '操作系统', dataIndex: 'os', key: 'os',
      render: (v, r) => <Text style={{ fontSize: 12 }}>{v || '-'} {r.os_version ? `(${r.os_version})` : ''}</Text>,
    },
    { title: 'IP', dataIndex: 'ip', key: 'ip', render: (v) => <Text code style={{ fontSize: 12 }}>{v || '-'}</Text> },
    {
      title: 'CPU', dataIndex: 'cpu_model', key: 'cpu_model',
      render: (v, r) => v ? `${v} (${r.cpu_cores}核)` : '-',
    },
    { title: '内存', dataIndex: 'memory_total', key: 'memory_total', render: formatBytes },
    { title: '磁盘', dataIndex: 'disk_total', key: 'disk_total', render: formatBytes },
    {
      title: '状态', dataIndex: 'status', key: 'status',
      render: (s) => {
        const dotClass = s === 'online' ? 'status-dot online' : s === 'pending' ? 'status-dot pending' : 'status-dot offline';
        const label = s === 'online' ? '在线' : s === 'pending' ? '待接入' : '离线';
        return <span className={dotClass}>{label}</span>;
      },
    },
    {
      title: '操作', key: 'action',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<DesktopOutlined />} onClick={() => navigate(`/hosts/${record.uuid}`)}>详情</Button>
          <Popconfirm title="确认删除该主机？" onConfirm={() => handleDelete(record.uuid)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 20 }}>
        <div>
          <div className="page-title">主机资产列表</div>
          <Text type="secondary" style={{ fontSize: 13 }}>共 {total} 台主机</Text>
        </div>
      </div>

      {/* Filter bar */}
      <Card className="glass glass-card" style={{ marginBottom: 16, borderRadius: 16, border: 'none' }}>
        <Space wrap size={12}>
          <Input.Search
            placeholder="搜索主机名或IP"
            allowClear
            style={{ width: 220 }}
            prefix={<SearchOutlined style={{ color: 'rgba(0,0,0,0.3)' }} />}
            onSearch={(v) => { setKeyword(v); setPage(1); }}
            onChange={(e) => { if (e.target.value === '') { setKeyword(''); setPage(1); } }}
          />
          <Space size={4}>
            {['', 'online', 'offline'].map((s) => (
              <Tag.CheckableTag
                key={s}
                checked={status === s}
                onChange={() => { setStatus(s); setPage(1); }}
                style={{ borderRadius: 6, border: '1px solid', borderColor: status === s ? (s === 'online' ? '#52c41a' : '#ff4d4f') : 'rgba(0,0,0,0.15)' }}
              >
                {s === '' ? '全部' : s === 'online' ? '在线' : '离线'}
              </Tag.CheckableTag>
            ))}
          </Space>
          <Select
            placeholder="操作系统"
            allowClear
            showSearch
            optionFilterProp="label"
            style={{ width: 200 }}
            value={osFilter || undefined}
            onChange={(v) => { setOsFilter(v || ''); setPage(1); }}
            options={osOptions}
            suffixIcon={<DesktopOutlined />}
          />
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>刷新</Button>
          <Button icon={<DownloadOutlined />} onClick={() => exportCSV(data, [
            { title: '主机名', dataIndex: 'hostname' },
            { title: '操作系统', dataIndex: 'os' },
            { title: 'IP', dataIndex: 'ip' },
            { title: 'CPU', dataIndex: 'cpu_model' },
            { title: '内存', dataIndex: 'memory_total' },
            { title: '磁盘', dataIndex: 'disk_total' },
            { title: '状态', dataIndex: 'status' },
          ], 'hosts.csv')}>导出</Button>
          {selectedIDs.length > 0 && (
            <Popconfirm title={`确认删除选中的 ${selectedIDs.length} 台主机？`} onConfirm={handleBatchDelete}>
              <Button danger icon={<DeleteOutlined />}>批量删除 ({selectedIDs.length})</Button>
            </Popconfirm>
          )}
          <Button type="primary" icon={<PlusOutlined />} onClick={() => { setCreateOpen(true); setCreatedToken(null); }} style={{ marginLeft: 'auto', background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none' }}>
            添加主机
          </Button>
        </Space>
      </Card>

      {/* Table */}
      <Card className="glass glass-card" style={{ borderRadius: 16, border: 'none' }}>
        <Table
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          scroll={{ x: 'max-content' }}
          size="middle"
          rowSelection={{
            onChange: (_, rows) => setSelectedIDs(rows.map((r) => r.id)),
            preserveSelectedRowKeys: false,
          }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (t) => `共 ${t} 台`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps); },
          }}
          locale={{ emptyText: <Text type="secondary">暂无主机数据</Text> }}
        />
      </Card>

      <Modal
        title={
          <Space>
            <PlusOutlined style={{ color: '#667eea' }} />
            <span>添加主机</span>
          </Space>
        }
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={handleCreate}
        confirmLoading={submitting}
        okText="添加"
        okButtonProps={{ disabled: !!createdToken }}
        styles={{ body: { paddingTop: 16 } }}
      >
        {createdToken ? (
          <div style={{ padding: '8px 0' }}>
            <Text type="secondary">主机已创建，请在目标机器上使用以下 Agent Token 注册：</Text>
            <div style={{ marginTop: 12, padding: '12px 16px', background: '#f6f8fa', borderRadius: 8, border: '1px solid #e8e8e8' }}>
              <Text copyable code style={{ fontSize: 13, wordBreak: 'break-all' }}>
                {createdToken}
              </Text>
            </div>
          </div>
        ) : (
          <Form form={form} layout="vertical">
            <Form.Item name="hostname" label="主机名" rules={[{ required: true, message: '请输入主机名' }]}>
              <Input placeholder="例如：web-01" />
            </Form.Item>
            <Form.Item name="ip" label="IP">
              <Input placeholder="可选" />
            </Form.Item>
            <Form.Item name="os" label="操作系统">
              <Select showSearch placeholder="选择操作系统" optionFilterProp="label" options={osOptions} />
            </Form.Item>
            <Form.Item name="arch" label="架构">
              <Input placeholder="例如：amd64" />
            </Form.Item>
          </Form>
        )}
      </Modal>
    </div>
  );
}
