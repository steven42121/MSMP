name: Pull Request
title: "feat(scope): <description>"
labels: []
body:
  - type: markdown
    attributes:
      value: |
        请遵循 [Conventional Commits](https://www.conventionalcommits.org/) 命名 PR。
  - type: input
    id: relates
    attributes:
      label: 关联 Issue
      description: 此 PR 修复了哪个 Issue？（如 #12）
  - type: textarea
    id: description
    attributes:
      label: 变更说明
      description: 简要描述此 PR 做了什么、为什么做
    validations:
      required: true
  - type: textarea
    id: testing
    attributes:
      label: 测试方法
      description: 如何验证此变更？（命令、步骤、预期结果）
    validations:
      required: true
  - type: checkboxes
    id: checklist
    attributes:
      label: 提交前检查
      options:
        - label: `make test` 通过
        - label: `make vet` 无警告
        - label: 前端 `npm run build` 无报错
        - label: CHANGELOG.md `[Unreleased]` 已更新
        - label: 涉及 API 变更时 `INTERFACES.md` 已同步
    validations:
      required: true
