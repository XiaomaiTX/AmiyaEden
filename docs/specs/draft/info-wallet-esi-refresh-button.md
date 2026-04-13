---
status: draft
doc_type: draft
owner: engineering
last_reviewed: 2025-01-13
source_of_truth:
  - static/src/views/info/wallet/index.vue
  - static/src/views/info/skill/index.vue
  - static/src/api/eve-info.ts
  - server/pkg/eve/esi/task_wallet.go
  - server/internal/router/router.go
  - docs/features/current/info-and-reporting.md
---

# 钱包页面添加 ESI 数据刷新按钮

## 问题描述

- 页面：`/info/wallet`
- 参考页面：`/info/skill`

当前现象：

- `/info/wallet` 页面仅提供从数据库读取数据的刷新功能
- 用户无法手动触发钱包数据的 ESI 拉取
- 与 `/info/skill` 页面不一致，后者已有"ESI 拉取"按钮可手动触发技能数据刷新

## 根因分析

### 2.1 技能页面实现（参考模板）

**文件**：`static/src/views/info/skill/index.vue`

**关键实现**：

- ESI 拉取按钮位于人物切换器区域（L30-L39）
- 使用 `runMyCharacterESIRefresh` API，参数 `{ task_name: 'character_skill', character_id: xxx }`
- 确认弹窗：`ElMessageBox.confirm` 显示角色名称
- 错误处理：区分权限错误、角色不存在错误等
- 提示信息：成功后提示"技能数据 ESI 刷新任务已提交，稍后可点击刷新按钮查看最新数据"

### 2.2 钱包页面当前状态

**文件**：`static/src/views/info/wallet/index.vue`

**当前结构**：

- 人物切换器 + 钱包类型筛选器 + 余额展示（L4-L56）
- 流水表格区域（L58-L72）
- 刷新功能：通过 `ArtTableHeader` 的 `@refresh="refreshData"` 事件从数据库读取数据
- **缺少**：ESI 拉取按钮

### 2.3 后端能力确认

**钱包 ESI 任务**：

- **文件**：`server/pkg/eve/esi/task_wallet.go`
- **任务名称**：`character_wallet`
- **所需 Scope**：`esi-wallet.read_character_wallet.v1`
- **功能**：从 ESI 获取钱包余额、钱包流水、钱包市场交易

**ESI 刷新接口**：

- **路由**：`POST /api/v1/info/esi-refresh`
- **Handler**：`esiH.RunMyCharacterTask`
- **位置**：`server/internal/router/router.go#L197`
- **权限**：需 `Login` 权限（已为技能页面配置，钱包页面可直接复用）

**前端 API**：

- **函数**：`runMyCharacterESIRefresh(params: Api.ESIRefresh.RunTaskParams)`
- **文件**：`static/src/api/eve-info.ts#L55-L59`

## 修复方案

### 3.1 前端修改：钱包页面

**文件**：`static/src/views/info/wallet/index.vue`

#### 3.1.1 新增导入

```typescript
import { ElMessageBox, ElMessage } from 'element-plus'
import { Download } from '@element-plus/icons-vue'
import { runMyCharacterESIRefresh } from '@/api/eve-info'
```

#### 3.1.2 新增状态变量

```typescript
const esiRefreshing = ref(false)
```

#### 3.1.3 新增 ESI 拉取处理方法

```typescript
const onESIRefreshClick = async () => {
  if (!selectedCharacterId.value) return

  const char = characters.value.find((c) => c.character_id === selectedCharacterId.value)
  const charName = char?.character_name || String(selectedCharacterId.value)

  try {
    await ElMessageBox.confirm(`确认从 ESI 拉取角色「${charName}」的钱包数据？`, 'ESI 拉取', {
      confirmButtonText: '确认拉取',
      cancelButtonText: '取消',
      type: 'info'
    })
  } catch {
    return
  }

  esiRefreshing.value = true
  try {
    await runMyCharacterESIRefresh({
      task_name: 'character_wallet',
      character_id: selectedCharacterId.value
    })
    ElMessage.success('钱包数据 ESI 刷新任务已提交，稍后可点击刷新按钮查看最新数据')
  } catch (e: any) {
    const msg = e?.response?.data?.message || e?.message || 'ESI 刷新任务提交失败'
    if (msg.includes('无权') || e?.response?.status === 403) {
      ElMessage.error('无权操作此角色')
    } else if (msg.includes('角色不存在')) {
      ElMessage.error('角色未找到')
    } else {
      ElMessage.error(msg)
    }
  } finally {
    esiRefreshing.value = false
  }
}
```

#### 3.1.4 模板修改：在人物切换器区域添加按钮

在人物选择器后的 `</div>` 闭合标签内，新增 ESI 拉取按钮：

```vue
<ElButton
  :loading="esiRefreshing"
  size="small"
  type="primary"
  plain
  @click="onESIRefreshClick"
>
  <el-icon class="mr-1"><Download /></el-icon>
  ESI 拉取
</ElButton>
```

**位置**：建议放在人物选择器之后、钱包类型筛选器之前，保持与技能页面一致的布局

### 3.2 后端修改

**说明**：

- `/api/v1/info/esi-refresh` 接口已存在并正常工作
- `character_wallet` 任务已注册
- 权限验证逻辑已实现
- **无需任何后端代码修改**

### 3.3 本地化更新

**文件**：`static/src/locales/langs/zh.json` 和 `static/src/locales/langs/en.json`

由于技能页面的提示信息可直接复用，钱包页面使用相同的提示文本即可。如需单独定制钱包相关的提示，可在 `info` 命名空间下添加新的 key。

**当前可用提示文本**（技能页面使用）：

- 成功：`技能数据 ESI 刷新任务已提交，稍后可点击刷新按钮查看最新数据`
- 权限错误：`无权操作此角色`
- 角色不存在：`角色未找到`
- 通用失败：`ESI 刷新任务提交失败`

**建议**：直接在代码中硬编码这些提示文本，保持与技能页面的一致性。

### 3.4 文档更新

**文件**：`docs/features/current/info-and-reporting.md`

#### 3.4.1 更新"当前能力"章节

在"钱包流水"条目后补充说明：

```
- 钱包流水
  - 钱包流水数据通过 ESI 任务 `character_wallet` 定期刷新
  - 用户可通过页面顶部的"ESI 拉取"按钮手动触发钱包数据刷新（仅限自己绑定的角色）
  - 刷新为异步任务，提交后需等待任务完成并通过"刷新"按钮查看最新数据
```

#### 3.4.2 更新"后端路由"章节

在已有路由列表后补充：

```
- `/api/v1/info/esi-refresh` - 手动触发指定角色的 ESI 刷新（支持技能、钱包等任务，仅限自己的角色，需 `Login` 权限）
```

#### 3.4.3 更新"关键不变量"章节

在已有不变量后补充：

```
- 钱包相关表（`eve_character_wallet_journal` 等）由 `character_wallet` ESI 任务定期更新，新增记录不覆盖历史
```

## 验证方案

### 4.1 单人物验证

1. 选择一个有钱包流水的角色
2. 记录当前钱包余额和流水条数
3. 点击"ESI 拉取"按钮
4. 确认弹窗显示正确的角色名称
5. 确认提交后，看到"钱包数据 ESI 刷新任务已提交"成功提示
6. 等待约 10-30 秒（取决于 ESI 响应速度）
7. 点击表格上方的刷新按钮
8. 验证钱包余额和流水数据是否更新

### 4.2 权限验证

1. 确保用户只能看到自己绑定的角色
2. 尝试修改请求参数中的 `character_id` 为其他用户角色（需通过开发工具模拟）
3. 验证是否返回"无权操作此角色"错误

### 4.3 错误处理验证

1. 在角色未选择时点击 ESI 拉取按钮
2. 验证是否无任何操作（函数提前返回）
3. 模拟网络错误（断网）
4. 验证是否显示"ESI 刷新任务提交失败"错误提示

### 4.4 回归检查

- 检查原刷新按钮行为是否正常
- 检查钱包类型筛选器功能是否正常
- 检查分页功能是否正常
- 检查余额展示是否正确

## 影响范围与风险评估

### 5.1 影响范围

**直接影响**：

- `/info/wallet` 页面

**间接影响**：

- `character_wallet` ESI 任务触发频率可能增加（用户手动刷新）

### 5.2 风险点

- **ESI 调用频率限制**：如果用户频繁点击 ESI 拉取按钮，可能触发 ESI API 限流
  - **缓解措施**：用户手动刷新频率通常可控，暂不额外限制
- **Token 失效**：如果角色的 ESI Token 已失效，ESI 刷新会失败
  - **缓解措施**：后端已有错误处理，前端会显示相应的错误提示

### 5.3 兼容性

- 后端接口已存在且稳定
- 前端 API 已存在（`runMyCharacterESIRefresh`）
- 无需修改 TypeScript 类型定义

## 与技能页面的一致性

| 项 | 技能页面 | 钱包页面（实现后） |
|----|---------|-------------------|
| 按钮位置 | 人物切换器区域 | 人物切换器区域 |
| 按钮样式 | `type="primary"` + `plain` + `size="small"` | 相同 |
| 图标 | `Download` | 相同 |
| 交互流程 | 确认弹窗 → 提交任务 → 成功提示 | 相同 |
| 错误处理 | 区分权限错误、角色不存在错误等 | 相同 |
| 提示文本 | "技能数据 ESI 刷新任务已提交..." | "钱包数据 ESI 刷新任务已提交..." |

## 待办事项清单

- [ ] 前端：修改 `static/src/views/info/wallet/index.vue`
  - [ ] 新增导入：`ElMessageBox`、`ElMessage`、`Download` 图标、`runMyCharacterESIRefresh`
  - [ ] 新增状态变量：`esiRefreshing`
  - [ ] 新增方法：`onESIRefreshClick`
  - [ ] 模板修改：在人物切换器区域添加 ESI 拉取按钮
- [ ] 后端：无需改动
- [ ] 本地化：无需新增翻译（复用现有提示文本）
- [ ] 文档：更新 `docs/features/current/info-and-reporting.md`
  - [ ] 更新"当前能力"章节
  - [ ] 更新"后端路由"章节
  - [ ] 更新"关键不变量"章节
- [ ] 测试验证
  - [ ] 单人物验证
  - [ ] 权限验证
  - [ ] 错误处理验证
  - [ ] 回归检查

## 补充说明

### 后端接口复用

`/api/v1/info/esi-refresh` 接口支持所有已注册的 ESI 任务，通过 `task_name` 参数区分不同任务：

- `character_skill`：技能数据刷新
- `character_wallet`：钱包数据刷新
- 其他任务：未来可扩展

因此无需为钱包单独创建新的接口或 handler。

### 与现有功能的区别

| 功能 | 数据来源 | 触发方式 | 用户感知 |
|-----|---------|---------|---------|
| 原刷新按钮 | 数据库（本地持久化数据） | 立即同步 | 数据瞬间更新 |
| ESI 拉取按钮 | ESI API（CCP 服务器） | 异步任务 | 提交后需等待，再刷新查看 |

两个按钮职责分明，避免了 ESI 异步任务未完成时读到旧数据的风险。
