import React, { useEffect, useState } from 'react';
import { Card, Form, Input, InputNumber, Button, Descriptions, message } from 'antd';
import { useAuthStore } from '../store/auth';
import client from '../api/client';

export default function Settings() {
  const { user, tenant } = useAuthStore();
  const [pwdForm] = Form.useForm();
  const [sysForm] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [savingSys, setSavingSys] = useState(false);

  useEffect(() => {
    client.get('/settings').then((data) => {
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
      await client.put(`/users/${user.id}`, { password: values.new_password });
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
      await client.put('/settings', {
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
      <h2>系统设置</h2>
      <Card title="账户信息" style={{ marginBottom: 16 }}>
        <Descriptions column={{ xs: 1, sm: 2 }}>
          <Descriptions.Item label="用户名">{user?.username || '-'}</Descriptions.Item>
          <Descriptions.Item label="角色">{user?.role || '-'}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{user?.email || '-'}</Descriptions.Item>
          <Descriptions.Item label="租户ID">{tenant?.id || user?.tenant_id || '-'}</Descriptions.Item>
        </Descriptions>
      </Card>
      {user?.role === 'admin' && (
        <Card title="通知与监控" style={{ marginBottom: 16 }}>
          <Form form={sysForm} layout="vertical" style={{ maxWidth: 480 }} onFinish={handleSaveSystem}>
            <Form.Item name="webhook_url" label="告警 Webhook 地址">
              <Input placeholder="https://example.com/webhook （留空不发送）" />
            </Form.Item>
            <Form.Item name="offline_after_sec" label="离线判定阈值（秒）">
              <InputNumber min={30} max={86400} style={{ width: 200 }} />
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" loading={savingSys}>保存</Button>
            </Form.Item>
          </Form>
        </Card>
      )}
      <Card title="修改密码">
        <Form form={pwdForm} layout="vertical" style={{ maxWidth: 480 }} onFinish={handleChangePassword}>
          <Form.Item name="new_password" label="新密码" rules={[{ required: true, message: '请输入新密码' }, { min: 6, message: '密码至少 6 位' }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item name="confirm_password" label="确认新密码" rules={[{ required: true, message: '请确认密码' }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={submitting}>保存</Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
