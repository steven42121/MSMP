import React, { useEffect, useState, useCallback } from 'react';
import { Table, Button, Modal, Form, Input, Select, Tag, Space, Popconfirm, message, Card, Typography } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import client from '../api/client';

const { Text } = Typography;
const roleColor = { admin: 'red', member: 'blue' };

export default function Users() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [createOpen, setCreateOpen] = useState(false);
  const [editUser, setEditUser] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await client.get()('/users', { params: { keyword } });
      setData(Array.isArray(resp) ? resp : resp.data || []);
    } catch (e) {
      setData([]);
    } finally {
      setLoading(false);
    }
  }, [keyword]);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      setSubmitting(true);
      await client.post()('/users', values);
      message.success('用户已创建');
      setCreateOpen(false);
      createForm.resetFields();
      load();
    } catch (e) {
      if (e.errorFields) return;
    } finally {
      setSubmitting(false);
    }
  };

  const openEdit = (user) => {
    setEditUser(user);
    editForm.setFieldsValue({ email: user.email, role: user.role });
    setEditOpen(true);
  };

  const [editOpen, setEditOpen] = useState(false);

  const handleEdit = async () => {
    try {
      const values = await editForm.validateFields();
      setSubmitting(true);
      await client.put()(`/users/${editUser.id}`, values);
      message.success('已更新');
      setEditOpen(false);
      setEditUser(null);
      load();
    } catch (e) {
      if (e.errorFields) return;
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id) => {
    try {
      await client.delete()(`/users/${id}`);
      message.success('已删除');
      load();
    } catch (e) {}
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
    { title: '用户名', dataIndex: 'username', key: 'username' },
    { title: '邮箱', dataIndex: 'email', key: 'email' },
    {
      title: '角色', dataIndex: 'role', key: 'role',
      render: (r) => <Tag color={roleColor[r] || 'default'}>{r}</Tag>,
    },
    {
      title: '创建时间', dataIndex: 'created_at', key: 'created_at',
      render: (t) => dayjs(t).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作', key: 'action',
      render: (_, r) => (
        <Space>
          <Button type="link" onClick={() => openEdit(r)}>编辑</Button>
          <Popconfirm title="确认删除该用户？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 20 }}>
        <div>
          <div className="page-title">用户管理</div>
          <Text type="secondary" style={{ fontSize: 13 }}>管理平台用户账号与角色权限</Text>
        </div>
      </div>
      <Card className="liquid-glass" style={{ marginBottom: 16, borderRadius: 16, border: 'none' }}>
        <Space wrap className="filter-bar">
          <Input.Search
            placeholder="用户名"
            allowClear
            style={{ width: 240 }}
            onSearch={(v) => setKeyword(v)}
          />
          <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)} style={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none' }}>新建用户</Button>
        </Space>
      </Card>
      <Card className="liquid-glass" style={{ borderRadius: 16, border: 'none' }}>
        <Table columns={columns} dataSource={data} rowKey="id" loading={loading} scroll={{ x: 'max-content' }} />
      </Card>
      <Modal
        title="新建用户"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={handleCreate}
        confirmLoading={submitting}
        okText="创建"
      >
        <Form form={createForm} layout="vertical">
          <Form.Item name="username" label="用户名" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input />
          </Form.Item>
          <Form.Item name="role" label="角色" initialValue="member">
            <Select options={[
              { value: 'admin', label: '管理员' },
              { value: 'member', label: '普通成员' },
            ]} />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title="编辑用户"
        open={editOpen}
        onCancel={() => { setEditOpen(false); setEditUser(null); }}
        onOk={handleEdit}
        confirmLoading={submitting}
        okText="保存"
      >
        <Form form={editForm} layout="vertical">
          <Form.Item name="email" label="邮箱">
            <Input />
          </Form.Item>
          <Form.Item name="role" label="角色">
            <Select options={[
              { value: 'admin', label: '管理员' },
              { value: 'member', label: '普通成员' },
            ]} />
          </Form.Item>
          <Form.Item name="password" label="新密码" extra="留空则不修改密码">
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
