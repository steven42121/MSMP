import React, { useEffect, useState } from 'react';
import { Layout, Menu, Button, Dropdown, Space, Grid, Drawer } from 'antd';
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
  MenuOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../store/auth';
import { useThemeStore } from '../store/theme';

const { Header, Sider, Content } = Layout;

export default function MainLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
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
    { key: '/ai-chat', icon: <BulbOutlined />, label: 'AI 助手' },
    { key: '/probes', icon: <ThunderboltOutlined />, label: '可用性探测' },
    { key: '/cron-jobs', icon: <ClockCircleOutlined />, label: '定时任务' },
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

  const currentLabel = menuItems.find(m => m.key === location.pathname)?.label || 'MSMP';

  const navContent = (
    <>
      <div style={{
        height: 64, display: 'flex', alignItems: 'center', justifyContent: 'center',
        borderBottom: dark ? '0.5px solid rgba(255,255,255,0.06)' : '0.5px solid rgba(0,0,0,0.05)',
        padding: '0 20px',
      }}>
        <div style={{
          display: 'flex', alignItems: 'center', gap: 10, width: '100%',
        }}>
          <div style={{
            width: 34, height: 34, borderRadius: 9, flexShrink: 0,
            background: dark
              ? 'linear-gradient(135deg, rgba(100,140,255,0.9) 0%, rgba(160,80,255,0.9) 100%)'
              : 'linear-gradient(135deg, #5e7cff 0%, #8b5cf6 100%)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: dark
              ? '0 2px 12px rgba(100,140,255,0.3), inset 0 0.5px 0 rgba(255,255,255,0.2)'
              : '0 2px 12px rgba(94,124,255,0.35), inset 0 0.5px 0 rgba(255,255,255,0.5)',
          }}>
            <DesktopOutlined style={{ color: '#fff', fontSize: 16 }} />
          </div>
          <span style={{
            color: dark ? 'rgba(255,255,255,0.92)' : 'rgba(0,0,0,0.88)',
            fontSize: 17, fontWeight: 700, letterSpacing: 1.2,
          }}>
            MSMP
          </span>
        </div>
      </div>
      <Menu
        theme={dark ? 'dark' : 'light'}
        mode="inline"
        selectedKeys={[location.pathname]}
        items={menuItems}
        onClick={({ key }) => { navigate(key); setMobileOpen(false); }}
        style={{
          background: 'transparent',
          border: 'none',
          paddingTop: 4,
          flex: 1,
        }}
      />
    </>
  );

  return (
    <Layout style={{ minHeight: '100vh', position: 'relative', background: dark ? '#000' : '#f2f2f7' }}>
      {/* iOS 环境光晕 */}
      <div className="ambient-orb orb-a" />
      <div className="ambient-orb orb-b" />
      <div className="ambient-orb orb-c" />

      {/* 全局统一背景光斑 — 跟随鼠标/重力流动 */}
      <div className="global-glow" />

      {/* Desktop Sider */}
      {!isMobile && (
        <Sider
          trigger={null}
          collapsible
          collapsed={collapsed}
          breakpoint="lg"
          collapsedWidth={80}
          width={220}
          style={{
            background: dark
              ? 'linear-gradient(180deg, rgba(28,28,30,0.97) 0%, rgba(18,18,20,0.97) 100%)'
              : 'linear-gradient(180deg, rgba(245,245,247,0.97) 0%, rgba(238,238,242,0.97) 100%)',
            backdropFilter: 'saturate(180%) blur(20px)',
            WebkitBackdropFilter: 'saturate(180%) blur(20px)',
            transition: 'all 0.3s cubic-bezier(0.4,0,0.2,1)',
            borderRight: dark ? '0.5px solid rgba(255,255,255,0.08)' : '0.5px solid rgba(0,0,0,0.06)',
            position: 'sticky',
            top: 0,
            height: '100vh',
            zIndex: 100,
          }}
        >
          {navContent}
        </Sider>
      )}

      <Layout>
        <Header style={{
          background: dark
            ? 'linear-gradient(180deg, rgba(20,20,22,0.95) 0%, rgba(16,16,18,0.95) 100%)'
            : 'linear-gradient(180deg, rgba(255,255,255,0.85) 0%, rgba(248,248,250,0.85) 100%)',
          backdropFilter: 'saturate(180%) blur(20px)',
          WebkitBackdropFilter: 'saturate(180%) blur(20px)',
          padding: isMobile ? '0 12px' : '0 24px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          borderBottom: dark ? '0.5px solid rgba(255,255,255,0.07)' : '0.5px solid rgba(0,0,0,0.06)',
          transition: 'all 0.3s cubic-bezier(0.4,0,0.2,1)',
          position: 'sticky',
          top: 0,
          zIndex: 99,
          height: 64,
          lineHeight: '64px',
        }}>
          <Space>
            {isMobile && (
              <Button
                type="text"
                icon={<MenuOutlined style={{ fontSize: 18 }} />}
                onClick={() => setMobileOpen(true)}
                style={{
                  color: dark ? 'rgba(255,255,255,0.7)' : 'rgba(0,0,0,0.6)',
                  borderRadius: 8,
                  minWidth: 32,
                  height: 32,
                }}
              />
            )}
            {!isMobile && (
              <Button
                type="text"
                icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                onClick={() => setCollapsed(!collapsed)}
                style={{
                  color: dark ? 'rgba(255,255,255,0.7)' : 'rgba(0,0,0,0.6)',
                  borderRadius: 8,
                  minWidth: 32,
                  height: 32,
                }}
              />
            )}
            <span style={{
              fontSize: isMobile ? 14 : 13,
              fontWeight: 500,
              color: dark ? 'rgba(255,255,255,0.55)' : 'rgba(0,0,0,0.4)',
              letterSpacing: 0.2,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              maxWidth: isMobile ? 140 : 200,
              display: 'block',
            }}>
              {currentLabel}
            </span>
          </Space>
          <Space size={6}>
            <Button
              type="text"
              icon={dark ? <BulbFilled /> : <BulbOutlined />}
              onClick={toggle}
              style={{
                color: dark ? 'rgba(255,255,255,0.65)' : 'rgba(0,0,0,0.55)',
                borderRadius: 8,
                minWidth: 32,
                height: 32,
              }}
            />
            <Dropdown menu={userMenu} placement="bottomRight">
              <Space
                style={{
                  cursor: 'pointer', padding: isMobile ? '2px 6px 2px 4px' : '3px 8px 3px 4px',
                  borderRadius: 10,
                  border: dark ? '0.5px solid rgba(255,255,255,0.1)' : '0.5px solid rgba(0,0,0,0.08)',
                  background: dark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.03)',
                  transition: 'all 0.2s cubic-bezier(0.4,0,0.2,1)',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = dark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.06)';
                  e.currentTarget.style.borderColor = dark ? 'rgba(255,255,255,0.18)' : 'rgba(0,0,0,0.14)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = dark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.03)';
                  e.currentTarget.style.borderColor = dark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.08)';
                }}
              >
                <div style={{
                  width: 26, height: 26, borderRadius: 7, flexShrink: 0,
                  background: dark
                    ? 'linear-gradient(135deg, #648cff 0%, #a050ff 100%)'
                    : 'linear-gradient(135deg, #5e7cff 0%, #8b5cf6 100%)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  color: '#fff', fontSize: 12, fontWeight: 700,
                  boxShadow: '0 1px 4px rgba(0,0,0,0.15)',
                }}>
                  {(user?.username || 'U').charAt(0).toUpperCase()}
                </div>
                {!isMobile && (
                  <span style={{
                    color: dark ? 'rgba(255,255,255,0.85)' : 'rgba(0,0,0,0.8)',
                    fontSize: 13, fontWeight: 500,
                    maxWidth: 80,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    display: 'block',
                  }}>
                    {user?.username}
                  </span>
                )}
              </Space>
            </Dropdown>
          </Space>
        </Header>

        <Content style={{
          margin: isMobile ? 8 : 20,
          padding: isMobile ? 12 : 24,
          background: dark
            ? 'linear-gradient(160deg, rgba(255,255,255,0.05) 0%, rgba(255,255,255,0.02) 100%)'
            : 'linear-gradient(160deg, rgba(255,255,255,0.7) 0%, rgba(255,255,255,0.4) 100%)',
          backdropFilter: 'saturate(180%) blur(20px)',
          WebkitBackdropFilter: 'saturate(180%) blur(20px)',
          borderRadius: isMobile ? 12 : 20,
          minHeight: 280,
          border: dark
            ? '0.5px solid rgba(255,255,255,0.08)'
            : '0.5px solid rgba(255,255,255,0.7)',
          borderTop: dark ? '0.5px solid rgba(255,255,255,0.14)' : '0.5px solid rgba(255,255,255,0.8)',
          boxShadow: dark
            ? '0 2px 16px rgba(0,0,0,0.4), 0 8px 40px rgba(0,0,0,0.3), inset 0 0.5px 0 rgba(255,255,255,0.06)'
            : '0 2px 16px rgba(0,0,0,0.06), 0 8px 40px rgba(0,0,0,0.06), inset 0 0.5px 0 rgba(255,255,255,0.8)',
          transition: 'all 0.3s cubic-bezier(0.4,0,0.2,1)',
        }}>
          <Outlet />
        </Content>
      </Layout>

      {/* Mobile Drawer */}
      <Drawer
        open={mobileOpen}
        onClose={() => setMobileOpen(false)}
        placement="left"
        width={260}
        styles={{
          body: { padding: 0 },
          mask: { backgroundColor: 'rgba(0,0,0,0.4)' },
        }}
        style={{
          background: dark
            ? 'linear-gradient(180deg, rgba(28,28,30,0.98) 0%, rgba(18,18,20,0.98) 100%)'
            : 'linear-gradient(180deg, rgba(245,245,247,0.98) 0%, rgba(238,238,242,0.98) 100%)',
          backdropFilter: 'saturate(180%) blur(20px)',
          WebkitBackdropFilter: 'saturate(180%) blur(20px)',
        }}
        closeIcon={null}
      >
        {navContent}
      </Drawer>
    </Layout>
  );
}
