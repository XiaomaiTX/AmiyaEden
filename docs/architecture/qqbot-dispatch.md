---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-10
---

# OneBot V11 基础设施 PRD 与设计

## 产品边界及不变量

单机器人、Universal 反向 WebSocket、单实例持有连接，默认关闭。超管且具备 system.manage 可配置及运维。不提供任意 Action 透传，通知、舰队、建筑预警与群业务使用独立模型接入，均独立开关默认关闭。已核实第一阶段清理基线 `b90b56a`；不复用旧表或键，不自动删除历史生产数据。

接收成功等价于发送意图已持久化（业务事务内返回须等待外层提交）；平台确认成功不等价于用户已读；未知不得伪装成成功或失败；不承诺严格一次发送；不持有数据库事务等待 OneBot；所有 QQ 外部调用经过唯一私有执行入口。

## 状态机

操作 queued → leased → executing → succeeded / failed / unknown / queued。queued 可取消、过期；执行前重查期限与取消。executing 取消只能请求停止等待，不能撤回已发操作。租约回收：leased 回 queued；executing 的读取及绝对值名片设置回 queued，消息/申请/清退进 unknown。后者未知后禁止自动及人工重试。明确永久失败终止；明确临时失败最多三次。断连（调用前）、限流、Redis 故障、风控暂停只记录等待原因，不消耗失败预算。优先级 0–2，每分钟老化一级，最终按创建时间先后避免饥饿。

消费 pending → leased → done / pending / dead；失败退避最多五次，过期租约回收，死信运维受限重试。至少一次消费，消费者须幂等，本地副作用与 ack 可同事务。消费者启动前注册，无消费者的事件仍保留收件箱。

风控按 read/write 分开 closed → open（连续三次失败）→ half_open（30 秒后一个真实操作）→ closed / open。暂停不视为探测，手工暂停不自动恢复，状态持久化。

## 数据模型

独立 qqbot 前缀：operation（稳定业务键唯一、请求 hash、类型化参数、状态、优先级、有效期、失败数、租约 token、结果、取消）；attempt（执行开始及结果分类）；inbox（规范 JSON hash 去重、原文、接收时间）；delivery（topic/item/consumer 唯一、租约、重试、状态、错误）；circuit（读写状态、计数、探测时间）；setting（唯一配置行）。

随机租约 token 条件更新 fencing；DB 事务只处理持久化，不跨 Redis 或协议等待。开始调用前持久化 executing 和 attempt；崩溃窗口保守按未知处理。结果终态和消费记录同事务。事件七天（未完成/死信消费关联事件随消费记录保留），操作/尝试/投递及死信九十天物理删除；未终态通过有效期收敛后清理，Redis 键 TTL。业务幂等窗口九十天，生产者不得重复使用过期业务键。

## 内部服务契约

Submit(ctx, tx, stableKey, priority, expiresAt, typedRequest)：可复用业务事务，键冲突同内容返回原操作，异内容拒绝。仅支持群文本消息、群资料/列表/成员读取、申请处理、名片修改、清退。GetOperation 查询结果；RegisterConsumer(event/result, name, callback) 提供数据库租约至少一次消费。所有业务资格判断留给调用者。无通用公开 Call。

## 传输及安全

Bearer Token 常量时间比较、X-Self-ID、X-Client-Role=Universal 和 TCP 源 IP CIDR 全部校验，不信任转发头。配置更新断开连接；替换/关闭及时结束旧 pending；调用默认十秒，连接空闲九十秒。有界事件队列独立持久化，不阻塞 echo；队列满关闭连接并显式记录错误。入库错误重试并关闭连接阻止积压。OneBot 不提供持久 ack/replay，不能承诺传输中和持久化前事件无损。

响应严格检查 echo/status/整数 retcode/data 及操作结果结构；ok/0 是平台成功；failed/非零为明确失败；async、格式异常、调用后失联为未知。不匹配平台错误文本；尝试日志区分未知超时、取消、断连、发送、协议/数据格式及 async 结果。所有读写及半开探测必须通过 Redis Lua 原子全局和群限流，Redis 不可用禁止绕过。

## 运维验收

双前端 /system/qqbot 提供脱敏配置、连接、限流等待、读写风控、收件箱/消费失败、操作队列、尝试、取消和受限重试。提供通用通知受控固定测试，禁止任意协议调用。后端所有管理接口 system.manage + super_admin。

验证真实 WebSocket、数据库、Redis，覆盖任务全部关键场景。生产 PostgreSQL 并发领取须单独验证，不以 SQLite 推断锁行为。

协议依据：[公开 API](https://github.com/botuniverse/onebot-11/blob/master/api/public.md)、[反向 WebSocket](https://github.com/botuniverse/onebot-11/blob/master/communication/ws-reverse.md)。

## 业务通知边界

`QQNotificationService` 负责稳定业务键、关联 ID、冻结目标及逐群业务结果；`QQBotService` 继续独占操作状态机、协议、重试和连接能力。通知与调度提交复用调用方事务，结果消费者用传入事务更新业务状态和 ack。业务 pending 不代表操作 queued：执行中及等待消费期间也保持 pending。执行准入读取当前通知渠道，关闭后未执行任务取消，避免重新启用时补发旧任务，停用事务同时持久取消旧排队操作。

## 群业务与巡检

`QQGroupBusinessService` 和 `QQGroupBusinessRepository` 承载独立申请、群规则、成员及巡检状态，不复用旧治理代码或数据。DefaultQQBotService 在 worker 启动前注册事件/结果持久消费者；执行准入在同一调度执行锁下复核当前业务状态。资格依赖异常只等待/复查，不复制传输错误分类或重试。通知和申请、名片、清退、巡检操作均禁止通用人工重试，以免子操作脱离业务关联。

巡检快照结果冻结成员并集到 `qq_group_inspection_item`，每成员事务同时提交资格判断、确认链、动作意图和 done。该 done 是重启恢复检查点，至少一次重试不会再次确认。成员事件使用持久语义键去重，记录平台时间排序与本地接收时间；巡检请求之前的快照不得覆盖接收于请求之后的成员事件。规则行锁序列化同群开轮，单实例执行锁序列化配置与外部动作准入；不持有数据库事务等待协议响应。

每分钟 `qq_group_business` 复查最多一百个待复查申请、启动到期群巡检并推进最多一百个成员检查点。每群十五分钟一轮，清退门槛由规则配置。巡检、申请、成员和业务事件审计记录当前无自动清理策略，审计列表有上限；QQ 传输记录仍按原调度保留期清理。生产 PostgreSQL 的业务并发锁仍须部署环境验证。

执行准入锁顺序：先复核并锁定业务行，再 RefreshOwned 锁定调度操作，最后检查取消/期限并转 executing。业务侧同样先锁规则/成员再更新取消标志，避免操作锁与业务锁反向获取。清退读取已提交的巡检依据时使用普通 MVCC 读取，不反向锁定正在推进的 run/item。所有锁在协议调用前释放。
