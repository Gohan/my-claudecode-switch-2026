# GUI 开发指南

本文档详细说明 Fyne GUI 版本的开发规范、组件使用和最佳实践。

## 1. Fyne 基础

### 1.1 核心概念

| 概念 | 说明 | 类比 |
|------|------|------|
| App | 应用程序实例 | 整个程序 |
| Window | 窗口 | 浏览器窗口 |
| CanvasObject | 可绘制对象 | DOM 元素 |
| Container | 容器，布局管理 | div |
| Widget | 交互组件 | button, input |

### 1.2 生命周期

```
app.New() → NewWindow() → SetContent() → ShowAndRun()
                 ↑
                 └── CanvasObject (Container/Widget)
```

### 1.3 基本示例

```go
package main

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
)

func main() {
    a := app.New()
    w := a.NewWindow("Hello")
    w.Resize(fyne.NewSize(400, 300))

    // 创建组件
    label := widget.NewLabel("Hello Fyne!")
    button := widget.NewButton("Click", func() {
        label.SetText("Clicked!")
    })

    // 布局
    content := container.NewVBox(label, button)
    w.SetContent(content)

    w.ShowAndRun()
}
```

## 2. 项目结构规范

### 2.1 GUI 目录结构

```
cmd/claude-switch-gui/
├── main.go              # 入口
├── app.go               # App 初始化
├── window.go            # 主窗口
├── state.go             # 状态绑定
└── components/          # 组件目录
    ├── list.go          # Profile 列表
    ├── toolbar.go       # 工具栏
    ├── form.go          # 表单
    ├── dialogs.go       # 对话框
    └── statusbar.go     # 状态栏
```

### 2.2 文件职责

| 文件 | 职责 | 禁止 |
|------|------|------|
| main.go | 初始化 App，启动窗口 | 包含业务逻辑 |
| app.go | App 配置，全局设置 | 操作 UI 组件 |
| window.go | 窗口布局，组件组装 | 直接调用 core 方法 |
| state.go | 状态绑定，事件转发 | 包含布局代码 |
| components/*.go | 具体组件实现 | 依赖其他组件 |

## 3. 组件开发规范

### 3.1 组件模板

```go
package components

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/widget"
    "claude-switch/internal/core"
)

// ComponentName 组件描述
type ComponentName struct {
    widget.BaseWidget  // 嵌入基础组件

    // 依赖
    manager *core.Manager
    window  fyne.Window

    // 内部组件
    label *widget.Label
    button *widget.Button

    // 状态
    value string
}

// NewComponentName 构造函数
func NewComponentName(m *core.Manager, w fyne.Window) *ComponentName {
    c := &ComponentName{
        manager: m,
        window:  w,
        label:   widget.NewLabel("Default"),
        button:  widget.NewButton("Action", nil),
    }

    // 设置回调
    c.button.OnTapped = c.onAction

    // 必须调用，初始化基础组件
    c.ExtendBaseWidget(c)

    return c
}

// CreateRenderer 实现 fyne.Widget 接口
func (c *ComponentName) CreateRenderer() fyne.WidgetRenderer {
    // 返回渲染内容
    return widget.NewSimpleRenderer(c.label)
}

// 私有方法
func (c *ComponentName) onAction() {
    // 处理点击
}

// 公共方法
func (c *ComponentName) SetValue(v string) {
    c.value = v
    c.label.SetText(v)
    c.Refresh()  // 触发重绘
}
```

### 3.2 ProfileList 完整实现

```go
package components

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/canvas"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/layout"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
    "claude-switch/internal/core"
    "claude-switch/internal/profile"
)

// ProfileList Profile 列表组件
type ProfileList struct {
    widget.List
    manager *core.Manager
    selected int
}

// NewProfileList 创建列表
func NewProfileList(m *core.Manager) *ProfileList {
    pl := &ProfileList{
        manager: m,
        selected: -1,
    }

    pl.List = widget.NewList(
        // 长度函数
        func() int {
            return len(m.Profiles())
        },
        // 创建项
        func() fyne.CanvasObject {
            return newProfileListItem()
        },
        // 更新项
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

    pl.OnUnselected = func(id widget.ListItemID) {
        pl.selected = -1
    }

    pl.ExtendBaseWidget(pl)
    return pl
}

// SelectedIndex 获取选中索引
func (pl *ProfileList) SelectedIndex() int {
    return pl.selected
}

// RefreshData 刷新数据
func (pl *ProfileList) RefreshData() {
    pl.Refresh()
}

// profileListItem 列表项内部组件
type profileListItem struct {
    container *fyne.Container
    icon      *widget.Icon
    name      *widget.Label
    details   *widget.Label
    badge     *canvas.Text
}

func newProfileListItem() fyne.CanvasObject {
    item := &profileListItem{
        icon:    widget.NewIcon(theme.DocumentIcon()),
        name:    widget.NewLabel("Profile Name"),
        details: widget.NewLabel("details"),
        badge:   canvas.NewText("active", theme.PrimaryColor()),
    }

    item.name.TextStyle = fyne.TextStyle{Bold: true}
    item.details.TextSize = theme.CaptionTextSize()
    item.badge.TextSize = theme.CaptionTextSize()
    item.badge.Hide()  // 默认隐藏

    textContainer := container.NewVBox(item.name, item.details)

    item.container = container.NewHBox(
        item.icon,
        textContainer,
        layout.NewSpacer(),
        item.badge,
    )

    return item.container
}

func (item *profileListItem) Update(p profile.Profile, isActive bool) {
    item.name.SetText(p.Name)

    model, url := profile.GetSummary(p.Settings)
    item.details.SetText(fmt.Sprintf("model: %s", model))
    if url != "" {
        item.details.SetText(fmt.Sprintf("model: %s | %s", model, url))
    }

    if isActive {
        item.badge.Show()
    } else {
        item.badge.Hide()
    }
}
```

### 3.3 Toolbar 实现

```go
package components

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
    "claude-switch/internal/core"
)

// Toolbar 工具栏组件
type Toolbar struct {
    container *fyne.Container
    manager   *core.Manager
    window    fyne.Window
    list      *ProfileList

    btnNew    *widget.Button
    btnRun    *widget.Button
    btnApply  *widget.Button
    btnDelete *widget.Button
}

// NewToolbar 创建工具栏
func NewToolbar(m *core.Manager, w fyne.Window, list *ProfileList) *Toolbar {
    t := &Toolbar{
        manager: m,
        window:  w,
        list:    list,
    }

    // 创建按钮
    t.btnNew = widget.NewButtonWithIcon("New", theme.ContentAddIcon(), t.onNew)
    t.btnRun = widget.NewButtonWithIcon("Run", theme.MediaPlayIcon(), t.onRun)
    t.btnApply = widget.NewButtonWithIcon("Apply", theme.DocumentSaveIcon(), t.onApply)
    t.btnDelete = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), t.onDelete)

    // 初始禁用操作按钮
    t.updateButtons(-1)

    // 监听列表选择变化
    list.OnSelected = func(id widget.ListItemID) {
        t.updateButtons(int(id))
    }
    list.OnUnselected = func(id widget.ListItemID) {
        t.updateButtons(-1)
    }

    // 布局
    t.container = container.NewHBox(
        t.btnNew,
        widget.NewSeparator(),
        t.btnRun,
        t.btnApply,
        widget.NewSeparator(),
        t.btnDelete,
    )

    return t
}

// Container 返回容器对象
func (t *Toolbar) Container() fyne.CanvasObject {
    return t.container
}

// updateButtons 根据选择更新按钮状态
func (t *Toolbar) updateButtons(selectedIdx int) {
    hasSelection := selectedIdx >= 0
    t.btnRun.Disabled = !hasSelection
    t.btnApply.Disabled = !hasSelection
    t.btnDelete.Disabled = !hasSelection
    t.container.Refresh()
}

// onNew 新建 Profile
func (t *Toolbar) onNew() {
    ShowCreateProfileDialog(t.window, t.manager, func() {
        t.list.Refresh()
    })
}

// onRun 运行 Profile
func (t *Toolbar) onRun() {
    idx := t.list.SelectedIndex()
    if idx < 0 {
        return
    }

    profiles := t.manager.Profiles()
    if idx >= len(profiles) {
        return
    }

    p := profiles[idx]

    confirm := dialog.NewConfirm(
        "Run Profile",
        fmt.Sprintf("Run claude with profile '%s'?\n\nThis will start a new claude process.", p.Name),
        func(ok bool) {
            if !ok {
                return
            }
            // 异步运行，避免阻塞 UI
            go func() {
                if err := t.manager.Run(idx); err != nil {
                    dialog.ShowError(err, t.window)
                }
            }()
        },
        t.window,
    )
    confirm.Show()
}

// onApply 应用 Profile
func (t *Toolbar) onApply() {
    idx := t.list.SelectedIndex()
    if idx < 0 {
        return
    }

    if err := t.manager.Apply(idx); err != nil {
        dialog.ShowError(err, t.window)
        return
    }

    t.list.Refresh()
    dialog.ShowInformation("Success", "Profile applied successfully", t.window)
}

// onDelete 删除 Profile
func (t *Toolbar) onDelete() {
    idx := t.list.SelectedIndex()
    if idx < 0 {
        return
    }

    profiles := t.manager.Profiles()
    if idx >= len(profiles) {
        return
    }

    p := profiles[idx]

    confirm := dialog.NewConfirm(
        "Delete Profile",
        fmt.Sprintf("Delete profile '%s'?\n\nThis cannot be undone.", p.Name),
        func(ok bool) {
            if !ok {
                return
            }
            if err := t.manager.Delete(p.Name); err != nil {
                dialog.ShowError(err, t.window)
                return
            }
            t.list.UnselectAll()
            t.list.Refresh()
        },
        t.window,
    )
    confirm.Show()
}
```

## 4. 对话框开发

### 4.1 创建 Profile 对话框

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

// ShowCreateProfileDialog 显示创建对话框
func ShowCreateProfileDialog(w fyne.Window, m *core.Manager, onSuccess func()) {
    // 类型选择
    typeSelect := widget.NewSelect([]string{
        "Custom",
        "z.ai API",
        "Tencent Cloud",
        "Kimi",
        "Ali BaiLian",
    }, nil)
    typeSelect.SetSelected("z.ai API")

    // 名称输入
    nameEntry := widget.NewEntry()
    nameEntry.SetPlaceHolder("my-profile")

    // API Key
    apiKeyEntry := widget.NewPasswordEntry()
    apiKeyEntry.SetPlaceHolder("your-api-key")

    // 根据类型设置默认名称
    typeSelect.OnChanged = func(value string) {
        switch value {
        case "z.ai API":
            nameEntry.SetText("z.ai Coding Plan")
        case "Tencent Cloud":
            nameEntry.SetText("Tencent Coding Plan")
        case "Kimi":
            nameEntry.SetText("Kimi Coding")
        case "Ali BaiLian":
            nameEntry.SetText("Ali Coding Plan")
        default:
            nameEntry.SetText("")
        }
    }

    // 触发一次默认值
    typeSelect.OnChanged(typeSelect.Selected)

    items := []*widget.FormItem{
        widget.NewFormItem("Type", typeSelect),
        widget.NewFormItem("Name", nameEntry),
        widget.NewFormItem("API Key", apiKeyEntry),
    }

    var d dialog.Dialog
    d = dialog.NewForm(
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

            // 获取默认配置
            var settings map[string]interface{}
            switch typeSelect.Selected {
            case "z.ai API":
                settings = profile.DefaultZAIProfile()
            case "Tencent Cloud":
                settings = profile.DefaultTencentCloudProfile()
            case "Kimi":
                settings = profile.DefaultKimiProfile()
            case "Ali BaiLian":
                settings = profile.DefaultAliProfile()
            default:
                settings = make(map[string]interface{})
            }

            // 设置 API Key
            if apiKeyEntry.Text != "" {
                if env, ok := settings["env"].(map[string]interface{}); ok {
                    env["ANTHROPIC_AUTH_TOKEN"] = apiKeyEntry.Text
                }
            }

            if err := m.Create(name, settings); err != nil {
                dialog.ShowError(err, w)
                return
            }

            onSuccess()
        },
        w,
    )

    d.Resize(fyne.NewSize(400, 0))
    d.Show()
}
```

## 5. 状态管理

### 5.1 状态绑定模式

```go
package gui

import (
    "fyne.io/fyne/v2"
    "claude-switch/internal/core"
)

// StateManager 管理 UI 状态与业务状态的绑定
type StateManager struct {
    manager *core.Manager
    window  fyne.Window

    // 组件引用
    list    *ProfileList
    toolbar *Toolbar
    status  *StatusBar
}

// NewStateManager 创建状态管理器
func NewStateManager(m *core.Manager, w fyne.Window) *StateManager {
    sm := &StateManager{
        manager: m,
        window:  w,
    }

    // 订阅业务事件
    m.Subscribe(sm.onEvent)

    return sm
}

// SetComponents 设置组件引用
func (sm *StateManager) SetComponents(list *ProfileList, toolbar *Toolbar, status *StatusBar) {
    sm.list = list
    sm.toolbar = toolbar
    sm.status = status
}

// onEvent 处理业务事件
func (sm *StateManager) onEvent(e core.Event) {
    switch e.Type {
    case core.EventProfilesLoaded:
        sm.list.Refresh()
        sm.status.SetMessage(fmt.Sprintf("Loaded %d profiles", len(sm.manager.Profiles())))

    case core.EventProfileCreated:
        sm.list.Refresh()
        sm.status.SetSuccess(fmt.Sprintf("Created profile: %s", e.Payload))

    case core.EventProfileDeleted:
        sm.list.Refresh()
        sm.status.SetMessage(fmt.Sprintf("Deleted profile: %s", e.Payload))

    case core.EventProfileApplied:
        sm.list.Refresh()
        sm.status.SetSuccess(fmt.Sprintf("Applied profile: %s", e.Payload))

    case core.EventError:
        sm.status.SetError(e.Error.Error())
        dialog.ShowError(e.Error, sm.window)
    }
}
```

## 6. 最佳实践

### 6.1 性能优化

```go
// ✅ 批量更新后再 Refresh
list.Data = newData
list.Refresh()  // 只刷新一次

// ❌ 频繁刷新
for _, item := range items {
    list.Add(item)
    list.Refresh()  // 每次都要重绘
}

// ✅ 异步操作
button.OnTapped = func() {
    go func() {
        result := doHeavyWork()
        fyne.CurrentApp().Driver().CallOnMainThread(func() {
            label.SetText(result)
        })
    }()
}
```

### 6.2 错误处理

```go
// ✅ 用户友好的错误提示
if err := manager.Load(); err != nil {
    dialog.ShowError(
        fmt.Errorf("Failed to load profiles:\n%v", err),
        window,
    )
    return
}

// ✅ 区分错误类型
if os.IsNotExist(err) {
    // 第一次运行，没有配置
    showWelcomeDialog()
} else {
    dialog.ShowError(err, window)
}
```

### 6.3 资源管理

```go
// ✅ 使用 fyne.URI 处理文件
uri, _ := storage.ParseURI("file://" + path)
reader, _ := storage.Reader(uri)
defer reader.Close()

// ✅ 图片资源缓存
var iconResource fyne.Resource

func getIcon() fyne.Resource {
    if iconResource == nil {
        iconResource = fyne.NewStaticResource("icon", iconData)
    }
    return iconResource
}
```

## 7. 调试技巧

### 7.1 日志输出

```go
import "log"

// 在关键位置添加日志
func (c *Component) DoSomething() {
    log.Printf("[Component] DoSomething called")
    // ...
}
```

### 7.2 性能分析

```go
import "runtime/pprof"

// CPU 分析
f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()
```

### 7.3 界面调试

```go
// 显示边界（开发模式）
// fyne 目前没有内置边界显示，可用不同背景色区分
container := container.NewVBox(
    canvas.NewRectangle(color.NRGBA{R: 255, A: 50}),  // 红色背景
    widget.NewLabel("Debug"),
)
```

## 8. 常见问题

### Q: 如何设置窗口最小尺寸？

```go
w.Resize(fyne.NewSize(600, 400))
w.SetFixedSize(false)  // 允许调整大小
// fyne 目前没有 SetMinSize，需要手动限制
```

### Q: 如何响应窗口关闭事件？

```go
w.SetOnClosed(func() {
    // 清理资源
    manager.Save()
})
```

### Q: 如何实现系统托盘？

```go
// 使用 fyne.io/systray
import "fyne.io/systray"

func onReady() {
    systray.SetIcon(iconData)
    systray.SetTitle("Claude Switch")
    mShow := systray.AddMenuItem("Show", "Show window")
    mQuit := systray.AddMenuItem("Quit", "Quit app")

    go func() {
        for {
            select {
            case <-mShow.ClickedCh:
                window.Show()
            case <-mQuit.ClickedCh:
                systray.Quit()
            }
        }
    }()
}
```
