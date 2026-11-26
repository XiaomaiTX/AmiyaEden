---
status: active
doc_type: api
owner: engineering
last_reviewed: 2026-11-26
source_of_truth:
  - contracts/openapi.yaml
  - contracts/modules/
  - server/internal/router/openapi_routes.go
  - server/internal/router/openapi_policy.go
---

# API 约定

## Base URL

```text
/api/v1
```

## 认证方式

需要认证的接口通过以下任一方式携带 JWT：

- `Authorization: Bearer <token>`
- `?token=<token>`

说明：

- 持有有效 JWT 的请求方可能仍是 `guest`
- `jwt` 本身不排除 `guest`；附加的登录、职权及军团能力要求见操作的 `x-gin-access`

## 统一响应

成功：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

分页成功：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "list": [],
    "total": 0,
    "page": 1,
    "pageSize": 20
  }
}
```

说明：

- 401 / 403 使用对应 HTTP 状态码
- 许多业务失败仍会返回 HTTP 200，但 `code != 200`。两套前端的生成客户端都返回这份业务响应，由通用解包函数判断业务是否成功。联系冲突可以同时触发原有提示；认证失效（401）仍由请求层拒绝并触发原有退出流程。

## 契约同步顺序

站内 `/api/v1` 的 HTTP 契约来自 [OpenAPI 根规范](../../contracts/openapi.yaml) 与 [业务模块片段](../../contracts/modules/)；生成时将两者合并校验，路径、组件键和 `operationId` 不得重复。内部服务接口、静态资源和上游 ESI 不在此契约范围内。

1. 先修改契约，再运行 `pnpm -C contracts install --frozen-lockfile` 和 `pnpm -C contracts api:generate`。
2. 实现生成的 Go strict 接口：handler 接收生成请求并返回生成响应，不重复注册路径、解析 HTTP 参数或维护 DTO；请求处理函数放在对应业务的 handler 文件，由生成路由直接调用，不再为此单设转发层；业务规则仍由 service 处理。这里的 strict 指生成接口的类型签名，不需要手写 `Strict…Handler` 包装原 handler。路由直接组装各业务 handler；必要的输入、响应转换留在所属业务文件，共用错误处理仍可共享。验证路由或读参行为的测试走生成入口，不另外维护一套旧入口。Vue、React 在页面或现有 composable、hook、store 中直接从各自的生成客户端导入操作和原始接口类型。不要另建文件只做类型转出或改名；手写类型用于表达实际的前端状态、表单或视图模型。普通 JSON 响应复用通用解包函数，不为逐接口改名、转发或解包另建 API 包装。只有共享流程或必要的视图适配才抽取；不在页面直接发 HTTP 请求。SSO 回调由浏览器导航，不用生成客户端替代。
3. 提交 Go、Vue、React 生成物及相关测试；仅在业务入口或功能行为改变时同步相应文档。不手改生成文件。提交前运行 `pnpm -C contracts api:check`；CI 检查生成差异，并通过[启动路由测试](../../server/bootstrap/openapi_routes_test.go)对比 OpenAPI 与 Gin 启动注册后的整张 `/api/v1` method/path 表；两边各自缺失的项会分别报告。

从仓库根目录运行 `node contracts/inventory.cjs` 可导出逐操作的 Go strict 实现和两端生成客户端引用清单（需已安装契约与 Vue 依赖）。清单将前端引用标为 `direct`、`workflow`、`browser-navigation`、`no-client-reference` 或待核对的 `review-adapter`。`workflow` 需要记录保留原因；`no-client-reference` 不表示接口可以删除。清单里的模块导入是排查线索，不等同于页面运行时调用。使用 `node contracts/inventory.cjs --check` 检查未记录的流程适配和手写 `/api/v1` URL。保留适配须在 [例外清单](../../contracts/inventory-exceptions.cjs) 里按 `operationId` 说明职责；若职责可以留在单一消费点，应移除包装而不是登记例外。

### 路由与绑定

- 站内路由只在生成的 Gin 注册表注册。标准 `security` 声明接受的凭据，`x-gin-access` 提供有序的 Gin 权限链：`[]` 表示公开；其余链以 `jwt` 开始，`login` 排除 guest，`role:a,b` 表示组内任选，`cap:` 要求指定军团能力。链中各步骤必须同时通过，并在解码前执行。所有权等业务权限仍在 service 检查，不另维护一份逐操作的权限描述。
- `x-gin-body` 声明既有正文限制、空值或异常输入回退等特殊行为。生成的 JSON 绑定从请求流解码，不设默认 Go 层大小上限；仅显式声明 `maxBytes` 的操作保留服务端限额，反向代理的限制独立生效。
- `x-gin-ignore-invalid-query` 指定可回退的可选查询参数。参数级 `x-gin-trim-query` 保留星系登记 `system_config_id` 的空白归一化；`x-gin-legacy-atoi` 保留用户列表 `status` 的 `Atoi` 行为。不要在业务 handler 中重新绑定 HTTP 请求。绑定失败沿用 API 的错误 envelope；生成代码不替代业务校验，也不承诺自动验证所有响应 schema。

## 文档分工

- `docs/api/conventions.md`: 统一规则
- `docs/api/route-index.md`: 接口阅读索引（只作参照；路径与权限以 OpenAPI 契约为准）
- 具体请求、响应和字段结构：以 OpenAPI 契约为准，由 Go 和前端生成物消费。

React 侧使用本地模块化类型，不回退引用 Vue 的 `Api.*` 全局命名空间；迁移期间两端都存在时，后端响应和行为是共同契约，两个前端的生成客户端调用和类型必须分别通过检查。

## 禁止事项

- 前后端偷偷改字段名不更新类型
- 在 Markdown 另行维护 `/api/v1` 路径、权限或 DTO 清单
- 把未来计划接口写进 route index 冒充已实现
