import { useEffect, useRef } from 'react';

const LERP = 0.07;
const IDLE_DELAY = 3000;

/**
 * 全局高光位置跟踪
 * - 桌面：跟随鼠标，平滑插值
 * - 手机/平板：跟随重力感应（设备倾斜），iOS 13+ 首次触摸时静默请求权限
 * - 空闲 3 秒后：高光做缓慢的李萨如曲线漂移，玻璃永远处于流动状态
 */
export function useGlobalMouseTracker() {
  const rafRef = useRef(null);
  const posRef = useRef({ x: 50, y: 50 });
  const targetRef = useRef({ x: 50, y: 50 });
  const lastInputRef = useRef(0);

  useEffect(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

    const handleMouseMove = (e) => {
      lastInputRef.current = performance.now();
      targetRef.current = {
        x: (e.clientX / window.innerWidth) * 100,
        y: (e.clientY / window.innerHeight) * 100,
      };
    };

    // 重力感应：设备倾斜映射高光位置
    const handleOrientation = (e) => {
      if (e.gamma == null && e.beta == null) return;
      lastInputRef.current = performance.now();
      // gamma 左右倾斜 ±35° 满量程；beta 前后倾斜 -20°~50° 映射上下
      const gx = Math.max(-35, Math.min(35, e.gamma ?? 0));
      const by = Math.max(-20, Math.min(50, e.beta ?? 0));
      targetRef.current = {
        x: 50 + (gx / 35) * 45,
        y: 50 + (by / 70) * 60,
      };
    };

    // iOS 13+ 要求用户手势触发权限请求；其他平台直接监听
    const requestOrientation = () => {
      const DO = window.DeviceOrientationEvent;
      if (DO && typeof DO.requestPermission === 'function') {
        DO.requestPermission()
          .then((state) => {
            if (state === 'granted') {
              window.addEventListener('deviceorientation', handleOrientation);
            }
          })
          .catch(() => {});
      } else if (DO) {
        window.addEventListener('deviceorientation', handleOrientation);
      }
      document.removeEventListener('touchend', requestOrientation);
    };
    document.addEventListener('touchend', requestOrientation);

    const animate = (t) => {
      // 空闲漂移：高光沿李萨如曲线缓慢游走
      if (performance.now() - lastInputRef.current > IDLE_DELAY) {
        const s = t / 1000;
        targetRef.current = {
          x: 50 + Math.sin(s * 0.32) * 26 + Math.sin(s * 0.11) * 10,
          y: 50 + Math.cos(s * 0.24) * 22 + Math.cos(s * 0.09) * 8,
        };
      }

      const current = posRef.current;
      const target = targetRef.current;
      posRef.current = {
        x: current.x + (target.x - current.x) * LERP,
        y: current.y + (target.y - current.y) * LERP,
      };
      document.documentElement.style.setProperty('--mouse-x', `${posRef.current.x}%`);
      document.documentElement.style.setProperty('--mouse-y', `${posRef.current.y}%`);
      rafRef.current = requestAnimationFrame(animate);
    };

    window.addEventListener('mousemove', handleMouseMove);
    rafRef.current = requestAnimationFrame(animate);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('touchend', requestOrientation);
      window.removeEventListener('deviceorientation', handleOrientation);
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, []);
}
