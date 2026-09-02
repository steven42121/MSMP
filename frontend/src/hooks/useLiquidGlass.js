import { useEffect, useRef, useState } from 'react';

/**
 * 鼠标跟踪高光 Hook
 * 返回 ref 挂载到元素，自动计算鼠标相对位置并更新 CSS 变量
 */
export function useMouseGlow(intensity = 0.6, smoothness = 8) {
  const ref = useRef(null);
  const [glow, setGlow] = useState({ x: 50, y: 50 });
  const rafRef = useRef(null);
  const targetRef = useRef({ x: 50, y: 50 });

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const handleMouseEnter = () => {
      el.style.transition = 'none';
    };

    const handleMouseMove = (e) => {
      const rect = el.getBoundingClientRect();
      const x = ((e.clientX - rect.left) / rect.width) * 100;
      const y = ((e.clientY - rect.top) / rect.height) * 100;
      targetRef.current = { x, y };
    };

    const handleMouseLeave = () => {
      targetRef.current = { x: 50, y: 50 };
    };

    const animate = () => {
      const current = glow;
      const target = targetRef.current;
      const dx = target.x - current.x;
      const dy = target.y - current.y;

      setGlow({
        x: current.x + dx / smoothness,
        y: current.y + dy / smoothness,
      });

      if (Math.abs(dx) > 0.1 || Math.abs(dy) > 0.1) {
        rafRef.current = requestAnimationFrame(animate);
      }
    };

    el.addEventListener('mouseenter', handleMouseEnter);
    el.addEventListener('mousemove', handleMouseMove);
    el.addEventListener('mouseleave', handleMouseLeave);

    rafRef.current = requestAnimationFrame(animate);

    return () => {
      el.removeEventListener('mouseenter', handleMouseEnter);
      el.removeEventListener('mousemove', handleMouseMove);
      el.removeEventListener('mouseleave', handleMouseLeave);
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, [smoothness]);

  return { ref, glow, intensity };
}

/**
 * 3D 倾斜效果 Hook
 * 根据鼠标位置计算卡片倾斜角度
 */
export function useTilt(maxTilt = 5, smoothness = 10) {
  const ref = useRef(null);
  const [tilt, setTilt] = useState({ rotateX: 0, rotateY: 0 });
  const rafRef = useRef(null);
  const targetRef = useRef({ rotateX: 0, rotateY: 0 });

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const handleMouseMove = (e) => {
      const rect = el.getBoundingClientRect();
      const x = (e.clientX - rect.left) / rect.width - 0.5;
      const y = (e.clientY - rect.top) / rect.height - 0.5;
      targetRef.current = {
        rotateX: -y * maxTilt,
        rotateY: x * maxTilt,
      };
    };

    const handleMouseLeave = () => {
      targetRef.current = { rotateX: 0, rotateY: 0 };
    };

    const animate = () => {
      const current = tilt;
      const target = targetRef.current;
      const dx = target.rotateX - current.rotateX;
      const dy = target.rotateY - current.rotateY;

      setTilt({
        rotateX: current.rotateX + dx / smoothness,
        rotateY: current.rotateY + dy / smoothness,
      });

      if (Math.abs(dx) > 0.01 || Math.abs(dy) > 0.01) {
        rafRef.current = requestAnimationFrame(animate);
      }
    };

    el.addEventListener('mousemove', handleMouseMove);
    el.addEventListener('mouseleave', handleMouseLeave);

    rafRef.current = requestAnimationFrame(animate);

    return () => {
      el.removeEventListener('mousemove', handleMouseMove);
      el.removeEventListener('mouseleave', handleMouseLeave);
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, [maxTilt, smoothness]);

  return { ref, tilt };
}

/**
 * 重力感应 Hook（移动端）
 * 使用 deviceorientation 事件
 */
export function useGravity(maxTilt = 8) {
  const ref = useRef(null);
  const [tilt, setTilt] = useState({ rotateX: 0, rotateY: 0 });

  useEffect(() => {
    const handleOrientation = (e) => {
      const gamma = e.gamma || 0; // 左右倾斜 -90 到 90
      const beta = e.beta || 0;   // 前后倾斜 -180 到 180

      // 限制范围并映射到倾斜角度
      const clamp = (v, min, max) => Math.min(max, Math.max(min, v));
      setTilt({
        rotateX: clamp((beta - 45) / 45 * maxTilt, -maxTilt, maxTilt),
        rotateY: clamp(gamma / 45 * maxTilt, -maxTilt, maxTilt),
      });
    };

    if (window.DeviceOrientationEvent) {
      window.addEventListener('deviceorientation', handleOrientation);
    }

    return () => {
      window.removeEventListener('deviceorientation', handleOrientation);
    };
  }, [maxTilt]);

  return { ref, tilt };
}
