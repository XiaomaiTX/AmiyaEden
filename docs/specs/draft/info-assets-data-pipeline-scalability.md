---
status: draft
doc_type: spec
owner: engineering
last_reviewed: 2026-07-22
source_of_truth:
  - server/internal/router/router.go
  - server/internal/handler/eve_info.go
  - server/internal/service/asset.go
  - server/internal/repository/asset.go
  - server/internal/model/esi/asset.go
  - server/pkg/eve/esi/task_assets.go
  - static/src/views/info/assets/index.vue
  - static/src/api/eve-info.ts
  - static-react/src/pages/info-assets-page.tsx
  - static-react/src/api/eve-info.ts
  - docs/features/current/info-and-reporting.md
---

# `/info/assets` 数据链路扩容与可观测性修复草案

> 本文是修复草案，不代表当前已实现行为。

## 背景

当前 `/info/assets` 页面在用户绑定单角色或多角色且资产量较小时可以正常工作，但当单角色资产规模很大、或多个角色合并后总资产条目和根位置数明显增加时，页面会出现“暂无资产数据，请等待数据刷新”。

基于当前实现检查，问题不在 ESI 资产同步任务本身，而更可能出现在“展示查询链路”：

- ESI 端已经使用 `GetPaginated()` 拉取资产分页，并按 1000 个 `item_id` 分批查询名称、按 500 条分批入库。
- `/api/v1/info/assets` 查询会一次性拉取当前用户所有角色的全部资产。
- 查询接口会在服务层把全部资产构建成完整位置树，并对每个根位置同步解析名称。
- 根位置名解析会触发空间站缓存查询、SDE 查询、以及玩家建筑 ESI 查询。
- 旧 Vue 页面会把请求异常直接吞掉并回退为空态，导致真实错误被伪装成“暂无数据”。

因此，本草案聚焦于“展示查询链路扩容、错误可观测性、接口分页化和读写职责拆分”。

## 当前状态

### 当前前端行为

- 旧 Vue 页面：`static/src/views/info/assets/index.vue`
  - 请求 `fetchInfoAssets()`
  - `catch` 分支仅将 `assetsData` 置空，不展示真实错误
  - 最终统一显示 `$t('info.noAssetData')`
- React 页面：`static-react/src/pages/info-assets-page.tsx`
  - 请求失败会显示 `infoAssets.loadFailed`
  - 空数据会显示 `infoAssets.empty`

结论：

- 用户当前提到的“暂无资产数据，请等待数据刷新”更符合旧 Vue 页面行为。
- 当前线上或主要入口大概率仍存在旧 Vue 页面使用场景，至少该页面的空态/错误态逻辑仍会误导排障。

### 当前后端行为

入口：

- `server/internal/handler/eve_info.go`
- `server/internal/service/asset.go`
- `server/internal/repository/asset.go`

当前 `GetUserAssets()` 的主要处理步骤：

1. 查询当前用户绑定的全部角色。
2. 使用 `GetAssetsByCharacterIDs()` 一次性查询全部角色资产。
3. 收集全部 `type_id`，批量查询 SDE 类型信息。
4. 在内存中建立：
   - `item_id -> asset`
   - `parent_item_id -> children`
   - `location_id -> root items`
5. 对全部根位置同步执行 `resolveLocationName()`。
6. 递归构建完整 `locations -> items -> children` 树。
7. 将整棵树一次性返回给前端。

### 当前 ESI 同步行为

入口：

- `server/pkg/eve/esi/task_assets.go`

现状：

- `GET /characters/{id}/assets/` 通过 `GetPaginated()` 自动分页。
- 名称查询 `POST /characters/{id}/assets/names/` 按 1000 个 `item_id` 分批。
- 本地入库采用“先删后插”，按 500 条批量写入。

结论：

- 当前没有明确证据表明“资产同步任务因数据量过大而直接丢数或截断”。
- 当前更可疑的是展示接口的全量读、全量组树、同步位置名解析和响应体过大。

## 问题拆解

### 1. 错误被前端吞掉，导致误判

旧 Vue 页面当前逻辑：

- 请求失败
- `assetsData = null`
- `loading = false`
- 页面显示“暂无资产数据，请等待数据刷新”

这会把以下场景全部伪装成“没有数据”：

- 后端超时
- Nginx 504
- 后端业务报错
- 前端 Axios 15 秒超时
- 查询中同步解析建筑名导致整体慢查询

### 2. 查询接口存在明显的全量放大问题

当前接口是“全量用户资产树”模式，放大点包括：

- 一次性读取该用户全部角色全部资产
- 一次性查询全部 `type_id` 对应名称
- 一次性递归构建完整树
- 一次性返回全部位置和全部嵌套物品

这会导致：

- 服务层 CPU 和内存消耗随总资产数近似线性甚至叠加增长
- JSON 序列化时间增长
- 响应体体积过大
- 前端首屏渲染节点数过多

### 3. 位置名解析在读接口里做了外部依赖调用

`resolveLocationName()` 目前会：

- 空间站：查本地缓存 -> 查 SDE `staStations` -> 必要时请求 ESI 公共接口
- 建筑：查本地缓存 -> 遍历角色 token -> 请求 ESI `/universe/structures/{id}/`

这意味着一个“读页面接口”可能：

- 触发多次外部 HTTP 请求
- 触发多个 token 校验
- 在根位置数很大时串行叠加耗时

这会直接把展示接口变成“查询 + 补缓存 + 外部 IO”的混合路径，不适合高数据量场景。

### 4. 当前超时配置对大响应不友好

- 旧 Vue HTTP 超时：15 秒
  - `static/src/utils/http/index.ts`
- 反向代理超时：60 秒
  - `static/nginx.conf`
- 资产服务内部 HTTP Client 超时：30 秒
  - `server/internal/service/asset.go`

在同步解析位置名的情况下，即使数据库可承受，全链路也容易因为某个慢位置名解析请求而触发前端超时。

## 目标

- 修复 `/info/assets` 的错误可观测性，区分“无数据”和“请求失败”。
- 让资产展示查询不再依赖运行时外部 ESI 位置名解析。
- 将“全量资产树接口”改造成适合大数据量的分页/懒加载接口。
- 将大部分查询成本前移为：
  - 刷新阶段预处理
  - 本地缓存读取
  - 分页式展示查询
- 为该链路补齐回归测试和诊断日志。

## 非目标

- 本阶段不重写 ESI 资产同步任务的核心拉取逻辑。
- 本阶段不做资产价值估算、价格聚合或筛选能力扩展。
- 本阶段不同时重构旧 Vue 和 React 的 UI 风格，只保证行为与接口正确。
- 本阶段不做无限层级前端虚拟树组件重构，首要目标是链路稳定和接口收敛。

## 核心设计决策

### 1. 读接口禁止同步请求外部 ESI

决策：

- `GetUserAssets()` 及其拆分后的读接口不能在请求链路内触发 ESI 建筑名/空间站名解析。

理由：

- 展示查询必须是本地可预测耗时。
- 外部依赖应该前移到同步任务或独立补全任务。
- 否则数据量越大，根位置越多，接口越不稳定。

### 2. 资产展示从“整棵树全量返回”改为“分页摘要 + 懒加载”

决策：

- `/info/assets` 不再使用一个接口返回完整树。
- 改为：
  - 位置摘要分页
  - 位置根物品分页
  - 子物品按需加载

理由：

- 大部分用户不需要一次看完所有位置和所有容器内容。
- 首屏只需要位置列表和统计。
- 展开位置、展开容器才需要进一步查询。

### 3. 位置名缓存缺失时返回占位名，不阻塞查询

决策：

- 查询阶段若本地没有结构名/空间站名缓存，直接返回稳定占位名。
- 缺失位置名通过后台补全任务异步回填。

理由：

- 占位名虽不完美，但比接口失败更可接受。
- 该策略可以将“弱一致性展示”换取“强稳定性”。

### 4. 先修可观测性，再做接口拆分

决策：

- 迁移阶段先保证后端链路和两端错误态语义一致；若问题只存在于 Vue 或 React，应分别修复对应消费者，不再默认优先修改旧 Vue。

理由：

- 当前首先需要把“真实故障”暴露出来。
- 否则后续接口优化期间仍难以判断问题是否真正消除。

## 目标形态

### 前端行为

#### 旧 Vue 页面

文件：

- `static/src/views/info/assets/index.vue`

目标：

- 请求失败时显示真实错误信息，不再落到空态。
- 空态仅用于接口成功但没有任何位置数据。
- 首屏只加载位置摘要列表。
- 用户点击位置后再加载该位置的根物品。
- 用户展开容器/舰船后再加载子物品。

#### React 页面

文件：

- `static-react/src/pages/info-assets-page.tsx`

目标：

- 与新接口保持一致的数据获取方式。
- 继续保留“失败”和“空数据”的明确区分。
- 改为分页与懒加载，不再依赖全量树响应。

### 后端行为

目标：

- 资产查询接口完全基于本地数据库。
- 不在请求链路内触发 ESI 请求。
- 每个接口只返回当前视图所需的最小数据集。

## 接口改造草案

> 现有 `POST /api/v1/info/assets` 可短期保留用于迁移，但不作为长期主接口。

### 1. `POST /api/v1/info/assets/locations`

用途：

- 获取当前用户资产位置摘要分页。

请求字段建议：

```json
{
  "language": "zh",
  "page": 1,
  "page_size": 20,
  "keyword": ""
}
```

响应字段建议：

```json
{
  "total_locations": 0,
  "total_items": 0,
  "locations": [
    {
      "location_id": 0,
      "location_type": "station",
      "location_name": "Jita IV - Moon 4 - Caldari Navy Assembly Plant",
      "top_level_count": 0,
      "total_item_count": 0,
      "character_count": 0
    }
  ]
}
```

约束：

- 只返回位置维度摘要。
- 不返回根物品数组。
- `keyword` 首版可先只匹配位置名；若扩展到物品名搜索，需单独优化查询计划。

### 2. `POST /api/v1/info/assets/location-items`

用途：

- 获取某个位置下的根物品分页。

请求字段建议：

```json
{
  "language": "zh",
  "location_id": 0,
  "page": 1,
  "page_size": 50,
  "keyword": ""
}
```

响应字段建议：

```json
{
  "location_id": 0,
  "location_name": "",
  "total_root_items": 0,
  "items": [
    {
      "item_id": 0,
      "type_id": 0,
      "type_name": "",
      "group_name": "",
      "category_id": 0,
      "quantity": 0,
      "location_flag": "",
      "is_singleton": false,
      "is_blueprint_copy": false,
      "asset_name": "",
      "character_id": 0,
      "character_name": "",
      "has_children": false,
      "child_count": 0
    }
  ]
}
```

约束：

- 只返回根物品，不内嵌 `children`。
- 新增 `has_children` 与 `child_count`，供前端决定是否展示展开按钮。

### 3. `POST /api/v1/info/assets/children`

用途：

- 获取某个父物品的直接子物品列表。

请求字段建议：

```json
{
  "language": "zh",
  "parent_item_id": 0
}
```

响应字段建议：

```json
{
  "parent_item_id": 0,
  "items": [
    {
      "item_id": 0,
      "type_id": 0,
      "type_name": "",
      "group_name": "",
      "category_id": 0,
      "quantity": 0,
      "location_flag": "",
      "is_singleton": false,
      "is_blueprint_copy": false,
      "asset_name": "",
      "character_id": 0,
      "character_name": "",
      "has_children": false,
      "child_count": 0
    }
  ]
}
```

约束：

- 默认只返回直接子级，不递归返回整棵子树。
- 若未来要支持“递归展开到全树”，新增显式参数，不隐式回退旧行为。

## 后端代码修改方案

## 1. Handler 层

文件：

- `server/internal/handler/eve_info.go`

改动：

- 保留现有 `GetAssets()`，但标记为迁移过渡接口。
- 新增：
  - `GetAssetLocations()`
  - `GetAssetLocationItems()`
  - `GetAssetChildren()`
- 每个 handler 只做：
  - `ShouldBindJSON`
  - `middleware.GetUserID`
  - 调用 `AssetService`
  - 统一响应

具体点：

- 新增请求结构体对应 service 层 DTO。
- `GetAssets()` 后续可以改为内部调用分页接口并返回受限结果，或直接下线；实现时二选一，不要长期双轨演进。

## 2. Router 层

文件：

- `server/internal/router/router.go`

改动：

- 在 `info` 路由组里新增：
  - `POST /assets/locations`
  - `POST /assets/location-items`
  - `POST /assets/children`

要求：

- 与现有 `requireInfoAssetsRead` 保持同一权限边界。

## 3. Service 层

文件：

- `server/internal/service/asset.go`

改动原则：

- 将当前 `GetUserAssets()` 拆成多个小型读操作。
- 查询阶段不再调用：
  - `resolveLocationName()`
  - `resolveStationName()`
  - `resolveStructureName()`
  - `fetchAndCacheStation()`
  - `fetchAndCacheStructure()`
  - `esiGet()` / `esiGetPublic()`

具体修改建议：

### 3.1 新增请求/响应 DTO

新增：

- `InfoAssetLocationsRequest`
- `InfoAssetLocationItemsRequest`
- `InfoAssetChildrenRequest`
- `InfoAssetLocationsResponse`
- `InfoAssetLocationSummary`
- `InfoAssetLocationItemsResponse`
- `InfoAssetChildrenResponse`
- `AssetListItemNode`

其中 `AssetListItemNode` 与当前 `AssetItemNode` 区别：

- 去掉递归 `Children []AssetItemNode`
- 新增：
  - `HasChildren bool`
  - `ChildCount int`

### 3.2 重构 `GetUserAssets()`

建议处理方式：

- 短期：
  - 保留方法名，但改成使用本地缓存位置名和受限响应。
  - 绝不触发外部 ESI。
- 中长期：
  - 让旧接口仅用于兼容，内部按最大位置数、最大根物品数做硬限制并返回可诊断错误。

### 3.3 新增分页查询方法

新增方法：

- `GetUserAssetLocations(userID uint, req *InfoAssetLocationsRequest) (*InfoAssetLocationsResponse, error)`
- `GetUserAssetLocationItems(userID uint, req *InfoAssetLocationItemsRequest) (*InfoAssetLocationItemsResponse, error)`
- `GetUserAssetChildren(userID uint, req *InfoAssetChildrenRequest) (*InfoAssetChildrenResponse, error)`

实现要求：

- 统一先取 `listOwnedCharacters()`。
- 基于 `character_ids` 做数据库查询。
- 仅对当前页涉及的 `type_id` 做 SDE 名称补全。
- 仅通过本地缓存表或本地占位名生成 `location_name`。

### 3.4 位置名读取逻辑改为纯本地

建议新增本地函数：

- `resolveLocationNameLocal(locationID int64, locationType string) string`

逻辑：

- `station`
  - 先查 `eve_station`
  - 无缓存则查 SDE `staStations`
  - 仍无则返回 `Station-<id>`
- `structure` / `other`
  - 先查 `eve_structure`
  - 无缓存则返回 `Structure-<id>`
- `solar_system`
  - 查 SDE 名称
  - 无则 `System-<id>`

明确禁止：

- 在该函数中请求 ESI。

> 顶层 `item` 类型位置的特殊处理：资产数据中 `location_type=item` 且
> `location_id` 不在任何资产行 `item_id` 列的位置是顶层位置，可能是玩家建筑
> （Upwell Structure 等大型 ID）。解析这类位置时优先查 `eve_structures` 缓存，命中建筑名
> 则返回建筑名并把展示用 `location_type` 规范为 `structure`，未命中再走容器
> 位置逻辑（`asset_name` → SDE `type_name` → `Item-<id>`）。建筑缓存读取
> 不按 `update_at` 做新鲜度过滤，资产页应优先展示已有建筑名，即使缓存较旧。

### 3.5 缺失位置名补全改成异步入口

建议新增后台补全能力：

- `EnsureAssetLocationsCached(characterIDs []int64)` 或独立 job/service

职责：

- 扫描资产表中涉及的根位置
- 对缺失的 `eve_structure` / `eve_station` 缓存异步补齐
- 失败写冷却时间，避免每次都重试

首版可只写设计并预留函数，不强制与页面修复同批完成；但读接口必须先去除同步外部调用。

### 3.6 增加链路日志

在 `server/internal/service/asset.go` 增加结构化日志，至少记录：

- `user_id`
- `character_count`
- `asset_count`
- `root_location_count`
- `query_stage`
- `duration_ms`
- `response_location_count`
- `response_item_count`

建议分阶段打点：

- 角色查询完成
- 资产查询完成
- SDE 类型补全完成
- 位置摘要组装完成
- 根物品分页组装完成
- 子物品列表组装完成

## 4. Repository 层

文件：

- `server/internal/repository/asset.go`

当前问题：

- 只有按角色全量拉取接口：
  - `GetAssetsByCharacterID()`
  - `GetAssetsByCharacterIDs()`

需要新增面向分页展示的查询：

- `ListAssetLocationSummaries(characterIDs []int64, keyword string, page, pageSize int) (...)`
- `CountAssetLocationSummaries(characterIDs []int64, keyword string) (...)`
- `ListRootAssetsByLocation(characterIDs []int64, locationID int64, keyword string, page, pageSize int) (...)`
- `CountRootAssetsByLocation(characterIDs []int64, locationID int64, keyword string) (...)`
- `ListAssetChildren(characterIDs []int64, parentItemID int64) (...)`
- `ListChildCountsByParentIDs(characterIDs []int64, parentItemIDs []int64) (...)`

实现要求：

- 根物品定义沿用现有规则：
  - `location_type != "item"`
  - 或 `location_type == "item"` 但 `location_id` 不在同一用户资产 `item_id` 集合中
- 该规则最好下沉到 SQL 层或最少量的预处理层，不要每次先全量读再在内存中过滤。

### 4.1 查询形态建议

位置摘要查询至少要能输出：

- `location_id`
- `location_type`
- `top_level_count`
- `total_item_count`
- `character_count`

根物品查询至少要能输出：

- 当前页根物品基础字段
- `child_count`

### 4.2 索引补强

文件：

- `server/bootstrap/db.go`
- `server/internal/model/esi/asset.go`

建议新增或显式创建索引：

- `character_id, location_id`
- `character_id, location_type, location_id`
- `character_id, item_id`
- `location_id`
- `type_id`

若采用自定义 SQL 创建部分/组合索引，则在 `ensureCustomIndexes()` 中补充语句，而不是仅依赖 GORM 默认单列索引。

## 5. Model 层

文件：

- `server/internal/model/esi/asset.go`

改动：

- 保持现有字段不变，避免影响同步任务。
- 如需新增“位置名补全失败冷却”能力，不直接改资产表，优先放在缓存表：
  - `eve_structure`
  - `eve_station`
  - 或新增独立位置缓存表

理由：

- 资产事实表应保持原始资产数据，不混入查询期状态字段。

## 6. ESI 任务联动

文件：

- `server/pkg/eve/esi/task_assets.go`

本阶段不改核心拉取逻辑，但建议补两类改造：

### 6.1 增加刷新结果日志

补充输出：

- `asset_count`
- `nameable_item_count`
- `named_item_count`
- `batch_insert_count`
- `duration_ms`

### 6.2 为后续异步位置补全预留入口

在资产入库成功后，可选触发：

- “补全本次资产根位置名称缓存”的后台任务

注意：

- 该任务必须异步，不允许阻塞 `Execute()` 主流程提交。

## 前端代码修改方案

## 1. Vue 页面（过渡期消费者）

文件：

- `static/src/views/info/assets/index.vue`

具体改动：

### 1.1 修正错误态

当前：

- `catch { assetsData.value = null }`

目标：

- 增加 `errorMessage = ref('')`
- 请求失败时：
  - `errorMessage.value = error.message || t('info.assetLoadFailed')`
  - 不进入空态
- 页面新增错误提示区

### 1.2 从全量树改成分段加载

新增本地状态：

- `locationPage`
- `locationPageSize`
- `locations`
- `selectedLocationId`
- `locationItemsMap`
- `locationLoadingMap`
- `childrenLoadingMap`

修改行为：

- 页面挂载只调用 `fetchInfoAssetLocations`
- 点击位置时调用 `fetchInfoAssetLocationItems`
- 点击容器时调用 `fetchInfoAssetChildren`

### 1.3 国际化文案

文件：

- `static/src/locales/langs/zh.json`
- `static/src/locales/langs/en.json`

新增文案建议：

- `info.assetLoadFailed`
- `info.assetLocationLoadFailed`
- `info.assetChildrenLoadFailed`
- `info.assetNoLocationData`
- `info.assetNoItemsInLocation`

## 2. Vue API 包装

文件：

- `static/src/api/eve-info.ts`

改动：

- 保留 `fetchInfoAssets()` 兼容旧调用。
- 新增：
  - `fetchInfoAssetLocations()`
  - `fetchInfoAssetLocationItems()`
  - `fetchInfoAssetChildren()`

要求：

- 接口命名与后端路由一一对应。

## 3. Vue 类型定义

文件：

- `static/src/types/api/api.d.ts`

改动：

- 在 `Api.EveInfo` 下新增：
  - `AssetLocationsRequest`
  - `AssetLocationsResponse`
  - `AssetLocationSummary`
  - `AssetLocationItemsRequest`
  - `AssetLocationItemsResponse`
  - `AssetChildrenRequest`
  - `AssetChildrenResponse`
  - `AssetListItemNode`

要求：

- 不再要求 `AssetItemNode.children` 为接口主结构字段。
- 若保留旧类型，必须明确区分“旧全量树”和“新分页列表”。

## 4. React 页面与 API

文件：

- `static-react/src/pages/info-assets-page.tsx`
- `static-react/src/api/eve-info.ts`
- `static-react/src/types/api/eve-info.ts` 或对应类型文件

改动：

- 新增对应的 3 个 API wrapper。
- 页面从一次 `fetchInfoAssets()` 改成：
  - 初始加载位置摘要
  - 展开位置加载根物品
  - 展开条目加载子物品

要求：

- 保持现有失败态优于空态的行为。
- 不要把新接口适配硬塞回旧树结构再一次性渲染。

## 测试方案

## 1. 后端测试

建议新增：

- `server/internal/service/asset_test.go`
- `server/internal/repository/asset_test.go`
- `server/internal/handler/eve_info_asset_test.go` 或补入现有 handler test

覆盖点：

- 用户无角色时返回空位置列表
- 用户有资产但位置名缓存缺失时仍能成功返回占位名
- 大量位置摘要查询可分页返回
- 指定位置根物品分页返回且含 `has_children` / `child_count`
- 指定父物品子项查询只返回直接子级
- 读接口不触发外部 ESI 调用
- 旧接口在大响应场景下会返回可诊断错误或受限结果，而不是超时挂死

## 2. 前端测试

Vue：

- `static/src/views/info/assets/index.vue` 对应页面测试

覆盖点：

- 接口失败时显示错误，不显示“暂无资产数据”
- 空位置数据时显示空态
- 点击位置会触发根物品查询
- 点击容器会触发子物品查询

React：

- `static-react/src/pages/info-assets-page.test.tsx`

覆盖点：

- 初始只请求位置摘要
- 展开位置后才请求位置物品
- 展开子节点后才请求 children

## 3. 验证命令

最低验证建议：

```bash
cd server && go test ./...
cd server && go build ./...
cd static && pnpm exec vue-tsc --noEmit
cd static && pnpm test:unit
cd static-react && pnpm test
```

## 风险与缓解

风险：

- 旧接口与新接口并存时间过长，导致双重维护。
- 根物品判定若直接下沉 SQL，查询实现复杂度上升。
- 位置名补全异步化后，短期内用户会看到占位名。

缓解：

- 先约定迁移窗口，完成 React 页面与 API 回归后再执行入口替换；不得因为 React 已有页面就提前删除 Vue 消费者或共享后端接口。
- 分两阶段实现：
  - 第一阶段先在 service 层去掉同步 ESI，允许部分内存预处理
  - 第二阶段再把位置/根物品判定进一步下沉 repository/SQL
- 占位名策略配合异步补全和冷却机制，接受短期弱一致性。

## 实施阶段

### 阶段 1：止血与可观测性

目标：

- 不再把错误伪装成空数据。
- 读接口不再同步请求 ESI。

改动文件：

- `static/src/views/info/assets/index.vue`
- `static/src/locales/langs/zh.json`
- `static/src/locales/langs/en.json`
- `server/internal/service/asset.go`

完成标准：

- 请求失败时前端显示错误。
- 即使位置缓存缺失，接口仍能返回。

### 阶段 2：接口拆分

目标：

- 引入位置摘要/位置物品/子物品 3 个接口。

改动文件：

- `server/internal/handler/eve_info.go`
- `server/internal/router/router.go`
- `server/internal/service/asset.go`
- `server/internal/repository/asset.go`
- `static/src/api/eve-info.ts`
- `static/src/types/api/api.d.ts`
- `static/src/views/info/assets/index.vue`
- `static-react/src/api/eve-info.ts`
- `static-react/src/pages/info-assets-page.tsx`

完成标准：

- 双端页面都不再依赖全量树接口。

### 阶段 3：查询优化与缓存补全

目标：

- 加组合索引。
- 补异步位置名缓存任务。

改动文件：

- `server/bootstrap/db.go`
- `server/internal/model/esi/asset.go`
- `server/internal/service/asset.go`
- `server/pkg/eve/esi/task_assets.go`

完成标准：

- 位置名补全不阻塞查询。
- 组合索引覆盖新的主要查询路径。

## TODO

### 阶段 1

- [ ] `static/src/views/info/assets/index.vue`：新增独立错误态，失败时不再回退为空态。
- [ ] `static/src/locales/langs/zh.json`：新增资产加载失败与子查询失败文案。
- [ ] `static/src/locales/langs/en.json`：同步新增英文文案。
- [ ] `server/internal/service/asset.go`：移除查询链路中的 `fetchAndCacheStation()` / `fetchAndCacheStructure()` 调用。
- [ ] `server/internal/service/asset.go`：新增仅本地缓存的 `resolveLocationNameLocal()`。
- [ ] `server/internal/service/asset.go`：为资产查询增加阶段耗时日志。

### 阶段 2

- [ ] `server/internal/handler/eve_info.go`：新增 `GetAssetLocations()`。
- [ ] `server/internal/handler/eve_info.go`：新增 `GetAssetLocationItems()`。
- [ ] `server/internal/handler/eve_info.go`：新增 `GetAssetChildren()`。
- [ ] `server/internal/router/router.go`：注册 `/info/assets/locations`。
- [ ] `server/internal/router/router.go`：注册 `/info/assets/location-items`。
- [ ] `server/internal/router/router.go`：注册 `/info/assets/children`。
- [ ] `server/internal/service/asset.go`：新增位置摘要、位置根物品、子物品 3 组 DTO。
- [ ] `server/internal/service/asset.go`：新增 `GetUserAssetLocations()`。
- [ ] `server/internal/service/asset.go`：新增 `GetUserAssetLocationItems()`。
- [ ] `server/internal/service/asset.go`：新增 `GetUserAssetChildren()`。
- [ ] `server/internal/repository/asset.go`：新增位置摘要分页查询。
- [ ] `server/internal/repository/asset.go`：新增指定位置根物品分页查询。
- [ ] `server/internal/repository/asset.go`：新增指定父物品子项查询。
- [ ] `static/src/api/eve-info.ts`：新增 Vue 侧 3 个 API wrapper。
- [ ] `static/src/types/api/api.d.ts`：新增分页资产接口类型。
- [ ] `static/src/views/info/assets/index.vue`：改为首屏位置摘要、按需加载位置物品与子物品。
- [ ] `static-react/src/api/eve-info.ts`：新增 React 侧 3 个 API wrapper。
- [ ] `static-react/src/pages/info-assets-page.tsx`：切换到分页/懒加载模式。

### 阶段 3

- [ ] `server/bootstrap/db.go`：补资产查询组合索引创建语句。
- [ ] `server/internal/model/esi/asset.go`：按需要补充 GORM 索引标签，避免和自定义索引冲突。
- [ ] `server/pkg/eve/esi/task_assets.go`：补充刷新结果统计日志。
- [ ] `server/internal/service/asset.go`：设计并接入异步位置名缓存补全入口。
- [ ] `server/internal/service/asset.go`：为位置名补全失败增加冷却策略，避免高频重试。

### 测试与收尾

- [ ] `server/internal/service/asset_test.go`：补充空数据、占位名、分页与 children 查询测试。
- [ ] `server/internal/repository/asset_test.go`：补充位置摘要和根物品查询测试。
- [ ] `static/src/views/info/assets`：补页面错误态和懒加载交互测试。
- [ ] `static-react/src/pages/info-assets-page.test.tsx`：补分页/懒加载测试。
- [ ] `docs/features/current/info-and-reporting.md`：实现完成后更新当前行为描述。
- [ ] `docs/api/route-index.md`：实现完成后登记新增资产接口。

## 明确声明

- 本文档是提案，不代表当前已实现行为。
- 本文档仅定义 `/info/assets` 资产展示链路的修复方向，不扩展到钱包、合同、装配等其他 Info 页面。
- 实现时必须保持后端分层：`router -> handler -> service -> repository -> model`。
- 所有新增前端文案必须同步更新中英文资源。
