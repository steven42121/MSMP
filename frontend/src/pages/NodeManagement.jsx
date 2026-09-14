import React, { useEffect, useState } from 'react';
import { Table, Tag, Button, Space, Modal, Form, Input, Switch, Popconfirm, message, Descriptions, Card } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import client from '../api/client';

export default function NodeManagement() {
  const [nodes, setNodes] = useState([]);
  const [info, setInfo] = useState(null);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [nodeResp, infoResp] = await Promise.all([
        client.get()('/cluster/nodes'),
        client.get()('/cluster/info'),
      ]);
      setNodes(Array.isArray(nodeResp) ? nodeResp : []);
      setInfo(typeof infoResp === 'object' ? infoResp : null);
    } catch (e) { message.error('加载失败'); }
    finally { setLoading(false); }
  };

  useEffect(() => { load(); }, []);

  const openCreate = () => { setEditing(null); form.resetFields(); setModalOpen(true); };
  const openEdit = (n) => { setEditing(n); form.setFieldsValue({ name: n.name, address: n.address }); setModalOpen(true); };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editing) await client.put()(`/cluster/nodes/${editing.id}`, values);
      else await client.post()('/cluster/nodes', values);
      message.success(editing ? '已更新' : '已添加');
      setModalOpen(false);
      load();
    } catch (e) { if (e.errorFields) return; message.error('保存失败'); }
  };

  const handleToggle = async (n, enabled) => {
    try { await client.put()(`/cluster/nodes/${n.id}`, { enabled }); load(); }
    catch (e) { message.error('更新失败'); }
  };

  const handleDelete = async (id) => {
    try { await client.delete()(`/cluster/nodes/${id}`); message.success('已删除'); load(); }
    catch (e) { message.error('删除失败'); }
  };

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '地址', dataIndex: 'address', key: 'address', render: (v) => <span style={{ fontFamily: 'monospace' }}>{v}</span> },
    {
      title: '在线', dataIndex: 'alive', key: 'alive', width: 80,
      render: (v) => <Tag color={v ? 'green' : 'default'}>{v ? '在线' : '离线'}</Tag>,
    },
    {
      title: '启用', dataIndex: 'enabled', key: 'enabled', width: 90,
      render: (v, r) => <Switch size="small" checked={v} onChange={(c) => handleToggle(r, c)} />,
    },
    {
      title: '操作', key: 'action', width: 160,
      render: (_, r) => (
        <Space>
          <Button type="link" size="small" onClick={() => openEdit(r)}>编辑</Button>
          <Popconfirm title="确认删除该节点？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 16 }}>
        <div className="page-title">集群节点</div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>添加节点</Button>
        </Space>
      </div>

      {info && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Descriptions column={3} size="small">
            <Descriptions.Item label="模式">
              <Tag color={info.mode === 'cluster' ? 'blue' : 'default'}>{info.mode === 'cluster' ? '集群' : '单机'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="当前节点">{info.address || '-'}</Descriptions.Item>
            <Descriptions.Item label="Leader">{info.leader || '-'}</Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      <Table
        rowKey="id"
        loading={loading}
        dataSource={nodes}
        columns={columns}
        pagination={false}
        locale={{ emptyText: '暂无节点，点击右上角「添加节点」接入集群' }}
      />

      <Modal
        title={editing ? '编辑节点' : '添加节点'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        okText="保存"
        cancelText="取消"
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="节点名称" rules={[{ required: true, message: '名称必填' }]}>
            <Input placeholder="如 node-1" />
          </Form.Item>
          <Form.Item name="address" label="节点地址" rules={[{ required: true, message: '地址必填' }]}>
            <Input placeholder="如 http://10.0.0.2:8080" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}