import { useEffect, useRef } from 'react';

/**
 * 全局鼠标跟踪 Hook
 * 在 document 级别跟踪鼠标位置，并更新 CSS 变量
 */
export function useGlobalMouseTracker() {
  const rafRef = useRef(null);
  const posRef = useRef({ x: 50, y: 50 });
  const targetRef = useRef({ x: 50, y: 50 });

  useEffect(() => {
    const handleMouseMove = (e) => {
      // 归一化到 0-100 范围
      targetRef.current = {
        x: (e.clientX / window.innerWidth) * 100,
        y: (e.clientY / window.innerHeight) * 100,
      };
    };

    const animate = () => {
      const current = posRef.current;
      const target = targetRef.current;
      const dx = target.x - current.x;
      const dy = target.y - current.y;

      // 平滑插值
      posRef.current = {
        x: current.x + dx * 0.08,
        y: current.y + dy * 0.08,
      };

      // 更新全局 CSS 变量
      document.documentElement.style.setProperty('--mouse-x', `${posRef.current.x}%`);
      document.documentElement.style.setProperty('--mouse-y', `${posRef.current.y}%`);

      rafRef.current = requestAnimationFrame(animate);
    };

    window.addEventListener('mousemove', handleMouseMove);
    rafRef.current = requestAnimationFrame(animate);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, []);
}
