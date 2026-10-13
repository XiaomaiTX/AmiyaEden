---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-10-12
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

完整后端路由及权限见 [API 路由索引](../../api/route-index.md)。前缀为 `/api/v1/dashboard/skill-readiness`。Vue 路径为 `/dashboard/skill-readiness`；React 使用其 hash 路由对应页面。

所有 POST 请求接受：

```json
{"corporation_ids": [10, 20], "plan_id": 1, "page": 1, "page_size": 20, "language": "zh"}
```

`plan_id` 必填正整数；军团 ID 必须为正整数。分页省略或 0 使用默认值（1、20），负值或 `page_size>100` 拒绝。语言仅支持 `zh`、`en`，默认 `zh`。明细接口接受同一范围请求，分页字段不影响明细。

- `GET /options`：`corporations[{corporation_id, corporation_name}]` 与 `plans[{id, title, description, ship_type_id}]`，完整候选而非分页切片。军团取有效用户绑定角色的正数军团 ID；方案按 `sort_order ASC, id DESC` 排序。名称解析失败返回空名称，调用方显示军团 ID。候选中的空方案会在执行查询时被拒绝。
- `POST /query`：返回 `summary`、`list`、`total`、`page`、`pageSize`。summary 为 `character_count`、`user_count`、`satisfied_user_count`、`satisfied_character_count`、`unknown_character_count`；不受分页影响。list 行为 `user_id`、`nickname`、`character_count`、`satisfied_character_count`、`unknown_character_count`，按达标角色数降序、用户 ID 升序。`total` 是用户行数。没有昵称时返回空字符串，调用方显示用户 ID。
- `POST /users/:user_id/characters`：返回范围内角色数组，包含 `user_id`、`character_id`、`character_name`、`corporation_id`、`corporation_name`、`status`、`matched_skills`、`total_skills`、`skill_updated_at`、`token_invalid`。按角色名、角色 ID 升序。无有效用户或没有范围内角色时返回空数组。
- `POST /characters/:character_id/skills`：返回 `character`、`requirements`、`skills`。必须仍为有效绑定且符合军团范围的角色。requirements 包含全部方案技能，skills 包含全部已同步技能，按技能 ID 升序。

技能行：`skill_type_id`、`skill_name`、`group_name`、`active_level`、`trained_level`、`skillpoints_in_skill`、`current_level`。要求技能行额外包含 `required_level`、`satisfied`。未同步时等级、技能点为 null，`satisfied` 不返回，全部技能为空数组，避免把未知误呈现为真实 0 级。技能名称复用技能规划的 SDE 查询及既有英文/Type ID 兜底。

外层使用统一 `{code,msg,data}`。参数错误为 code 400；无效方案、越界角色及内部读取失败为 code 500；权限/认证使用 HTTP 403/401。内部异常不向调用方泄漏。空范围正常返回零统计、空列表；越界页返回空列表而保留总统计。

## 查询一致性与性能决策

PostgreSQL 每请求使用只读 `REPEATABLE READ` 事务读取方案、范围内绑定、同步摘要与技能，避免后台技能替换事务或解绑造成同一响应内部错配。一次请求的统计与分页共享同一份结果；展开明细是新的请求，可能反映更新后的快照。

统计只批量读取方案涉及的技能，不逐角色查询；角色 ID 每批 500 个。Service 按角色判断后聚合用户，最后分页，保证全范围统计与每用户数量准确。全部技能仅在角色详情按需加载。名称解析与 SDE 展示信息在快照事务结束后处理，避免把外部调用放进数据库事务。

不新增业务表、达标结果缓存、索引迁移或同步任务。统计内存开销随范围内角色及方案相关技能数量增长；查询数据来源仍是已绑定角色，不是 ESI 全军团成员清单。

## 验收场景

用户 A 在 X 有一个达标角色、在 Y 有一个不达标角色；用户 B 在 X 有一个未同步角色：查询 X 时角色 2、用户 2、达标用户 1、达标角色 1、未同步 1；查询 X+Y 时角色变为 3，其他统计不变。A 的范围内角色数从 1 变为 2，达标数仍为 1。

测试覆盖范围过滤、等级口径、未知与真实缺失、跨角色不能拼接达标、分页统计、批量查询次数、无效方案、明细范围、路由角色/capability 门禁与内部异常屏蔽。
