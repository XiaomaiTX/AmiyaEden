---
status: draft
doc_type: draft
owner: backend
last_reviewed: 2026-03-23
source_of_truth:
  - server/internal/service/srp.go
  - server/internal/router/router.go
  - server/internal/service/eve_sso.go
  - server/bootstrap/scopes.go
---

# SRP 发放后官员邮件通知（Issue #15）

## 已确认需求

- 在每次 SRP 发放完成后，系统需要以 SRP officer 身份向领取人主角色发送 EVE 游戏内邮件。
- SRP officer 采用“当次点击发放的登录用户”，发信角色取该用户的主角色（`primary_character_id`）。
- 邮件接收人是该补损申请对应领取人的主角色（primary character）。
- 邮件内容必须包含本次发放关联的 KM 明细，以及每条 KM 对应的发放金额。
- 该能力属于 SRP 发放流程的一部分，触发点在“发放成功之后”。

## 当前状态

- 当前 SRP 已有审核与发放能力，但发放后没有 EVE Mail 发送逻辑。
- 当前已注册 `esi-ui.open_window.v1`，尚未注册邮件发送 scope。
- 当前发放接口为单条申请发放：`PUT /api/v1/srp/applications/:id/payout`。

## 细化后的实现范围（本期）

### 1. 触发语义

- 单条发放：每次单条发放成功后立即发送 1 封邮件。
- 批次发放：当一次操作内发放多条申请时，按“接收人主角色”聚合，每个接收人发送 1 封邮件。
- 失败隔离：发放成功不因邮件失败而回滚；邮件失败要记录并可重试。

### 2. 发送身份（SRP officer）

- 发信人固定为“当次发放操作人（payer）”的主角色（`primary_character_id`）。
- 该角色必须满足：
  - 在本系统已绑定；
  - token 可用；
  - 已授权 `esi-mail.send_mail.v1`。
- 若主角色缺失、角色不存在或授权不合法：
  - 允许发放继续完成；
  - 在响应和日志中标记邮件未发送原因。

### 3. 接收人解析

- 接收角色来自申请所属用户的 `primary_character_id`。
- 若用户无主角色或主角色不存在：
  - 该条进入邮件失败记录；
  - 不影响发放主流程。

### 4. 邮件内容结构

- 标题建议：`[SRP Payout] <character_name> - <total_isk> ISK`。
- 正文固定包含：
  - 收件人角色名；
  - 发放总金额；
  - 明细列表（按申请逐条）：
    - killmail_id
    - killmail_time
    - ship_type_id / ship_name（有则显示）
    - final_amount
  - 操作人（发放人）信息；
  - 说明性尾注（自动发送）。

### 5. 权限与 Scope

- 新增并注册 scope：`esi-mail.send_mail.v1`（模块 `srp`，required=true）。
- SRP 发放接口权限保持现状（`srp:review`），不新增前端按钮权限模型。
- 发送邮件的授权检查放在 service 层，不放在 handler。

### 6. 幂等与重试

- 新增“发放邮件日志”记录，至少包含：
  - 发放申请 ID
  - recipient_character_id
  - sender_character_id
  - mail_id（ESI 返回）
  - status（success/failed）
  - error_message
  - created_at
- 同一申请不重复发送成功邮件（幂等键：`application_id`）。
- 提供后台重试入口（仅重试失败记录）。

### 7. 可观测性

- 记录结构化日志字段：
  - application_id
  - payer_user_id
  - recipient_character_id
  - sender_character_id
  - esi_status_code
  - error
- 管理端可查看最近邮件发送结果（成功/失败）。

## 建议接口变更

- 保持现有单条发放接口不变。
- 新增批量发放接口（建议）：`PUT /api/v1/srp/applications/payout-batch`。
- 请求体建议：
  - `application_ids: number[]`
  - `final_amount_map?: Record<number, number>`
- 响应体建议附带邮件结果汇总：
  - `mail_success_count`
  - `mail_failed_count`
  - `mail_failures[]`

## 验收标准（确认版）

- 用例 A：单条已审批申请发放成功后，接收人主角色收到 1 封邮件，正文含该条 KM 与金额。
- 用例 B：批量发放多条、同一接收人多条申请时，仅收到 1 封聚合邮件。
- 用例 C：邮件发送失败时，发放状态仍为已发放，且失败记录可查询。
- 用例 D：发送角色缺少 `esi-mail.send_mail.v1` 时，系统给出明确错误原因并记录日志。
- 用例 E：重复触发同一申请的邮件发送不会产生重复成功记录。

## 非目标（本期不做）

- 不改造 SRP 审核流程。
- 不改变 SRP 金额计算规则。
- 不在前端暴露复杂邮件模板编辑器。

## 迁移到 current 文档条件

- 当邮件发送链路（配置、发放触发、日志、重试）全部落地并通过联调后，
  - 将已实现部分迁移到 `docs/features/current/srp.md`；
  - 本草案只保留后续优化项或直接删除。
