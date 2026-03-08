# 实施计划

本文档详细描述 TUI + Fyne GUI 混合方案的开发计划、任务分解和里程碑。

## 1. 项目目标

### 1.1 最终交付物

| 产物 | 说明 |
|------|------|
| `claude-switch` | TUI 版本，终端可执行文件 |
| `claude-switch-gui` | GUI 版本，桌面应用程序 |
| 完整源码 | 包含构建脚本和文档 |

### 1.2 功能矩阵

| 功能 | TUI | GUI |
|------|-----|-----|
| 列出 Profile | ✅ | ✅ |
| 创建 Profile | ✅ | ✅ |
| 编辑 Profile | ❌ | ✅ |
| 删除 Profile | ✅ | ✅ |
| 应用 Profile | ✅ | ✅ |
| 运行 Profile | ✅ | ✅ |
| 预览 Profile Diff | ✅ | ✅ |
| 快捷键支持 | ✅ | ✅ |

## 2. 实施阶段

### Phase 0: 代码重构（Day 1）

**目标**：提取业务逻辑到 core 模块，TUI 使用 core.Manager

#### 2.0.1 创建 internal/core/manager.go

```go
package core

import (
    "sync"
    "claude-switch/internal/profile"
    "claude-switch/internal/runner"
)

type Manager struct {
    mu sync.RWMutex
    profiles []profile.Profile
    current map[string]interface{}
    listeners []func(Event)
}

type Event struct {
    Type EventType
    Payload interface{}
    Error error
}

type EventType int

const (
    EventProfilesLoaded EventType = iota
    EventProfileCreated
    EventProfileUpdated
    EventProfileDeleted
    EventProfileApplied
    EventProfileRun
    EventError
)

// 构造函数
func NewManager() *Manager {
    return &Manager{
        current: make(map[string]interface{}),
    }
}

// 加载数据
func (m *Manager) Load() error

// CRUD 操作
func (m *Manager) Create(name string, settings map[string]interface{}) error
func (m *Manager) Update(name string, settings map[string]interface{}) error
func (m *Manager) Delete(name string) error
func (m *Manager) Apply(idx int) error
func (m *Manager) Run(idx int) error

// 查询
func (m *Manager) Profiles() []profile.Profile
func (m *Manager) Current() map[string]interface{}
func (m *Manager) IsActive(idx int) bool

// 事件订阅
func (m *Manager) Subscribe(fn func(Event))
```

#### 2.0.2 重构 TUI 使用 Manager

修改 `internal/tui/model.go`：

```go
type Model struct {
    manager *core.Manager  // 新增
    // ... 其他字段
}

func NewModel() Model {
    m := Model{
        manager: core.NewManager(),
        // ...
    }
    m.manager.Subscribe(m.onEvent)  // 监听事件
    m.loadData()
    return m
}

func (m *Model) loadData() {
    if err := m.manager.Load(); err != nil {
        m.message = fmt.Sprintf("Error: %v", err)
    }
    m.profiles = m.manager.Profiles()
    m.current = m.manager.Current()
}
```

#### 2.0.3 验证清单

- [ ] core.Manager 编译通过
- [ ] TUI 编译通过
- [ ] TUI 所有功能正常
- [ ] 单元测试通过（添加 manager_test.go）

---

### Phase 1: GUI 基础框架（Day 2-3）

**目标**：搭建 Fyne 基础，显示 Profile 列表

#### 2.1.1 添加 Fyne 依赖

```bash
go get fyne.io/fyne/v2
go get fyne.io/fyne/v2/cmd/fyne@latest  # 打包工具
```

#### 2.1.2 创建 cmd/claude-switch-gui/main.go

```go
package main

import (
    "fyne.io/fyne/v2/app"
    "claude-switch/gui"
)

func main() {
    a := app.NewWithID("com.claude-switch.gui")
    a.SetIcon(resourceIconPng)  // 嵌入图标

    w := a.NewWindow("Claude Switch")
    w.Resize(fyne.NewSize(800, 600))
    w.CenterOnScreen()

    mainGui := gui.NewMainWindow(w)
    mainGui.Initialize()

    w.ShowAndRun()
}
```

#### 2.1.3 创建 GUI 核心文件

**cmd/claude-switch-gui/gui/window.go**：

```go
package gui

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
    "claude-switch/internal/core"
)

type MainWindow struct {
    window fyne.Window
    manager *core.Manager

    // 组件
    toolbar *Toolbar
    list *ProfileList
    status *widget.Label
}

func NewMainWindow(w fyne.Window) *MainWindow {
    return &MainWindow{
        window: w,
        manager: core.NewManager(),
    }
}

func (mw *MainWindow) Initialize() {
    // 初始化 Manager
    if err := mw.manager.Load(); err != nil {
        dialog.ShowError(err, mw.window)
    }

    // 订阅事件
    mw.manager.Subscribe(mw.onEvent)

    // 创建组件
    mw.toolbar = NewToolbar(mw.manager, mw.window)
    mw.list = NewProfileList(mw.manager, mw.window)
    mw.status = widget.NewLabel("Ready")

    // 布局
    content := container.NewBorder(
        mw.toolbar,    // Top
        mw.status,     // Bottom
        nil,           // Left
        nil,           // Right
        mw.list,       // Center
    )

    mw.window.SetContent(content)
}

func (mw *MainWindow) onEvent(e core.Event) {
    switch e.Type {
    case core.EventProfilesLoaded:
        mw.list.Refresh()
    case core.EventError:
        dialog.ShowError(e.Error, mw.window)
    }
}
```

**cmd/claude-switch-gui/gui/components/list.go**：

```go
package components

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/widget"
    "claude-switch/internal/core"
    "claude-switch/internal/profile"
)

type ProfileList struct {
    widget.List
    manager *core.Manager
    window  fyne.Window
    selected int
}

func NewProfileList(m *core.Manager, w fyne.Window) *ProfileList {
    pl := &ProfileList{
        manager: m,
        window:  w,
    }

    pl.List = widget.NewList(
        func() int {
            return len(m.Profiles())
        },
        func() fyne.CanvasObject {
            return newProfileListItem()
        },
        func(id widget.ListItemID, item fyne.CanvasObject) {
            profiles := m.Profiles()
            if id < 0 || id >= len(profiles) {
                return
            }
            p := profiles[id]
            item.(*profileListItem).Update(p, m.IsActive(int(id)))
        },
    )

    pl.OnSelected = func(id widget.ListItemID) {
        pl.selected = int(id)
    }

    pl.ExtendBaseWidget(pl)
    return pl
}

func (pl *ProfileList) SelectedIndex() int {
    return pl.selected
}
```

#### 2.1.4 验证清单

- [ ] GUI 编译通过
- [ ] 能显示 Profile 列表
- [ ] 列表数据与 TUI 一致
- [ ] 窗口大小可调整

---

### Phase 2: GUI 核心功能（Day 4-6）

**目标**：实现所有操作功能

#### 2.2.1 Toolbar 组件

**cmd/claude-switch-gui/gui/components/toolbar.go**：

```go
package components

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/widget"
    "claude-switch/internal/core"
)

type Toolbar struct {
    container *fyne.Container
    manager   *core.Manager
    window    fyne.Window
    list      *ProfileList

    // 按钮
    btnNew    *widget.Button
    btnRun    *widget.Button
    btnApply  *widget.Button
    btnDelete *widget.Button
}

func NewToolbar(m *core.Manager, w fyne.Window, list *ProfileList) *Toolbar {
    t := &Toolbar{
        manager: m,
        window:  w,
        list:    list,
    }

    t.btnNew = widget.NewButtonWithIcon("New", theme.ContentAddIcon(), t.onNew)
    t.btnRun = widget.NewButtonWithIcon("Run", theme.MediaPlayIcon(), t.onRun)
    t.btnApply = widget.NewButtonWithIcon("Apply", theme.DocumentSaveIcon(), t.onApply)
    t.btnDelete = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), t.onDelete)

    // 初始状态
    t.updateButtons(-1)

    t.container = container.NewHBox(t.btnNew, t.btnRun, t.btnApply, t.btnDelete)
    return t
}

func (t *Toolbar) Container() fyne.CanvasObject {
    return t.container
}

func (t *Toolbar) onNew() {
    ShowCreateProfileDialog(t.window, t.manager)
}

func (t *Toolbar) onRun() {
    idx := t.list.SelectedIndex()
    if idx < 0 {
        return
    }

    profiles := t.manager.Profiles()
    if idx >= len(profiles) {
        return
    }

    // 确认对话框
    d := dialog.NewConfirm(
        "Run Profile",
        fmt.Sprintf("Run claude with profile '%s'?", profiles[idx].Name),
        func(ok bool) {
            if ok {
                go func() {
                    if err := t.manager.Run(idx); err != nil {
                        dialog.ShowError(err, t.window)
                    }
                }()
            }
        },
        t.window,
    )
    d.Show()
}

func (t *Toolbar) onApply() {
    idx := t.list.SelectedIndex()
    if idx < 0 {
        return
    }

    if err := t.manager.Apply(idx); err != nil {
        dialog.ShowError(err, t.window)
    }
}

func (t *Toolbar) onDelete() {
    idx := t.list.SelectedIndex()
    if idx < 0 {
        return
    }

    profiles := t.manager.Profiles()
    if idx >= len(profiles) {
        return
    }

    d := dialog.NewConfirm(
        "Delete Profile",
        fmt.Sprintf("Delete profile '%s'?", profiles[idx].Name),
        func(ok bool) {
            if ok {
                if err := t.manager.Delete(profiles[idx].Name); err != nil {
                    dialog.ShowError(err, t.window)
                }
            }
        },
        t.window,
    )
    d.Show()
}

func (t *Toolbar) updateButtons(selectedIdx int) {
    hasSelection := selectedIdx >= 0
    t.btnRun.Disable = !hasSelection
    t.btnApply.Disable = !hasSelection
    t.btnDelete.Disable = !hasSelection
}
```

#### 2.2.2 Create Profile 对话框

**cmd/claude-switch-gui/gui/components/dialogs.go**：

```go
package components

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/widget"
    "claude-switch/internal/core"
    "claude-switch/internal/profile"
)

func ShowCreateProfileDialog(w fyne.Window, m *core.Manager) {
    // 类型选择
    typeSelect := widget.NewSelect([]string{
        "Custom",
        "z.ai API",
        "Tencent Cloud",
        "Kimi",
        "Ali BaiLian",
    }, nil)
    typeSelect.SetSelected("Custom")

    // 名称输入
    nameEntry := widget.NewEntry()
    nameEntry.SetPlaceHolder("profile-name")

    // API Key 输入（密码模式）
    apiKeyEntry := widget.NewPasswordEntry()
    apiKeyEntry.SetPlaceHolder("your-api-key")

    // 动态显示/隐藏 API Key
    typeSelect.OnChanged = func(value string) {
        if value == "Custom" {
            apiKeyEntry.Hide()
        } else {
            apiKeyEntry.Show()
        }
    }
    apiKeyEntry.Hide()

    items := []*widget.FormItem{
        widget.NewFormItem("Type", typeSelect),
        widget.NewFormItem("Name", nameEntry),
        widget.NewFormItem("API Key", apiKeyEntry),
    }

    d := dialog.NewForm(
        "Create Profile",
        "Create",
        "Cancel",
        items,
        func(ok bool) {
            if !ok {
                return
            }

            name := nameEntry.Text
            if name == "" {
                dialog.ShowError(fmt.Errorf("name is required"), w)
                return
            }

            // 根据类型获取默认配置
            var settings map[string]interface{}
            switch typeSelect.Selected {
            case "z.ai API":
                settings = profile.DefaultZAIProfile()
                if env, ok := settings["env"].(map[string]interface{}); ok {
                    env["ANTHROPIC_AUTH_TOKEN"] = apiKeyEntry.Text
                }
            case "Tencent Cloud":
                settings = profile.DefaultTencentCloudProfile()
                if env, ok := settings["env"].(map[string]interface{}); ok {
                    env["ANTHROPIC_AUTH_TOKEN"] = apiKeyEntry.Text
                }
            case "Kimi":
                settings = profile.DefaultKimiProfile()
                if env, ok := settings["env"].(map[string]interface{}); ok {
                    env["ANTHROPIC_AUTH_TOKEN"] = apiKeyEntry.Text
                }
            case "Ali BaiLian":
                settings = profile.DefaultAliProfile()
                if env, ok := settings["env"].(map[string]interface{}); ok {
                    env["ANTHROPIC_AUTH_TOKEN"] = apiKeyEntry.Text
                }
            default:
                settings = make(map[string]interface{})
            }

            if err := m.Create(name, settings); err != nil {
                dialog.ShowError(err, w)
            }
        },
        w,
    )

    d.Resize(fyne.NewSize(400, 0))
    d.Show()
}
```

#### 2.2.3 Preview 对话框

```go
func ShowPreviewDialog(w fyne.Window, p profile.Profile, current map[string]interface{}) {
    diff := profile.Diff(current, p.Settings)

    var content strings.Builder
    for _, d := range diff {
        switch d.Status {
        case profile.DiffUnchanged:
            content.WriteString(fmt.Sprintf("  %s: %s\n", d.Key, d.OldValue))
        case profile.DiffModified:
            content.WriteString(fmt.Sprintf("- %s: %s\n", d.Key, d.OldValue))
            content.WriteString(fmt.Sprintf("+ %s: %s\n", d.Key, d.NewValue))
        case profile.DiffAdded:
            content.WriteString(fmt.Sprintf("+ %s: %s\n", d.Key, d.NewValue))
        case profile.DiffRemoved:
            content.WriteString(fmt.Sprintf("- %s: %s\n", d.Key, d.OldValue))
        }
    }

    text := widget.NewMultiLineEntry()
    text.SetText(content.String())
    text.Disable()

    scroll := container.NewScroll(text)
    scroll.SetMinSize(fyne.NewSize(500, 400))

    d := dialog.NewCustom(
        "Preview: "+p.Name,
        "Close",
        scroll,
        w,
    )
    d.Show()
}
```

#### 2.2.4 验证清单

- [ ] 能创建 Profile（所有类型）
- [ ] 能删除 Profile（带确认）
- [ ] 能 Apply Profile
- [ ] 能 Run Profile（启动独立 claude）
- [ ] 能 Preview Profile Diff
- [ ] 操作有错误提示

---

### Phase 3: 细节优化（Day 7-8）

**目标**：提升用户体验

#### 2.3.1 列表项美化

```go
type profileListItem struct {
    container *fyne.Container
    icon      *widget.Icon
    name      *widget.Label
    details   *widget.Label
    active    *widget.Label
}

func newProfileListItem() fyne.CanvasObject {
    item := &profileListItem{
        icon:    widget.NewIcon(theme.DocumentIcon()),
        name:    widget.NewLabel("Profile Name"),
        details: widget.NewLabel("model: opus | url: ..."),
        active:  widget.NewLabel(""),
    }
    item.active.TextStyle = fyne.TextStyle{Bold: true}

    item.container = container.NewHBox(
        item.icon,
        container.NewVBox(item.name, item.details),
        layout.NewSpacer(),
        item.active,
    )

    return item
}

func (i *profileListItem) Update(p profile.Profile, isActive bool) {
    i.name.SetText(p.Name)

    model, url := profile.GetSummary(p.Settings)
    i.details.SetText(fmt.Sprintf("model: %s | url: %s", model, url))

    if isActive {
        i.active.SetText("[active]")
        i.active.Importance = widget.HighImportance
    } else {
        i.active.SetText("")
    }
}
```

#### 2.3.2 状态栏增强

```go
type StatusBar struct {
    container *fyne.Container
    label     *widget.Label
    progress  *widget.ProgressBarInfinite
}

func (sb *StatusBar) SetMessage(msg string)
func (sb *StatusBar) SetLoading(loading bool)
func (sb *StatusBar) SetSuccess(msg string)  // 绿色
func (sb *StatusBar) SetError(msg string)    // 红色
```

#### 2.3.3 快捷键

```go
func (mw *MainWindow) setupShortcuts() {
    // Ctrl+N: New
    mw.window.Canvas().AddShortcut(
        &desktop.CustomShortcut{KeyName: fyne.KeyN, Modifier: desktop.ControlModifier},
        func(shortcut fyne.Shortcut) {
            mw.toolbar.onNew()
        },
    )

    // F5: Refresh
    mw.window.Canvas().AddShortcut(
        &desktop.CustomShortcut{KeyName: fyne.KeyF5},
        func(shortcut fyne.Shortcut) {
            mw.manager.Load()
        },
    )
}
```

#### 2.3.4 验证清单

- [ ] 列表项美观，信息清晰
- [ ] 操作时有加载状态
- [ ] 成功/失败有视觉反馈
- [ ] 快捷键工作正常
- [ ] 窗口大小记忆

---

### Phase 4: 打包分发（Day 9-10）

**目标**：生成可分发文件

#### 2.4.1 资源嵌入

```bash
# 安装 fyne 工具
go install fyne.io/fyne/v2/cmd/fyne@latest

# 打包图标
fyne bundle -name resourceIcon -package main -o cmd/claude-switch-gui/icon.go assets/icon.png
```

#### 2.4.2 Makefile 构建

```makefile
# 完整 Makefile 见 docs/build-guide.md
.PHONY: all
all: tui gui

.PHONY: tui
tui:
	go build -o bin/claude-switch ./cmd/claude-switch

.PHONY: gui
gui:
	go build -o bin/claude-switch-gui ./cmd/claude-switch-gui

.PHONY: release-windows
release-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" \
		-o dist/windows/claude-switch-gui.exe ./cmd/claude-switch-gui

.PHONY: release-macos
release-macos:
	fyne package -os darwin -name "Claude Switch" \
		-icon assets/icon.png -appID com.claude-switch.gui \
		-sourceDir cmd/claude-switch-gui

.PHONY: release-linux
release-linux:
	go build -o dist/linux/claude-switch-gui ./cmd/claude-switch-gui
```

#### 2.4.3 验证清单

- [ ] Windows 可执行文件运行正常
- [ ] macOS App Bundle 运行正常
- [ ] Linux 可执行文件运行正常
- [ ] 图标显示正确
- [ ] 文件大小合理

---

## 3. 里程碑

| 里程碑 | 日期 | 产出物 | 验收标准 |
|--------|------|--------|----------|
| M0: 重构完成 | Day 1 | core.Manager + 重构后 TUI | TUI 功能完整 |
| M1: GUI 框架 | Day 3 | 可运行的 GUI，显示列表 | 能显示 Profile |
| M2: 功能完整 | Day 6 | 所有功能实现 | CRUD + Run 可用 |
| M3: 体验优化 | Day 8 | 优化版本 | 界面美观流畅 |
| M4: 发布就绪 | Day 10 | 三个平台可执行文件 | 全平台测试通过 |

---

## 4. 风险与应对

| 风险 | 概率 | 应对策略 |
|------|------|----------|
| Fyne 组件不够灵活 | 中 | 使用 canvas 自绘 |
| Windows Defender 误报 | 中 | 添加数字签名或说明 |
| macOS 公证失败 | 低 | 提供命令行绕过方法 |
| 二进制体积过大 | 低 | upx 压缩 |

---

## 5. 后续扩展

完成基础版本后可考虑：

1. **系统托盘** - 最小化到托盘，快速切换
2. **全局快捷键** - 呼出切换窗口
3. **自动更新** - 检查 GitHub Release
4. **导入/导出** - JSON 备份 Profile
