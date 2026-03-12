# ESI 新功能开发流程指南

本文档以「角色装配 (Fittings)」功能为完整示例，说明在 AmiyaEden 项目中新增一个 ESI 数据模块的全部步骤。

---

## 目录

1. [总览：文件清单](#1-总览文件清单)
2. [Step 1：定义数据模型](#2-step-1定义数据模型)
3. [Step 2：注册模型别名 & 数据库迁移](#3-step-2注册模型别名--数据库迁移)
4. [Step 3：编写 ESI 刷新任务](#4-step-3编写-esi-刷新任务)
5. [Step 4：编写 Repository 层](#5-step-4编写-repository-层)
6. [Step 5：编写 Service 层](#6-step-5编写-service-层)
7. [Step 6：编写 Handler 层](#7-step-6编写-handler-层)
8. [Step 7：注册路由](#8-step-7注册路由)
9. [Step 8：前端 API & 类型定义](#9-step-8前端-api--类型定义)
10. [Step 9：前端路由 & i18n](#10-step-9前端路由--i18n)
11. [Step 10：前端页面组件](#11-step-10前端页面组件)
12. [Step 11：菜单种子数据 & 角色权限](#12-step-11菜单种子数据--角色权限)
13. [Step 12：ESI Scope 检查](#13-step-12esi-scope-检查)
14. [注意事项](#14-注意事项)

---

## 1. 总览：文件清单

一个完整的 ESI 功能涉及以下文件（以 `fittings` 为例）：

| 层级 | 文件路径 | 动作 |
|------|---------|------|
| **Model** | `server/internal/model/esi/fittings.go` | 新建 |
| **别名** | `server/internal/model/esi_data.go` | 修改 |
| **DB 迁移** | `server/bootstrap/db.go` | 修改 |
| **ESI 任务** | `server/pkg/eve/esi/task_fittings.go` | 新建 |
| **Repository** | `server/internal/repository/fittings.go` | 新建 |
| **Service** | `server/internal/service/fittings.go` | 新建 |
| **Handler** | `server/internal/handler/fittings.go` | 新建 |
| **Router** | `server/internal/router/router.go` | 修改 |
| **Scope** | `server/config/scopes.json` | 修改（如需新 scope） |
| **菜单种子** | `server/internal/model/menu.go` | 修改 |
| **前端 API** | `static/src/api/eve-info.ts` | 修改 |
| **前端类型** | `static/src/types/api/api.d.ts` | 修改 |
| **前端路由** | `static/src/router/modules/info.ts` | 修改 |
| **i18n zh** | `static/src/locales/langs/zh.json` | 修改 |
| **i18n en** | `static/src/locales/langs/en.json` | 修改 |
| **前端页面** | `static/src/views/info/fittings/index.vue` | 新建 |

---

## 2. Step 1：定义数据模型

在 `server/internal/model/esi/` 下新建模型文件。

**文件**: `server/internal/model/esi/fittings.go`

```go
package esi

// EveCharacterFitting 角色装配主表
type EveCharacterFitting struct {
    ID          uint  `gorm:"primaryKey;autoIncrement"`
    FittingID   int64 `gorm:"index;uniqueIndex:idx_fitting_char"`
    CharacterID int64 `gorm:"index;uniqueIndex:idx_fitting_char"`
    Name        string
    ShipTypeID  int64
    Description string
}

func (EveCharacterFitting) TableName() string { return "eve_character_fittings" }

// EveCharacterFittingItem 装配物品详情表
type EveCharacterFittingItem struct {
    ID          uint  `gorm:"primaryKey;autoIncrement"`
    FittingID   int64 `gorm:"index"`
    CharacterID int64 `gorm:"index"`
    TypeID      int64
    Quantity    int
    Flag        string `gorm:"size:50"`
}

func (EveCharacterFittingItem) TableName() string { return "eve_character_fitting_items" }
```

**设计要点**：
- 主从表拆分：一对多关系的数据（如装配-物品、KM-物品）使用独立表，方便查询和维护
- `uniqueIndex` 保证同一角色下 fitting_id 唯一
- 所有 ESI 模型放在 `model/esi/` 子包中

---

## 3. Step 2：注册模型别名 & 数据库迁移

### 2a. 类型别名

**文件**: `server/internal/model/esi_data.go`

在 `import` 块中已有 `esimodel "amiya-eden/internal/model/esi"`，添加类型别名：

```go
type EveCharacterFitting     = esimodel.EveCharacterFitting
type EveCharacterFittingItem = esimodel.EveCharacterFittingItem
```

> **为什么需要别名？** `model/esi/` 是子包，其他层（repository、service）统一通过 `model.XXX` 引用，避免混乱的包路径。

### 2b. 数据库自动迁移

**文件**: `server/bootstrap/db.go`

在 `autoMigrate()` 函数的模型列表中追加：

```go
&model.EveCharacterFitting{},
&model.EveCharacterFittingItem{},
```

---

## 4. Step 3：编写 ESI 刷新任务

ESI 定时刷新任务负责定期从 CCP 服务器拉取数据并写入本地数据库。

**文件**: `server/pkg/eve/esi/task_fittings.go`

```go
package esi

import (
    "amiya-eden/global"
    "amiya-eden/internal/model"
    "context"
    "fmt"
    "time"
    "go.uber.org/zap"
)

func init() {
    Register(&FittingsTask{})
}

type FittingsTask struct{}

func (t *FittingsTask) Name() string        { return "character_fittings" }
func (t *FittingsTask) Description() string { return "角色装配" }
func (t *FittingsTask) Priority() Priority  { return PriorityNormal }

func (t *FittingsTask) Interval() RefreshInterval {
    return RefreshInterval{
        Active:   6 * time.Hour,
        Inactive: 48 * time.Hour,
    }
}

func (t *FittingsTask) RequiredScopes() []TaskScope {
    return []TaskScope{
        {Scope: "esi-fittings.read_fittings.v1", Description: "读取角色装配"},
    }
}

func (t *FittingsTask) Execute(ctx context.Context, taskCtx *TaskContext) error {
    path := fmt.Sprintf("/characters/%d/fittings/", taskCtx.CharacterID)

    var raw []struct { /* ESI 响应结构 */ }
    if err := taskCtx.Client.Get(ctx, path, taskCtx.AccessToken, &raw); err != nil {
        return err
    }

    db := global.DB
    tx := db.Begin()

    // 1. 删除旧数据（先 items 再主表）
    tx.Where("character_id = ?", taskCtx.CharacterID).Delete(&model.EveCharacterFittingItem{})
    tx.Where("character_id = ?", taskCtx.CharacterID).Delete(&model.EveCharacterFitting{})

    // 2. 批量插入新数据
    for _, f := range raw {
        fitting := model.EveCharacterFitting{
            FittingID:   f.FittingID,
            CharacterID: taskCtx.CharacterID,
            Name:        f.Name,
            ShipTypeID:  f.ShipTypeID,
            Description: f.Description,
        }
        if err := tx.Create(&fitting).Error; err != nil {
            tx.Rollback()
            return err
        }

        // 批量插入物品
        // ...
    }

    return tx.Commit().Error
}
```

**任务系统要点**：
- 实现 `RefreshTask` 接口：`Name/Description/Priority/Interval/RequiredScopes/Execute`
- 在 `init()` 中调用 `Register()` 自动注册
- `TaskContext` 包含 `CharacterID`、`AccessToken`、`Client`、`IsActive`
- `Interval` 区分活跃/不活跃用户的刷新频率
- 使用事务保证数据一致性：先删旧数据再插入新数据

---

## 5. Step 4：编写 Repository 层

**文件**: `server/internal/repository/fittings.go`

```go
package repository

import (
    "amiya-eden/global"
    "amiya-eden/internal/model"
)

type FittingsRepository struct{}

func NewFittingsRepository() *FittingsRepository {
    return &FittingsRepository{}
}

func (r *FittingsRepository) ListByCharacterIDs(ids []int64) ([]model.EveCharacterFitting, error) {
    var list []model.EveCharacterFitting
    err := global.DB.Where("character_id IN ?", ids).Find(&list).Error
    return list, err
}

func (r *FittingsRepository) GetItemsByCharacterIDs(ids []int64) ([]model.EveCharacterFittingItem, error) {
    var items []model.EveCharacterFittingItem
    err := global.DB.Where("character_id IN ?", ids).Find(&items).Error
    return items, err
}

// SaveFitting 事务保存（主表+物品表）
func (r *FittingsRepository) SaveFitting(f *model.EveCharacterFitting, items []model.EveCharacterFittingItem) error {
    return global.DB.Transaction(func(tx *gorm.DB) error {
        if err := tx.Create(f).Error; err != nil {
            return err
        }
        if len(items) > 0 {
            return tx.Create(&items).Error
        }
        return nil
    })
}

// DeleteFitting 事务删除（先 items 再主表）
func (r *FittingsRepository) DeleteFitting(fittingID, characterID int64) error {
    return global.DB.Transaction(func(tx *gorm.DB) error {
        tx.Where("fitting_id = ? AND character_id = ?", fittingID, characterID).
            Delete(&model.EveCharacterFittingItem{})
        return tx.Where("fitting_id = ? AND character_id = ?", fittingID, characterID).
            Delete(&model.EveCharacterFitting{}).Error
    })
}
```

**约定**：
- Repository 只负责数据库 CRUD
- 所有方法通过 `global.DB` 获取数据库连接
- 涉及多表操作用事务

---

## 6. Step 5：编写 Service 层

**文件**: `server/internal/service/fittings.go`

```go
package service

type FittingsService struct {
    charRepo    *repository.EveCharacterRepository
    fittingRepo *repository.FittingsRepository
    sdeRepo     *repository.SdeRepository
}

func NewFittingsService() *FittingsService { /* ... */ }
```

**Service 层职责**：
1. 校验角色归属（`validateCharacterOwnership`）
2. 调用 Repository 获取数据
3. 调用 SDE Repository 翻译 TypeID → 本地化名称
4. 组装响应结构（分组、排序等业务逻辑）
5. 如需调用 ESI API（如创建/删除装配），直接使用 `net/http`

**⚠️ 重要：禁止在 Service 层 import `pkg/eve/esi`**

`pkg/eve/esi/queue.go` 已经 import 了 `internal/service`，如果 Service 再 import `esi` 会造成 **import cycle**。需要 ESI HTTP 调用时，直接用 `net/http` + `encoding/json`：

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
    "time"
)

httpClient := &http.Client{Timeout: 30 * time.Second}
postURL := fmt.Sprintf("https://esi.evetech.net/characters/%d/fittings/", characterID)
req, _ := http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewReader(bodyBytes))
req.Header.Set("Authorization", "Bearer "+accessToken)
req.Header.Set("Content-Type", "application/json")
resp, err := httpClient.Do(req)
```

**请求/响应结构体定义在 Service 文件中**（不单独建 DTO 文件）。

---

## 7. Step 6：编写 Handler 层

**文件**: `server/internal/handler/fittings.go`

```go
package handler

import (
    "amiya-eden/internal/middleware"
    "amiya-eden/internal/service"
    "amiya-eden/pkg/response"
    "github.com/gin-gonic/gin"
)

type FittingsHandler struct {
    svc *service.FittingsService
}

func NewFittingsHandler() *FittingsHandler {
    return &FittingsHandler{svc: service.NewFittingsService()}
}

func (h *FittingsHandler) GetFittings(c *gin.Context) {
    userID := middleware.GetUserID(c)
    var req service.FittingsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Fail(c, "参数错误: "+err.Error())
        return
    }
    result, err := h.svc.GetFittings(userID, &req)
    if err != nil {
        response.Fail(c, err.Error())
        return
    }
    response.OK(c, result)
}
```

**Handler 层约定**：
- 通过 `middleware.GetUserID(c)` 获取当前用户 ID
- `c.ShouldBindJSON(&req)` 绑定请求体
- `response.OK(c, data)` / `response.Fail(c, msg)` 统一响应

---

## 8. Step 7：注册路由

**文件**: `server/internal/router/router.go`

在 `auth` 分组的 `info` 子分组中添加路由：

```go
// ─── 装配 ───
fittingsH := handler.NewFittingsHandler()
info.POST("/fittings", fittingsH.GetFittings)
info.POST("/fittings/save", fittingsH.SaveFitting)
```

**路由约定**：
- 查询类接口用 `POST`（因为请求体可能较复杂）
- 路径风格：`/api/v1/info/{resource}`
- 需要登录的接口放在 `auth` 分组下

---

## 9. Step 8：前端 API & 类型定义

### 8a. API 函数

**文件**: `static/src/api/eve-info.ts`

```typescript
/** 获取用户所有角色的装配列表 */
export function fetchInfoFittings(data: Api.EveInfo.FittingsRequest) {
  return request.post<Api.EveInfo.FittingsListResponse>({ url: '/api/v1/info/fittings', data })
}

/** 保存装配 */
export function saveInfoFitting(data: Api.EveInfo.SaveFittingRequest) {
  return request.post<Api.EveInfo.FittingResponse>({ url: '/api/v1/info/fittings/save', data })
}
```

### 8b. TypeScript 类型

**文件**: `static/src/types/api/api.d.ts`

在 `namespace EveInfo` 中追加接口定义，与后端 Service 层的 Response 结构体一一对应：

```typescript
namespace EveInfo {
  // ... 已有类型 ...

  interface FittingsRequest {
    language?: string
  }

  interface FittingItemResponse {
    type_id: number
    type_name: string
    quantity: number
    flag: string
  }

  interface FittingSlotGroup {
    flag_name: string
    flag_text: string
    order_id: number
    items: FittingItemResponse[]
  }

  interface FittingResponse {
    fitting_id: number
    character_id: number
    name: string
    description: string
    ship_type_id: number
    ship_name: string
    group_id: number
    group_name: string
    race_id: number
    race_name: string
    slots: FittingSlotGroup[]
  }

  interface FittingsListResponse {
    total: number
    fittings: FittingResponse[]
  }

  interface SaveFittingRequest {
    character_id: number
    fitting_id?: number
    name: string
    description?: string
    ship_type_id: number
    items: { type_id: number; quantity: number; flag: string }[]
  }
}
```

---

## 10. Step 9：前端路由 & i18n

### 9a. 路由模块

**文件**: `static/src/router/modules/info.ts`

在 `children` 数组中添加：

```typescript
{
  path: 'fittings',
  name: 'EveInfoFittings',
  component: '/info/fittings',
  meta: { title: 'menus.info.fittings', keepAlive: true }
}
```

> `name` 必须与 `menu.go` 种子数据中的 `Name` 字段完全一致。

### 9b. i18n 国际化

**文件**: `static/src/locales/langs/zh.json`

```json
// menus 部分（导航栏标题）
"menus": {
  "info": {
    "fittings": "我的装配"
  }
}

// info 部分（页面内文案）
"info": {
  "fittingsTitle": "我的装配",
  "searchFitting": "搜索装配",
  "noFittingData": "暂无装配数据，请等待数据刷新",
  "fittingDetail": "装配详情",
  "fittingCount": "装配数量"
}
```

**文件**: `static/src/locales/langs/en.json` — 同理添加英文翻译。

---

## 11. Step 10：前端页面组件

**文件**: `static/src/views/info/fittings/index.vue`

组件模式参考：

| 参考组件 | 用途 |
|---------|------|
| `views/info/ships/index.vue` | 角色选择器、筛选栏、分组展示（Group → Race → Ship 网格） |
| `components/business/KmPreviewDialog.vue` | 物品详情弹窗（按槽位分组显示物品图标+名称+数量） |

**典型组件结构**：

```vue
<template>
  <div class="info-xxx-page art-full-height">
    <!-- 1. 顶栏：角色选择 / 刷新按钮 / 统计 -->
    <ElCard shadow="never" class="mb-2">...</ElCard>

    <!-- 2. 主体：筛选栏 + 分组数据 -->
    <div v-loading="loading" class="xxx-main">
      <div class="filter-bar">...</div>
      <div class="xxx-groups">
        <div v-for="grp in groupedData" :key="grp.name" class="market-group-section">
          <div class="market-group-header" @click="toggleGroup(grp.name)">...</div>
          <div v-if="!collapsed" class="xxx-grid">
            <div v-for="item in grp.items" class="xxx-card" @click="openDetail(item)">
              <img :src="eveImageURL(item.type_id)" />
              <span>{{ item.name }}</span>
            </div>
          </div>
        </div>
      </div>
      <ElEmpty v-else-if="!loading" />
    </div>

    <!-- 3. 详情弹窗 -->
    <ElDialog v-model="detailVisible">...</ElDialog>
  </div>
</template>
```

**常用模式**：
- `defineOptions({ name: 'EveInfoFittings' })` — 组件名与路由 name 一致
- `useUserStore().language` — 获取当前语言
- `fetchMyCharacters()` → ElSelect 角色切换
- `computed` 筛选 + 分组
- EVE 图标：`https://images.evetech.net/types/${typeId}/icon?size=64`

---

## 12. Step 11：菜单种子数据 & 角色权限

**文件**: `server/internal/model/menu.go`

### 11a. 添加菜单种子

在 `GetSystemMenuSeeds()` 的 `EVE 角色信息` 区块中添加：

```go
{ParentName: "EveInfo", Menu: Menu{
    Type: MenuTypeMenu, Name: "EveInfoFittings",
    Path: "fittings", Component: "/info/fittings",
    Title: "menus.info.fittings", Sort: 50,
    KeepAlive: true, Status: 1,
}},
```

> `Name` 字段必须与前端路由的 `name` 完全一致。`Sort` 值越大越靠前。

### 11b. 添加到角色权限

在 `DefaultRoleMenuMap()` 中，将 `"EveInfoFittings"` 添加到所有需要此功能的角色列表中：

```go
RoleAdmin: { ..., "EveInfoFittings", ... },
RoleFC:    { ..., "EveInfoFittings", ... },
RoleUser:  { ..., "EveInfoFittings", ... },
RoleGuest: { ..., "EveInfoFittings", ... },
```

---

## 13. Step 12：ESI Scope 检查

**文件**: `server/config/scopes.json`

确认所需的 ESI scope 已在列表中。如果是新 scope，追加到 JSON 数组：

```json
["...", "esi-fittings.read_fittings.v1", "esi-fittings.write_fittings.v1", "..."]
```

> 添加新 scope 后，**已绑定的角色需要重新 SSO 授权**才能获取新权限。

---

## 14. 注意事项

### Import Cycle 问题

```
pkg/eve/esi/queue.go → imports → internal/service
internal/service/xxx.go → imports → pkg/eve/esi  ← ❌ 循环依赖
```

**解决方案**：Service 层中需要调用 ESI API 时，使用 `net/http` 直接发 HTTP 请求，不要 import `pkg/eve/esi`。

### ESI POST 返回 201

ESI 创建类接口（如 POST fittings）返回 `201 Created` 而非 `200 OK`，注意状态码判断：

```go
if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
    return fmt.Errorf("ESI error %d", resp.StatusCode)
}
```

### SDE 翻译

使用 `SdeRepository` 的方法进行 TypeID → 名称翻译：

- `GetTypes(typeIDs, groupIDs, lang)` — 批量翻译物品
- `GetShipsByCategoryID(lang)` — 获取所有舰船（含 raceID）
- `GetAllRaces()` — 获取种族列表
- `GetNames(ids, category, lang)` — 通用名称翻译

### 开发验证清单

- [ ] `go build ./...` 通过（无 import cycle）
- [ ] 数据库表自动创建
- [ ] ESI 定时刷新任务正常注册（查看日志）
- [ ] API 接口返回正确数据
- [ ] 前端页面正常渲染
- [ ] 菜单在各角色中正确显示
- [ ] i18n 中英文切换正常
