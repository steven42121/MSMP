import React, { useEffect, useState, useCallback } from 'react';
import { Table, Button, Modal, Form, Input, InputNumber, Tag, Space, Card, Typography, Popconfirm, Select, message } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';

function linuxInstall(serverUrl, token) {
  return `export MSMP_SERVER_URL=${serverUrl}
export AGENT_TOKEN=${token}
export AGENT_UUID=$(hostname)
chmod +x msmp-agent && ./msmp-agent`;
}

function windowsInstall(serverUrl, token) {
  return `$env:MSMP_SERVER_URL="${serverUrl}"
$env:AGENT_TOKEN="${token}"
$env:AGENT_UUID=$env:COMPUTERNAME
.\\msmp-agent.exe`;
}

function macosInstall(serverUrl, token) {
  return `export MSMP_SERVER_URL=${serverUrl}
export AGENT_TOKEN=${token}
export AGENT_UUID=$(scutil --get ComputerName)
chmod +x msmp-agent && ./msmp-agent`;
}

export default function AgentTokens() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [installToken, setInstallToken] = useState('');
  const [installOS, setInstallOS] = useState('linux');
  const [installOpen, setInstallOpen] = useState(false);
  const [form] = Form.useForm();
  const serverUrl = window.location.origin;

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await client.get('/agent-tokens');
      setData(Array.isArray(resp) ? resp : resp.data || []);
    } catch (e) {
      setData([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      await client.post('/agent-tokens', values);
      message.success('Token 已创建');
      setCreateOpen(false);
      form.resetFields();
      load();
    } catch (e) {
      if (e.errorFields) return;
    } finally {
      setSubmitting(false);
    }
  };

  const handleRevoke = async (id) => {
    try {
      await client.delete(`/agent-tokens/${id}`);
      message.success('已吊销');
      load();
    } catch (e) {}
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
    {
      title: 'Token', dataIndex: 'token', key: 'token',
      render: (v) => <Typography.Text code copyable style={{ fontSize: 12 }}>{v}</Typography.Text>,
    },
    { title: '描述', dataIndex: 'description', key: 'description' },
    { title: '主机ID', dataIndex: 'host_id', key: 'host_id', render: (v) => v || '-' },
    {
      title: '过期', dataIndex: 'expires_at', key: 'expires_at',
      render: (t) => t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '永久',
    },
    {
      title: '状态', dataIndex: 'revoked', key: 'revoked',
      render: (v) => v ? <Tag color="red">已吊销</Tag> : <Tag color="green">有效</Tag>,
    },
    {
      title: '创建时间', dataIndex: 'created_at', key: 'created_at',
      render: (t) => dayjs(t).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作', key: 'action', width: 180,
      render: (_, r) => (
        <Space>
          <Button type="link" size="small" onClick={() => { setInstallToken(r.token); setInstallOpen(true); }}>安装命令</Button>
          {!r.revoked && (
            <Popconfirm title="确认吊销该 Token？" onConfirm={() => handleRevoke(r.id)}>
              <Button type="link" size="small" danger>吊销</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <h2>Agent 接入</h2>
      <Card style={{ marginBottom: 16 }}>
        <Typography.Paragraph>
          在目标主机上设置以下环境变量后运行 Agent：
        </Typography.Paragraph>
        <Typography.Paragraph code copyable style={{ background: '#f5f5f5', padding: 12 }}>
{`export MSMP_SERVER_URL=https://your-server
export AGENT_TOKEN=your-token
export AGENT_UUID=$(hostname)
./msmp-agent`}
        </Typography.Paragraph>
      </Card>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>新建 Token</Button>
      </Space>
      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} scroll={{ x: 'max-content' }} />
      <Modal
        title="新建 Agent Token"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={handleCreate}
        confirmLoading={submitting}
        okText="创建"
      >
        <Form form={form} layout="vertical">
          <Form.Item name="description" label="描述">
            <Input placeholder="例如：web-01 接入" />
          </Form.Item>
          <Form.Item name="host_id" label="绑定主机ID（可选）">
            <Input placeholder="留空则不绑定" />
          </Form.Item>
          <Form.Item name="expires_days" label="有效期（天）" extra="0 或留空表示永久">
            <InputNumber min={0} max={3650} style={{ width: 200 }} />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title="安装命令"
        open={installOpen}
        onCancel={() => setInstallOpen(false)}
        footer={null}
        width="min(680px, 92vw)"
      >
        <Space direction="vertical" style={{ width: '100%' }} size={12}>
          <Select
            value={installOS}
            onChange={setInstallOS}
            style={{ width: 200 }}
            options={[
              { value: 'linux', label: 'Linux' },
              { value: 'windows', label: 'Windows' },
              { value: 'macos', label: 'macOS' },
            ]}
          />
          <Typography.Text>
            下载 Agent 后在目标主机执行：
          </Typography.Text>
          <Typography.Paragraph
            code
            copyable
            style={{ background: '#f5f5f5', padding: 12, whiteSpace: 'pre-wrap' }}
          >
            {installOS === 'windows'
              ? windowsInstall(serverUrl, installToken)
              : installOS === 'macos'
                ? macosInstall(serverUrl, installToken)
                : linuxInstall(serverUrl, installToken)}
          </Typography.Paragraph>
        </Space>
      </Modal>
    </div>
  );
}
