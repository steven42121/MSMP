// 公共格式化工具函数，供各页面复用，避免多处重复定义导致行为不一致。

// formatBytes 字节数转可读字符串（1024 进制）。
// 无效值(空/NaN/负)返回 '-'，0 保留为 '0 B'（0 是有效值，图表刻度需正常显示）。
export function formatBytes(bytes) {
  const n = Number(bytes);
  if (!Number.isFinite(n) || n < 0) return '-';
  if (n === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const i = Math.min(Math.floor(Math.log(n) / Math.log(k)), sizes.length - 1);
  return parseFloat((n / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

// formatNetSpeed 网络速率转可读字符串（B/s 到 MB/s）。
export function formatNetSpeed(v) {
  if (!v || v <= 0) return '0 B/s';
  if (v >= 1e6) return (v / 1e6).toFixed(2) + ' MB/s';
  if (v >= 1e3) return (v / 1e3).toFixed(1) + ' KB/s';
  return v.toFixed(0) + ' B/s';
}