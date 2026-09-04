import React, { useEffect, useRef, useState } from 'react';
import { Modal, message, Spin } from 'antd';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useAuthStore } from '../store/auth';

function WebSSHTerminal({ open, host, onClose }) {
  const terminalRef = useRef(null);
  const termRef = useRef(null);
  const fitRef = useRef(null);
  const wsRef = useRef(null);
  const [status, setStatus] = useState('connecting'); // connecting | connected | disconnected

  useEffect(() => {
    if (!open) return;
    if (termRef.current) return;

    const terminal = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: 'Menlo, Monaco, Consolas, "Courier New", monospace',
      theme: {
        background: '#1e1e2e',
        foreground: '#cdd6f4',
        cursor: '#f5e0dc',
        black: '#45475a',
        red: '#f38ba8',
        green: '#a6e3a1',
        yellow: '#f9e2af',
        blue: '#89b4fa',
        magenta: '#f5c2e7',
        cyan: '#94e2d5',
        white: '#bac2de',
      },
      scrollback: 2000,
    });
    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(terminalRef.current);
    fitAddon.fit();

    termRef.current = terminal;
    fitRef.current = fitAddon;

    // 建立 WebSocket 连接
    const token = useAuthStore.getState().token;
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${proto}//${window.location.host}/api/hosts/${host.uuid}/ssh?token=${encodeURIComponent(token || '')}`;
    const ws = new WebSocket(wsUrl);
    ws.binaryType = 'arraybuffer';
    wsRef.current = ws;
    setStatus('connecting');

    ws.onopen = () => {
      setStatus('connected');
      // 发送初始终端尺寸
      ws.send(JSON.stringify({ width: terminal.cols, height: terminal.rows }));
      terminal.focus();
    };

    ws.onmessage = (ev) => {
      if (ev.data instanceof ArrayBuffer) {
        terminal.write(new Uint8Array(ev.data));
      } else if (typeof ev.data === 'string') {
        terminal.write(ev.data);
      }
    };

    ws.onclose = (ev) => {
      setStatus('disconnected');
      if (ev.code !== 1000) {
        terminal.write('\r\n\x1b[31m[连接已断开]\x1b[0m\r\n');
      }
    };

    ws.onerror = () => {
      terminal.write('\r\n\x1b[31m[连接错误]\x1b[0m\r\n');
    };

    // 键盘输入 → WebSocket
    terminal.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    // 窗口 resize → 服务器
    const handleResize = () => {
      try {
        fitAddon.fit();
      } catch (e) { /* ignore */ }
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', width: terminal.cols, height: terminal.rows }));
      }
    };
    window.addEventListener('resize', handleResize);

    // 首次 fit 后发送尺寸
    const fitTimer = setInterval(() => {
      try {
        fitAddon.fit();
      } catch (e) { /* ignore */ }
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', width: terminal.cols, height: terminal.rows }));
      }
      clearInterval(fitTimer);
    }, 100);

    return () => {
      window.removeEventListener('resize', handleResize);
      clearInterval(fitTimer);
      ws.close();
      wsRef.current = null;
      terminal.dispose();
      termRef.current = null;
      fitRef.current = null;
    };
  }, [open, host.uuid]);

  return (
    <Modal
      title={
        <span>
          终端 <span style={{ fontSize: 12, color: 'rgba(0,0,0,0.45)' }}>{host.hostname}</span>
          <span style={{
            marginLeft: 12, fontSize: 12,
            color: status === 'connected' ? '#52c41a' : status === 'connecting' ? '#faad14' : '#ff4d4f',
          }}>
            {status === 'connected' ? '已连接' : status === 'connecting' ? '连接中…' : '已断开'}
          </span>
        </span>
      }
      open={open}
      onCancel={onClose}
      footer={null}
      width={900}
      style={{ top: 40 }}
      destroyOnClose
    >
      <div
        ref={terminalRef}
        style={{
          width: '100%',
          height: 480,
          background: '#1e1e2e',
          borderRadius: 8,
          overflow: 'hidden',
          padding: 8,
        }}
      />
    </Modal>
  );
}

export default WebSSHTerminal;