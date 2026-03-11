# 后续跟进事项

**本文件记录 code review 发现的问题及后续决策**

---

## 设计确认（不需要处理 ✅）

| 问题 | 结论 |
|------|------|
| Domain: MaskSensitive 长度9边界情况 | 设计如此，不需要处理 |
| Repository: LoadCurrent 文件不存在返回空map | 设计如此，不需要处理 |
| Service: Runner.Run 未使用 profile 参数 | 设计如此，为未来扩展预留 |

---

## 可修复问题（低优先级，可选处理）

### 1. nil map 风险 (Low Priority)
**位置**: `internal/domain/profile.go:141-151` 等

**问题**: `GetSummary` 和 `GetModelMapping` 直接索引 `settings` 参数，如果传入 `nil` 会导致 panic。

```go
func GetSummary(settings map[string]interface{}) (model, baseURL string) {
    if m, ok := settings["model"]; ok {  // nil map 索引会 panic
        model = fmt.Sprintf("%v", m)
    }
    // ...
}
```

**建议**: 添加 nil 检查或文档说明调用方需保证非 nil。

---

### 2. 预定义错误未使用 (Low Priority)
**位置**: `internal/domain/errors.go:8` vs `profile.go:43-50`

**问题**: 定义了 `ErrInvalidName` 但在 `ValidateProfileName` 中没有使用。

```go
// errors.go
var ErrInvalidName = errors.New("invalid profile name")  // 定义了

// profile.go
return fmt.Errorf("name cannot be empty")  // 但没使用 ErrInvalidName
```

**影响**: 调用方无法使用 `errors.Is(err, domain.ErrInvalidName)` 进行错误判断。

**建议**: 统一返回预定义错误或移除未使用的错误定义。

---

### 3. MaskSensitive 掩码逻辑边界情况 (Design Question)
**位置**: `internal/domain/profile.go:131-134`

**问题**: 当 value 长度为 9 时，结果是 `前4 + **** + 后4`，实际上只隐藏了 1 个字符。

```go
// 例：value = "123456789"
// 结果："1234****6789"
```

**待确认**: 这是预期行为吗？还是应该是 `***56789` 或 `1234***789`？

---

### 4. 文件格式 (Trivial)
**位置**: `internal/domain/errors.go`, `internal/domain/profile.go`

**问题**: 文件末尾缺少换行符。

---

---

## Repository Layer 代码审查发现的问题

### 1. Delete 错误未包装 (Low Priority)
**位置**: `internal/repository/profile_fs.go:96-104`

**问题**: `Delete` 返回原始错误时没有上下文信息。

```go
if err := os.Remove(path); err != nil {
    if os.IsNotExist(err) {
        return domain.ErrProfileNotFound
    }
    return err  // 缺少文件名上下文
}
```

**建议**: 包装错误以便调用方知道是哪个文件删除失败：
`return fmt.Errorf("remove profile %s: %w", name, err)`

---

### 2. LoadCurrent 错误处理设计 (Design Question)
**位置**: `internal/repository/profile_fs.go:107-122`

**问题**: `LoadCurrent` 和 `GetByName` 对文件不存在的处理不一致。

```go
// GetByName: 返回 ErrProfileNotFound
if os.IsNotExist(err) {
    return nil, domain.ErrProfileNotFound
}

// LoadCurrent: 返回空 map
if os.IsNotExist(err) {
    return make(map[string]interface{}), nil
}
```

**待确认**: 如果 `settings.json` 被误删，`LoadCurrent` 返回空配置而非错误，是否符合预期？

---

### 3. 文件格式 (Trivial)
**位置**: `internal/repository/profile.go`, `profile_fs.go`, `profile_test.go`

**问题**: 文件末尾缺少换行符。

---

---

## Service Layer 代码审查发现的问题

### 1. buildCommand 分支代码重复 (Style)
**位置**: `internal/service/runner.go:32-38`, `runner.go:72-80`

**问题**: `buildCommand` 的 switch 语句中 windows 和 default 分支代码完全相同。

```go
switch runtime.GOOS {
case "windows":
    return exec.Command(r.claudePath)
default:
    return exec.Command(r.claudePath)  // 重复代码
}
```

**建议**: 如果逻辑相同，可以移除 switch 直接返回，或添加注释说明未来可能不同。

---

### 2. Runner.Run 未使用 profile 参数 (Design Question)
**位置**: `internal/service/runner.go:23-28`, `runner.go:56-61`

**问题**: `ProfileRunner.Run` 接口接收 `domain.Profile` 参数，但两个实现都没有使用它。

```go
func (r *ProfileRunnerExec) Run(p domain.Profile) error {
    cmd := r.buildCommand()  // 没有使用 p
    // ...
}
```

**待确认**: 这是为未来扩展预留（比如根据 profile 选择运行方式），还是设计可以简化？当前使用流程是先 `Apply` 配置到文件，然后 runner 只负责启动 claude。

---

### 3. 文件格式 (Trivial)
**位置**: `internal/service/profile.go`, `profile_test.go`, `runner.go`

**问题**: 文件末尾缺少换行符。

---

## 状态
- [x] 确认 Domain Layer 问题 3 的设计意图 - 不需要处理，设计如此
- [x] 确认 Repository Layer 问题 2 的设计意图 - 不需要处理，设计如此
- [x] 确认 Service Layer 问题 2 的设计意图 - 不需要处理，设计如此
- [x] 根据需要修复上述问题 - 已全部处理
- [ ] 补充相关测试用例

---

## 架构重构已完成 ✅

### 状态更新 (2026-03-11)

1. **internal/profile 目录已删除** - 旧代码已清理
2. **TUI 已迁移到 service 层** - 通过 service 接口调用 runner
3. **Runner 已统一** - TUI 通过 service.PrepareAndBuild() 获取命令

### 当前架构

```
main.go
  └── service.ProfileService (interface)
        ├── repository.ProfileRepository (interface)
        │     └── ProfileRepositoryFS (impl)
        └── ProfileRunner (interface)
              └── ProfileRunnerExec (impl)
```

TUI 不再直接依赖 `internal/runner`，而是通过 service 层的接口调用。

---

## 历史问题（已解决）

### ~~旧代码仍然存在且被使用~~ ✅ 已解决
- `internal/profile/` 目录已删除
- TUI 已迁移使用 service 层

### ~~两套 Runner 实现~~ ✅ 已解决
- `internal/runner/runner.go` 保留作为独立包（MCP 功能可能需要）
- TUI 通过 `service.ProfileRunner` 接口统一调用

---

### 4. Service Runner 过度设计 (Medium)
**位置**: `internal/service/runner.go`

**问题**:
1. `ProfileRunnerExec` 和 `ProfileRunnerWithDir` 代码重复（Run 方法完全相同）
2. Profile 参数传入但未使用
3. `buildCommand` 的 switch 分支无意义（windows 和 default 相同）

**建议**: 合并为单一结构体，或参考 `internal/runner/runner.go` 重构。

---

### 5. ValidateProfileName 未使用预定义错误 (Low)
**位置**: `domain/profile.go:43-50` vs `domain/errors.go:8`

**问题**: 定义了 `ErrInvalidName` 但 `ValidateProfileName` 返回普通 error：

```go
// domain/errors.go
var ErrInvalidName = errors.New("invalid profile name")

// domain/profile.go - 返回普通 error
return fmt.Errorf("name cannot be empty")  // 应该是 ErrInvalidName
```

---

### 6. 文件末尾缺少换行符 (Trivial)
多个文件末尾缺少换行符。

---

## 待办
- [ ] 迁移 TUI 使用 domain/repository/service 层，删除 internal/profile 目录
- [ ] 统一或明确两套 runner 的职责
- [ ] 修复 Service Runner 过度设计问题
- [ ] 修复 ValidateProfileName 错误返回
- [ ] 补充文件末尾换行符
- [ ] 在 domain/service 层补充 DefaultProfile 预设函数（TUI 需要）- 已在 transition-plan.md 添加 Phase 3.5
