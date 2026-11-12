---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-12
source_of_truth:
  - server/internal/model/admission.go
  - server/internal/repository/admission.go
  - server/bootstrap/admission_migration.go
  - server/internal/service/admission_application.go
  - server/internal/handler/admission.go
  - static/src/views/admissions/index.vue
  - static-react/src/pages/admissions-page.tsx
---

# 军团加入申请

## 模块目的

独立保存账号向伏羲或 FMA 提交的整单申请、绑定人物、人工处理历史及招新工资依据。与通用工单分开建模，避免其分类、回复和权限语义限制新流程。

## 当前入口

Vue 和 React 均在“军团申请”提供 `/admissions/new`（新人）、`/admissions/alt`（小号）、`/admissions/history`（本人历史）、`/admissions/applications/:id`（详情）与 `/admissions/review`（招新管理）。完整 API 与鉴权边界见[路由索引](../../api/route-index.md#corporation-admissions)。

## 申请与人工检查

申请以账号为单位，选择伏羲 `98185110` 或 FMA `98824507`，并选择一个或多个已绑定人物。申请人须完成资料，至少登记 QQ 或 Discord；只有 Discord 也可申请。联系人沿用账号绑定规则，不为每个人物分别填写。

中文目标军团显示为“伏羲新手团（Fuxi Military Academy）”（`98824507`）和“伏羲主团 （主号需7百万以上技能。已在伏羲的人，小号无要求）”（`98185110`）；新申请默认选择伏羲新手团，编辑已有申请保留原目标。新人表单不重复展示账号联系资料，说明中的“联盟SeAT”链接至 `https://seat.winterco.org`，引导绑定登录器截图中的所有人物及 QQ，并展开教学图核对。两个截图上传标签均以红色 `*` 和“必填”标明必交；账号只有 Discord 仍可申请，后端联系方式资格不变。

小号 SeAT 绑定提醒仅显示在申请人的填写及编辑表单中，审核详情不显示。招新管理列表不显示单条申请的邀请操作说明。

新成员申请的待处理审核区提醒审核员比对登录器与 SeAT 截图的人物数量及逐一对应关系；缺漏时联系申请人补全绑定，缺漏较多或拒绝补全时可驳回。该提醒不显示在小号审核区或列表中。

任一绑定人物命中 `allowed_corporations` 即为成员：成员使用小号入口，非成员使用新人入口。主人物、展示职权均不替代该判定；资格未知或读取失败时隐藏创建入口，允许重试。资格变化不剥夺本人历史读取权。

新人恰好提供以下两类材料，由招新官人工核对，无 SeAT API、OCR 或技能点自动判定：

1. 游戏登录器截图：核对登录器所有人物均已绑定到联盟 SeAT。
2. 联盟 SeAT 完成页截图：同一画面显示绑定人物和 QQ／Discord；申请伏羲还须显示至少 700 万技能点，FMA 无此门槛。

不存在独立的 QQ、Discord 或技能点上传项。小号免截图，表单显示红色 SeAT 绑定提醒。要一起申请的人物，先在“人物管理”通过 EVE SSO 绑定到同一账号。已在目标军团的人物不能重复提交，但等待处理期间实际入团不阻止招新官登记原单。

审核人读取待处理申请时，系统按本地[联盟黑名单](administration.md#联盟黑名单)比对申请人：
- 人物取本单人物与账号当前绑定人物的并集，按 character_id 精确匹配；
- QQ、Discord 取本单快照与账号当前联系方式的并集，按黑名单的同一规则规范化后精确匹配：Discord 不区分大小写、忽略开头 `@`，QQ 不做前缀或子串匹配；
- 账号已删除时只比对申请快照。

比对在读取时计算、不落库；invited、rejected 申请不再比对。命中时不能登记 Invited，Reject 不受影响。

教学图 `public/admissions/qq-binding.png`、`seat-completed.png` 在两端均保留用户提供的原图，依次演示 QQ 填写与完成示例，支持打开原图；图片中不追加 Discord 注释。

## 使用步骤

申请人绑定人物并完成联系方式后，从对应入口选择目标及本单人物。新人上传登录器和 SeAT 完成页两图，提交后在本人历史查看。pending 可修改；rejected 允许修改并以原 ID 补正重提；已过期材料必须在重提时重新上传两图。重提保留此前的驳回记录。invited 只读，无撤回入口。

招新官在管理页按状态、申请种类、目标筛选，查看全部人物、联系人和证据。在游戏内完成本单全部邀请后登记 **Invited**；或填写必填原因后 **Reject**。网站不调用游戏邀请接口，也不支持部分人物结案。冲突时刷新当前版本再处理；新人申请遇钱包禁用时按提示联系超管调整军团钱包策略后重试。

## 数据与状态

| 表 | 职责 |
| --- | --- |
| `admission_application` | 账号、申请种类、目标军团、当前状态、版本、提交联系人及处理人快照、结案／证据期限 |
| `admission_character` | 本单人物 ID 与名称快照 |
| `admission_character_claim` | pending 人物占用；人物 ID 唯一，结案释放，重提重新取得 |
| `admission_evidence` | 独立存放压缩 JPEG 与尺寸元数据，不在列表加载图像 |
| `admission_processing_event` | 每次处理／补正的版本和快照，不复制图片 |
| `admission_salary` | 每申请至多一笔工资及唯一钱包流水关联 |

状态为 pending、invited、rejected。pending 可编辑，rejected 可编辑并以原 ID 重提到 pending；invited 为申请人不可修改的最终结果。每次写入必须检查提交版本；行锁和条件更新防止旧审阅结果覆盖新材料。

## 权限边界

服务负责 owner、成员资格和目标军团审核范围校验。列表先附 owner 或军团 scope 再计数、分页；无审核 scope 时拒绝管理请求。不能把全局 recruiter 标签当成全部军团的读取授权。申请人（含 admin）不能审核或处理本人申请：详情对 owner 不返回审核动作，本人待处理申请不计入审核徽章。派生 recruiter 只授权审核，不构成网站成员资格。派生权限、来源军团校验、管理员绕过及 24 小时时效以[权限架构](../../architecture/auth-and-permissions.md#军团申请与派生-recruiter)为准。

黑名单命中只在审核时显示：详情 `blacklist_hits` 列出命中值及黑名单理由，审核列表的 pending 行带 `blacklisted`。申请人读取本人申请或历史时（包括审核人查看本人申请）得到空数组和 `false`。

列表读取人物、详情读取关联记录、私有图片读取 bytes，均在事务中持有申请行的共享锁。目标军团与本次授权必须在整次读取期间保持一致，避免申请人同时修改目标／材料时，旧军团审核者读到新军团材料。修改、处理及清理使用同一申请行的排他锁；PostgreSQL 并发回归覆盖三种读取路径。

## 工资与证据期限

招新工资只奖励新成员入团：仅新人申请按工资管理的招新类别逐单计薪（默认 10 伏羲币，0 停发），收款人为实际登记 Invited 的处理人，包含 admin／super_admin。小号申请的申请人已是成员，登记 Invited 不计薪、不写工资行或钱包流水，也不受 `wallet.user.enabled` 限制。申请、人物占用释放、工资、钱包流水、处理历史及必需审计在同一事务中提交。重复或并发请求至多成功一次；任一步失败不结案、不发薪。新人申请在 `wallet.user.enabled` 禁用时整单保持 pending，Reject 不受该发薪限制。新人申请不形成零工资结案或待补工资。流水类型见[交易与钱包](commerce.md#招新官工资)。

浏览器先压缩，服务器再次验证 PNG／JPEG 并重新编码，只保存最长边 1280px、JPEG 质量 70 的图片；每图最多 2 MiB，multipart 请求体最多 5 MiB。服务器限制解码尺寸、像素数并去掉原始元数据。私有证据必须携带 JWT，经 owner／目标军团授权读取，响应 `private, no-store`；列表和审计不包含图片 bytes，公开 uploads 无证据路径。

从 Invited／Reject 结案时间起保留三个 UTC 自然月，目标月份没有对应日期时截到月末。rejected 仅编辑不延长期限，重新提交才清除旧期限。到期读取立即拒绝，日常清理任务删除图片，申请、人物、处理历史、工资和审计继续保留；任务与重提使用同一申请锁，避免清理新材料。频率及重试见[任务管理](task-manager.md#申请证据清理)。教学静态图不参与清理。

## 部署、切换与回退

启动时 additive 迁移六张申请表，同时为现有旧加团工单分类建立永久停用标记；见[工单切换](ticket-system.md#加团工单切换)。成员允许列表的实际配置键为 `app.allow_corporations`；部署时确认包含应为成员的军团，尤其 FMA `98824507`。此检查不改变招新官按目标军团职权派生的审核范围。军团职权来源新增字段默认为 0，旧快照在下一次成功 ESI 同步补齐前不授予招新审核范围；不人为缩短现有 24 小时周期。

回退应用版本时保留申请、证据、工资、审计及新列，不执行破坏性降级。旧二进制不理解永久停用标记，因此回退期间必须维持旧分类 disabled，并阻止通过分类管理恢复其创建路径。不要把旧工单迁移成申请或补发工资，也不要把取消新入口视为恢复旧加团工单的授权。QQ 群通知本期未接入。

## 持久设计与不变量

- 一份申请对应一个 user、一个目标军团、一个或多个人物；数据库限制同一 user 同时至多一份 pending 新人单。
- 人物占用独立采用唯一键，避免并发先查再写产生多个可独立处理的副本；失败与申请变更同事务回滚。
- 工资表以申请 ID 为主键，钱包流水关联另加唯一约束。钱包 ref_id 本身不保证唯一，不能取代工资约束和申请行锁。
- 图片与申请／处理／工资表分离，使证据到期后可以删除 bytes，仍保留业务结果、处理快照和计薪依据。
- additive 迁移可重复执行，回退应用不应删除已有申请、工资或审计数据。
- 登记 Invited 在处理事务内重新比对黑名单，位置在绑定人物校验之后、任何写入之前，因此页面打开后刷新的黑名单同样生效。命中返回 `admissions.errors.blacklisted`，申请保持 pending，版本、处理历史、工资和钱包流水均不变。
- 黑名单比对（`blacklistHitsTx`）不论申请数量，每次固定三次不加锁的批量查询：绑定人物、账号联系方式、黑名单条目。列表不产生 N+1；只读路径不复用带 `FOR UPDATE` 的处理查询。

## 主要代码文件

- `server/internal/model/admission.go`
- `server/internal/repository/admission.go`
- `server/bootstrap/admission_migration.go`
- `server/internal/service/admission_application.go`、`admission_evidence.go`、`recruiter_role.go`、`blacklist_match.go`
- `server/internal/handler/admission.go`、`server/jobs/admission_evidence_cleanup.go`
- `static/src/views/admissions/`、`static-react/src/pages/admissions-page.tsx`

验证与工具要求见[测试与验证规范](../../standards/testing-and-verification.md)。

## 详情、编辑与待处理提示

详情始终只读，审核人仅可登记“已邀请”或“拒绝”。申请人从本人历史进入独立 `/admissions/applications/:id/edit` 编辑页；已邀请后不可编辑，驳回后仍可修改重提。页面不重复侧边导航，也不显示工资金额／工资处理人。

列表的“查看详情”与本人历史的“修改申请”均为按钮。小号申请免截图核对，招新管理列表在待处理小号行的“查看详情”前提供“已邀请”按钮，确认后按该行版本直接登记，无需进入详情；与详情处理相同，版本已变化时不提交，确认框不显示工资。新人申请和驳回仍须进入详情处理；包含当前用户自己人物的申请不显示列表邀请入口，后端仍拒绝处理本人申请。

黑名单命中在两端前端表现一致：

- 详情：审核卡片顶部显示红色提示（Vue `ElAlert type="error"`，React `role="alert"` 红框块），逐条列出类型、值（人物显示为“名字（ID）”）和黑名单理由，理由保留换行。“已邀请”禁用，“拒绝”照常可用。
- 列表：招新管理中 `blacklisted` 行在状态旁显示红色“黑名单”标签；该行若有列表“已邀请”按钮，则渲染为禁用。
- 处理失败：后端返回 `admissions.errors.blacklisted`（例如打开页面后黑名单刚刷新）时，页面显示本地化提示并重新加载详情或列表，以显示最新命中。

全局 `admission_pending` 仅统计当前用户可审核军团内的 pending 申请，在“招新管理”及“军团申请”父菜单显示红色数字；其他军团申请不计入，零值或无权限不显示。创建、修改重提和处理成功后刷新提示，失败不影响已完成的业务。
