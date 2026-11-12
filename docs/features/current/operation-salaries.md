---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-12
source_of_truth:
  - server/internal/service/salary.go
  - server/bootstrap/salary_migration.go
  - server/internal/service/admission_application.go
  - server/internal/service/welfare.go
  - server/internal/service/shop.go
  - server/internal/service/srp.go
  - server/internal/service/fleet.go
---

# 工资管理

Vue 和 React 的「系统管理 → 工资管理」（`/system/salary`）统一维护操作工资和 FC 工资。只有 admin／super_admin 可以读取和修改，普通 admin 同时须满足既有 `system.manage` 军团能力要求。

| 类别 | 配置键 | 计薪时点 | 新安装默认 |
| --- | --- | --- | --- |
| 招新 | `salary.recruitment` | 新人申请整单标记已邀请（小号申请不计薪） | 10 |
| 福利 | `salary.welfare_delivery` | 人工成功发放一份申请 | 10 |
| 商城 | `salary.shop_delivery` | 人工成功发货一份订单 | 10 |
| 补损 | `salary.srp_payout` | 一次成功的最终发放操作，含手工转账登记和伏羲币发放；批量一次也只计一份工资 | 10 |
| FC | `pap.fc_salary`、`pap.fc_salary_limit` | 沿用舰队 PAP 发放工资计算规则 | 保留现值；缺省 400、每月 5 次 |

招新、福利、商城工资按申请／订单逐份计件；补损按成功发放操作计薪，单笔和批量各算一次，不乘以申请数量。普通职权操作人、admin 和 super_admin 金额相同。招新工资只奖励新成员入团，邀请现有成员的小号不计薪。创建福利、初步批准补损、驳回及系统自动发放不计操作工资。金额允许 0，表示停发该类工资；金额最多两位小数。FC 月次数上限为 0 沿用旧语义：不发放新的 FC 工资，既有重新发放规则保持不变。燃料官设置保留原入口。

## 事务与历史

最终状态变更、申请人入账（如有）及处理人工资在同一数据库事务中提交。服务先锁定来源申请／订单并检查仍可处理；重复请求或并发结案不能再次计薪。补损批量发放为申请人逐份入账，处理人工资整批只入账一次，任一失败全部回滚；空批次和重复请求不计薪。批量工资以本批首份申请 ID 作为稳定来源标识（`srp_batch:<id>`），原因记录批次数量和起始申请。手工转账登记失败时网站保留未发放状态，不撤销游戏内已经执行的转账。

工资遵守 `wallet.user.enabled`，正金额工资的钱包写入失败会阻止结案。零工资表示管理员明确停发，不写零额流水，也不因工资钱包禁用阻止该次完成；申请人入账仍遵守原策略。

工资按完成时读取的配置支付。历史流水金额和来源关联不因后续设置变更而修改，也不补发旧单。招新流水为 `recruitment_invitation_salary`，其余三类分别为 `welfare_delivery_salary`、`shop_delivery_salary`、`srp_payout_salary`，关联稳定的来源 ID；FC 继续使用 `pap_fc_salary`。招新另保留每申请唯一的 `admission_salary` 关联，工资为 0 时不创建计薪行。

启动迁移将旧 `pap.admin_award` 的自定义值（含 0）分别初始化为福利、商城工资。新键已存在时不会覆盖；FC 原键原值不迁移、不重置。旧配置键仅保留历史数据，PAP 兑换页面和 API 不再编辑工资。

## 接口

四类人工计薪操作在提交前均提供确定／取消选项，并以统一格式显示本次可获得的工资合计，不展示单价和计薪数量明细。招新仅新人申请按整单一份；福利和商城各按单一份；补损单笔和批量都显示一次工资，批量先刷新汇总确认存在可发放申请，不按申请数量乘算。零工资也明确显示。取消或读取工资失败时不发送完成请求。工资以成功完成的操作和当时设置为准，驳回、小号申请邀请与系统自动处理不显示领取工资提示。

确认框通过 `GET /api/v1/operation-salary-rates` 读取四项当前单价，使用与全局角标相同的登录用户门槛（guest 不可读），不暴露 FC 配置，也不授予任何处理权限。各业务完成接口继续独立执行原权限校验。

- `GET /api/v1/system/salary-settings`
- `PUT /api/v1/system/salary-settings`

读写结构均为 `recruitment`、`welfare_delivery`、`shop_delivery`、`srp_payout`、`fc_salary`、`fc_salary_monthly_limit`；更新必须完整提供六项，显式 0 有效。配置批量写入并记录 `salary_settings_update` 审计。
