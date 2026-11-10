---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-10
source_of_truth:
  - server/internal/router/router.go
  - server/internal/handler/eve_sso.go
  - server/internal/handler/me.go
  - server/internal/service/eve_sso.go
  - server/internal/service/user.go
  - static/src/api/auth.ts
  - static/src/router/guards/beforeEach.ts
  - static/src/views/auth
  - static/src/views/dashboard/characters
  - static-react/src/api/auth.ts
  - static-react/src/auth/session-bootstrap.tsx
  - static-react/src/auth/profile-lock.ts
  - static-react/src/pages/dashboard-characters-page.tsx
---

# 认证与人物绑定

## 当前能力

- 通过 EVE SSO 登录
- 首次 EVE 授权后先填写昵称及 QQ / Discord，提交成功才创建用户
- 一个用户绑定多个人物
- 设置主人物
- 解绑人物
- 通过 `/api/v1/me` 获取当前用户、职权、权限与绑定人物信息
- 通过 `/api/v1/me` 维护昵称、QQ、Discord ID 资料
- 当用户仅剩 1 个绑定人物且该人物为主人物时，可在人物管理页面发起自助注销账号
- 未填写昵称或未提供 QQ / Discord 任一联系方式时，前端强制停留在 `/dashboard/characters`，且尝试访问其他页面时弹出原因提示
- 可选启用：任一已绑定人物 ESI 失效时，前端强制停留在 `/dashboard/characters`，且尝试访问其他页面时弹出原因提示
- 主人物 ESI 已失效时，`/api/v1/me` 仍返回启动上下文，前端强制停留在 `/dashboard/characters` 直到主人物重新授权，且尝试访问其他页面时弹出原因提示

## 入口

### 前端

- `/auth/login`
- `/auth/callback`

### 后端

- `GET /api/v1/sso/eve/login`
- `GET /api/v1/sso/eve/callback`
- `GET /api/v1/sso/eve/scopes`
- `GET /api/v1/sso/eve/registration`
- `POST /api/v1/sso/eve/registration`
- `GET /api/v1/sso/eve/characters`
- `GET /api/v1/sso/eve/bind`
- `PUT /api/v1/sso/eve/primary/:character_id`
- `DELETE /api/v1/sso/eve/characters/:character_id`
- `GET /api/v1/me`
- `DELETE /api/v1/me`
- `PUT /api/v1/me`

## 权限边界

- 登录入口与回调是 `Public`
- `/api/v1/me` 与人物绑定相关接口要求有效 `JWT`，允许 `guest` 使用
- `/api/v1/me` 的 `DELETE` 入口仅允许删除当前 JWT 对应的用户，不接受外部 `user_id`，用于人物管理页中的自助注销
- `guest` 通过这些接口完成权限上下文建立、人物绑定与资料补全，再决定是否能进入 `Login` 边界的业务页面
- `/api/v1/me` 会返回主人物 `token_invalid` 状态，供前端将用户锁定在 `/dashboard/characters` 直到主人物重新授权
- `/api/v1/me` 同时返回 `enforce_character_esi_restriction`，供前端路由守卫决定是否对非主人物失效 ESI 启用页面停留限制
- 首次 SSO 登录时，若主人物所属军团在 `allow_corporations` 内，后端会直接赋予 `user`；该列表运行时始终包含代码常量中的伏羲军团 Fuxi Legion（`98185110`）
- 首次 SSO 登录时，若主人物 ID 在 `config.yaml` 的 `app.super_admins` 列表中，后端会直接赋予 `super_admin`
- 每次 SSO 登录时，`SyncConfigSuperAdmins` 会根据配置文件自动同步 `super_admin` 职权
- 职权和权限的最终决策在后端完成，前端只消费结果

## 关键不变量

- 当前受支持的产品登录方式是 EVE SSO
- `register` 页面源码仍存在，但不是当前产品规范；`forget-password` 页面已移除
- `/api/v1/me` 是前端启动权限上下文的关键接口；`GET` 必须返回 `profile_complete`，由后端按昵称和至少一种联系方式计算
- `/api/v1/me` 不是"非 guest 才可访问"的业务接口，而是登录后立即可用的自助上下文接口
- `/api/v1/me` 的自助注销能力只针对当前登录用户，不等同于 `/api/v1/system/user/:id` 管理端删除
- 新用户必须在登录回调页完成注册后才能取得应用 JWT；现有账号仍保留资料完成状态与前端资料锁
- React 在后端滚动部署期间若暂时未收到 `profile_complete`，会按昵称与 QQ / Discord ID 重新计算，避免持久化会话误判资料未完成
- QQ / Discord ID 的默认管理入口是 `/api/v1/me`；管理员侧 `/api/v1/system/user/:id` 仅允许 `super_admin` 维护非 `super_admin` 用户的联系方式
- 当前登录后若系统配置 `auth.enforce_character_esi_restriction = true`，则还必须保证所有已绑定人物的 ESI 有效，才允许离开 `/dashboard/characters`
- 无论系统配置是否开启，主人物 ESI 已失效都会强制前端停留在 `/dashboard/characters`，直到主人物重新授权；不会自动退出登录
- 用户仍被锁定在 `/dashboard/characters` 时，尝试导航到其他页面会弹出警告消息框，说明资料未完成、主人物 ESI 失效或其他绑定人物 ESI 失效等原因
- 重新绑定已存在人物会沿用当前 SSO 回调流程刷新该人物 token 并清除 `token_invalid`
- 人物 `scopes` 列是“历史授予并集 ∩ 当前注册 scope 集”的快照：SSO 回调按并集合并保存（含此前单独授予的可选 scope），后台 token 刷新不再覆盖该字段
- 管理员专属可选 scope（军团击杀邮件）在归属用户失去管理员职权后的下一次 SSO 回调写入时被剥离；普通可选 scope 不受职权影响
- SSO 回调对已有人物的读-改-写与后台 token 刷新共用 per-character 互斥锁，防止轮换式 refresh token 被旧值覆盖；生日等旁路字段只做目标列更新，不做整行覆盖
- 已有人物重新登录时，若历史授权含当前 token 链缺失的非管理员专属可选 scope，后端自动 302 引导到 EVE 授权页补授一次（state 携带 `scope_refresh_attempted` 标记防循环）；补授链仍缺 scope 时正常颁发 JWT
- QQ / Discord ID 的唯一性由后端和数据库索引共同校验，包括软删除账号占用的联系方式；本人可保留原值
- QQ / Discord 已被其他账号占用时，Vue 与 React 显示本地化弹窗，使用占用账号的主人物名字，提示先登录该人物并点击“绑定新人物”添加当前人物；不自动转移或合并账号
- 本人及管理员更新资料均不可清空最后一个联系方式；数据库 CHECK 同时拒绝空字符串、NULL 和纯空白组合
- 职权编码与权限列表必须与后端返回保持一致，不做前端别名映射
- 所有 EVE SSO 端点（授权、令牌、图片服务等）通过 `server/config/config.go` 中的 `EveSSOConfig` 配置管理，禁止硬编码 URL

## 主要代码文件

- `server/internal/handler/eve_sso.go`
- `server/internal/handler/me.go`
- `server/internal/service/eve_sso.go`
- `server/internal/service/user.go`
- `server/internal/service/registration.go`
- `server/internal/repository/registration.go`
- `server/bootstrap/user_contact_migration.go`
- `server/internal/router/router.go`
- `static/src/api/auth.ts`
- `static/src/views/auth`
- `static/src/views/dashboard/characters`

## 注册状态与部署迁移

- 未注册或原账号已删除的人物完成 SSO 后，仅在 Redis 保存待注册状态（包含 EVE 授权），有效期 15 分钟。应用 JWT、user、eve_character、user_role 均在验证联系方式后才创建；三类数据库记录在同一事务提交。
- 注册票据是 256 位随机不透明值，只能通过 `X-Registration-Token` 使用注册接口。Vue / React 进入回调页后将票据移出地址栏，暂存 sessionStorage 以支持刷新；完成注册或返回登录时移除。票据过期、Redis 不可用或 state 无效时要求重新登录，不降级创建无联系方式用户。
- Vue / React 注册页使用居中的人物资料卡片，提供中英文切换，切换语言保留表单和联系方式选择。昵称默认留空，去除首尾空白后须为 2–20 个字符，前后端均校验；昵称与当前联系方式标签使用红色星号表示必填。QQ / Discord 单选切换，默认 QQ；只显示并提交当前选择的联系方式。切换时保留已输入内容，当前选项为空时禁止注册，不使用隐藏选项的值代替。
- 注册提交使用 Redis 票据锁及 PostgreSQL 人物事务锁；不同票据不能重复创建同一个人物。联系方式竞争由唯一索引兜底，失败事务不留下半个账号，冲突票据仍可重试。
- 部署启动在 AutoMigrate 后、开始服务请求前运行 `20261110_require_user_contact`。迁移事务删除所有 QQ 与 Discord 均为空白的 user（包含软删除账号），以及所属人物授权、user_role、Mumble 身份和明确列出的个人 ESI 表记录。订单、账务、补损、福利、工单、PAP 和审计等业务历史保留。
- 迁移建立 `chk_user_required_contact`、QQ / Discord 非空值唯一索引，成功后在 `schema_migration` 记版本；重复启动不再清理。若现存 Discord 重复或其他步骤失败，整个迁移回滚并阻止启动，须先修复冲突。
- JWT 中间件在读取角色缓存前核实用户仍存在且启用，因此被迁移删除的账号即使持有旧 JWT / 角色缓存也无法认证。
- 部署前备份 PostgreSQL。清理是物理删除；回滚须配合恢复部署前数据库备份及旧应用。被清理人物也可重新 SSO 授权，填写联系方式后注册。

## 前端实现映射（迁移期）

- 人物资料、绑定、主人物切换、ESI 锁定和自助注销是双端必须保持一致的行为，不限定为“旧 Vue 页面”。
- Vue 实现位于 `static/src`；React 实现位于 `static-react/src/api/auth.ts`、启动 session bootstrap、资料锁 helper、人物页及对应路由。
- React 已注册顶层 JWT `/characters`，并将其作为资料或 ESI 锁定的自动落点；锁定原因由人物页本地化展示。冻结范围内已有的 `/dashboard/characters` 复用同一修复页。
