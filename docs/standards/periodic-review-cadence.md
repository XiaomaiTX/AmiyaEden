---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-07-21
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/standards/pre-completion-checklist.md
  - docs/standards/documentation-governance.md
---

# 周期性核查节奏

## 适用范围

本标准管理不依附于单次代码变更、按固定周期重复执行的核查任务。这类任务用于在时间维度上复核某类覆盖度、漂移或卫生状况，并产出可追溯的记录。

本文档只是**已登记周期性任务的索引**，不复制任务正文。每条条目指向真正承载该任务清单的文档。

## 与其他文档的边界

- `docs/standards/pre-completion-checklist.md` 是单次变更的完成门槛。
- `docs/standards/regression-test-plan.md` 是单次缺陷修复的测试选型指南。
- `docs/standards/testing-and-verification.md` 是命令与覆盖策略的唯一权威来源。
- 本文档专门承载"即使近期没有相关变更也应当按日历节奏执行"的复核任务。

如果某项检查属于单次变更流程，仍留在 `pre-completion-checklist.md`，不在这里登记。只有当某项复核在没有任何相关变更落地时也有意义，才登记到本文档。

## 核心规则

- 每条已登记任务必须写明：节奏、判定口径、输出物、负责角色、工作清单指针。
- 每条任务的工作清单放在真正承载该主题的文档里：
  - 进行中的补齐工作 -> `docs/specs/draft/*.md`
  - 已稳定的覆盖度跟踪 -> `docs/features/current/*.md`
- 每个周期必须留下带日期的产出。可接受的形式：
  - 在输出物中追加一条带日期的复核记录，说明本次发生了什么变化
  - 在工作清单中新增或调整工作项
  - 明确写出"无漂移"结论并附复核人日期
- 默认节奏为季度。仅在漂移频繁时收紧，在信号稳定时放宽。
- 任务关闭时（缺口补齐完成、信号稳定），将其移至「归档」小节并附关闭日期，不要静默删除条目。

## 已登记的周期性核查任务

### 审计覆盖缺口复核

- **节奏：** 季度
- **判定口径：** 后端 `service` 层任何写路径未调用 `AuditService.RecordEvent` / `RecordEventTx` 即视为覆盖缺口。纯查询接口与敏感导出相关规则以工作清单为准。
- **输出物：** 更新 `docs/features/current/audit-log.md § 审计接入现状` 中的覆盖快照，附本次复核日期与当前缺口清单。
- **负责角色：** engineering
- **工作清单：** `docs/specs/draft/audit-coverage-gap-remediation.md`

## 单周期自检清单

关闭每个周期前需确认：

- [ ] 判定口径仍与当前代码库一致；若信号定义发生变化，先更新口径。
- [ ] 已产出并标注日期的输出物。
- [ ] 开放工作项已关闭、保留或升级处理。
- [ ] 负责角色仍为正确联系人。
- [ ] 当前节奏仍然合适；根据观察到的漂移收紧或放宽。

## 允许的例外

- 节奏之外的一次性审计无需登记到本文档。
- 每个周期允许跳过一次，但负责人必须在输出物中写明跳过原因与重新排期日期。

## 归档

_已关闭的周期性核查任务连同关闭日期与简要结论保留在此小节。当前为空。_
