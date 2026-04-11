---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-04-11
source_of_truth:
  - server/internal/router/router.go
  - server/internal/service/hall_of_fame.go
  - static/src/api/hall-of-fame.ts
  - static/src/views/hall-of-fame
---

# 伏羲中心 / Fuxi Center

## 当前能力

- 提供一个独立于用户/人物主数据的名人堂模块，管理员可独立维护英雄条目
- 伏羲名人堂页面向所有 `Login` 用户展示名人堂画布、背景图与全部可见卡片
- 编辑名人堂页面向 `admin` 开放，支持新增、编辑、删除、拖拽摆放、修改卡片尺寸，以及编辑卡片文案、人物 ID、主题、称号颜色、字体大小和头像下方小图
- 当前伏羲管理页面向已登录用户开放，管理员可在页内直接编辑
- 管理页支持上传主殿背景图，直接保存为 base64 data URL，不落盘
- 卡片样式支持 `gold`、`silver`、`darkred`、`yellow`、`bronze`、`rose`、`jade`、`midnight`、`custom` 九种主题；自定义主题允许单独配置背景色、文字色和边框色
- 卡片支持独立于颜色主题的 `border_style` 选择，可选 `none` 与 8 种 decorative SVG frame overlays；未选择装饰边框时继续使用 `border-color` 作为回退
- 画布尺寸可由管理员调整；卡片坐标按百分比持久化，便于不同尺寸画布保持相对布局
- 伏羲名人堂页面不再显示顶部标题区，画布按真实尺寸渲染并支持水平滚动浏览；卡片支持悬浮发光效果
- 编辑名人堂页画布支持缩放编辑，便于在超大画布上精细摆放卡片
- 编辑名人堂页预览会把当前未保存的本地草稿写入浏览器存储，再在新标签页中按草稿内容渲染伏羲名人堂预览
- 卡片改为更紧凑的横向信息布局，默认尺寸更矮，更适合在同一画布中摆放大量卡片

## 入口

### 前端页面

- `static/src/views/hall-of-fame/temple` — 伏羲名人堂页（所有已登录用户）
- `static/src/views/hall-of-fame/manage` — 编辑名人堂页（管理员）
- `static/src/views/hall-of-fame/current-manage` — 当前伏羲管理页（所有已登录用户，管理员可编辑）

### 后端路由

用户端：

- `GET /api/v1/hall-of-fame/temple` — 获取主殿配置与可见卡片

管理端：

- `GET /api/v1/system/hall-of-fame/config`
- `PUT /api/v1/system/hall-of-fame/config`
- `POST /api/v1/system/hall-of-fame/upload-background`
- `GET /api/v1/system/hall-of-fame/cards`
- `POST /api/v1/system/hall-of-fame/cards`
- `PUT /api/v1/system/hall-of-fame/cards/batch-layout`
- `PUT /api/v1/system/hall-of-fame/cards/:id`
- `DELETE /api/v1/system/hall-of-fame/cards/:id`

## 权限边界

- 左侧菜单根节点 `伏羲中心 / Fuxi Center` 对所有 `Login` 用户可见
- `伏羲名人堂 / Fuxi Hall of Heroes` 子页面要求 `login: true`
- `编辑名人堂 / Edit Hall of Heroes` 子页面要求 `roles: ['super_admin', 'admin']`
- `当前伏羲管理 / Current Fuxi Management` 子页面要求 `login: true`，页内编辑控件仍由管理员角色控制
- 后端管理接口统一挂在 `/system/hall-of-fame/*` 下，由 `admin` 路由组保护

## 数据模型

### `hall_of_fame_config`

- 单例配置表，保存：
  - `background_image`
  - `canvas_width`
  - `canvas_height`
- 若首次读取时不存在记录，服务层会自动创建默认值（1920×1080）

### `hall_of_fame_card`

- 每张卡片包含：
  - 基础文案：`name`、`title`、`description`
  - 人物：`character_id`（前端按 EVE 图片服务拼接头像 URL）
  - 附加图像：`badge_image`（头像下方的小图，当前管理端限制上传文件不超过 300KB）
  - 布局：`pos_x`、`pos_y`、`width`、`height`、`z_index`
  - 视觉：`style_preset`、`custom_bg_color`、`custom_text_color`、`custom_border_color`、`border_style`、`title_color`、`font_size`
  - 显示控制：`visible`
- 采用软删除

## 关键不变量

- 编辑名人堂页可以读取全部卡片；伏羲名人堂页只返回 `visible = true` 的卡片
- 画布配置始终存在；服务层负责 get-or-create 默认单例
- 背景上传最大 5MB，仅允许 `jpeg/png/webp`
- 卡片坐标在服务层与前端拖拽辅助逻辑中都会被约束在 `0–100` 范围
- 卡片布局保存走批量接口，避免拖拽时逐条请求
- 当前编辑名人堂 UI 不再暴露 `visible` 与 `z_index` 手动编辑项，但兼容既有数据字段

## 主要代码文件

- `server/internal/model/hall_of_fame.go`
- `server/internal/repository/hall_of_fame.go`
- `server/internal/service/hall_of_fame.go`
- `server/internal/handler/hall_of_fame.go`
- `server/internal/router/router.go`
- `static/src/api/hall-of-fame.ts`
- `static/src/router/modules/hall-of-fame.ts`
- `static/src/types/api/api.d.ts` (`Api.HallOfFame`)
- `static/src/views/hall-of-fame/`
