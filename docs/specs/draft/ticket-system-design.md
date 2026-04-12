---
status: draft
doc_type: spec
owner: engineering
last_reviewed: 2026-04-10
---

# 工单系统设计方案

## 当前状态

- 已实现：
- 未实现：全部功能

## 背景

当前平台缺乏系统化的工单管理机制，成员在遇到问题或需要帮助时缺乏规范的提交流程，管理员处理反馈和协调问题也缺乏统一的管理界面。工单系统需要提供成员提交问题、管理员处理跟踪、分类管理、状态流转等基础能力，并与现有用户体系、权限体系、通知体系整合。

## 提案内容

### 核心功能

#### 成员侧功能

- **提交工单**
  - 工单标题（必填）
  - 工单描述（必填，支持多行文本）
  - 工单分类（下拉选择，由管理员配置）
  - 优先级（可选：低、中、高）
  - 附件支持（图片上传，可选）
  - 提交后默认状态为"待处理"

- **我的工单列表**
  - 分页展示当前用户提交的所有工单
  - 显示工单标题、分类、状态、优先级、创建时间、最后更新时间
  - 支持按状态筛选
  - 点击进入工单详情

- **工单详情**
  - 显示完整工单信息
  - 显示处理人（如有）、处理时间
  - 显示所有回复记录（用户和管理员消息）
  - 支持用户追加回复或补充信息
  - 显示工单状态变更历史

#### 管理员侧功能

- **工单管理列表**
  - 分页展示所有工单
  - 显示工单ID、标题、提交人、分类、状态、优先级、创建时间、最后更新时间
  - 支持按状态、分类、提交人筛选
  - 支持搜索标题或描述内容
  - 待处理工单数量徽标
  - 点击进入工单详情处理

- **工单详情与处理**
  - 查看完整工单信息
  - 显示提交人信息：昵称、主人物名、QQ、Discord ID
  - 修改工单状态：待处理 → 处理中 → 已完成
  - 修改工单优先级
  - 管理员回复功能
  - 标记处理人（自动记录当前处理管理员）
  - 工单转分类
  - 工单关闭（无需回复即可关闭）

- **工单分类管理**
  - 创建、编辑、删除工单分类
  - 分类名称（中英双语）
  - 分类描述（可选）
  - 排序
  - 启用/禁用状态
  - 仅 super_admin 和 admin 可访问

- **工单统计看板**
  - 工单总数、待处理数、处理中数、已完成数
  - 各分类工单分布
  - 近7天/30天工单趋势
  - 管理员处理量统计

### 数据模型设计

#### ticket

```go
type Ticket struct {
    ID              uint      `gorm:"primarykey"`
    UserID          uint      `gorm:"not null;index"`
    CategoryID      uint      `gorm:"not null;index"`
    Title           string    `gorm:"size:200;not null"`
    Description     string    `gorm:"type:text;not null"`
    Status          string    `gorm:"size:20;not null;default:'pending';index"`
    Priority        string    `gorm:"size:20;not null;default:'medium'"`
    HandledBy       uint      `gorm:"index"`
    HandledAt       *time.Time
    ClosedAt        *time.Time
    AttachmentURL   string    `gorm:"size:500"`
    CreatedAt       time.Time `gorm:"autoCreateTime"`
    UpdatedAt       time.Time `gorm:"autoUpdateTime"`
    DeletedAt       gorm.DeletedAt `gorm:"index"`

    User            User      `gorm:"foreignKey:UserID"`
    Category        TicketCategory `gorm:"foreignKey:CategoryID"`
    Handler         User      `gorm:"foreignKey:HandledBy"`
    Replies         []TicketReply `gorm:"foreignKey:TicketID"`
    StatusHistories []TicketStatusHistory `gorm:"foreignKey:TicketID"`
}
```

**状态枚举**：
- `pending`: 待处理
- `in_progress`: 处理中
- `completed`: 已完成

**优先级枚举**：
- `low`: 低
- `medium`: 中
- `high`: 高

#### ticket_category

```go
type TicketCategory struct {
    ID          uint      `gorm:"primarykey"`
    Name        string    `gorm:"size:50;not null;uniqueIndex"`
    NameEN      string    `gorm:"size:50;not null;uniqueIndex"`
    Description string    `gorm:"size:200"`
    SortOrder   int       `gorm:"not null;default:0"`
    Enabled     bool      `gorm:"not null;default:true"`
    CreatedAt   time.Time `gorm:"autoCreateTime"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime"`
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}
```

#### ticket_reply

```go
type TicketReply struct {
    ID          uint      `gorm:"primarykey"`
    TicketID    uint      `gorm:"not null;index"`
    UserID      uint      `gorm:"not null;index"`
    Content     string    `gorm:"type:text;not null"`
    IsInternal  bool      `gorm:"not null;default:false"`
    CreatedAt   time.Time `gorm:"autoCreateTime"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime"`
    DeletedAt   gorm.DeletedAt `gorm:"index"`

    Ticket      Ticket    `gorm:"foreignKey:TicketID"`
    User        User      `gorm:"foreignKey:UserID"`
}
```

`IsInternal` 标记管理员内部备注，不向提交人展示。

#### ticket_status_history

```go
type TicketStatusHistory struct {
    ID          uint      `gorm:"primarykey"`
    TicketID    uint      `gorm:"not null;index"`
    FromStatus  string    `gorm:"size:20"`
    ToStatus    string    `gorm:"size:20;not null"`
    ChangedBy   uint      `gorm:"not null"`
    ChangedAt   time.Time `gorm:"autoCreateTime"`

    Ticket      Ticket    `gorm:"foreignKey:TicketID"`
    ChangedByUser User     `gorm:"foreignKey:ChangedBy"`
}
```

### API 设计

#### 成员侧 API

```
POST   /api/v1/ticket/tickets
GET    /api/v1/ticket/tickets/me
GET    /api/v1/ticket/tickets/:id
POST   /api/v1/ticket/tickets/:id/replies
GET    /api/v1/ticket/tickets/:id/replies
GET    /api/v1/ticket/categories
```

#### 管理员侧 API

```
GET    /api/v1/system/ticket/tickets
GET    /api/v1/ticket/tickets/:id
PUT    /api/v1/ticket/tickets/:id/status
PUT    /api/v1/ticket/tickets/:id/priority
POST   /api/v1/ticket/tickets/:id/replies
GET    /api/v1/ticket/tickets/:id/replies
GET    /api/v1/ticket/tickets/:id/status-history

GET    /api/v1/system/ticket/categories
POST   /api/v1/system/ticket/categories
PUT    /api/v1/system/ticket/categories/:id
DELETE /api/v1/system/ticket/categories/:id

GET    /api/v1/system/ticket/statistics
```

### 权限边界

- **提交工单**：`Login`（任意已登录非 guest 用户）
- **我的工单**：`Login`（只能查看自己的工单）
- **工单回复**：`Login`（只能回复自己的工单）
- **工单管理**：`admin` 或 `super_admin`
- **分类管理**：`admin` 或 `super_admin`
- **统计看板**：`admin` 或 `super_admin`

### 前端路由设计

#### 成员侧路由

```
/ticket/my-tickets          # 我的工单列表
/ticket/create              # 提交工单
/ticket/detail/:id          # 工单详情
```

#### 管理员侧路由

```
/system/ticket-management   # 工单管理
/system/ticket-categories   # 工单分类管理
/system/ticket-statistics   # 工单统计看板
```

### 通知机制

- **工单创建**：向 super_admin 和 admin 角色用户发送通知
- **状态变更**：向工单提交人发送通知
- **新回复**：向对方（提交人或管理员）发送通知
- **通知渠道**：
  - 系统内通知（基于现有通知机制）
  - 可选：邮件通知（如果用户配置了邮箱）
  - 可选：游戏内邮件通知（通过 ESI）

### 国际化要求

所有用户界面文本需要中英双语支持，更新以下文件：
- `static/src/locales/langs/zh.json`
- `static/src/locales/langs/en.json`

### 默认分类配置

系统初始化时创建默认工单分类：

1. 账号问题 / Account Issues
2. 舰船装备问题 / Ship & Equipment
3. 游戏操作问题 / Gameplay Issues
4. 平台功能建议 / Platform Feedback
5. 其他问题 / Other Issues

### 前端组件设计

#### 通用组件

- `TicketStatusBadge`: 工单状态徽标
- `TicketPriorityBadge`: 优先级徽标
- `TicketCategoryTag`: 分类标签
- `TicketReplyItem`: 回复条目组件

#### 页面组件

- `TicketListPage`: 工单列表页面
- `TicketCreateForm`: 创建工单表单
- `TicketDetailPage`: 工单详情页面
- `TicketManagementPage`: 工单管理页面
- `TicketCategoryManagePage`: 分类管理页面
- `TicketStatisticsPage`: 统计看板页面

## 未决问题

1. 附件存储方式：使用本地存储还是对象存储（S3/MinIO）？
2. 通知机制优先级：是否所有通知都需要推送，还是仅重要通知？
3. 工单关闭后是否允许重新开启？
4. 是否需要工单评分机制（用户对处理结果评分）？
5. 是否需要工单模板功能（预设常用问题描述）？
6. 是否需要工单转派功能（管理员之间转派）？
7. 工单保留期限：已完成工单是否需要定期归档或清理？

## 明确声明

- 本文档是提案，不代表当前已实现行为
- 不能覆盖 `docs/ai/repo-rules.md`、`docs/architecture/`、`docs/api/`、`docs/features/current/`
- 具体实现细节需遵循项目分层架构：router → middleware → handler → service → repository → model
- 所有用户界面文本必须支持中英双语
- 权限校验必须在后端严格执行，前端控制仅用于 UX

## 升级路径

当以下部分落地后，需要迁移到 `docs/features/current/`：

1. 数据库模型创建并初始化
2. 后端 API 实现完成
3. 前端页面实现完成
4. 基础功能测试通过
5. 文档更新与实际行为对齐

## 实施优先级建议

### 第一阶段（MVP）

1. 数据库模型创建
2. 工单提交功能
3. 我的工单列表与详情
4. 工单管理列表与详情
5. 基础状态流转（待处理 → 处理中 → 已完成）
6. 管理员回复功能
7. 默认分类配置

### 第二阶段（增强）

1. 工单分类管理
2. 优先级管理
3. 基础通知机制
4. 工单统计看板
5. 状态变更历史

### 第三阶段（优化）

1. 附件上传功能
2. 高级筛选与搜索
3. 工单模板功能
4. 工单评分机制
5. 工单转派功能
6. 高级通知配置

## 主要代码文件（预估）

- `server/internal/model/ticket.go` - 数据模型定义
- `server/internal/repository/ticket.go` - 数据访问层
- `server/internal/service/ticket_service.go` - 业务逻辑层
- `server/internal/service/ticket_category_service.go` - 分类业务逻辑
- `server/internal/handler/ticket_handler.go` - 成员侧 API
- `server/internal/handler/ticket_admin_handler.go` - 管理侧 API
- `server/internal/router/router.go` - 路由注册
- `server/bootstrap/db.go` - 数据库迁移
- `static/src/api/ticket.ts` - 前端 API 封装
- `static/src/types/api/ticket.d.ts` - TypeScript 类型定义
- `static/src/views/ticket/` - 成员侧页面组件
- `static/src/views/system/ticket-management/` - 管理侧页面组件
- `static/src/router/modules/ticket.ts` - 前端路由配置
- `static/src/components/ticket/` - 通用组件
- `static/src/locales/langs/zh.json` - 中文翻译
- `static/src/locales/langs/en.json` - 英文翻译
