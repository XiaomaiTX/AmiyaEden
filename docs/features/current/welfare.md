---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-03-22
source_of_truth:
  - server/internal/router/router.go
  - server/internal/service/welfare.go
  - static/src/api/welfare.ts
  - static/src/views/welfare
---

# 军团福利模块

## 当前能力

- 管理员福利定义 CRUD（创建、编辑、删除、列表）
- 两种发放模式：按自然人（per_user）、按人物（per_character）
- 可选技能计划检查：关联军团技能计划，技能合格才允许发放
- 发放记录表（welfare_distribution）追踪 character_id、user_id、QQ、discord_id
- 按自然人发放时按 user_id 去重，按人物发放时按 character_id 去重
- 福利存在发放记录时禁止删除

## 入口

### 前端页面

- `static/src/views/welfare/my` — 我的福利（所有已登录用户）
- `static/src/views/welfare/approval` — 福利审批（福利官、管理员）
- `static/src/views/welfare/settings` — 福利设置（管理员）

### 后端路由

- `/api/v1/system/welfare/list`
- `/api/v1/system/welfare/add`
- `/api/v1/system/welfare/edit`
- `/api/v1/system/welfare/delete`

## 权限边界

- 军团福利导航栏要求 `Login`（guest 不可见）
- 我的福利页面要求 `Login`
- 福利审批页面要求 `welfare` 或 `admin` 或 `super_admin`
- 福利设置页面及后端 `/system/welfare/*` 接口要求 `admin`
- `welfare` 角色（福利官）为系统默认角色，优先级 50

## 关键不变量

- 不论按自然人还是按人物发放，都不允许重复发放
- 福利系统是纯记录型，实际发放在外部完成（游戏内合同等），系统只追踪分发
- 技能计划检查复用 skill_plan 模块，福利定义通过 skill_plan_id 关联
- 我的福利、福利审批页面当前为占位，待后续实现

## 主要代码文件

- `server/internal/model/welfare.go`
- `server/internal/repository/welfare.go`
- `server/internal/service/welfare.go`
- `server/internal/handler/welfare.go`
- `server/internal/router/router.go`
- `static/src/api/welfare.ts`
- `static/src/views/welfare`
