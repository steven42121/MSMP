import axios from 'axios';
import { message } from 'antd';
import { useAuthStore } from '../store/auth';

const client = axios.create({
  baseURL: '/api',
  timeout: 15000,
});

client.interceptors.request.use((config) => {
  const { token, tenant } = useAuthStore.getState();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  if (tenant?.id) config.headers['X-Tenant-Id'] = tenant.id;
  return config;
});

client.interceptors.response.use(
  (resp) => resp.data,
  (err) => {
    const status = err.response?.status;
    if (status === 401) {
      useAuthStore.getState().logout();
      message.error('登录已失效，请重新登录');
      if (location.pathname !== '/login') location.href = '/login';
    } else if (status >= 500) {
      message.error('服务异常，请稍后重试');
    } else if (err.response?.data?.message) {
      message.error(err.response.data.message);
    }
    return Promise.reject(err);
  }
);

export default client;
