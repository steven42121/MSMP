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
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        breakpoint="lg"
        collapsedWidth={isMobile ? 0 : 80}
        width={220}
      >
        <div style={{ height: 64, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <h1 style={{ color: '#fff', margin: 0, fontSize: collapsed ? 16 : 20 }}>
            {collapsed ? 'MS' : 'MSMP'}
          </h1>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ background: dark ? '#141414' : '#fff', padding: isMobile ? '0 12px' : '0 24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed(!collapsed)}
            />
            <Button
              type="text"
              icon={dark ? <BulbFilled /> : <BulbOutlined />}
              onClick={toggle}
            />
          </Space>
          <Dropdown menu={userMenu}>
            <Button type="text" icon={<UserOutlined />}>{user?.username}</Button>
          </Dropdown>
        </Header>
        <Content style={{ margin: isMobile ? 8 : 24, padding: isMobile ? 12 : 24, background: dark ? '#141414' : '#fff', borderRadius: 8, minHeight: 280 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}