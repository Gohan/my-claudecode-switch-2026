# 架构设计

本文档描述 Claude Switch 的系统架构、模块划分和代码组织。

## 1. 整体架构

### 1.1 分层架构

```
┌─────────────────────────────────────────────┐
│              表现层 (UI)                     │
│  ┌─────────────┐      ┌─────────────────┐   │
│  │    TUI      │      │      GUI        │   │
│  │  (BubbleTea)│      │     (Fyne)      │   │
│  └──────┬──────┘      └────────┬────────┘   │
│         │                      │            │
│         └──────────┬───────────┘            │
│                    ▼                        │
│           ┌─────────────────┐               │
│           │   UI Adapter    │               │
│           │  (状态转换层)    │               │
│           └────────┬────────┘               │
└────────────────────┼────────────────────────┘
                     │
┌────────────────────┼────────────────────────┐
│                    ▼                        │
│           ┌─────────────────┐               │
│           │    业务层        │               │
│           │   (core/manager)│               │
│           └────────┬────────┘               │
│                    │                        │
│         ┌─────────┼─────────┐              │
│         ▼         ▼         ▼              │
│  ┌──────────┐ ┌────────┐ ┌────────┐        │
│  │ profile  │ │ runner │ │ config │        │
│  │ 存储管理  │ │进程启动 │ │ 配置   │        │
│  └──────────┘ └────────┘ └────────┘        │
│                                             │
│              基础设施层                      │
└─────────────────────────────────────────────┘
```

### 1.2 架构原则

1. **单一职责** - 每个模块只负责一件事
2. **依赖倒置** - 业务层不依赖 UI 层
3. **开闭原则** - 新增 UI 类型无需修改业务代码
4. **DRY** - 业务逻辑只写一次，多处复用

## 2. 模块详解

### 2.1 internal/core - 业务核心

**职责**：封装所有业务逻辑，提供统一接口给 UI 层

**主要类型**：

```go
type Manager struct {
    mu sync.RWMutex
    profiles []profile.Profile
    current  map[string]interface{}
    listeners []func(Event)
}

type Event struct {
    Type    EventType
    Payload interface{}
    Error   error
}
```

**核心方法**：

| 方法 | 说明 |
|------|------|
| `Load()` | 加载所有 Profile |
| `Create(name, settings)` | 创建新 Profile |
| `Update(name, settings)` | 更新 Profile |
| `Delete(name)` | 删除 Profile |
| `Apply(idx)` | 应用 Profile 到当前配置 |
| `Run(idx)` | 用指定 Profile 启动 Claude |

**事件机制**：

UI 层通过 `Subscribe()` 监听业务事件，实现数据绑定：

```go
manager.Subscribe(func(e Event) {
    switch e.Type {
    case EventProfilesLoaded:
        // 刷新列表
    case EventProfileApplied:
        // 显示成功提示
    case EventError:
        // 显示错误
    }
})
```

### 2.2 internal/profile - Profile 存储

**职责**：Profile 的持久化存储和序列化

**关键函数**：

```go
func ProfilesDir() string                    // 存储目录
func List() ([]Profile, []ListError)         // 列出所有 Profile
func GetByName(name string) (*Profile, error) // 按名称获取
func Save(name string, settings map[string]interface{}) error
func Delete(name string) error
func ApplyProfile(p Profile) error           // 应用到 ~/.claude/settings.json
func LoadCurrent() (map[string]interface{}, error)
```

**存储格式**：

```
~/.claude-switch/
├── profiles/
│   ├── zai.json
│   ├── tencent.json
│   └── kimi.json
└── runs/              # 运行时目录
    ├── zai/
    │   └── settings.json
    └── tencent/
        └── settings.json
```

### 2.3 internal/runner - 进程启动

**职责**：用指定的 CLAUDE_CONFIG_DIR 启动 Claude 进程

**关键函数**：

```go
func RunDir() string                                    // 运行时目录
func PrepareRunDir(profileName string, settings map[string]interface{}) (string, error)
func Run(configDir string) error                        // 启动并阻塞
func BuildCommand(configDir string) *exec.Cmd          // 构建命令（给 GUI 用）
```

**跨平台处理**：

| 平台 | 实现方式 |
|------|----------|
| Unix | `cmd.Env = append(os.Environ(), "CLAUDE_CONFIG_DIR="+configDir)` |
| Windows | 同上，Go 的 exec 包自动处理 |

### 2.4 internal/config - 配置管理

**职责**：管理应用配置路径

```go
func ConfigDir() string      // ~/.claude-switch
func ProfilesDir() string    // ~/.claude-switch/profiles
func RunDir() string         // ~/.claude-switch/runs
```

## 3. UI 层设计

### 3.1 TUI 架构 (BubbleTea)

**模式**：Elm 架构 (Model-Update-View)

```
┌─────────┐    Msg     ┌─────────┐
│  Model  │◄───────────│  Update │
└────┬────┘            └─────────┘
     │
     │ View
     ▼
┌─────────┐
│  String │ ──► 终端渲染
└─────────┘
```

**状态流转**：

```
viewList ──► viewConfirmApply ──► 应用成功 ──► viewList
    │
    ├──► viewPreview ──► viewConfirmApply
    │
    ├──► viewCreateMenu ──► viewSaveXXX ──► 保存成功 ──► viewList
    │
    ├──► viewConfirmDelete ──► 删除成功 ──► viewList
    │
    └──► viewConfirmRun ──► 启动 Claude（退出 TUI）
```

### 3.2 GUI 架构 (Fyne)

**模式**：组件化 + 数据绑定

```
┌─────────────────────────────────────┐
│           MainWindow                │
├─────────────────────────────────────┤
│  Toolbar: [New][Run][Apply][Delete] │
├─────────────────────────────────────┤
│           ProfileList               │
│  ┌─────────────────────────────┐    │
│  │ ProfileItem                 │    │
│  │ ProfileItem                 │    │
│  │ ProfileItem                 │    │
│  └─────────────────────────────┘    │
├─────────────────────────────────────┤
│           StatusBar                 │
└─────────────────────────────────────┘
```

**状态绑定**：

```go
type GUIState struct {
    manager *core.Manager
    window  fyne.Window
    list    *widget.List
    toolbar *Toolbar
    status  *widget.Label
}

// 监听业务事件，更新 UI
func (s *GUIState) bindEvents() {
    s.manager.Subscribe(func(e core.Event) {
        s.refreshUI()
    })
}
```

## 4. 代码复用策略

### 4.1 复用矩阵

| 模块 | TUI | GUI | 复用方式 |
|------|-----|-----|----------|
| core.Manager | ✅ | ✅ | 直接 import |
| profile | ✅ | ✅ | 直接 import |
| runner | ✅ | ✅ | 直接 import |
| config | ✅ | ✅ | 直接 import |
| TUI model | ✅ | ❌ | - |
| GUI components | ❌ | ✅ | - |

### 4.2 避免重复

**业务逻辑** - 只在 core 中实现一次：

```go
// ❌ 错误：在 TUI 和 GUI 各写一遍
func (m TUIModel) createProfile() { ... }
func (g GUIState) createProfile() { ... }

// ✅ 正确：统一到 core
func (m *Manager) Create(name string, settings map[string]interface{}) { ... }
```

**平台差异** - 用运行时检测：

```go
func buildCommand(configDir string) *exec.Cmd {
    switch runtime.GOOS {
    case "windows":
        return buildWindowsCommand(configDir)
    default:
        return buildUnixCommand(configDir)
    }
}
```

## 5. 数据流

### 5.1 创建 Profile 流程

```
用户操作
    │
    ▼
┌─────────────┐
│  UI Layer   │ ──► 显示表单，收集输入
└──────┬──────┘
       │
       ▼ 调用
┌─────────────┐
│   Manager   │ ──► 校验、保存到文件
│   .Create() │
└──────┬──────┘
       │ 触发
       ▼
┌─────────────┐
│    Event    │ ──► EventProfileCreated
└──────┬──────┘
       │
       ▼ 通知
┌─────────────┐
│  UI Update  │ ──► 刷新列表、显示成功
└─────────────┘
```

### 5.2 Run Profile 流程

```
用户点击 Run
    │
    ▼
┌─────────────┐
│   Manager   │
│   .Run()    │
└──────┬──────┘
       │
       ├──► runner.PrepareRunDir() ──► 创建隔离目录
       │
       └──► runner.Run() ──► 启动 Claude 进程
                                 │
                                 ▼
                           阻塞直到 Claude 退出
                                 │
                                 ▼
                           返回结果给 UI
```

## 6. 扩展性设计

### 6.1 新增服务商

添加新的预设配置，只需修改：

1. `internal/profile/profile.go` - 添加 DefaultXXXProfile()
2. `internal/tui/model.go` - 添加菜单项和视图（TUI）
3. `cmd/gui/components/form.go` - 添加选项（GUI）

### 6.2 新增 UI 类型

如需新增 Web UI（Wails）：

1. 创建 `cmd/claude-switch-web/` 目录
2. 复用 `internal/core/manager.go`
3. 实现 Web 前端，通过 HTTP/WebSocket 调用 Manager

### 6.3 插件机制（未来）

预留扩展点：

```go
type Plugin interface {
    Name() string
    OnProfileCreate(p Profile) error
    OnProfileApply(p Profile) error
}

func (m *Manager) RegisterPlugin(p Plugin) {
    m.plugins = append(m.plugins, p)
}
```

## 7. 依赖关系图

```
cmd/claude-switch/
    └── internal/tui
            ├── internal/core ◄──────┐
            ├── internal/profile     │
            ├── internal/runner      │
            └── internal/config      │
                                      │
cmd/claude-switch-gui/                │
    └── cmd/gui/components           │
            ├── internal/core ◄──────┘
            ├── internal/profile
            ├── internal/runner
            └── internal/config

internal/core
    ├── internal/profile
    ├── internal/runner
    └── internal/config
```

## 8. 错误处理策略

### 8.1 错误分类

| 类型 | 示例 | 处理方式 |
|------|------|----------|
| 用户错误 | Profile 名已存在 | UI 提示，不触发 Event |
| 系统错误 | 磁盘满 | EventError + 详细日志 |
| 外部错误 | Claude 启动失败 | EventError + 用户提示 |

### 8.2 错误传播

```go
// 业务层包装错误
if err := profile.Save(name, settings); err != nil {
    return fmt.Errorf("save profile failed: %w", err)
}

// UI 层处理
manager.Subscribe(func(e Event) {
    if e.Error != nil {
        showErrorDialog(e.Error.Error())
        log.Printf("Error: %+v", e.Error) // 详细堆栈
    }
})
```
