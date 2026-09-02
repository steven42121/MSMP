import React, { useState } from 'react';
import { Card, Form, Input, InputNumber, Button, Space, Typography, message, Alert } from 'antd';
import { CloudServerOutlined, KeyOutlined, BulbOutlined } from '@ant-design/icons';
import client from '../api/client';

const { Text } = Typography;

export default function LLMConfig() {
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);

  React.useEffect(() => {
    client.get()('/llm/settings').then((data) => {
      form.setFieldsValue({
        base_url: data['llm.base_url'] || '',
        api_key: data['llm.api_key'] || '',
        model: data['llm.model'] || 'gpt-4o-mini',
      });
    }).catch(() => {}).finally(() => setLoading(false));
  }, [form]);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);
      await client.put()('/llm/settings', {
        'llm.base_url': values.base_url || '',
        'llm.api_key': values.api_key || '',
        'llm.model': values.model || 'gpt-4o-mini',
      });
      message.success('LLM 配置已保存');
      if (values.api_key) {
        form.setFieldValue('api_key', '');
      }
    } catch (e) {
      if (e.errorFields) return;
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card
      style={{ borderRadius: 16, border: 'none', boxShadow: 'none' }}
      className="liquid-glass"
      loading={loading}
      title={
        <Space>
          <CloudServerOutlined style={{ color: '#667eea' }} />
          <span>LLM 智能配置</span>
        </Space>
      }
      extra={
        <Text type="secondary" style={{ fontSize: 12 }}>配置后可使用 AI 智能问答、告警根因分析、运维报告生成等功能</Text>
      }
    >
      <Alert
        style={{ borderRadius: 12, marginBottom: 20, background: 'rgba(102,126,234,0.1)', borderColor: 'rgba(102,126,234,0.3)' }}
        icon={<BulbOutlined />}
        message="支持 OpenAI 兼容 API（如 DeepSeek、通义千问、智谱等），填写 Base URL、API Key 和模型名称即可"
        type="info"
        showIcon
      />

      <Form form={form} layout="vertical" style={{ maxWidth: 560 }}>
        <Form.Item
          name="base_url"
          label="Base URL"
          extra="API 端点地址，例如：https://api.deepseek.com/v1 或 https://api.openai.com/v1"
          rules={[{ required: true, message: '请输入 Base URL' }]}
        >
          <Input prefix={<CloudServerOutlined style={{ color: '#667eea' }} />} placeholder="https://api.example.com/v1" />
        </Form.Item>

        <Form.Item
          name="api_key"
          label="API Key"
          extra="用于身份验证的密钥，保存后仅显示脱敏信息"
          rules={[{ required: true, message: '请输入 API Key' }]}
        >
          <Input.Password
            prefix={<KeyOutlined style={{ color: '#667eea' }} />}
            placeholder="sk-xxxxxxxx"
            style={{ background: 'rgba(255,255,255,0.06)', borderColor: 'rgba(255,255,255,0.12)' }}
          />
        </Form.Item>

        <Form.Item
          name="model"
          label="模型名称"
          extra="支持的模型，例如：gpt-4o-mini、deepseek-chat、glm-4 等"
          rules={[{ required: true, message: '请输入模型名称' }]}
        >
          <Input prefix={<span style={{ color: '#888' }}>model</span>} placeholder="gpt-4o-mini" />
        </Form.Item>

        <Form.Item style={{ marginBottom: 0 }}>
          <Button
            type="primary"
            htmlType="submit"
            onClick={handleSave}
            loading={saving}
            style={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none', minWidth: 120 }}
          >
            保存配置
          </Button>
        </Form.Item>
      </Form>
    </Card>
  );
}
