import React, { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import MainLayout from './layouts/MainLayout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import HostList from './pages/HostList';
import HostDetail from './pages/HostDetail';
import Monitor from './pages/Monitor';
import Tasks from './pages/Tasks';
import TaskDetail from './pages/TaskDetail';
import Tenants from './pages/Tenants';
import Users from './pages/Users';
import Alerts from './pages/Alerts';
import AlertRules from './pages/AlertRules';
import AgentTokens from './pages/AgentTokens';
import AuditLogs from './pages/AuditLogs';
import Settings from './pages/Settings';
import NotFound from './pages/NotFound';
import AIChat from './pages/AIChat';
import LLMConfig from './pages/LLMConfig';
import Probes from './pages/Probes';
import CronJobs from './pages/CronJobs';
import { useAuthStore } from './store/auth';
import { useGlobalMouseTracker } from './hooks/useGlobalMouseTracker';

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
        </Route>
        <Route path="*" element={<NotFound />} />
      </Routes>
    </BrowserRouter>
  );
}
