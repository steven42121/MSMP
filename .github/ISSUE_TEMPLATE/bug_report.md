name: Bug Report
labels: ["bug"]
body:
  - type: markdown
    attributes:
      value: |
        感谢报告 Bug！请提供以下信息帮助我们定位问题。
  - type: input
    id: version
    attributes:
      label: 版本
      description: MSMP 版本号（如 v0.1.0），以及 Agent 版本
    validations:
      required: true
  - type: input
    id: os
    attributes:
      label: 操作系统
      description: 服务器 OS 和版本（如 Ubuntu 22.04、Windows Server 2022）
    validations:
      required: true
  - type: textarea
    id: what-happened
    attributes:
      label: 复现步骤
      description: 描述触发 Bug 的操作步骤
      placeholder: |
        1. 登录页面
        2. 点击 XX
        3. 看到错误 YY
    validations:
      required: true
  - type: textarea
    id: expected
    attributes:
      label: 期望行为
      description: 你期望发生什么
    validations:
      required: true
  - type: textarea
    id: actual
    attributes:
      label: 实际行为
      description: 实际发生了什么（错误信息、截图等）
    validations:
      required: true
  - type: textarea
    id: logs
    attributes:
      label: 相关日志
      description: 后端日志（server/logs）或浏览器控制台输出
      render: text
  - type: checkboxes
    id: checklist
    attributes:
      label: 确认
      options:
        - label: 我已搜索现有 Issue，确认没有重复报告
          required: true
