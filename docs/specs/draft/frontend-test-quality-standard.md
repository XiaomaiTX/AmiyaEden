---
status: draft
doc_type: spec
owner: engineering
last_reviewed: 2026-09-10
source_of_truth:
  - docs/standards/testing-and-verification.md
  - docs/guides/testing-guide.md
  - static-react/vitest.config.ts
  - static-react/src/test/setup.ts
  - docs/specs/draft/react-frontend-test-system.md
---

# 前端测试质量标准与检验规范（草案）

## 1. 定位与适用范围

本草案定义前端测试的**质量判定标准**与**检验方法**，用于三类场景：

1. PR 评审时判定新增/修改的测试是否达标（第 5.1 节清单）；
2. 定期审计存量测试质量（第 5.2 节清单 + 第 4 节评分）;
3. 决定某个测试应当保留、修补、重写还是删除。

与现有文档的分工：

| 文档 | 职责 |
| --- | --- |
| `docs/standards/testing-and-verification.md` | 仓库级测试政策（何时必须补测试、允许的例外、最低验证命令） |
| `docs/guides/testing-guide.md` | 测试放置位置与选型速查 |
| `docs/specs/draft/react-frontend-test-system.md` | 测试体系建设路线图（分阶段改进计划） |
| 本草案 | 测试本身的质量标准与检验手段 |

适用范围：

- **硬性要求**针对 `static-react/`（迁移目标前端，具备 jsdom 行为测试能力）。
- `static/`（Vue，`tsx --test` 原生运行器，仅纯逻辑单测）仅适用第 3、4 节的通用质量要求，不要求补组件层；进入 React 迁移的页面按本草案执行。
- 本草案经评审定稿后应晋升为 `docs/standards/` 下的正式标准。

## 2. 分层测试标准（该有什么测试）

| 层级 | 验证对象 | 强制性 | 工具 |
| --- | --- | --- | --- |
| L1 业务逻辑单元 | 纯函数、hook、状态流转、i18n 解析、格式化 | 拥有相应逻辑时**必须** | vitest |
| L2 组件/页面行为 | 用户所见、操作结果、状态分支、Router/Store/API 协作 | 页面与关键业务组件**必须**；纯展示组件**不要求** | vitest + Testing Library + jsdom |
| L3 浏览器交互 | React Aria 键盘导航、焦点圈定/恢复、Portal/Overlay | 使用对应复杂组件的高风险页面**必须**（建设见路线图阶段三） | vitest browser mode |
| L4 E2E | 真实环境用户旅程 | **暂不要求**（见路线图第 8 节，待迁移收尾后决策） | 未选型 |

**页面级必备场景矩阵**（L2，按适用性勾选；不适用须能在测试名或注释中看出原因）：

| 场景 | 何时必须 |
| --- | --- |
| 成功加载并渲染关键数据 | 所有数据页面 |
| 请求失败并展示错误反馈 | 所有数据页面 |
| 空数据状态 | 所有列表/统计页面 |
| 无权限/未登录 | 有 route meta 限制的页面（守卫已在 `router.test.tsx` 覆盖的，页面内按钮级权限仍需覆盖） |
| 表单提交的请求体契约 | 所有写操作表单 |
| 提交失败的错误反馈与可重试 | 所有写操作表单 |
| 竞态/陈旧响应防护 | 有搜索、筛选、防抖请求的页面 |
| 危险操作确认 | 有删除、发放、审批等不可逆操作 |

## 3. 编写规范（怎么写）

以下为硬性规则。**必须**违反即不达标；标注【存量清理】的条目对存量测试按路线图阶段一渐进清理，新测试立即生效。

### 3.1 放置与命名

- 测试文件与被测文件同目录，`*.test.ts(x)`。
- 用例名描述行为而非实现：`round-trips an empty-string option`、`ignores stale response from slow search` 为范式；`test 1`、`works` 不合格。
- 一个用例只验证一条业务规则；该规则应能从用例名直接读出。

### 3.2 Mock 边界【核心规则】

- **必须**在网络边界 Mock：`vi.spyOn(globalThis, 'fetch')` 配合按 URL/method 分发的假响应（范式：`dashboard-characters-page.test.tsx:112-177`）。
- 分发式 Mock 对未匹配的 URL **必须**抛错——未预期请求即失败，禁止静默错位。
- **禁止**顺序依赖的 `.mockResolvedValueOnce` 长链与 `fetchMock.mock.calls[N]` 下标断言【存量清理】；请求体断言改为"按 URL + method 查找调用"。
- **禁止**`vi.mock` 项目自身的 API 模块、组件、Store、Hook（除非被测边界确实在模块之上且无法走 fetch，须注释原因）。`migration-drift-pages.test.tsx` 为待清理反例。
- Store 状态用 `setState` 直接种入会话快照（现有惯例），不算违反 Mock 边界——Store 是被协作的真实对象。

### 3.3 查询与断言

- **必须**优先使用角色/标签/名称查询：`getByRole`、`getByLabelText`、`getByPlaceholderText`（占位符仅当其即业务标识时）。
- **禁止**按 `getAllByRole(...)` 下标定位表单字段【存量清理】；用 label 关联或 `within(dialog)` 语义定位。
- **必须**断言用户可观察结果（渲染文案、可访问属性、请求契约）或关键内部契约（Store 会话状态、路由跳转目标）。
- **禁止**断言：内部 state、私有函数、Hook 调用次数、CSS class、DOM 层级结构。
- 请求契约断言必须校验 URL、method 与请求体关键字段（范式：`shop-pages.test.tsx` 对 `/api/v1/shop/buy` body 的全量断言）。

### 3.4 异步与时间

- **必须**用 `waitFor` / `findBy*` 等待断言本身，而非等待后断言。
- **禁止**固定时长 `sleep`/`advanceTimersByTime` 掩盖未就绪状态；竞态测试用受控慢 Promise（手动 resolve，范式：`info-assets-page.test.tsx:217-330`），定时器行为用 `vi.useFakeTimers`（范式：`feedback.test.ts`）。
- **禁止**依赖真实网络、真实时区敏感的相对时间断言；时间相关断言必须给定绝对输入。

### 3.5 国际化

- 涉及新文案的变更**必须**同步 `zh.json` 与 `en.json`（repo 规则），且测试不得因缺 key 而依赖回退行为。
- 新测试断言文案时**必须**明确 locale 来源：统一走 i18n 解析，或显式固定 `usePreferenceStore.setState({ locale: ... })` 并在文件头注明。
- 存量硬编码 zh-CN 断言按路线图决策点 2 处理，不强制立即迁移。

### 3.6 数据与装置

- 重复使用的 payload（角色、订单、商品、结构等）**必须**沉淀到共享数据工厂/fixture，不在用例间复制粘贴大段 JSON。
- 页面测试脚手架**必须**复用共享 `renderApp` 入口（路线图阶段二交付前，允许沿用各文件现有 beforeEach 样板，新文件不得再复制超过 20 行脚手架）。

## 4. 质量要求（判定维度与评分）

审计时对每个测试文件按五个维度判定 P（通过）/ F（不通过）：

| 维度 | 判定标准 |
| --- | --- |
| D1 风险对应 | 每个用例对应一条可陈述的业务风险或契约；删除该用例会损失明确的保护 |
| D2 可观察断言 | 断言用户所见或请求契约；无实现细节断言（3.3 节） |
| D3 分支完整性 | 满足第 2 节场景矩阵中适用的条目；纯成功路径的页面级测试本维度为 F |
| D4 稳定性 | 无顺序依赖、无真实时间/网络依赖；仅重构实现不改行为时保持稳定 |
| D5 可维护性 | Mock/装置/数据复用共享设施；重复 payload 已工厂化；无复制粘贴脚手架 |

总体分级（取最低维度决定）：

- **A 保留**：五维全 P，或 D5 单项 F 但改动成本低于价值。
- **B 修补**：D1、D2 全 P，D3/D4/D5 有 F——补分支、替换 Mock 方式、抽工厂，保留用例骨架。
- **C 重写或删除**：D1 或 D2 为 F（不知道保护什么、断言不了行为）——render 冒烟、纯实现细节测试直接删除，不为覆盖率重写。

判定时**禁止**以覆盖率数字替代上述维度；覆盖率仅用于发现"哪里没有测试"，不用于评判"测试好不好"。

## 5. 检验清单

### 5.1 新增/修改测试的 PR 检验

**阻断项（任一不过即要求修改）：**

- [ ] 用例名能读出被保护的业务行为（D1）
- [ ] Mock 位于网络边界，未预期请求会失败（3.2）
- [ ] 断言为用户可观察结果或请求契约，含 URL/method/body（3.3）
- [ ] 查询使用角色/标签/名称定位（3.3）
- [ ] 页面级测试含至少一条非成功分支（失败/空数据，按第 2 节矩阵适用项）（D3）
- [ ] 无固定等待、无 Once 链下标断言、无模块级自 Mock（3.2、3.4）
- [ ] 新文案已同步 zh/en，测试 locale 来源明确（3.5）
- [ ] `pnpm lint`、`pnpm exec tsc -b`、`pnpm test` 本地通过（政策要求，见 testing-and-verification.md）

**建议项（提示但不阻断）：**

- [ ] 重复 payload 抽入数据工厂（D5）
- [ ] 脚手架复用共享装置（D5）
- [ ] 竞态场景受控慢 Promise 覆盖（如有搜索/筛选）
- [ ] 用例放在拥有该行为的层（纯逻辑不写页面测试，页面行为不写 E2E）

### 5.2 存量测试审计清单

按文件过检，输出 A/B/C 分级与处置动作：

- [ ] 是否为 render 冒烟（只断言数据出现，无操作无分支）→ C，删除或升级为行为测试
- [ ] 是否顺序依赖 Once 链 / 下标断言 → B，迁移到 URL 分发 Mock
- [ ] 是否模块级 Mock 自身 API/Store → B，迁移到 fetch 边界
- [ ] 是否表单下标定位 → B，改语义定位
- [ ] 是否只有成功路径 → B，按场景矩阵补分支
- [ ] 是否断言实现细节（state/调用次数/CSS/DOM 层级）→ C
- [ ] 是否与另一用例重复覆盖同一行为 → C，删除较弱者
- [ ] 是否长期 skip/todo 或依赖串行复跑才通过 → 单独标记 flaky，限期修复或删除
- [ ] 是否在重复第三方库自身行为（如测 React Aria 内部）→ C，改为测项目封装的键盘/焦点契约

### 5.3 例外与豁免

沿用 `docs/standards/testing-and-verification.md` 的例外条款（纯文档、纯格式化、保持行为的重命名、基础设施成本严重失衡、外部依赖不可测）。豁免必须在 PR 中写明原因；本草案不新增例外类型。

## 6. 检验流程与频次

1. **PR 评审**：reviewer 按 5.1 清单执行；阻断项不过不合并。清单并入 `docs/standards/pre-completion-checklist.md` 的测试小节（定稿时执行）。
2. **定期审计**：每迭代一次，按 5.2 抽检当期有变更的测试文件 + 路线图阶段一列出的脆弱文件；结果以分级清单记录在案。
3. **flaky 处理**：本机需串行复跑或偶发超时的用例，计入 flaky 台账；连续两次迭代未修复即降级 C 删除，不允许长期 `skip` 挂账。
4. **标准晋升**：本草案运行 1-2 个迭代并修订后，晋升为 `docs/standards/frontend-test-quality.md`，同时在 `docs/guides/testing-guide.md` 增加指向本标准的链接。

## 7. 参考范式（达标示例）

| 范式 | 文件 |
| --- | --- |
| 路由守卫行为矩阵 | `static-react/src/app/router.test.tsx` |
| 竞态/陈旧响应回归 | `static-react/src/pages/info-assets-page.test.tsx` |
| 写操作请求体契约 | `static-react/src/pages/shop-pages.test.tsx` |
| URL 分发式 fetch Mock + 未预期请求守卫 | `static-react/src/pages/dashboard-characters-page.test.tsx` |
| 权限纯逻辑单测 | `static-react/src/app/route-access.test.ts` |
| React Aria 键盘行为 | `static-react/src/components/ui/select.test.tsx` |
| fake timers 定时行为 | `static-react/src/feedback/feedback.test.ts` |

## 8. 待确认决策

1. 第 2 节场景矩阵中"竞态防护"是否列为搜索类页面的阻断项（当前为按适用勾选）。
2. L3 浏览器层的具体工具与用例准入门槛（依赖路线图决策点 1）。
3. 存量硬编码 zh-CN 断言是否限期迁移（依赖路线图决策点 2）。
4. 审计分级结果是否需要进 CI（如 A/B/C 徽章或 markdown 报告），还是仅人工记录。
