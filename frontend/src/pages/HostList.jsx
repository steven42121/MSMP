import React, { useEffect, useState } from 'react';
import { Table, Button } from 'antd';
import { fetchAgentAssets } from '../api/agents';

const columns = [
    { title: '主机名', dataIndex: 'hostname', key: 'hostname' },
    { title: '操作系统', dataIndex: 'os', key: 'os' },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    { title: 'CPU', dataIndex: 'cpu', key: 'cpu' },
    { title: '内存', dataIndex: 'memory', key: 'memory' },
    { title: '磁盘', dataIndex: 'disk', key: 'disk' },
    { title: 'UUID', dataIndex: 'uuid', key: 'uuid' },
    { title: '操作', key: 'action', render: (_, record) => (
        <Button type="link" href={`/host/${record.uuid}`}>详情</Button>
    )},
];

export default function HostList() {
    const [data, setData] = useState([]);
    useEffect(() => {
        fetchAgentAssets().then(res => setData(res.data)).catch(() => {
            // 如果后端没接口或没数据，可以用临时mock
            setData([
                { hostname: 'Win-Test', os: 'Windows', ip: '192.168.1.1', cpu: 'Intel', memory: '8GB', disk: '256GB', uuid: 'xx' }
            ]);
        });
    }, []);
    return (
        <div>
            <h2>主机资产列表</h2>
            <Table columns={columns} dataSource={data} rowKey="uuid" />
        </div>
    );
}