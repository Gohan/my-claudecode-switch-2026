# 项目过渡方案

本文档描述从当前架构向可测试、分层架构的过渡计划，采用芝加哥派 TDD（测试先行）。

## 目录

1. [目标架构](#目标架构)
2. [实施总览](#实施总览)
3. [Phase 1: Domain 模型](#phase-1-domain-模型)
4. [Phase 2: Repository 层](#phase-2-repository-层)
5. [Phase 3: Service 层](#phase-3-service-层)
6. [Phase 4: TUI 重构](#phase-4-tui-重构)
7. [Phase 5: BubbleTea v2 迁移](#phase-5-bubbletea-v2-迁移)

---

## 目标架构

### 分层架构

```
┌─────────────────────────────────────────┐
│              UI Layer (TUI)              │
│         internal/tui/model.go            │
│  - 只负责 UI 状态管理                     │
│  - 调用 Service 接口                      │
├─────────────────────────────────────────┤
│            Service Layer                 │
│       internal/service/profile.go        │
│  - 业务逻辑编排                           │
│  - 定义接口                               │
├─────────────────────────────────────────┤
│           Repository Layer               │
│    internal/repository/profile_fs.go     │
│  - 数据持久化                             │
│  - 文件系统操作                           │
├─────────────────────────────────────────┤
│           Domain Layer                   │
│      internal/domain/profile.go          │
│  - 纯数据结构                             │
│  - 业务规则                               │
└─────────────────────────────────────────┘
```

### 依赖关系

```
tui.Model ──► ProfileService (interface)
                  │
                  ├──► ProfileRepository (interface)
                  │         └── ProfileRepositoryFS (impl)
                  │
                  └──► ProfileRunner (interface)
                            └── ProfileRunnerExec (impl)
```

---

## 实施总览

| Phase | 任务 | 工时 | 产出 |
|-------|------|------|------|
| 1 | Domain 模型 + 业务规则 | 1天 | `internal/domain/` |
| 2 | Repository 接口 + 实现 + 测试 | 2天 | `internal/repository/` |
| 3 | Service 接口 + 实现 + 测试 | 2天 | `internal/service/` |
| 4 | TUI 重构（使用新架构） | 2天 | `internal/tui/` 重构 |
| 5 | BubbleTea v2 迁移 | 1天 | 更新 import，适配 API |

**总计：8天**

---

## Phase 1: Domain 模型

### 目标

提取纯数据结构和业务规则，无外部依赖。

### TDD 循环

#### Step 1: 测试 - Profile 结构创建

创建 `internal/domain/profile_test.go`：

```go
package domain

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestProfile_CanBeCreatedWithNameAndSettings(t *testing.T) {
    p := Profile{
        Name: "my-profile",
        Settings: map[string]interface{}{
            "model": "opus",
        },
    }

    assert.Equal(t, "my-profile", p.Name)
    assert.Equal(t, "opus", p.Settings["model"])
}

func TestProfile_IsEmptyWhenNameIsBlank(t *testing.T) {
    tests := []struct {
        name     string
        expected bool
    }{
        {"", true},
        {"   ", true},
        {"profile", false},
        {"  profile  ", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := Profile{Name: tt.name}
            assert.Equal(t, tt.expected, p.IsEmpty())
        })
    }
}
```

#### Step 2: 实现

创建 `internal/domain/profile.go`：

```go
package domain

import "strings"

// Profile 代表一个 Claude Code 配置
type Profile struct {
    Name     string
    Settings map[string]interface{}
}

// IsEmpty 检查 Profile 是否为空（名称为空）
func (p Profile) IsEmpty() bool {
    return strings.TrimSpace(p.Name) == ""
}
```

#### Step 3: 测试 - 名称验证规则

```go
func TestProfileName_Validation(t *testing.T) {
    tests := []struct {
        name    string
        wantErr bool
        errMsg  string
    }{
        {"valid-name", false, ""},
        {"valid_name", false, ""},
        {"ValidName123", false, ""},
        {"", true, "name cannot be empty"},
        {"   ", true, "name cannot be empty"},
        {"has/slash", true, "invalid character: /"},
        {"has\\backslash", true, "invalid character: \\\\"},
        {"has*star", true, "invalid character: *"},
        {"has?question", true, "invalid character: ?"},
        {"has:colon", true, "invalid character: :"},
        {'has"quote', true, "invalid character: \""},
        {"has<less>", true, "invalid character: <"},
        {"has|pipe", true, "invalid character: |"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateProfileName(tt.name)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### Step 4: 实现

```go
package domain

import (
    "fmt"
    "regexp"
    "strings"
)

var invalidNameChars = regexp.MustCompile(`[\\/:*?"<>|]`)

// ValidateProfileName 验证 Profile 名称
func ValidateProfileName(name string) error {
    if strings.TrimSpace(name) == "" {
        return fmt.Errorf("name cannot be empty")
    }
    if match := invalidNameChars.FindString(name); match != "" {
        return fmt.Errorf("invalid character: %s", match)
    }
    return nil
}
```

#### Step 5: 测试 - DiffEntry 结构

```go
func TestDiffEntry_RepresentsChanges(t *testing.T) {
    entry := DiffEntry{
        Key:      "model",
        OldValue: "opus",
        NewValue: "sonnet",
        Status:   DiffModified,
    }

    assert.Equal(t, "model", entry.Key)
    assert.Equal(t, DiffModified, entry.Status)
}
```

#### Step 6: 实现

```go
package domain

// DiffStatus 表示 diff 状态
type DiffStatus int

const (
    DiffUnchanged DiffStatus = iota
    DiffModified
    DiffAdded
    DiffRemoved
)

// DiffEntry 表示单个配置的变更
type DiffEntry struct {
    Key      string
    OldValue string
    NewValue string
    Status   DiffStatus
}
```

### Phase 1 验证清单

- [ ] `go test ./internal/domain/...` 通过
- [ ] 测试覆盖率 > 90%
- [ ] 无外部依赖（标准库 only）

---

## Phase 2: Repository 层

### 目标

实现数据持久化接口，使用真实文件系统（芝加哥派 TDD）。

### TDD 循环

#### Step 1: 测试 - 接口契约

创建 `internal/repository/profile_test.go`：

```go
package repository

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "claude-switch/internal/domain"
)

// ProfileRepositoryTestSuite 测试 Repository 行为
func ProfileRepositoryTestSuite(t *testing.T, factory func() ProfileRepository) {
    t.Run("SaveAndGet", func(t *testing.T) {
        repo := factory()
        settings := map[string]interface{}{"model": "opus"}

        err := repo.Save("test", settings)
        require.NoError(t, err)

        p, err := repo.GetByName("test")
        require.NoError(t, err)
        assert.Equal(t, "test", p.Name)
        assert.Equal(t, "opus", p.Settings["model"])
    })

    t.Run("GetNonExistent", func(t *testing.T) {
        repo := factory()

        _, err := repo.GetByName("nonexistent")
        assert.ErrorIs(t, err, domain.ErrProfileNotFound)
    })

    t.Run("ListEmpty", func(t *testing.T) {
        repo := factory()

        profiles, errs := repo.List()
        assert.Empty(t, profiles)
        assert.Empty(t, errs)
    })

    t.Run("ListMultiple", func(t *testing.T) {
        repo := factory()
        repo.Save("p1", map[string]interface{}{"model": "opus"})
        repo.Save("p2", map[string]interface{}{"model": "sonnet"})

        profiles, errs := repo.List()
        assert.Len(t, profiles, 2)
        assert.Empty(t, errs)
    })

    t.Run("ListWithErrors", func(t *testing.T) {
        // 这个测试 FS 实现特有，内存实现可以 skip
    })

    t.Run("Delete", func(t *testing.T) {
        repo := factory()
        repo.Save("todelete", map[string]interface{}{})

        err := repo.Delete("todelete")
        require.NoError(t, err)

        _, err = repo.GetByName("todelete")
        assert.ErrorIs(t, err, domain.ErrProfileNotFound)
    })

    t.Run("DeleteNonExistent", func(t *testing.T) {
        repo := factory()

        err := repo.Delete("nonexistent")
        assert.Error(t, err)
    })
}
```

#### Step 2: 实现接口

创建 `internal/repository/profile.go`：

```go
package repository

import "claude-switch/internal/domain"

// ProfileRepository 定义数据访问接口
type ProfileRepository interface {
    List() ([]domain.Profile, []ListError)
    GetByName(name string) (*domain.Profile, error)
    Save(name string, settings map[string]interface{}) error
    Delete(name string) error
    LoadCurrent() (map[string]interface{}, error)
    Apply(p domain.Profile) error
}

type ListError struct {
    Name string
    Err  error
}
```

#### Step 3: 测试 - FS 实现

```go
func TestProfileRepositoryFS(t *testing.T) {
    factory := func() ProfileRepository {
        tmpDir := t.TempDir()
        return NewProfileRepositoryFS(tmpDir, filepath.Join(tmpDir, "settings.json"))
    }

    ProfileRepositoryTestSuite(t, factory)

    t.Run("PersistAcrossInstances", func(t *testing.T) {
        tmpDir := t.TempDir()
        settingsPath := filepath.Join(tmpDir, "settings.json")

        repo1 := NewProfileRepositoryFS(tmpDir, settingsPath)
        repo1.Save("persisted", map[string]interface{}{"model": "opus"})

        repo2 := NewProfileRepositoryFS(tmpDir, settingsPath)
        p, err := repo2.GetByName("persisted")

        require.NoError(t, err)
        assert.Equal(t, "opus", p.Settings["model"])
    })

    t.Run("ListSkipsInvalidFiles", func(t *testing.T) {
        tmpDir := t.TempDir()
        repo := NewProfileRepositoryFS(tmpDir, filepath.Join(tmpDir, "settings.json"))

        // 创建有效文件
        repo.Save("valid", map[string]interface{}{})

        // 创建无效文件
        os.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("not json"), 0644)

        profiles, errs := repo.List()
        assert.Len(t, profiles, 1)
        assert.Len(t, errs, 1)
        assert.Equal(t, "invalid", errs[0].Name)
    })
}
```

#### Step 4: 实现 FS Repository

创建 `internal/repository/profile_fs.go`：

```go
package repository

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "claude-switch/internal/domain"
)

type ProfileRepositoryFS struct {
    profilesDir  string
    settingsPath string
}

func NewProfileRepositoryFS(profilesDir, settingsPath string) *ProfileRepositoryFS {
    return &ProfileRepositoryFS{
        profilesDir:  profilesDir,
        settingsPath: settingsPath,
    }
}

func (r *ProfileRepositoryFS) Save(name string, settings map[string]interface{}) error {
    if err := domain.ValidateProfileName(name); err != nil {
        return err
    }

    if err := os.MkdirAll(r.profilesDir, 0755); err != nil {
        return fmt.Errorf("create directory: %w", err)
    }

    path := filepath.Join(r.profilesDir, name+".json")
    data, err := json.MarshalIndent(settings, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal settings: %w", err)
    }

    return os.WriteFile(path, append(data, '\n'), 0644)
}

func (r *ProfileRepositoryFS) GetByName(name string) (*domain.Profile, error) {
    path := filepath.Join(r.profilesDir, name+".json")
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, domain.ErrProfileNotFound
        }
        return nil, err
    }

    var settings map[string]interface{}
    if err := json.Unmarshal(data, &settings); err != nil {
        return nil, fmt.Errorf("unmarshal settings: %w", err)
    }

    return &domain.Profile{Name: name, Settings: settings}, nil
}

func (r *ProfileRepositoryFS) List() ([]domain.Profile, []ListError) {
    entries, err := os.ReadDir(r.profilesDir)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil
        }
        return nil, []ListError{{Name: "", Err: err}}
    }

    var profiles []domain.Profile
    var errs []ListError

    for _, e := range entries {
        if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
            continue
        }

        name := strings.TrimSuffix(e.Name(), ".json")
        p, err := r.GetByName(name)
        if err != nil {
            errs = append(errs, ListError{Name: name, Err: err})
            continue
        }
        profiles = append(profiles, *p)
    }

    return profiles, errs
}

func (r *ProfileRepositoryFS) Delete(name string) error {
    path := filepath.Join(r.profilesDir, name+".json")
    if err := os.Remove(path); err != nil {
        if os.IsNotExist(err) {
            return domain.ErrProfileNotFound
        }
        return err
    }
    return nil
}

func (r *ProfileRepositoryFS) LoadCurrent() (map[string]interface{}, error) {
    data, err := os.ReadFile(r.settingsPath)
    if err != nil {
        if os.IsNotExist(err) {
            return make(map[string]interface{}), nil
        }
        return nil, err
    }

    var settings map[string]interface{}
    if err := json.Unmarshal(data, &settings); err != nil {
        return nil, err
    }
    return settings, nil
}

func (r *ProfileRepositoryFS) Apply(p domain.Profile) error {
    data, err := json.MarshalIndent(p.Settings, "", "  ")
    if err != nil {
        return err
    }

    // 原子写入：先写临时文件，再重命名
    tmpPath := r.settingsPath + ".tmp"
    if err := os.WriteFile(tmpPath, append(data, '\n'), 0644); err != nil {
        return fmt.Errorf("write temp file: %w", err)
    }

    // os.Rename 在同一文件系统上是原子操作
    if err := os.Rename(tmpPath, r.settingsPath); err != nil {
        os.Remove(tmpPath) // 清理临时文件
        return fmt.Errorf("rename to settings: %w", err)
    }

    return nil
}
```

#### Step 5: 错误定义

创建 `internal/domain/errors.go`：

```go
package domain

import "errors"

var (
    ErrProfileNotFound = errors.New("profile not found")
    ErrProfileExists   = errors.New("profile already exists")
    ErrInvalidName     = errors.New("invalid profile name")
)
```

### 应用程序初始化

重构不改变现有文件存储结构。Repository 初始化时使用现有路径：

```go
// internal/app/dependencies.go

import (
    "claude-switch/internal/repository"
    "claude-switch/internal/service"
)

func NewApp() (*App, error) {
    // 使用现有路径，不创建新的目录结构
    profilesDir := filepath.Join(os.Getenv("HOME"), ".claude-switch", "profiles")
    settingsPath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")

    repo := repository.NewProfileRepositoryFS(profilesDir, settingsPath)
    runner := service.NewProfileRunnerExec("claude")
    svc := service.NewProfileService(repo, runner)

    return &App{service: svc}, nil
}
```

**文件结构保持不变：**
- Profile 存储：`~/.claude-switch/profiles/<name>.json`
- Claude 配置：`~/.claude/settings.json`

### Phase 2 验证清单

- [ ] `go test ./internal/repository/...` 通过
- [ ] 测试覆盖率 > 85%
- [ ] 使用 `t.TempDir()`，测试隔离

---

## Phase 3: Service 层

### 目标

实现业务逻辑，组合 Repository 和 Runner。

### TDD 循环

#### Step 1: 测试 - 内存 Repository 实现

创建 `internal/service/profile_test.go`：

```go
package service

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "claude-switch/internal/domain"
)

// memoryRepo 内存实现，用于单元测试
type memoryRepo struct {
    profiles map[string]domain.Profile
    current  map[string]interface{}
}

func newMemoryRepo() *memoryRepo {
    return &memoryRepo{
        profiles: make(map[string]domain.Profile),
        current:  make(map[string]interface{}),
    }
}

func (m *memoryRepo) Save(name string, settings map[string]interface{}) error {
    if err := domain.ValidateProfileName(name); err != nil {
        return err
    }
    m.profiles[name] = domain.Profile{Name: name, Settings: settings}
    return nil
}

func (m *memoryRepo) GetByName(name string) (*domain.Profile, error) {
    p, ok := m.profiles[name]
    if !ok {
        return nil, domain.ErrProfileNotFound
    }
    return &p, nil
}

func (m *memoryRepo) List() ([]domain.Profile, []ListError) {
    var result []domain.Profile
    for _, p := range m.profiles {
        result = append(result, p)
    }
    return result, nil
}

func (m *memoryRepo) Delete(name string) error {
    if _, ok := m.profiles[name]; !ok {
        return domain.ErrProfileNotFound
    }
    delete(m.profiles, name)
    return nil
}

func (m *memoryRepo) LoadCurrent() (map[string]interface{}, error) {
    return m.current, nil
}

func (m *memoryRepo) Apply(p domain.Profile) error {
    m.current = p.Settings
    return nil
}

// memoryRunner 用于测试
type memoryRunner struct {
    runs []string
}

func (m *memoryRunner) Run(p domain.Profile) error {
    m.runs = append(m.runs, p.Name)
    return nil
}
```

#### Step 2: 测试 - Service 创建 Profile

```go
func TestProfileService_CreatesProfileWithValidData(t *testing.T) {
    repo := newMemoryRepo()
    svc := NewProfileService(repo, nil)

    err := svc.Create("my-profile", map[string]interface{}{
        "model": "opus",
    })

    require.NoError(t, err)

    profiles, _ := svc.List()
    assert.Len(t, profiles, 1)
    assert.Equal(t, "my-profile", profiles[0].Name)
}

func TestProfileService_RejectsDuplicateName(t *testing.T) {
    repo := newMemoryRepo()
    repo.Save("existing", map[string]interface{}{})

    svc := NewProfileService(repo, nil)
    err := svc.Create("existing", map[string]interface{}{})

    assert.ErrorIs(t, err, domain.ErrProfileExists)
}

func TestProfileService_ValidatesName(t *testing.T) {
    tests := []struct {
        name string
    }{
        {""},
        {"has/slash"},
        {"has*star"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := NewProfileService(newMemoryRepo(), nil)
            err := svc.Create(tt.name, map[string]interface{}{})
            assert.ErrorIs(t, err, domain.ErrInvalidName)
        })
    }
}
```

#### Step 3: 实现

创建 `internal/service/profile.go`：

```go
package service

import (
    "claude-switch/internal/domain"
    "claude-switch/internal/repository"
)

// ProfileService 业务逻辑接口
type ProfileService interface {
    List() ([]domain.Profile, error)
    GetByName(name string) (*domain.Profile, error)
    Create(name string, settings map[string]interface{}) error
    Update(name string, settings map[string]interface{}) error
    Delete(name string) error
    Apply(name string) error
    Run(name string) error
    LoadCurrent() (map[string]interface{}, error)
    IsActive(p domain.Profile) bool
}

type profileRunner interface {
    Run(p domain.Profile) error
}

type profileService struct {
    repo   repository.ProfileRepository
    runner profileRunner
}

func NewProfileService(
    repo repository.ProfileRepository,
    runner profileRunner,
) ProfileService {
    return &profileService{
        repo:   repo,
        runner: runner,
    }
}

func (s *profileService) Create(name string, settings map[string]interface{}) error {
    if _, err := s.repo.GetByName(name); err == nil {
        return domain.ErrProfileExists
    }
    return s.repo.Save(name, settings)
}

func (s *profileService) List() ([]domain.Profile, error) {
    profiles, errs := s.repo.List()
    if len(errs) > 0 {
        // 返回有效数据 + 警告错误，调用方可选择处理或忽略
        return profiles, &ListWarningError{Errors: errs}
    }
    return profiles, nil
}

// ListWarningError 表示部分 profile 加载失败的警告
type ListWarningError struct {
    Errors []repository.ListError
}

func (e *ListWarningError) Error() string {
    return fmt.Sprintf("%d profiles failed to load", len(e.Errors))
}

func (e *ListWarningError) Is(target error) bool {
    _, ok := target.(*ListWarningError)
    return ok
}

func (s *profileService) GetByName(name string) (*domain.Profile, error) {
    return s.repo.GetByName(name)
}

func (s *profileService) Update(name string, settings map[string]interface{}) error {
    if _, err := s.repo.GetByName(name); err != nil {
        return err
    }
    return s.repo.Save(name, settings)
}

func (s *profileService) Delete(name string) error {
    return s.repo.Delete(name)
}

func (s *profileService) Apply(name string) error {
    p, err := s.repo.GetByName(name)
    if err != nil {
        return err
    }
    return s.repo.Apply(*p)
}

func (s *profileService) Run(name string) error {
    p, err := s.repo.GetByName(name)
    if err != nil {
        return err
    }
    return s.runner.Run(*p)
}

func (s *profileService) LoadCurrent() (map[string]interface{}, error) {
    return s.repo.LoadCurrent()
}

func (s *profileService) IsActive(p domain.Profile) bool {
    current, err := s.repo.LoadCurrent()
    if err != nil || len(current) == 0 {
        return false
    }

    // 深度比较，规范化处理 JSON 类型差异
    normalizedCurrent := normalizeSettings(current)
    normalizedProfile := normalizeSettings(p.Settings)

    return reflect.DeepEqual(normalizedCurrent, normalizedProfile)
}

// normalizeSettings 规范化 settings，处理 JSON 反序列化的类型问题
func normalizeSettings(m map[string]interface{}) map[string]interface{} {
    result := make(map[string]interface{})
    for k, v := range m {
        result[k] = normalizeValue(v)
    }
    return result
}

func normalizeValue(v interface{}) interface{} {
    switch val := v.(type) {
    case map[string]interface{}:
        return normalizeSettings(val)
    case []interface{}:
        result := make([]interface{}, len(val))
        for i, item := range val {
            result[i] = normalizeValue(item)
        }
        return result
    case float64:
        // JSON 数字默认解析为 float64，整数场景转换为 int64
        if val == float64(int64(val)) {
            return int64(val)
        }
        return val
    default:
        return val
    }
}
```

#### Step 4: 更多测试

```go
func TestProfileService_RunCallsRunner(t *testing.T) {
    repo := newMemoryRepo()
    runner := &memoryRunner{}
    svc := NewProfileService(repo, runner)

    repo.Save("test", map[string]interface{}{"model": "opus"})

    err := svc.Run("test")

    require.NoError(t, err)
    assert.Equal(t, []string{"test"}, runner.runs)
}

func TestProfileService_ApplySavesToCurrent(t *testing.T) {
    repo := newMemoryRepo()
    svc := NewProfileService(repo, nil)

    repo.Save("prod", map[string]interface{}{"model": "opus"})

    err := svc.Apply("prod")

    require.NoError(t, err)
    current, _ := repo.LoadCurrent()
    assert.Equal(t, "opus", current["model"])
}

func TestProfileService_IsActive(t *testing.T) {
    repo := newMemoryRepo()
    svc := NewProfileService(repo, nil)

    repo.Save("active", map[string]interface{}{"model": "opus"})
    svc.Apply("active")

    p, _ := repo.GetByName("active")
    assert.True(t, svc.IsActive(*p))
}
```

#### Step 5: ProfileRunner 接口与实现

创建 `internal/service/runner.go`：

```go
package service

import (
    "fmt"
    "os"
    "os/exec"
    "strings"

    "claude-switch/internal/domain"
)

// ProfileRunner 定义运行 profile 的接口
type ProfileRunner interface {
    Run(p domain.Profile) error
}

// ProfileRunnerExec 使用 exec 命令运行 claude
type ProfileRunnerExec struct {
    claudePath string
}

func NewProfileRunnerExec(claudePath string) *ProfileRunnerExec {
    return &ProfileRunnerExec{claudePath: claudePath}
}

func (r *ProfileRunnerExec) Run(p domain.Profile) error {
    cmd := exec.Command(r.claudePath)

    // 设置环境变量（从 profile settings 提取）
    cmd.Env = os.Environ()
    if apiKey, ok := p.Settings["api_key"].(string); ok && apiKey != "" {
        cmd.Env = append(cmd.Env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", apiKey))
    }
    if model, ok := p.Settings["model"].(string); ok && model != "" {
        cmd.Env = append(cmd.Env, fmt.Sprintf("CLAUDE_MODEL=%s", model))
    }

    // 非阻塞启动，让 claude 在前台运行
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    return cmd.Run()
}
```

#### Step 6: Runner 测试

```go
func TestProfileRunnerExec_SetsEnvironmentFromProfile(t *testing.T) {
    // 使用 mock exec，避免真正启动进程
    // 实际项目中可以用 exec.CommandContext + context cancellation
}

func TestProfileRunnerExec_ReturnsErrorIfClaudeNotFound(t *testing.T) {
    runner := NewProfileRunnerExec("/nonexistent/claude")

    err := runner.Run(domain.Profile{Name: "test"})

    assert.Error(t, err)
}
```

### Phase 3 验证清单

- [ ] `go test ./internal/service/...` 通过
- [ ] 测试覆盖率 > 85%
- [ ] 所有业务场景有测试覆盖

---

## Phase 4: TUI 重构

### 目标

重构 Model 使用 Service，保持 UI 行为不变。

### 策略

**保持现有测试通过的前提下重构**：

1. 先写 TUI 的单元测试（模拟 Service）
2. 重构 Model 使用 Service 接口
3. 验证所有原有功能正常

### TDD 循环

#### Step 1: 测试 - TUI Model 初始化

创建 `internal/tui/model_test.go`：

```go
package tui

import (
    "testing"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "claude-switch/internal/domain"
)

// mockService 用于 TUI 测试
type mockService struct {
    profiles []domain.Profile
    current  map[string]interface{}
    created  []string
    applied  []string
    deleted  []string
    runs     []string
}

func (m *mockService) List() ([]domain.Profile, error) {
    return m.profiles, nil
}

func (m *mockService) GetByName(name string) (*domain.Profile, error) {
    for _, p := range m.profiles {
        if p.Name == name {
            return &p, nil
        }
    }
    return nil, domain.ErrProfileNotFound
}

func (m *mockService) Create(name string, settings map[string]interface{}) error {
    m.created = append(m.created, name)
    m.profiles = append(m.profiles, domain.Profile{Name: name, Settings: settings})
    return nil
}

func (m *mockService) Update(name string, settings map[string]interface{}) error {
    return nil
}

func (m *mockService) Delete(name string) error {
    m.deleted = append(m.deleted, name)
    return nil
}

func (m *mockService) Apply(name string) error {
    m.applied = append(m.applied, name)
    return nil
}

func (m *mockService) Run(name string) error {
    m.runs = append(m.runs, name)
    return nil
}

func (m *mockService) LoadCurrent() (map[string]interface{}, error) {
    return m.current, nil
}

func (m *mockService) IsActive(p domain.Profile) bool {
    return false
}

func TestModel_InitializesWithProfiles(t *testing.T) {
    svc := &mockService{
        profiles: []domain.Profile{
            {Name: "p1"},
            {Name: "p2"},
        },
    }

    m := NewModel(svc)

    assert.Len(t, m.profiles, 2)
    assert.Equal(t, "p1", m.profiles[0].Name)
}

func TestModel_CursorNavigation(t *testing.T) {
    svc := &mockService{
        profiles: []domain.Profile{
            {Name: "p1"}, {Name: "p2"}, {Name: "p3"},
        },
    }

    m := NewModel(svc)

    // 向下
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
    assert.Equal(t, 1, m.cursor)

    // 继续向下
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
    assert.Equal(t, 2, m.cursor)

    // 边界
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
    assert.Equal(t, 2, m.cursor) // 不超边界

    // 向上
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
    assert.Equal(t, 1, m.cursor)
}

func TestModel_EnterAppliesProfile(t *testing.T) {
    svc := &mockService{
        profiles: []domain.Profile{
            {Name: "test"},
        },
    }

    m := NewModel(svc)
    m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

    // 应该有命令返回
    require.NotNil(t, cmd)

    // 执行命令
    msg := cmd()

    // 验证调用了 Apply
    assert.Equal(t, []string{"test"}, svc.applied)
}
```

#### Step 2: 重构 Model

修改 `internal/tui/model.go`：

```go
// 内部消息类型
type profilesReloadedMsg struct{}

type profileAppliedMsg struct {
    name string
}

type profileRunMsg struct {
    name string
}

type errorMsg struct {
    err error
}

// viewState 视图状态
type viewState int

const (
    viewList viewState = iota
    viewCreateMenu
    viewCreateForm
    viewEdit
)

type Model struct {
    // 依赖
    service service.ProfileService

    // UI 状态
    state            viewState
    cursor           int
    createMenuCursor int     // 创建菜单光标
    message          string
    errMsg           error   // 错误状态
    loading          bool    // 异步操作中
    width            int
    height           int

    // 数据（从 Service 获取）
    profiles         []domain.Profile
    current          map[string]interface{}

    // 表单状态
    input            textinput.Model
    apiKeyInput      textinput.Model
    apiStep          int
    pendingSaveName  string
    existingProfile  *domain.Profile
    saveOriginalName string
}

// NewModel 构造函数，依赖注入
func NewModel(svc service.ProfileService) Model {
    ti := textinput.New()
    ti.Placeholder = "profile-name"
    ti.CharLimit = 64

    aki := textinput.New()
    aki.Placeholder = "your-api-key"
    aki.CharLimit = 128
    aki.EchoMode = textinput.EchoPassword
    aki.EchoCharacter = '*'

    m := Model{
        service:     svc,
        state:       viewList,
        input:       ti,
        apiKeyInput: aki,
        apiStep:     0,
    }
    m.loadData()
    return m
}

func (m *Model) loadData() {
    var err error
    m.profiles, err = m.service.List()
    if err != nil {
        m.message = fmt.Sprintf("Error: %v", err)
    }
    m.current, _ = m.service.LoadCurrent()
}

// Update 只处理 UI 状态和调用 Service
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil

    case tea.KeyMsg:
        if msg.String() == "ctrl+c" {
            return m, tea.Quit
        }

    case profilesReloadedMsg:
        m.loadData()
        return m, nil
    }

    // 分派到具体处理
    switch m.state {
    case viewList:
        return m.handleList(msg)
    case viewCreateMenu:
        return m.handleCreateMenu(msg)
    // ... 其他状态
    }

    return m, nil
}

// 重构后的按键处理
func (m Model) handleList(msg tea.Msg) (tea.Model, tea.Cmd) {
    keyMsg, ok := msg.(tea.KeyMsg)
    if !ok {
        return m, nil
    }

    switch keyMsg.String() {
    case "q":
        return m, tea.Quit
    case "up", "k":
        if m.cursor > 0 {
            m.cursor--
        }
    case "down", "j":
        if m.cursor < len(m.profiles)-1 {
            m.cursor++
        }
    case "enter":
        return m.applySelected()
    case "r":
        return m.runSelected()
    case "n":
        m.state = viewCreateMenu
        m.createMenuCursor = 0
    case "d":
        return m.deleteSelected()
    }

    return m, nil
}

func (m Model) applySelected() (tea.Model, tea.Cmd) {
    if !m.safeProfileIndex() {
        return m, nil
    }

    p := m.profiles[m.cursor]
    return m, func() tea.Msg {
        if err := m.service.Apply(p.Name); err != nil {
            return errorMsg{err: err}
        }
        return profileAppliedMsg{name: p.Name}
    }
}

func (m Model) runSelected() (tea.Model, tea.Cmd) {
    if !m.safeProfileIndex() {
        return m, nil
    }

    p := m.profiles[m.cursor]
    return m, func() tea.Msg {
        if err := m.service.Run(p.Name); err != nil {
            return errorMsg{err: err}
        }
        return profileRunMsg{name: p.Name}
    }
}
```

### Phase 4 验证清单

- [ ] 所有原有功能正常
- [ ] TUI 测试通过
- [ ] 集成测试（真实 Service + 临时目录）通过

---

## Phase 5: BubbleTea v2 迁移

### 目标

从 v1 迁移到 v2，适配新的 API。

### 迁移清单

| 变更 | v1 | v2 |
|------|-----|-----|
| Import | `github.com/charmbracelet/bubbletea` | `charm.land/bubbletea/v2` |
| View 返回 | `string` | `tea.View` |
| AltScreen | `tea.WithAltScreen()` | `v.AltScreen = true` |
| KeyMsg | `tea.KeyMsg` | `tea.KeyPressMsg` |
| Key 字段 | `msg.String()` | `msg.Code` / `msg.Text` |

### 迁移步骤

1. 更新 go.mod
2. 全局替换 import
3. 修改 View() 方法
4. 适配 KeyMsg 变化
5. 运行测试验证

---

## 测试策略总结

### 测试金字塔

```
       /\
      /  \      E2E (2-3个) 完整用户流程
     /----\
    /      \
   /--------\   Integration (15-20个) Service + 真实 Repo
  /          \
 /------------\
/              \
/   Unit Tests   \  (50+个) Domain + Service + TUI
/__________________\
```

### E2E 测试场景

```go
// e2e/smoke_test.go

// TestFullCRUDLifecycle 完整 CRUD 流程
func TestFullCRUDLifecycle(t *testing.T) {
    tmpDir := t.TempDir()
    settingsPath := filepath.Join(tmpDir, "settings.json")

    repo := repository.NewProfileRepositoryFS(tmpDir, settingsPath)
    svc := service.NewProfileService(repo, nil)

    // 1. 创建
    err := svc.Create("test-profile", map[string]interface{}{"model": "opus"})
    require.NoError(t, err)

    // 2. 应用
    err = svc.Apply("test-profile")
    require.NoError(t, err)

    // 3. 验证生效
    current, err := svc.LoadCurrent()
    require.NoError(t, err)
    assert.Equal(t, "opus", current["model"])

    // 4. 删除
    err = svc.Delete("test-profile")
    require.NoError(t, err)

    _, err = svc.GetByName("test-profile")
    assert.ErrorIs(t, err, domain.ErrProfileNotFound)
}

// TestCorruptedFileTolerance 容错能力
func TestCorruptedFileTolerance(t *testing.T) {
    tmpDir := t.TempDir()
    settingsPath := filepath.Join(tmpDir, "settings.json")

    repo := repository.NewProfileRepositoryFS(tmpDir, settingsPath)

    // 创建有效 profile
    repo.Save("valid", map[string]interface{}{"model": "opus"})

    // 创建损坏的文件
    os.WriteFile(filepath.Join(tmpDir, "corrupted.json"), []byte("not json"), 0644)

    // 列表应该返回有效数据，同时报告错误
    profiles, errs := repo.List()

    assert.Len(t, profiles, 1)
    assert.Equal(t, "valid", profiles[0].Name)
    assert.Len(t, errs, 1)
    assert.Equal(t, "corrupted", errs[0].Name)
}
```

### 测试运行

```bash
# 全部测试
go test ./...

# 带覆盖率
go test -cover ./...

# 特定包
go test ./internal/domain/...
go test ./internal/repository/...
go test ./internal/service/...
go test ./internal/tui/...
```

### CI/CD 集成

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test -cover ./...
      - name: Coverage check
        run: |
          COVERAGE=$(go test -cover ./... -coverprofile=coverage.out | grep -oP '\d+\.\d+%' | head -1)
          if (( $(echo "$COVERAGE < 80" | bc -l) )); then
            echo "Coverage $COVERAGE < 80%"
            exit 1
          fi

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go install golang.org/x/tools/cmd/golangci-lint@latest
      - run: golangci-lint run ./...
```

---

## 实施检查清单

### 每个 Phase 完成标准

- [ ] 所有测试通过
- [ ] 测试覆盖率 > 80%
- [ ] 代码审查通过
- [ ] 集成测试通过

### 最终交付标准

- [ ] `go test ./...` 100% 通过
- [ ] 原有 TUI 功能完全保留
- [ ] 新架构清晰分层
- [ ] 文档更新完成
