import React, { useEffect, useState } from 'react';
import { Card, Form, Input, InputNumber, Button, Descriptions, Typography, Space, Tag, message } from 'antd';
import { UserOutlined, SettingOutlined, LockOutlined, BellOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/auth';
import client from '../api/client';

const { Text } = Typography;

export default function Settings() {
  const { user, tenant } = useAuthStore();
  const [pwdForm] = Form.useForm();
  const [sysForm] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [savingSys, setSavingSys] = useState(false);

  useEffect(() => {
    client.get()('/settings').then((data) => {
      sysForm.setFieldsValue({
        webhook_url: data['notification.webhookurl'] || '',
        offline_after_sec: parseInt(data['agent.offlineaftersec'] || '120', 10),
      });
    }).catch(() => {});
  }, [sysForm]);

  const handleChangePassword = async () => {
    try {
      const values = await pwdForm.validateFields();
      if (values.new_password !== values.confirm_password) {
        message.error('两次输入的密码不一致');
        return;
      }
      setSubmitting(true);
      await client.put()(`/users/${user.id}`, { password: values.new_password });
      message.success('密码已修改，请重新登录');
      pwdForm.resetFields();
    } catch (e) {
      if (e.errorFields) return;
    } finally {
      setSubmitting(false);
    }
  };

  const handleSaveSystem = async () => {
    try {
      const values = await sysForm.validateFields();
      setSavingSys(true);
      await client.put()('/settings', {
        'notification.webhookurl': values.webhook_url || '',
        'agent.offlineaftersec': String(values.offline_after_sec || 120),
      });
      message.success('设置已保存');
    } catch (e) {
      if (e.errorFields) return;
    } finally {
      setSavingSys(false);
    }
  };

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 20 }}>
        <div>
          <div className="page-title">系统设置</div>
          <Text type="secondary" style={{ fontSize: 13 }}>账户信息与平台参数配置</Text>
        </div>
      </div>

      <Card
          style={{ borderRadius: 16, border: 'none', boxShadow: 'none', marginBottom: 16 }}
          className="liquid-glass"
        title={<Space><UserOutlined style={{ color: '#667eea' }} /><span>账户信息</span></Space>}
      >
        <Descriptions column={{ xs: 1, sm: 2 }} bordered size="middle">
          <Descriptions.Item label="用户名">{user?.username || '-'}</Descriptions.Item>
          <Descriptions.Item label="角色">
            <Tag color={user?.role === 'admin' ? 'gold' : 'blue'} style={{ borderRadius: 4 }}>{user?.role || '-'}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="邮箱">{user?.email || '-'}</Descriptions.Item>
          <Descriptions.Item label="租户ID"><Text code>{tenant?.id || user?.tenant_id || '-'}</Text></Descriptions.Item>
        </Descriptions>
      </Card>

      {user?.role === 'admin' && (
        <Card
        style={{ borderRadius: 16, border: 'none', boxShadow: 'none', marginBottom: 16 }}
        className="liquid-glass"
          title={<Space><BellOutlined style={{ color: '#faad14' }} /><span>通知与监控</span></Space>}
        >
          <Form form={sysForm} layout="vertical" style={{ maxWidth: 480 }} onFinish={handleSaveSystem}>
            <Form.Item name="webhook_url" label="告警 Webhook 地址" extra="接收告警通知的 Webhook 端点，留空则不发送">
              <Input placeholder="https://example.com/webhook" />
            </Form.Item>
            <Form.Item name="offline_after_sec" label="离线判定阈值（秒）" extra="心跳超过此时间视为离线，范围 30-86400">
              <InputNumber min={30} max={86400} style={{ width: 200 }} />
            </Form.Item>
            <Form.Item style={{ marginBottom: 0 }}>
              <Button type="primary" htmlType="submit" loading={savingSys} style={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none' }}>保存设置</Button>
            </Form.Item>
          </Form>
        </Card>
      )}

      <Card
        style={{ borderRadius: 16, border: 'none', boxShadow: 'none' }}
        className="liquid-glass"
        title={<Space><LockOutlined style={{ color: '#52c41a' }} /><span>修改密码</span></Space>}
      >
        <Form form={pwdForm} layout="vertical" style={{ maxWidth: 480 }} onFinish={handleChangePassword}>
          <Form.Item name="new_password" label="新密码" rules={[{ required: true, message: '请输入新密码' }, { min: 6, message: '密码至少 6 位' }]}>
            <Input.Password placeholder="至少 6 位" />
          </Form.Item>
          <Form.Item name="confirm_password" label="确认新密码" rules={[{ required: true, message: '请确认密码' }]}>
            <Input.Password placeholder="再次输入新密码" />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" loading={submitting} style={{ background: 'linear-gradient(135deg, #52c41a 0%, #13c2c2 100%)', border: 'none' }}>修改密码</Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}

