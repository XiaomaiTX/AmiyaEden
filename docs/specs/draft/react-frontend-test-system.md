---
status: draft
doc_type: spec
owner: engineering
last_reviewed: 2026-09-10
source_of_truth:
  - static-react/vitest.config.ts
  - static-react/package.json
  - static-react/src/test/setup.ts
  - static-react/eslint.config.js
  - .github/workflows/verify-ci.yaml
  - docs/standards/testing-and-verification.md
  - docs/guides/testing-guide.md
---

# React 前端测试体系建设草案

## 1. 背景与当前状态

`static-react/` 处于 Vue -> React 迁移期（见 `docs/specs/draft/frontend-react-migration-plan/`），测试随页面迁移逐个补充，尚无一次系统性盘点。本草案基于 2026-09-10 对实际代码的全量检查，不追求覆盖率数字，目标是用有限测试预算覆盖真实生产风险。

当前状态概览：

- 测试基础设施：可用且轻量，但缺少共享测试装置与请求层 Mock 规范。
- 测试资产：54 个测试文件、约 190 个用例；权限/路由守卫质量高，页面测试呈两极分化。
- 缺失层级：无真实浏览器交互测试、无 E2E、无覆盖率报告、无测试专用 ESLint 规则。
- CI：`verify-ci.yaml` 的 `frontend_react` job 运行 lint、`vitest run`、`check:api-contract` 和 build；build 内执行 `tsc -b`。

## 2. 实际发现

### 2.1 测试基础设施现状

| 项目 | 现状 | 依据 |
| --- | --- | --- |
| 测试运行器 | vitest 4，`jsdom` 环境，`globals: true`，超时 15s | `static-react/vitest.config.ts` |
| DOM/交互工具 | `@testing-library/react` + `jest-dom` + `user-event` 均已安装并在用 | `static-react/package.json`、各 `*.test.tsx` |
| setup | `src/test/setup.ts`：jest-dom、`matchMedia`、Pointer Capture、`scrollIntoView` polyfill、`asyncUtilTimeout: 10_000` | `static-react/src/test/setup.ts` |
| API Mock | 两种模式并存：`vi.spyOn(globalThis, 'fetch')` 拼假 `Response`；或 `vi.mock('@/api/*')` 整模块替换。无 MSW | 各页面测试；`package.json` 无 msw 依赖 |
| Router/i18n/Store 封装 | 无共享渲染入口；每个文件自行 `createMemoryRouter(appRoutes)` + 手动 `useSessionStore.setState()` 重置 | `src/pages/*.test.tsx`、`src/app/router.test.tsx` |
| 覆盖率 | 未配置 `coverage`，CI 无覆盖率产出 | `vitest.config.ts`、`verify-ci.yaml` |
| 浏览器/E2E | 无 Playwright（lockfile 中仅为 `shadcn` 传递依赖）、无 vitest browser mode、无任何 E2E | `package.json`、全仓检索 |
| ESLint 测试规则 | 无 `eslint-plugin-testing-library`、无 vitest 插件 | `static-react/eslint.config.js` |
| CI 门禁 | lint、单测、契约隔离检查、构建（含类型检查），按 React 路径触发 | `.github/workflows/verify-ci.yaml` job `frontend_react` |
| 测试类型检查 | `tsconfig.spec.json` 独立覆盖 `*.test.*` | `static-react/tsconfig.spec.json` |

### 2.2 现有测试分类（按质量）

**高质量、应保留并作为范式的测试：**

- 路由守卫集成测试：`src/app/router.test.tsx`（19 例）覆盖未登录重定向、角色/能力/newbro 403、404、profile 锁定重定向（含 `profileLockReasons` 断言）、401 事件跳转登录、已下线路由 404。
- 权限纯逻辑单测：`src/app/route-access.test.ts` 覆盖能力 AND/OR 语义、super_admin 旁路、路由声明与能力对齐。
- 会话引导：`src/auth/session-bootstrap.test.tsx` 验证 StrictMode 下 token 刷新仅一次。
- 竞态回归：`src/pages/info-assets-page.test.tsx` 的 "ignores stale response from slow search" 用手动 resolve 慢 Promise 验证陈旧响应不覆盖新状态——这是全仓最有价值的前端测试范式。
- 完整业务流：`src/pages/shop-pages.test.tsx` 覆盖购买（弹窗表单 + 请求体断言）、商品创建、订单发放。
- 身份切换：`src/pages/system-pages.test.tsx` 覆盖 impersonation 换 token 后加载目标用户。
- 通用组件状态优先级：`src/components/ui/data-table.test.tsx` 覆盖 loading/error/empty 状态优先级、服务端分页/排序。
- React Aria Select：`src/components/ui/select.test.tsx` 覆盖空字符串值往返、键盘选择、禁用项。
- 纯逻辑单测：`src/lib/format.test.ts`（13 例）、`src/pages/dashboard-console/pap-trend.test.ts`、`src/pages/ticket-category.test.ts`、`src/pages/info-esi-check-logic.test.ts`、`src/i18n/context.test.ts`。

**已存在但质量不足的测试：**

- 顺序依赖的 fetch Mock 链：`shop-pages.test.tsx`、`info-wallet-page.test.tsx` 等依赖 `.mockResolvedValueOnce` 的调用顺序，并用 `fetchMock.mock.calls[2]` 这类下标断言请求体。页面一旦新增一个请求（如角标轮询），整链错位。
- 仅快乐路径的单用例页面测试：`info-wallet`、`info-skill`、`info-ships`、`info-npc-kills`、`info-implants`、`info-esi-check-page`、`info-contracts`、`dashboard-npc-kills`、`ticket-detail`、`ticket-my-tickets` 各只有 1 个"加载并渲染"用例，无失败、空数据分支。
- render 冒烟测试：`src/pages/migration-drift-pages.test.tsx` 3 个用例只断言数据出现，且整模块 `vi.mock` 自身 API 层（Mock 了自己的代码而非网络边界）。
- locale 耦合：多数页面测试直接断言 zh-CN 硬编码文案（如 'EVE SSO 登录'），依赖 preference store 默认 zh-CN；`dashboard-characters-page.test.tsx` 依赖 inputs 下标定位表单字段。

**缺失的关键场景（页面存在但零测试）：**

高风险页面约 25 个无任何测试，包括：`srp-apply/srp-manage/srp-prices`、`welfare-approval/welfare-my/welfare-settings`、`newbro-select-captain/newbro-select-mentor/newbro-manage` 等 7 个 newbro 页面、`operation-*` 6 个页面、`skill-plans` 3 个页面、`ticket-management/ticket-admin-detail/ticket-categories/ticket-statistics`、`system-webhook/system-auto-role/system-pap-exchange/system-audit` 等、`fuxi-hall-manage`、`info-fittings`、`system-user-center`。

**其他缺口：**

- `http-client.ts` 仅 2 个用例（Bearer 注入、401 事件）；204、非 JSON 响应、网络异常、`assertSuccess`（`src/api/response.ts` 的 `code !== 0/200` 分支）无直接覆盖。
- 人物主号切换 `setPrimaryCharacter`（`dashboard-characters-page.tsx:273`）无测试——这是"身份切换"核心操作。
- 约 25 个 API 模块仅 4 个有测试（`auth`、`qq-governance`、`skill-plan`、`tool-bookmark`）。
- React Aria 复杂组件中仅 Select 有键盘测试；Dialog（focus trap/恢复）、Combobox、Menu、Tabs 均无。
- 无未预期请求守卫（`dashboard-characters-page.test.tsx:112-177` 的 URL 分发式 Mock + 未匹配即 throw 是全仓唯一例外，未推广）。
- 无 console error / React warning 检查；无测试数据工厂（大 JSON payload 在用例间复制粘贴）。

### 2.3 已确认无问题的方面

- CI 命令与 `docs/standards/testing-and-verification.md` 一致且可按字面运行。
- 无 `.skip`/`.todo`/`.only` 遗留，无 snapshot 测试，无 CSS class 定位（唯一 `getByTestId('asset-stats')` 用于聚合断言，可接受）。
- 本机（Windows）偶发并行 vitest 超时需串行复跑，CI（ubuntu）未见；属环境噪音而非测试本身不稳定。

## 3. 主要问题（按风险排序）

1. **高风险业务流零覆盖**：SRP 审批、福利审批、newbro 选导师/舰长、工单后台流转等写操作页面无测试，是当前最可能发生生产回归的区域。
2. **Mock 基础设施脆弱**：顺序依赖 fetch 链 + 下标断言使"新增一个请求"成为高频破坏点，且破坏方式是错位而非明确失败。
3. **失败/空数据分支几乎只靠 info-assets 一处示范**，其余页面只有成功路径。
4. **身份/主号切换与 http-client 边界行为未覆盖**。
5. **无共享测试装置**，每个文件重复搭 Router/Store/i18n 脚手架，新增页面测试成本高，直接抑制覆盖增长。
6. **React Aria 键盘/焦点行为无验证手段**（jsdom 不可靠，需要浏览器层）。

## 4. 测试建设原则

> 测试用户可观察的业务行为和关键业务规则，不测试实现细节。

- 在网络边界 Mock（fetch 或 api 模块二选一并统一），不 Mock 自身组件、Store、Hook。
- 断言用户所见（角色查询、文案、请求体），不断言内部 state、私有函数、调用次数细节、DOM 层级。
- 仅重构实现不改行为时测试必须保持稳定。
- Mock 层对未预期请求应失败，而不是静默错位。
- 不为覆盖率数字补测试；每个用例对应一条业务风险。
- 保留并复制现有最佳范式：info-assets 的竞态测试、router.test 的守卫矩阵、shop-pages 的请求体断言。

## 5. 目标测试体系（四层，按需配备）

1. **业务逻辑单元测试**（现有主力，继续保持）：权限判定、数据转换、格式化、i18n 解析、reducer/selector 逻辑。代表：`route-access.test.ts`、`format.test.ts`。
2. **组件和页面行为测试**（测试主体，需补强）：真实路由树 + 真实 Provider + fetch 边界 Mock；覆盖加载/失败/空数据/无权限状态与表单提交。
3. **真实浏览器交互测试**（新增层，小而精）：仅用于 jsdom 不可靠的行为——React Aria Dialog 焦点圈定与恢复、Combobox/Menu 键盘导航、Portal/Overlay。用 vitest browser mode 实现，不为此引入 Playwright 全家桶。
4. **关键流程 E2E**（本轮不建，见第 8 节）：迁移期双前端并行、E2E 需要真实后端与会话，投入产出比低，推迟到迁移收尾阶段再决策。

## 6. 用例选型

按[前端行为测试建议](frontend-test-quality-standard.md)选择风险对应的断言与测试层。优先修复已经对正常修改造成阻塞的偶然顺序断言；存量用例由实际保护的行为决定保留、改写或删除。

## 7. 分阶段改进计划

### 阶段一：测试资产盘点收口与脆弱点清理（优先级：高）

- **要解决的问题**：顺序依赖 Mock 链和下标断言是当前最高频的维护摩擦；render 冒烟测试制造虚假安全感。
- **范围**：`shop-pages`、`info-wallet-page`、`dashboard-characters-page`、`dashboard-console-page`、`system-pages`、`migration-drift-pages` 等使用 Once 链的文件。
- **计划动作**：
  1. 推广 `dashboard-characters-page.test.tsx:112-177` 的 URL/method 分发式 fetch Mock 为 `src/test/mock-api.ts` helper，未匹配 URL 即 throw（天然充当未预期请求守卫）；
  2. 将下标断言改为"按 URL 查找调用"的断言 helper；
  3. `migration-drift-pages.test.tsx` 的模块级 `vi.mock` 改走 fetch 边界；
  4. 表单下标定位改 label/name 定位。
- **验收标准**：无业务依赖的读请求调序时用例仍通过；错误的写请求或额外写入仍使相关用例失败。
- **依赖/风险**：无新依赖；改动纯测试代码，可逐文件独立提交。

### 阶段二：统一测试基础设施（优先级：高，与阶段一可并行）

- **要解决的问题**：每个测试文件重复搭脚手架，新增页面测试成本高；locale 与 store 状态隐式耦合。
- **范围**：`src/test/` 目录新增共享装置；`vitest.config.ts`、`eslint.config.js`。
- **计划动作**：
  1. `renderApp(initialPath, { session, locale })`：统一 createMemoryRouter + Provider + store 重置（整合现有各文件的 beforeEach 样板，包括 `router.test.tsx:64-81`、`shop-pages.test.tsx:22-47`）；
  2. 阶段一的 `mock-api.ts` 提供 `expectRequest(url, method)` 断言与测试数据工厂（沉淀 shop/order/character 等重复 payload）；
  3. 覆盖率报告：`vitest.config.ts` 加 `coverage.provider: 'v8'`，CI 上传 artifact 仅作观测，不设阈值门禁；
  4. 在请求契约测试的 fake transport 上报意外请求；只有真实回归已被全局日志噪声掩盖时，才增加针对性的断言。
- **验收标准**：共享装置替代实际重复的初始化，不隐藏用例的会话、语言或相关请求。覆盖率若收集，仅用于定位缺口。
- **依赖/风险**：先复用现有工具，测量重复工作和信号质量再决定是否增加依赖。

### 阶段三：按风险补齐覆盖（优先级：高，持续多个迭代）

优先级顺序（由 2.2/2.3 的实际缺口决定）：

1. **会话与身份**：`setPrimaryCharacter` 主号切换流（含失败 toast）；`http-client` 补 204、非 JSON、网络异常、`assertSuccess` 错误码分支。
2. **高风险写操作页面**（每页至少：成功流 + 请求体断言 + 一条失败/空数据分支）：`srp-apply`、`srp-manage`、`welfare-approval`、`newbro-select-captain/mentor`、`ticket-management`（状态流转）、`skill-plan-management`（增删改）。
3. **存量页面补实际缺口**：错误或空状态由页面自己处理且有业务影响时才补用例；共享组件已覆盖的行为不逐页重测。
4. **React Aria 浏览器层**：启用 vitest browser mode（chromium），首批仅 3 个用例：Dialog 打开后焦点圈定与 Esc/关闭后焦点恢复、Combobox 键盘筛选、Menu 方向键导航。文件放 `src/components/ui/*.browser.test.tsx` 并在 vitest 配置独立 workspace/glob。
5. **纯逻辑补漏**：`menu-config` 已有测试，其余 hooks（`use-permission`、`use-mobile`、`use-galaxy-registry-timeout-notification`）按 testing-guide 的轻量单测标准评估，不值得测的明确跳过。

- **验收标准**：第 1、2 项所列场景各至少 1 个行为级用例进入 CI；浏览器层用例在 CI 稳定运行（如 ubuntu runner 缺浏览器则用 `--browser.headless` + actions 缓存）。
- **依赖/风险**：browser mode 需要 CI 装 chromium（`pnpm exec playwright install chromium` 级别的环境步骤）；写操作页面测试需先阅读对应 Vue 侧行为与 feature docs 对齐预期。

### 阶段四：按回归信号维护

保留现有 CI。评审以所改行为和既有保障为准，遇到不稳定用例时先找原因并保留独有覆盖。没有具体漏测缺陷和可度量证据前，不增加覆盖率阈值、全局日志门禁和例行分级审计。

## 8. 不在本轮范围内

- E2E 层建设（需要真实后端 + EVE SSO，迁移期投入产出比低；待 React 前端成为唯一前端后再立项决策）。
- Vue `static/` 侧测试体系（另行治理，本草稿仅限 `static-react/`）。
- 后端 Go 测试（已有独立规范与覆盖）。
- 全项目统一覆盖率数字目标（如 80%/100%）。
- 一次性重写存量测试——现有高质量测试（2.2 节第一组）原样保留。

## 9. 相关引用

- 配置：`static-react/vitest.config.ts`、`static-react/src/test/setup.ts`、`static-react/eslint.config.js`、`static-react/tsconfig.spec.json`、`.github/workflows/verify-ci.yaml`
- 政策：`docs/standards/testing-and-verification.md`、`docs/guides/testing-guide.md`、`docs/standards/regression-test-plan.md`
- 用例选型：[前端行为测试建议](frontend-test-quality-standard.md)
- 范式参考：`static-react/src/pages/info-assets-page.test.tsx`（竞态）、`static-react/src/app/router.test.tsx`（守卫矩阵）、`static-react/src/pages/shop-pages.test.tsx`（请求体断言）、`static-react/src/pages/dashboard-characters-page.test.tsx`（URL 分发式 Mock）
- 迁移上下文：`docs/specs/draft/frontend-react-migration-plan/index.md`

## 10. 待确认决策

1. **浏览器层技术选型**：vitest browser mode（本草案假设）vs Playwright component testing；影响 CI 环境步骤与依赖体积。
2. **优先补哪些业务回归**：按当前迁移与生产反馈更新阶段三的清单；不按测试写法统一迁移存量用例。
