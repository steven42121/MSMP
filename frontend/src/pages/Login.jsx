import React, { useState, useEffect } from 'react';
import { Form, Input, Button, Card, Typography } from 'antd';
import { UserOutlined, LockOutlined, DesktopOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import client from '../api/client';
import { useAuthStore } from '../store/auth';

const { Title, Text } = Typography;

// 背景装饰球
function BGOrbs() {
  return (
    <div style={{ position: 'fixed', inset: 0, overflow: 'hidden', pointerEvents: 'none', zIndex: 0 }}>
      <div style={{
        position: 'absolute', width: 420, height: 420, borderRadius: '50%',
        background: 'radial-gradient(circle, rgba(102,126,234,0.45) 0%, transparent 70%)',
        top: '-8%', left: '-6%', animation: 'float 8s ease-in-out infinite',
      }} />
      <div style={{
        position: 'absolute', width: 360, height: 360, borderRadius: '50%',
        background: 'radial-gradient(circle, rgba(118,75,162,0.4) 0%, transparent 70%)',
        bottom: '-10%', right: '-4%', animation: 'float 10s ease-in-out infinite reverse',
      }} />
      <div style={{
        position: 'absolute', width: 200, height: 200, borderRadius: '50%',
        background: 'radial-gradient(circle, rgba(72,219,251,0.3) 0%, transparent 70%)',
        top: '40%', left: '60%', animation: 'float 6s ease-in-out infinite 2s',
      }} />
    </div>
  );
}

export default function Login() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const setAuth = useAuthStore((s) => s.setAuth);

  useEffect(() => {
    document.title = 'MSMP - 登录';
    return () => { document.title = 'MSMP - 运维管理平台'; };
  }, []);

  const onFinish = async (values) => {
    setLoading(true);
    try {
      const resp = await client.post()('/auth/login', {
        username: values.username,
        password: values.password,
      });
      setAuth({
        token: resp.token,
        user: resp.user,
        tenant: { id: resp.user.tenant_id },
      });
      navigate('/dashboard', { replace: true });
    } catch (err) {
      // handled by interceptor
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      minHeight: '100vh',
      background: 'linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%)',
      position: 'relative',
      overflow: 'hidden',
    }}>
      <BGOrbs />
      <div style={{
        width: '100%', maxWidth: 420, padding: '0 16px', position: 'relative', zIndex: 1,
        animation: 'fadeInUp 0.6s ease forwards',
      }}>
        {/* Logo */}
        <div style={{ textAlign: 'center', marginBottom: 28 }}>
          <div style={{
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            width: 64, height: 64, borderRadius: 18,
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            boxShadow: '0 8px 24px rgba(102,126,234,0.4)',
            marginBottom: 14,
          }}>
            <DesktopOutlined style={{ color: '#fff', fontSize: 30 }} />
          </div>
          <Title level={3} style={{ color: '#fff', margin: 0, fontWeight: 700, letterSpacing: 1 }}>
            MSMP 运维管理平台
          </Title>
          <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 13 }}>
            Multi-Server Monitoring Platform
          </Text>
        </div>

        {/* 登录卡片 */}
        <Card className="liquid-glass login-glass" style={{
          borderRadius: 20,
        }}>
          <Form name="login" onFinish={onFinish} size="large" layout="vertical">
            <Form.Item
              name="username"
              rules={[{ required: true, message: '请输入用户名' }]}
              style={{ marginBottom: 18 }}
            >
              <Input
                prefix={<UserOutlined style={{ color: 'rgba(255,255,255,0.4)' }} />}
                placeholder="用户名"
                style={{
                  background: 'rgba(255,255,255,0.08)',
                  border: '1px solid rgba(255,255,255,0.15)',
                  color: '#fff',
                  borderRadius: 10,
                }}
                classNames={{ input: 'login-input' }}
              />
            </Form.Item>
            <Form.Item
              name="password"
              rules={[{ required: true, message: '请输入密码' }]}
              style={{ marginBottom: 24 }}
            >
              <Input.Password
                prefix={<LockOutlined style={{ color: 'rgba(255,255,255,0.4)' }} />}
                placeholder="密码"
                style={{
                  background: 'rgba(255,255,255,0.08)',
                  border: '1px solid rgba(255,255,255,0.15)',
                  color: '#fff',
                  borderRadius: 10,
                }}
              />
            </Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
              style={{
                height: 48,
                borderRadius: 10,
                background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                border: 'none',
                fontSize: 16,
                fontWeight: 600,
                letterSpacing: 2,
                boxShadow: '0 4px 16px rgba(102,126,234,0.4)',
                transition: 'all 0.3s ease',
              }}
            >
              {loading ? '登录中...' : '登 录'}
            </Button>
          </Form>
        </Card>

        {/* Footer text */}
        <div style={{ textAlign: 'center', marginTop: 20 }}>
          <Text style={{ color: 'rgba(255,255,255,0.3)', fontSize: 12 }}>
            MSMP v1.0 &mdash; Secure Infrastructure Monitoring
          </Text>
        </div>
      </div>

      <style>{`
        @keyframes float {
          0%, 100% { transform: translateY(0px) scale(1); }
          50% { transform: translateY(-24px) scale(1.04); }
        }
        @keyframes fadeInUp {
          from { opacity: 0; transform: translateY(28px); }
          to { opacity: 1; transform: translateY(0); }
        }
        .login-input input::placeholder { color: rgba(255,255,255,0.35) !important; }
      `}</style>
    </div>
  );
}
