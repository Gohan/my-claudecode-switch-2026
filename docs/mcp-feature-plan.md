# MCP 管理功能实施计划

## 概述

为 claude-switch 增加 MCP (Model Context Protocol) 服务器管理功能，支持为每个 profile 独立配置 MCP 服务器。

## 设计原则

1. **Single Source of Truth**: Profile 配置存储在 `~/.claude-switch/profiles/<name>/` 目录
   - `settings.json` - 环境变量、模型等配置
   - `.claude.json` - MCP 服务器配置

2. **Symlink 启动方案**: Runner 创建临时目录，通过符号链接指向 profile 配置，保持 profile 目录纯净

3. **UI 一致性**: 复用现有的弹窗、列表、表单等交互模式

---

## 阶段划分

### Phase 0: 重构现有配置存储（前置准备）

**目标**: 将现有的 "复制 settings 到 runs 目录" 改为 "symlink 指向纯净配置"

#### 0.1 现状分析
- 当前: `PrepareRunDir(profileName, settings)` 将 settings **写入** `runs/<name>/settings.json`
- 目标: `PrepareRunDir(profileName)` 创建 **symlink** 指向 `profiles/<name>/settings.json`

#### 0.2 Profile Repository 调整
- [ ] 确保 `internal/repository/profile.go` 有 `Save(profileName, settings)` 方法
  - 将 settings 写入 `~/.claude-switch/profiles/<name>/settings.json`
  - 而不是写入 runs 目录

#### 0.3 Service 层调整
- [ ] 修改 `internal/service/profile.go` 的 `Apply()` 方法
  - Apply 时：调用 repository.Save() 保存到 profiles 目录
  - 不再直接操作 runs 目录

#### 0.4 Runner 改造
- [ ] 修改 `internal/runner/runner.go`
  - `PrepareRunDir(profileName string) (string, error)`
    - 保持运行目录: `~/.claude-switch/runs/<profileName>/`
    - 如果 `runs/<name>/settings.json` 是普通文件，删除它
    - 创建 symlink: `runs/<name>/settings.json` → `profiles/<name>/settings.json`
    - 如果 profiles 目录没有该文件，报错（profile 未保存）
  - 移除 `writeSettings()` 函数（不再需要）

#### 0.5 TUI 调用方调整
- [ ] 修改 `internal/tui/model.go`
  - `updateConfirmRun()` 中: `runner.PrepareRunDir(p.Name)`（去掉 `p.Settings` 参数）
  - `updateConfirmApply()` 中: 确保先调用 `service.Apply()` 保存到 profiles，再提示成功

#### 0.6 Default Profile 处理
- [ ] 识别 "default" 虚拟 profile
- [ ] default profile 的 symlink 指向 `~/.claude/settings.json`

**验收标准**:
- 现有功能（创建、应用、运行 profile）正常工作
- Profile 配置保存在 `profiles/<name>/settings.json`（纯净配置）
- Runs 目录通过 symlink 引用 profiles 目录
- Default profile 正确映射到 ~/.claude/settings.json

---

### Phase 1: MCP 基础设施 (Domain & Repository)

**目标**: 建立 MCP 数据模型和存储层（依赖 Phase 0 完成）

#### 1.1 Domain 层
- [ ] 创建 `internal/domain/mcp.go`
  - `MCPServer` 结构体: Name, Type, Command, Args, URL, Env, Enabled
  - `MCPConfig` 结构体: Servers 列表
  - 验证函数: `ValidateMCPServerName()`, `ValidateMCPServer()`

#### 1.2 Repository 层
- [ ] 创建 `internal/repository/mcp.go`
  - `MCPRepository` 接口
    - `Load(profileName string) (*domain.MCPConfig, error)`
    - `Save(profileName string, config *domain.MCPConfig) error`
  - `MCPRepositoryFS` 实现
    - 读取/写入 `~/.claude-switch/profiles/<name>/.claude.json`

#### 1.3 Profile 结构调整
- [ ] 修改 `internal/domain/profile.go`
  - `Profile` 结构体增加 `HasMCP` 标记（用于 UI 显示指示器）

**验收标准**:
- 单元测试通过
- 能正确读写 `.claude.json` 文件
- 文件不存在时返回空配置而非错误

---

### Phase 2: Runner 支持 MCP 配置

**目标**: 在 Phase 0 基础上，增加 MCP 配置的 symlink（依赖 Phase 0 和 Phase 1）

#### 2.1 Runner 修改
- [ ] 修改 `internal/runner/runner.go` 的 `PrepareRunDir()`
  - 创建 symlink: `.claude.json` -> `~/.claude-switch/profiles/<profileName>/.claude.json`
  - 如果 profile 没有 `.claude.json`（空 MCP 配置）：
    - 方案 A: 不创建 symlink（Claude 使用默认 MCP 配置）
    - 方案 B: 创建空的 `.claude.json` 文件

#### 2.2 Default Profile MCP 处理
- [ ] default profile 的 `.claude.json` symlink 指向 `~/.claude/.claude.json`

**验收标准**:
- MCP 配置能通过 symlink 正确加载
- 空 MCP 配置处理合理
- default profile 的 MCP 配置正确映射

---

### Phase 3: TUI - Profile 操作菜单

**目标**: 在主列表增加二级菜单入口

#### 3.1 新增 View State
- [ ] `viewProfileMenu` - Profile 操作选择菜单

#### 3.2 修改主列表交互
- [ ] 修改 `updateList()`
  - `[enter]` 从直接 apply 改为进入 `viewProfileMenu`
  - `[a]` 保持为快捷 apply（向后兼容）

#### 3.3 Profile 菜单实现
- [ ] `updateProfileMenu()` - 按键处理
  - `[1]/[a]` - Apply Profile → `viewConfirmApply`
  - `[2]/[m]` - Manage MCP → `viewMCPList`
  - `[esc]/[q]` - 返回 `viewList`

- [ ] `viewProfileMenu()` - 渲染
  ```
  ┌─────────────────────────────┐
  │  Profile: xxx               │
  │                             │
  │  [1] Apply Profile          │
  │  [2] Manage MCP Servers (3) │
  │                             │
  │  [esc] cancel               │
  └─────────────────────────────┘
  ```

**验收标准**:
- 菜单弹窗正确显示
- 选项计数正确（如 MCP 数量）
- 按键响应正确

---

### Phase 4: TUI - MCP 列表管理

**目标**: 实现 MCP 服务器的列表查看、启用/禁用

#### 4.1 新增 View States
- [ ] `viewMCPList` - MCP 列表
- [ ] `viewMCPConfirmDelete` - 删除确认

#### 4.2 MCP 列表界面
- [ ] `updateMCPList()` - 按键处理
  - `[↑/↓]` - 光标移动
  - `[enter]` - 切换 Enable/Disable
  - `[e]` - 编辑 → `viewMCPEdit` (Phase 5 实现)
  - `[n]` - 新建 → `viewMCPEdit`
  - `[d]` - 删除 → `viewMCPConfirmDelete`
  - `[q]/[esc]` - 返回 `viewProfileMenu`

- [ ] `viewMCPList()` - 渲染
  ```
  🔌 MCP Servers for: my-profile

    ● fetch         stdio  npx ...
    ○ github        http   (disabled)

  [enter] toggle  [e] edit  [n] new  [d] delete  [q] back
  ```

#### 4.3 删除确认弹窗
- [ ] `updateMCPConfirmDelete()` - y/n/esc 处理
- [ ] `viewMCPConfirmDeleteModal()` - 弹窗渲染

**验收标准**:
- 列表正确显示所有 MCP
- Enable/Disable 状态切换即时生效
- 删除操作有确认保护

---

### Phase 5: TUI - MCP 编辑界面

**目标**: 实现 MCP 服务器的新增和编辑

#### 5.1 新增 View State
- [ ] `viewMCPEdit` - MCP 编辑表单

#### 5.2 表单字段
- Name (文本输入)
- Type (选择: stdio / http)
- Command (文本输入, stdio 时显示)
- Args (文本输入，空格分隔，stdio 时显示)
- URL (文本输入, http 时显示)
- Env (键值对列表，可添加/删除)

#### 5.3 交互设计
- [ ] 分步骤表单或单屏表单（根据复杂度决定）
- [ ] `[tab]` 切换字段
- [ ] `[enter]` 保存
- [ ] `[esc]` 取消

#### 5.4 验证
- [ ] 名称唯一性检查
- [ ] 必填字段验证
- [ ] 类型相关字段条件显示

**验收标准**:
- 能成功新建 MCP
- 能成功编辑现有 MCP
- 验证错误有明确提示

---

### Phase 6: 集成与测试

**目标**: 端到端验证和细节打磨

#### 6.1 集成测试
- [ ] 完整流程测试
  1. 创建 profile
  2. 添加 MCP
  3. 启动验证 MCP 加载
  4. 切换 profile 验证隔离性

#### 6.2 Default Profile 测试
- [ ] default profile 的 MCP 管理
- [ ] 正确读写 ~/.claude/.claude.json

#### 6.3 边界情况
- [ ] 空 MCP 列表处理
- [ ] 无效 JSON 文件处理
- [ ] 并发访问处理（如同时编辑）

#### 6.4 帮助文本更新
- [ ] 更新主列表 help 文本
- [ ] 各界面帮助文本完善

**验收标准**:
- 端到端流程完整可用
- 错误处理完善
- 帮助信息准确

---

## 文件变更清单

### 新增文件
```
internal/domain/mcp.go           # MCP 数据结构
internal/repository/mcp.go       # MCP 存储接口
internal/repository/mcp_fs.go    # MCP 文件系统实现
internal/repository/mcp_test.go  # MCP 存储测试
```

### 修改文件
```
internal/domain/profile.go       # 增加 HasMCP 标记
internal/runner/runner.go        # Symlink 启动机制
internal/tui/model.go            # 新增 view states 和 handlers
internal/tui/styles.go           # 可能需要新增样式
```

---

## 技术细节

### .claude.json 格式
```json
{
  "mcpServers": {
    "fetch": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-fetch"],
      "env": {},
      "enabled": true
    },
    "github": {
      "type": "http",
      "url": "https://api.github.com/mcp",
      "enabled": false
    }
  }
}
```

### Symlink 结构示例
```
~/.claude-switch/runs/my-profile/        # 运行目录（CLAUDE_CONFIG_DIR）
├── projects/                            # 运行时创建（Claude Code 生成）
├── ide/                                 # 运行时创建（Claude Code 生成）
├── settings.json -> ../../profiles/my-profile/settings.json
└── .claude.json -> ../../profiles/my-profile/.claude.json
```

### 目录关系
```
~/.claude-switch/
├── profiles/                            # 配置存储（纯净，只有配置）
│   ├── my-profile/
│   │   ├── settings.json               # 用户编辑或 TUI 修改
│   │   └── .claude.json                # MCP 配置
│   └── ...
├── runs/                                # 运行目录（含运行时生成的缓存）
│   ├── my-profile/                     # CLAUDE_CONFIG_DIR 指向这里
│   │   ├── projects/                   # Claude Code 生成的项目历史
│   │   ├── ide/                        # IDE 锁文件等
│   │   ├── settings.json -> ...        # symlink 到 profiles
│   │   └── .claude.json -> ...         # symlink 到 profiles
│   └── ...
```

---

## 风险与应对

| 风险 | 影响 | 应对策略 |
|------|------|---------|
| Claude Code 不支持某些 MCP 配置 | 高 | 先验证最小可用配置，逐步扩展 |
| Symlink 在 Windows 需要权限 | 中 | 使用 Junction 点或回退到复制方案 |
| 现有用户 profile 迁移 | 低 | 自动创建空 .claude.json，无感升级 |

---

## 里程碑

| 阶段 | 预估时间 | 交付物 |
|------|---------|--------|
| Phase 0 | 1-2h | 重构配置存储为 symlink 模式 |
| Phase 1 | 1h | MCP Domain + Repository 层 |
| Phase 2 | 1h | Runner 支持 MCP symlink |
| Phase 3 | 1h | Profile 菜单界面 |
| Phase 4 | 2h | MCP 列表管理界面 |
| Phase 5 | 2-3h | MCP 编辑界面 |
| Phase 6 | 2h | 测试与集成 |

**总计**: 约 1-2 天工作量

---

## 备注

- 保持与现有代码风格一致
- 优先实现核心路径（stdio 类型 MCP）
- http 类型 MCP 可作为第二阶段增强
