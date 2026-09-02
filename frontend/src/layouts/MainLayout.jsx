import React, { useEffect, useState } from 'react';
import { Layout, Menu, Button, Dropdown, Space, Grid } from 'antd';
import {
  DashboardOutlined,
  DesktopOutlined,
  LineChartOutlined,
  CodeOutlined,
  AlertOutlined,
  TeamOutlined,
  UserOutlined,
  SettingOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  LogoutOutlined,
  BulbOutlined,
  BulbFilled,
} from '@ant-design/icons';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../store/auth';
import { useThemeStore } from '../store/theme';

const { Header, Sider, Content } = Layout;

export default function MainLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { dark, toggle } = useThemeStore();
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.lg;

  useEffect(() => {
    setCollapsed(isMobile);
  }, [isMobile]);

  useEffect(() => {
    if (dark) {
      document.body.classList.add('msmp-dark');
    } else {
      document.body.classList.remove('msmp-dark');
    }
  }, [dark]);

  const menuItems = [
    { key: '/dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
    { key: '/hosts', icon: <DesktopOutlined />, label: '主机管理' },
    { key: '/monitor', icon: <LineChartOutlined />, label: '监控' },
    { key: '/tasks', icon: <CodeOutlined />, label: '任务' },
    { key: '/alerts', icon: <AlertOutlined />, label: '告警' },
    { key: '/alert-rules', icon: <AlertOutlined />, label: '告警规则' },
    { key: '/agent-tokens', icon: <DesktopOutlined />, label: 'Agent 接入' },
    { key: '/tenants', icon: <TeamOutlined />, label: '租户' },
    { key: '/users', icon: <UserOutlined />, label: '用户' },
    { key: '/audit-logs', icon: <SettingOutlined />, label: '审计日志' },
    { key: '/settings', icon: <SettingOutlined />, label: '设置' },
  ];

  const userMenu = {
    items: [
      { key: 'profile', label: user?.username || '用户' },
      { type: 'divider' },
      { key: 'logout', label: '退出登录', icon: <LogoutOutlined />, danger: true },
    ],
    onClick: ({ key }) => {
      if (key === 'logout') {
        logout();
        navigate('/login', { replace: true });
      }
    },
  };

  return (
    <Layout style={{ minHeight: '100vh', position: 'relative' }}>
      {/* 背景光晕 */}
      <div className="glow-orb primary" />
      <div className="glow-orb secondary" />
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        breakpoint="lg"
        collapsedWidth={isMobile ? 0 : 80}
        width={220}
        style={{
          background: dark ? '#0d1117' : 'linear-gradient(180deg, #1a1a2e 0%, #16213e 100%)',
          transition: 'background 0.3s ease',
          boxShadow: '2px 0 12px rgba(0,0,0,0.15)',
        }}
      >
        {/* Logo area */}
        <div style={{
          height: 64, display: 'flex', alignItems: 'center', justifyContent: 'center',
          borderBottom: '1px solid rgba(255,255,255,0.08)',
        }}>
          <div style={{
            display: 'flex', alignItems: 'center', gap: 10,
          }}>
            <div style={{
              width: 36, height: 36, borderRadius: 10,
              background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              boxShadow: '0 4px 12px rgba(102,126,234,0.4)',
            }}>
              <DesktopOutlined style={{ color: '#fff', fontSize: 18 }} />
            </div>
            {!collapsed && (
              <span style={{
                color: '#fff', fontSize: 18, fontWeight: 700, letterSpacing: 1.5,
                textShadow: '0 2px 8px rgba(102,126,234,0.5)',
              }}>
                MSMP
              </span>
            )}
          </div>
        </div>

        {/* Navigation */}
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{
            background: 'transparent',
            border: 'none',
            paddingTop: 8,
          }}
        />
      </Sider>

      <Layout>
        <Header style={{
          background: dark ? '#141414' : '#fff',
          padding: isMobile ? '0 12px' : '0 24px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          borderBottom: dark ? '1px solid rgba(255,255,255,0.06)' : '1px solid #f0f0f0',
          boxShadow: dark ? '0 2px 8px rgba(0,0,0,0.3)' : '0 2px 8px rgba(0,0,0,0.04)',
          transition: 'all 0.3s ease',
        }}>
          <Space>
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed(!collapsed)}
              style={{ color: dark ? 'rgba(255,255,255,0.65)' : 'rgba(0,0,0,0.65)' }}
            />
            <span style={{
              fontSize: 14, fontWeight: 500,
              color: dark ? 'rgba(255,255,255,0.65)' : 'rgba(0,0,0,0.45)',
            }}>
              {menuItems.find(m => m.key === location.pathname)?.label || 'MSMP'}
            </span>
          </Space>
          <Space size={8}>
            <Button
              type="text"
              icon={dark ? <BulbFilled /> : <BulbOutlined />}
              onClick={toggle}
              style={{ color: dark ? 'rgba(255,255,255,0.65)' : 'rgba(0,0,0,0.65)' }}
            />
            <Dropdown menu={userMenu} placement="bottomRight">
              <Space
                style={{
                  cursor: 'pointer', padding: '4px 10px',
                  borderRadius: 8, border: dark ? '1px solid rgba(255,255,255,0.12)' : '1px solid #e8e8e8',
                  transition: 'all 0.2s ease',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = dark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.04)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'transparent';
                }}
              >
                <div style={{
                  width: 28, height: 28, borderRadius: 8,
                  background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  color: '#fff', fontSize: 13, fontWeight: 600,
                }}>
                  {(user?.username || 'U').charAt(0).toUpperCase()}
                </div>
                <span style={{
                  color: dark ? 'rgba(255,255,255,0.85)' : 'rgba(0,0,0,0.85)',
                  fontSize: 13,
                }}>
                  {user?.username}
                </span>
              </Space>
            </Dropdown>
          </Space>
        </Header>

        <Content style={{
          margin: isMobile ? 8 : 24,
          padding: isMobile ? 12 : 24,
          background: dark ? '#1a1d23' : '#fff',
          borderRadius: 12,
          minHeight: 280,
          boxShadow: dark ? '0 2px 12px rgba(0,0,0,0.2)' : '0 2px 8px rgba(0,0,0,0.04)',
          transition: 'all 0.3s ease',
        }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
