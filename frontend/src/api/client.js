import axios from 'axios';
import { message } from 'antd';
import { useAuthStore } from '../store/auth';

// ClusterClient 支持多节点轮询 + 自动故障转移
class ClusterClient {
  constructor(urls) {
    this.nodes = urls.map(url => ({
      url: url.replace(/\/$/, ''),
      failures: 0,
      disabledUntil: 0,
      successCount: 0,
    }));
    this.currentIndex = 0;
    this.baseURL = '/api';
    this.timeout = 15000;
  }

  getActiveNodes() {
    const now = Date.now();
    return this.nodes.filter(n => n.disabledUntil === 0 || n.disabledUntil < now);
  }

  async request(method, path, options = {}) {
    const activeNodes = this.getActiveNodes();
    if (activeNodes.length === 0) {
      return Promise.reject(new Error('所有后端节点暂时不可用'));
    }

    const errors = [];
    let lastError = null;

    for (let i = 0; i < activeNodes.length; i++) {
      const node = activeNodes[this.currentIndex % activeNodes.length];
      this.currentIndex = (this.currentIndex + 1) % activeNodes.length;

      try {
        const resp = await this.fetchWithTimeout(node.url, method, path, options);
        node.failures = 0;
        node.successCount++;
        return resp;
      } catch (err) {
        lastError = err;
        node.failures++;
        errors.push(`${node.url}: ${err.message}`);
        if (node.failures >= 3) {
          node.disabledUntil = Date.now() + 60_000;
        }
      }
    }

    return Promise.reject(new Error(`所有节点请求失败: ${errors.join('; ')}`));
  }

  async fetchWithTimeout(baseURL, method, path, options = {}) {
    const url = baseURL + this.baseURL + path;
    const config = {
      method,
      url,
      timeout: this.timeout,
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    };

    const { token, tenant } = useAuthStore.getState();
    if (token) config.headers.Authorization = `Bearer ${token}`;
    if (tenant?.id) config.headers['X-Tenant-Id'] = tenant.id;

    const resp = await axios(config);
    return resp.data;
  }

  get() {
    return (path, params) => this.request('GET', path, params ? { params } : {});
  }

  post() {
    return (path, data) => this.request('POST', path, { data });
  }

  put() {
    return (path, data) => this.request('PUT', path, { data });
  }

  delete() {
    return (path) => this.request('DELETE', path);
  }
}

// 从环境变量读取多节点地址，或 fallback 到单节点
const rawUrls = import.meta.env.VITE_MSMP_SERVER_URLS || '';
const urls = rawUrls
  ? rawUrls.split(',').map(u => u.trim()).filter(Boolean)
  : [import.meta.env.VITE_MSMP_SERVER_URL || window.location.origin];

const clusterClient = new ClusterClient(urls);

// 兼容原有 export default client 的调用方式
export const client = {
  ...clusterClient,
  interceptors: {
    request: { use: () => {}, eject: () => {} },
    response: { use: () => {}, eject: () => {} },
  },
  create: () => client,
};

export default clusterClient;
