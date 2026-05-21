import React, { useEffect, useState } from 'react';
import { Table, Card, Button, Modal, Form, Input } from 'antd';
import client from '../api/client';

export default function Tenants() {
  const [tenants, setTenants] = useState([]);
  const [loading, setLoading] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const resp = await client.get('/tenants');
      setTenants(Array.isArray(resp) ? resp : resp.data || []);
    } catch (e) {
      setTenants([]);
    }
    setLoading(false);
  };

  useEffect(() => { 累oad(); }, []);

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '标识', dataIndex: 'slug', key: 'slug' },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
  ];

  return (
    <div>
      <h2>租户管理</h2>
      <Card>
        <Table columns={columns} dataSource={tenants} rowKey="id" loading={loading} />
      </Card>
    </div>
  );
}