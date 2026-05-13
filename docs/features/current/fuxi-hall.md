---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-05-12
source_of_truth:
  - server/internal/router/router.go
  - server/internal/handler/fuxi_hall.go
  - server/internal/service/fuxi_hall.go
  - static/src/router/modules/fuxi-hall.ts
  - static/src/views/fuxi-hall
---

# 伏羲大厅（fuxi-hall）

## 目标

以 `fuxi-hall` 模块替换旧的 `hall-of-fame` 与 `fuxi-admins` 实现，固定两组登录可见展示页：

- `leadership`（管理层）
- `contributors`（重大贡献成员）

管理员通过独立管理页维护页面配置与卡片内容。

## 路由与权限

### 登录可见

- `GET /api/v1/fuxi-hall/leadership`
- `GET /api/v1/fuxi-hall/contributors`

要求 `Login`（非 guest）。

### 管理端

- `GET /api/v1/system/fuxi-hall/pages/:page_key`
- `PUT /api/v1/system/fuxi-hall/pages/:page_key`
- `GET /api/v1/system/fuxi-hall/cards/:page_key`
- `POST /api/v1/system/fuxi-hall/cards`
- `PUT /api/v1/system/fuxi-hall/cards/:id`
- `PUT /api/v1/system/fuxi-hall/cards/reorder`
- `DELETE /api/v1/system/fuxi-hall/cards/:id`

要求 `RequireRole(admin)`。

## 后端规则

- `page_key` 仅允许 `leadership | contributors`
- 卡片必填：`nickname`、`main_character_id`、`main_character_name`、`title`
- `description_html` 在服务层白名单清洗后入库与返回
- `sort_order` 作为页面内手动排序主键，由 `reorder` 接口维护
- 样式约束：
  - `accent_color` 仅允许十六进制颜色值
  - `avatar_shape` 仅允许预设枚举值
  - `font_scale` 有上下限
  - `style_preset`、`badge_tone`、`cover_height` 已固定，不再通过接口暴露

## 前端结构

- 路由模块：`static/src/router/modules/fuxi-hall.ts`
- 页面：
  - `static/src/views/fuxi-hall/leadership/index.vue`
  - `static/src/views/fuxi-hall/contributors/index.vue`
  - `static/src/views/fuxi-hall/manage/index.vue`
- API：`static/src/api/fuxi-hall.ts`

管理页能力固定为：

- 页面标题/副标题/富文本说明编辑
- 卡片增删改
- 头像统一由 `main_character_id` 生成 ESI 人物头像（不支持自定义头像上传）
- 封面上传（复用 `/api/v1/upload/image`）
- 受控样式编辑（强调色/头像形状/字体大小）
- 显隐切换
- 手动排序（上移/下移）
- 管理态实时预览（未保存草稿即时映射）

不包含画布拖拽布局。

## 数据模型

- `fuxi_hall_page`
- `fuxi_hall_card`

旧 `hall_of_fame_*` 与 `fuxi_admin_*` 表不做迁移与兼容映射；新模块独立维护数据。
