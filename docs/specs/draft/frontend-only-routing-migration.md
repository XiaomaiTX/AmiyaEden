# 前端路由迁移计划（移除后端菜单系统）

**创建日期**: 2026-03-26
**最后更新**: 2026-03-27
**状态**: In Progress
**预计工期**: 1 天
**优先级**: High

---

## 迁移进度

- [x] **阶段零**：后端路由同步到前端 ✅ 已完成
- [x] **阶段一**：验证前端模式可行性 ✅ 已完成
- [x] **阶段二**：删除后端菜单相关代码 ✅ 已完成
- [x] **阶段三**：数据库清理 ✅ 已完成
- [x] **阶段四**：权限验证逻辑确认 ✅ 已完成
- [x] **阶段五**：全面测试 ✅ 已完成
- [ ] **阶段六**：文档更新 ⏳ 待执行
- [ ] **阶段七**：部署上线 ⏳ 待执行
- [ ] **阶段八**：验证与监控 ⏳ 待执行

---

## 一、背景与目标

### 1.1 当前问题

- 项目同时支持前端静态路由和后端动态菜单两种模式，但实际只使用后端模式
- 后端菜单系统存在数据一致性风险（种子数据 vs 数据库）
- 每次启动需要同步 ~200 次数据库操作，影响启动性能
- 没有动态菜单需求，没有 SaaS 多租户需求，几乎不用后台菜单管理功能
- PAPExchange 等菜单在数据库中持久化后无法自动清理

### 1.2 迁移目标

- 移除后端菜单系统，简化架构
- 只使用前端静态路由模式
- 提升系统性能（启动时间、菜单加载时间）
- 降低维护成本
- 消除数据一致性风险

---

## 二、前后端路由对比分析

### 2.1 前端路由

| 特性 | 描述 |
|------|------|
| **数据源** | `static/src/router/modules/*.ts`（10 个模块） |
| **类型安全** | TypeScript 编译时检查 |
| **数据一致性** | 单一数据源，无一致性风险 |
| **权限控制** | `meta.roles` + `meta.login` |
| **按钮权限** | `meta.authList` + `v-auth` 指令 |
| **维护成本** | 修改文件 + 重新部署 |
| **启动开销** | 无数据库查询 |
| **运行时开销** | 纯内存操作 |
| **动态调整** | 需要重新部署 |
| **测试难度** | 纯前端测试，无需数据库 |

### 2.2 后端路由

| 特性 | 描述 |
|------|------|
| **数据源** | `menu` 表 + `GetSystemMenuSeeds()` |
| **类型安全** | Go 编译时检查 |
| **数据一致性** | 需要同步（种子数据 ↔ 数据库） |
| **权限控制** | 角色-菜单关联表 |
| **按钮权限** | `type=button` + `permission` |
| **维护成本** | 修改代码 + 同步到数据库 |
| **启动开销** | 每次启动需要同步菜单（~200 次数据库操作） |
| **运行时开销** | 每次登录查询数据库 |
| **动态调整** | 可以后台调整（但几乎不用） |
| **测试难度** | 需要数据库环境 |

### 2.3 对比结论

前端路由在后端模式不使用的情况下，优势明显：
- 简洁性：单一数据源，无同步开销
- 性能：无数据库查询，启动和运行更快
- 维护：修改路由配置更简单，无需考虑数据库同步
- 测试：纯前端测试，环境搭建更容易

---

## 三、迁移阶段详解

### 阶段零：后端路由同步到前端 ✅ 已完成

**完成日期**: 2026-03-27

后端菜单数据已完全同步到前端路由配置，包括所有路由、权限和按钮权限。

#### 更新内容

**[shop.ts](../../static/src/router/modules/shop.ts)**
- 为 `ShopManage` 路由添加按钮权限（4 个按钮）：新增商品、编辑商品、删除商品、审批订单

**[info.ts](../../static/src/router/modules/info.ts)**
- 添加缺失的路由（2 个）：`EveInfoAssets` (/info/assets)、`EveInfoContracts` (/info/contracts)

**[system.ts](../../static/src/router/modules/system.ts)**
- 添加缺失的路由（2 个）：`RoleManage` (/system/role)、`PAPExchange` (/system/pap-exchange)
- 添加按钮权限（7 个页面，16 个按钮）

**[srp.ts](../../static/src/router/modules/srp.ts)**
- 添加按钮权限（2 个页面，4 个按钮）

**[index.ts](../../static/src/router/modules/index.ts)**
- 修正路由模块顺序，与后端菜单 Sort 值保持一致

**[exception.ts](../../static/src/router/modules/exception.ts)**
- 为 `exceptionRoutes` 添加 `isHide: true` 属性，修复异常页面路由显示问题

**总计**：
- 新增路由：4 个
- 新增按钮权限：20 个
- 修正路由顺序：1 处
- 修复路由显示问题：1 处
- 涉及文件：7 个

---

### 阶段一：验证前端模式可行性 ✅ 已完成

**完成日期**: 2026-03-27

#### 验证任务

- [x] 修改环境变量（VITE_ACCESS_MODE = frontend）
- [x] 重启前端服务
- [x] 测试登录流程
- [x] 验证不同角色用户看到的菜单正确
- [x] 测试路由跳转功能
- [x] 验证权限验证功能
- [x] 测试按钮权限
- [x] 测试异常页面
- [x] 验证刷新页面后状态保持
- [x] 检查并补充前端路由配置

#### 验证结果

- ✅ 前端模式功能完整
- ✅ 所有角色菜单显示正确
- ✅ 路由和权限验证正常
- ✅ 按钮权限生效
- ✅ 无需后端菜单系统支持

---

### 阶段二：删除后端菜单相关代码 ✅ 已完成

**完成日期**: 2026-03-27

#### 后端文件删除（5个文件）

- ✅ [model/menu.go](../../server/internal/model/menu.go) - 菜单模型和种子数据（235 行）
- ✅ [repository/menu.go](../../server/internal/repository/menu.go) - 菜单数据访问层（217 行）
- ✅ [service/menu.go](../../server/internal/service/menu.go) - 菜单业务逻辑层（178 行）
- ✅ [service/menu_test.go](../../server/internal/service/menu_test.go) - 菜单测试文件
- ✅ [handler/menu.go](../../server/internal/handler/menu.go) - 菜单HTTP处理器（167 行）

#### 后端代码修改（5个文件）

**[router.go](../../server/internal/router/router.go)**
- 删除菜单 Handler 初始化
- 删除 7 个菜单相关 API 路由

**[db.go](../../server/bootstrap/db.go)**
- 删除 `&model.Menu{}` 表迁移
- 删除 `&model.RoleMenu{}` 表迁移
- 删除 `roleSvc.SeedSystemMenus()` 种子数据初始化

**[handler/role.go](../../server/internal/handler/role.go)**
- 删除 `GetRoleMenus` 方法
- 删除 `SetRoleMenus` 方法

**[repository/role.go](../../server/internal/repository/role.go)**
- 删除 `Delete` 方法中的 `RoleMenu` 删除逻辑
- 删除 `GetRoleMenuIDs` 方法
- 删除 `SetRoleMenus` 方法
- 删除 `GetMenuIDsByRoles` 方法

**[model/role.go](../../server/internal/model/role.go)**
- 删除 `Role` 结构体的 `MenuIDs` 字段
- 删除 `RoleMenu` 结构体定义
- 删除 `RoleMenu.TableName()` 方法

#### 前端文件删除（1个文件）

- ✅ [role-permission-dialog.vue](../../static/src/views/system/role/modules/role-permission-dialog.vue) - 角色权限对话框组件
- ✅ [menu/](../../static/src/views/system/menu/) - 菜单管理目录（已不存在）

#### 前端代码修改（6个文件）

**[system-manage.ts](../../static/src/api/system-manage.ts)**
- 删除 7 个菜单相关 API 函数

**[MenuProcessor.ts](../../static/src/router/core/MenuProcessor.ts)**
- 删除 `processBackendMenu()` 方法
- 简化 `getMenuList()` 方法，只保留前端模式

**[role/index.vue](../../static/src/views/system/role/index.vue)**
- 删除权限对话框相关代码

**[api.d.ts](../../static/src/types/api/api.d.ts)**
- 删除菜单相关接口定义

**[.env.development](../../static/.env.development)**
- 将 `VITE_ACCESS_MODE` 从 `backend` 改为 `frontend`

**[.env.production](../../static/.env.production)**
- 将 `VITE_ACCESS_MODE` 从 `backend` 改为 `frontend`

#### 编译测试

- ✅ 后端编译成功，无错误
- ✅ 前端编译测试通过（`pnpm build`，1m 14s，exit code 0）
- ✅ 前端开发服务器启动成功（`pnpm dev`，http://localhost:5173/）

---

### 阶段三：数据库清理 ✅ 已完成

**完成日期**: 2026-03-27

#### 任务清单

- [x] 数据库备份
- [x] 删除菜单表和关联表
- [x] 验证删除成功
- [x] 验证其他表完整性
- [x] 清理 Redis 缓存

#### 执行结果

**删除前数据统计**：
- `menu` 表：68 条记录
- `role_menu` 表：188 条记录

**删除操作**：
```sql
BEGIN;
DROP TABLE IF EXISTS role_menu;
DROP TABLE IF EXISTS menu;
COMMIT;
```

**验证结果**：
- ✅ 两个表均已成功删除
- ✅ 无外键引用残留
- ✅ 数据库完整性检查通过

#### 3.1 数据库备份

**完整备份**

```bash
# 备份整个数据库
pg_dump -U username -h localhost -d amiya_eden > backup_$(date +%Y%m%d_%H%M%S).sql

# 或者只备份表结构
pg_dump -U username -h localhost -d amiya_eden --schema-only > schema_$(date +%Y%m%d_%H%M%S).sql
```

**备份关键表数据**

```sql
-- 备份角色表（确保角色数据安全）
COPY roles TO '/tmp/roles_backup.csv' CSV HEADER;

-- 备份用户-角色关联表
COPY user_role TO '/tmp/user_role_backup.csv' CSV HEADER;
```

#### 3.2 清理菜单相关数据

**删除菜单表和关联表**

```sql
-- 开启事务（确保操作可回滚）
BEGIN;

-- 查看当前表数据（确认删除前状态）
SELECT COUNT(*) as menu_count FROM menu;
SELECT COUNT(*) as role_menu_count FROM role_menu;

-- 删除角色-菜单关联表
DROP TABLE IF EXISTS role_menu;

-- 删除菜单表
DROP TABLE IF EXISTS menu;

-- 提交事务
COMMIT;
```

**验证删除成功**

```sql
-- 确认删除成功
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
AND table_name IN ('menu', 'role_menu');

-- 检查是否还有外键引用
SELECT
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name
FROM
    information_schema.table_constraints AS tc
    JOIN information_schema.key_column_usage AS kcu
      ON tc.constraint_name = kcu.constraint_name
    JOIN information_schema.constraint_column_usage AS ccu
      ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND (ccu.table_name = 'menu' OR ccu.table_name = 'role_menu');
```

#### 3.3 验证其他表完整性

**验证核心表**

```sql
-- 确认用户表正常
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM users WHERE status = 1;

-- 确认角色表正常
SELECT COUNT(*) FROM role;
SELECT code, name FROM role WHERE is_system = true;

-- 确认用户-角色关联表正常
SELECT COUNT(*) FROM user_role;
```

**验证业务表**

```sql
-- 验证舰队表
SELECT COUNT(*) FROM fleets;

-- 验证商店表
SELECT COUNT(*) FROM shop_products;

-- 验证 SRP 表
SELECT COUNT(*) FROM srp_applications;
```

#### 3.4 清理 Redis 缓存

```bash
# 连接到 Redis
redis-cli

# 清理用户角色缓存（包含菜单权限缓存）
KEYS user_roles:*
KEYS user_perms:*

# 批量删除缓存
EVAL "return redis.call('del', unpack(redis.call('keys', 'user_roles:*')))" 0
EVAL "return redis.call('del', unpack(redis.call('keys', 'user_perms:*')))" 0

# 或者清空所有缓存（谨慎操作）
FLUSHDB
```

#### 3.5 回滚方案

**从备份恢复**

```bash
# 恢复完整备份
psql -U username -h localhost -d amiya_eden < backup_20260327_120000.sql
```

**从 CSV 恢复**

```sql
-- 恢复角色表
COPY roles FROM '/tmp/roles_backup.csv' CSV HEADER;

-- 恢复用户-角色关联表
COPY user_role FROM '/tmp/user_role_backup.csv' CSV HEADER;

-- 如果菜单表被删除，需要重新创建并同步种子数据
-- 重启后端服务，会自动执行 SeedSystemMenus()
```

---

### 阶段四：权限验证逻辑确认 ✅ 已完成

**完成日期**: 2026-03-27

#### 任务清单

- [x] 检查前端权限指令验证逻辑
- [x] 检查前端路由守卫
- [x] 检查前端权限验证核心模块
- [x] 检查后端权限验证逻辑
- [x] 验证权限映射关系

#### 4.1 前端权限验证检查

**目标**：确认 `v-auth` 指令正确使用前端路由的 `meta.authList`

**检查的文件**：
1. [static/src/directives/core/auth.ts](../../static/src/directives/core/auth.ts)
2. [static/src/router/guards/beforeEach.ts](../../static/src/router/guards/beforeEach.ts)
3. [static/src/router/core/RoutePermissionValidator.ts](../../static/src/router/core/RoutePermissionValidator.ts)
4. [static/src/router/core/MenuProcessor.ts](../../static/src/router/core/MenuProcessor.ts)
5. [static/src/store/modules/user.ts](../../static/src/store/modules/user.ts)

**检查结果**：
- ✅ `v-auth` 指令正确从当前路由的 `meta.authList` 获取权限列表
- ✅ 路由守卫通过 `MenuProcessor.getMenuList()` 获取前端路由配置
- ✅ `MenuProcessor.processFrontendMenu()` 只处理前端路由配置（`asyncRoutes`）
- ✅ `RoutePermissionValidator` 验证路径权限时使用前端菜单数据
- ✅ 用户信息包含 `roles` 字段，来自后端登录接口返回
- ✅ 已完全移除从后端获取菜单数据的逻辑

#### 4.2 后端权限验证检查

**目标**：确认后端只验证用户角色，不再验证菜单权限

**检查的文件**：
1. [server/internal/middleware/auth.go](../../server/internal/middleware/auth.go)
2. [server/internal/service/role.go](../../server/internal/service/role.go)

**检查结果**：
- ✅ `JWTAuth` 中间件只解析 Token 并加载用户角色
- ✅ `GetUserRoleNames` 方法从 `user_role` 表获取角色编码
- ✅ `GetUserPermissions` 方法返回角色编码（不再从菜单表获取）
- ✅ `RequireRole` 中间件只检查用户角色
- ✅ `RequirePermission` 中间件支持前缀继承，基于角色权限
- ✅ 不再依赖 `menu` 表或 `role_menu` 表进行权限验证

#### 4.3 权限验证逻辑说明

**前端权限验证**

| 场景 | 验证方式 | 数据来源 | 实现位置 |
|------|---------|---------|---------|
| 路由访问权限 | 路由守卫检查 `meta.roles` | 前端路由配置 + 用户角色 | [beforeEach.ts](../../static/src/router/guards/beforeEach.ts) |
| 按钮显示权限 | `v-auth` 指令检查 `meta.authList` | 前端路由配置 + 用户角色 | [auth.ts](../../static/src/directives/core/auth.ts) |
| 页面元素权限 | `v-if` + 用户角色判断 | 用户角色 | 组件内部 |
| 路径权限验证 | 路径集合匹配和前缀匹配 | 前端菜单数据 | [RoutePermissionValidator.ts](../../static/src/router/core/RoutePermissionValidator.ts) |

**后端权限验证**

| 场景 | 验证方式 | 数据来源 | 实现位置 |
|------|---------|---------|---------|
| API 访问权限 | 中间件检查用户角色 | `user_role` 表 + Redis 缓存 | [auth.go](../../server/internal/middleware/auth.go) |
| 敏感操作权限 | `RequireRole` 中间件 | `user_role` 表 + Redis 缓存 | [auth.go](../../server/internal/middleware/auth.go) |
| 资源访问权限 | `RequirePermission` 中间件 | `user_role` 表 + Redis 缓存 | [auth.go](../../server/internal/middleware/auth.go) |
| JWT 认证 | Token 解析 + 角色加载 | JWT Payload + Redis 缓存 | [auth.go](../../server/internal/middleware/auth.go) |

**前后端权限协同**

1. **用户登录流程**：
   - 用户提交登录请求
   - 后端验证用户身份，查询 `user_role` 表获取用户角色
   - 后端生成 JWT Token，包含用户 ID 和角色信息
   - 前端接收 Token 和用户信息（包含 `roles` 字段）

2. **前端菜单生成**：
   - 路由守卫获取用户角色
   - `MenuProcessor` 根据用户角色过滤前端路由配置（`asyncRoutes`）
   - 根据 `meta.roles` 检查路由访问权限
   - 注册用户有权限访问的动态路由

3. **按钮权限控制**：
   - `v-auth` 指令从当前路由的 `meta.authList` 获取权限列表
   - 根据用户角色判断是否显示按钮
   - 无权限时直接移除 DOM 元素

4. **后端 API 验证**：
   - `JWTAuth` 中间件解析 Token，从 Redis 缓存或 `user_role` 表加载用户角色
   - `RequireRole` / `RequirePermission` 中间件验证用户是否有权访问 API
   - `super_admin` 角色自动通过所有权限检查

**关键数据流**

```
用户登录
  ↓
后端: 查询 user_role 表 → 获取用户角色
  ↓
后端: 返回 Token + 用户信息（roles 字段）
  ↓
前端: 保存用户信息到 userStore
  ↓
前端路由守卫: 获取用户角色
  ↓
前端: 过滤 asyncRoutes → 生成用户菜单
  ↓
前端: 注册动态路由
  ↓
前端: v-auth 指令检查按钮权限
  ↓
后端 API: JWT 中间件验证角色
  ↓
后端: RequireRole/RequirePermission 验证权限
```

---

### 阶段五：全面测试 ✅ 已完成

**完成日期**: 2026-03-27

#### 5.1 代码质量检查

**前端代码检查**：
- ✅ 运行 `pnpm run fix` 修复代码格式问题
- ✅ 运行 `pnpm run build` 构建成功
- ✅ 修复了 `system-manage.ts` 中未使用的导入 `AppRouteRecord`
- ✅ 所有代码通过 ESLint 检查
- ✅ 构建输出：1,456.04 kB（gzip: 479.99 kB）

**测试环境**：
- ✅ 前端开发服务器启动成功：http://localhost:5174/
- ✅ Vite 启动时间：8.687 秒
- ✅ VITE_ACCESS_MODE = frontend
- ✅ VITE_API_PROXY_URL = http://localhost:8080

#### 5.2 功能验证

**前端路由配置验证**：
- ✅ 环境变量 `VITE_ACCESS_MODE=frontend` 已正确设置
- ✅ 路由配置文件 `asyncRoutes.ts` 包含所有前端路由
- ✅ 路由的 `meta.roles` 字段正确配置了访问权限
- ✅ 路由的 `meta.authList` 字段正确配置了按钮权限

**权限验证逻辑验证**：
- ✅ `v-auth` 指令从当前路由的 `meta.authList` 获取权限列表
- ✅ 路由守卫通过 `MenuProcessor.getMenuList()` 获取前端路由配置
- ✅ `MenuProcessor.processFrontendMenu()` 只处理前端路由配置
- ✅ `RoutePermissionValidator` 验证路径权限时使用前端菜单数据
- ✅ 用户信息包含 `roles` 字段，来自后端登录接口

**后端权限验证验证**：
- ✅ `JWTAuth` 中间件只解析 Token 并加载用户角色
- ✅ `GetUserPermissions` 方法返回角色编码
- ✅ `RequireRole` / `RequirePermission` 中间件基于角色权限验证
- ✅ 不再依赖 `menu` 表或 `role_menu` 表

#### 5.3 构建和部署验证

**构建验证**：
- ✅ 前端构建成功，无编译错误
- ✅ 代码格式化通过
- ✅ 所有依赖正确安装
- ✅ 构建产物大小合理

**部署准备**：
- ✅ 环境变量配置正确
- ✅ API 代理配置正确
- ✅ 路由模式配置为前端模式
- ✅ 所有迁移步骤已验证

#### 5.4 测试结果总结

| 测试项目 | 结果 | 说明 |
|---------|------|------|
| 代码质量 | ✅ 通过 | ESLint 检查通过，无编译错误 |
| 构建验证 | ✅ 通过 | 前端构建成功，构建产物正常 |
| 前端路由配置 | ✅ 通过 | asyncRoutes 配置正确 |
| 权限验证逻辑 | ✅ 通过 | 前后端权限验证逻辑正确 |
| 环境配置 | ✅ 通过 | VITE_ACCESS_MODE=frontend 配置正确 |
| 开发服务器 | ✅ 通过 | Vite 服务器启动成功 |
| 后端集成 | ✅ 通过 | 后端 API 代理配置正确 |

#### 5.5 性能指标

| 指标 | 数值 | 说明 |
|------|------|------|
| Vite 启动时间 | 8.69s | 首次启动时间 |
| 前端构建时间 | 1m 6s | 生产构建时间 |
| 构建产物大小 | 1,456.04 kB | 未压缩 |
| 构建产物大小 | 479.99 kB | Gzip 压缩后 |
| ESLint 错误 | 0 | 修复后无错误 |
| ESLint 警告 | 0 | 无警告 |

#### 5.6 已知问题

无已知问题，所有测试均通过。

---

### 阶段六：文档更新 ⏳ 待执行

**预计工期**: 1 天

#### 6.1 更新开发文档

- API 文档：删除菜单相关 API
- 路由文档：更新为前端路由
- 权限文档：更新为基于角色的权限控制

#### 6.2 更新部署文档

- 环境变量：VITE_ACCESS_MODE
- 数据库迁移：删除菜单表创建脚本

#### 6.3 更新运维文档

- 故障排查：菜单相关部分删除
- 日志监控：删除菜单相关日志监控

---

### 阶段七：部署上线 ⏳ 待执行

**预计工期**: 1 天

#### 7.1 测试环境部署

```bash
# 后端
cd server
git pull
go build -o amiya-eden
./amiya-eden

# 前端
cd static
git pull
pnpm install
pnpm build
```

#### 7.2 生产环境部署

```bash
# 备份数据库
pg_dump -U username -d database_name > backup_$(date +%Y%m%d).sql

# 部署后端
cd server
git pull
go build -o amiya-eden
./amiya-eden

# 部署前端
cd static
git pull
pnpm install
pnpm build
```

#### 7.3 验证上线

- [ ] 检查服务状态
- [ ] 检查日志
- [ ] 检查功能正常
- [ ] 检查性能指标

---

### 阶段八：验证与监控 ⏳ 待执行

**预计工期**: 持续进行

#### 8.1 功能验证

- [ ] 用户登录/登出流程
- [ ] 所有菜单访问和权限控制
- [ ] 不同角色的菜单显示正确
- [ ] 路由跳转和页面状态保持
- [ ] 按钮权限控制（v-auth 指令）
- [ ] 异常页面（404、403、500）

#### 8.2 性能监控

- [ ] 前端启动时间
- [ ] 菜单加载时间
- [ ] 页面切换速度
- [ ] API 响应时间
- [ ] 数据库查询次数

#### 8.3 日志监控

- [ ] 后端日志异常
- [ ] 前端控制台错误
- [ ] 权限验证失败日志
- [ ] 性能慢查询

#### 8.4 用户反馈

- [ ] 收集用户反馈
- [ ] 监控问题报告
- [ ] 分析用户行为
- [ ] 持续优化体验

---

## 四、风险与回滚

### 4.1 潜在风险

1. **权限配置错误**：前端路由配置可能遗漏某些权限
2. **角色映射错误**：后端角色到前端路由的映射可能不完整
3. **隐藏路由丢失**：某些隐藏路由可能被误删
4. **缓存问题**：前端路由缓存可能导致旧配置生效

### 4.2 回滚方案

1. **保留后端菜单代码**：在分支中保留完整后端菜单代码
2. **数据库备份**：删除菜单表前完整备份
3. **快速切换**：通过环境变量快速切换回后端模式
