import React, { useEffect, useRef, useState } from 'react';
import { Table, Button, Space, Breadcrumb, Modal, Input, Popconfirm, message, Upload, Tag, Tooltip } from 'antd';
import {
  FolderOutlined, FileOutlined, ArrowUpOutlined, ReloadOutlined,
  UploadOutlined, PlusOutlined, DownloadOutlined, DeleteOutlined, EditOutlined, HomeOutlined,
} from '@ant-design/icons';
import axios from 'axios';
import { useAuthStore } from '../store/auth';
import client from '../api/client';

function formatSize(bytes) {
  if (!bytes && bytes !== 0) return '-';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function FileManager({ host }) {
  const [path, setPath] = useState('/');
  const [files, setFiles] = useState([]);
  const [loading, setLoading] = useState(false);
  const [mkdirOpen, setMkdirOpen] = useState(false);
  const [mkdirName, setMkdirName] = useState('');
  const [renameTarget, setRenameTarget] = useState(null);
  const [renameName, setRenameName] = useState('');
  const uploadRef = useRef(null);

  const token = () => useAuthStore.getState().token;

  const apiHeaders = () => ({
    Authorization: `Bearer ${token()}`,
  });

  const loadFiles = async (dir) => {
    setLoading(true);
    try {
      const data = await client.get()(`/hosts/${host.uuid}/files`, { path: dir });
      setPath(data.path || dir);
      setFiles(data.files || []);
    } catch (e) {
      message.error(e?.response?.data?.error || e.message || '加载文件列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadFiles('/');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [host.uuid]);

  const navigateTo = (p) => loadFiles(p);

  const goUp = () => {
    if (path === '/') return;
    const parent = path.split('/').slice(0, -1).join('/') || '/';
    navigateTo(parent);
  };

  const breadcrumbItems = () => {
    const parts = path.split('/').filter(Boolean);
    const items = [{ title: <span onClick={() => navigateTo('/')} style={{ cursor: 'pointer' }}><HomeOutlined /> 根目录</span> }];
    let acc = '';
    parts.forEach((p) => {
      acc += '/' + p;
      const target = acc;
      items.push({ title: <span onClick={() => navigateTo(target)} style={{ cursor: 'pointer' }}>{p}</span> });
    });
    return items;
  };

  const downloadFile = async (filePath) => {
    try {
      const resp = await axios.get(`/api/hosts/${host.uuid}/files/download`, {
        params: { path: filePath },
        headers: apiHeaders(),
        responseType: 'blob',
        timeout: 120000,
      });
      const url = URL.createObjectURL(resp.data);
      const a = document.createElement('a');
      a.href = url;
      a.download = filePath.split('/').pop() || 'download';
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (e) {
      message.error(e?.response?.data?.error || '下载失败');
    }
  };

  const uploadFile = async (file) => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('path', path);
    try {
      await axios.post(`/api/hosts/${host.uuid}/files/upload`, formData, {
        headers: { ...apiHeaders(), 'Content-Type': 'multipart/form-data' },
        timeout: 120000,
      });
      message.success(`已上传 ${file.name}`);
      loadFiles(path);
    } catch (e) {
      const errText = e?.response?.data?.error;
      message.error(typeof errText === 'string' ? errText : '上传失败');
    }
    return false;
  };

  const deletePath = async (targetPath, isDir) => {
    try {
      await client.delete()(`/hosts/${host.uuid}/files`, { params: { path: targetPath } });
      message.success('已删除');
      loadFiles(path);
    } catch (e) {
      message.error(e?.response?.data?.error || '删除失败');
    }
  };

  const createDir = async () => {
    if (!mkdirName.trim()) return;
    try {
      const newPath = (path === '/' ? '' : path) + '/' + mkdirName.trim();
      await client.post()(`/hosts/${host.uuid}/files/mkdir`, { path: newPath });
      message.success('目录已创建');
      setMkdirOpen(false);
      setMkdirName('');
      loadFiles(path);
    } catch (e) {
      message.error(e?.response?.data?.error || '创建目录失败');
    }
  };

  const doRename = async () => {
    if (!renameTarget || !renameName.trim()) return;
    try {
      const newPath = renameTarget.path.split('/').slice(0, -1).concat(renameName.trim()).join('/');
      await client.post()(`/hosts/${host.uuid}/files/rename`, { old_path: renameTarget.path, new_path: newPath });
      message.success('已重命名');
      setRenameTarget(null);
      setRenameName('');
      loadFiles(path);
    } catch (e) {
      message.error(e?.response?.data?.error || '重命名失败');
    }
  };

  const columns = [
    {
      title: '名称', dataIndex: 'name', key: 'name',
      render: (name, record) => (
        <Space size={6} style={{ cursor: record.is_dir ? 'pointer' : 'default' }} onClick={() => record.is_dir && navigateTo(record.path)}>
          {record.is_dir ? <FolderOutlined style={{ color: '#faad14' }} /> : <FileOutlined style={{ color: '#8c8c8c' }} />}
          <span style={{ color: record.is_dir ? '#1677ff' : 'inherit' }}>{name}</span>
        </Space>
      ),
    },
    { title: '大小', dataIndex: 'size', key: 'size', width: 100, render: (v, r) => r.is_dir ? '-' : formatSize(v) },
    { title: '权限', dataIndex: 'mode', key: 'mode', width: 110, render: (v) => <Tag style={{ fontSize: 11, fontFamily: 'monospace' }}>{v}</Tag> },
    { title: '修改时间', dataIndex: 'mod_time', key: 'mod_time', width: 160 },
    {
      title: '操作', key: 'action', width: 200,
      render: (_, record) => (
        <Space size={0}>
          {!record.is_dir && (
            <Tooltip title="下载">
              <Button type="link" size="small" icon={<DownloadOutlined />} onClick={() => downloadFile(record.path)} />
            </Tooltip>
          )}
          <Tooltip title="重命名">
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => { setRenameTarget(record); setRenameName(record.name); }} />
          </Tooltip>
          <Popconfirm
            title={record.is_dir ? '确认删除该目录？（需为空目录）' : '确认删除该文件？'}
            onConfirm={() => deletePath(record.path, record.is_dir)}
            okText="删除" cancelText="取消"
          >
            <Button type="link" danger size="small" icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Space direction="vertical" style={{ width: '100%' }} size={12}>
        <Space wrap>
          <Breadcrumb items={breadcrumbItems()} />
        </Space>
        <Space wrap>
          <Button size="small" icon={<ArrowUpOutlined />} onClick={goUp} disabled={path === '/'}>上级</Button>
          <Button size="small" icon={<ReloadOutlined />} onClick={() => loadFiles(path)}>刷新</Button>
          <Button size="small" icon={<PlusOutlined />} onClick={() => setMkdirOpen(true)}>新建目录</Button>
          <Button size="small" type="primary" icon={<UploadOutlined />} onClick={() => uploadRef.current?.click()}>上传文件</Button>
          <input
            ref={uploadRef}
            type="file"
            style={{ display: 'none' }}
            onChange={(e) => { if (e.target.files[0]) uploadFile(e.target.files[0]); e.target.value = ''; }}
          />
          <span style={{ color: '#888', fontSize: 12 }}>当前路径：{path}</span>
        </Space>
        <Table
          size="small"
          rowKey={(r) => r.path}
          dataSource={files}
          loading={loading}
          columns={columns}
          pagination={false}
          locale={{ emptyText: path === '/' ? '空目录' : '无法访问该路径' }}
        />
      </Space>

      <Modal
        title="新建目录"
        open={mkdirOpen}
        onOk={createDir}
        onCancel={() => { setMkdirOpen(false); setMkdirName(''); }}
        okText="创建" cancelText="取消"
      >
        <Input
          placeholder="目录名"
          value={mkdirName}
          onChange={(e) => setMkdirName(e.target.value)}
          onPressEnter={createDir}
        />
      </Modal>

      <Modal
        title="重命名"
        open={!!renameTarget}
        onOk={doRename}
        onCancel={() => { setRenameTarget(null); setRenameName(''); }}
        okText="确定" cancelText="取消"
      >
        <Input
          placeholder="新名称"
          value={renameName}
          onChange={(e) => setRenameName(e.target.value)}
          onPressEnter={doRename}
        />
      </Modal>
    </div>
  );
}

export default FileManager;