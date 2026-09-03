import React from 'react';

/**
 * 液态玻璃容器 — 统一磨砂材质与均匀折射。
 * 高光由全局背景层 .global-glow 提供，卡片本身不绘制独立光点。
 */
export default function LiquidGlass({
  children,
  style = {},
  className = '',
  childrenClassName = '',
  ...props
}) {
  return (
    <div
      className={`liquid-glass ${className}`}
      style={style}
      {...props}
    >
      <div className="glass-fluid" />
      <div className="glass-reflection" />
      <div className={childrenClassName}>{children}</div>
    </div>
  );
}