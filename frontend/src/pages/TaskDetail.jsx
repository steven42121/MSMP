import React from 'react';
import { useParams } from 'react-router-dom';
import { Card } from 'antd';

export default function TaskDetail() {
  const { id } = useParams();
  return (
    <div>
      <h2>任务详情</h2>
      <Card><p>任务 ID: {id}</p></Card>
    </div>
  );
}