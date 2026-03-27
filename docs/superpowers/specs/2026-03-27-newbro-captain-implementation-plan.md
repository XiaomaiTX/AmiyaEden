---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2026-03-27
---

# 新人帮扶实施计划

## 输入规格

本计划基于以下已确认设计文档：

- [新人帮扶设计](/home/merlin01/projects/AmiyaE/docs/superpowers/specs/2026-03-27-newbro-captain-design.md)

## 目标

按仓库既有分层与契约同步规则，落地 `新人帮扶` 模块，包含：

- 新增 `captain` 角色与 `新人帮扶` 菜单
- 新人资格快照表与帮扶关系历史表
- 队长收益归因台账与同步入口
- 新人选队长页、队长视图、管理员视图
- 后端/前端/文档/测试同步完成

## 总体顺序

按以下顺序实施，避免后续返工：

1. 数据模型与资格/归因核心服务
2. 角色、菜单、路由与最小后端接口
3. 归因同步与管理入口
4. 前端 API、类型、页面
5. 文档同步与全量验证

## Phase 1: 数据模型与核心领域服务

### 1.1 新增模型

新增文件建议：

- `server/internal/model/newbro_player_state.go`
- `server/internal/model/newbro_captain_affiliation.go`
- `server/internal/model/captain_bounty_attribution.go`
- `server/internal/model/captain_bounty_sync_state.go`

模型要求：

- `newbro_player_state`
  - `user_id` 唯一
  - `is_currently_newbro`
  - `evaluated_at`
  - `rule_version`
  - `disqualified_reason`
- `newbro_captain_affiliation`
  - 支持单玩家单时刻仅一条有效关系
  - `player_primary_character_id_at_start`
- `captain_bounty_attribution`
  - `wallet_journal_id` 唯一
  - 包含 `captain_wallet_journal_id`
  - 包含 `captain_character_id`
- `captain_bounty_sync_state`
  - 支持按同步键维护增量进度

### 1.2 注册迁移

修改：

- `server/bootstrap/db.go`

任务：

- 将新增模型加入 `AutoMigrate`
- 保持现有迁移顺序风格
- 不额外引入 SQL migration 目录

### 1.3 资格规则服务

新增文件建议：

- `server/internal/service/newbro_eligibility.go`
- `server/internal/repository/newbro_player_state.go`

实现目标：

- 计算用户是否是当前规则下的新人
- 读取用户全部绑定角色
- 检查是否存在任一角色 `total_sp >= 20_000_000`
- 统计 `allow_corporations` 内的绑定角色数量是否达到 `4`
- 生成并保存 `newbro_player_state`
- 支持基于 `rule_version` 的缓存失效与重算

`rule_version` 建议：

- 第一版直接用字符串版本号，由服务层统一生成
- 组成建议：资格规则关键参数的稳定串联值
  - 例如 `sp:<threshold>;allowcorp-count:<threshold>`
- 后续若规则进入系统配置，也沿用同一生成逻辑

### 1.4 帮扶关系服务

新增文件建议：

- `server/internal/service/newbro_affiliation.go`
- `server/internal/repository/newbro_captain_affiliation.go`

实现目标：

- 查询当前有效关系
- 查询最近 `10` 条关系历史
- 选择/切换队长
- 确保同一玩家同一时刻只存在一条有效关系
- 校验目标用户具备 `captain` 角色
- 优先使用资格快照，必要时触发资格重算

### 1.5 队长候选查询服务

实现目标：

- 返回当前可选队长列表
- 队长展示口径使用其当前主角色
- 返回队长基础信息与当前活跃新人数量

## Phase 2: RBAC、菜单、路由与后端接口骨架

### 2.1 新增角色

修改：

- `server/internal/model/role.go`
- 相关角色展示位置

任务：

- 新增 `RoleCaptain = "captain"`
- 将 `captain` 加入系统角色种子
- 确认排序位置
- 后续前端角色文案同步增加 `captain`

### 2.2 菜单与默认权限

修改：

- `server/internal/model/menu.go`

任务：

- 新增根菜单 `新人帮扶`
- 新增子菜单：
  - `新人选队长`
  - `队长帮扶`
  - `帮扶管理`
- 为 `captain` 提供队长页访问菜单
- 为 `admin` 提供管理页访问菜单
- 普通登录用户保留新人页菜单入口，由 service 决定当前是否可使用

### 2.3 路由与 Handler 骨架

新增文件建议：

- `server/internal/handler/newbro_user.go`
- `server/internal/handler/newbro_captain.go`
- `server/internal/handler/newbro_admin.go`

修改：

- `server/internal/router/router.go`

路由分组：

- 用户侧：`login.Group("/newbro")`
- 队长侧：`login.Group("/newbro/captain", middleware.RequireRole(model.RoleCaptain))`
- 管理侧：`admin.Group("/newbro")`

首批路由：

- `GET /api/v1/newbro/captains`
- `GET /api/v1/newbro/affiliation/me`
- `POST /api/v1/newbro/affiliation/select`
- `GET /api/v1/newbro/captain/overview`
- `GET /api/v1/newbro/captain/players`
- `GET /api/v1/newbro/captain/attributions`
- `GET /api/v1/system/newbro/captains`
- `GET /api/v1/system/newbro/captains/:user_id`
- `POST /api/v1/system/newbro/attribution/sync`

### 2.4 最小接口契约先落地

要求：

- 先定义 handler request/response struct
- 再落 service 返回结构
- 后续前端类型直接对齐这些字段

特别约束：

- `GET /api/v1/system/newbro/captains`
  - 默认排序固定为：
    - `attributed_bounty_total DESC`
    - `captain_user_id ASC`
- `GET /api/v1/system/newbro/captains/:user_id`
  - 第一版不返回无限嵌套大列表
  - 推荐返回：
    - `overview`
    - `players_query`
    - `attributions_query`
  - 具体列表继续通过现有列表接口按筛选获取

## Phase 3: 归因台账与同步

### 3.1 仓储查询

新增/扩展仓储建议：

- `server/internal/repository/captain_bounty_attribution.go`
- 复用并扩展：
  - `server/internal/repository/npc_kill.go`
  - `server/internal/repository/eve_character.go`
  - `server/internal/repository/eve_skill.go`
  - 视情况新增钱包流水专用查询仓储

查询能力：

- 最近 1 个月、尚未归因的玩家钱包流水
- 角色到用户的映射
- 某时刻有效的帮扶关系
- 队长当前主角色
- 队长候选钱包流水查询

### 3.2 归因算法

新增文件建议：

- `server/internal/service/captain_bounty_sync.go`

算法要求：

- 玩家侧只处理 `bounty_prizes`
- 只处理最近 `1` 个月记录
- 只对“当前资格快照仍为新人”的用户继续归因
- 匹配条件：
  - 同 `system_id`
  - `reason` 解析出的 NPC ID 与数量一致
  - 时间差 `<= 15 min`
- 多候选时固定 tie-breaker：
  - 绝对时间差最小
  - 再按原始钱包流水 `date ASC`
  - 再按 `id ASC`
- 写入 `captain_wallet_journal_id`
- 对同一 `wallet_journal_id` 幂等

### 3.3 首次上线补算

实现要求：

- 支持对当前仍是新人的用户补算历史未归因记录
- 补算范围仅限最近 `1` 个月
- 当前已不是新人的用户不补算
- 已入账归因记录永不重算删除

### 3.4 管理入口

管理接口目标：

- 手动触发同步
- 返回：
  - `processed_count`
  - `inserted_count`
  - `skipped_count`
  - `last_wallet_journal_id`

第一版不要求定时任务，但服务实现要可复用，后续可接入 jobs。

## Phase 4: 队长/管理报表服务

### 4.1 队长视角查询

新增文件建议：

- `server/internal/service/newbro_captain_report.go`
- `server/internal/repository/newbro_captain_report.go`

能力：

- `overview`
  - 当前队长基本信息
  - `active_player_count`
  - `historical_player_count`
  - `attributed_bounty_total`
  - `attribution_record_count`
- `players`
  - 分页
  - `active | historical | all` 筛选
  - 每个玩家带收益汇总
- `attributions`
  - 分页
  - 支持 `player_user_id`、`ref_type`、时间范围过滤

### 4.2 管理视角查询

能力：

- 全部队长绩效分页列表
- 单个队长详情概览
- 队长详情页复用队长侧列表接口的查询参数模型，避免单接口塞入两个大数组

## Phase 5: 前端 API、类型与页面

### 5.1 API 与类型

新增文件建议：

- `static/src/api/newbro.ts`

修改：

- `static/src/types/api/api.d.ts`

要求：

- 按后端接口顺序同步新增命名空间与类型
- 所有新页面只通过 API wrapper 调后端

### 5.2 路由与页面

新增页面建议：

- `static/src/views/newbro/select-captain/index.vue`
- `static/src/views/newbro/captain/index.vue`
- `static/src/views/newbro/manage/index.vue`

如项目当前仍维护静态路由模式，还需同步：

- `static/src/router/modules/*.ts`

页面目标：

- 新人选队长页
  - 展示资格状态
  - 当前队长
  - 候选队长列表
  - 切换动作
- 队长帮扶页
  - 总览卡片
  - 玩家列表
  - 收益归因列表
- 帮扶管理页
  - 全部队长绩效表
  - 单队长下钻
  - 手动同步按钮

### 5.3 权限与可见性

前端只做 UX 级控制：

- 新人页：登录用户可见菜单后，页面仍需根据接口返回状态决定可用性
- 队长页：按 `roles.includes('captain')` 做前端展示控制
- 管理页：按 `admin`

### 5.4 本地化

修改：

- `static/src/locales/langs/zh.json`
- `static/src/locales/langs/en.json`

要求：

- 所有用户可见文本双语同步
- 增加 `captain` 角色名称文案
- 增加 `新人帮扶` 菜单与页面文案

## Phase 6: 文档同步

必须更新：

- `docs/architecture/database-schema.md`
  - 新增业务表说明
- `docs/api/route-index.md`
  - 新增 `newbro` 路由
- 视情况新增：
  - `docs/features/current/newbro-support.md`

若新增 feature doc，内容至少覆盖：

- 新人资格规则
- 关系历史规则
- 归因台账规则
- 权限边界
- 1 个月回溯窗口

## Phase 7: 测试与验证

### 7.1 后端单测优先级

新增测试建议：

- `server/internal/service/newbro_eligibility_test.go`
- `server/internal/service/newbro_affiliation_test.go`
- `server/internal/service/captain_bounty_sync_test.go`

关键用例：

- 任一角色达到 `20,000,000 SP` 时不再属于新人
- `allow_corporations` 内角色数达到 `4` 时不再属于新人
- `rule_version` 变化触发资格重算
- 玩家切换队长只保留一条有效关系
- 重复选择相同队长不重复创建关系
- 归因只处理最近 `1` 个月
- 当前非新人用户不补算
- 匹配 tie-breaker 稳定
- 同一玩家流水不重复入账

### 7.2 前端验证重点

- 类型检查通过
- 页面能消费最小接口契约
- 表格分页/筛选与默认排序符合设计
- 队长与管理视图切换正常

### 7.3 执行验证

- `cd server && go test ./...`
- `cd server && go build ./...`
- `cd static && pnpm lint .`
- `cd static && pnpm exec vue-tsc --noEmit`

## 推荐执行切片

若按最小风险拆分实现，建议分 4 个代码切片：

1. 后端表结构、角色、菜单、资格服务
2. 后端关系接口与队长/管理查询接口
3. 归因同步与管理触发
4. 前端页面、文档、验证

## 风险与注意事项

- `captain` 是新系统角色，注意用户管理页与角色展示文案同步
- 新人页对普通登录用户可见，但 service 才是最终资格裁决点
- 归因算法必须稳定且幂等，否则后续奖金结算会漂移
- 不要在第一版里把管理详情接口做成超大嵌套响应，优先复用分页子查询
- 文档必须明确“当前资格快照可重算”和“只回溯 1 个月”
