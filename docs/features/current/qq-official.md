---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-26
source_of_truth:
  - contracts/modules/qq-official.yaml
  - server/internal/service/qq_official.go
  - server/internal/service/qq_official_group.go
  - server/internal/service/qq_official_gateway.go
  - server/internal/service/qq_official_dispatch.go
---

# QQ 官方机器人

AmiyaEden 支持向已授权的 QQ 群主动发送文本通知；目标机器人具备相应平台权限时，可在站内人工审批入群申请。不支持私聊、频道、全体提及、任意消息、自动资格判断或批量群管理。账号资料中的 QQ 联系方式不是平台 OpenID，不作关联或推导。

## 群授权与业务通知

1. 后端主动连接腾讯官方 WebSocket 网关接收机器人进退群及群主动接收开关事件，不需要向 QQ 暴露回调地址。已加入的群也可通过有权限的群状态查询核验。管理员给群填写可识别的别名并确认用途及授权。发现/核验不会自动发消息。
2. 经固定文本测试得到 QQ 成功回执后，才可开启该群的业务通知。对舰队和建筑分别指定允许的军团，不设置时不订阅该来源。业务范围和时效见[舰队行动](operation.md)、[军团建筑](corporation-structures.md)。
3. 群关闭主动接收时停发；退群失去授权。重新允许接收或重新加群不会恢复旧任务，重新启用需核验和测试。全局暂停保留未过期待办，停用作废尚未准入的操作。操作已开始后无法由本站保证撤回。

## 入群申请

只为已授权群保存最少申请数据。开通审批写入后，机器人有平台申请事件订阅权限并在群内担任管理员时，WebSocket 可接收入群申请。管理员可通过官方 HTTPS 接口手动刷新列表，不依赖实时事件。机器人需有 QQ 群管理员身份、接口权限，且本站显式开启审批写入。通过列表读取验证相应能力后，超管逐条确认通过或拒绝；同一申请不接受相反的待执行决定，不自动拉黑。平台自动通过、由其他管理员处理或列表已查不到，都不记为本站审批成功。超时或结果未知时先在 QQ 核对，不自动重做。

## 运维界面

Vue、React 都提供「系统管理 → QQ 官方机器人」（`/system/qq-official`），仅超管使用，后端沿用系统管理权限链。AppID、AppSecret、人工审批写入开关均在此配置；AppSecret 加密保存，API 只返回是否已保存，留空且不更换 AppID 时保留原密钥。首次保存、更换 AppID 或轮换 AppSecret 后，投递和人工审批写入均关闭，须分别启用。关闭人工审批写入会作废未开始的审批，不影响申请查询；已经发出的请求不能撤回。WebSocket 收事件独立于出站投递开关，暂停或停用投递仍会更新群状态。页面显示网关连接状态、最近入箱事件时间、群关系、任务账本、下次可用时间和脱敏尝试，可以取消还未开始的任务。`已入队`不等于`QQ 已确认`，后者也不等于用户已读。待审核没有可核实的终审回执时转未知，未知不提供通用重发。不在页面展示 AppSecret、Access Token、平台群/成员标识、申请验证答案或原始事件。

使用[部署与接入指南](../../guides/qq-official-operations.md)实测后启用。[统一调度架构](../../architecture/qq-official-dispatch.md)记录预算、故障及恢复规则；[站内接口](../../../contracts/modules/qq-official.yaml)定义契约。
