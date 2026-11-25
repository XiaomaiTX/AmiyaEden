---
status: draft
doc_type: spec
owner: engineering
last_reviewed: 2026-11-25
source_of_truth:
  - docs/standards/testing-and-verification.md
  - docs/guides/testing-guide.md
  - static-react/vitest.config.ts
related_docs:
  - docs/specs/draft/react-frontend-test-system.md
---

# 前端行为测试建议（草案）

本草案用于改进前端行为测试，不新增 PR 或 CI 门槛。测试政策和例外由[测试与验证规范](../../standards/testing-and-verification.md)定义；测试选型见[测试指南](../../guides/testing-guide.md)。

## 按风险选择场景

| 要保护的行为 | 最小有用验证 |
| --- | --- |
| 格式化、归一化、权限判定 | 通过纯函数或 hook 覆盖有意义的输入和输出。 |
| 提交、审批、发放等写操作 | 验证合法操作能完成并发送正确载荷；若重复提交会改变业务结果，断言写入次数。拒绝或失败用例覆盖该操作特有的风险。 |
| 权限、空状态、失败恢复 | 验证能否进入页面、能否执行目标操作，或者失败后用户可以做什么。复用共享守卫或组件的覆盖；只有页面自有分支才补该页面用例。 |
| 搜索和状态竞态 | 用两个可控响应验证旧结果不能覆盖新状态，而非靠固定等待。 |
| 焦点、键盘、Portal | jsdom 能覆盖的先在现有测试中验证；只有明确依赖真实浏览器的关键行为才考虑浏览器测试。 |

先选择能覆盖实际风险的最小一层。成功用例能锁住曾经被阻断的合法路径时，本身就是有效回归，不要求机械补空态或失败态。Vue 可继续用现有轻量测试和必要的手工验证；不为与 React 形式对称而增加组件测试框架。

## 写法与维护

- 按行为命名，用角色、标签或名称查找可交互元素；断言页面结果、持久化请求或公开状态契约。样式或 DOM 结构只有在本身属于对外契约时才值得断言。
- Mock 被测行为的真实输入边界。测试请求契约时按 URL 和 method 分发假响应、拒绝未预期的写请求；不将无关读请求的全局顺序和数量写入断言。对同一操作的失败→重试，可用顺序响应描述业务过程。
- 用可观察状态、受控 Promise 或假时钟推进异步测试；需要断言文案时明确测试语言，文案内容本身不是目标时优先断言语义。
- 装置和数据适合复用时再提取；不以脚手架行数、调用次数、覆盖率分数或是否使用特定 helper 单独判定测试质量。
- 清理旧用例前先确认它在保护什么。若唯一覆盖了关键行为，先改写或补等效覆盖；若只是重复、断言内部细节且会阻塞正常重构，再合并或删除。

## 仓库内参考

- [路由与权限行为](../../../static-react/src/app/router.test.tsx)
- [旧请求不能覆盖新结果](../../../static-react/src/pages/info-assets-page.test.tsx)
- [按 URL 分发假响应](../../../static-react/src/pages/dashboard-characters-page.test.tsx)
- [确认、权限与单次写请求](../../../static-react/src/pages/system-pages.test.tsx)
- [键盘选择行为](../../../static-react/src/components/ui/select.test.tsx)

先在[React 测试改进草案](react-frontend-test-system.md)列出的高风险流程应用，出现可复用的实证后再沉淀到现有测试指南或规范，无需另设评分类别或周期核查。
