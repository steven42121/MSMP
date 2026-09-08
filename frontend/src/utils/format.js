// 公共格式化工具函数，供各页面复用，避免多处重复定义导致行为不一致。

// formatBytes 字节数转可读字符串（1024 进制），空/0/非法值返回 '-'。
export function formatBytes(bytes) {
  if (bytes === null || bytes === undefined || bytes <= 0) return '-';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1);
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

// formatNetSpeed 网络速率转可读字符串（B/s 到 MB/s）。
export function formatNetSpeed(v) {
  if (!v || v <= 0) return '0 B/s';
  if (v >= 1e6) return (v / 1e6).toFixed(2) + ' MB/s';
  if (v >= 1e3) return (v / 1e3).toFixed(1) + ' KB/s';
  return v.toFixed(0) + ' B/s';
}