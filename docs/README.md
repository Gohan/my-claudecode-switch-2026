# Claude Switch 项目文档

本文档包含 Claude Switch 项目的完整技术文档，涵盖架构设计、开发指南和构建说明。

## 文档索引

| 文档 | 说明 |
|------|------|
| [架构设计](architecture.md) | 系统架构、模块划分、代码组织 |
| [实施计划](implementation-plan.md) | 详细开发计划、里程碑、任务分解 |
| [构建指南](build-guide.md) | 编译、打包、分发说明 |
| [GUI 开发指南](gui-development.md) | Fyne GUI 开发详细说明 |
| [过渡方案](transition-plan.md) | 数据分离、TDD、架构重构计划 |

## 项目概述

Claude Switch 是一个跨平台的 Claude Code 配置管理工具，支持：

- **TUI 版本** - 终端界面，适合开发者和 SSH 环境
- **GUI 版本** - 图形界面，基于 Fyne，适合桌面用户

### 核心功能

- 管理多个 Claude Code API 配置（Profile）
- 支持多家服务商：z.ai、腾讯云、Kimi、阿里百炼
- 快速切换和启动不同配置的 Claude
- 隔离运行环境，互不干扰

### 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| TUI | BubbleTea + Lipgloss |
| GUI | Fyne v2 |
| 构建 | Make + Go build |

## 快速开始

### 安装依赖

```bash
go mod download
```

### 构建 TUI 版本

```bash
make tui
```

### 构建 GUI 版本

```bash
make gui
```

### 运行测试

```bash
go test ./...
```

## 项目结构

```
.
├── internal/           # 内部包
│   ├── core/          # 业务逻辑核心
│   ├── profile/       # Profile 存储
│   ├── runner/        # 进程启动
│   └── config/        # 配置管理
├── cmd/               # 程序入口
│   ├── claude-switch/      # TUI 版本
│   └── claude-switch-gui/  # GUI 版本
├── docs/              # 本文档
├── assets/            # 图标资源
├── build/             # 构建资源
├── Makefile           # 构建脚本
└── go.mod             # Go 模块
```

## 贡献指南

1. 阅读 [架构设计](architecture.md) 了解代码组织
2. 阅读 [实施计划](implementation-plan.md) 了解开发进度
3. 提交 PR 前确保通过所有测试

## 许可证

MIT License
