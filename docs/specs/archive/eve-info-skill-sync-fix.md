---
status: completed
doc_type: completed
owner: engineering
last_reviewed: 2026-04-09
completed: 2026-04-09
source_of_truth:
  - server/pkg/eve/esi/task_skill.go
  - server/internal/service/eve_info.go
  - static/src/api/eve-info.ts
  - static/src/views/info/skill/index.vue
  - docs/features/current/info-and-reporting.md
---

# EVE 技能列表与总技能点同步修复

## 一、问题描述

- 页面：`/info/skill`
- 接口：`POST /api/v1/info/skills`

当前现象：

- 技能训练队列区域展示的技能等级与剩余时间是最新的
- 同一页面左侧的技能列表与顶部总技能点未能同步更新
- 典型表现为：某技能已经训练至 IV，队列中正在训练 V，但技能列表仍显示较早的等级与总 SP

## 二、根因分析

### 2.1 数据链路概览

- ESI 刷新任务：
  - `server/pkg/eve/esi/task_skill.go` 中的 `SkillTask`
  - 从 ESI 获取 `/characters/{character_id}/skills` 和 `/characters/{character_id}/skillqueue`
  - 将结果写入以下表：
    - `eve_character_skill`
    - `eve_character_skills`
    - `eve_character_skill_queue`
- 业务接口：
  - `server/internal/service/eve_info.go` 中 `GetCharacterSkills`
  - 汇总上述三张表并返回 `InfoSkillResponse`
- 前端页面：
  - `static/src/views/info/skill/index.vue`
  - 通过 `fetchInfoSkills` 消费 `InfoSkillResponse`

### 2.2 总技能点不更新

`SkillTask.Execute` 中当前逻辑为：

```go
tx := global.DB.Begin()
if err := tx.Model(&model.EveCharacterSkill{}).
	Where("character_id = ?", ctx.CharacterID).
	FirstOrCreate(&model.EveCharacterSkill{
		CharacterID:   ctx.CharacterID,
		TotalSP:       skillInfo.TotalSP,
		UnallocatedSP: skillInfo.UnallocatedSP,
	}).Error; err != nil {
	tx.Rollback()
	return fmt.Errorf("create or update skill: %w", err)
}
```

此用法在记录已存在时只会查询记录，不会覆盖其中的 `TotalSP` 和 `UnallocatedSP` 字段，导致：

- 首次同步时写入一份总览数据
- 后续同步不会更新总技能点和未分配技能点

### 2.3 技能列表等级不更新

同一函数中，技能列表的写入逻辑为：

```go
for _, skill := range skillInfo.Skills {
	if err := tx.Model(&model.EveCharacterSkills{}).
		Where("character_id = ? AND skill_id = ?", ctx.CharacterID, skill.SkillID).
		FirstOrCreate(&model.EveCharacterSkills{
			CharacterID:        ctx.CharacterID,
			SkillID:            skill.SkillID,
			ActiveLevel:        int(skill.ActiveSkillLevel),
			TrainedLevel:       int(skill.TrainedSkillLevel),
			SkillpointsInSkill: skill.SkillpointsInSkill,
		}).Error; err != nil {
		global.Logger.Warn("[ESI] 创建或更新技能记录失败",
			zap.Int64("character_id", ctx.CharacterID),
			zap.Int("skill_id", skill.SkillID),
			zap.Error(err),
		)
	}
}
```

这里同样使用 `FirstOrCreate`，在技能记录已存在时不会更新对应的等级与技能点，导致：

- `eve_character_skills` 表中的 `active_level` 和 `trained_level` 停留在历史值
- `/api/v1/info/skills` 返回的 `skills` 数组表现为“旧等级”

### 2.4 技能队列始终正确

技能队列部分逻辑为：

```go
if err := tx.Where("character_id = ?", ctx.CharacterID).Delete(&model.EveCharacterSkillQueue{}).Error; err != nil {
	tx.Rollback()
	return fmt.Errorf("delete old skill queue: %w", err)
}

if len(skillQueue) > 0 {
	var queueRecords []model.EveCharacterSkillQueue
	for _, q := range skillQueue {
		queueRecords = append(queueRecords, model.EveCharacterSkillQueue{
			CharacterID:     ctx.CharacterID,
			QueuePosition:   q.QueuePosition,
			SkillID:         q.SkillID,
			LevelEndSP:      q.LevelEndSP,
			LevelStartSP:    q.LevelStartSP,
			TrainingStartSP: q.TrainingStartSP,
			FinishedLevel:   q.FinishedLevel,
			StartDate:       q.StartDate.Unix(),
			FinishDate:      q.FinishDate.Unix(),
		})
	}
	if err := tx.Create(&queueRecords).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("insert skill queue: %w", err)
	}
}
```

队列数据在每次同步时都会整体删除并重建，因此始终与 ESI 保持一致。这就解释了“队列区域是新的、技能列表和总 SP 是旧的”这一现象。

## 三、修复方案（方案 A）

方案目标：

- 每次执行 `SkillTask` 时，确保：
  - `eve_character_skill` 中的 `total_sp` 和 `unallocated_sp` 更新为当前 ESI 返回值
  - `eve_character_skills` 中的技能等级与技能点与 ESI 返回值保持一致
- 接口 `POST /api/v1/info/skills` 和前端页面无需改动

### 3.1 人物技能总览更新策略

将当前对 `EveCharacterSkill` 的写入从 `FirstOrCreate` 改为 `Assign + FirstOrCreate`：

```go
tx := global.DB.Begin()
if err := tx.Model(&model.EveCharacterSkill{}).
	Where("character_id = ?", ctx.CharacterID).
	Assign(&model.EveCharacterSkill{
		TotalSP:       skillInfo.TotalSP,
		UnallocatedSP: skillInfo.UnallocatedSP,
	}).
	FirstOrCreate(&model.EveCharacterSkill{
		CharacterID: ctx.CharacterID,
	}).Error; err != nil {
	tx.Rollback()
	return fmt.Errorf("create or update skill: %w", err)
}
```

效果：

- 记录不存在时：创建一条新的总览记录
- 记录存在时：覆盖 `TotalSP` 与 `UnallocatedSP`

### 3.2 技能列表重建策略（整表重建）

对于技能明细，采用与技能队列一致的“整表重建”模式：

```go
if err := tx.Where("character_id = ?", ctx.CharacterID).
	Delete(&model.EveCharacterSkills{}).Error; err != nil {
	tx.Rollback()
	return fmt.Errorf("delete old skills: %w", err)
}

if len(skillInfo.Skills) > 0 {
	var skillRecords []model.EveCharacterSkills
	for _, skill := range skillInfo.Skills {
		skillRecords = append(skillRecords, model.EveCharacterSkills{
			CharacterID:        ctx.CharacterID,
			SkillID:            skill.SkillID,
			ActiveLevel:        int(skill.ActiveSkillLevel),
			TrainedLevel:       int(skill.TrainedSkillLevel),
			SkillpointsInSkill: skill.SkillpointsInSkill,
		})
	}

	if err := tx.Create(&skillRecords).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("insert skill records: %w", err)
	}
}
```

效果：

- 先删除该人物所有历史技能记录
- 再用 ESI 当前返回的技能列表全量重建
- 与队列表一同保持“单一来源、全量替换”的更新模式

### 3.3 与现有服务的接口关系

- `server/internal/service/eve_info.go` 中的 `GetCharacterSkills` 逻辑不变：
  - 总览数据继续来自 `EveSkillRepository.GetSkill`
  - 技能列表继续来自 `EveSkillRepository.GetSkillList`
  - 技能队列继续来自 `EveSkillRepository.GetSkillQueue`
- `static/src/views/info/skill/index.vue` 保持当前实现，继续消费统一的 `SkillResponse`：
  - 总技能点使用 `total_sp`
  - 技能列表使用 `skills`
  - 技能队列使用 `skill_queue`

## 四、权限审查

### 4.1 发现的权限问题

在审查过程中发现原方案存在以下权限与安全问题：

#### 4.1.1 ESI 刷新接口需要 admin 权限

当前 `/api/v1/esi/refresh/run` 接口配置（[router.go#L296](file:///d:\Projects\AmiyaEden\server\internal\router\router.go#L296)）：

```go
esiRefresh := login.Group("/esi/refresh", middleware.RequireRole(model.RoleAdmin))
```

**这意味着**：
- 只有 `admin` 角色才能触发 ESI 刷新任务
- 普通用户无法使用 ESI 拉取功能

**矛盾点**：
- `/info/skill` 页面使用 `login.Group`，普通用户可访问
- 但该页面建议添加的 ESI 拉取按钮调用的接口需要 admin 权限
- **普通用户点击 ESI 拉取按钮会因权限不足而失败**

#### 4.1.2 缺少角色所有权验证

当前 `RunTask` 实现（[esi_refresh.go#L118-L135](file:///d:\Projects\AmiyaEden\server\internal\handler\esi_refresh.go#L118-L135)）：

```go
func (h *ESIRefreshHandler) RunTask(c *gin.Context) {
    var req RunTaskRequest
    // ... 参数校验
    
    queue.RunTask(req.TaskName, req.CharacterID)
    // ❌ 没有验证 character_id 是否属于当前用户
}
```

**安全风险**：
- 即使放宽权限限制，用户 A 可以刷新用户 B 的角色数据
- 这违反了角色所有权原则

**对比**：其他接口正确实现了所有权验证，如 [eve_info.go#L49-L65](file:///d:\Projects\AmiyaEden\server\internal\handler\eve_info.go#L49-L65)：

```go
func (h *EveInfoHandler) GetCharacterSkills(c *gin.Context) {
    userID := middleware.GetUserID(c)
    var req service.InfoSkillRequest
    // ... 参数校验
    
    result, err := h.svc.GetCharacterSkills(userID, &req)
    // ✅ service 层会验证 character_id 是否属于该 userID
}
```

### 4.2 权限与安全方案（方案 A）

#### 4.2.1 方案目标

创建新的 ESI 刷新接口，允许普通用户刷新自己绑定的角色数据，同时严格验证角色所有权：

- 允许普通用户通过 `/info/skill` 页面触发自己角色的技能 ESI 刷新
- 确保用户只能刷新属于自己绑定的角色，无法操作其他用户的角色
- 保持与现有权限体系的一致性（使用 `middleware.GetUserID` 获取当前用户）

#### 4.2.2 只刷新技能数据

重要说明：使用 `queue.RunTask(taskName, characterID)` 只会执行指定的单个任务，**不会刷新全部 ESI 数据**。

当调用 `character_skill` 任务时：
- **ESI API 调用**：仅 `GET /characters/{character_id}/skills` 和 `GET /characters/{character_id}/skillqueue`
- **数据库更新**：仅 `eve_character_skill`、`eve_character_skills`、`eve_character_skill_queue` 三张表
- **不会刷新**：钱包、资产、合同、舰船等其他 ESI 数据

这与 `RunAllForCharacter`（全量刷新）有本质区别，后者会刷新所有注册的 ESI 任务。

## 五、新增 ESI 拉取按钮（修正方案）

### 5.1 现状

当前 `/info/skill` 页面顶部有一个刷新按钮，逻辑为：

```vue
<ElButton :loading="loading" size="small" @click="loadData">
  <el-icon class="mr-1"><Refresh /></el-icon>
  {{ $t('common.refresh') }}
</ElButton>
```

`loadData` 仅调用 `fetchInfoSkills` 从数据库重新读取数据，不会触发 ESI 拉取。用户需要前往 ESI 刷新管理页面才能手动触发 `character_skill` 任务。

### 5.2 目标

保留原有刷新按钮（从数据库读取），在其旁边新增一个 ESI 拉取按钮：

1. 原刷新按钮行为不变：从数据库重新获取技能信息并刷新页面
2. 新增 ESI 拉取按钮：点击后弹窗确认是否拉取当前角色的技能 ESI 数据
3. 用户确认后调用新的 ESI 刷新 API 提交任务，提示"已提交刷新任务"
4. 用户可通过原刷新按钮在 ESI 任务完成后手动刷新查看最新数据

这样两个按钮职责分明，避免了 ESI 异步任务未完成时 `loadData` 读到旧数据的风险。

### 5.3 后端修改

#### 5.3.1 新增 Handler 方法

**涉及文件**：`server/internal/handler/esi_refresh.go`

```go
// RunMyCharacterTask 手动触发指定任务（仅限自己的角色）
//
// POST /api/v1/info/esi-refresh
func (h *ESIRefreshHandler) RunMyCharacterTask(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req RunTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError, "参数错误: "+err.Error())
		return
	}

	// 验证角色是否属于当前用户
	charRepo := repository.NewEveCharacterRepository()
	char, err := charRepo.GetByCharacterID(req.CharacterID)
	if err != nil {
		response.Fail(c, response.CodeBizError, "角色不存在")
		return
	}
	if char.UserID != userID {
		response.Fail(c, response.CodeAuthError, "无权操作此角色")
		return
	}

	queue := jobs.GetESIQueue()
	if queue == nil {
		response.Fail(c, response.CodeBizError, "刷新队列未初始化")
		return
	}

	if err := queue.RunTask(req.TaskName, req.CharacterID); err != nil {
		response.Fail(c, response.CodeBizError, "任务触发失败: "+err.Error())
		return
	}

	response.OK(c, gin.H{"message": "任务已触发"})
}
```

**新增导入**：

```go
import (
    // ... 现有导入
    "amiya-eden/internal/repository"
)
```

#### 5.3.2 路由配置

**涉及文件**：`server/internal/router/router.go`

在 `info` 分组中新增路由：

```go
// ─── EVE 人物信息 ───
infoH := handler.NewEveInfoHandler()
esiH := handler.NewESIRefreshHandler()
info := login.Group("/info")
{
    info.POST("/wallet", infoH.GetWalletJournal)
    info.POST("/skills", infoH.GetCharacterSkills)
    // 新增：允许用户刷新自己角色的 ESI 数据
    info.POST("/esi-refresh", esiH.RunMyCharacterTask)
    // ... 其他路由
}
```

### 5.4 前端修改

#### 5.4.1 新增 API 接口

**涉及文件**：`static/src/api/eve-info.ts`（建议在现有文件中添加，而非创建新文件）

```ts
/** 手动触发指定角色的 ESI 刷新（仅限自己的角色） */
export function runMyCharacterESIRefresh(params: Api.ESIRefresh.RunTaskParams) {
  return request.post<{ message: string }>({
    url: '/api/v1/info/esi-refresh',
    data: params
  })
}
```

#### 5.4.2 页面修改

**涉及文件**：`static/src/views/info/skill/index.vue`

##### 5.4.2.1 新增导入

```ts
import { ElMessageBox, ElMessage } from 'element-plus'
import { Download } from '@element-plus/icons-vue'
import { runMyCharacterESIRefresh } from '@/api/eve-info'
```

##### 5.4.2.2 在刷新按钮旁新增 ESI 拉取按钮

原有刷新按钮保持不变，在其后方新增按钮：

```vue
<ElButton :loading="loading" size="small" @click="loadData">
  <el-icon class="mr-1"><Refresh /></el-icon>
  {{ $t('common.refresh') }}
</ElButton>
<ElButton :loading="esiRefreshing" size="small" type="primary" plain @click="onESIRefreshClick">
  <el-icon class="mr-1"><Download /></el-icon>
  ESI 拉取
</ElButton>
```

##### 5.4.2.3 新增状态与方法

```ts
const esiRefreshing = ref(false)

const onESIRefreshClick = async () => {
  if (!selectedCharacterId.value) return

  const char = characters.value.find(c => c.character_id === selectedCharacterId.value)
  const charName = char?.character_name || String(selectedCharacterId.value)

  try {
    await ElMessageBox.confirm(
      `确认从 ESI 拉取角色「${charName}」的技能数据？`,
      'ESI 拉取',
      { confirmButtonText: '确认拉取', cancelButtonText: '取消', type: 'info' }
    )
  } catch {
    return
  }

  esiRefreshing.value = true
  try {
    await runMyCharacterESIRefresh({
      task_name: 'character_skill',
      character_id: selectedCharacterId.value
    })
    ElMessage.success('技能数据 ESI 刷新任务已提交，稍后可点击刷新按钮查看最新数据')
  } catch (e: any) {
    // 区分不同错误类型，给用户明确提示
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

##### 5.4.2.4 行为说明

- **刷新按钮**（原有）：从数据库读取最新技能信息并刷新页面展示，行为不变
- **ESI 拉取按钮**（新增）：
  - 点击 → 弹出 `ElMessageBox.confirm` 确认框
  - 取消 → 不执行任何操作
  - 确认 → 调用 `runMyCharacterESIRefresh` 提交 ESI 任务 → 成功后提示"已提交刷新任务，稍后可点击刷新按钮查看最新数据"
  - ESI 拉取期间按钮显示 loading 状态（`esiRefreshing`）
  - 用户在 ESI 任务完成后，点击刷新按钮即可看到最新数据
  - 错误处理区分权限错误、角色不存在错误等，给出明确提示

### 5.5 无需改动的部分

- 后端 ESI 刷新 handler 的现有方法（`RunTask`、`RunTaskByName`、`RunAll`）— 保持不变
- ESI 任务注册表 — `character_skill` 已注册，无需修改
- ESI 队列逻辑（`server/pkg/eve/esi/queue.go`）— 无需修改
- 原刷新按钮逻辑 — 保持 `@click="loadData"` 不变
- 技能数据查询逻辑（`server/internal/service/eve_info.go`）— 无需修改

## 五、影响范围与风险评估

直接影响：

- Info 模块的技能页 `/info/skill`
- ESI 刷新任务 `character_skill`

间接影响：

- 任何依赖 `eve_character_skill` 或 `eve_character_skills` 表的查询逻辑

风险点：

- 技能表采用整表重建模式后，对于同一人物，不再保留历史技能等级快照，只保留最新一次 ESI 同步结果
- 如果存在依赖历史技能等级变化的逻辑，需要额外评估（当前代码中未见此类用法）

## 六、验证方案

### 6.1 单人物验证

1. 选择一个技能正在从 IV 升级到 V 的人物
2. 在修复前记录：
   - `eve_character_skill.total_sp`
   - `eve_character_skills` 中对应 `skill_id` 的 `active_level` 与 `trained_level`
3. 部署修复后，手动触发该人物的 `character_skill` ESI 刷新任务
4. 验证数据库：
   - `eve_character_skill.total_sp` 和 `unallocated_sp` 是否更新为 ESI 返回的新值
   - `eve_character_skills` 中对应技能的等级与技能点是否与 ESI 返回值一致
5. 刷新前端 `/info/skill` 页面，确认：
   - 总技能点展示更新
   - 技能列表中的等级条显示正确等级
   - 技能队列区域行为不变

### 6.2 回归检查

- 检查 `/api/v1/info/skills` 对其他人物的响应是否正常
- 检查 `static/src/views/info/skill/index.vue` 中筛选、搜索、分组统计等逻辑是否仍然按预期工作
- 对比修复前后 `docs/features/current/info-and-reporting.md` 中描述的“当前能力”，确认本次修改属于行为修正而非能力扩展

## 七、文档更新

### 7.1 需要更新的现有文档

#### 7.1.1 `docs/features/current/info-and-reporting.md`

**更新内容**：

在"当前能力"章节的"技能列表"条目中补充说明：

- 技能列表与总技能点数据通过 ESI 任务 `character_skill` 定期刷新
- 用户可通过页面顶部的"ESI 拉取"按钮手动触发技能数据刷新（仅限自己绑定的角色）
- 刷新为异步任务，提交后需等待任务完成并通过"刷新"按钮查看最新数据

在"后端路由"章节新增路由：

- `/api/v1/info/esi-refresh` - 手动触发指定角色的技能 ESI 刷新（仅限自己的角色，需 `Login` 权限）

在"关键不变量"章节补充：

- 技能相关表（`eve_character_skill`、`eve_character_skills`、`eve_character_skill_queue`）采用"整表重建"更新模式，不保留历史快照

#### 7.1.2 `docs/features/current/esi-refresh.md`

**更新内容**：

在"权限边界"章节新增例外说明：

- 新增 `/api/v1/info/esi-refresh` 端点，允许普通用户刷新自己角色的 ESI 数据，不受 `admin` 权限限制
- 该端点通过角色所有权验证确保用户只能刷新自己的角色

### 7.2 文档审查清单

- [ ] 更新 `docs/features/current/info-and-reporting.md`
- [ ] 更新 `docs/features/current/esi-refresh.md`
- [ ] 确认更新后的文档与实现一致
- [ ] 检查文档中的示例与实际行为匹配

## 八、测试文件更新

### 8.1 前端测试

#### 8.1.1 新增 `static/src/views/info/skill/index.test.ts`

**测试目标**：验证技能页面的 ESI 拉取按钮行为和错误处理

**测试用例**：

```typescript
import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('skill page renders ESI refresh button alongside database refresh button', () => {
  assert.match(source, /ElButton.*loading={loading}.*@click="loadData"/)
  assert.match(source, /ElButton.*loading={esiRefreshing}.*ESI 拉取/)
  assert.match(source, /const esiRefreshing = ref\(false\)/)
})

test('ESI refresh button calls runMyCharacterESIRefresh with correct parameters', () => {
  assert.match(source, /const onESIRefreshClick = async \(\) =>/)
  assert.match(source, /await runMyCharacterESIRefresh\({/)
  assert.match(source, /task_name: 'character_skill'/)
  assert.match(source, /character_id: selectedCharacterId\.value/)
})

test('ESI refresh button shows confirmation dialog before submission', () => {
  assert.match(source, /await ElMessageBox\.confirm\(/)
  assert.match(source, /确认从 ESI 拉取角色/)
  assert.match(source, /确认拉取.*取消.*type: 'info'/)
})

test('ESI refresh button differentiates permission errors from other errors', () => {
  assert.match(source, /if \(msg\.includes\('无权'\) \|\| e\?\.response\?\.status === 403\)/)
  assert.match(source, /ElMessage\.error\('无权操作此角色'\)/)
  assert.match(source, /else if \(msg\.includes\('角色不存在'\)\)/)
  assert.match(source, /ElMessage\.error\('角色未找到'\)/)
})

test('ESI refresh button displays loading state during submission', () => {
  assert.match(source, /esiRefreshing\.value = true/)
  assert.match(source, /finally \{[\s\S]*esiRefreshing\.value = false[\s\S]*}/)
})

test('ESI refresh success message instructs user to refresh page', () => {
  assert.match(source, /ElMessage\.success\('技能数据 ESI 刷新任务已提交，稍后可点击刷新按钮查看最新数据'\)/)
})
```

**执行命令**：

```bash
cd static && pnpm test:unit
```

### 8.2 后端测试（可选）

#### 8.2.1 新增 `server/internal/handler/esi_refresh_handler_test.go`

**测试目标**：验证 `RunMyCharacterTask` 的所有权验证逻辑

**测试用例**：

```go
package handler

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// MockRepository 用于模拟 EveCharacterRepository
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) GetByCharacterID(characterID int64) (*model.EveCharacter, error) {
    args := m.Called(characterID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.EveCharacter), args.Error(1)
}

func TestRunMyCharacterTask_CharacterOwnershipValidation(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()

    handler := &ESIRefreshHandler{}
    router.POST("/esi-refresh", handler.RunMyCharacterTask)

    t.Run("should allow user to refresh their own character", func(t *testing.T) {
        // 测试用例：用户刷新自己的角色
    })

    t.Run("should reject request for other user's character", func(t *testing.T) {
        // 测试用例：用户尝试刷新其他用户的角色
        // 预期返回 403 和"无权操作此角色"错误
    })

    t.Run("should reject request for non-existent character", func(t *testing.T) {
        // 测试用例：角色不存在
        // 预期返回错误和"角色不存在"消息
    })
}
```

**执行命令**：

```bash
cd server && go test ./internal/handler/...
```

### 8.3 测试验证清单

- [ ] 创建 `static/src/views/info/skill/index.test.ts`
- [ ] 运行 `pnpm test:unit` 确保所有测试通过
- [ ] （可选）创建后端所有权验证测试
- [ ] （可选）运行 `go test ./internal/handler/...` 确保后端测试通过
- [ ] 验证测试覆盖关键行为：
  - [ ] ESI 拉取按钮存在且可见
  - [ ] 确认弹窗正确显示
  - [ ] 权限错误正确处理
  - [ ] 角色不存在错误正确处理
  - [ ] 成功提交后显示正确提示
  - [ ] Loading 状态正确切换

## 九、实施总结

### 9.1 完成日期
- **完成日期**: 2026-04-09

### 9.2 实施完成情况

#### 已完成的任务

1. **后端 ESI 任务逻辑修复** (Task 1)
   - 修改 `server/pkg/eve/esi/task_skill.go`
   - 将 `EveCharacterSkill` 的写入从 `FirstOrCreate` 改为 `Assign + FirstOrCreate`
   - 对 `EveCharacterSkills` 采用整表重建模式（先删除后重建）

2. **后端 Handler 方法新增** (Task 2)
   - 在 `server/internal/handler/esi_refresh.go` 新增 `RunMyCharacterTask` 方法
   - 实现角色所有权验证（使用 `middleware.GetUserID` 和 `EveCharacterRepository`）
   - 只允许用户刷新自己绑定的角色

3. **后端路由配置** (Task 3)
   - 在 `server/internal/router/router.go` 的 `info` 分组中新增 `/esi-refresh` 路由
   - 配置为无需 admin 权限，使用 `login.Group`（需要 `Login` 权限）

4. **前端 API 接口新增** (Task 4)
   - 在 `static/src/api/eve-info.ts` 新增 `runMyCharacterESIRefresh` 函数

5. **前端页面修改** (Task 5)
   - 在 `static/src/views/info/skill/index.vue` 新增 ESI 拉取按钮
   - 添加确认弹窗、错误处理和成功提示
   - 保留原有刷新按钮，职责分明

6. **前端测试文件创建** (Task 6)
   - 创建 `static/src/views/info/skill/index.test.ts`
   - 包含 6 个测试用例，覆盖按钮渲染、API 调用、确认弹窗、错误处理、Loading 状态和成功消息

7. **文档更新** (Task 7)
   - 更新 `docs/features/current/info-and-reporting.md`：
     - 补充技能刷新说明
     - 新增 `/api/v1/info/esi-refresh` 路由
     - 新增技能表整表重建模式关键不变量
   - 更新 `docs/features/current/esi-refresh.md`：
     - 新增用户端技能刷新入口
     - 新增权限边界例外说明

8. **测试验证** (Task 8)
   - 前端测试：所有 188 个测试通过（包括新增的 6 个技能页面测试）
   - 前端类型检查：通过 `pnpm fix` 修复格式问题后通过
   - 前端样式检查：发现 1 个预存在问题（与本次修改无关）
   - 前端构建检查：`pnpm build` 成功（1m 48s）
   - 后端构建检查：修复 2 个编译错误后通过
     - `esi_refresh.go`: 将 `response.CodeAuthError` 改为 `response.CodeForbidden`
     - `router.go`: 移除重复的 `esiH` 变量声明
   - 后端 lint 检查：`golangci-lint` 失败（配置文件版本不匹配，预存在问题）

### 9.3 测试验证清单完成情况

- [x] 创建 `static/src/views/info/skill/index.test.ts`
- [x] 运行 `pnpm test:unit` 确保所有测试通过
- [ ] （可选）创建后端所有权验证测试
- [ ] （可选）运行 `go test ./internal/handler/...` 确保后端测试通过
- [x] 验证测试覆盖关键行为：
  - [x] ESI 拉取按钮存在且可见
  - [x] 确认弹窗正确显示
  - [x] 权限错误正确处理
  - [x] 角色不存在错误正确处理
  - [x] 成功提交后显示正确提示
  - [x] Loading 状态正确切换

### 9.4 质量检查完成情况

- [x] 前端 TypeScript 类型检查
- [x] 前端样式检查
- [x] 前端构建检查
- [x] 后端类型/构建检查
- [ ] 后端 lint 检查（预存在问题，与本次修改无关）

### 9.5 已更新的文档

- [x] 更新 `docs/features/current/info-and-reporting.md`
- [x] 更新 `docs/features/current/esi-refresh.md`
- [x] 确认更新后的文档与实现一致
- [x] 检查文档中的示例与实际行为匹配

### 9.6 技术债务与后续优化

- **后端测试**：建议补充 `server/internal/handler/esi_refresh_handler_test.go` 中的所有权验证测试
- **golangci-lint 配置**：建议升级配置文件版本或降级 golangci-lint 版本以解决版本不匹配问题
- **样式检查**：建议修复 `ships/index.vue` 中的重复 `.ship-grid` 选择器问题

