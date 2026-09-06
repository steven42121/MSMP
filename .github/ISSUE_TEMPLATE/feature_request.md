name: Feature Request
labels: ["enhancement"]
body:
  - type: markdown
    attributes:
      value: |
        感谢提出新功能建议！请描述你的需求。
  - type: textarea
    id: problem
    attributes:
      label: 痛点
      description: 当前缺少这个功能时，你遇到了什么问题？
    validations:
      required: true
  - type: textarea
    id: solution
    attributes:
      label: 期望方案
      description: 你希望这个功能如何工作？
    validations:
      required: true
  - type: textarea
    id: alternatives
    attributes:
      label: 替代方案
      description: 你有没有考虑过其他实现方式？
  - type: checkboxes
    id: checklist
    attributes:
      label: 确认
      options:
        - label: 我已搜索现有 Issue 和 PR，确认没有重复建议
          required: true
