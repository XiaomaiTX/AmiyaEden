---
status: draft
doc_type: spec
owner: engineering
last_reviewed: 2026-05-15
source_of_truth:
  - server/internal/router/router.go
  - server/internal/service/audit_service.go
  - server/internal/service/role.go
  - server/internal/service/sys_wallet.go
  - server/internal/service/welfare.go
  - server/internal/service/srp.go
  - server/internal/service/shop.go
  - server/internal/service/task.go
  - server/internal/service/sys_webhook.go
  - server/internal/service/auto_role.go
  - docs/features/current/audit-log.md
---

# 审计覆盖缺口补齐草案

## 背景

当前系统已具备统一审计查询与导出能力，但业务行为接入审计日志仍不完整。
本草案聚焦“哪些行为未接入审计系统”，并给出可直接执行到代码文件层面的补齐计划。

判定口径：以后端 `service` 层是否调用 `AuditService.RecordEvent` / `RecordEventTx` 作为“已接入审计”标准。

## 当前覆盖现状

已明确接入审计的行为（基于代码扫描）：

- 权限与职权：用户职权分配、ESI 自动职权映射
- 钱包：后台调账与统一钱包差量入口
- 审批：SRP、福利、商城订单审批
- 任务：任务手动执行、任务调度更新、审计导出任务与归档任务自身
- 配置：Webhook、新人/导师/福利设置、PAP 汇率、部分建筑配置

## 未接入审计的重点行为（首轮清单）

以下模块存在写操作接口，但 service 层未发现审计写入 hook：

1. 用户管理
- 用户更新、删除、身份切换等后台高风险行为
- 参考：`server/internal/service/user.go`

2. 工单管理
- 工单状态更新、后台回复、工单分类增删改
- 参考：`server/internal/service/ticket.go`

3. 工具书签管理
- 后台工具书签增删改
- 参考：`server/internal/service/tool_bookmark.go`

4. 伏羲堂内容管理
- 页面配置、卡片增删改与排序
- 参考：`server/internal/service/fuxi_hall.go`

5. 舰队与舰队配置
- 舰队管理、成员管理、邀约、PAP 发放、配置增删改等关键写操作
- 参考：`server/internal/service/fleet.go`、`server/internal/service/fleet_config.go`

6. 技能计划管理
- 技能计划与个人技能计划创建/编辑/删除/排序
- 参考：`server/internal/service/skill_plan.go`

7. 新人与导师关系行为
- 新人关联、导师申请与关系处理、奖励处理等
- 参考：`server/internal/service/newbro_service.go`、`server/internal/service/newbro_affiliation.go`、`server/internal/service/mentor_service.go`、`server/internal/service/mentor_reward.go`

8. 运维/批处理后台行为
- SDE 配置与更新、系统批量配置、联盟 PAP 管理等
- 参考：`server/internal/service/sde.go`、`server/internal/service/sys_config_batch.go`、`server/internal/service/alliance_pap.go`

## 目标与非目标

目标：

- 建立业务写操作的审计覆盖基线
- 分批补齐高风险行为的审计日志
- 为后续新增写操作提供防回退约束

非目标：

- 不在本阶段重构审计数据模型
- 不在本阶段恢复前端导出 tab 的完整交互
- 不将纯查询型接口强制纳入审计

## 审计接入统一规范（实施前置）

### action 命名

- 统一格式：`<domain>_<operation>`
- 示例：`user_update`、`ticket_reply`、`fleet_member_remove`

### category 建议映射

- `user_admin`
- `ticket_admin`
- `content_admin`
- `fleet_ops`
- `skill_plan`
- `newbro_mentor`
- `ops_batch`

### 字段最小集（所有写操作）

- `category`
- `action`
- `result`（`success` / `failed`）
- `actor_user_id`
- `target_user_id`（适用时）
- `request_id`
- `resource_id`（适用时）
- `details_json`

### details_json 结构建议

```json
{
  "input": "参数摘要(脱敏)",
  "before": "变更前关键字段摘要",
  "after": "变更后关键字段摘要",
  "reason": "失败原因或业务说明"
}
```

## 文件级实施计划

### 阶段 0：基线与清单固化

目标：把“接口 -> service 方法 -> 审计状态”固化为可追踪清单。

涉及文件：

- `docs/features/current/audit-log.md`
- `docs/specs/draft/audit-coverage-gap-remediation.md`（本文档）

TODO：

- [ ] 在 `docs/features/current/audit-log.md` 新增“覆盖矩阵”章节
- [ ] 补齐首轮模块 action 清单（含 category/action/result）
- [ ] 标记每个 action 的目标落地文件与 owner
- [ ] 本草案已登记为周期性任务，见 `docs/standards/periodic-review-cadence.md § Audit Coverage Gap Review`

### 阶段 1：高风险后台操作补齐（优先）

目标模块：`user`、`ticket`、`tool_bookmark`、`fuxi_hall`

涉及代码文件：

- `server/internal/service/user.go`
- `server/internal/service/ticket.go`
- `server/internal/service/tool_bookmark.go`
- `server/internal/service/fuxi_hall.go`
- `server/internal/service/audit_service.go`（仅在需要补充 helper 时）

TODO（文件级）：

- [ ] `user.go`：为更新、删除、身份切换等写路径补 `RecordEvent/RecordEventTx`
- [ ] `user.go`：补失败分支审计，确保业务错误与系统错误都落审计
- [ ] `ticket.go`：为状态流转、后台回复、分类增删改补审计
- [ ] `tool_bookmark.go`：为增删改及排序行为补审计
- [ ] `fuxi_hall.go`：为页面配置、卡片增删改、排序补审计
- [ ] 所有新增 action 同步登记到 `docs/features/current/audit-log.md`

建议 action（首批）：

- `user_update` `user_delete` `user_identity_switch`
- `ticket_status_update` `ticket_reply` `ticket_category_create` `ticket_category_update` `ticket_category_delete`
- `tool_bookmark_create` `tool_bookmark_update` `tool_bookmark_delete` `tool_bookmark_sort`
- `fuxi_page_update` `fuxi_card_create` `fuxi_card_update` `fuxi_card_delete` `fuxi_card_sort`

### 阶段 2：运营核心模块补齐

目标模块：`fleet`、`fleet_config`、`skill_plan`、`newbro*`、`mentor*`

涉及代码文件：

- `server/internal/service/fleet.go`
- `server/internal/service/fleet_config.go`
- `server/internal/service/skill_plan.go`
- `server/internal/service/newbro_service.go`
- `server/internal/service/newbro_affiliation.go`
- `server/internal/service/mentor_service.go`
- `server/internal/service/mentor_reward.go`

TODO（文件级）：

- [ ] `fleet.go`：舰队创建/修改/关闭、成员增减、邀约、PAP 发放补审计
- [ ] `fleet_config.go`：配置增删改补审计并记录 before/after 摘要
- [ ] `skill_plan.go`：计划创建/编辑/删除/排序补审计
- [ ] `newbro_service.go` + `newbro_affiliation.go`：关系绑定/解绑/转移补审计
- [ ] `mentor_service.go` + `mentor_reward.go`：导师申请流转、奖励发放/撤销补审计

建议 action（样例）：

- `fleet_create` `fleet_update` `fleet_close` `fleet_member_add` `fleet_member_remove` `fleet_pap_issue`
- `fleet_config_create` `fleet_config_update` `fleet_config_delete`
- `skill_plan_create` `skill_plan_update` `skill_plan_delete` `skill_plan_sort`
- `newbro_bind` `newbro_unbind` `mentor_application_review` `mentor_reward_issue` `mentor_reward_revoke`

### 阶段 3：运维与批处理模块补齐

目标模块：`sde`、`sys_config_batch`、`alliance_pap`

涉及代码文件：

- `server/internal/service/sde.go`
- `server/internal/service/sys_config_batch.go`
- `server/internal/service/alliance_pap.go`

TODO（文件级）：

- [ ] `sde.go`：配置写入、更新任务触发、手动同步入口补审计
- [ ] `sys_config_batch.go`：批量配置执行补审计（参数摘要 + 影响范围）
- [ ] `alliance_pap.go`：联盟 PAP 规则维护与批处理补审计
- [ ] 批处理行为统一写入任务标识（job_id/task_id）到 details

### 阶段 4：防回退机制

目标：把“审计接入”变为可持续规则。

涉及文件：

- `server/internal/service/*_test.go`（新增/补齐）
- `docs/standards/testing-and-verification.md`
- `docs/features/current/audit-log.md`

TODO：

- [ ] 为阶段 1~3 关键写路径补“成功 + 失败”双断言单测
- [ ] 增加审计断言 helper（如已有则复用）并统一断言字段最小集
- [ ] 在测试规范中加入“新增高风险写操作必须评估审计接入”
- [ ] PR checklist 增加“action 命名 + 字段完整性 + 失败分支审计”检查项

## 测试计划（代码文件粒度）

### 后端单元测试

建议新增或补齐：

- `server/internal/service/user_test.go`
- `server/internal/service/ticket_test.go`
- `server/internal/service/tool_bookmark_test.go`
- `server/internal/service/fuxi_hall_test.go`
- `server/internal/service/fleet_test.go`
- `server/internal/service/fleet_config_test.go`
- `server/internal/service/skill_plan_test.go`
- `server/internal/service/newbro_service_test.go`
- `server/internal/service/mentor_service_test.go`
- `server/internal/service/sys_config_batch_test.go`

每个关键 action 至少覆盖：

- 1 条成功审计断言
- 1 条失败审计断言
- 字段校验：`category/action/result/request_id/details_json`

### 集成验证

- 通过关键后台写接口触发后，验证 `/system/audit` 可检索到对应 action
- 验证 `target_user_id/resource_id` 在适用场景不为空

## 里程碑与交付清单

### M1（阶段 1 完成）

- [ ] `user/ticket/tool_bookmark/fuxi_hall` 写操作审计补齐
- [ ] 对应测试通过
- [ ] 文档矩阵更新

### M2（阶段 2 完成）

- [ ] `fleet/fleet_config/skill_plan/newbro/mentor` 写操作审计补齐
- [ ] 对应测试通过
- [ ] 文档矩阵更新

### M3（阶段 3+4 完成）

- [ ] 运维批处理模块补齐
- [ ] 防回退机制落地（测试与 checklist）
- [ ] 审计覆盖基线稳定

## 风险与缓解

风险：

- 审计粒度不一致导致检索价值下降
- 失败路径漏记，导致关键异常不可追溯
- details 过大或过杂，影响可读性

缓解：

- 统一 category/action 命名与字段最小集
- 明确失败也必须落审计
- 对 details 做“摘要优先”，避免全量冗余载荷

## 默认假设

- 本次“未接入”统计以后端 service 层实际写审计为准
- 纯查询型 POST 默认不纳入审计，除非具备敏感导出或批量执行语义
- 实施优先级：`user/ticket/tool_bookmark/fuxi_hall` > `fleet/fleet_config/skill_plan` > 其余模块
