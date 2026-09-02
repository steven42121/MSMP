# frontend — 前端应用

本目录是 MSMP 的 Web UI，使用 React + Vite + Ant Design 构建。

## 结构

```
frontend/
├── package.json            # 依赖与脚本
├── vite.config.js          # Vite 配置（代理、allowedHosts）
├── index.html              # SPA 入口
└── src/
    ├── main.jsx            # React 入口
    ├── App.jsx             # 路由定义与认证守卫
    ├── api/
    │   └── client.js       # axios 实例（自动附加 Authorization header）
    ├── layouts/
    │   └── MainLayout.jsx  # 侧边栏 + 顶栏布局
    ├── store/
    │   └── auth.js         # Zustand 认证状态（token、user）
    ├── styles/
    │   └── global.css      # 全局样式
    └── pages/
        ├── Login.jsx           # 登录页
        ├── Dashboard.jsx       # 概览
        ├── HostList.jsx        # 主机列表
        ├── HostDetail.jsx      # 主机详情（含采集渠道 tab）
        ├── Monitor.jsx         # 监控曲线页
        ├── Alerts.jsx          # 告警列表
        ├── AlertRules.jsx      # 告警规则管理
        ├── Tasks.jsx           # 任务列表
        ├── TaskDetail.jsx      # 任务详情
        ├── Tenants.jsx         # 租户管理
        ├── Users.jsx           # 用户管理
        ├── AgentTokens.jsx     # Agent Token 生成
        ├── AuditLogs.jsx       # 审计日志
        ├── Settings.jsx        # 系统设置
        └── NotFound.jsx        # 404
```

## 关键文件

| 文件 | 目的 |
|------|------|
| `App.jsx` | 路由表定义，RequireAuth 守卫未登录跳转 /login |
| `api/client.js` | axios 实例，自动从 Zustand store 读取 token 附加到请求头 |
| `store/auth.js` | Zustand store，管理 token、user 信息，提供 login/logout |
| `pages/HostDetail.jsx` | 主机详情页，含基本信息/监控/标签/资产/事件/采集渠道六个 tab |
| `pages/Monitor.jsx` | 全局监控页，四折线图（CPU/内存/负载/网络），30s 自动刷新 |

## 依赖

```json
{
  "antd": "^5.21.0",
  "@ant-design/icons": "^5.5.0",
  "axios": "^1.7.7",
  "dayjs": "^1.11.13",
  "echarts": "^5.5.1",
  "echarts-for-react": "^3.0.2",
  "react": "^18.3.1",
  "react-dom": "^18.3.1",
  "react-router-dom": "^6.27.0",
  "zustand": "^5.0.0"
}
```

## 规范

### 新增页面

1. 在 `src/pages/` 创建组件文件（PascalCase）
2. 在 `App.jsx` 的 Routes 中添加 `<Route path="..." element={<Component />}`
3. 在 `MainLayout.jsx` 的菜单中添加对应 menu item

### API 调用

统一使用 `client` 实例（import from `../api/client`），方法为 `client.get/post/put/delete(url, options)`，返回 Promise 解析为响应数据（axios 自动解包 response.data）。
