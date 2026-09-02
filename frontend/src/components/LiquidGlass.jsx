import React, { useEffect, useRef } from 'react';

/**
 * 带鼠标跟踪高光的液态玻璃容器
 * 高光会跟随鼠标位置流动
 */
export default function LiquidGlass({
  children,
  style = {},
  className = '',
  childrenClassName = '',
  ...props
}) {
  const innerRef = useRef(null);

  useEffect(() => {
    const el = innerRef.current;
    if (!el) return;

    const handleMouseMove = (e) => {
      const rect = el.getBoundingClientRect();
      const x = ((e.clientX - rect.left) / rect.width) * 100;
      const y = ((e.clientY - rect.top) / rect.height) * 100;
      el.style.setProperty('--glow-x', `${x}%`);
      el.style.setProperty('--glow-y', `${y}%`);
    };

    const handleMouseLeave = () => {
      el.style.setProperty('--glow-x', '50%');
      el.style.setProperty('--glow-y', '50%');
    };

    el.addEventListener('mousemove', handleMouseMove);
    el.addEventListener('mouseleave', handleMouseLeave);

    return () => {
      el.removeEventListener('mousemove', handleMouseMove);
      el.removeEventListener('mouseleave', handleMouseLeave);
    };
  }, []);

  return (
    <div
      ref={innerRef}
      className={`liquid-glass ${className}`}
      style={style}
      {...props}
    >
      <div className={`glass-fluid`} />
      <div className={`glass-reflection`} />
      <div className={childrenClassName}>{children}</div>
    </div>
  );
}
