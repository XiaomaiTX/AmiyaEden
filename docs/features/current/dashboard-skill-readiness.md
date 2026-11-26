---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-26
source_of_truth:
  - server/internal/router/router.go
  - server/internal/service/skill_readiness.go
  - server/internal/repository/skill_readiness.go
  - static/src/views/dashboard/skill-readiness/index.vue
  - static-react/src/pages/dashboard-skill-readiness-page.tsx
---

# 军团技能达标查询

## 当前交付状态

已提供 Vue 与 React 仪表盘入口、管理查询页面、API wrapper、类型和中英文文案。用户可筛选军团与方案，查看全范围统计和分页用户，按需展开角色并查看方案比对及全部已同步技能。

## 权限与范围

全部接口要求 `Login`、`RequireRole(admin)` 与 `RequireCorpCapability(menu.dashboard)`。`super_admin` 沿用现有自动放行；`senior_fc` 的方案维护权限不包含跨用户技能查阅权限。admin 可查询全站，不以管理员所属军团限制目标范围。

`corporation_ids` 为空表示全部，否则按角色数据库快照中的当前军团筛选，支持多选并去重。每用户仅统计范围内角色，不扩大到其跨军团角色。军团归属为 0 的角色计入全部查询，但不提供未知军团单独筛选候选。包含禁用但未删除的用户；排除已删除用户、已解绑/删除角色、没有有效所属用户的角色及孤立技能记录。

只接受包含至少一项技能要求的 `plan_scope=corp` 方案。现有军团方案全站共享，没有按军团归属绑定；个人方案不得用于本功能。

## 统计规则

- 技能判断等级为 `max(active_level, trained_level)`，复用现有技能完成度比较逻辑。
- 角色必须满足选定方案全部技能要求。不同角色的技能不得合并。
- 用户至少有一个范围内达标角色，即为达标用户。
- `eve_character_skill` 摘要记录存在代表已经成功同步。无摘要的角色为 `unknown`，计入角色总数和未同步数，不计入达标数。
- 已同步但没有技能明细或缺少要求技能时，缺失技能按 0 级比较，角色为 `unsatisfied`。
- 已有快照即按快照比较，令牌失效不抹掉快照结果；角色响应另提供 `token_invalid`。
- 不自行推算技能训练队列，不为旧快照设置新的过期阈值，不触发技能同步任务。展示 `skill_updated_at` 让调用方说明数据更新时间。

## 接口契约

方法、访问链、输入和输出见[技能达标 OpenAPI](../../../contracts/modules/dashboard-readiness.yaml)。Vue 路径为 `/dashboard/skill-readiness`；React 使用对应的 hash 路由。

- 候选项不分页。军团来自有效绑定角色，方案按 `sort_order ASC, id DESC` 排序。军团名称缺失时页面显示军团 ID；空方案不可用于实际查询。
- 概览统计覆盖完整范围，不受用户列表分页影响；用户按达标角色数降序、用户 ID 升序。缺少昵称时显示用户 ID。空范围返回零统计，越界页返回空列表并保留总统计。
- 用户角色明细不分页，按角色名、角色 ID 升序。无有效用户或无范围内角色时返回空数组。技能明细仅接受仍有效绑定且符合军团范围的角色，技能按 ID 升序。
- 未同步时等级与技能点为 null，缺省达标标志并返回空的全部技能列表，避免把未知呈现为真实 0 级。技能名称复用 SDE 的英文和 Type ID 兜底；内部异常不向调用方泄漏。

## 查询一致性与性能决策

PostgreSQL 每请求使用只读 `REPEATABLE READ` 事务读取方案、范围内绑定、同步摘要与技能，避免后台技能替换事务或解绑造成同一响应内部错配。一次请求的统计与分页共享同一份结果；展开明细是新的请求，可能反映更新后的快照。

统计只批量读取方案涉及的技能，不逐角色查询；角色 ID 每批 500 个。Service 按角色判断后聚合用户，最后分页，保证全范围统计与每用户数量准确。全部技能仅在角色详情按需加载。名称解析与 SDE 展示信息在快照事务结束后处理，避免把外部调用放进数据库事务。

不新增业务表、达标结果缓存、索引迁移或同步任务。统计内存开销随范围内角色及方案相关技能数量增长；查询数据来源仍是已绑定角色，不是 ESI 全军团成员清单。

## 验收场景

用户 A 在 X 有一个达标角色、在 Y 有一个不达标角色；用户 B 在 X 有一个未同步角色：查询 X 时角色 2、用户 2、达标用户 1、达标角色 1、未同步 1；查询 X+Y 时角色变为 3，其他统计不变。A 的范围内角色数从 1 变为 2，达标数仍为 1。

测试覆盖范围过滤、等级口径、未知与真实缺失、跨角色不能拼接达标、分页统计、批量查询次数、无效方案、明细范围、路由角色/capability 门禁与内部异常屏蔽。
