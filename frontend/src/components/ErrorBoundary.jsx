import React from 'react';
import { Result, Button } from 'antd';

export default class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { error: null };
  }

  static getDerivedStateFromError(error) {
    return { error };
  }

  componentDidCatch(error, info) {
    console.error('Page error:', error, info);
  }

  render() {
    if (this.state.error) {
      return (
        <Result
          status="500"
          title="页面出错了"
          subTitle={String(this.state.error)}
          extra={<Button type="primary" onClick={() => window.location.reload()}>刷新</Button>}
        />
      );
    }
    return this.props.children;
  }
}
