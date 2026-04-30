---
status: completed
doc_type: completed
owner: engineering
last_reviewed: 2026-04-30
completed: 2026-04-30
source_of_truth:
  - server/internal/router/router.go
  - server/internal/handler/sys_wallet.go
  - server/internal/service/sys_wallet.go
  - server/internal/repository/sys_wallet.go
  - server/internal/model/sys_wallet.go
  - static/src/views/system/wallet/index.vue
  - static/src/views/system/wallet/modules/wallet-list.vue
  - static/src/views/system/wallet/modules/wallet-transactions.vue
  - static/src/views/system/wallet/modules/wallet-logs.vue
  - static/src/views/system/wallet/modules/wallet-analysis.vue
  - static/src/api/sys-wallet.ts
  - static/src/types/api/api.d.ts
  - docs/features/current/commerce.md
  - docs/features/current/pap-exchange.md
---

# `/system/wallet` 分析概览 Tab

## 问题描述

- 页面：`/system/wallet`
- 现有 tab：`钱包列表`、`流水查询`、`操作日志`

当前系统钱包管理页只提供明细查询和人工调整，没有一层面向管理视角的分析概览。管理员在查看系统钱包时，需要在多个 tab 间手动拼接信息，才能回答下面的问题：

- 当前系统钱包总体规模如何
- 钱主要从哪些业务来源进入
- 钱主要流向哪些业务场景
- 哪些用户、哪些操作人、哪些调整行为更值得关注

## 目标

在 `system/wallet` 页面新增一个只读分析 tab，基于现有 `system_wallet`、`wallet_transaction`、`wallet_log` 三类数据输出聚合结果，帮助管理员快速判断：

- 钱包总体健康度
- 近 30 天资金趋势
- 收支来源结构
- 重点用户与重点操作人
- 可疑或值得审计的异常模式

## 非目标

- 不新增数据库表
- 不改变现有钱包写入逻辑
- 不改变现有钱包列表、流水、日志 tab 的行为
- 不做月粒度切换作为 v1 标准能力
- 不把分析页做成可编辑页面

## 现状

### 现有前端结构

**文件**：`static/src/views/system/wallet/index.vue`

当前页面结构：

- `wallets`：钱包列表
- `transactions`：流水查询
- `logs`：操作日志

页面已经具备：

- `ElTabs` 容器
- `WalletList`、`WalletTransactions`、`WalletLogs` 三个子模块
- 调整余额弹窗
- 通过 `adminAdjustWallet` 进行管理员余额调整

### 现有后端能力

**路由**：`POST /api/v1/system/wallet/*`

**处理器**：`server/internal/handler/sys_wallet.go`

**服务层**：`server/internal/service/sys_wallet.go`

**仓储层**：`server/internal/repository/sys_wallet.go`

当前已存在的管理端接口：

- 钱包列表
- 钱包详情
- 钱包调整
- 钱包流水
- 钱包操作日志

现有查询已经能拿到：

- 用户钱包余额
- 钱包流水明细
- 钱包操作日志
- 流水对应的 `ref_type`
- 流水对应的操作人信息
- 用户 / 主人物匹配信息

这些现有能力足够支持 v1 分析 tab 的实时聚合查询。

## 方案

### 1. 前端新增 tab

在 `static/src/views/system/wallet/index.vue` 中新增一个 `analysis` tab，tab 名称为：

- 中文：`分析概览`
- 英文：`Analysis`

默认行为：

- 页面初次进入时默认停留在 `wallets`
- 用户切换到 `analysis` 后，默认查询最近 30 天
- 分析结果只读，不提供调整入口

### 2. 页面布局

分析 tab 建议分成四个区域：

#### 2.1 核心指标卡

展示以下指标：

- 钱包总数
- 总余额
- 活跃钱包数
- 收入总额
- 支出总额
- 净流入

#### 2.2 趋势图

按自然日展示近 30 天趋势：

- 每日收入
- 每日支出
- 每日净流入

#### 2.3 结构图

展示资金结构与来源去向：

- 按 `ref_type` 的收入占比
- 按 `ref_type` 的支出占比
- Top 收入用户
- Top 支出用户

#### 2.4 异常面板

展示规则化异常榜单：

- 大额流水
- 频繁调整
- 操作人集中度异常

### 3. 后端新增聚合接口

新增接口：

- `POST /api/v1/system/wallet/analytics`

接口归属：

- `admin` 角色可访问
- 与现有 `/system/wallet/*` 权限边界一致

#### 3.1 请求体

建议请求字段：

```ts
{
  start_date: string
  end_date: string
  ref_types?: string[]
  user_keyword?: string
  top_n?: number
}
```

约束：

- `start_date` 必填
- `end_date` 必填
- `start_date <= end_date`
- 时间跨度最多 365 天
- `top_n` 默认 10，允许范围 1-50
- `ref_types` 可选，空数组表示不限制

#### 3.2 响应体

建议响应结构命名为 `Api.SysWallet.WalletAnalytics`，包含：

- `summary`
- `daily_series`
- `ref_type_breakdown`
- `top_inflow_users`
- `top_outflow_users`
- `admin_adjust_stats`
- `anomalies`

建议字段形态：

```ts
summary: {
  wallet_count: number
  active_wallet_count: number
  total_balance: number
  income_total: number
  expense_total: number
  net_flow: number
}

daily_series: Array<{
  date: string
  income: number
  expense: number
  net_flow: number
}>

ref_type_breakdown: Array<{
  ref_type: string
  income: number
  expense: number
  count: number
}>

top_inflow_users: Array<{
  user_id: number
  character_name?: string
  amount: number
}>

top_outflow_users: Array<{
  user_id: number
  character_name?: string
  amount: number
}>

admin_adjust_stats: {
  count: number
  amount_total: number
  by_operator: Array<{
    operator_id: number
    operator_name?: string
    count: number
    amount_total: number
  }>
}

anomalies: {
  large_transactions: Array<...>
  frequent_adjustments: Array<...>
  operator_concentration: Array<...>
}
```

## 统计口径

### 1. 金额口径

- `income_total = sum(amount > 0)`
- `expense_total = abs(sum(amount < 0))`
- `net_flow = income_total - expense_total`

### 2. 时间口径

- 趋势按 `wallet_transaction.created_at` 的自然日聚合
- 使用服务端时区
- 仅统计请求时间窗内的数据

### 3. 活跃钱包

- `active_wallet_count`：时间窗内至少出现过一笔流水的去重用户数

### 4. `ref_type` 口径

- 仅统计窗口内流水
- 未出现的 `ref_type` 不返回
- 如果传了 `ref_types`，则所有聚合区块都必须受该筛选约束

### 5. 用户关键词口径

`user_keyword` 的匹配语义与现有钱包列表保持一致：

- 用户昵称
- 任意已绑定人物名

该过滤必须同时作用于：

- summary
- daily_series
- ref_type_breakdown
- top_inflow_users
- top_outflow_users
- anomalies

## 异常规则

### 1. 大额流水

规则：

- `abs(amount) >= P95`
- 且金额至少为 `100`

输出内容：

- 流水 ID
- 用户 ID
- 主人物名
- 金额
- `ref_type`
- 创建时间

### 2. 频繁调整

规则：

- 同一 `target_uid`
- 在统计窗口内
- `admin_adjust` 日内次数 `>= 3`

输出内容：

- 目标用户
- 调整次数
- 调整总额
- 最近一次调整时间

### 3. 操作人集中度异常

规则：

- 单个操作人调整金额占窗口总调整金额 `>= 40%`

输出内容：

- 操作人 ID
- 操作人名称
- 调整次数
- 调整总额
- 占比

## 实现边界

### 后端

实现位置建议：

- `server/internal/repository/sys_wallet.go`
- `server/internal/service/sys_wallet.go`
- `server/internal/handler/sys_wallet.go`
- `server/internal/router/router.go`

实现原则：

- 仓储层只负责查询与聚合查询
- 服务层负责口径、聚合编排和异常规则
- 处理器只做请求绑定与响应输出

### 前端

实现位置建议：

- `static/src/views/system/wallet/index.vue`
- 新增 `static/src/views/system/wallet/modules/wallet-analysis.vue`
- `static/src/api/sys-wallet.ts`
- `static/src/types/api/api.d.ts`
- `static/src/locales/langs/zh.json`
- `static/src/locales/langs/en.json`

前端页面要求：

- 保持页面整体和现有系统钱包风格一致
- 分析 tab 只读
- 空态要明确提示“当前时间范围没有可分析数据”

## 验收标准

### 功能验收

1. 打开 `/system/wallet`
2. 切换到 `分析概览`
3. 默认自动拉取最近 30 天分析数据
4. 能看到核心指标卡、趋势图、结构图、异常面板
5. 修改时间范围后，所有区块同步刷新
6. 修改 `ref_type` 过滤后，所有区块同步刷新
7. 修改用户关键词后，所有区块同步刷新
8. 空数据时仍能正常渲染页面，不报错

### 后端验收

1. 接口支持合法时间窗请求
2. 时间反转返回参数错误
3. 超过 365 天返回参数错误
4. `top_n` 超界返回参数错误
5. 空数据返回完整结构
6. 过滤条件在所有聚合区块中一致生效

### 回归验收

- `wallets` tab 行为不变
- `transactions` tab 行为不变
- `logs` tab 行为不变
- 管理员调整余额流程不变
- 既有钱包列表、流水、日志测试不回归

## 测试计划

### 后端测试

- 时间窗边界测试
- 空数据测试
- 正负流水混合测试
- `ref_types` 过滤测试
- `user_keyword` 过滤测试
- 异常规则测试

### 前端测试

- tab 切换测试
- 默认查询参数测试
- 响应数据映射测试
- 空态测试
- 错误态测试

### 最低验证命令

```bash
cd server && go test ./...
cd static && pnpm lint .
cd static && pnpm exec vue-tsc --noEmit
cd static && pnpm test:unit
```

## 风险与取舍

- 这版采用实时聚合，不新建分析表，优点是实现快、成本低，缺点是数据量大时查询压力更高
- 异常规则采用固定阈值，优点是可解释、落地快，缺点是后续可能需要参数化
- 默认按日展示趋势，足以支撑 v1 运营分析；按月视图可作为后续增强

## 待办

- [x] 新增后端分析接口
- [x] 补充 service / repository / handler 测试
- [x] 新增前端分析 tab
- [x] 补充前端 API 与类型定义
- [x] 增加本地化文案
- [x] 更新 feature 文档与实现说明

## 当前进度

- 后端：`/api/v1/system/wallet/analytics` 已接入，支持 `start_date` / `end_date` / `ref_types` / `user_keyword` / `top_n`
- 前端：`/system/wallet` 已新增 `分析概览` tab，默认近 30 天自动加载并支持筛选刷新
- 文档：路由索引、feature 文档与本草案已同步到当前实现
- 验证：后端 `go test` / `go build` 与前端 `vue-tsc` 已通过
