import React from 'react';
import ReactDOM from 'react-dom/client';
import { ConfigProvider, theme as antdTheme } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import 'dayjs/locale/zh-cn';
import App from './App';
import ErrorBoundary from './components/ErrorBoundary';
import { useThemeStore } from './store/theme';
import './styles/global.css';

function ThemeWrapper({ children }) {
  const dark = useThemeStore((s) => s.dark);
  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: dark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
        token: {
          colorPrimary: '#1677ff',
          borderRadius: 6,
        },
      }}
    >
      {children}
    </ConfigProvider>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ThemeWrapper>
      <ErrorBoundary>
        <App />
      </ErrorBoundary>
    </ThemeWrapper>
  </React.StrictMode>
);
