---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-04-12
---

# 伏羲管理名录

## 当前能力

- 对外展示伏羲管理人员名录，按可配置的层级分组展示
- 所有登录用户都可以查看
- 管理员可在页面内直接维护层级和名录内容
- 初次加载失败时，页面显示错误状态而不是空态

## 数据模型

- `fuxi_admin_config` - 单例配置：`base_font_size`（8–32 px）、`card_width`、`page_background_color`、`card_background_color`、`card_border_color`、`tier_title_color`、`name_text_color`、`body_text_color`
- `fuxi_admin_tier` - 具名层级，包含 `sort_order`；按 `sort_order ASC, id ASC` 排序
- `fuxi_admin` - 单个管理人员条目；字段包括 `tier_id`、`nickname`、`character_name`、`description`、`contact_qq`、`contact_discord`、`character_id`

删除某个层级时，会级联删除该层级下的所有管理人员。

## 页面访问

路由：`/hall-of-fame/current-manage`

- 查看：所有登录用户均可访问，不要求管理权限
- 编辑控件：仅 `admin` 或 `super_admin` 可见

## 管理能力

- 新增、重命名、删除层级
- 新增、编辑、删除管理人员卡片
- 为管理人员卡片补充简介信息
- 为卡片设置 `character_id`，显示 EVE 角色头像
- 卡片编辑表单的实际字段已改为 `nickname`、`character_name`，并将 `tier_id` 的选择项显示为 `职位`
- 卡片视图中的 QQ 和 Discord 联系方式旁提供共享内联复制按钮，便于快速复制联系方式
- 调整页面背景、卡片背景、卡片边框，以及层级标题、昵称、其他文字的颜色
- 颜色配置仅接受十六进制颜色值，前端颜色选择器禁用 alpha 通道
- 调整卡片固定宽度
- 调整全局 `base_font_size` 作为主字号；标题、简介、联系方式会在 CSS 中按层次缩放，并允许长简介自动换行撑高卡片

## API

| 方法   | 路径                                 | 认证     | 说明                                     |
| ------ | ------------------------------------ | -------- | ---------------------------------------- |
| GET    | /api/v1/fuxi-admins                  | 登录用户 | 已登录用户名录（配置 + 层级 + 管理人员） |
| GET    | /api/v1/system/fuxi-admins/config    | admin    | 获取配置                                 |
| PUT    | /api/v1/system/fuxi-admins/config    | admin    | 更新 `base_font_size`                    |
| GET    | /api/v1/system/fuxi-admins/tiers     | admin    | 获取层级列表                             |
| POST   | /api/v1/system/fuxi-admins/tiers     | admin    | 创建层级                                 |
| PUT    | /api/v1/system/fuxi-admins/tiers/:id | admin    | 更新层级                                 |
| DELETE | /api/v1/system/fuxi-admins/tiers/:id | admin    | 删除层级（级联）                         |
| POST   | /api/v1/system/fuxi-admins           | admin    | 创建管理人员                             |
| PUT    | /api/v1/system/fuxi-admins/:id       | admin    | 更新管理人员                             |
| DELETE | /api/v1/system/fuxi-admins/:id       | admin    | 删除管理人员                             |
