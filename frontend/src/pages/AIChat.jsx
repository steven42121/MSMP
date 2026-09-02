import React, { useEffect, useState, useRef, useCallback } from 'react';
import { Card, Input, Button, Space, Typography, message, Spin, Divider, Tag, Modal, Statistic } from 'antd';
import {
  SendOutlined, RobotOutlined, BulbOutlined, ThunderboltOutlined,
  CheckCircleOutlined, CloseCircleOutlined, ConsoleSqlOutlined,
  DesktopOutlined, FileTextOutlined, PlayCircleOutlined,
} from '@ant-design/icons';
import client from '../api/client';

const { Text, Paragraph } = Typography;

const SUGGEST_QUERIES = [
  { icon: <BulbOutlined style={{ color: '#667eea' }} />, label: '系统健康状态如何？' },
  { icon: <ThunderboltOutlined style={{ color: '#faad14' }} />, label: '最近 24 小时有哪些告警？' },
  { icon: <DesktopOutlined style={{ color: '#52c41a' }} />, label: '列出所有主机状态' },
  { icon: <ConsoleSqlOutlined style={{ color: '#1890ff' }} />, label: '检查 nginx 服务状态' },
];

const TOOL_ICONS = {
  list_hosts: <DesktopOutlined />,
  get_host_status: <ConsoleSqlOutlined />,
  get_recent_alerts: <ThunderboltOutlined />,
  execute_command: <PlayCircleOutlined />,
  check_service: <ConsoleSqlOutlined />,
  view_logs: <FileTextOutlined />,
  generate_report: <BulbOutlined />,
};

const TOOL_LABELS = {
  list_hosts: '列出主机',
  get_host_status: '查询主机状态',
  get_recent_alerts: '查看最近告警',
  execute_command: '执行命令',
  check_service: '检查服务',
  view_logs: '查看日志',
  generate_report: '生成报告',
};

export default function AIChat() {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [pendingTool, setPendingTool] = useState(null);
  const [toolResult, setToolResult] = useState(null);
  const [approvals, setApprovals] = useState([]);
  const bottomRef = useRef(null);
  const inputRef = useRef(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, pendingTool, toolResult]);

  // Poll for pending approvals
  useEffect(() => {
    const poll = async () => {
      try {
        const resp = await client.get()('/mcp/approvals');
        setApprovals(resp.approvals || []);
      } catch (_) {}
    };
    poll();
    const timer = setInterval(poll, 5000);
    return () => clearInterval(timer);
  }, []);

  const handleSend = async (query) => {
    const q = query || input.trim();
    if (!q) return;
    setInput('');
    setMessages((prev) => [...prev, { role: 'user', content: q }]);
    setLoading(true);
    try {
      const resp = await client.post()('/ai/chat', { query: q });
      const reply = resp.reply || '暂无回复';
      const toolCalls = resp.tool_calls || [];

      const assistantMsg = { role: 'assistant', content: reply };
      if (toolCalls.length > 0) {
        assistantMsg.toolCalls = toolCalls;
      }
      setMessages((prev) => [...prev, assistantMsg]);

      if (toolCalls.length > 0) {
        const tc = toolCalls[0];
        setPendingTool({
          id: tc.id,
          name: tc.name,
          args: tc.arguments ? JSON.parse(tc.arguments) : {},
          text: reply,
        });
      }
    } catch (e) {
      setMessages((prev) => [...prev, { role: 'assistant', content: '请求失败，请检查 LLM 配置。' }]);
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async () => {
    if (!pendingTool) return;
    setLoading(true);
    try {
      const resp = await client.post()(
        `/mcp/propose`,
        {
          tool_name: pendingTool.name,
          arguments: pendingTool.args,
          message: `AI 建议执行: ${pendingTool.name}`,
        }
      );
      const approvalId = resp.id;

      // Auto-approve after user confirms in UI
      await client.post()(`/mcp/approvals/${approvalId}/approve`);

      setMessages((prev) => [...prev, {
        role: 'system',
        content: `已批准操作: ${TOOL_LABELS[pendingTool.name] || pendingTool.name}`,
      }]);

      setPendingTool(null);

      // Poll for result
      const pollResult = async () => {
        try {
          const resp = await client.get()(`/mcp/approvals`);
          const completed = (resp.approvals || []).find((a) => a.id === approvalId && a.status !== 'pending');
          if (completed) {
            setToolResult(completed.result || '执行完成');
          } else {
            setTimeout(pollResult, 3000);
          }
        } catch (_) {
          setTimeout(pollResult, 3000);
        }
      };
      pollResult();
    } catch (e) {
      message.error('审批失败: ' + (e.message || '未知错误'));
      setPendingTool(null);
    } finally {
      setLoading(false);
    }
  };

  const handleReject = () => {
    setMessages((prev) => [...prev, {
      role: 'system',
      content: `已拒绝操作: ${TOOL_LABELS[pendingTool?.name] || pendingTool?.name}`,
    }]);
    setPendingTool(null);
  };

  const handleQuickQuery = (q) => {
    setInput(q);
    setTimeout(() => handleSend(q), 50);
  };

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 20 }}>
        <div>
          <div className="page-title">AI 智能助手</div>
          <Text type="secondary" style={{ fontSize: 13 }}>基于大模型的系统运维智能问答与 MCP 操作</Text>
        </div>
        {approvals.length > 0 && (
          <Tag color="gold" style={{ borderRadius: 20, padding: '2px 10px', fontSize: 12 }}>
            {approvals.length} 待审批
          </Tag>
        )}
      </div>

      <div className="ai-chat-grid">
        <Card
          style={{ borderRadius: 16, border: 'none', boxShadow: 'none', height: 'calc(100vh - 220px)', minHeight: 400, display: 'flex', flexDirection: 'column' }}
          className="liquid-glass"
        >
          <div style={{ flex: 1, overflowY: 'auto', padding: '8px 4px', className: 'chat-scroll' }}>
            {messages.length === 0 && (
              <div style={{ textAlign: 'center', paddingTop: 60 }}>
                <RobotOutlined style={{ fontSize: 48, color: '#667eea', marginBottom: 16 }} />
                <div style={{ fontSize: 18, fontWeight: 600, color: 'rgba(255,255,255,0.9)', marginBottom: 8 }}>你好，我是 MSMP 智能运维助手</div>
                <Text type="secondary">我可以帮你分析告警、查询系统状态、执行运维操作（需你授权）</Text>
              </div>
            )}

            {messages.map((msg, i) => (
              <div key={i} style={{ marginBottom: 16, display: 'flex', justifyContent: msg.role === 'user' ? 'flex-end' : msg.role === 'system' ? 'center' : 'flex-start' }}>
                {msg.role === 'system' ? (
                  <div style={{ padding: '6px 14px', borderRadius: 20, background: 'rgba(250,173,20,0.15)', border: '0.5px solid rgba(250,173,20,0.3)', color: '#faad14', fontSize: 12 }}>
                    {msg.content}
                  </div>
                ) : (
                  <div
                    className={msg.role === 'user' ? 'msg-bubble-user' : 'msg-bubble-assistant'}
                    style={{
                      maxWidth: '80%',
                      padding: '10px 16px',
                      borderRadius: msg.role === 'user' ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
                      color: '#fff',
                      whiteSpace: 'pre-wrap',
                      lineHeight: 1.6,
                      fontSize: 14,
                    }}
                  >
                    {msg.content}
                    {msg.toolCalls && msg.toolCalls.length > 0 && (
                      <div style={{ marginTop: 8, paddingTop: 8, borderTop: '0.5px solid rgba(255,255,255,0.15)' }}>
                        <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.5)' }}>等待审批操作...</Text>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}

            {loading && !pendingTool && (
              <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-start' }}>
                <div className="msg-bubble-assistant" style={{ padding: '10px 16px', borderRadius: '16px 16px 16px 4px', fontSize: 13 }}>
                  <Spin size="small" style={{ marginRight: 8 }} />
                  思考中...
                </div>
              </div>
            )}

            {pendingTool && (
              <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-start' }}>
                <div className="tool-proposal-card" style={{ padding: 16, minWidth: 320, maxWidth: 480 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
                    <span style={{ color: '#667eea', fontSize: 18 }}>{TOOL_ICONS[pendingTool.name] || <BulbOutlined />}</span>
                    <Text strong style={{ color: 'rgba(255,255,255,0.9)', fontSize: 14 }}>
                      {TOOL_LABELS[pendingTool.name] || pendingTool.name}
                    </Text>
                    <Tag color="orange" style={{ borderRadius: 8, fontSize: 11, marginLeft: 'auto' }}>需审批</Tag>
                  </div>
                  <div style={{ background: 'rgba(0,0,0,0.2)', borderRadius: 8, padding: '8px 12px', marginBottom: 12 }}>
                    <pre style={{ margin: 0, fontSize: 12, color: 'rgba(255,255,255,0.7)', whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
                      {JSON.stringify(pendingTool.args, null, 2)}
                    </pre>
                  </div>
                  {pendingTool.text && (
                    <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 12 }}>
                      {pendingTool.text}
                    </Text>
                  )}
                  <Space>
                    <Button
                      type="primary"
                      icon={<CheckCircleOutlined />}
                      onClick={handleApprove}
                      loading={loading}
                      style={{ background: 'linear-gradient(135deg, #52c41a 0%, #13c2c2 100%)', border: 'none', borderRadius: 8 }}
                    >
                      批准执行
                    </Button>
                    <Button
                      danger
                      icon={<CloseCircleOutlined />}
                      onClick={handleReject}
                      disabled={loading}
                      style={{ borderRadius: 8 }}
                    >
                      拒绝
                    </Button>
                  </Space>
                </div>
              </div>
            )}

            {toolResult && (
              <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-start' }}>
                <div className="msg-bubble-assistant" style={{ padding: '10px 16px', borderRadius: '16px 16px 16px 4px', fontSize: 13 }}>
                  <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 6 }} />
                  <Text style={{ color: 'rgba(255,255,255,0.8)' }}>执行结果：</Text>
                  <pre style={{ margin: '8px 0 0', fontSize: 12, color: 'rgba(255,255,255,0.6)', whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
                    {toolResult}
                  </pre>
                </div>
              </div>
            )}
            <div ref={bottomRef} />
          </div>

          <Divider style={{ margin: '8px 0', borderColor: 'rgba(255,255,255,0.1)' }} />
          <Space.Compact style={{ width: '100%' }}>
            <Input
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onPressEnter={handleSend}
              placeholder="输入运维问题，例如：检查 web-server-01 的磁盘使用情况，或重启 nginx 服务"
              style={{ background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,255,255,0.12)', color: '#fff' }}
              disabled={loading}
            />
            <Button
              type="primary"
              onClick={() => handleSend()}
              loading={loading}
              icon={<SendOutlined />}
              style={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none' }}
            >
              发送
            </Button>
          </Space.Compact>
        </Card>

        <div>
          <Card
            style={{ borderRadius: 16, border: 'none', boxShadow: 'none', marginBottom: 12 }}
            className="liquid-glass"
            title={<span style={{ color: '#667eea' }}>快速提问</span>}
          >
            {SUGGEST_QUERIES.map((q, i) => (
              <div
                key={i}
                onClick={() => handleQuickQuery(q.label)}
                style={{
                  padding: '10px 12px',
                  borderRadius: 10,
                  marginBottom: 8,
                  cursor: 'pointer',
                  background: 'rgba(102,126,234,0.1)',
                  border: '1px solid rgba(102,126,234,0.2)',
                  transition: 'all 0.2s',
                }}
                onMouseEnter={(e) => { e.currentTarget.style.background = 'rgba(102,126,234,0.2)'; }}
                onMouseLeave={(e) => { e.currentTarget.style.background = 'rgba(102,126,234,0.1)'; }}
              >
                <Space>
                  {q.icon}
                  <Text style={{ fontSize: 13 }}>{q.label}</Text>
                </Space>
              </div>
            ))}
          </Card>

          <Card
            style={{ borderRadius: 16, border: 'none', boxShadow: 'none' }}
            className="liquid-glass"
            title={<span style={{ color: '#52c41a' }}>可用工具</span>}
          >
            <div style={{ color: '#aaa', fontSize: 12, lineHeight: 2 }}>
              {Object.entries(TOOL_LABELS).map(([name, label]) => (
                <div key={name} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span style={{ color: '#667eea', fontSize: 10 }}>{TOOL_ICONS[name]}</span>
                  <span>{label}</span>
                  {name !== 'list_hosts' && name !== 'get_host_status' && name !== 'get_recent_alerts' && name !== 'generate_report' && (
                    <Tag color="orange" style={{ fontSize: 10, marginLeft: 'auto', padding: '0 4px' }}>需审批</Tag>
                  )}
                </div>
              ))}
            </div>
          </Card>

          {approvals.length > 0 && (
            <Card
              style={{ borderRadius: 16, border: 'none', boxShadow: 'none' }}
              className="liquid-glass"
              title={<span style={{ color: '#faad14' }}>待审批操作</span>}
              extra={<span style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)' }}>{approvals.length} 项</span>}
            >
              {approvals.map((a) => (
                <div key={a.id} style={{ padding: '8px 10px', borderRadius: 8, background: 'rgba(250,173,20,0.1)', border: '0.5px solid rgba(250,173,20,0.2)', marginBottom: 8 }}>
                  <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.8)', marginBottom: 4 }}>{a.message}</div>
                  <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)' }}>工具: {a.tool_name}</div>
                </div>
              ))}
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
