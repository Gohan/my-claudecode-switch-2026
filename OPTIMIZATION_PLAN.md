# Claude Code Switch 优化计划

## 优化项优先级评估

### P0 - 高优先级（功能正确性/安全性）

| # | 优化项 | 风险 | 预估工作量 |
|---|--------|------|------------|
| 1 | 边界检查：多处 `m.profiles[m.cursor]` 缺少越界检查 | 可能导致 panic | 15min |
| 2 | Profile 名称验证：Save 函数未校验非法字符 | 文件名注入风险 | 20min |

### P1 - 中优先级（代码质量/可维护性）

| # | 优化项 | 影响 | 预估工作量 |
|---|--------|------|------------|
| 3 | 代码重复：updateSaveZAI/updateSaveTencentCloud 重复 80%+ | 维护困难，bug 扩散 | 30min |
| 4 | List() 静默忽略错误：加载失败的 profile 被跳过 | 用户不知情 | 20min |
| 5 | 视图重复：viewSaveZAI/viewSaveTencentCloud 模板相同 | 可维护性 | 15min |

### P2 - 低优先级（性能/小优化）

| # | 优化项 | 影响 | 预估工作量 |
|---|--------|------|------------|
| 6 | IsActive 函数优化：可减少一次长度比较 | 极小 | 10min |
| 7 | MaskSensitive 优化：预编译敏感词检查 | 极小 | 10min |
| 8 | 目录创建优化：避免重复 mkdir | 极小 | 10min |
| 9 | 视图模型映射：viewList 复用 GetModelMapping | 代码整洁 | 10min |

---

## 执行计划

### Phase 1: 安全与稳定性（P0）✅

- [x] **1.1** 添加边界检查函数 `safeProfileIndex()`
- [x] **1.2** 在所有使用 cursor 访问 profiles 的地方添加检查
- [x] **1.3** 在 `profile.Save()` 中添加名称验证（禁止字符：/ \ : * ? " < > |）

### Phase 2: 代码质量（P1）✅

- [x] **2.1** 提取 `updateSaveAPI()` 通用函数处理 ZAI/Tencent 的公共逻辑
- [x] **2.2** 提取 `viewSaveAPICommon()` 通用视图函数（延后，与 2.3 一起做）
- [x] **2.3** 重构 `updateSaveZAI` 和 `updateSaveTencentCloud` 使用通用函数
- [x] **2.4** 在 `profile.List()` 中收集并返回加载错误

### Phase 3: 小优化（P2）

- [ ] **3.1** 优化 `IsActive()` 函数逻辑
- [ ] **3.2** 优化 `MaskSensitive()` 预编译敏感词列表为 map
- [ ] **3.3** 优化 `Save()` 先检查目录是否存在
- [ ] **3.4** 重构 `viewList()` 使用 `GetModelMapping()` 减少硬编码

---

预计总工作量：约 2.5 小时

建议按 Phase 顺序执行，每个 Phase 完成后可单独测试。
