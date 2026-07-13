---
status: active
doc_type: feature
owner: backend
last_reviewed: 2026-05-11
source_of_truth:
  - server/internal/handler/sde.go
  - server/internal/handler/sys_config.go
  - server/internal/service/sde.go
  - server/internal/middleware/apikey.go
  - static/src/api/sde.ts
  - static/src/api/sys-config.ts
---

# SDE 模块

## 当前能力

- 查询当前已导入的 SDE 版本
- 批量查询 type 信息
- 批量查询 ID 到名称映射
- 模糊搜索物品 / 成员名称
- `super_admin` 可通过系统设置页面配置 SDE 参数
- 为 EVE 信息、舰队配置、SRP、自动 SRP 等模块提供名称与静态数据支撑
- **当前已禁用启动时与 cron 自动导入**，仅保留手动触发所需的配置与服务能力

## 入口

### SDE 数据查询
- `GET /api/v1/sde/version`
- `POST /api/v1/sde/types`
- `POST /api/v1/sde/names`
- `POST /api/v1/sde/search`

前端封装位于 `static/src/api/sde.ts`。

### SDE 配置管理（super_admin 权限）
- `GET /api/v1/system/sde-config`
- `PUT /api/v1/system/sde-config`
- `GET /api/v1/system/sde-config/status`
- `POST /api/v1/system/sde-config/check`
- `POST /api/v1/system/sde-config/update`

前端封装位于 `static/src/api/sys-config.ts`。

## 运行时行为

- SDE 版本记录保存在 `sde_versions`
- 当前不会在启动时或 cron 中自动检查并导入 SDE
- **配置来源**：`system_config` 表（键：`sde.api_key`、`sde.proxy`、`sde.download_url`）；缺失时写入系统内置默认值。
- **API Key 用途**：
  - `sde.api_key` 是用于访问上游 GitHub SDE 下载时的标识符
  - 不是用于保护当前应用的 SDE 查询接口
- **代理配置**：
  - 若配置了代理但代理不可达，下载器会自动回退为直连
- **导入目标**：当前业务 PostgreSQL，而非独立的只读 SDE 库
- **自动任务状态**：`server/jobs/sde.go` 中的启动检查与定时任务注册当前均已禁用
- **手动运维能力**：
  - `POST /system/sde-config/check` 仅探测上游最新版本，不触发导入
  - `POST /system/sde-config/update` 执行手动导入（即使当前已是最新版本，也允许强制重导入）
  - `GET /system/sde-config/status` 返回状态快照（当前版本、最新版本、最近检查/更新时间与错误）
  - 状态快照保存在 `system_config`（键：`sde.status`）
  - SDE 查询类错误（如名称/类型查询失败）由 repository 层统一写入状态快照，供基础配置页排障

## 权限边界

- **SDE 查询接口**：公开访问，无需鉴权
- **SDE 配置管理**：需要 `super_admin` 职权
- 语言优先级由 body / header / cookie 决定，最终默认 `en`

## 验证

标准校验命令参见 `docs/standards/testing-and-verification.md`（`Default Commands` 节）。

功能专项回归：`cd static && pnpm test:unit`（SDE 名称解析回归测试）

## 关键不变量

- **配置来源**：只使用数据库与系统内置默认值，不读取 `config.yaml`
- **配置缓存**：使用 SysConfigRepository 的 Redis 缓存机制
- **公开访问**：SDE 查询接口无需鉴权，任何前端调用方都可以使用
- **共享基础能力**：SDE 是共享基础能力，修改返回结构时要检查多个业务模块
- **版本检查与导入**：当前不通过启动任务或 cron 自动执行；如恢复此能力，需要同步更新运行文档与运维预期
- **管理端异常提示口径**：检测到 `has_update=true` 时，管理端显式提示“存在可更新版本”
- **查询错误可见性**：`last_query_error` / `last_query_error_at` / `last_query_error_source` 会记录最近一次 SDE 查询失败上下文
  - 典型来源：`sde_repository.GetTypes`、`sde_repository.GetNames`、`npc_kill.GetSolarSystemNames`
  - repository 层对同一错误签名默认做 60 秒节流，避免高频重复覆盖状态快照
- **英文名称回退**：英文名称缺失时，type/group/category/market group 查询会回退到 SDE 基础名称列
- **`POST /api/v1/sde/names`**：返回 `flat` 与 `names` 两套映射
  - `names` 是按 namespace 分组的权威结果
  - `flat` 仅用于兼容单 namespace 调用方
- **`docs/reference/sde-schema.sql`**：仅是历史参考资产，不代表当前应用的实时 schema

## 主要代码文件

### 后端
- `server/internal/handler/sde.go` - SDE 数据查询接口
- `server/internal/handler/sys_config.go` - SDE 配置管理接口
- `server/internal/service/sde.go` - SDE 业务逻辑（含配置读取、下载、导入）
- `server/internal/middleware/apikey.go` - API Key 鉴权中间件
- `server/internal/repository/sde.go` - SDE 基础查询
- `server/internal/repository/sde_version.go` - 版本管理
- `server/internal/repository/sde_types.go` - 类型查询
- `server/internal/repository/sde_search.go` - 模糊搜索
- `server/internal/repository/sde_ships.go` - 舰船数据
- `server/internal/repository/sys_config.go` - 系统配置存储
- `server/internal/router/router.go` - 路由定义与中间件应用

### 前端
- `static/src/api/sde.ts` - SDE 数据查询 API
- `static/src/api/sys-config.ts` - 系统配置 API
- `static/src/views/system/basic-config/index.vue` - 系统设置页面（含 SDE 配置）
- `static/src/types/api/api.d.ts` - API 类型定义
- `static/src/locales/langs/zh.json` - 中文国际化
- `static/src/locales/langs/en.json` - 英文国际化

## PostgreSQL Bool Mapping Constraint

- mapSolarSystems border, fringe, corridor, hub, international, regional, constellation are flag fields and must map to nullable boolean types in Go model MapSolarSystem.
- Numeric identity fields such as regionID, constellationID, and solarSystemID remain integer types.
- Query paths that only need system names (for example npc_kill.GetSolarSystemNames) should select only solarSystemID and solarSystemName to avoid unrelated scan failures.

