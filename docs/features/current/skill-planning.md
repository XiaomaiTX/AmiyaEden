---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-04-27
source_of_truth:
  - server/internal/router/router.go
  - server/internal/service/skill_plan.go
  - static/src/api/skill-plan.ts
  - static/src/views/skill-planning
---

# Skill Planning 模块

## 当前能力

- 军团技能计划的列表、详情、创建、编辑、删除
- 创建 / 编辑技能计划时可选一个舰船图标，并在列表中展示
- 粘贴技能文本解析为技能计划条目
- 管理员可拖拽调整当前页内的技能计划顺序，也可直接编辑 `sort_order` 做跨分页排序
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

### 后端路由

- `/api/v1/skill-planning/skill-plans/*`
- `/api/v1/skill-planning/personal-skill-plans/*`

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
- `static/src/api/skill-plan.ts`
- `static/src/router/modules/skill-planning.ts`
- `static/src/views/skill-planning/skill-plans`
- `static/src/views/skill-planning/personal-skill-plans`
- `static/src/views/skill-planning/completion-check`

## Compatibility Notes (2026-04-25)

- Skill plan `skills_text` parsing now accepts localized client export lines such as `<localized hint="Large Artillery Specialization">大型火炮专业研究*</localized> 4`.
- During save, parser normalization removes markup tags and trailing `*`, then resolves with the existing skill-name mapping and stores in the current concise structured format.
