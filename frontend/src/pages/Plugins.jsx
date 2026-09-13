import React, { useEffect, useState } from 'react';
import { Table, Tag, Button, Space, Modal, Form, Input, Select, Switch, Popconfirm, message, Tabs, Typography, Empty } from 'antd';
import { PlusOutlined, ReloadOutlined, SendOutlined } from '@ant-design/icons';
import client from '../api/client';

const { Text } = Typography;

const typeLabel = { notifier: '通知', collector: '采集', probe: '探测' };
const typeColor = { notifier: 'blue', collector: 'green', probe: 'orange' };

export default function Plugins() {
  const [plugins, setPlugins] = useState([]);
  const [instances, setInstances] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState(null); // null=创建，否则为实例对象
  const [selMeta, setSelMeta] = useState(null); // 当前表单对应的插件元数据
  const [form] = Form.useForm();

  const notifiers = plugins.filter((p) => p.type === 'notifier');

  const load = async () => {
    setLoading(true);
    try {
      const [plugs, insts] = await Promise.all([
        client.get()('/plugins'),
        client.get()('/plugins/instances'),
      ]);
      setPlugins(Array.isArray(plugs) ? plugs : []);
      setInstances(Array.isArray(insts) ? insts : []);
    } catch (e) { message.error('加载失败'); }
    finally { setLoading(false); }
  };

  useEffect(() => { load(); }, []);

  const openCreate = () => {
    setEditing(null);
    setSelMeta(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (inst) => {
    setEditing(inst);
    const meta = plugins.find((p) => p.id === inst.plugin_id);
    setSelMeta(meta || null);
    form.setFieldsValue({ plugin_id: inst.plugin_id, name: inst.name, config: inst.config || {} });
    setModalOpen(true);
  };

  const onSelectPlugin = (id) => {
    const meta = plugins.find((p) => p.id === id);
    setSelMeta(meta || null);
    form.setFieldValue('config', {});
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      const payload = { plugin_id: values.plugin_id, name: values.name, config: values.config || {} };
      if (editing) await client.put()(`/plugins/instances/${editing.id}`, payload);
      else await client.post()('/plugins/instances', payload);
      message.success(editing ? '已更新' : '已创建');
      setModalOpen(false);
      load();
    } catch (e) { if (e.errorFields) return; message.error('保存失败'); }
  };

  const handleToggle = async (inst, enabled) => {
    try {
      await client.put()(`/plugins/instances/${inst.id}`, { enabled });
      load();
    } catch (e) { message.error('更新失败'); }
  };

  const handleTest = async (inst) => {
    try {
      const resp = await client.post()(`/plugins/instances/${inst.id}/test`);
      if (resp.ok) message.success('测试通知已发送');
      else message.error(resp.error || '测试失败');
    } catch (e) { message.error('测试请求失败'); }
  };

  const handleDelete = async (id) => {
    try {
      await client.delete()(`/plugins/instances/${id}`);
      message.success('已删除');
      load();
    } catch (e) { message.error('删除失败'); }
  };

  const instanceColumns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    {
      title: '插件', dataIndex: 'plugin_id', key: 'plugin_id',
      render: (id) => {
        const m = plugins.find((p) => p.id === id);
        return m ? m.name : id;
      },
    },
    {
      title: '启用', dataIndex: 'enabled', key: 'enabled', width: 90,
      render: (v, r) => <Switch size="small" checked={v} onChange={(c) => handleToggle(r, c)} />,
    },
    {
      title: '操作', key: 'action', width: 220,
      render: (_, r) => (
        <Space>
          <Button type="link" size="small" icon={<SendOutlined />} onClick={() => handleTest(r)}>测试</Button>
          <Button type="link" size="small" onClick={() => openEdit(r)}>编辑</Button>
          <Popconfirm title="确认删除该通知渠道？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const pluginColumns = [
    {
      title: '类型', dataIndex: 'type', key: 'type', width: 90,
      render: (v) => <Tag color={typeColor[v]}>{typeLabel[v] || v}</Tag>,
    },
    { title: '名称', dataIndex: 'name', key: 'name', width: 200 },
    { title: '标识', dataIndex: 'id', key: 'id', width: 180, render: (v) => <Text code>{v}</Text> },
    { title: '描述', dataIndex: 'description', key: 'description' },
  ];

  const renderConfigFields = () => {
    if (!selMeta) return <Empty description="请先选择插件" image={Empty.PRESENTED_IMAGE_SIMPLE} />;
    const fields = selMeta.config_fields || [];
    if (fields.length === 0) return <Empty description="该插件无需额外配置" image={Empty.PRESENTED_IMAGE_SIMPLE} />;
    return fields.map((f) => (
      <Form.Item
        key={f.key}
        name={['config', f.key]}
        label={f.label}
        rules={f.required ? [{ required: true, message: `${f.label} 必填` }] : []}
      >
        {f.secret ? <Input.Password placeholder={f.label} /> : <Input placeholder={f.label} />}
      </Form.Item>
    ));
  };

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 16 }}>
        <div className="page-title">插件管理</div>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>添加通知渠道</Button>
      </div>

      <Tabs
        items={[
          {
            key: 'instances',
            label: `通知渠道（${instances.length}）`,
            children: (
              <Table
                rowKey="id"
                loading={loading}
                dataSource={instances}
                columns={instanceColumns}
                pagination={false}
                locale={{ emptyText: '暂无通知渠道，点击右上角「添加通知渠道」' }}
              />
            ),
          },
          {
            key: 'plugins',
            label: `可用插件（${plugins.length}）`,
            children: (
              <Table
                rowKey="id"
                loading={loading}
                dataSource={plugins}
                columns={pluginColumns}
                pagination={false}
                size="small"
              />
            ),
          },
        ]}
      />

      <Modal
        title={editing ? '编辑通知渠道' : '添加通知渠道'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        okText="保存"
        cancelText="取消"
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="plugin_id" label="插件" rules={[{ required: true, message: '请选择插件' }]}>
            <Select
              placeholder="选择通知插件"
              disabled={!!editing}
              onChange={onSelectPlugin}
              options={notifiers.map((p) => ({ value: p.id, label: `${p.name} — ${p.description}` }))}
            />
          </Form.Item>
          <Form.Item name="name" label="渠道名称（可选）">
            <Input placeholder="不填则使用插件默认名" />
          </Form.Item>
          {renderConfigFields()}
        </Form>
      </Modal>
    </div>
  );
}