import React from 'react';
import { Card, Button, message } from 'antd';

export default function Settings() {
  return (
    <div>
      <h2>系统设置</h2>
      <Card>
        <p>配置管理系统参数</p>
        <Button onClick={() => message.info('功能开发中')}>保存设置</Button>
      </Card>
    </div>
  );
}