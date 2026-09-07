# PM2 Ecosystem Config for MSMP
# 用法: pm2 start ecosystem.config.js
#       pm2 save && pm2 startup

module.exports = {
  apps: [
    {
      name: 'msmp-server',
      script: './dist/msmp-server',
      cwd: '/opt/msmp',
      instances: 1,
      exec_mode: 'fork',
      env_production: {
        NODE_ENV: 'production',
      },
      // 崩溃自动重启
      restart_delay: 5000,
      max_restarts: 10,
      min_uptime: '10s',
      // 日志
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      error_file: '/var/log/msmp/server-error.log',
      out_file: '/var/log/msmp/server-out.log',
      merge_logs: true,
      // 内存限制
      max_memory_restart: '1G',
      // 优雅退出
      kill_timeout: 5000,
      listen_timeout: 8000,
    },
    {
      name: 'msmp-frontend',
      script: './node_modules/.bin/vite',
      cwd: './frontend',
      instances: 1,
      exec_mode: 'fork',
      env_production: {
        NODE_ENV: 'production',
      },
      restart_delay: 3000,
      max_restarts: 5,
      error_file: '/var/log/msmp/frontend-error.log',
      out_file: '/var/log/msmp/frontend-out.log',
      merge_logs: true,
    },
  ],
};
