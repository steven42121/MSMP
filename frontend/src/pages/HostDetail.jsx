import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Descriptions, Card, Spin } from 'antd';
import client from '../api/client';

export default function HostDetail() {
  const { uuid } = useParams();
  const [host, setHost] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    client.get(`/hosts/${uuid}`)
      .then(setHost)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [uuid]);

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  if (!host) return <div>主机不存在</div>;

  return (
    <div>
      <h2>{host.hostname}</h2>
      <Card style={{ marginBottom: 24 }}>
        <Descriptions bordered column={2}>
          <Descriptions.Item label="主机名">{host.hostname}</Descriptions.Item>
          <Descriptions.Item label="UUID">{host.uuid}</Descriptions.Item>
          <Descriptions.Item label="操作系统">{host.os} {host.os_version}</Descriptions.Item>
          <Descriptions.Item label="架构">{host.arch}</Descriptions.Item>
          <Descriptions.Item label="IP">{host.ip}</Descriptions.Item>
          <Descriptions.Item label="公网IP">{host.public_ip}</Descriptions.Item>
          <Descriptions.Item label="CPU">{host.cpu_model} ({host.cpu_cores}核)</Descriptions.Item>
          <Descriptions.Item label="内存">{formatBytes(host.memory_total)}</Descriptions.Item>
          <Descriptions.Item label="磁盘">{formatBytes(host.disk_total)}</Descriptions.Item>
          <Descriptions.Item label="Agent版本">{host.agent_version}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <span style={{ color: host.status === 'online' ? 'green' : 'red' }}>
              {host.status === 'online' ? '在线' : '离线'}
            </span>
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  );
}

function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}