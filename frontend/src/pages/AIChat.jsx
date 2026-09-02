import React, { useEffect, useState, useRef } from 'react';
import { Card, Input, Button, Space, Typography, message, Spin, Divider } from 'antd';
import { SendOutlined, RobotOutlined, BulbOutlined, ThunderboltOutlined } from '@ant-design/icons';
import client from '../api/client';

const { Text, Paragraph } = Typography;

const SUGGEST_QUERIES = [
  { icon: <BulbOutlined style={{ color: '#667eea' }} />, label: '系统健康状态如何？' },
  { icon: <ThunderboltOutlined style={{ color: '#faad14' }} />, label: '最近 24 小时有哪些告警？' },
];

export default function AIChat() {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const bottomRef = useRef(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = async () => {
    const q = input.trim();
    if (!q) return;
    setInput('');
    setMessages((prev) => [...prev, { role: 'user', content: q }]);
    setLoading(true);
    try {
      const resp = await client.post()('/ai/chat', { query: q });
      setMessages((prev) => [...prev, { role: 'assistant', content: resp.reply || '暂无回复' }]);
    } catch (e) {
      setMessages((prev) => [...prev, { role: 'assistant', content: '请求失败，请检查 LLM 配置。' }]);
    } finally {
      setLoading(false);
    }
  };

  const handleQuickQuery = (q) => {
    setInput(q);
    setTimeout(() => handleSend(), 50);
  };

  return (
    <div>
      <div className="page-header" style={{ marginBottom: 20 }}>
        <div>
          <div className="page-title">AI 智能助手</div>
          <Text type="secondary" style={{ fontSize: 13 }}>基于大模型的系统运维智能问答与分析</Text>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 320px', gap: 16, alignItems: 'start' }}>
        <Card
          style={{ borderRadius: 16, border: 'none', boxShadow: 'none', height: 'calc(100vh - 220px)', display: 'flex', flexDirection: 'column' }}
          className="liquid-glass"
        >
          <div style={{ flex: 1, overflowY: 'auto', padding: '8px 4px' }}>
            {messages.length === 0 && (
              <div style={{ textAlign: 'center', paddingTop: 80 }}>
                <RobotOutlined style={{ fontSize: 48, color: '#667eea', marginBottom: 16 }} />
                <div style={{ fontSize: 18, fontWeight: 600, color: '#fff', marginBottom: 8 }}>你好，我是 MSMP 智能运维助手</div>
                <Text type="secondary">我可以帮你分析告警、查询系统状态、生成运维报告</Text>
              </div>
            )}
            {messages.map((msg, i) => (
              <div key={i} style={{ marginBottom: 16, display: 'flex', justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start' }}>
                <div
                  style={{
                    maxWidth: '80%',
                    padding: '10px 16px',
                    borderRadius: msg.role === 'user' ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
                    background: msg.role === 'user'
                      ? 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
                      : 'rgba(255,255,255,0.08)',
                    color: '#fff',
                    whiteSpace: 'pre-wrap',
                    lineHeight: 1.6,
                    fontSize: 14,
                  }}
                >
                  {msg.content}
                </div>
              </div>
            ))}
            {loading && (
              <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-start' }}>
                <div style={{ padding: '10px 16px', borderRadius: '16px 16px 16px 4px', background: 'rgba(255,255,255,0.08)', color: '#aaa', fontSize: 13 }}>
                  <Spin size="small" style={{ marginRight: 8 }} />
                  思考中...
                </div>
              </div>
            )}
            <div ref={bottomRef} />
          </div>
          <Divider style={{ margin: '8px 0', borderColor: 'rgba(255,255,255,0.1)' }} />
          <Space.Compact style={{ width: '100%' }}>
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onPressEnter={handleSend}
              placeholder="输入运维问题，例如：最近的告警原因是什么？"
              style={{ background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,255,255,0.12)', color: '#fff' }}
              disabled={loading}
            />
            <Button
              type="primary"
              onClick={handleSend}
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
            title={<span style={{ color: '#52c41a' }}>功能说明</span>}
          >
            <div style={{ color: '#aaa', fontSize: 12, lineHeight: 1.8 }}>
              <p style={{ margin: '4px 0' }}>• 支持自然语言运维问答</p>
              <p style={{ margin: '4px 0' }}>• 自动关联主机与告警上下文</p>
              <p style={{ margin: '4px 0' }}>• 提供可操作的修复建议</p>
              <p style={{ margin: '4px 0' }}>• 需要先在系统设置中配置 LLM</p>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}
