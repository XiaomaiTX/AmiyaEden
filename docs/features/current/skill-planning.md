---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-10-12
source_of_truth:
  - server/internal/router/router.go
  - server/internal/service/skill_plan.go
  - static/src/api/generated/client.ts
  - static/src/views/skill-planning
---

# Skill Planning 模块

## 当前能力

- 军团技能计划的列表、详情、创建、编辑、删除
- 创建 / 编辑技能计划时可选一个舰船图标，并在列表中展示
- 粘贴技能文本解析为技能计划条目
- 管理员可拖拽调整当前页内的技能计划顺序，也可直接编辑 `sort_order` 做跨分页排序
- 军团与个人技能计划列表每页显示 16 张卡片，列表面板随当前页内容自然增高；桌面和窄屏均由页面承接纵向滚动，列表正常不产生内部滚动条，保留 overflow: auto 以在容器确实受限时提供滚动兜底，分页位于列表末尾。
- 通过独立顶级菜单“技能规划”进入技能计划页面
- 任意 `Login` 用户都可查看技能计划列表 / 详情；管理按钮只对可维护角色显示
- 用户可在”检查完成度”页面保存自己的人物选择和规划选择，并把人物技能与选中的军团规划逐项比对
- 用户可在 `/skill-planning/personal-skill-plans` 管理仅属于自己的个人技能规划（增删改查与排序）
- 完成度检查页面的“选择规划”会合并军团规划与当前用户个人规划，并在列表中显示来源前缀（军团/个人）
- 完成度检查页的缺失技能列表支持一键复制未训练的技能，复制内容采用技能文本格式；表头悬浮预览仍保持只读文本
- 规划选择默认包含全部规划，用户可取消选择不需要检查的规划，选择持久化保存

## 入口

### 前端页面

- 页面：
  - `static/src/views/skill-planning/skill-plans`
  - `static/src/views/skill-planning/completion-check`

### 后端契约

[技能规划 OpenAPI：军团与个人计划](../../../contracts/modules/skill-plans.yaml)。

## 权限边界

- `skill-plans` 的列表、详情查询要求 `Login`
- `skill-plans` 的创建、修改、删除、排序要求 `admin` 或 `senior_fc`（`super_admin` 仍会自动通过 `RequireRole`）
- `check/selection`、`check/plan-selection` 与 `check/run` 要求 `Login`
- 当前技能规划只承载军团技能计划，不与 EVE 人物技能查询页面混用
- 福利系统仅允许绑定军团技能计划，不允许绑定个人技能计划

## 关键不变量

- 技能规划是独立顶级导航，不再挂在舰队行动下
- 前端静态路由与后端 API 当前都归属 `SkillPlanning` 模块
- 页面实现位于 `static/src/views/skill-planning` 目录，修改时保持模块边界一致
- 技能计划列表按 `sort_order ASC, id DESC` 排序；当前页拖拽只重排该页已有排序区间，跨分页移动依赖显式 `sort_order`
- 完成度检查页面与技能计划列表页共享同一 `Login` 读权限边界，避免普通登录用户在检查页看到误导性的访问拒绝提示
- 人物选择和规划选择都会按用户持久化保存，用户再次进入”检查完成度”时不需要重新选择
- 完成度检查只允许比较当前用户自己绑定的人物
- 完成度检查只会比对用户选中的规划，未选中的规划不参与检查
- 完成度检查可见规划集合 = 全部军团规划 + 当前用户个人规划

## 主要代码文件

- `server/internal/service/skill_plan.go`
- `server/internal/router/router.go`
- `static/src/api/generated/client.ts`
- `static/src/router/modules/skill-planning.ts`
- `static/src/views/skill-planning/skill-plans`
- `static/src/views/skill-planning/personal-skill-plans`
- `static/src/views/skill-planning/completion-check`

## Compatibility Notes (2026-04-25)

- Skill plan `skills_text` parsing now accepts localized client export lines such as `<localized hint="Large Artillery Specialization">大型火炮专业研究*</localized> 4`.
- During save, parser normalization removes markup tags and trailing `*`, then resolves with the existing skill-name mapping and stores in the current concise structured format.

## 前端实现映射（迁移期）

- Vue 当前实现位于 `static/src`。
- React 已承接技能计划、个人技能计划和完成度检查页面；使用 [React 生成客户端](../../../static-react/src/api/generated/client.ts)中的操作与契约类型。
- 解析、排序和权限行为由本文定义，迁移不改变后端契约。

## 仪表盘管理查询

[军团技能达标查询](dashboard-skill-readiness.md) 在 Vue 与 React 仪表盘提供独立的 admin 查询页面，按军团筛选跨用户绑定角色并比较一个共享军团方案。复用当前等级判断口径，但不复用或放宽个人完成度检查的归属校验。该管理功能不读取或修改个人页面保存的角色、方案选择。

## EFT 导入与停用

公共和个人计划的 `skills_text` 同时支持技能列表和单份英文 EFT 装配文本。以 `[舰船, 装配名]` 开头时，服务端提取船体及全部所列物品，包含内联弹药、货舱探针和无人机。数量及重复条目不影响要求。SDE 批量递归展开六组技能前置，同技能取最高等级，保存为既有技能条目。未知物品会报告名称及行号并拒绝整次保存。EFT 只在请求期间参与推导。导入结果表达 SDE 技能门槛，CPU、PG 与实战辅助技能需由管理者按需补充。

创建和更新请求支持可选 `disabled`，默认 `false`，更新省略时保留原值。列表和详情响应包含 `disabled`。GORM 启动迁移新增非空列并以 `false` 填充已有计划。

公共计划在编辑表单中可停用和重新启用。`super_admin`、`admin`、`senior_fc` 可继续查看和管理已停用计划，普通成员的列表分页总数和详情均受服务端过滤。完成度检查及保存的检查计划选择始终排除已停用计划。两端管理表单也支持停用个人计划，所有者仍可管理，停用期间跳过完成度检查。
