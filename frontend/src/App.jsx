import React, { Suspense, lazy } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import MainLayout from './layouts/MainLayout';
import Login from './pages/Login';
import { useAuthStore } from './store/auth';
import { useGlobalMouseTracker } from './hooks/useGlobalMouseTracker';

// 路由级代码分割：登录页首屏直载，其余按需加载（配合 vite manualChunks vendor 分包）
const Dashboard = lazy(() => import('./pages/Dashboard'));
const HostList = lazy(() => import('./pages/HostList'));
const HostDetail = lazy(() => import('./pages/HostDetail'));
const Monitor = lazy(() => import('./pages/Monitor'));
const Tasks = lazy(() => import('./pages/Tasks'));
const TaskDetail = lazy(() => import('./pages/TaskDetail'));
const Tenants = lazy(() => import('./pages/Tenants'));
const Users = lazy(() => import('./pages/Users'));
const Alerts = lazy(() => import('./pages/Alerts'));
const AlertRules = lazy(() => import('./pages/AlertRules'));
const AgentTokens = lazy(() => import('./pages/AgentTokens'));
const AuditLogs = lazy(() => import('./pages/AuditLogs'));
const Settings = lazy(() => import('./pages/Settings'));
const NotFound = lazy(() => import('./pages/NotFound'));
const AIChat = lazy(() => import('./pages/AIChat'));
const LLMConfig = lazy(() => import('./pages/LLMConfig'));
const Probes = lazy(() => import('./pages/Probes'));
const CronJobs = lazy(() => import('./pages/CronJobs'));
const AssetInventory = lazy(() => import('./pages/AssetInventory'));
const Plugins = lazy(() => import('./pages/Plugins'));
const NodeManagement = lazy(() => import('./pages/NodeManagement'));
const Security = lazy(() => import('./pages/Security'));

function RouteFallback() {
  return (
    <div style={{ padding: '40px 8px' }}>
      <div className="route-skeleton route-skeleton-title" />
      <div className="route-skeleton" style={{ width: '62%' }} />
      <div className="route-skeleton" style={{ width: '88%' }} />
      <div className="route-skeleton" style={{ width: '74%' }} />
    </div>
  );
}

function RequireAuth({ children }) {
  const token = useAuthStore((s) => s.token);
  if (!token) return <Navigate to="/login" replace />;
  return children;
}

export default function App() {
  useGlobalMouseTracker();

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route
          path="/"
          element={
            <RequireAuth>
              <MainLayout />
            </RequireAuth>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Suspense fallback={<RouteFallback />}>
            <Route path="dashboard" element={<Dashboard />} />
            <Route path="hosts" element={<HostList />} />
            <Route path="hosts/:uuid" element={<HostDetail />} />
            <Route path="monitor" element={<Monitor />} />
            <Route path="tasks" element={<Tasks />} />
            <Route path="tasks/:id" element={<TaskDetail />} />
            <Route path="alerts" element={<Alerts />} />
            <Route path="alert-rules" element={<AlertRules />} />
            <Route path="agent-tokens" element={<AgentTokens />} />
            <Route path="audit-logs" element={<AuditLogs />} />
            <Route path="tenants" element={<Tenants />} />
            <Route path="users" element={<Users />} />
            <Route path="settings" element={<Settings />} />
            <Route path="ai-chat" element={<AIChat />} />
            <Route path="llm-config" element={<LLMConfig />} />
            <Route path="probes" element={<Probes />} />
            <Route path="cron-jobs" element={<CronJobs />} />
            <Route path="plugins" element={<Plugins />} />
            <Route path="cluster-nodes" element={<NodeManagement />} />
            <Route path="security" element={<Security />} />
            <Route path="hosts/:uuid/assets" element={<AssetInventory />} />
          </Suspense>
        </Route>
        <Route path="*" element={<NotFound />} />
      </Routes>
    </BrowserRouter>
  );
}
